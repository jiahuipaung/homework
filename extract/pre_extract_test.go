package extract

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/flashcatcloud/fc-stash/types"
)

type test struct {
	log    map[string]interface{}
	result map[string]interface{}
	rule   *JsonPruneV2
	want   error
}

func Test_jsonFlatten(t *testing.T) {
	var log []map[string]interface{}
	json.Unmarshal([]byte(logs), &log)
	for i := 0; i < len(testData); i++ {
		testData[i].log = log[i]
		testData[i].rule.preParserPrefix = make(map[string]struct{})
		testData[i].rule.preParser = make(map[string][]types.TextParser)
		fieldCount := make(map[string]int)

		for _, extract := range testData[i].rule.PreExtract {
			fieldCount[extract.Field]++
		}
		for field, count := range fieldCount {
			testData[i].rule.preParser[field] = make([]types.TextParser, count)
		}
		for _, extract := range testData[i].rule.PreExtract {
			fieldCount[extract.Field] = 0
		}

		for _, extract := range testData[i].rule.PreExtract {
			err := extract.Compile(false)
			if err != nil {
				t.Fatal(err)
				continue
			}
			testData[i].rule.preParser[extract.Field][fieldCount[extract.Field]] = extract.parser
			fieldCount[extract.Field]++
			sources := strings.Split(extract.Field, ".")
			for j := 0; j < len(sources); j++ { // preParser的父节点, "" 空字符串代表的是root
				testData[i].rule.preParserPrefix[strings.Join(sources[0:j], ".")] = struct{}{}
			}
		}
		err := testData[i].rule.jsonFlatten(testData[i].log, "")
		if err != nil {
			testData[i].rule.addErrorTags(testData[i].log, err)
		}
		fmt.Println(i, "\n", testData[i].log, "\n", testData[i].want, "\n", err, "\n", "")
	}
}

