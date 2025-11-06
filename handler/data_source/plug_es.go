package data_source

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"reflect"
	"time"

	"github.com/flashcatcloud/fc-stash/utils"

	"github.com/mitchellh/mapstructure"
	"github.com/parnurzeal/gorequest"
)

type PlugES struct {
	DataSourceID int64 `json:"-" mapstructure:"-"`
	// 只在GetAllElasticSources() 方法中做了初始化
	AgentRegion string `json:"-" mapstructure:"-"`

	Nodes        []string          `json:"es.nodes" mapstructure:"es.nodes"`
	Timeout      int64             `json:"es.timeout" mapstructure:"es.timeout"` // millis
	Basic        *BasicAuth        `json:"es.basic" mapstructure:"es.basic"`
	TLS          *TLS              `json:"es.tls" mapstructure:"es.tls"`
	Version      string            `json:"es.version" mapstructure:"es.version"`
	Headers      map[string]string `json:"es.headers" mapstructure:"es.headers"`
	MinInterval  int               `json:"es.min_interval" mapstructure:"es.min_interval"` // seconds
	MaxShard     int               `json:"es.max_shard" mapstructure:"es.max_shard"`
	EnableWrite  bool              `json:"es.enable_write" mapstructure:"es.enable_write"`   // 允许写操作
	ClusterName  string            `json:"es.cluster_name" mapstructure:"es.cluster_name"`   // 告警引擎集群名称
	InternalAddr string            `json:"es.internal_addr" mapstructure:"es.internal_addr"` // 读取数据的内网地址
}

type TLS struct {
	SkipTlsVerify bool `json:"es.tls.skip_tls_verify" mapstructure:"es.tls.skip_tls_verify"`
}

type BasicAuth struct {
	Enable   bool   `json:"es.auth.enable" mapstructure:"es.auth.enable"`
	Username string `json:"es.user" mapstructure:"es.user"`
	Password string `json:"es.password" mapstructure:"es.password"`
}

func NewPlugESWithSettings(settings interface{}) (*PlugES, error) {
	newest := new(PlugES)

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

type ElasticBuildInfo struct {
	Name        string `json:"name"`
	ClusterName string `json:"cluster_name"`
	Version     struct {
		Number string `json:"number"`
	} `json:"version"`
}

// 按照data_source_id缓存, 缓存10分钟, 一般不会发生变化
func (p *PlugES) GetVersion(ctx context.Context) string {
	for _, node := range p.Nodes {
		request := gorequest.New().Timeout(time.Second * 1).Get(node)
		if p.TLS != nil && p.TLS.SkipTlsVerify {
			request.TLSClientConfig(&tls.Config{InsecureSkipVerify: true})
		}
		if p.Basic != nil && p.Basic.Enable {
			request.SetBasicAuth(p.Basic.Username, p.Basic.Password)
		}
		if p.Headers != nil {
			request.Header = make(map[string]string)
			for k, v := range p.Headers {
				request.Header[k] = v
			}
		}
		_, body, err := request.EndBytes()
		if err != nil {
			continue
		}
		info := new(ElasticBuildInfo)
		if len(body) > 0 {
			if err := json.Unmarshal(body, info); err == nil {
				return info.Version.Number
			}
		}
	}
	return ""
}
