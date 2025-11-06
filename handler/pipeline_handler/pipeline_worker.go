package pipeline_handler

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/flashcatcloud/fc-stash/extract"
	"github.com/flashcatcloud/fc-stash/handler/data_source"
	"github.com/flashcatcloud/fc-stash/handler/logtask"
	"github.com/flashcatcloud/fc-stash/input"
	"github.com/flashcatcloud/fc-stash/pipeline"
	"github.com/flashcatcloud/fc-stash/sink"
	output_doris "github.com/flashcatcloud/fc-stash/sink/doris"
	"github.com/flashcatcloud/fc-stash/sink/esv6"
	output_vmlogs "github.com/flashcatcloud/fc-stash/sink/vmlogs"
	"github.com/flashcatcloud/fc-stash/transform/filter"
	"github.com/flashcatcloud/fc-stash/types"
	"github.com/flashcatcloud/fc-stash/utils"
	"github.com/flashcatcloud/go-pkg/x/prom_exporter"
	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

var (
	logger *zap.Logger
)

const (
	offsetNewest = "newest"
	offsetOldest = "oldest"
)

const (
	NullVersion = "null"
)

var offsetStrToInt64 = map[string]int64{
	offsetNewest: kafka.LastOffset,
	offsetOldest: kafka.FirstOffset,
}

var (
	workers = map[string]*PipelineWorker{}
)

type PipelineWorker struct {
	UUID string `json:"uuid"` // 即data_source_id
	*pipeline.Pipeline
}

// PipelineWorkerManage
// sources代表配置在日志提取处的，状态为开启的日志提取任务
// dataSources: 既包括源，又包括目标, key为数据源的idg
func PipelineWorkerManage(ctx context.Context,
	sources []*logtask.LogeventTask, dataSources map[int64]*logtask.DatasourceSettings) {
	defer func() {
		if errP := recover(); errP != nil {
			log.Printf("go routine run error: %+v, stack: %s\n",
				errP, utils.Stack(3))
		}
	}()

	dsPlugins := logtask.DecodeDatasourceIndex(dataSources)

	list, err := GetAvailableLogeventPipelines(ctx, sources, dsPlugins)
	if err != nil {
		logger.Warn("get available log pipelines failed", zap.Error(err))
		return
	}
	var (
		stopped     int
		rebuilded   int
		updated     int
		startedSucc int
		startedFail int
	)
	// 删除无效的任务
	for _, old := range workers {
		shouldDelete := true
		for _, newest := range list {
			if newest.UUID == old.UUID {
				shouldDelete = false
				break
			}
		}
		if shouldDelete {
			stopSingleWorker(ctx, old.UUID)
			stopped++
		}
	}
	// 更新旧的任务
	// 如果旧任务需要重建, 则整个销毁
	for _, old := range workers {
		for _, newest := range list {
			if newest.UUID == old.UUID {
				if shouldRebuildPipelineByCompare(old, newest) {
					// 需要重建, 直接停掉
					stopSingleWorker(ctx, old.UUID)
					rebuilded++
				} else {
					// 新旧配置相同, 不需要重建
					old.Update(ctx, newest)
					logger.Debug("update pipeline worker success", zap.String("data_source_id", newest.UUID))
					updated++
				}
				break
			}
		}
	}
	// 新增任务
	// 其中包括需要重建的任务
	for _, newest := range list {
		_, found := workers[newest.UUID]
		// 新增: 旧的任务列表中不存在
		if !found {
			if err := newest.Start(context.TODO(), logger); err != nil {
				startedFail++
				// stop if any running
				newest.Stop(ctx)
				// log error
				logger.Warn("start new pipeline worker failed", zap.String("data_source_id", newest.UUID),
					zap.Error(err))
			} else {
				logger.Debug("start pipeline worker success", zap.String("data_source_id", newest.UUID))
				workers[newest.UUID] = newest
				startedSucc++
			}
		}
	}
	logger.Info("statistics",
		zap.Int("started_succ", startedSucc), zap.Int("started_fail", startedFail),
		zap.Int("stopped", stopped), zap.Int("updated", updated),
		zap.Int("rebuilded", rebuilded),
	)
}

