package extract

import (
	"fmt"
	"testing"
	"time"

	"github.com/flashcatcloud/fc-stash/transform/field_format"
	"github.com/flashcatcloud/fc-stash/transform/text_parser"
	"github.com/flashcatcloud/fc-stash/utils"
)

func Test_AddByPath(t *testing.T) {
	origin := make(map[string]interface{})
	origin["http"] = map[string]interface{}{"code": 200}
	addByPath(origin, []string{"http", "host"}, "x")
	fmt.Println(origin)

	addByPath(origin, []string{"__root__"}, map[string]interface{}{"status": 200})
	fmt.Println(origin)
}

func Test_JsonPruneLog(t *testing.T) {
	le, err := newJsonPrune()
	if err != nil {
		t.Fatal(err)
	}
	vmap, err := le.OriginJsonLogPrune(origin)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(utils.MustToJsonString(vmap))
}

func Test_JsonPruneTypes(t *testing.T) {
	le, err := newJsonPrune()
	if err != nil {
		t.Fatal(err)
	}
	vmap, err := le.OriginJsonLogPrune(origin)
	if err != nil {
		t.Fatal(err)
	}
	fields, values, types := JsonFlattenFromMap(vmap)
	for i := range fields {
		if fields[i] == "@timestamp" {
			_, ok := values[i].(time.Time)
			fmt.Println(fields[i], "is date:", ok)
		}
		if fields[i] == "fields.strings" {
			_, ok := values[i].([]interface{})
			fmt.Println(fields[i], "is slice:", ok)
		}
		if fields[i] == "fields.ints" {
			_, ok := values[i].([]interface{})
			fmt.Println(fields[i], "is slice:", ok)
		}
		fmt.Println(fields[i], types[i].String())
	}
}

func newJsonPrune() (*JsonPrune, error) {
	le := new(JsonPrune)
	le.PrefixMatch = []string{"message", "@timestamp", "nginx", "fields"}
	le.PreExtract = []*JsonPreSourceExtract{
		{
			Mode:   text_parser.PresetGonx,
			Field:  "nginx",
			Format: `$date|$pid|$log_level|[$app_name,$biz_id,$user_id]|$thread_name|$file_name,$line:$message`,
		},
		{
			Mode:  text_parser.PresetJson,
			Field: "message",
		},
	}
	le.Fields = append(le.Fields, &FieldExtract{
		RuleType:    LogExtractRuleTypeSubMatch,
		OriginField: "nginx.app_name",
		Key:         "app_name",
		ValueType:   LogExtractValueTypeText,
		Extract: ExtractRule{
			Regexp: "(.*)",
		},
	})
	le.Fields = append(le.Fields, &FieldExtract{
		RuleType:    LogExtractRuleTypeSubMatch,
		OriginField: "message.remote_addr",
		Key:         "remote_addr",
		ValueType:   LogExtractValueTypeText,
		Extract: ExtractRule{
			Regexp: "(.*)",
		},
	})
	le.Fields = append(le.Fields, &FieldExtract{
		RuleType:    LogExtractRuleTypeSubMatch,
		OriginField: "@timestamp",
		Key:         "@timestamp",
		ValueType:   LogExtractValueTypeDate,
		Extract: ExtractRule{
			Regexp:           "(.*)",
			DateFormat:       "2006-01-02T15:04:05.999Z",
			DateFixUTCOffset: 8 * 3600,
		},
	})
	le.Fields = append(le.Fields, &FieldExtract{
		RuleType:    LogExtractRuleTypeSubMatch,
		OriginField: "message.@timestamp",
		Key:         "@timestamp",
		ValueType:   LogExtractValueTypeDate,
		Extract: ExtractRule{
			Regexp:     "(.*)",
			DateFormat: "2006-01-02T15:04:05-07:00",
		},
	})
	le.Fields = append(le.Fields, &FieldExtract{
		RuleType:    LogExtractRuleTypeSubMatch,
		OriginField: "message.request",
		Key:         "request",
		ValueType:   LogExtractValueTypeText,
		Extract: ExtractRule{
			PresetFilter: field_format.PresetUriPath,
		},
	})
	le.Fields = append(le.Fields, &FieldExtract{
		RuleType:    LogExtractRuleTypeSubMatch,
		OriginField: "message.request_time",
		Key:         "request_time",
		ValueType:   LogExtractValueTypeFloat,
		Extract: ExtractRule{
			Regexp: "(.*)",
		},
	})
	le.Fields = append(le.Fields, &FieldExtract{
		RuleType:    LogExtractRuleTypeSubMatch,
		OriginField: "message.upstream_status",
		Key:         "__root__.upstream_status",
		ValueType:   LogExtractValueTypeLong,
		Extract: ExtractRule{
			Regexp: "(.*)",
		},
	})
	le.Fields = append(le.Fields, &FieldExtract{
		RuleType:  LogExtractRuleTypeAppend,
		Key:       "__root__.region",
		ValueType: LogExtractValueTypeText,
		Extract: ExtractRule{
			AppendValue: "default",
		},
	})
	le.Fields = append(le.Fields, &FieldExtract{
		RuleType:    LogExtractRuleTypeRemove,
		OriginField: "message.http_referrer",
	})
	le.Fields = append(le.Fields, &FieldExtract{
		RuleType:    LogExtractRuleTypeGraft,
		OriginField: "message",
		Key:         "__root__", // message整个移动到根
	})
	return le, le.Compile()
}

