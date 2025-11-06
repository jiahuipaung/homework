package config

import (
	"bytes"
	"errors"
	"io/ioutil"
	"math/rand"
	"path/filepath"

	"github.com/flashcatcloud/go-pkg/srv"
	"github.com/flashcatcloud/go-pkg/utils"
	"github.com/flashcatcloud/go-pkg/x/log"
	"github.com/flashcatcloud/go-pkg/x/redisx"
	"gopkg.in/yaml.v2"
)

type Config struct {
	Srv   srv.Config               `yaml:"srv"`
	HTTP  srv.SvcConfig            `yaml:"-"`
	Log   log.Config               `yaml:"log"`
	Stash StashConfig              `yaml:"stash"`
	Redis map[string]redisx.Config `yaml:"redis"`
}

type StashConfig struct {
	HandlerConfig `yaml:",inline"`
	N9eService    struct { // 获取logevent规则
		Username string   `yaml:"username"`
		Password string   `yaml:"password"`
		Webapis  []string `yaml:"webapis"` // n9e-webapi
	} `yaml:"n9e_service"`
	InsightService struct {
		Webapis []string `yaml:"webapis"` // fc-insight
	} `yaml:"insight_service"`
	Plugins []PluginPipelineConfig `yaml:"plugins"` // 目前只有国泰的两个插件, 后续新的插件重新设计方案
}

// 暂时不支持认证
type InputKafkaConfig struct {
	DataSourceName string   `yaml:"data_source_name"`
	Brokers        []string `yaml:"brokers"`
	Topic          string   `yaml:"topic"`
	GroupID        string   `yaml:"group_id"`
}

type OutputElasticConfig struct {
	DataSourceName string   `yaml:"data_source_name"`
	Index          string   `yaml:"index"`
	KeyDate        string   `yaml:"key_date"`
	IndexFormat    string   `yaml:"-"`
	SuffixFormat   string   `yaml:"suffix_format"` // one of hourly/daily/weekly
	Retention      string   `yaml:"retention"`     // 空或0 代表永不过期, 1d、12h
	Servers        []string `yaml:"servers"`
	Username       string   `yaml:"username"`
	Password       string   `yaml:"password"`
	SkipTlsVerify  bool     `yaml:"skip_tls_verify"`
}

type PluginPipelineConfig struct {
	Type                     string              `yaml:"type"`                         //
	RedisEnv                 string              `yaml:"redis_env"`                    // 使用独立的redis节点
	NumOfWorker              int                 `yaml:"num_of_worker"`                // 日志处理的并发数, 默认8个
	WorkerID                 string              `yaml:"worker_id"`                    // 如果为空则用数组序号填充
	MaxMergeSize             int                 `yaml:"max_merge_size"`               // merger最大允许保留的记录数, 默认10W条
	MergeWindowSeconds       int                 `yaml:"merge_window_seconds"`         // 合并窗口的大小, 默认300秒
	DuplicateWindowSeconds   int                 `yaml:"duplicate_window_seconds"`     // 去重窗口的大小, 默认1小时
	AccountDictExpireSeconds int                 `yaml:"account_dict_expires_seconds"` // 词表的过期时间, 默认24小时
	LocalCacheSize           int                 `yaml:"local_cache_size"`             // 本地缓存的大小
	Input                    InputKafkaConfig    `yaml:"input"`                        // kafka数据源
	Output                   OutputElasticConfig `yaml:"output"`                       // es数据源
}

type HandlerConfig struct {
	GrokPattern             string   `yaml:"grok_pattern"`
	ReplacePattern          string   `yaml:"replace_pattern"`
	GeoIPFile               string   `yaml:"geoip_file"`
	ConfigUpdateSeconds     int      `yaml:"config_update_seconds"` // 策略同步频率, 容器化部署时可以很低
	ConfigCheckSeconds      int      `yaml:"config_check_seconds"`  // 策略同步频率, 容器化部署时可以很低
	DebugDataSourceID       int64    `yaml:"debug_data_source_id"`  // 按数据源开启DEBUG
	EnableInputFilter       bool     `yaml:"enable_input_filter"`   // 开启筛选， 包括正选和反选
	InputFilterTopics       []string `yaml:"input_filter_topics"`   // 就是include参数, 兼容旧版
	InputIncludeTopics      []string `yaml:"input_include_topics"`  // 包含
	InputExcludeTopics      []string `yaml:"input_exclude_topics"`  // 排除
	KafkaStartOffset        string   `yaml:"kafka_start_offset"`
	KafkaQueueSize          int      `yaml:"kafka_queue_size"`
	ElasticBulkActions      int      `yaml:"elastic_bulk_actions"`       // 高优先配置
	ElasticBulkSizeBytes    int      `yaml:"elastic_bulk_size_bytes"`    // 次有配置
	ElasticBulkFlushSeconds int      `yaml:"elastic_bulk_flush_seconds"` // 兜底配置
	DorisBulkActions        int      `yaml:"doris_bulk_actions"`         // 高优先配置
	DorisBulkFlushSeconds   int      `yaml:"doris_bulk_flush_seconds"`   // 兜底配置
	VMLogsBatchActions      int      `yaml:"vm_logs_batch_actions"`
	VMLogsBatchSizeBytes    int      `yaml:"vm_logs_batch_size_bytes"`
	VMLogsBatchFlushSeconds int      `yaml:"vm_logs_batch_flush_seconds"`
	VMLogsMaxRetryBackoff   int      `yaml:"vm_logs_max_retry_backoff"`
	PipelineConfig
	DisableCommonLog bool     `yaml:"disable_common_log"`
	RelatedClusters  []string `yaml:"related_clusters"`
}

type PipelineConfig struct {
	ChSize            int     `yaml:"ch_size"`
	NumOfWorker       int     `yaml:"num_of_worker"`        // 解析并发
	NumOfOutputWorker int     `yaml:"num_of_output_worker"` // 写并发
	DropRate          float64 `yaml:"drop_rate"`
	DropRandn         int     `yaml:"-"`
	Rand              *rand.Rand
}

var (
	C Config

	errEtcPathInvalid = errors.New("etc path is invalid")
	errLogDirInvalid  = errors.New("log dir is invalid")
)

func MustLoad(etcPath, logDir string) {
	if err := Load(etcPath, logDir); err != nil {
		panic(err)
	}
}

func Load(etcPath, logDir string) error {
	if len(etcPath) == 0 {
		return errEtcPathInvalid
	}

	if len(logDir) == 0 || (!utils.IsDir(logDir) && logDir != "stdout") {
		return errLogDirInvalid
	}

	ymlFiles, err := filepath.Glob(etcPath + "*.yml")
	if err != nil {
		return err
	}

	if len(ymlFiles) == 0 {
		return errEtcPathInvalid
	}

	buffer := new(bytes.Buffer)
	for _, f := range ymlFiles {
		_, _ = buffer.Write([]byte("\n"))

		data, err := ioutil.ReadFile(f)
		if err != nil {
			return err
		}

		_, err = buffer.Write(data)
		if err != nil {
			return err
		}
	}

	err = yaml.Unmarshal(buffer.Bytes(), &C)
	if err != nil {
		return err
	}

	srv.SetConfig(C.Srv)
	C.HTTP = srv.GetConfig().HttpConfig[srv.MODULE]
	log.SetConfig(C.Log, logDir)
	redisx.SetConfig(C.Redis)
	return nil
}

func (c Config) GetRelatedClusters() []string {
	return c.Stash.RelatedClusters
}
