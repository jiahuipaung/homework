package extract

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/flashcatcloud/fc-stash/types"
	"github.com/flashcatcloud/go-pkg/utils"
)

// JsonPruneV2的正常情况, 不覆盖原字段
func Test_JsonPruneV2_Normal(t *testing.T) {
	prune := new(JsonPruneV2)
	err := json.Unmarshal([]byte(jsonPruneV2Settings), prune)
	if err != nil {
		t.Fatal(err)
	}
	if err := prune.Compile(); err != nil {
		t.Fatal(err)
	}
	result, err := prune.Handle(context.Background(), &types.LogEvent{
		Message: jsonPruneV2Message,
	})
	if err != nil {
		t.Fatal(err)
	}
	if utils.ShouldToJsonString(result) != jsonPruneV2Output {
		t.Fatal("result not match")
	}
}

// JsonPruneV2的正常情况, 覆盖原字段
func Test_JsonPruneV2_OverwriteOrigin(t *testing.T) {
	prune := new(JsonPruneV2)
	err := json.Unmarshal([]byte(jsonPruneV2Settings), prune)
	if err != nil {
		t.Fatal(err)
	}
	prune.OverwriteOrigin = true
	if err := prune.Compile(); err != nil {
		t.Fatal(err)
	}
	result, err := prune.Handle(context.Background(), &types.LogEvent{
		Message: jsonPruneV2Message,
	})
	if err != nil {
		t.Fatal(err)
	}
	if utils.ShouldToJsonString(result) != jsonPruneV2OutputOverwrite {
		t.Fatal("result not match")
	}
}

func Test_Transformer_Normal(t *testing.T) {
	prune := new(JsonPruneV2)
	err := json.Unmarshal([]byte(jsonPruneV2Settings), prune)
	if err != nil {
		t.Fatal(err)
	}
	convert, err := ConvertJsonPruneV2ToTransformSettings(prune)
	if err != nil {
		t.Fatal(err)
	}
	transformer := new(LogTransformer)
	transformer.Settings = convert
	if err := transformer.Compile(); err != nil {
		t.Fatal(err)
	}
	result, err := transformer.Handle(context.Background(), &types.LogEvent{
		Message: jsonPruneV2Message,
	})
	if err != nil {
		t.Fatal(err)
	}
	if utils.ShouldToJsonString(result) != jsonPruneV2Output {
		t.Fatal("result not match")
	}
}

func Test_Transformer_OverwriteOrigin(t *testing.T) {
	prune := new(JsonPruneV2)
	err := json.Unmarshal([]byte(jsonPruneV2Settings), prune)
	if err != nil {
		t.Fatal(err)
	}
	prune.OverwriteOrigin = true
	convert, err := ConvertJsonPruneV2ToTransformSettings(prune)
	if err != nil {
		t.Fatal(err)
	}
	transformer := new(LogTransformer)
	transformer.Settings = convert
	if err := transformer.Compile(); err != nil {
		t.Fatal(err)
	}
	result, err := transformer.Handle(context.Background(), &types.LogEvent{
		Message: jsonPruneV2Message,
	})
	if err != nil {
		t.Fatal(err)
	}
	if utils.ShouldToJsonString(result) != jsonPruneV2OutputOverwrite {
		t.Fatal("result not match")
	}
}

func Benchmark_JsonPruneV2_Normal(b *testing.B) {
	prune := new(JsonPruneV2)
	err := json.Unmarshal([]byte(jsonPruneV2Settings), prune)
	if err != nil {
		b.Fatal(err)
	}
	if err := prune.Compile(); err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		prune.Handle(context.Background(), &types.LogEvent{
			Message: jsonPruneV2Message,
		})
	}
}

func Benchmark_JsonPruneV2_OverwriteOrigin(b *testing.B) {
	prune := new(JsonPruneV2)
	err := json.Unmarshal([]byte(jsonPruneV2Settings), prune)
	if err != nil {
		b.Fatal(err)
	}
	prune.OverwriteOrigin = true
	if err := prune.Compile(); err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		prune.Handle(context.Background(), &types.LogEvent{
			Message: jsonPruneV2Message,
		})
	}
}

func Benchmark_LogTransformer_Normal(b *testing.B) {
	prune := new(JsonPruneV2)
	err := json.Unmarshal([]byte(jsonPruneV2Settings), prune)
	if err != nil {
		b.Fatal(err)
	}
	convert, err := ConvertJsonPruneV2ToTransformSettings(prune)
	if err != nil {
		b.Fatal(err)
	}
	transformer := new(LogTransformer)
	transformer.Settings = convert
	if err := transformer.Compile(); err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		transformer.Handle(context.Background(), &types.LogEvent{
			Message: jsonPruneV2Message,
		})
	}
}

