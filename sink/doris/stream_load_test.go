package doris

import (
	"encoding/json"
	"testing"

	"github.com/flashcatcloud/fc-stash/types"
)

/*var testCli *Client

func init() {
	output := &DorisOutput{
		Host:     "10.99.1.214:9030",
		Database: "lfn_test",
		Table:    "demo",
		Username: "root",
		Password: "",
	}
	cli, err := output.NewClient(context.Background())
	if err != nil {
		panic(err)
	}
	testCli = cli
}*/

func TestClient_StreamLoad(t *testing.T) {
	testCli.WithLoader(&StreamLoader{client: testCli})
	raw := `{
  "agent_hostname": "dev-flasheye-01",
  "fcservice": "fc-insight",
  "fcsource": "9000",
  "fctags": "{\"filename\":\"fc-insight.log\"}",
  "info": {
    "_msg": "ok",
    "agent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/135.0.0.0 Safari/537.36",
    "cip": "10.99.1.106",
    "domain": "fc-insight",
    "form": "null",
    "inBody": "{\"kafka.brokers\":[\"10.99.1.105:9092\"],\"kafka.topic\":\"fc-insight-self\",\"kafka.group\":\"flashcat\",\"kafka.auth\":{}}",
    "inHeader": "{\"Accept\":[\"application/json\"],\"Accept-Encoding\":[\"gzip, deflate\"],\"Accept-Language\":[\"zh-CN,zh;q=0.9,en;q=0.8,en-US;q=0.7\"],\"Authorization\":[\"Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJhY2Nlc3NfdXVpZCI6ImQxNzJjZTNmLTNjOWMtNGE4Yi1hZmI0LTZjMzEwMTE0ZTI3NCIsImF1dGhvcml6ZWQiOnRydWUsImV4cCI6MTc0NTYzOTEwNiwidXNlcl9pZGVudGl0eSI6IjgxLXNob3Rib3QifQ.u2rx0ygH5FOfyZjM-_sgMlyP7hZKcyGOxF4TNzUqQFI\"],\"Connection\":[\"close\"],\"Content-Length\":[\"111\"],\"Content-Type\":[\"application/json;charset=UTF-8\"],\"Fc-Workspace-Id\":[\"823147571215\"],\"Origin\":[\"http://10.99.1.106:9000\"],\"Referer\":[\"http://10.99.1.106:9000/settings/logsource/edit/kafka/10001?mode=copy\"],\"User-Agent\":[\"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/135.0.0.0 Safari/537.36\"],\"User-Name\":[\"shotbot\"],\"X-Language\":[\"zh_CN\"]}",
    "logid": "19393812",
    "method": "POST",
    "proto": "HTTP/1.0",
    "spanid": "7708248711814214632",
    "traceid": "0a63016a680b5bff845041ec713f48b0",
    "uri": "POST:/api/v1/datasource/kafka/message/sample"
  },
  "level": "INFO",
  "location": "/srv/middleware.go:90",
  "time": "2025-04-25T17:55:11.231+08:00",
  "topic": "fc-insight-self"
}`
	raw2 := `
{
  "agent_hostname": "dev-flasheye-01",
  "fcservice": "fc-insight",
  "fcsource": "9000",
  "fctags": "{\"filename\":\"fc-insight.log\"}",
  "info": {
    "_msg": "ok",
    "agent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/135.0.0.0 Safari/537.36",
    "cip": "10.99.1.106",
    "domain": "fc-insight",
    "form": "null",
    "inBody": "{\"kafka.brokers\":[\"10.99.1.105:9092\"],\"kafka.topic\":\"fc-insight-self\",\"kafka.group\":\"flashcat\",\"kafka.auth\":{}}",
    "inHeader": "{\"Accept\":[\"application/json\"],\"Accept-Encoding\":[\"gzip, deflate\"],\"Accept-Language\":[\"zh-CN,zh;q=0.9,en;q=0.8,en-US;q=0.7\"],\"Authorization\":[\"Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJhY2Nlc3NfdXVpZCI6ImQxNzJjZTNmLTNjOWMtNGE4Yi1hZmI0LTZjMzEwMTE0ZTI3NCIsImF1dGhvcml6ZWQiOnRydWUsImV4cCI6MTc0NTYzOTEwNiwidXNlcl9pZGVudGl0eSI6IjgxLXNob3Rib3QifQ.u2rx0ygH5FOfyZjM-_sgMlyP7hZKcyGOxF4TNzUqQFI\"],\"Connection\":[\"close\"],\"Content-Length\":[\"111\"],\"Content-Type\":[\"application/json;charset=UTF-8\"],\"Fc-Workspace-Id\":[\"823147571215\"],\"Origin\":[\"http://10.99.1.106:9000\"],\"Referer\":[\"http://10.99.1.106:9000/settings/logsource/edit/kafka/10001?mode=copy\"],\"User-Agent\":[\"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/135.0.0.0 Safari/537.36\"],\"User-Name\":[\"shotbot\"],\"X-Language\":[\"zh_CN\"]}",
    "logid": "19393812",
    "method": "POST",
    "proto": "HTTP/1.0",
    "spanid": "7708248711814214632",
    "traceid": "0a63016a680b5bff845041ec713f48b0",
    "uri": "POST:/api/v1/datasource/kafka/message/sample"
  },
  "level": "INFO",
  "location": "/srv/middleware.go:90",
  "time": "2025-05-16 11:38:01.739+0800",
  "topic": "fc-insight-self"
}`
	var (
		extracted  types.ExtractedLog
		extracted2 types.ExtractedLog
	)
	if err := json.Unmarshal([]byte(raw), &extracted); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(raw2), &extracted2); err != nil {
		t.Fatal(err)
	}
	// extracted["time"] = time.Now()

	if err := testCli.loader.Write([]types.ExtractedLog{extracted2}); err != nil {
		t.Fatal(err)
	}
}
