package victorialogs

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"

	"github.com/flashcatcloud/fc-stash/types"
	"github.com/flashcatcloud/fc-stash/utils"
	"github.com/flashcatcloud/go-pkg/x/prom_exporter"
)

const (
	DefaultInitialBackoff = 200 * time.Millisecond
	DefaultMaxBackoff     = 15 * time.Second
	DefaultMaxRetries     = 10 // 默认最大重试次数
	DefaultNumOfWorkers   = 1
	DefaultBatchActions   = 1000
	DefaultBatchSize      = 5 << 20 // 5MB
	DefaultFlushInterval  = 10 * time.Second
	HealthCheckInterval   = 2 * time.Second
	HealthCheckTimeout    = 1 * time.Second
)

// EndpointStatus 存储单个 VMLogs 节点的状态
type EndpointStatus struct {
	URL       string
	IsHealthy bool

	mutex        sync.RWMutex
	retryBackoff time.Duration // 当前退避时间
	retryAfter   time.Time     // 可以再次尝试的时间点
	failCount    int
}

// MarkAsHealthy 将节点标记为健康，并重置退避状态
func (e *EndpointStatus) MarkAsHealthy() {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	if !e.IsHealthy {
		if utils.Logger != nil {
			utils.Logger.Sugar().Info("endpoint is healthy again", zap.String("url", e.URL))
		}
	}
	e.IsHealthy = true
	e.failCount = 0
	e.retryBackoff = DefaultInitialBackoff
}

// MarkAsUnhealthy 将节点标记为不健康，并采用指数退避算法增加下次重试的等待时间
func (e *EndpointStatus) MarkAsUnhealthy(maxRetryBackoff time.Duration) {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	if e.IsHealthy {
		if utils.Logger != nil {
			utils.Logger.Sugar().Warnf("marking endpoint as unhealthy", zap.String("url", e.URL))
		}
	}
	e.IsHealthy = false
	e.failCount++
	e.retryAfter = time.Now().Add(e.retryBackoff)
	e.retryBackoff *= 2 // 指数增加退避时间
	if e.retryBackoff > maxRetryBackoff {
		e.retryBackoff = maxRetryBackoff
	}
}

// CanTry 判断是否可以尝试向该节点发送数据
func (e *EndpointStatus) CanTry() bool {
	e.mutex.RLock()
	defer e.mutex.RUnlock()
	// 如果节点是健康的，或者已到达可以重试的时间点，则返回 true
	return e.IsHealthy || time.Now().After(e.retryAfter)
}

type BatchProcessor struct {
	config    *VMLogsOutput
	client    *http.Client
	requestsC chan types.ExtractedLog
	requests  []cachedLog
	endpoints []*EndpointStatus

	batchActions      int
	batchSize         int
	flushInterval     time.Duration
	maxBackoff        time.Duration // 失败重试的最大退避时间
	maxRetries        int           // 发送失败时的最大重试次数
	numWorkers        int
	workWg            *utils.WaitGroupWrapper
	sizeInBytes       int64
	sizeInBytesCursor int

	flushC           chan struct{}
	flushStopC       chan struct{}
	healthCheckStopC chan struct{} // 用于停止健康检查器通道

	startedMu sync.Mutex
	started   bool
	counter   uint64 // 用于轮询的计数器

	// 反压机制所需的状态
	stateMu       sync.Mutex // 保护 sinkIsBlocked
	stateCond     *sync.Cond // 用于高效挂起和唤醒 work 协程
	sinkIsBlocked bool       // 标志下游 sink 是否已阻塞
}

type cachedLog struct {
	log   types.ExtractedLog
	bytes []byte
}

type VMLogsHTTPError struct {
	StatusCode int
	Status     string
	Body       []byte
}

func NewBatchProcessor(output *VMLogsOutput) *BatchProcessor {
	bp := &BatchProcessor{
		config:           output,
		numWorkers:       DefaultNumOfWorkers,
		client:           output.Client,
		batchActions:     DefaultBatchActions,
		batchSize:        DefaultBatchSize,
		flushInterval:    DefaultFlushInterval,
		maxBackoff:       DefaultMaxBackoff,
		maxRetries:       DefaultMaxRetries,
		requests:         make([]cachedLog, 0),
		endpoints:        make([]*EndpointStatus, 0, len(output.Endpoints)),
		healthCheckStopC: make(chan struct{}),
	}

	// 初始化用于反压的条件变量
	bp.stateCond = sync.NewCond(&bp.stateMu)

	for _, epStr := range output.Endpoints {
		bp.endpoints = append(bp.endpoints, &EndpointStatus{
			URL:          epStr,
			IsHealthy:    true,
			retryBackoff: DefaultInitialBackoff,
		})
	}
	return bp
}