func stopSingleWorker(ctx context.Context, uuid string) {
	worker, found := workers[uuid]
	if !found {
		return
	}
	worker.Stop(ctx)
	logger.Debug("stop pipeline worker", zap.String("data_source_id", uuid))
	delete(workers, uuid)
}

// 只需要更新 ExtractPipeline.Extract 和 ElasticOutput.TimestampIndexField
// todo
func (w *PipelineWorker) Update(ctx context.Context, newest *PipelineWorker) {
	for _, extp := range newest.Extract {
		for _, old := range w.Extract {
			oldext := old.Extract.Load().(types.Extract)
			newext := extp.Extract.Load().(types.Extract)
			if oldext.GetUUID() == newext.GetUUID() {
				// 直接替换, 暂时不做并发处理
				old.Extract.Store(newext)
				if _, ok := extp.Output[0].(*output_doris.DorisOutput); ok {
					old.Output[0].(*output_doris.DorisOutput).Database = extp.Output[0].(*output_doris.DorisOutput).Database
					old.Output[0].(*output_doris.DorisOutput).Table = extp.Output[0].(*output_doris.DorisOutput).Table
				} else if _, ok := extp.Output[0].(*sink.ElasticOutput); ok {
					old.Output[0].(*sink.ElasticOutput).TimestampIndexField =
						extp.Output[0].(*sink.ElasticOutput).TimestampIndexField
					old.Output[0].(*sink.ElasticOutput).Index =
						extp.Output[0].(*sink.ElasticOutput).Index
				} else if _, ok := extp.Output[0].(*output_vmlogs.VMLogsOutput); ok {
					old.Output[0].(*output_vmlogs.VMLogsOutput).MsgField =
						extp.Output[0].(*output_vmlogs.VMLogsOutput).MsgField
					old.Output[0].(*output_vmlogs.VMLogsOutput).TimeField =
						extp.Output[0].(*output_vmlogs.VMLogsOutput).TimeField
					old.Output[0].(*output_vmlogs.VMLogsOutput).StreamFields =
						extp.Output[0].(*output_vmlogs.VMLogsOutput).StreamFields
				}
				break
			}
		}
	}
}

// 1. kafka input的brokers列表、topic名称、groupID
// 2. ExtractPipeline的UUID
func shouldRebuildPipelineByCompare(
	old *PipelineWorker, newest *PipelineWorker) bool {
	oldInput := old.Input[0].(*input.InputKafka)
	newestInput := newest.Input[0].(*input.InputKafka)
	// 旧的消费端已经关闭(上游kafka导致), 整个任务需要重建
	if oldInput.Stopped {
		return true
	}

	// 数据源配置发生了变化, 需要重建
	if strings.Join(oldInput.Brokers, ",") != strings.Join(newestInput.Brokers, ",") {
		return true
	}
	if oldInput.Topic != newestInput.Topic {
		return true
	}
	if oldInput.GroupID != newestInput.GroupID {
		return true
	}
	if oldInput.AuthMethod != newestInput.AuthMethod {
		return true
	}
	if oldInput.AuthMethod == data_source.PlugLoggingKafkaAuthMethodmTLS {
		if !(oldInput.MTLS.CACert == newestInput.MTLS.CACert &&
			oldInput.MTLS.ClientCert == newestInput.MTLS.ClientCert &&
			oldInput.MTLS.ClientKey == newestInput.MTLS.ClientKey) {
			return true
		}
	}
	if oldInput.AuthMethod == data_source.PlugLoggingKafkaAuthMethodPlain {
		if !(oldInput.SASL.Username == newestInput.SASL.Username &&
			oldInput.SASL.Password == newestInput.SASL.Password) {
			return true
		}
	}

	// Extract配置发生变化
	return checkExtractChanged(old.Extract, newest.Extract)
}

