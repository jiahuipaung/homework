package data_source

import (
	"errors"
	"reflect"

	"github.com/flashcatcloud/fc-stash/utils"
	"github.com/mitchellh/mapstructure"
)

type PlugDoris struct {
	dataSourceID int64
	IsEncrypt    bool   `json:"doris.is_encrypt" mapstructure:"doris.is_encrypt"`
	ClusterName  string `json:"doris.cluster_name" mapstructure:"doris.cluster_name"` // 告警引擎集群名称

	Addr     string `json:"doris.addr" mapstructure:"doris.addr"`         // be node
	FeAddr   string `json:"doris.fe_addr" mapstructure:"doris.fe_addr"`   // fe node
	User     string `json:"doris.user" mapstructure:"doris.user"`         //
	Password string `json:"doris.password" mapstructure:"doris.password"` //
	Timeout  int    `json:"doris.timeout" mapstructure:"doris.timeout"`

	// 是否允许写入
	EnableWrite bool `json:"doris.enable_write" mapstructure:"doris.enable_write"`
}

func NewPlugDorisWithSettings(settings interface{}) (*PlugDoris, error) {
	newest := new(PlugDoris)

	settingsMap := map[string]interface{}{}
	if reflect.TypeOf(settings).Kind() == reflect.String {
		if err := utils.JsonDecodeString(settings.(string), &settingsMap); err != nil {
			return nil, err
		}
	} else {
		var assert bool
		settingsMap, assert = settings.(map[string]interface{})
		if !assert {
			return nil, errors.New("settings type invalid")
		}
	}
	if err := mapstructure.Decode(settingsMap, newest); err != nil {
		return nil, err
	}

	return newest, nil
}