func (p *BatchProcessor) Start(ctx context.Context) error {
	p.startedMu.Lock()
	defer p.startedMu.Unlock()
	if p.started {
		return nil
	}
	if p.numWorkers < 1 {
		p.numWorkers = 1
	}

	p.requestsC = make(chan types.ExtractedLog)
	// 参考es bulk的单生产者+多消费者机制
	p.workWg = utils.NewWG(p.numWorkers)
	go p.work(ctx)

	// 起一个goroutine定时flush
	if int64(p.flushInterval) > 0 {
		p.flushC = make(chan struct{})
		p.flushStopC = make(chan struct{})
		go p.flush(p.flushInterval)
	}

	// 启动健康检查
	go p.healthChecker()

	p.started = true

	return nil
}

func (p *BatchProcessor) Stop() error {
	p.startedMu.Lock()
	defer p.startedMu.Unlock()
	if !p.started {
		return nil
	}

	// 停止后台的协程
	if p.flushStopC != nil {
		close(p.flushStopC)
	}
	if p.healthCheckStopC != nil {
		close(p.healthCheckStopC)
	}

	// 关闭数据通道
	close(p.requestsC)
	p.started = false
	// 等待所有正在处理的任务完成
	p.workWg.Wait()
	return nil
}

func (p *BatchProcessor) Add(log types.ExtractedLog) {
	p.requestsC <- log
}

func (p *BatchProcessor) flush(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			p.flushC <- struct{}{}
		case <-p.flushStopC:
			return
		}
	}
}

// 判断batch是否满足commit条件
func (p *BatchProcessor) commitRequired() bool {
	if p.batchSize > 0 && p.sizeInBytes >= int64(p.batchSize) {
		return true
	}
	if p.batchActions > 0 && p.NumberOfActions() >= p.batchActions {
		return true
	}
	return false
}

// work 主要的消费协程，负责从上游接收数据并打包
func (p *BatchProcessor) work(ctx context.Context) {
	wrapFn := func(ctx context.Context, reqs []cachedLog) {
		prom_exporter.Inc("log_event_output_vmlogs_batch_processor_worker")
		if len(reqs) == 0 {
			return
		}
		if err := p.commit(ctx, reqs); err != nil {
			if utils.Logger != nil {
				utils.Logger.Sugar().Warnf("vmlogs: batch processor worker failed to commit logs")
			}
		}
	}

	for {
		// --- 反压机制 ---
		// 如果 sink 完全被阻塞, 则在此处等待, 直到healthCheck唤醒
		p.stateMu.Lock()
		for p.sinkIsBlocked {
			if utils.Logger != nil {
				utils.Logger.Sugar().Warnf("vmlogs sink is blocked, pausing consumption")
			}
			p.stateCond.Wait()
		}
		p.stateMu.Unlock()

		select {
		case req, open := <-p.requestsC:
			if !open {
				if p.NumberOfActions() > 0 {
					p.MarshalAndEstimateSizeInBytes()
					reqs := p.dumpAndReset()
					p.workWg.WrapFunc(wrapFn, ctx, reqs)
				}
				return
			}
			p.requests = append(p.requests, cachedLog{log: req})
			p.MarshalAndEstimateSizeInBytes()
			if p.commitRequired() {
				reqs := p.dumpAndReset()
				p.workWg.WrapFunc(wrapFn, ctx, reqs)
			}
		case <-p.flushC:
			if p.NumberOfActions() > 0 {
				p.MarshalAndEstimateSizeInBytes()
				reqs := p.dumpAndReset()
				p.workWg.WrapFunc(wrapFn, ctx, reqs)
			}
		}
	}
}

func (p *BatchProcessor) dumpAndReset() (reqs []cachedLog) {
	reqs = p.requests
	p.requests = make([]cachedLog, 0)
	p.sizeInBytes = 0
	p.sizeInBytesCursor = 0
	return
}