func Benchmark_LogTransformer_OverwriteOrigin(b *testing.B) {
	prune := new(JsonPruneV2)
	err := json.Unmarshal([]byte(jsonPruneV2Settings), prune)
	if err != nil {
		b.Fatal(err)
	}
	prune.OverwriteOrigin = true
	convert, err := ConvertJsonPruneV2ToTransformSettings(prune)
	if err != nil {
		b.Fatal(err)
	}
	transformer := new(LogTransformer)
	transformer.Settings = convert
	if err := transformer.Compile(); err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		transformer.Handle(context.Background(), &types.LogEvent{
			Message: jsonPruneV2Message,
		})
	}
}

var (
	jsonPruneV2Output          = `{"@timestamp":"2025-08-29T15:29:32.872+08:00","body_bytes_sent":0,"domain":"bfs.gaojihealth.com","filename":"https_bfs.gaojihealth.com_443_access.log","http_host":"bfs.gaojihealth.com","http_referrer":"https://bfs.gaojihealth.com/bfsprd/main.do?mainName=main1","http_user_agent":"Mozilla/4.0 (compatible; MSIE 7.0; Windows NT 6.1; WOW64; Trident/7.0; SLCC2; .NET CLR 2.0.50727; .NET CLR 3.5.30729; .NET CLR 3.0.30729; .NET4.0C; .NET4.0E; InfoPath.3)","method":"GET","new_field":"xyz","remote_addr":"10.4.37.96","request":"/api/v1/frontendPerfData","request_time":2.1054408197980203,"request_uri":"/bfsprd/sysIntroduction/fbs.html","scheme":"https","source":"/var/log/nginx/https_bfs.gaojihealth.com_443_access.log","status":"200","traceid":"","unix_sec_move":"1756452572","unix_sec_move_v2":"1756452572","upstream_addr":"10.8.182.41:9010","upstream_response_time":0.003,"upstream_status":"304","x_forwarded_for":""}`
	jsonPruneV2OutputOverwrite = `{"@timestamp":"2025-08-29T15:29:32.872+08:00","body_bytes_sent":0,"domain":"bfs.gaojihealth.com","http_host":"bfs.gaojihealth.com","http_referrer":"https://bfs.gaojihealth.com/bfsprd/main.do?mainName=main1","http_user_agent":"Mozilla/4.0 (compatible; MSIE 7.0; Windows NT 6.1; WOW64; Trident/7.0; SLCC2; .NET CLR 2.0.50727; .NET CLR 3.5.30729; .NET CLR 3.0.30729; .NET4.0C; .NET4.0E; InfoPath.3)","method":"GET","new_field":"xyz","remote_addr":"10.4.37.96","request":"/api/v1/frontendPerfData","request_time":2.1054408197980203,"request_uri":"/bfsprd/sysIntroduction/fbs.html","scheme":"https","source":{"filename":"https_bfs.gaojihealth.com_443_access.log"},"status":"200","traceid":"","unix_sec_move":"1756452572","unix_sec_move_v2":"1756452572","upstream_addr":"10.8.182.41:9010","upstream_response_time":0.003,"upstream_status":"304","x_forwarded_for":""}`
)