func checkExtractChanged(old []*pipeline.ExtractPipeline, newest []*pipeline.ExtractPipeline) bool {
	if len(old) != len(newest) {
		return true
	}
	var (
		oldUUID    []string
		newestUUID []string
	)
	for i := range old {
		uuid := old[i].Extract.Load().(types.Extract).GetUUID()
		oldUUID = append(oldUUID, uuid)
	}
	for i := range newest {
		uuid := newest[i].Extract.Load().(types.Extract).GetUUID()
		newestUUID = append(newestUUID, uuid)
	}
	sort.Strings(oldUUID)
	sort.Strings(newestUUID)
	// 有新增的source规则, 需要重建
	// 也可能是source的version变化, 直接触发重建
	if strings.Join(oldUUID, ",") != strings.Join(newestUUID, ",") {
		return true
	}
	for i := 0; i < len(old); i++ {
		if len(old[i].Output) != len(newest[i].Output) {
			return true
		}
		for j := 0; j < len(old[i].Output); j++ {
			// elasticsearch 的server端配置发生了变化, 需要重建
			// elasticsearch 的index发生了变化, 不需要重建
			if old[i].Output[j].IsDiff(newest[i].Output[j]) {
				return true
			}
		}
	}
	return false
}

// GetAvailableLogeventPipelines
// @modify 去掉LogTheme的依赖, 直接处理LogSource{}
// 首先, log_source按照theme_uuid分组
// 然后, log_source按照data_source_id分组, 生成log_pipeline, 一个data_source_id对应一个pipeline, 简化处理流程
// 最后 是pipeline的更新, 包括停掉旧的, 启动新的, 更新已有的(更新, 还是直接停掉, 启动新的?)
// pipeline不计划支持运行时更新, 一般的做法是只需要启动/停止两个动作
// 针对日志主题的产品需求, 额外新增一个 "extract的规则更新"动作
// 输入只支持kafka.logging, 输出只支持elasticsearch和doris
// 根据input的region, 过滤掉非中心端执行的任务
func GetAvailableLogeventPipelines(ctx context.Context, sources []*logtask.LogeventTask,
	dsIndex map[int64]interface{},
) ([]*PipelineWorker, error) {
	// 任务已清空
	if len(sources) == 0 || len(dsIndex) == 0 {
		return []*PipelineWorker{}, nil
	}

	sourceByDsID := make(map[int64][]*logtask.LogeventTask)
	for _, source := range sources {
		if source.DataSourceID == 0 {
			continue
		}
		// 目前只支持.Elasticsearch和doris
		if source.Storage == nil || (source.Storage.Elasticsearch == nil && source.Storage.Doris == nil && source.Storage.VictoriaLogs == nil) {
			continue
		}
		// 错误的配置, 直接跳过
		if source.Settings == nil {
			continue
		}
		if _, found := sourceByDsID[source.DataSourceID]; !found {
			sourceByDsID[source.DataSourceID] = make([]*logtask.LogeventTask, 0)
		}
		sourceByDsID[source.DataSourceID] = append(sourceByDsID[source.DataSourceID], source)
	}
	initCheckFail := make(map[int64]struct{})

	var workers []*PipelineWorker
	var versionCount = make(map[string]int)
	for dsID, sources := range sourceByDsID {
		_plugkafka, found := dsIndex[dsID]
		if !found {
			logger.Warn("parse log pipelines warning when get kafka, not found",
				zap.Int64("data_source_id", dsID))
			continue
		}
		// 只支持kafka.logging
		plugkafka, ok := _plugkafka.(*data_source.PlugLoggingKafka)
		if !ok {
			logger.Warn("parse log pipelines warning when get kafka, not match",
				zap.Int64("data_source_id", dsID))
			continue
		}

		// 字符串排序, 便于后续比较
		sort.Strings(plugkafka.Brokers)
		if conf.EnableInputFilter {
			if !inputFilterTopicMatched(plugkafka.Topic) {
				logger.Debug("parse log pipelines debug when filter input.topics: ignore",
					zap.Int64("data_source_id", dsID),
					zap.String("topic", plugkafka.Topic))
				continue
			}
		}

		var (
			extPipelines  []*pipeline.ExtractPipeline
			inputPrefuncs []*filter.StringPreFunction

			extPipeline *pipeline.ExtractPipeline
			err         error
		)

		for _, source := range sources {
			var (
				servers    []string
				username   string
				password   string
				outputDsID int64
			)

			if source.Storage.Elasticsearch != nil {
				outputDsID = source.Storage.Elasticsearch.DataSourceID
				extPipeline, err = buildPipelineEs(ctx, source, dsIndex, dsID, initCheckFail)
				if err != nil {
					logger.Warn("build es pipeline", zap.Error(err))
					continue
				}
			} else if source.Storage.Doris != nil {
				outputDsID = source.Storage.Doris.DataSourceID
				extPipeline, err = buildPipelineDoris(source, dsIndex, dsID, initCheckFail)
				if err != nil {
					logger.Warn("build doris pipeline failed:", zap.Error(err))
					continue
				}
			} else if source.Storage.VictoriaLogs != nil {
				outputDsID = source.Storage.VictoriaLogs.DataSourceID
				extPipeline, err = buildPipelineVMLogs(source, dsID, dsIndex, initCheckFail)
				if err != nil {
					logger.Warn("build victoria logs pipeline failed:", zap.Error(err))
					continue
				}
			} else {
				logger.Warn("output storage not set, skip")
				continue
			}

			var extractor types.Extract
			// 根据配置的版本决策, 是否开启剪枝模式
			// 页面上做过修改, 统一按照剪枝模式处理
			if source.Settings.Version == logtask.LogExtractVersionPrune {
				extractor = &extract.JsonPrune{
					UUID:        strconv.FormatInt(source.ID, 10) + "_v2", // ${source_id}_${version}
					IsCompiled:  false,
					PreFunction: source.Settings.JsonSettings.PreFunction,
					PrefixMatch: source.Settings.JsonSettings.PrefixMatch,
					PreExtract:  source.Settings.JsonSettings.PreExtract,
					Fields:      source.Settings.JsonSettings.Fields,
				}
				versionCount["JsonPrune"]++

			} else if source.Settings.Version == logtask.LogExtractVersionPruneV2 {
				// 最新版, 只有n9e-plus中适用
				extractor = &extract.JsonPruneV2{
					UUID:             strconv.FormatInt(source.ID, 10) + "_v3", // ${source_id}_${version}
					IsCompiled:       false,
					PreFunction:      source.Settings.JsonSettings.PreFunction,
					PrefixMatch:      source.Settings.JsonSettings.PrefixMatch,
					PreExtract:       source.Settings.JsonSettings.PreExtract,
					MultiPreExtract:  source.Settings.JsonSettings.MultiPreExtract,
					Fields:           source.Settings.JsonSettings.Fields,
					SkipExtractError: source.Settings.JsonSettings.SkipExtractError,
					OverwriteOrigin:  source.Settings.JsonSettings.OverwriteOrigin,
				}
				versionCount["JsonPruneV2"]++

			} else if source.Settings.Version == logtask.LogExtractVersionTransform {
				// 最新版, 只有n9e-plus中适用
				extractor = &extract.LogTransformer{
					UUID:       strconv.FormatInt(source.ID, 10) + "_v4", // ${source_id}_${version}
					IsCompiled: false,
					Settings:   &source.Settings.JsonSettings, // 直接使用json_settings, 不进行转换
				}
				versionCount["LogTransformer"]++

			} else {
				extractor = &extract.JsonExtract{
					UUID:        strconv.FormatInt(source.ID, 10) + "_v1", // ${source_id}_${version}
					IsCompiled:  false,
					PrefixMatch: source.Settings.JsonSettings.PrefixMatch,
					PreExtract:  source.Settings.JsonSettings.PreExtract,
					Fields:      source.Settings.JsonSettings.Fields,
				}
				versionCount["JsonExtract"]++
			}
			// 单个extract规则失败, 不应该影响同数据源下的其他规则
			if err = extractor.Init(ctx); err == nil {
				// 单个output不可达, 不应该影响同数据源下的其他规则
				if err := extPipeline.Output[0].Init(ctx, conf.NumOfOutputWorker); err == nil {
					extPipeline.Extract.Store(extractor)
					extPipelines = append(extPipelines, extPipeline)
				} else {
					initCheckFail[outputDsID] = struct{}{}
					logger.Warn("parse log pipelines warning when init output",
						zap.Int64("data_source_id", dsID),
						zap.Int64("log_source_id", source.ID),
						zap.Reflect("servers", servers),
						zap.String("username", username), zap.String("password", password),
						zap.Error(err))
				}
			} else {
				logger.Warn("parse log pipelines warning when init single extract",
					zap.Int64("data_source_id", dsID),
					zap.Int64("log_source_id", source.ID),
					zap.Error(err))
			}

			// 整理input pre function, 全量同步到input.kafka
			inputPrefuncs = append(inputPrefuncs, source.Settings.JsonSettings.PreFunction...)
		}

		kinput := &input.InputKafka{
			Brokers:     plugkafka.Brokers,
			Topic:       plugkafka.Topic,
			GroupID:     plugkafka.Group,
			StartOffset: offsetStrToInt64[conf.KafkaStartOffset],
			QueueSize:   conf.KafkaQueueSize,
			PreFunction: inputPrefuncs,
		}

		if plugkafka.Authentication.Method != "" {
			if err := setKafkaAuth(plugkafka, kinput); err != nil {
				logger.Warn("parse log pipelines warning when set kafka auth",
					zap.Int64("data_source_id", dsID),
					zap.Error(err))
				continue
			}
		}

		pipeline, err := pipeline.NewLogEventPipeline(ctx,
			[]types.Input{
				kinput,
			},
			extPipelines,
			conf.PipelineConfig,
		)
		if err != nil {
			logger.Warn("parse log pipelines warning when init log event pipeline",
				zap.Int64("data_source_id", dsID),
				zap.Error(err))
			continue
		}
		if conf.DebugDataSourceID == dsID {
			pipeline.SetDebug(true)
		}
		workers = append(workers, &PipelineWorker{
			UUID:     strconv.FormatInt(dsID, 10),
			Pipeline: pipeline,
		})
	}
	for k, v := range versionCount {
		prom_exporter.Set("log_extract_version_count", float64(v), map[string]string{
			"version": k,
		})
	}
	return workers, nil
}

