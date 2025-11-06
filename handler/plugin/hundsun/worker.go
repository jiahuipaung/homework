package hundsun

import (
	"errors"
	"strings"
	"time"

	"github.com/flashcatcloud/fc-stash/types"
	"github.com/flashcatcloud/go-pkg/srv"
	"github.com/flashcatcloud/go-pkg/x/prom_exporter"
	"go.uber.org/zap"
)

// 恒生电子的客户请求日志定制化的处理逻辑
// 1. 通过redis缓存request日志, 进而实现与response的一一对应
// 2. request匹配不到通过redis的TTL机制丢掉, response匹配不到仍然写入ES
// 3. 前置过滤: 包含关键字 agent_hostname(categraf采集) 和 CLIENT_REQ(恒生标准日志)
// 4. 对每一条日志, 做JSON反序列化, 提取agent_hostname字段
// 5. 对每一条日志, 从message中提取 request/response字段、no字段、funcNo字段、时间字段(并转为unix_milli)
// 6. request日志直接写入redis, rediskey由 worker_id + no + agent_hostname 生成
// 6.1. 从request日志提取fund_account(如果有), 一并写入redis
// 6.2. TTL=merge_window
// 7. response日志查找匹配的request日志, rediskey是同一个
// 7.1. 提取fund_account字段, 如果request中不存在
// 7.2. 计算request_time
// 7.3. 补全error_no=0, 如果有则复用已有的
// 7.4. tag字段/data字段均保留, 分别加上request/response前缀
// 7.5. 如果匹配不到, 写入一个循环队列, 循环匹配直到超时

// 写入ES中的日志, key要保持完整
type RequestMergedLog struct {
	Date          time.Time `json:"date"`
	AgentHostname string    `json:"agent_hostname"`
	ReqNo         string    `json:"req_no"`
	FuncNo        string    `json:"func_no"`
	FundAccount   string    `json:"fund_account"`
	RequestTime   float64   `json:"request_time"`  // 毫秒
	RequestTag    string    `json:"request_tag"`   // request
	RequestData   string    `json:"request_data"`  // request
	ResponseTag   string    `json:"response_tag"`  // response
	ResponseData  string    `json:"response_data"` // response
	ErrorNo       string    `json:"error_no"`      // 如果不存在则写入空字符串
	ErrorInfo     string    `json:"error_info"`

	RequestFormat  MultilineData `json:"request_format"`
	ResponseFormat MultilineData `json:"response_format"`
}

type RequestExtractedLog struct {
	Type          string    `json:"type"` // one of request/response
	Date          time.Time `json:"date"`
	AgentHostname string    `json:"agent_hostname"`
	ReqNo         string    `json:"req_no"`
	FuncNo        string    `json:"func_no"`
	FundAccount   string    `json:"fund_account"`
	Tag           string    `json:"tag"`
	Data          string    `json:"data"`
	ErrorNo       string    `json:"error_no"`
	ErrorInfo     string    `json:"error_info"`
}

// 写入redis中的日志, 尽量缩写
type RequestCachedLog struct {
	UnixMilli   int64  `json:"u_mi"`
	FundAccount string `json:"acc"`
	Tag         string `json:"tag"`
	Data        string `json:"data"`
}

func (p *PipelineWorker) Handle(ctx *srv.Context, event *types.LogEvent) {
	if event == nil || event.Message == nil {
		return
	}
	extlog, err := ExtractLog(ctx, event)
	if err != nil {
		ctx.Logger().Debug("extract log warning", zap.Reflect("message", event.Message),
			zap.Error(err))
		return
	}
	key := EncodeCachedKey(ctx, extlog, p.WorkerID)
	if extlog.Type == LogTypeRequest {
		p.localCache.Set(key, extlog)
		if err := p.WriteToCacheBackend(ctx, extlog); err != nil {
			ctx.Logger().Warn("write to redis failed", zap.Reflect("log", extlog), zap.Error(err))
		}
	}
	if extlog.Type == LogTypeResponse {
		req, found := p.FindRequestLog(ctx, extlog)
		if found {
			prom_exporter.Inc("plugin_hundsun_find_request_log", map[string]string{"retry": "0"})
			merged := FormatMergedLog(ctx, req, extlog)
			if err := p.Output.Sink(ctx, merged); err != nil {
				ctx.Logger().Warn("sink merged log warning", zap.Reflect("merged", merged),
					zap.Error(err))
			}
		} else {
			// 失败不重试, 直接丢掉
			if err := p.SetMergeCache(ctx, key, extlog); err != nil {
				prom_exporter.Inc("plugin_hundsun_set_merge_cache_error")
				ctx.Logger().Warn("set merge cache warning", zap.Reflect("log", extlog),
					zap.Error(err))
			}
		}
	}
}

func ExtractLog(ctx *srv.Context, event *types.LogEvent) (*RequestExtractedLog, error) {
	msgstr, ok := event.Message.(string)
	if !ok {
		return nil, errors.New("nil message")
	}
	var result types.ExtractedLog
	var err error
	if strings.Contains(msgstr, "\\\\n") { // 旧版的日志, 待categraf统一升级后, 下掉这里的逻辑
		result, err = extractor2.Handle(ctx, event)
		if err != nil {
			return nil, err
		}
	} else {
		result, err = extractor.Handle(ctx, event)
		if err != nil {
			return nil, err
		}
	}
	ret := new(RequestExtractedLog)
	for _, key := range musts {
		value, found := result[key]
		if !found {
			return nil, ErrorExtractFieldMissingFn(key)
		}
		if key == LogKeyDate {
			vtime, ok := value.(time.Time)
			if !ok {
				return nil, ErrorExtractFieldTypeNotMatchFn(key)
			}
			ret.Date = vtime
		} else {
			vstr, ok := value.(string)
			if !ok {
				return nil, ErrorExtractFieldTypeNotMatchFn(key)
			}
			// 如果有新增的字段, 依次添加
			switch key {
			case LogKeyAgentHostname:
				ret.AgentHostname = vstr

			case LogKeyReqNo:
				ret.ReqNo = vstr

			case LogKeyFuncNo:
				ret.FuncNo = vstr

			case LogKeyType:
				ret.Type = vstr

			case LogKeyTag:
				ret.Tag = vstr

			case LogKeyData:
				// 特殊处理, 把 '\\\\n转化为回车'
				vstr = strings.ReplaceAll(vstr, "\\\\n", "\n")
				vstr = strings.ReplaceAll(vstr, "\\n", "\n")
				ret.Data = vstr
			}
		}
	}
	for _, key := range shoulds {
		value, found := result[key]
		if found {
			vstr, ok := value.(string)
			if ok {
				switch key {
				case LogKeyFundAccount:
					ret.FundAccount = vstr

				case LogKeyErrorNo:
					ret.ErrorNo = vstr

				case LogKeyErrorInfo:
					ret.ErrorInfo = vstr
				}
			}
		}
	}
	return ret, nil
}
