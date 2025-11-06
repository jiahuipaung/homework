package task

import (
	"time"

	"github.com/flashcatcloud/fc-stash/config"
	"github.com/flashcatcloud/fc-stash/handler/logtask"
	"github.com/flashcatcloud/fc-stash/handler/pipeline_handler"
	"github.com/flashcatcloud/fc-stash/sink"
	"github.com/flashcatcloud/fc-stash/types"
	"github.com/flashcatcloud/go-pkg/srv"
	"github.com/flashcatcloud/go-pkg/utils"
	"github.com/flashcatcloud/go-pkg/x/locker"
	"github.com/flashcatcloud/go-pkg/x/log"
	"github.com/xhit/go-str2duration/v2"
	"go.uber.org/zap"
)

// 全局的
func StartElasticRolloverCron(confs []config.PluginPipelineConfig) {
	ctx := srv.NewCtx(log.CommonLogger)
	ticker := time.NewTicker(time.Minute * 30) // every 30 minutes
	for {
		<-ticker.C
		ElasticRollover(ctx, confs)
	}
}

// pipeline_handler的rollover
func ElasticRollover(ctx *srv.Context, confs []config.PluginPipelineConfig) {
	// 分布式抢锁
	mutexlock := "logx_plugin_pipeline_es_rollover_mutex"
	locked, _ := locker.Lock(ctx, mutexlock, time.Second*5)
	if !locked {
		return
	}
	if locked {
		defer locker.Unlock(ctx, mutexlock)
	}
	for _, conf := range confs {
		if len(conf.Output.Index) == 0 ||
			len(conf.Output.SuffixFormat) == 0 ||
			len(conf.Output.Retention) == 0 {
			continue
		}
		allows := []string{"hourly", "daily", "weekly"}
		if !utils.StringArrayContains(allows, conf.Output.SuffixFormat) {
			continue
		}
		indexFormat := conf.Output.Index + logtask.ElasticIndexSuffixFormat[conf.Output.SuffixFormat]
		expired, err := str2duration.ParseDuration(conf.Output.Retention)
		if err != nil {
			continue
		}
		if expired <= 0 {
			continue
		}
		var suffixDuration time.Duration
		var lookbackMax int
		switch conf.Output.SuffixFormat {
		case "hourly":
			suffixDuration = time.Hour
			lookbackMax = 24 * 7
		case "daily":
			suffixDuration = time.Hour * 24
			lookbackMax = 24
		case "weekly":
			suffixDuration = time.Hour * 24 * 7
			lookbackMax = 10
		case "monthly":
			suffixDuration = time.Hour * 24 * 30
			lookbackMax = 6 // 半年
		case "quarterly":
			suffixDuration = time.Hour * 24 * 90
			lookbackMax = 4 // 1年
		}
		if suffixDuration == 0 {
			continue
		}
		pluges := &sink.ElasticOutput{
			Servers:       conf.Output.Servers,
			Username:      conf.Output.Username,
			Password:      conf.Output.Password,
			SkipTlsVerify: conf.Output.SkipTlsVerify,
		}
		escli, err := pluges.NewClient(ctx)
		if err != nil {
			continue
		}
		// 查找名称前缀是 {{自定义名称}}+"_",  "_" 代表存在分片规则
		indices, err := pipeline_handler.GetIndexByPrefix(ctx, escli, conf.Output.Index)
		if err != nil {
			ctx.Logger().Debug("expire index failed", zap.String("index", conf.Output.Index),
				zap.String("worker_id", conf.WorkerID),
				zap.Error(err))
			continue
		}
		if len(indices) == 0 {
			continue
		}
		remainIndex := make(map[string]struct{})
		for i := range indices {
			remainIndex[indices[i]] = struct{}{}
		}
		todelete := make(map[string]struct{})
		// 计算N倍的时长, 避免有遗漏
		base := time.Now().Add(-1 * expired)
		for i := 1; i < lookbackMax; i++ {
			expiredTime := base.Add(suffixDuration * time.Duration(i) * -1)
			expiredIndex := types.FormatWithEventTime(indexFormat, expiredTime)
			if _, found := remainIndex[expiredIndex]; found {
				todelete[expiredIndex] = struct{}{}
			}
		}
		if len(todelete) == 0 {
			continue
		}
		for index := range todelete {
			if err := pipeline_handler.DeleteIndex(ctx, escli, index); err != nil {
				ctx.Logger().Debug("expire index failed", zap.String("index", index),
					zap.String("worker_id", conf.WorkerID),
					zap.Error(err))
			} else {
				ctx.Logger().Debug("expire index success", zap.String("index", index),
					zap.String("worker_id", conf.WorkerID),
					zap.String("retention", conf.Output.Retention))
			}
		}
	}
}
