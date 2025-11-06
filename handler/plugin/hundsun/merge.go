package hundsun

import (
	"encoding/json"
	"errors"
	"math/rand"
	"strconv"
	"sync"
	"time"

	"github.com/flashcatcloud/fc-stash/types"
	"github.com/flashcatcloud/go-pkg/srv"
	"github.com/flashcatcloud/go-pkg/utils"
	"github.com/flashcatcloud/go-pkg/x/prom_exporter"
	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"
)

var (
	MultiMatchFuncIDs = []string{ // 特殊的功能号, 一条request对应多条response, 因此request只能自动过期
		"111",
		"112",
		"113",
	}
)

type RequestMergeItem struct {
	*RequestExtractedLog
	NextUnix int64 `json:"next"`
	Retry    int   `json:"retry"`
}

type PipelineMerge struct {
	sync.RWMutex
	sync.WaitGroup
	Size    int
	TTL     int
	Ticker  *time.Ticker
	Queue   map[string]*RequestMergeItem
	stop    chan struct{}
	stopped bool
}

func NewPipelineMerge(size int, ttl int, tick time.Duration) *PipelineMerge {
	return &PipelineMerge{
		Size:   size,
		TTL:    ttl,
		Ticker: time.NewTicker(tick),
		Queue:  make(map[string]*RequestMergeItem),
		stop:   make(chan struct{}),
	}
}

func (p *PipelineWorker) StartMerge(ctx *srv.Context) {
	p.merge.WaitGroup.Add(1)

	for {
		select {
		case <-p.merge.stop:
			p.FlushMerge(ctx, true)
			p.merge.WaitGroup.Done()
			return

		case <-p.merge.Ticker.C:
			p.FlushMerge(ctx, false)
		}
	}
}

func (p *PipelineWorker) StopMerge(ctx *srv.Context) {
	p.merge.stopped = true
	p.merge.stop <- struct{}{}
}

func (p *PipelineWorker) SetMergeCache(ctx *srv.Context, key string, log *RequestExtractedLog) error {
	p.merge.Lock()
	defer p.merge.Unlock()
	if p.merge.stopped {
		return errors.New("merge already stopped")
	}
	// 历史的没有清理掉, 新进来的只能报错
	// 让上游去等待
	if len(p.merge.Queue) >= p.merge.Size {
		return errors.New("merge queue overflow")
	}
	p.merge.Queue[key] = &RequestMergeItem{
		RequestExtractedLog: log,
		NextUnix:            time.Now().Unix() + 1,
		Retry:               1,
	}
	return nil
}

func (p *PipelineWorker) FlushMerge(ctx *srv.Context, all bool) {
	var keys []string
	var items []*RequestMergeItem
	now := time.Now().Unix()
	p.merge.RLock()
	for key, item := range p.merge.Queue {
		keys = append(keys, key)
		items = append(items, item)
	}
	p.merge.RUnlock()

	var todeletes []string
	for i := range keys {
		resp := items[i].RequestExtractedLog
		must := false
		// 已过期, 必须写入ES
		if now-resp.Date.Unix() >= int64(p.merge.TTL) {
			must = true
		}
		// 全量写入, worker即将停止
		if all {
			must = true
		}
		if !must {
			// 等待下一次执行
			// 每次匹配不到都会延后一段时间执行
			if now < items[i].NextUnix {
				continue
			}
		}
		req, found := p.FindRequestLog(ctx, resp)
		if !found {
			if !must {
				items[i].Retry += 1
				// 每次匹配不到都会延后一段时间执行
				items[i].NextUnix = now + int64(rand.Intn(10))
				continue
			}
		}
		prom_exporter.Inc("plugin_hundsun_find_request_log", map[string]string{"retry": strconv.Itoa(items[i].Retry)})
		merged := FormatMergedLog(ctx, req, resp)
		if len(merged) > 0 {
			if err := p.Output.Sink(ctx, merged); err != nil {
				ctx.Logger().Warn("sink merged log warning", zap.Reflect("log", merged),
					zap.Error(err))
			}
		}
		todeletes = append(todeletes, keys[i])
	}
	if len(todeletes) > 0 {
		p.merge.Lock()
		for _, key := range todeletes {
			delete(p.merge.Queue, key)
		}
		p.merge.Unlock()
	}
}

