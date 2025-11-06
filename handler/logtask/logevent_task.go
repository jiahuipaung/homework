package logtask

import (
	"errors"
	"regexp"

	"github.com/flashcatcloud/fc-stash/extract"
	"github.com/flashcatcloud/fc-stash/types"

	"github.com/xhit/go-str2duration/v2"
)

const (
	DefaultElasticSettings = "default"
)

const (
	LogThemeElasticIndexFormat = "_%{+@20060102}" // 多维分析入ES的默认分片格式, ${index}_20220510, 按天切分
)

const (
	LogExtractVersionPrune     = "log_prune"     // 剪枝模式, 对应新版, v2版本
	LogExtractVersionPruneV2   = "log_prune_v2"  // 剪枝模式, 对应新版, v3版本
	LogExtractVersionTransform = "log_transform" // 转换模式, 对应新版, v4版本
)

var (
	elasticIndexNamePattern = regexp.MustCompile(`^[\.\da-z][\.\d\-\_a-z]+$`)
)

var (
	ElasticIndexSuffixFormat = map[string]string{
		DefaultElasticSettings: "_%{+@20060102}",                                 // 平台托管
		"none":                 "",                                               // 不按时间切分
		"hourly":               "_%{+@2006010215}",                               // 小时
		"daily":                "_%{+@20060102}",                                 // 天
		"weekly":               "_%{+@" + types.ConstantEventTimeWeekly + "}",    // 周
		"monthly":              "_%{+@200601}",                                   // 月
		"quarterly":            "_%{+@" + types.ConstantEventTimeQuarterly + "}", // 季
	}
)

type LogeventTask struct {
	ID           int64                        `json:"id"`             // 辅助参数
	DataSourceID int64                        `json:"data_source_id"` // 数据源
	Settings     *LogeventTaskSettings        `json:"settings"`       // 提取规则, 包括日志模式、路径、规则
	Storage      *LogeventTaskStorageSettings `json:"storage"`        // 日志提取中使用
}

type LogeventTaskSettings struct {
	Mode         string                    `json:"mode"`
	Version      string                    `json:"version,omitempty"`
	Message      string                    `json:"message,omitempty"`
	Sample       string                    `json:"sample,omitempty"`
	JsonSettings extract.TransformSettings `json:"json_settings"`
}

type LogeventTaskStorageSettings struct {
	DatasourceType string                   `json:"data_source_type"`
	Elasticsearch  *LogeventElasticSettings `json:"elasticsearch,omitempty"`
	// Doris与es二选一
	Doris *LogeventDorisSettings `json:"doris,omitempty"`
	// 适配VMLogs
	VictoriaLogs *LogeventVictoriaLogSettings `json:"victoria_logs,omitempty"`
}

type LogeventVictoriaLogSettings struct {
	DataSourceID         int64 `json:"data_source_id"`
	MsgFieldSettings     `json:"msg_field_settings"`
	TimeFieldSettings    `json:"time_field_settings"`
	StreamFieldsSettings `json:"stream_fields_settings"`
}

type MsgFieldSettings struct {
	AutoMsg bool `json:"auto_msg"` // 是否自动生成_msg
	// 手动指定的字段名 （AutoMsg == false 时必填）
	MsgFieldName string `json:"msg_field_name,omitempty"`
}

type TimeFieldSettings struct {
	AutoTime      bool   `json:"auto_time"` // 是否自动生成_time
	TimeFieldName string `json:"time_field_name,omitempty"`
}

type StreamFieldsSettings struct {
	Unspecified      bool     `json:"unspecified"` //true 表示不指定流字段
	StreamFieldNames []string `json:"stream_field_names"`
}

type LogeventElasticSettings struct {
	DataSourceID      int64  `json:"data_source_id"`
	IndexName         string `json:"index_name"`          // 索引名, 用于展示
	IndexSuffixFormat string `json:"index_suffix_format"` // 后缀, 用于展示
	RetentionDuration string `json:"retention_duration"`  // 0代表永不过期, 1d、12h
	KeyTimestamp      string `json:"key_timestamp"`       // 时间字段
}

type LogeventDorisSettings struct {
	DataSourceID             int64    `json:"data_source_id"`
	Database                 string   `json:"database"`        // 数据库
	Table                    string   `json:"table"`           // 表名
	IsCreateTable            bool     `json:"is_create_table"` // 是否创建表还是选择已有表
	SortKeys                 []string `json:"sort_keys"`
	DorisPartitionSettings   `json:"partition_setting"`
	DorisColdDownSettings    `json:"cold_down_settings"`
	DorisReplicationSettings `json:"replication_settings"`
}

type DorisPartitionSettings struct {
	Enable               bool   `json:"enable"` // 页面开关是否打开
	PartitionField       string `json:"partition_field"`
	PartitionReserveDays int64  `json:"partition_reserve_days"`
}

type DorisColdDownSettings struct {
	Enable       bool   `json:"enable"` // 页面开关是否打开
	ResourceName string `json:"resource_name"`
	TTLHours     int64  `json:"ttl_hours"`
}

type DorisReplicationSettings struct {
	Enable bool  `json:"enable"` // 页面开关是否打开
	Num    int64 `json:"num"`
}

func FormatIndexFormat(index string, format string) string {
	suffix, found := ElasticIndexSuffixFormat[format]
	if !found {
		// 未设置 或 由平台托管, 则设置为默认值
		suffix = LogThemeElasticIndexFormat
	}
	return index + suffix
}

func (es *LogeventElasticSettings) CheckAndInit() error {
	// set default
	if len(es.IndexSuffixFormat) == 0 ||
		es.IndexSuffixFormat == DefaultElasticSettings {
		// 否则默认按天分片
		es.IndexSuffixFormat = "daily"
	}
	if len(es.RetentionDuration) == 0 ||
		es.RetentionDuration == DefaultElasticSettings {
		es.RetentionDuration = "2d"
	}
	// basic check
	if len(es.IndexName) == 0 {
		return errors.New("index_name missing")
	}
	if !elasticIndexNamePattern.MatchString(es.IndexName) {
		return errors.New("索引名必须以[. a-z 0-9]开头, 且只允许包含[._- a-z 0-9]")
	}
	if _, found := ElasticIndexSuffixFormat[es.IndexSuffixFormat]; !found {
		return errors.New("不支持的分割规则")
	}
	if es.RetentionDuration != DefaultElasticSettings {
		if _, err := str2duration.ParseDuration(es.RetentionDuration); err != nil {
			return errors.New("请输入有效的保存时长")
		}
	}
	return nil
}

func (vmlogs *LogeventVictoriaLogSettings) CheckAndInit() error {
	//
	return nil
}
