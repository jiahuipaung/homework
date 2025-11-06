package plugin

import (
	"encoding/json"
	"errors"
	"strings"

	"github.com/flashcatcloud/fc-stash/config"
	"github.com/flashcatcloud/fc-stash/extract"
	"github.com/flashcatcloud/fc-stash/handler/logtask"
	"github.com/flashcatcloud/fc-stash/types"
	"github.com/flashcatcloud/go-pkg/srv"
	"github.com/flashcatcloud/go-pkg/utils"
	"github.com/flashcatcloud/go-pkg/x/redisx"
	"github.com/xhit/go-str2duration/v2"
)

const (
	PluginTypeHundsumRequestLog = "hundsun_request_log" // 恒生电子的客户请求日志
	PluginTypeEnrichGAccount    = "enrich_g_account"    // 期货日志enrich
)

type Plugin interface {
	Start(ctx *srv.Context, queue chan *types.LogEvent, output types.Output) error
	Stop(ct *srv.Context)
	Handle(ctx *srv.Context, event *types.LogEvent)
}

func ParseJsonExtractorByRuleStr(rule string) (*extract.JsonPrune, error) {
	// 两层嵌套结构
	var rulebytes []byte
	if strings.Contains(rule, "json_settings") {
		rulemap := make(map[string]interface{})
		if err := json.Unmarshal([]byte(rule), &rulemap); err != nil {
			return nil, err
		}
		var err error
		rulebytes, err = json.Marshal(rulemap["json_settings"])
		if err != nil {
			return nil, err
		}
	} else {
		rulebytes = []byte(rule)
	}
	ext := new(extract.JsonPrune)
	if err := json.Unmarshal(rulebytes, ext); err != nil {
		return nil, err
	}
	if err := ext.Compile(); err != nil {
		return nil, err
	}
	return ext, nil
}

func CheckAndSetDefaultValue(ctx *srv.Context, conf *config.PluginPipelineConfig) error {
	if conf == nil {
		return errors.New("nil pointer: pipeline config")
	}
	_, err := redisx.GetByEnv(ctx, conf.RedisEnv)
	if err != nil {
		return err
	}
	if conf.AccountDictExpireSeconds == 0 { // 默认24小时
		conf.AccountDictExpireSeconds = 3600 * 24
	}
	if conf.DuplicateWindowSeconds == 0 { // 默认1小时
		conf.DuplicateWindowSeconds = 3600
	}
	if conf.LocalCacheSize <= 0 {
		conf.LocalCacheSize = 10000 // 设置默认值为 10000
	}
	if conf.NumOfWorker <= 0 {
		conf.NumOfWorker = 8
	}
	if conf.MaxMergeSize <= 0 {
		conf.MaxMergeSize = 100000 // 默认10W条
	}
	if conf.MergeWindowSeconds == 0 { // 默认300秒
		conf.MergeWindowSeconds = 300
	}
	if len(conf.WorkerID) == 0 {
		if len(conf.Input.DataSourceName) == 0 {
			conf.WorkerID = conf.Input.Topic + "_" + conf.Input.GroupID + "_" + conf.Output.Index
		} else {
			conf.WorkerID = conf.Input.DataSourceName + "_" + conf.Output.DataSourceName
		}
	}
	if len(conf.Input.DataSourceName) == 0 {
		if len(conf.Input.Topic) == 0 {
			return errors.New("empty kafka topic")
		}
		if len(conf.Input.GroupID) == 0 {
			return errors.New("empty kafka consumer_group")
		}
	}
	if len(conf.Output.Index) == 0 {
		return errors.New("empty elastic index")
	}
	if len(conf.Output.SuffixFormat) > 0 {
		allows := []string{"hourly", "daily", "weekly"}
		if !utils.StringArrayContains(allows, conf.Output.SuffixFormat) {
			return errors.New("suffix_format must be one of " + strings.Join(allows, ","))
		}
		conf.Output.IndexFormat = conf.Output.Index + logtask.ElasticIndexSuffixFormat[conf.Output.SuffixFormat]
	} else {
		conf.Output.IndexFormat = conf.Output.Index
	}
	if len(conf.Output.Retention) > 0 {
		_, err := str2duration.ParseDuration(conf.Output.Retention)
		if err != nil {
			return errors.New("invalid retention:" + err.Error())
		}
	}
	return nil
}