var testData = []test{
	// 0
	// 双层单次json失败
	{
		rule: &JsonPruneV2{
			PreExtract: []*JsonPreSourceExtract{
				{
					Mode:   "json",
					Field:  "message.level2",
					Format: "",
				},
			},
			SkipExtractError: false,
			MultiPreExtract:  true,
		},
		want: errors.New("jsonFail"),
	},
	// 1
	// json成功，正则失败
	{
		rule: &JsonPruneV2{
			PreExtract: []*JsonPreSourceExtract{
				{
					Mode:   "json",
					Field:  "message.level2",
					Format: "",
				},
				{
					Mode:   "regexp",
					Field:  "source",
					Format: "\"source\":\\s*\"([^\"]+)\"",
				},
			},
			MultiPreExtract:  true,
			SkipExtractError: false,
		},
		want: errors.New("regexpFail"),
	},
	// 2
	// json成功，分隔符成功
	{
		rule: &JsonPruneV2{
			PreExtract: []*JsonPreSourceExtract{
				{
					Mode:   "json",
					Field:  "message",
					Format: "",
				},
				{
					Mode:   "split",
					Field:  "source",
					Format: "/",
				},
			},
			SkipExtractError: false,
			MultiPreExtract:  true,
		},
		want: nil,
	},
	// 3
	// 两层两次json解析都成功
	{
		rule: &JsonPruneV2{
			PreExtract: []*JsonPreSourceExtract{
				{
					Mode:   "json",
					Field:  "message",
					Format: "",
				},
				{
					Mode:   "json",
					Field:  "message.remote_addr",
					Format: "",
				},
			},
			SkipExtractError: false,
			MultiPreExtract:  true,
		},
		want: nil,
	},
	// 4
	// 顶层json失败，第二层json依赖于第一层json的成功
	{
		rule: &JsonPruneV2{
			PreExtract: []*JsonPreSourceExtract{
				{
					Mode:   "json",
					Field:  "message",
					Format: "",
				},
				{
					Mode:   "json",
					Field:  "message.remote_addr",
					Format: "",
				},
			},
			SkipExtractError: true,
			MultiPreExtract:  true,
		},
		want: errors.New("jsonFail"),
	},
	// 5
	// json失败，分隔符正确，不执行分隔符解析
	{
		rule: &JsonPruneV2{
			PreExtract: []*JsonPreSourceExtract{
				{
					Mode:   "json",
					Field:  "message",
					Format: "",
				},
				{
					Mode:   "split",
					Field:  "source",
					Format: "/",
				},
			},
			SkipExtractError: false,
			MultiPreExtract:  true,
		},
		want: errors.New("jsonFail"),
	},
	// 6
	// json失败，执行分隔符解析
	{
		rule: &JsonPruneV2{
			PreExtract: []*JsonPreSourceExtract{
				{
					Mode:   "json",
					Field:  "message",
					Format: "",
				},
				{
					Mode:   "split",
					Field:  "source",
					Format: "/",
				},
			},
			SkipExtractError: true,
			MultiPreExtract:  true,
		},
		want: errors.New("jsonFail"),
	},
	// 7
	// 单字段多规则,两项规则都成功,第二次解析覆盖第一次解析
	{
		rule: &JsonPruneV2{
			PreExtract: []*JsonPreSourceExtract{
				{
					Mode:   "split",
					Field:  "message",
					Format: "/",
				},
				{
					Mode:   "json",
					Field:  "message",
					Format: "",
				},
			},
			SkipExtractError: false,
			MultiPreExtract:  true,
		},
		want: nil,
	},
	// 8
	// 单字段多规则,第一项规则失败，不执行第二项规则
	{
		rule: &JsonPruneV2{
			PreExtract: []*JsonPreSourceExtract{
				{
					Mode:   "json",
					Field:  "message",
					Format: "",
				},
				{
					Mode:   "split",
					Field:  "message",
					Format: "/",
				},
			},
			SkipExtractError: false,
			MultiPreExtract:  true,
		},
		want: errors.New("jsonFail"),
	},
	// 9
	// 单字段多规则,第一项规则成功，第二项规则失败
	{
		rule: &JsonPruneV2{
			PreExtract: []*JsonPreSourceExtract{
				{
					Mode:   "json",
					Field:  "message",
					Format: "",
				},
				{
					Mode:   "regexp",
					Field:  "message",
					Format: "\"message\":\\s*\"([^\"]+)\"",
				},
			},
			SkipExtractError: false,
			MultiPreExtract:  true,
		},
		want: errors.New("regexpFail"),
	},
	// 10
	// 单字段多规则,第一项规则失败,跳过失败规则，第二项规则成功
	{
		rule: &JsonPruneV2{
			PreExtract: []*JsonPreSourceExtract{
				{
					Mode:   "json",
					Field:  "message",
					Format: "",
				},
				{
					Mode:   "split",
					Field:  "message",
					Format: "/",
				},
			},
			SkipExtractError: true,
			MultiPreExtract:  true,
		},
		want: errors.New("jsonFail"),
	},
	// 11
	// 单字段多规则,第一项规则失败,跳过失败规则,第二项规则成功，第三项规则失败，错误覆盖第一条规则
	{
		rule: &JsonPruneV2{
			PreExtract: []*JsonPreSourceExtract{
				{
					Mode:   "json",
					Field:  "message",
					Format: "",
				},
				{
					Mode:   "split",
					Field:  "message",
					Format: "/",
				},
				{
					Mode:   "regexp",
					Field:  "message",
					Format: "\"message\":\\s*\"([^\"]+)\"",
				},
			},
			SkipExtractError: true,
			MultiPreExtract:  true,
		},
		want: errors.New("regexpFail"),
	},
	// 12
	// 混合规则，message只取第一次成功的解析，默认规则都成功
	{
		rule: &JsonPruneV2{
			PreExtract: []*JsonPreSourceExtract{
				{
					Mode:   "split",
					Field:  "message",
					Format: "/",
				},
				{
					Mode:   "json",
					Field:  "message",
					Format: "",
				},
				{
					Mode:   "split",
					Field:  "source",
					Format: "/",
				},
			},
			SkipExtractError: true,
			MultiPreExtract:  false,
		},
		want: nil,
	},
	// 13
	// 混合规则，message后面的规则覆盖前面的规则，source字段的规则失败，跳过失败规则，不影响message后面的规则
	{
		rule: &JsonPruneV2{
			PreExtract: []*JsonPreSourceExtract{
				{
					Mode:   "split",
					Field:  "message",
					Format: "/",
				},
				{
					Mode:   "regexp",
					Field:  "source",
					Format: "\"source\":\\s*\"([^\"]+)\"",
				},
				{
					Mode:   "json",
					Field:  "message",
					Format: "",
				},
			},
			SkipExtractError: true,
			MultiPreExtract:  true,
		},
		want: errors.New("regexpFail"),
	},
	// 14
	// 混合规则，message只执行第一次成功的规则，source字段的规则失败，不影响后续unix_sec和message规则
	{
		rule: &JsonPruneV2{
			PreExtract: []*JsonPreSourceExtract{
				{
					Mode:   "split",
					Field:  "message",
					Format: "/",
				},
				{
					Mode:   "regexp",
					Field:  "source",
					Format: "\"source\":\\s*\"([^\"]+)\"",
				},
				{
					Mode:   "json",
					Field:  "message",
					Format: "",
				},
				{
					Mode:   "split",
					Field:  "unix_sec",
					Format: "/",
				},
			},
			SkipExtractError: true,
			MultiPreExtract:  false,
		},
		want: errors.New("regexpFail"),
	},
}