func buildPipelineVMLogs(source *logtask.LogeventTask, dsID int64,
	dsIndex map[int64]interface{}, initCheckFail map[int64]struct{}) (*pipeline.ExtractPipeline, error) {
	extPipeline := &pipeline.ExtractPipeline{}
	vmlogs := source.Storage.VictoriaLogs

	var (
		endpoints        []string
		username         string
		password         string
		skiptls          bool
		msgFieldName     string
		timeFieldName    string
		streamFieldNames []string
	)

	// 校验失败的集群, 不要重复执行
	if _, found := initCheckFail[vmlogs.DataSourceID]; found {
		logger.Warn("parse log pipelines warning when init victorialogs output",
			zap.Int64("data_source_id", dsID),
			zap.Int64("log_source_id", source.ID),
			zap.Reflect("servers", endpoints),
			zap.String("username", username), zap.String("password", password))
		return nil, fmt.Errorf("datasource id %d already in fail map, skip", vmlogs.DataSourceID)
	}

	_plugVmlogs, found := dsIndex[vmlogs.DataSourceID]
	if !found {
		logger.Warn("get victorialogs data_source not found",
			zap.Int64("log_source_id", source.ID),
			zap.Int64("data_source_id", vmlogs.DataSourceID))
		return nil, fmt.Errorf("datasource [id:%d] not found", vmlogs.DataSourceID)
	}
	plugVmlogs, ok := _plugVmlogs.(*data_source.PlugVMLogs)
	if !ok {
		return nil, fmt.Errorf("plugin not VictoriaLogs type")
	}
	endpoints = plugVmlogs.Endpoints
	if plugVmlogs.Basic != nil { // 暂时忽略 basic.Enable 标志位
		username = plugVmlogs.Basic.Username
		password = plugVmlogs.Basic.Password
	}
	if plugVmlogs.TLS != nil {
		skiptls = plugVmlogs.TLS.SkipTlsVerify
	}
	if vmlogs.AutoMsg {
		// 用户选择自动配置，自动指定_msg:"default_message"
		msgFieldName = "_msg"
	} else {
		msgFieldName = vmlogs.MsgFieldName
	}
	if vmlogs.AutoTime {
		// 用户选择自动配置，自动指定_time:"",VMLogs将自动生成_time字段
		timeFieldName = ""
	} else {
		// 这里需要判断用户选中的时间字段 value格式是否符合要求
		timeFieldName = vmlogs.TimeFieldName
	}
	if vmlogs.Unspecified == true { // 用户不配置流字段
		streamFieldNames = []string{}
	} else {
		streamFieldNames = vmlogs.StreamFieldNames
	}

	extPipeline.Output = []types.Output{
		&output_vmlogs.VMLogsOutput{
			Endpoints:          endpoints,
			Username:           username,
			Password:           password,
			SkipTlsVerify:      skiptls,
			Headers:            plugVmlogs.Header,
			RequestTimeout:     time.Duration(plugVmlogs.Timeout) * time.Second,
			MaxRetryBackoff:    time.Duration(conf.VMLogsMaxRetryBackoff) * time.Second,
			BatchActions:       conf.VMLogsBatchActions,
			BatchSize:          conf.VMLogsBatchSizeBytes,
			BatchFlushInterval: time.Duration(conf.VMLogsBatchFlushSeconds) * time.Second,
			MsgField:           msgFieldName,
			TimeField:          timeFieldName,
			StreamFields:       streamFieldNames,
			// NumOfWork在Init时传递
		},
	}

	return extPipeline, nil
}

