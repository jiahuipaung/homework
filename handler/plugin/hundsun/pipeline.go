package hundsun

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/flashcatcloud/fc-stash/handler/plugin"

	"github.com/flashcatcloud/fc-stash/config"
	"github.com/flashcatcloud/fc-stash/extract"
	"github.com/flashcatcloud/fc-stash/input"
	"github.com/flashcatcloud/fc-stash/sink"
	"github.com/flashcatcloud/fc-stash/types"
	"github.com/flashcatcloud/go-pkg/srv"
	"github.com/flashcatcloud/go-pkg/x/log"
	"github.com/flashcatcloud/go-pkg/x/redisx"
	"github.com/flashcatcloud/go-pkg/x/wg"
	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"
)

type PipelineWorker struct {
	config.PluginPipelineConfig
	Queue  chan *types.LogEvent
	Input  *input.InputKafka
	Output *sink.ElasticOutput
	closed bool

	// 缓存request日志, 优先走本地缓存, 统计本地命中概率
	localCache *FixedSizeRandomDropCache
	// 缓存request日志, 分布式的中心端
	remoteCache redis.Cmdable

	// 缓存response日志, 等待合并request日志
	merge *PipelineMerge
}

var (
	workers []*PipelineWorker
)

// 定制的日志提取器, 规则是硬编码的
var (
	extractor  = new(extract.JsonPrune)
	extractor2 = new(extract.JsonPrune)
)

// 目前看只有一个pipe
func StartPipelineWorker(cfgs []config.PluginPipelineConfig) error {
	ctx := srv.NewCtx(log.CommonLogger)
	// 已启动, 不允许重复调用
	if len(workers) > 0 {
		return nil
	}

	if err := json.Unmarshal([]byte(settings), extractor); err != nil {
		return errors.New("extractor init failed:" + err.Error())
	}
	if err := extractor.Compile(); err != nil {
		return errors.New("extractor init failed:" + err.Error())
	}
	if err := json.Unmarshal([]byte(settings2), extractor2); err != nil {
		return errors.New("extractor init failed:" + err.Error())
	}
	if err := extractor2.Compile(); err != nil {
		return errors.New("old extractor init failed:" + err.Error())
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
	p.remoteCache, err = redisx.GetByEnv(ctx, p.RedisEnv)
	if err != nil {
		return err
	}
	p.localCache = NewFixedSizeRandomDropCache(p.LocalCacheSize)

	p.Queue = make(chan *types.LogEvent, 10) // queue init, 设置一个比较小的缓存区
	if err := p.Output.Start(ctx, ctx.Logger().Logger); err != nil {
		return err
	}

	p.merge = NewPipelineMerge(p.MaxMergeSize,
		p.MergeWindowSeconds,
		time.Second*5, // 固定的5秒频率
	)
	go p.StartMerge(ctx)

	newCtx := srv.NewCtx(
		ctx.Logger().Clone().
			WithBF(context.TODO()).
			With(zap.String("worker", "hundsun_request_log_pipieline")),
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
		p.closed = true
	}
	if p.merge != nil {
		p.StopMerge(ctx)
		// 等待直到所有数据已刷新
		p.merge.WaitGroup.Wait()
	}
	if p.Output != nil {
		p.Output.Stop(ctx)
	}
}