var jsonPruneV2Settings = `
{
  "pre_function": [
    
  ],
  "prefix_match": [
    "message",
    "source"
  ],
  "pre_extract": [
    {
      "mode": "json",
      "field": "message",
      "format": ""
    },
    {
      "mode": "regexp",
      "field": "source",
      "format": "/var/log/nginx/(?P\u003cfilename\u003e.*)"
    }
  ],
  "multi_pre_extract": false,
  "fields": [
    {
      "rule_type": "append",
      "origin_field": "",
      "key": "new_field",
      "value_type": "text",
      "extention": {
        "regexp": "(.*)",
        "default_value": "",
        "append_value": "xyz"
      },
      "required": false
    },
    {
      "rule_type": "graft",
      "origin_field": "message",
      "key": "__root__",
      "value_type": "text",
      "extention": {
        "preset_filter": "allmatch",
        "default_value": ""
      },
      "required": false
    },
    {
      "rule_type": "submatch",
      "origin_field": "message.@timestamp",
      "key": "message.@timestamp",
      "value_type": "date",
      "extention": {
        "preset_filter": "allmatch",
        "default_value": "",
        "date_format": "2006-01-02T15:04:05.999Z",
        "date_location": "UTC",
        "date_utc_offset": 28800
      },
      "required": false
    },
    {
      "rule_type": "submatch",
      "origin_field": "message.request",
      "key": "message.request",
      "value_type": "text",
      "extention": {
        "preset_filter": "uripath",
        "default_value": ""
      },
      "required": false
    },
    {
      "rule_type": "submatch",
      "origin_field": "message.request_time",
      "key": "message.request_time",
      "value_type": "float",
      "extention": {
        "preset_filter": "allmatch",
        "default_value": ""
      },
      "required": false
    },
    {
      "rule_type": "submatch",
      "origin_field": "message.request_uri",
      "key": "message.request_uri",
      "value_type": "text",
      "extention": {
        "preset_filter": "uripath",
        "default_value": ""
      },
      "required": false
    },
    {
      "rule_type": "remove",
      "origin_field": "message.unix_micro",
      "key": "message.unix_micro",
      "value_type": "date",
      "extention": {
        "preset_filter": "allmatch",
        "default_value": "",
        "date_format": "unix_micro",
        "date_location": "Local"
      },
      "required": false
    },
    {
      "rule_type": "remove",
      "origin_field": "message.unix_milli",
      "key": "message.unix_milli",
      "value_type": "date",
      "extention": {
        "preset_filter": "allmatch",
        "default_value": "",
        "date_format": "unix_milli",
        "date_location": "Local"
      },
      "required": false
    },
    {
      "rule_type": "submatch",
      "origin_field": "message.unix_sec",
      "key": "message.unix_sec_move",
      "value_type": "text",
      "extention": {
        "preset_filter": "allmatch",
        "default_value": ""
      },
      "required": false
    },
    {
      "rule_type": "submatch",
      "origin_field": "message.unix_sec",
      "key": "message.unix_sec_move_v2",
      "value_type": "text",
      "extention": {
        "preset_filter": "allmatch",
        "default_value": ""
      },
      "required": false
    },
    {
      "rule_type": "submatch",
      "origin_field": "message.upstream_response_time",
      "key": "message.upstream_response_time",
      "value_type": "float",
      "extention": {
        "preset_filter": "allmatch",
        "default_value": ""
      },
      "required": false
    }
  ],
  "skip_extract_error": false,
  "overwrite_origin": false
}
`

var jsonPruneV2Message string = `
{
  "@metadata": {
    "beat": "filebeat",
    "topic": "ELK_NGINX",
    "type": "doc",
    "version": "6.2.4"
  },
  "@timestamp": "2025-08-29T07:29:32.872Z",
  "beat": {
    "hostname": "EA-Money-nginx-1",
    "name": "EA-Money-nginx-1",
    "version": "6.2.4"
  },
  "message": "{\"@timestamp\":\"2025-08-29T07:29:32.872Z\",\"body_bytes_sent\":0,\"domain\":\"bfs.gaojihealth.com\",\"http_host\":\"bfs.gaojihealth.com\",\"http_referrer\":\"https://bfs.gaojihealth.com/bfsprd/main.do?mainName=main1\",\"http_user_agent\":\"Mozilla/4.0 (compatible; MSIE 7.0; Windows NT 6.1; WOW64; Trident/7.0; SLCC2; .NET CLR 2.0.50727; .NET CLR 3.5.30729; .NET CLR 3.0.30729; .NET4.0C; .NET4.0E; InfoPath.3)\",\"method\":\"GET\",\"remote_addr\":\"10.4.37.96\",\"request\":\"/api/v1/frontendPerfData\",\"request_time\":\"2.1054408197980203\",\"request_uri\":\"/bfsprd/sysIntroduction/fbs.html\",\"scheme\":\"https\",\"status\":\"200\",\"traceid\":\"\",\"unix_micro\":1756452572872723,\"unix_milli\":1756452572872,\"unix_sec\":\"1756452572\",\"upstream_addr\":\"10.8.182.41:9010\",\"upstream_response_time\":\"0.003\",\"upstream_status\":\"304\",\"x_forwarded_for\":\"\"}",
  "source": "/var/log/nginx/https_bfs.gaojihealth.com_443_access.log",
  "unix_micro": 1,
  "unix_milli": 1,
  "unix_sec": "1"
}
`
