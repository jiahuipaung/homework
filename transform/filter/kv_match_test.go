package filter

import (
	"encoding/json"
	"fmt"
	"testing"
)

type test struct {
	rule   *StringPreFunction
	log    map[string]interface{}
	want   bool
	result bool
	err    error
}

func Test_kvMatch(t *testing.T) {
	log := make(map[string]interface{})
	originLog := kvMatchOrigin
	//origin1 = strings.Replace(origin1, "METADATA", "filebeat", 1)
	json.Unmarshal([]byte(originLog), &log)
	fmt.Println(log)
	length := len(testData)
	for i := 0; i < length; i++ {
		testData[i].log = log
		testData[i].rule.KvMatchPreProcess()
		if testData[i].rule.kvMatch != nil {
			//	//fmt.Println(i, testData[i].rule.kvMatch.matchValue)
			testData[i].result = testData[i].rule.KvMatchOrNotMatch(log)
		}
	}
	for i := 0; i < len(testData); i++ {
		if testData[i].rule.kvMatch != nil {
			fmt.Println(testData[i].rule.kvMatch.index, testData[i].rule.kvMatch.subParts, testData[i].rule.kvMatch.matchValue)
		}
		fmt.Println(i, testData[i].want, testData[i].result)
	}
}

func Benchmark_kvMatch_test(b *testing.B) {
	log := make(map[string]interface{})
	originLog := kvMatchOrigin
	//origin1 = strings.Replace(origin1, "METADATA", "filebeat", 1)
	json.Unmarshal([]byte(originLog), &log)
	//fmt.Println(log)
	testData[0].rule.KvMatchPreProcess()
	for i := 0; i < b.N; i++ {
		_ = testData[0].rule.KvMatchOrNotMatch(log)
	}
}

var testData = []test{
	// 0
	{
		rule: &StringPreFunction{
			Mode:   "match_key_value",
			Format: "@metadata.beat=filebeat",
		},
		want: true,
	},
	// 1
	{
		rule: &StringPreFunction{
			Mode:   "match_key_value",
			Format: "@metadata.beat=notfilebeat",
		},
		want: false,
	},
	// 2
	{
		rule: &StringPreFunction{
			Mode:   "match_key_value",
			Format: "randomstring:xxxxjduenfsdifnisdfiwefio",
		},
		want: false,
	},
	// 3
	{
		rule: &StringPreFunction{
			Mode:   "match_key_value",
			Format: " xxxx = dddd = yyy",
		},
		want: false,
	},
	// 4
	{
		rule: &StringPreFunction{
			Mode:   "match_key_value",
			Format: `" "\"[y]r/.t[d]d.yy.g.d.r.u"= \/ ' \" "ddd.yyy"`,
		},
		want: false,
	},
	// 5
	{
		rule: &StringPreFunction{
			Mode:   "match_key_value",
			Format: "filebeat=@metadata.beat",
		},
		want: false,
	},
	// 6
	{
		rule: &StringPreFunction{
			Mode:   "match_key_value",
			Format: "unix_sec=1",
		},
		want: true,
	},
	// 7
	{
		rule: &StringPreFunction{
			Mode:   "match_key_value",
			Format: "1=unix_sec",
		},
		want: false,
	},
	// 8
	{
		rule: &StringPreFunction{
			Mode:   "not_match_key_value",
			Format: "filebeat=@metadata.beat",
		},
		want: false,
	},
	// 9
	{
		rule: &StringPreFunction{
			Mode:   "not_match_key_value",
			Format: "@metadata.beat=filebeat",
		},
		want: true,
	},
	// 10
	{
		rule: &StringPreFunction{
			Mode:   "match_key_value",
			Format: "@metadata.beat = filebeat",
		},
		want: true,
	},
	// 11
	{
		rule: &StringPreFunction{
			Mode:   "match_key_value",
			Format: " @metadata.beat = filebeat ",
		},
		want: true,
	},
	// 12
	{
		rule: &StringPreFunction{
			Mode:   "match_key_value",
			Format: " filebeat=@metadata.beat ",
		},
		want: false,
	},
	// 13
	{
		rule: &StringPreFunction{
			Mode:   "match_key_value",
			Format: "\"@metadata.beat\" = \"filebeat\"",
		},
		want: false,
	},
}

var kvMatchOrigin = `
{
	"@timestamp": "2022-04-25T03:48:33.724Z",
	"@metadata": {
	  "beat": "filebeat",
	  "type": "doc",
	  "version": "6.2.4",
	  "topic": "ELK_NGINX"
	},
	"message": "{\"remote_addr\":\"10.4.37.96\",\"@timestamp\":\"2022-04-25T11:48:33+08:00\"\"}",
	"source": "/var/log/nginx/https_bfs.gaojihealth.com_443_access.log",
	"unix_sec": "1",
	"unix_milli": 1,
	"beat": {
	  "version": "6.2.4",
	  "name": "EA-Money-nginx-1",
	  "hostname": "EA-Money-nginx-1"
	}
  }
`