// 有两个地方会调用
// 1. response日志从kafka消费到的时候, 先尝试匹配一次, 如果匹配成功则直接写入ES
// 2. 匹配失败后, 加入队列, 不断轮询尝试, 直到匹配到或超时
func (p *PipelineWorker) FindRequestLog(ctx *srv.Context, resp *RequestExtractedLog,
) (*RequestExtractedLog, bool) {
	key := EncodeCachedKey(ctx, resp, p.WorkerID)
	remove := true
	if utils.StringArrayContains(MultiMatchFuncIDs, resp.FuncNo) {
		remove = false
	}
	if p.localCache != nil {
		prom_exporter.Inc("plugin_hundsun_local_cache_search", map[string]string{"code": "all"})
		req, found := p.localCache.GetAndRemove(key, remove)
		if found {
			prom_exporter.Inc("plugin_hundsun_local_cache_search", map[string]string{"code": "hit"})
			return req, true
		}
		prom_exporter.Inc("plugin_hundsun_local_cache_search", map[string]string{"code": "miss"})
	}
	if p.remoteCache != nil {
		prom_exporter.Inc("plugin_hundsun_remote_cache_search", map[string]string{"code": "all"})
		data, err := p.remoteCache.Get(ctx, key).Result()
		if err != nil {
			if err == redis.Nil {
				prom_exporter.Inc("plugin_hundsun_remote_cache_search", map[string]string{"code": "miss"})
			} else {
				prom_exporter.Inc("plugin_hundsun_remote_cache_search", map[string]string{"code": "error"})
			}
			return nil, false
		}
		prom_exporter.Inc("plugin_hundsun_remote_cache_search", map[string]string{"code": "hit"})
		if remove {
			_ = p.remoteCache.Del(ctx, key).Err()
		}

		if len(data) > 0 {
			req := new(RequestCachedLog)
			if err := json.Unmarshal([]byte(data), req); err == nil {
				return &RequestExtractedLog{
					Type:        LogTypeRequest,
					Date:        time.UnixMilli(req.UnixMilli),
					FundAccount: req.FundAccount,
					Tag:         req.Tag,
					Data:        req.Data,
				}, true
			}
		}
	}
	return nil, false
}

// 与 RequestMergedLog 保持一致
func FormatMergedLog(ctx *srv.Context, req, resp *RequestExtractedLog) types.ExtractedLog {
	if resp == nil {
		return nil
	}
	ret := make(types.ExtractedLog)
	ret["date"] = resp.Date
	ret["agent_hostname"] = resp.AgentHostname
	ret["req_no"] = resp.ReqNo
	ret["func_no"] = resp.FuncNo
	if len(resp.FundAccount) > 0 {
		ret["fund_account"] = resp.FundAccount
	}
	ret["error_no"] = resp.ErrorNo
	if len(resp.ErrorInfo) > 0 {
		ret["error_info"] = resp.ErrorInfo
	}
	ret["response_tag"] = resp.Tag
	ret["response_data"] = resp.Data
	if req != nil {
		prom_exporter.Inc("plugin_hundsun_merge_with_req")
		ret["date"] = req.Date
		if len(req.FundAccount) > 0 {
			ret["fund_account"] = req.FundAccount
		}
		ret["request_tag"] = req.Tag
		ret["request_data"] = req.Data
		ret["request_time"] = resp.Date.UnixMilli() - req.Date.UnixMilli()
	} else {
		prom_exporter.Inc("plugin_hundsun_merge_without_req")
		// 找不到req, 先把response丢掉
		return nil
	}
	reqkeys, _ := parser.ParseModeTable(req.Data)
	if len(reqkeys) > 0 {
		// 28017 尝试从request.account_content中提取
		if resp.FuncNo == "28017" &&
			len(req.FundAccount) == 0 &&
			len(resp.FundAccount) == 0 {
			ac, found := reqkeys[LogKeyAccountContent]
			if found && len(ac) == 1 && len(ac[0]) > 0 {
				ret["fund_account"] = ac[0]
			}
		}
		// reqkeys 全部打平到结果中, 并写入ES
		for key, value := range reqkeys {
			if len(value) != 1 {
				continue
			}
			// 系统内置的字段不允许覆盖
			if utils.StringArrayContains(musts, key) ||
				utils.StringArrayContains(shoulds, key) {
				continue
			}
			ret[key] = value[0]
		}
	}
	return ret
}