func (p *BatchProcessor) commit(ctx context.Context, reqs []cachedLog) error {
	if len(reqs) == 0 {
		return nil
	}
	payloadBuf := &bytes.Buffer{}
	p.preparePayload(reqs, payloadBuf)
	if payloadBuf.Len() == 0 {
		return nil
	}

	payloadBytes := payloadBuf.Bytes()
	var lastErr error
	for attempt := 0; attempt < p.maxRetries; attempt++ {
		endpoint, err := p.nextEndpoint()
		if err != nil {
			// 没有可用的健康节点, 触发反压机制
			p.stateMu.Lock()
			if !p.sinkIsBlocked {
				if utils.Logger != nil {
					utils.Logger.Sugar().Warnf("no healthy vmlogs endpoints available, blocking sink")
				}
				p.sinkIsBlocked = true
			}
			p.stateMu.Unlock()

			time.Sleep(1 * time.Second)
			attempt-- // 不消耗重试次数
			continue
		}

		// 尝试发送数据
		lastErr = p.sendRequest(ctx, bytes.NewReader(payloadBytes), endpoint)

		if lastErr == nil {
			prom_exporter.IncN("log_event_output_vmlogs_batch_items", len(reqs), map[string]string{"code": "ok"})
			return nil // 发送成功
		}

		// 检查是否为不可重试的错误 (如 4xx)
		var httpErr *VMLogsHTTPError
		isRetryable := true
		if errors.As(lastErr, &httpErr) {
			if httpErr.StatusCode >= http.StatusBadRequest &&
				httpErr.StatusCode < http.StatusInternalServerError &&
				httpErr.StatusCode != http.StatusTooManyRequests {
				isRetryable = false
			}
		}

		if !isRetryable {
			if utils.Logger != nil {
				utils.Logger.Error("non-retriable error sending vmlogs request, dropping batch", zap.Error(lastErr))
			}
			prom_exporter.IncN("log_event_output_error", len(reqs), map[string]string{"code": "non_retryable_error", "target": "vmlogs"})
			return lastErr
		}

		// 对于可重试的错误, sendRequest 内部已将节点标记为不健康, 这里只需记录日志, 循环自动尝试下一个可用节点
		if utils.Logger != nil {
			utils.Logger.Warn("retrying vmlogs request after failure",
				zap.String("endpoint", endpoint.URL),
				zap.Int("current_attempt", attempt+1),
				zap.Int("max_attempts", p.maxRetries),
				zap.Error(lastErr))
		}
		prom_exporter.IncN("log_event_output_error", 1, map[string]string{"code": "retryable_error", "target": "vmlogs"})
	}

	// 如果循环结束后 lastErr 仍然不是 nil, 说明所有重试都失败了
	if utils.Logger != nil {
		utils.Logger.Error("failed to send vmlogs batch after max retries, dropping data",
			zap.Int("max_retries", p.maxRetries),
			zap.Error(lastErr))
	}
	prom_exporter.IncN("log_event_output_error", len(reqs), map[string]string{"code": "retries_exhausted", "target": "vmlogs"})

	return fmt.Errorf("failed to send batch after %d attempts: %w", p.maxRetries, lastErr)
}

func (p *BatchProcessor) preparePayload(reqs []cachedLog, buffer *bytes.Buffer) {
	for i := range reqs {
		if len(reqs[i].bytes) > 0 {
			buffer.Write(reqs[i].bytes)
			buffer.WriteByte('\n')
		}
	}
}

