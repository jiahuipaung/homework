package sink

import (
	"context"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/flashcatcloud/fc-stash/utils"
	"github.com/flashcatcloud/go-pkg/x/prom_exporter"
	"github.com/olivere/elastic/v7"
)

var (
	defaultRetryItemStatusCodes = map[int]struct{}{
		408: {}, 429: {}, 503: {}, 507: {},
	}
)

// bulk_size 修正为原始数据的大小
// bulk_size/bulk_actions 以pipeline维度生效, 原pkg是worker维度生效, 会带来"巨大"的延迟
// 多worker不再是多个独立的goroutine, 而是bulk request的并发数

type BulkProcessor struct {
	c                    *elastic.Client
	labelIndex           string
	afterFn              elastic.BulkAfterFunc
	bulkActions          int
	bulkSize             int // in byte
	flushInterval        time.Duration
	flushC               chan struct{}
	flusherStopC         chan struct{}
	requestsC            chan elastic.BulkableRequest
	numWorkers           int
	workerWg             *utils.WaitGroupWrapper
	backoff              elastic.Backoff // a custom Backoff to use for errors
	executionId          int64
	startedMu            sync.Mutex // guards the following block
	started              bool
	retryItemStatusCodes map[int]struct{}
	requests             []elastic.BulkableRequest
	sizeInBytes          int64
	sizeInBytesCursor    int
	servicePool          *stack
}

func NewBulkProcessor(client *elastic.Client) *BulkProcessor {
	return &BulkProcessor{
		c:           client,
		numWorkers:  1,
		bulkActions: 1000,
		bulkSize:    5 << 20, // 5 MB
		backoff: elastic.NewExponentialBackoff(
			time.Duration(200)*time.Millisecond,
			time.Duration(10000)*time.Millisecond,
		),
		retryItemStatusCodes: defaultRetryItemStatusCodes,
		requests:             make([]elastic.BulkableRequest, 0),
	}
}

func (p *BulkProcessor) Workers(num int) *BulkProcessor {
	p.numWorkers = num
	return p
}

func (p *BulkProcessor) BulkActions(bulkActions int) *BulkProcessor {
	p.bulkActions = bulkActions
	return p
}

func (p *BulkProcessor) BulkSize(bulkSize int) *BulkProcessor {
	p.bulkSize = bulkSize
	return p
}

func (p *BulkProcessor) FlushInterval(interval time.Duration) *BulkProcessor {
	p.flushInterval = interval
	return p
}

func (p *BulkProcessor) LabelIndex(index string) *BulkProcessor {
	p.labelIndex = index
	return p
}

func (p *BulkProcessor) After(fn elastic.BulkAfterFunc) *BulkProcessor {
	p.afterFn = fn
	return p
}

func (p *BulkProcessor) Add(request elastic.BulkableRequest) {
	p.requestsC <- request
}

func (p *BulkProcessor) Start(ctx context.Context) error {
	p.startedMu.Lock()
	defer p.startedMu.Unlock()

	if p.started {
		return nil
	}

	// We must have at least one worker.
	if p.numWorkers < 1 {
		p.numWorkers = 1
	}

	p.servicePool = new(stack)
	for i := 0; i < p.numWorkers+2; i++ { // extra 2 more
		p.servicePool.Push(elastic.NewBulkService(p.c))
	}
	p.executionId = 0
	p.requestsC = make(chan elastic.BulkableRequest)
	p.workerWg = utils.NewWG(p.numWorkers)
	go p.work(ctx)

	if int64(p.flushInterval) > 0 {
		p.flushC = make(chan struct{})
		p.flusherStopC = make(chan struct{})
		go p.flush(p.flushInterval)
	}

	p.started = true

	return nil
}

func (p *BulkProcessor) Stop() error {
	return p.Close()
}

func (p *BulkProcessor) Close() error {
	p.startedMu.Lock()
	defer p.startedMu.Unlock()

	// Already stopped? Do nothing.
	if !p.started {
		return nil
	}

	// stop flusher (if enabled)
	if p.flusherStopC != nil {
		close(p.flusherStopC)
	}
	if p.flushC != nil {
		close(p.flushC)
	}

	// stop worker
	close(p.requestsC)
	p.workerWg.Wait()

	p.started = false
	return nil
}