var (
	origin string = `
	{
		"@timestamp": "2022-04-25T03:48:33.724Z",
		"@metadata": {
			"beat": "filebeat",
			"type": "doc",
			"version": "6.2.4",
			"topic": "ELK_NGINX"
		},
		"offset": 31077475,
		"message": "{\"remote_addr\":\"10.4.37.96\",\"@timestamp\":\"2022-04-25T11:48:33+08:00\",\"http_host\":\"bfs.gaojihealth.com\",\"domain\":\"bfs.gaojihealth.com\",\"scheme\":\"https\",\"method\":\"GET\",\"request\":\"GET /bfsprd/sysIntroduction/fbs.html HTTP/1.1\",\"status\":\"304\",\"body_bytes_sent\":0,\"http_user_agent\":\"Mozilla/4.0 (compatible; MSIE 7.0; Windows NT 6.1; WOW64; Trident/7.0; SLCC2; .NET CLR 2.0.50727; .NET CLR 3.5.30729; .NET CLR 3.0.30729; .NET4.0C; .NET4.0E; InfoPath.3)\",\"request_time\":\"0.003\",\"request_uri\":\"/bfsprd/sysIntroduction/fbs.html\",\"http_referrer\":\"https://bfs.gaojihealth.com/bfsprd/main.do?mainName=main1\",\"x_forwarded_for\":\"\",\"upstream_status\":\"304\",\"upstream_addr\":\"10.8.182.41:9010\",\"upstream_response_time\":\"0.003\",\"traceid\":\"\"}",
		"nginx": "2022-06-16 15:41:33,595||INFO |[vn,ws4zs,]|Net_Business_Thread_1-6-1|NetService.java,173:网型变更:ws4zs开始,变更时间戳:1655365290514",
		"source": "/var/log/nginx/https_bfs.gaojihealth.com_443_access.log",
		"prospector": {
			"type": "log"
		},
		"disable": true,
		"fields": {
			"strings": ["a","b"],
			"ints": [1,2],
			"log_name": "nginx_access",
			"log_ip": "10.8.8.50",
			"cls_name": [
				"zijin",
				"BFS"
			],
			"schema": {
				"name": "source",
				"meta": {
					"value": "nginx"
				}
			},
			"log_topic": "ELK_NGINX"
		},
		"beat": {
			"version": "6.2.4",
			"name": "EA-Money-nginx-1",
			"hostname": "EA-Money-nginx-1"
		}
	}
	`
)
