package srvs

import (
	"time"

	"github.com/flashcatcloud/fc-stash/app/stash/tasks"
	"github.com/flashcatcloud/fc-stash/config"
	"github.com/flashcatcloud/fc-stash/extract"
	"github.com/flashcatcloud/fc-stash/handler/logtask"
	"github.com/flashcatcloud/fc-stash/handler/pipeline_handler"
	"github.com/flashcatcloud/go-pkg/srv"
	"github.com/flashcatcloud/go-pkg/utils"
	"github.com/flashcatcloud/go-pkg/x/log"
	"go.uber.org/zap"
)

var (
	workerStopSig = make(chan struct{}, 1)
)

func PipelineWorkerManage(ctx *srv.Context) error {
	dsmap1, tasks1, err := tasks.GetLogeventTaskListFromInsightService(ctx)
	if err != nil {
		return err
	}
	task2, err := tasks.GetLogeventTaskListFromN9eService(ctx)
	if err != nil {
		return err
	}
	dsmerge := make(map[int64]*logtask.DatasourceSettings)
	taskmerge := make([]*logtask.LogeventTask, 0)
	for k, v := range dsmap1 {
		if _, found := dsmerge[k]; !found {
			dsmerge[k] = v
		}
	}
	for k, v := range task2.DataSources {
		if _, found := dsmerge[k]; !found {
			dsmerge[k] = v
		}
	}
	taskmerge = append(taskmerge, tasks1...)
	taskmerge = append(taskmerge, task2.Tasks...)
	if len(task2.LabelMappings) > 0 {
		extract.SetLabelMappingSchemas(task2.LabelMappings)
	}

	// worker统一在内部调度
	// 包括index rollover
	pipeline_handler.PipelineWorkerManage(ctx, taskmerge, dsmerge)
	return nil
}

func PipelineWorkerManageStart() {
	ctx := srv.NewCtx(log.NewLogger(utils.GenerateId(), "stash"))
	pipeline_handler.SetConfig(config.C.Stash.HandlerConfig, log.CommonLogger.Logger)
	// 启动后先初始化
	if err := PipelineWorkerManage(ctx); err != nil {
		ctx.Logger().Warn("pipeline worker start first warning", zap.Error(err))
	}

	go func() {
		ticker := time.NewTicker(time.Second * 120) // 2分钟更新一次
		if config.C.Stash.ConfigUpdateSeconds > 0 {
			ticker = time.NewTicker(time.Duration(config.C.Stash.ConfigUpdateSeconds) * time.Second)
		}

		for {
			select {
			case <-ticker.C:
				if err := PipelineWorkerManage(ctx); err != nil {
					ctx.Logger().Warn("pipeline worker loop warning", zap.Error(err))
				}

			case <-workerStopSig:
				return
			}
		}
	}()
}

func PipelineWorkerManageStop() {
	close(workerStopSig)
}
