package filter

import (
	"bytes"
	"compress/gzip"
	"compress/zlib"
	"encoding/json"
	"math/rand"
	"testing"

	"github.com/flashcatcloud/fc-stash/utils"
)

func Test_Gzip(t *testing.T) {
	message := make(map[string]interface{})
	if err := json.Unmarshal([]byte(origin), &message); err != nil {
		t.Fatal(err)
	}
	line := utils.MustToJsonString(message)
	var lines []string
	for i := 0; i < rand.Intn(2)+1; i++ {
		lines = append(lines, line)
	}
	origin := utils.MustToJsonString(lines)
	compressed := gzipCompress(origin)

	decompressed, err := Decompress(compressed, "gzip")
	if err != nil {
		t.Fatal(err)
	}
	if origin != decompressed {
		t.Fatal("should be equal")
	}
}

func Test_Zlib(t *testing.T) {
	message := make(map[string]interface{})
	if err := json.Unmarshal([]byte(origin), &message); err != nil {
		t.Fatal(err)
	}
	line := utils.MustToJsonString(message)
	var lines []string
	for i := 0; i < rand.Intn(2)+1; i++ {
		lines = append(lines, line)
	}
	origin := utils.MustToJsonString(lines)
	compressed := zlibCompress(origin)

	decompressed, err := Decompress(compressed, "zlib")
	if err != nil {
		t.Fatal(err)
	}
	if origin != decompressed {
		t.Fatal("should be equal")
	}
}

func gzipCompress(line string) string {
	var b bytes.Buffer
	gz := gzip.NewWriter(&b)
	gz.Write([]byte(line))
	gz.Flush()
	gz.Close()
	return b.String()
}

func zlibCompress(line string) string {
	var b bytes.Buffer
	gz := zlib.NewWriter(&b)
	gz.Write([]byte(line))
	gz.Flush()
	gz.Close()
	return b.String()
}

var origin = `
{
	"@timestamp": "2022-04-25T03:48:33.724Z",
	"@metadata": {
	  "beat": "filebeat",
	  "type": "doc",
	  "version": "6.2.4",
	  "topic": "ELK_NGINX"
	},
	"message": "{\"remote_addr\":\"10.4.37.96\",\"@timestamp\":\"2022-04-25T11:48:33+08:00\",\"http_host\":\"bfs.gaojihealth.com\",\"domain\":\"bfs.gaojihealth.com\",\"scheme\":\"https\",\"method\":\"GET\",\"request\":\"GET /bfsprd/sysIntroduction/fbs.html HTTP/1.1\",\"status\":\"304\",\"body_bytes_sent\":0,\"http_user_agent\":\"Mozilla/4.0 (compatible; MSIE 7.0; Windows NT 6.1; WOW64; Trident/7.0; SLCC2; .NET CLR 2.0.50727; .NET CLR 3.5.30729; .NET CLR 3.0.30729; .NET4.0C; .NET4.0E; InfoPath.3)\",\"request_time\":\"0.003\",\"request_uri\":\"/bfsprd/sysIntroduction/fbs.html\",\"http_referrer\":\"https://bfs.gaojihealth.com/bfsprd/main.do?mainName=main1\",\"x_forwarded_for\":\"\",\"upstream_status\":\"304\",\"upstream_addr\":\"10.8.182.41:9010\",\"upstream_response_time\":\"0.003\",\"traceid\":\"\"}",
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
