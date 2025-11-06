package data_source

import (
	"errors"
	"github.com/flashcatcloud/fc-stash/utils"
	"github.com/mitchellh/mapstructure"
	"reflect"
)

type PlugVMLogs struct {
	DataSourceID int64 `json:"-" mapstructure:"-"`
	// 只在GetAllElasticSources() 方法中做了初始化
	AgentRegion string `json:"-" mapstructure:"-"`

	Endpoints   []string          `json:"vmlogs.endpoints" mapstructure:"vmlogs.endpoints"`
	Timeout     int64             `json:"vmlogs.timeout" mapstructure:"vmlogs.timeout"` // millis
	Basic       *VMLogsBasicAuth  `json:"vmlogs.basic" mapstructure:"vmlogs.basic"`
	TLS         *VMLogsTLS        `json:"vmlogs.tls" mapstructure:"vmlogs.tls"`
	Header      map[string]string `mapstructure:"header"`
	ClusterName string            `json:"vmlogs.cluster_name" mapstructure:"vmlogs.cluster_name"` // 告警引擎集群名称
}

// 与ES前端页面不同，json tag不同，暂无法复用
type VMLogsTLS struct {
	SkipTlsVerify bool `json:"vmlogs.tls.skip_tls_verify" mapstructure:"vmlogs.tls.skip_tls_verify"`
}

type VMLogsBasicAuth struct {
	Enable   bool   `json:"vmlogs.auth.enable" mapstructure:"vmlogs.auth.enable"`
	Username string `json:"vmlogs.user" mapstructure:"vmlogs.user"`
	Password string `json:"vmlogs.password" mapstructure:"vmlogs.password"`
}

func NewPlugVMLogsWithSettings(settings interface{}) (*PlugVMLogs, error) {
	newest := new(PlugVMLogs)
	settingsMap := map[string]interface{}{}

	if reflect.TypeOf(settings).Kind() == reflect.String {
		if err := utils.JsonDecodeString(settings.(string), &settingsMap); err != nil {
			return nil, err
		}
	} else {
		var assert bool
		settingsMap, assert = settings.(map[string]interface{})
		if !assert {
			return nil, errors.New("invalid settings format")
		}
	}
	if err := mapstructure.Decode(settingsMap, newest); err != nil {
		return nil, err
	}
	return newest, nil
}