func (p *BulkProcessor) NumberOfActions() int {
	return len(p.requests)
}

func (p *BulkProcessor) EstimatedSizeInBytes() int64 {
	if p.sizeInBytesCursor == len(p.requests) {
		return p.sizeInBytes
	}
	for _, r := range p.requests[p.sizeInBytesCursor:] {
		p.sizeInBytes += p.estimateSizeInBytes(r)
		p.sizeInBytesCursor++
	}
	return p.sizeInBytes
}

func (p *BulkProcessor) work(ctx context.Context) {
	wrapFn := func(ctx context.Context, reqs []elastic.BulkableRequest) {
		prom_exporter.Inc("log_event_output_elastic_bulk_processor_worker")

		if len(reqs) == 0 {
			return
		}
		if err := p.commit(ctx, reqs); err != nil {
			if utils.Logger != nil {
				utils.Logger.Sugar().Warnf("elastic: bulk processor was unable to perform work: %v", err)
			}
		}
	}
	var stop bool
	for !stop {
		select {
		case req, open := <-p.requestsC:
			if open {
				if _, err := req.Source(); err == nil {
					p.requests = append(p.requests, req)
					if p.commitRequired() {
						reqs := p.dumpAndreset()
						p.workerWg.WrapFunc(wrapFn, ctx, reqs)
					}
				}
			} else {
				// Channel closed: Stop.
				stop = true
				if p.NumberOfActions() > 0 {
					reqs := p.dumpAndreset()
					p.workerWg.WrapFunc(wrapFn, ctx, reqs)
				}
			}

		case <-p.flushC:
			if p.NumberOfActions() > 0 {
				reqs := p.dumpAndreset()
				p.workerWg.WrapFunc(wrapFn, ctx, reqs)
			}
		}
	}
}

func (p *BulkProcessor) commitRequired() bool {
	// 以estimatesize优先
	if p.bulkSize > 0 && p.EstimatedSizeInBytes() >= int64(p.bulkSize) {
		return true
	}
	if p.bulkActions > 0 && p.NumberOfActions() >= p.bulkActions {
		return true
	}
	return false
}

func (p *BulkProcessor) estimateSizeInBytes(r elastic.BulkableRequest) int64 {
	lines, _ := r.Source()
	size := 0
	if len(lines) == 2 {
		size += len(lines[1]) // line[0] 是meta
	} else {
		for _, line := range lines {
			// +1 for the \n
			size += len(line) + 1
		}
	}
	return int64(size)
}