func buildPipelineEs(ctx context.Context, source *logtask.LogeventTask,
	dsIndex map[int64]interface{}, dsID int64, initCheckFail map[int64]struct{}) (*pipeline.ExtractPipeline, error) {
	extPipeline := &pipeline.ExtractPipeline{}
	es := source.Storage.Elasticsearch
	// reset output elastic config if set
	outputDsID := int64(0)

	var (
		servers  []string
		username string
		password string
		skiptls  bool
		version  string
	)

	// 校验失败的集群, 不要重复执行
	if _, found := initCheckFail[es.DataSourceID]; found {
		logger.Warn("parse log pipelines warning when init elastic output",
			zap.Int64("data_source_id", dsID),
			zap.Int64("log_source_id", source.ID),
			zap.Reflect("servers", servers),
			zap.String("username", username), zap.String("password", password))
		return nil, fmt.Errorf("datasource id %d already in fail map, skip", es.DataSourceID)
	}
	outputDsID = es.DataSourceID
	_pluges, found := dsIndex[es.DataSourceID]
	if !found {
		logger.Warn("get elasticsearch data_source not found",
			zap.Int64("log_source_id", source.ID),
			zap.Int64("data_source_id", es.DataSourceID))
		return nil, fmt.Errorf("datasource [id:%d] not found", es.DataSourceID)
	}
	pluges, ok := _pluges.(*data_source.PlugES)
	if !ok {
		return nil, fmt.Errorf("plugin not es type")
	}
	servers = pluges.Nodes
	if pluges.Basic != nil { // 暂时忽略 basic.Enable 标志位
		username = pluges.Basic.Username
		password = pluges.Basic.Password
	}
	if pluges.TLS != nil {
		skiptls = pluges.TLS.SkipTlsVerify
	}

	version = pluges.GetVersion(ctx)
	if strings.HasPrefix(version, "6.") {
		extPipeline.Output = []types.Output{
			&esv6.ElasticOutput{ // 修正的bulk_processor
				Index:               logtask.FormatIndexFormat(es.IndexName, es.IndexSuffixFormat),
				TimestampIndexField: es.KeyTimestamp,
				Servers:             servers,
				Username:            username,
				Password:            password,
				SkipTlsVerify:       skiptls,
				BulkActions:         conf.ElasticBulkActions,
				BulkSize:            conf.ElasticBulkSizeBytes,
				BulkFlushInterval:   time.Duration(conf.ElasticBulkFlushSeconds) * time.Second,
			},
		}
		return extPipeline, nil
	}

	// es version 7+
	extPipeline.Output = []types.Output{
		&sink.ElasticOutput{ // 修正的bulk_processor
			Index:               logtask.FormatIndexFormat(es.IndexName, es.IndexSuffixFormat),
			TimestampIndexField: es.KeyTimestamp,
			Servers:             servers,
			Username:            username,
			Password:            password,
			SkipTlsVerify:       skiptls,
			BulkActions:         conf.ElasticBulkActions,
			BulkSize:            conf.ElasticBulkSizeBytes,
			BulkFlushInterval:   time.Duration(conf.ElasticBulkFlushSeconds) * time.Second,
		},
	}
	// 本期只支持es v7及以上版本的过期删除
	AppendElasticRolloverTask(outputDsID, extPipeline.Output[0], source.Storage.Elasticsearch)
	return extPipeline, nil
}

