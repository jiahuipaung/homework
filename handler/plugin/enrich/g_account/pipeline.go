package g_account

import (
	"context"
	"errors"
	"time"

	"github.com/flashcatcloud/fc-stash/config"
	"github.com/flashcatcloud/fc-stash/extract"
	"github.com/flashcatcloud/fc-stash/handler/plugin"
	"github.com/flashcatcloud/fc-stash/input"
	"github.com/flashcatcloud/fc-stash/sink"
	"github.com/flashcatcloud/fc-stash/types"
	"github.com/flashcatcloud/go-pkg/srv"
	"github.com/flashcatcloud/go-pkg/x/log"
	"github.com/flashcatcloud/go-pkg/x/redisx"
	"github.com/flashcatcloud/go-pkg/x/wg"
	"github.com/patrickmn/go-cache"
	"go.uber.org/zap"
)

// 日志enrich为期货做的定制化
// 词表是从日志流中读取的

type PipelineWorker struct {
	config.PluginPipelineConfig
	Queue  chan *types.LogEvent
	Input  *input.InputKafka
	Output *sink.ElasticOutput

	retry      *PipelineRetryWorker
	closed     bool
	localCache *cache.Cache
}

var (
	workers []*PipelineWorker
)

// 定制的日志提取器, 规则是硬编码的
var (
	dictext *extract.JsonPrune
	dataext *extract.JsonPrune
)

func StartPipelineWorker(cfgs []config.PluginPipelineConfig) error {
	ctx := srv.NewCtx(log.CommonLogger)
	// 已启动, 不允许重复调用
	if len(workers) > 0 {
		return nil
	}
	var err error
	dictext, err = plugin.ParseJsonExtractorByRuleStr(dictrule)
	if err != nil {
		return errors.New("dict extractor init failed:" + err.Error())
	}
	dataext, err = plugin.ParseJsonExtractorByRuleStr(datarule)
	if err != nil {
		return errors.New("data extractor init failed:" + err.Error())
	}

	for i := range cfgs {
		conf := cfgs[i]
		if err := plugin.CheckAndSetDefaultValue(ctx, &conf); err != nil {
			return err
		}
		kafka := &input.InputKafka{
			Brokers: conf.Input.Brokers,
			Topic:   conf.Input.Topic,
			GroupID: conf.Input.GroupID,
		}
		if err := kafka.Init(ctx); err != nil {
			return err
		}
		es := &sink.ElasticOutput{
			Index:               conf.Output.IndexFormat,
			TimestampIndexField: LogKeyDate,
			Servers:             conf.Output.Servers,
			Username:            conf.Output.Username,
			Password:            conf.Output.Password,
			SkipTlsVerify:       conf.Output.SkipTlsVerify,
		}
		if err := es.Init(ctx, 2); err != nil {
			return err
		}
		workers = append(workers, &PipelineWorker{
			PluginPipelineConfig: conf,
			Input:                kafka,
			Output:               es,
		})
	}
	for i := range workers {
		if err := workers[i].Start(ctx); err != nil {
			workers[i].Stop(ctx)
			return err
		}
	}
	return nil
}

func StopPipelineWorker() {
	ctx := srv.NewCtx(log.CommonLogger)
	if len(workers) == 0 {
		return
	}
	for i := range workers {
		workers[i].Stop(ctx)
	}
}

func (p *PipelineWorker) Start(ctx *srv.Context) error {
	if p.Input == nil {
		return errors.New("nil pointer: kafka input")
	}
	if p.Output == nil {
		return errors.New("nil pointer: es output")
	}
	var err error
	_, err = redisx.GetByEnv(ctx, p.RedisEnv)
	if err != nil {
		return err
	}
	p.localCache = cache.New(time.Minute*5, time.Hour)

	p.Queue = make(chan *types.LogEvent, 10) // queue init, 设置一个比较小的缓存区
	if err := p.Output.Start(ctx, ctx.Logger().Logger); err != nil {
		return err
	}
	p.retry = NewPipelineRetryWorker(time.Second*5, 3) // 5秒轮询一次, 最多3次
	go p.StartRetryWorker(ctx)

	newCtx := srv.NewCtx(
		ctx.Logger().Clone().
			WithBF(context.TODO()).
			With(zap.String("worker", "enrich_gaccount_log_pipieline")),
	)
	go func(ctx *srv.Context, pipe *PipelineWorker) {
		extractWG := wg.NewWG(p.NumOfWorker) // 并发数, 可配置
		for e := range p.Queue {
			extractWG.WrapFunc(p.Handle, ctx, e)
		}
	}(newCtx, p)

	// 最后启动input, 避免下游没有准备好导致的丢数据
	if err := p.Input.Start(ctx, ctx.Logger().Logger, p.Queue); err != nil {
		return err
	}
	return nil
}

func (p *PipelineWorker) Stop(ctx *srv.Context) {
	if p.Input != nil {
		p.Input.Stop(ctx)
	}
	if !p.closed {
		if p.Queue != nil {
			close(p.Queue)
		}
		p.StopRetryWorker(ctx)
		p.closed = true
	}
	if p.Output != nil {
		p.Output.Stop(ctx)
	}
}