func (p *BulkProcessor) commit(ctx context.Context, reqs []elastic.BulkableRequest) error {
	var res *elastic.BulkResponse
	var retry int = 0
	// reqs and res when retry
	var _reqs []elastic.BulkableRequest
	var _res *elastic.BulkResponse

	// bulk service 在单个commit中唯一, 用于对重试请求处理
	service := p.getBulkService()
	defer p.putBulkService(service)

	// commitFunc will commit bulk requests and, on failure, be retried
	// via exponential backoff
	commitFunc := func() error {
		start := time.Now()
		defer func() {
			if retry == 1 { // 只记录完整的reqs请求的提交耗时
				// 成功的bulk http请求的耗时
				prom_exporter.Histogram("log_event_output_elastic_bulk_request_milliseconds",
					float64(time.Since(start).Milliseconds()))
			}
		}()

		var err error
		if retry == 0 { // 第一次commit or 整个bulk http失败后的重试
			res, err = service.Add(reqs...).Do(ctx)
		} else {
			// retry with part of reqs
			_res, err = service.Add(_reqs...).Do(ctx)
		}
		if err != nil {
			// if successed, service will be reset in service.Do
			// if err happened, redo all the reqs, reset service by hand
			service.Reset()
			prom_exporter.Inc("log_event_output_elastic_bulk_request_error")
		}
		if err == nil {
			retry++ // http请求成功, 即使重试只会发生在部分元素上
			var _toretry []elastic.BulkableRequest
			if retry <= 1 {
				// Overall bulk request was OK.  But each bulk response item also has a status
				if p.retryItemStatusCodes != nil && len(p.retryItemStatusCodes) > 0 {
					// Check res.Items since some might be soft failures
					if res.Items != nil && res.Errors {
						// res.Items will be 1 to 1 with reqs in same order
						for i, item := range res.Items {
							for _, result := range item {
								if _, found := p.retryItemStatusCodes[result.Status]; found {
									if i < len(reqs) { // 防止越界, 实际不会发生
										_toretry = append(_toretry, reqs[i])
									}
									prom_exporter.Inc("log_event_output_elastic_bulk_retry_status", map[string]string{
										"code": strconv.Itoa(result.Status),
									})
								}
							}
						}
					}
				}
			} else {
				if res != nil && res.Items != nil && _res.Items != nil {
					// 统一汇总到res.Items中, 用于bulkAfter()的汇总
					res.Items = append(res.Items, _res.Items...)
				}
				// Overall bulk request was OK.  But each bulk response item also has a status
				if p.retryItemStatusCodes != nil && len(p.retryItemStatusCodes) > 0 {
					// Check res.Items since some might be soft failures
					if _res.Items != nil && _res.Errors {
						// res.Items will be 1 to 1 with reqs in same order
						for i, item := range _res.Items {
							for _, result := range item {
								if _, found := p.retryItemStatusCodes[result.Status]; found {
									if i < len(_reqs) {
										_toretry = append(_toretry, _reqs[i])
									}
									prom_exporter.Inc("log_event_output_elastic_bulk_retry_status", map[string]string{
										"code": strconv.Itoa(result.Status),
									})
								}
							}
						}
					}
				}
			}
			if len(_toretry) > 0 {
				err = elastic.ErrBulkItemRetry
				_reqs = _toretry // 待重试的请求
			}
		}
		return err
	}
	notifyFunc := func(err error) {
		if utils.Logger != nil {
			utils.Logger.Sugar().Errorf("elastic: bulk processor failed but may retry: %v", err)
		}
	}
	id := atomic.AddInt64(&p.executionId, 1)

	// Commit bulk requests
	err := elastic.RetryNotify(commitFunc, p.backoff, notifyFunc)
	if err != nil {
		if utils.Logger != nil {
			utils.Logger.Sugar().Errorf("elastic: bulk processor failed: %v", err)
		}
	}

	// Invoke after callback
	if p.afterFn != nil {
		p.afterFn(id, reqs, res, err)
	}

	return err
}

func (p *BulkProcessor) getBulkService() *elastic.BulkService {
	item := p.servicePool.Pop()
	if item == nil {
		return elastic.NewBulkService(p.c)
	}
	return item.(*elastic.BulkService)
}

func (p *BulkProcessor) putBulkService(s *elastic.BulkService) {
	p.servicePool.Push(s)
}

func (p *BulkProcessor) dumpAndreset() (reqs []elastic.BulkableRequest) {
	reqs = p.requests
	prom_exporter.IncN("log_event_out_bytes", int(p.sizeInBytes), map[string]string{
		"index": p.labelIndex,
	})
	p.requests = make([]elastic.BulkableRequest, 0)
	p.sizeInBytes = 0
	p.sizeInBytesCursor = 0
	return
}

func (p *BulkProcessor) flush(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C: // Periodic flush
			p.flushC <- struct{}{}

		case <-p.flusherStopC:
			return
		}
	}
}

type stack struct {
	sync.Mutex
	elems []interface{}
}

func (s *stack) Push(x interface{}) {
	s.Lock()
	defer s.Unlock()
	s.elems = append(s.elems, x)

}

func (s *stack) Pop() interface{} {
	s.Lock()
	defer s.Unlock()
	if len(s.elems) == 0 {
		return nil
	}
	size := len(s.elems)
	item := s.elems[size-1]
	s.elems = s.elems[:size-1]
	return item
}