func buildPipelineDoris(source *logtask.LogeventTask,
	dsIndex map[int64]interface{}, dsID int64, initCheckFail map[int64]struct{}) (*pipeline.ExtractPipeline, error) {
	extPipeline := &pipeline.ExtractPipeline{}
	doris := source.Storage.Doris
	// reset output elastic config if set

	var (
		servers  []string
		username string
		password string
	)

	// 校验失败的集群, 不要重复执行
	if _, found := initCheckFail[doris.DataSourceID]; found {
		logger.Warn("parse log pipelines warning when init doris output",
			zap.Int64("data_source_id", dsID),
			zap.Int64("log_source_id", source.ID),
			zap.Reflect("servers", servers),
			zap.String("username", username), zap.String("password", password))
		return nil, fmt.Errorf("datasource id %d already in fail map, skip", doris.DataSourceID)
	}
	_plugDoris, found := dsIndex[doris.DataSourceID]
	if !found {
		logger.Warn("get doris data_source not found",
			zap.Int64("log_source_id", source.ID),
			zap.Int64("data_source_id", doris.DataSourceID))
		return nil, fmt.Errorf("datasource [id:%d] not found", doris.DataSourceID)
	}
	plugDoris, ok := _plugDoris.(*data_source.PlugDoris)
	if !ok {
		return nil, fmt.Errorf("plugin not doris type")
	}

	if !plugDoris.EnableWrite {
		return nil, fmt.Errorf("given doris not enabled writting")
	}

	out := &output_doris.DorisOutput{
		Host:              plugDoris.Addr,
		FeHost:            plugDoris.FeAddr,
		Database:          doris.Database,
		Table:             doris.Table,
		Username:          plugDoris.User,
		Password:          plugDoris.Password,
		BulkActions:       conf.DorisBulkActions,
		BulkFlushInterval: time.Duration(conf.DorisBulkFlushSeconds) * time.Second,
	}

	extPipeline.Output = []types.Output{
		out,
	}
	return extPipeline, nil
}

