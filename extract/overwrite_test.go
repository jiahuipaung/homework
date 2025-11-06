package extract

import (
	"fmt"
	"log"
	"testing"

	"github.com/flashcatcloud/fc-stash/transform/text_parser"
	"github.com/flashcatcloud/fc-stash/utils"
)

func Test_Overwrite(t *testing.T) {
	prune := &JsonPruneV2{
		PrefixMatch: []string{"message"},
		PreExtract: []*JsonPreSourceExtract{
			{
				Mode:   text_parser.PresetRegexp,
				Field:  "message",
				Format: `request_in(?P<data>.*)`,
			},
			{
				Mode:   text_parser.PresetKeyValue,
				Field:  "message.data",
				Format: `{"field_delimiter":"||", "key_value_delimiter":"="}`,
			},
		},
		OverwriteOrigin: true,
	}
	if err := prune.Compile(); err != nil {
		t.Fatal(err)
	}
	level0, err := prune.OriginJsonLogPrune(message)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(utils.ToJsonString(level0))
	fields, _, _ := JsonFlattenFromMap(level0)
	log.Println(utils.MustToJsonString(fields))
}

var message = `
{
  "message": "[INFO][2025-04-25 17:55:11.231+0800][@v0.1.15/srv/middleware.go:90] _request_in||_msg=ok||traceid=0a63016a680b5bff845041ec713f48b0||spanid=7708248711814214632||logid=19393812||agent=Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/135.0.0.0 Safari/537.36||cip=10.99.1.106||domain=fc-insight||form=null||inBody={\"kafka.brokers\":[\"10.99.1.105:9092\"],\"kafka.topic\":\"fc-insight-self\",\"kafka.group\":\"flashcat\",\"kafka.auth\":{}}||inHeader={\"Accept\":[\"application/json\"],\"Accept-Encoding\":[\"gzip, deflate\"],\"Accept-Language\":[\"zh-CN,zh;q=0.9,en;q=0.8,en-US;q=0.7\"],\"Authorization\":[\"Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJhY2Nlc3NfdXVpZCI6ImQxNzJjZTNmLTNjOWMtNGE4Yi1hZmI0LTZjMzEwMTE0ZTI3NCIsImF1dGhvcml6ZWQiOnRydWUsImV4cCI6MTc0NTYzOTEwNiwidXNlcl9pZGVudGl0eSI6IjgxLXNob3Rib3QifQ.u2rx0ygH5FOfyZjM-_sgMlyP7hZKcyGOxF4TNzUqQFI\"],\"Connection\":[\"close\"],\"Content-Length\":[\"111\"],\"Content-Type\":[\"application/json;charset=UTF-8\"],\"Fc-Workspace-Id\":[\"823147571215\"],\"Origin\":[\"http://10.99.1.106:9000\"],\"Referer\":[\"http://10.99.1.106:9000/settings/logsource/edit/kafka/10001?mode=copy\"],\"User-Agent\":[\"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/135.0.0.0 Safari/537.36\"],\"User-Name\":[\"shotbot\"],\"X-Language\":[\"zh_CN\"]}||method=POST||proto=HTTP/1.0||uri=POST:/api/v1/datasource/kafka/message/sample",
  "status": "info",
  "timestamp": 1745574911983,
  "agent_hostname": "dev-flasheye-01",
  "fcservice": "fc-insight",
  "fcsource": "9000",
  "fctags": "{\"filename\":\"fc-insight.log\"}",
  "topic": "fc-insight-self",
  "msg_key": "dev-flasheye-01/"
}
`