var logs = `
	[
		{
			"message": {
				"level2": "{remote_addr\":\"10.4.37.96\",\"@timestamp\":\"2022-04-25T11:48:33+08:00\"}"
			},
			"source": "/var/log/nginx/https_bfs.gaojihealth.com_443_access.log",
			"unix_sec": "1",
			"unix_milli": 1
		},
		{
			"message": {
				"level2": "{\"remote_addr\":\"10.4.37.96\",\"@timestamp\":\"2022-04-25T11:48:33+08:00\"}"
			},
			"source": "/var/log/nginx/https_bfs.gaojihealth.com_443_access.log",
			"unix_sec": "1",
			"unix_milli": 1
		},
       {
			"message": "{\"remote_addr\":\"10.4.37.96\",\"@timestamp\":\"2022-04-25T11:48:33+08:00\"}",
			"source": "/var/log/nginx/https_bfs.gaojihealth.com_443_access.log",
			"unix_sec": "1",
			"unix_milli": 1
		},
       {
            "message": "{\"remote_addr\":\"{\\\"ip\\\": \\\"10.4.37.96\\\"}\",\"@timestamp\":\"2022-04-25T11:48:33+08:00\"}",
			"source": "/var/log/nginx/https_bfs.gaojihealth.com_443_access.log",
			"unix_sec": "1",
			"unix_milli": 1
		},
     {
            "message": "{remote_addr\":\"{\\\"ip\\\": \\\"10.4.37.96\\\"}\",\"@timestamp\":\"2022-04-25T11:48:33+08:00\"}",
			"source": "/var/log/nginx/https_bfs.gaojihealth.com_443_access.log",
			"unix_sec": "1",
			"unix_milli": 1
		},
{
			"message": "{remote_addr\":\"10.4.37.96\",\"@timestamp\":\"2022-04-25T11:48:33+08:00\"}",
			"source": "/var/log/nginx/https_bfs.gaojihealth.com_443_access.log",
			"unix_sec": "1",
			"unix_milli": 1
		},
{
			"message": "{remote_addr\":\"10.4./37.96\",\"@time/stamp\":\"2022-04-25T11:48:33+08:00\"}",
			"source": "/var/log/nginx/https_bfs.gaojihealth.com_443_access.log",
			"unix_sec": "1",
			"unix_milli": 1
		},
{
			"message": "{\"remote/_addr\":\"10.4.37.96\",\"@timestamp\":\"2022-04-25T11:48:33+08:00\"}",
			"source": "/var/log/nginx/https_bfs.gaojihealth.com_443_access.log",
			"unix_sec": "1",
			"unix_milli": 1
		},
{
			"message": "{remote/_addr\":\"10.4.37.96\",\"@timestamp\":\"2022-04-25T11:48:33+08:00\"}",
			"source": "/var/log/nginx/https_bfs.gaojihealth.com_443_access.log",
			"unix_sec": "1",
			"unix_milli": 1
		},
{
			"message": "{\"remote/_addr\":\"10.4.37.96\",\"@timestamp\":\"2022-04-25T11:48:33+08:00\"}",
			"source": "/var/log/nginx/https_bfs.gaojihealth.com_443_access.log",
			"unix_sec": "1",
			"unix_milli": 1
		},
{
			"message": "{remote/_addr\":\"10.4.37.96\",\"@timestamp\":\"2022-04-25T11:48:33+08:00\"}",
			"source": "/var/log/nginx/https_bfs.gaojihealth.com_443_access.log",
			"unix_sec": "1",
			"unix_milli": 1
		},
{
			"message": "{remote/_addr\":\"10.4.37.96\",\"@timestamp\":\"2022-04-25T11:48:33+08:00\"}",
			"source": "/var/log/nginx/https_bfs.gaojihealth.com_443_access.log",
			"unix_sec": "1",
			"unix_milli": 1
		},
{
			"message": "{\"remote/_addr\":\"10.4.37.96\",\"@timestamp\":\"2022-04-25T11:48:33+08:00\"}",
			"source": "/var/log/nginx/https_bfs.gaojihealth.com_443_access.log",
			"unix_sec": "1",
			"unix_milli": 1
		},
{
			"message": "{\"remote/_addr\":\"10.4.37.96\",\"@timestamp\":\"2022-04-25T11:48:33+08:00\"}",
			"source": "/var/log/nginx/https_bfs.gaojihealth.com_443_access.log",
			"unix_sec": "1",
			"unix_milli": 1
		},
{
			"message": "{\"remote/_addr\":\"10.4.37.96\",\"@timestamp\":\"2022-04-25T11:48:33+08:00\"}",
			"source": "/var/log/nginx/https_bfs.gaojihealth.com_443_access.log",
			"unix_sec": "1/6321/4342/7423",
			"unix_milli": 1
		}
       
	]
	`