func inputFilterTopicMatched(topic string) bool {
	// 没有开启筛选
	if !conf.EnableInputFilter {
		return true
	}
	var include []string = conf.InputIncludeTopics
	var exclude []string = conf.InputExcludeTopics
	// 兼容旧的配置字段
	if len(conf.InputFilterTopics) > 0 &&
		len(conf.InputIncludeTopics) == 0 {
		include = conf.InputFilterTopics
	}
	// 先判断是否include
	// 再判断是否exclude: 只有include的, 才会被exclude
	// include列表为空, 代表全匹配
	var included bool
	if len(include) == 0 {
		included = true
	}
	for i := range include {
		isreg, pattern := utils.TrimRegexpPattern(include[i])
		if isreg {
			reg, err := regexp.Compile(pattern)
			if err == nil && reg != nil {
				if reg.MatchString(topic) {
					included = true
					break
				}
			}

		} else {
			if topic == pattern {
				included = true
				break
			}
		}
	}
	if !included {
		return false
	}
	for i := range exclude {
		isreg, pattern := utils.TrimRegexpPattern(exclude[i])
		if isreg {
			reg, err := regexp.Compile(pattern)
			if err == nil && reg != nil {
				if reg.MatchString(topic) {
					return false
				}
			}
		} else {
			if topic == pattern {
				return false
			}
		}
	}
	return true
}

func setKafkaAuth(settings *data_source.PlugLoggingKafka, kinput *input.InputKafka) error {
	// if mTLS, set cert
	if settings.Authentication.Method == data_source.PlugLoggingKafkaAuthMethodmTLS {
		kinput.AuthMethod = data_source.PlugLoggingKafkaAuthMethodmTLS
		kinput.MTLS = &input.InputKafkaAuthmTLS{
			ClientCert: settings.Authentication.MTLS.ClientCert,
			ClientKey:  settings.Authentication.MTLS.ClientKey,
			CACert:     settings.Authentication.MTLS.CACert,
		}
	}
	// if SASL_PLAINTEXT, set username & password
	if settings.Authentication.Method == data_source.PlugLoggingKafkaAuthMethodPlain {
		kinput.AuthMethod = data_source.PlugLoggingKafkaAuthMethodPlain
		kinput.SASL = &input.InputKafkaAuthSASL{
			Username: settings.Authentication.SASL.Username,
		}
		kinput.SASL.Password = settings.Authentication.SASL.Password
	}

	return nil
}
