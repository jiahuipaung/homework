package g_account

import (
	"fmt"
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

type RetryLogData struct {
	Log      types.ExtractedLog
	Hostname string
	Ref      string
	Count    int
	tosink   bool
}

type PipelineRetryWorker struct {
	sync.RWMutex
	maxRetry int
	Ticker   *time.Ticker
	Cached   map[string]*RetryLogData

	stop    chan struct{}
	stopped bool
}

func NewPipelineRetryWorker(tick time.Duration, retry int) *PipelineRetryWorker {
	return &PipelineRetryWorker{
		maxRetry: retry,
		Ticker:   time.NewTicker(tick),
		Cached:   make(map[string]*RetryLogData),
		stop:     make(chan struct{}),
	}
}

// 单线程
func (p *PipelineWorker) StartRetryWorker(ctx *srv.Context) {
	for {
		select {
		case <-p.retry.stop:
			return

		case <-p.retry.Ticker.C:
			p.retry.Lock()
			if len(p.retry.Cached) == 0 {
				p.retry.Unlock()
				continue
			}

			copyCache := make(map[string]*RetryLogData)
			for k, v := range p.retry.Cached {
				copyCache[k] = &RetryLogData{
					Hostname: v.Hostname,
					Ref:      v.Ref,
					Count:    v.Count + 1,
					Log:      v.Log, // map的指针引用
				}
			}
			p.retry.Unlock()

			for _, v := range copyCache {
				refToID, _ := GetDictGAccountIDByRef(ctx, v.Hostname, v.Ref)
				// 否则直接写到下游
				if len(refToID) > 0 {
					// AccountID AccountRef 转化为整型
					v.Log[LogKeyAccountID], _ = strconv.ParseInt(refToID, 10, 64)
					v.Log[LogKeyAccountIDStr] = refToID
					v.tosink = true
					prom_exporter.Inc("logx_enrich_g_account_id_find")
				} else if v.Count > p.retry.maxRetry {
					v.tosink = true
					prom_exporter.Inc("logx_enrich_g_account_id_empty")
				}
			}

			p.retry.Lock()
			for k, v := range copyCache {
				if v.tosink {
					// AccountRef 转化为整型
					v.Log[LogKeyAccountRef], _ = strconv.ParseInt(v.Ref, 10, 64)
					v.Log[LogKeyAccountRefStr] = v.Ref
					v.Log[LogKeyOrderStatusStr] = fmt.Sprintf("%v", v.Log[LogKeyOrderStatus])
					delete(p.retry.Cached, k)
				} else {
					cached, found := p.retry.Cached[k]
					if found {
						cached.Count = v.Count
					}
				}
			}
			prom_exporter.IncN("logx_enrich_g_account_cache_left_size", len(p.retry.Cached))
			p.retry.Unlock()

			for _, v := range copyCache {
				if v.tosink {
					if err := p.Output.Sink(ctx, v.Log); err != nil {
						ctx.Logger().Warn("sink merged data warning", zap.Reflect("data", v.Log),
							zap.Error(err))
					}
				}
			}
		}
	}
}

func (p *PipelineWorker) StopRetryWorker(ctx *srv.Context) {
	p.retry.stopped = true
	p.retry.stop <- struct{}{}
}

func (p *PipelineWorker) AddToRetryWorker(ctx *srv.Context,
	hostname, ref string, logs types.ExtractedLog) {
	uuid := utils.NewUuid()

	p.retry.Lock()
	defer p.retry.Unlock()
	p.retry.Cached[uuid] = &RetryLogData{Log: logs, Hostname: hostname, Ref: ref, Count: 0}
	prom_exporter.Inc("logx_enrich_g_account_cache_count")
}

func (p *PipelineWorker) Handle(ctx *srv.Context, event *types.LogEvent) {
	if event == nil || event.Message == nil {
		return
	}

	// 读取词表, 并更新词表
	dictok, hostname, ref, ID, t, _ := extractDictLog(ctx, event)
	if dictok {
		if err := SetDictGAccountRefToID(ctx, hostname, ref, ID, t,
			time.Second*time.Duration(p.AccountDictExpireSeconds)); err != nil {
			ctx.Logger().Warn("extract logy set ref dict warning", zap.Reflect("message", event.Message),
				zap.String("hostname", hostname),
				zap.String("ref", ref), zap.String("ID", ID),
				zap.Error(err))
		}
		return
	}
	// 数据日志
	items, err := dataext.Handle(ctx, event)
	if err != nil {
		ctx.Logger().Debug("extract log by data rule warning", zap.Reflect("message", event.Message),
			zap.Error(err))
		return
	}
	if len(items) > 0 {
		value, found := items[LogKeyAccountRef]
		if !found {
			return
		}
		refstr, assert := value.(string)
		if !assert {
			return
		}
		value1, found := items[LogKeyAgentHostname]
		if !found {
			return
		}
		hoststr, assert := value1.(string)
		if !assert {
			return
		}
		refToID, err := GetDictGAccountIDByRef(ctx, hoststr, refstr)
		if err != nil && err == redis.Nil { // 没找到, append to retry worker, 循环
			p.AddToRetryWorker(ctx, hoststr, refstr, items)
			return
		}
		// 否则直接写到下游
		if len(refToID) > 0 {
			prom_exporter.Inc("logx_enrich_g_account_id_find")
			// AccountID 转化为整型
			// 并保留字符串
			items[LogKeyAccountID], _ = strconv.ParseInt(refToID, 10, 64)
			items[LogKeyAccountIDStr] = refToID
		}
		// AccountRef 转化为整型
		items[LogKeyAccountRef], _ = strconv.ParseInt(refstr, 10, 64)
		items[LogKeyAccountRefStr] = refstr
		items[LogKeyOrderStatusStr] = fmt.Sprintf("%v", items[LogKeyOrderStatus])
		if err := p.Output.Sink(ctx, items); err != nil {
			ctx.Logger().Warn("sink merged data warning", zap.Reflect("data", items),
				zap.Error(err))
		}
	}
}

// 缓存5分钟, 减轻redis压力
func (p *PipelineWorker) CacheGetDictGAccountIDByRef(ctx *srv.Context, hostname, ref string) (string, error) {
	cached, found := p.localCache.Get(ref)
	if found {
		return cached.(string), nil
	}
	refToID, err := GetDictGAccountIDByRef(ctx, hostname, ref)
	if err == nil {
		p.localCache.Set(ref, refToID, time.Minute*5)
		return refToID, nil
	}
	return "", err
}

func extractDictLog(ctx *srv.Context, event *types.LogEvent,
) (ok bool, hostname string, ref string, ID string, t time.Time, ext types.ExtractedLog) {
	items, err := dictext.Handle(ctx, event)
	if err != nil {
		ctx.Logger().Debug("extract log by dict rule warning", zap.Reflect("message", event.Message),
			zap.Error(err))
		return
	}
	if len(items) > 0 {
		for key, value := range items {
			if key == LogKeyDate {
				vtime, assert := value.(time.Time)
				if !assert {
					return
				}
				t = vtime
			} else if key == LogKeyAccountRef || key == LogKeyAccountID ||
				key == LogKeyAgentHostname {
				vstr, assert := value.(string)
				if !assert {
					return
				}
				if len(vstr) == 0 {
					return
				}
				if key == LogKeyAccountRef {
					ref = vstr
				} else if key == LogKeyAccountID {
					ID = vstr
				} else if key == LogKeyAgentHostname {
					hostname = vstr
				}
			}

		}
	}
	ok = len(hostname) > 0 && len(ref) > 0 && len(ID) > 0 && t.Unix() > 0
	return
}