func (p *BatchProcessor) sendRequest(_ context.Context, payload io.Reader, endpoint *EndpointStatus) error {
	u, err := url.Parse(endpoint.URL)
	if err != nil {
		return fmt.Errorf("invalid endpoint URL: %w", err)
	}
	u.Path = "/insert/jsonline"

	q := u.Query()
	if len(p.config.StreamFields) > 0 {
		q.Set("_stream_fields", strings.Join(p.config.StreamFields, ","))
	}
	if p.config.TimeField != "" {
		q.Set("_time_field", p.config.TimeField)
	}
	if p.config.MsgField != "" {
		q.Set("_msg_field", p.config.MsgField)
	}
	u.RawQuery = q.Encode()

	req, err := http.NewRequest("POST", u.String(), payload)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/stream+json")
	if p.config.Username != "" || p.config.Password != "" {
		req.SetBasicAuth(p.config.Username, p.config.Password)
	}
	for k, v := range p.config.Headers {
		req.Header.Set(k, v)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		endpoint.MarkAsUnhealthy(p.maxBackoff)
		return fmt.Errorf("http client to vmlogs failed on endpoint %s: %w", endpoint.URL, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		// 无法读取body，但可能请求已成功，这里不立即标记为不健康
		return fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode >= http.StatusOK && resp.StatusCode < http.StatusMultipleChoices {
		endpoint.MarkAsHealthy()
		return nil
	}

	errForReturn := &VMLogsHTTPError{
		StatusCode: resp.StatusCode,
		Status:     resp.Status,
		Body:       body,
	}

	// 5xx errors and 429 Too Many Requests 认为是可重试的服务端问题
	if resp.StatusCode >= http.StatusInternalServerError || resp.StatusCode == http.StatusTooManyRequests {
		endpoint.MarkAsUnhealthy(p.maxBackoff)
	}

	return errForReturn
}

func (p *BatchProcessor) nextEndpoint() (*EndpointStatus, error) {
	numEndpoints := len(p.endpoints)
	if numEndpoints == 0 {
		return nil, errors.New("no endpoints available")
	}

	// 轮询所有节点两次以确保不会错过任何一个刚恢复的节点
	startIndex := int(atomic.LoadUint64(&p.counter) % uint64(numEndpoints))
	for i := 0; i < numEndpoints*2; i++ {
		nextIndex := (startIndex + i) % numEndpoints
		endpoint := p.endpoints[nextIndex]

		if endpoint.CanTry() {
			atomic.StoreUint64(&p.counter, uint64(nextIndex+1))
			return endpoint, nil
		}
	}
	return nil, errors.New("no healthy endpoints available, all in backoff period")
}

// healthChecker 是独立的后台协程, 作为“闹钟”来探测不健康的节点
func (p *BatchProcessor) healthChecker() {
	ticker := time.NewTicker(HealthCheckInterval)
	defer ticker.Stop()

	if utils.Logger != nil {
		utils.Logger.Sugar().Info("vmlogs health checker started")
	}

	for {
		select {
		case <-ticker.C:
			for _, endpoint := range p.endpoints {
				endpoint.mutex.RLock()
				isHealthy := endpoint.IsHealthy
				retryAfter := endpoint.retryAfter
				endpoint.mutex.RUnlock()

				if !isHealthy && time.Now().After(retryAfter) {
					// 节点的退避期已过, 可以进行一次健康探测
					p.checkEndpointHealth(endpoint)
				}
			}
		case <-p.healthCheckStopC:
			if utils.Logger != nil {
				utils.Logger.Sugar().Info("vmlogs health checker stopped")
			}
			return
		}
	}
}

// checkEndpointHealth 对单个节点执行一次轻量级的健康探测
func (p *BatchProcessor) checkEndpointHealth(endpoint *EndpointStatus) {
	u, err := url.Parse(endpoint.URL)
	if err != nil {
		return
	}
	u.Path = "/health"

	// 复用 p.client，使用ctx做健康检查的超时控制
	ctx, cancel := context.WithTimeout(context.Background(), HealthCheckTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "HEAD", u.String(), nil)
	if err != nil {
		return
	}
	resp, err := p.client.Do(req)
	if err != nil || (resp != nil && (resp.StatusCode < 200 || resp.StatusCode >= 400)) {
		// 仍然不健康, 无需操作, 等待下一个退避周期
		return
	} else {
		// 节点已恢复健康
		endpoint.MarkAsHealthy()

		// --- 解除反压 ---
		p.stateMu.Lock()
		if p.sinkIsBlocked {
			if utils.Logger != nil {
				utils.Logger.Sugar().Info("vmlogs sink is unblocked")
			}
			p.sinkIsBlocked = false
			p.stateCond.Broadcast()
		}
		p.stateMu.Unlock()
	}
}

// BatchProcessor 配置函数
func (p *BatchProcessor) Works(num int) *BatchProcessor {
	p.numWorkers = num
	return p
}

func (p *BatchProcessor) BatchActions(batchActions int) *BatchProcessor {
	p.batchActions = batchActions
	return p
}

func (p *BatchProcessor) BatchSize(batchSize int) *BatchProcessor {
	p.batchSize = batchSize
	return p
}
func (p *BatchProcessor) FlushInterval(interval time.Duration) *BatchProcessor {
	p.flushInterval = interval
	return p
}

func (p *BatchProcessor) NumberOfActions() int {
	return len(p.requests)
}

func (p *BatchProcessor) MaxRetryBackoff(maxRetryBackoff time.Duration) *BatchProcessor {
	p.maxBackoff = maxRetryBackoff
	return p
}

func (p *BatchProcessor) MarshalAndEstimateSizeInBytes() int64 {
	if p.sizeInBytesCursor == len(p.requests) {
		return p.sizeInBytes
	}
	for i := p.sizeInBytesCursor; i < len(p.requests); i++ {
		// 检查是否已序列化, bytes==nil，则执行序列化
		if p.requests[i].bytes == nil {
			jsonBytes, err := json.Marshal(p.requests[i].log)
			if err != nil {
				// 序列化失败，暂视为空；
				p.requests[i].bytes = []byte{}
			} else {
				p.requests[i].bytes = jsonBytes
			}
		}
		p.sizeInBytesCursor++
		p.sizeInBytes += int64(len(p.requests[i].bytes))
	}
	return p.sizeInBytes
}

func (e *VMLogsHTTPError) Error() string {
	return fmt.Sprintf("vmlogs server returned http status %s: %s", e.Status, string(e.Body))
}
