package pipeline

import (
	"context"
	"errors"
	"log"
	"math"

	"strings"
	"sync/atomic"
	"time"

	"github.com/flashcatcloud/fc-stash/types"
	"github.com/flashcatcloud/fc-stash/utils"
	"github.com/flashcatcloud/go-pkg/x/prom_exporter"

	"go.uber.org/zap"
	"github.com/flashcatcloud/fc-stash/config"
	"math/rand"
)

// {input1,input2,iput3} ==> buff ==> extract ==> {output1,output2}
// 任何一个环节出错, 入数据就丢掉
type Pipeline struct {
	config.PipelineConfig
	Queue   chan *types.LogEvent
	Input   []types.Input
	Extract []*ExtractPipeline // 多路处理
	debug   bool
	closed  bool
	logger  *zap.Logger
}

type ExtractPipeline struct {
	// Extract types.Extract
	Extract atomic.Value
	Output  []types.Output
}

func NewLogEventPipeline(ctx context.Context,
	inputs []types.Input,
	extracts []*ExtractPipeline,
	cfg config.PipelineConfig) (*Pipeline, error) {
	if len(inputs) == 0 {
		return nil, errors.New("input should never be empty")
	}
	if len(extracts) == 0 {
		return nil, errors.New("extract pipeline should never be empty")
	}
	if cfg.ChSize <= 0 {
		cfg.ChSize = 1000
	}
	if cfg.NumOfWorker <= 0 {
		cfg.NumOfWorker = 8
	}
	if cfg.DropRate >= 0.01 && cfg.DropRate <= 0.99 {
		cfg.DropRandn = int(math.Floor(100 * cfg.DropRate))
		if cfg.DropRandn > 0 {
			cfg.Rand = rand.New(rand.NewSource(time.Now().UnixNano()))
		}
	}
	for _, extract := range extracts {
		if len(extract.Output) == 0 {
			return nil, errors.New("output should never be empty")
		}
	}
	for i := range inputs {
		if err := inputs[i].Init(ctx); err != nil {
			return nil, err
		}
	}
	return &Pipeline{
		PipelineConfig: cfg,
		Input:          inputs,
		Extract:        extracts,
	}, nil
}

func (p *Pipeline) SetDebug(enable bool) {
	p.debug = enable
}

func (p *Pipeline) Start(ctx context.Context, logger *zap.Logger) error {
	p.Queue = make(chan *types.LogEvent, p.ChSize) // queue init
	p.logger = logger
	for i := range p.Input {
		if err := p.Input[i].Start(ctx, logger, p.Queue); err != nil {
			return err
		}
	}
	for _, extract := range p.Extract {
		for i := range extract.Output {
			if err := extract.Output[i].Start(ctx, logger); err != nil {
				return err
			}
		}
	}

	go func(ctx context.Context, pipe *Pipeline) {
		extractWG := utils.NewWG(p.NumOfWorker) // 并发数, 可配置
		for e := range pipe.Queue {
			if pipe.Rand != nil {
				if pipe.Rand.Intn(100) < pipe.DropRandn {
					prom_exporter.Inc("log_event_extract", map[string]string{
						"code":   "drop",
						"source": e.Source,
					})
					continue
				}
			}

			fn := func(ctx context.Context, event *types.LogEvent) {
				defer func() {
					if errP := recover(); errP != nil {
						log.Printf("go routine run error: %+v, stack: %s\n",
							errP, utils.Stack(3))
					}
				}()
				if event == nil || event.Message == nil {
					return
				}

				for _, extract := range pipe.Extract {
					extractor := extract.Extract.Load().(types.Extract)
					ext, err := extractor.Handle(ctx, event)
					if err != nil {
						event.Errors = append(event.Errors, err.Error())
						prom_exporter.Inc("log_event_extract", map[string]string{
							// 空格统一替换为下划线
							"code":   strings.ReplaceAll(err.Error(), " ", "_"),
							"source": event.Source,
						})
						if pipe.debug {
							p.logger.Debug("pipeline debug extract failed",
								zap.Reflect("log event", event),
								zap.Error(err))
						}
						continue
					}
					if ext == nil {
						prom_exporter.Inc("log_event_extract", map[string]string{
							"code":   "empty_result",
							"source": event.Source,
						})
						if pipe.debug {
							p.logger.Debug("pipeline debug extract warn",
								zap.Reflect("log event", event),
								zap.String("error", "empty result"))
						}
						continue
					}
					if pipe.debug {
						p.logger.Debug("pipeline debug extract success",
							zap.Reflect("log event", event),
							zap.Reflect("result", ext))
					}
					prom_exporter.Inc("log_event_extract", map[string]string{
						"code":   "ok",
						"source": event.Source,
					})
					for i := range extract.Output {
						if err := extract.Output[i].Sink(ctx, ext); err != nil {
							if pipe.debug {
								p.logger.Debug("pipeline output failed", zap.Reflect("log event", event),
									zap.String("output", extract.Output[i].Name()),
									zap.Error(err))
							}
							event.Errors = append(e.Errors, err.Error())
						}
					}
				}
				if len(event.Errors) == 0 {
					prom_exporter.Inc("log_event_pipeline", map[string]string{
						"code": "ok",
					})
				}
			}

			extractWG.WrapFunc(fn, ctx, e)
		}

		// extractWG.Wait()
	}(ctx, p)
	return nil
}

func (p *Pipeline) Stop(ctx context.Context) {
	for i := range p.Input {
		p.Input[i].Stop(ctx)
	}
	if !p.closed {
		if p.Queue != nil {
			close(p.Queue)
		}
		p.closed = true
	}
	for _, extract := range p.Extract {
		for i := range extract.Output {
			extract.Output[i].Stop(ctx)
		}
	}
}
