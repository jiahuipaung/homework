package pipeline_handler

import (
	"context"
	"errors"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/flashcatcloud/fc-stash/handler/logtask"
	"github.com/flashcatcloud/fc-stash/sink"
	"github.com/flashcatcloud/fc-stash/types"
	"github.com/flashcatcloud/fc-stash/utils"
	"github.com/olivere/elastic/v7"
	"github.com/xhit/go-str2duration/v2"
	"go.uber.org/zap"
)

// rollover由stash实例执行, 每个stash实例只执行"自己负责的"那部分任务

type RolloverWorker struct {
	sync.RWMutex
	TaskIndex map[string]*RolloverTask
}

// 数据源ID + 索引名称
type RolloverTask struct {
	plugesID int64
	esV7     *sink.ElasticOutput // 暂时不支持esv6的rollover
	storage  *logtask.LogeventElasticSettings
}

var (
	rolloverOnce   = sync.Once{}
	rolloverWorker = NewRolloverWorker()
)

func NewRolloverWorker() *RolloverWorker {
	return &RolloverWorker{
		TaskIndex: make(map[string]*RolloverTask),
	}
}

func StartElasticRolloverWorker() {
	rolloverOnce.Do(func() {
		ticker := time.NewTicker(time.Minute * 30) // every 30 minutes, hard code
		go func() {
			for {
				<-ticker.C

				rolloverWorker.RLock()
				var tasks []*RolloverTask
				for _, task := range rolloverWorker.TaskIndex {
					tasks = append(tasks, task)
				}
				rolloverWorker.RUnlock()

				if len(tasks) == 0 {
					continue
				}
				for i := range tasks {
					tasks[i].RolloverIndexIfNecessary(context.TODO())
				}
			}
		}()
	})
}

func AppendElasticRolloverTask(plugesID int64, esoutput types.Output,
	storage *logtask.LogeventElasticSettings) {
	if storage == nil || esoutput == nil {
		return
	}
	// 没有开启分片规则, 直接退出, 不需要管理
	if len(storage.IndexName) == 0 ||
		storage.IndexSuffixFormat == "none" ||
		len(storage.IndexSuffixFormat) == 0 {
		return
	}
	esv7, ok := esoutput.(*sink.ElasticOutput)
	if !ok {
		return
	}
	// 只有真实添加了任务, 才触发后台任务的启动
	StartElasticRolloverWorker()

	taskID := strconv.FormatInt(plugesID, 10) + "_" + storage.IndexName
	rolloverWorker.Lock()
	defer rolloverWorker.Unlock()

	if _, found := rolloverWorker.TaskIndex[taskID]; !found {
		rolloverWorker.TaskIndex[taskID] = &RolloverTask{
			plugesID: plugesID,
			esV7:     esv7,
			storage:  storage,
		}
	}
}

func (t *RolloverTask) RolloverIndexIfNecessary(ctx context.Context) {
	defer func() {
		if errP := recover(); errP != nil {
			log.Printf("go routine run error: %+v, stack: %s\n",
				errP, utils.Stack(3))
		}
	}()
	if t.esV7 == nil || t.storage == nil {
		return
	}
	suffixFormat := t.storage.IndexSuffixFormat
	// 没有开启分片规则, 直接退出, 不需要管理
	if suffixFormat == "none" {
		return
	}
	if suffixFormat == logtask.DefaultElasticSettings {
		suffixFormat = "daily"
	}
	if len(suffixFormat) == 0 {
		return
	}

	retention := t.storage.RetentionDuration
	if retention == logtask.DefaultElasticSettings {
		retention = "2d"
	}
	if len(retention) == 0 {
		return
	}
	var suffixDuration time.Duration
	var lookbackMax int
	switch suffixFormat {
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
		return
	}
	// 复用output中的escli
	escli, err := t.esV7.NewClient(ctx)
	if err != nil {
		return
	}
	indexname := t.storage.IndexName
	indexformat := logtask.FormatIndexFormat(indexname, suffixFormat)

	// 配置了分割逻辑和保留时间
	expired, err := str2duration.ParseDuration(retention)
	if err != nil {
		logger.Debug("expire index failed", zap.String("index", indexname),
			zap.Int64("data_source_id", t.plugesID),
			zap.Error(err))
	}
	// 0 0s 代表永久保存, expired > 0 代表需要执行过期清理
	if expired > 0 {
		// 查找名称前缀是 {{自定义名称}}+"_",  "_" 代表存在分片规则
		indices, err := GetIndexByPrefix(ctx, escli, indexname+"_")
		if err != nil {
			logger.Debug("expire index failed", zap.String("index", indexname),
				zap.Int64("data_source_id", t.plugesID),
				zap.Error(err))
		}
		if len(indices) == 0 {
			return
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
			expiredIndex := types.FormatWithEventTime(indexformat, expiredTime)
			if _, found := remainIndex[expiredIndex]; found {
				todelete[expiredIndex] = struct{}{}
			}
		}
		if len(todelete) == 0 {
			return
		}
		logger.Debug("elastic index rollover expire", zap.String("index", indexname),
			zap.Int64("data_source_id", t.plugesID),
			zap.Reflect("indices", indices), zap.Reflect("todelete", todelete),
			zap.String("retention", retention), zap.String("index_suffix", suffixFormat),
		)
		for index := range todelete {
			if err := DeleteIndex(ctx, escli, index); err != nil {
				logger.Warn("expire index failed", zap.String("index", index),
					zap.Int64("data_source_id", t.plugesID),
					zap.Error(err))
			} else {
				logger.Info("expire index success", zap.String("index", index),
					zap.Int64("data_source_id", t.plugesID),
					zap.String("retention", retention))
			}
		}
	}

}

func GetIndexByPrefix(ctx context.Context, escli *elastic.Client, prefix string) ([]string, error) {
	resp, err := escli.IndexNames()
	if err != nil {
		return nil, err
	}
	var ret []string
	for _, k := range resp {
		if strings.HasPrefix(k, prefix) {
			ret = append(ret, k)
		}
	}
	return ret, nil
}

func DeleteIndex(ctx context.Context, escli *elastic.Client, index string) error {
	resp, err := escli.DeleteIndex(index).Do(ctx)
	if err != nil {
		return err
	}
	if !resp.Acknowledged {
		return errors.New("not acknowledged")
	}
	return nil
}
