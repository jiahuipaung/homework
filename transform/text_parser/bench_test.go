package text_parser

import (
	"fmt"
	"testing"
)

var (
	text = `10.4.37.96 - - [16/Apr/2024:14:07:11 +0800] "GET /digitalstore/api/broadcast/getDataSubtitles HTTP1.1" 200 196 2.948 "http://10.99.1.106:8766/dashboards/2?datasource=95&ident=dev-backup-01" "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/123.0.0.0 Safari/537.36" "-" "10.99.1.107:17001"`
)

var (
	gonxpattern = `$remote_addr - $remote_user [$time_local] "$request" $status $body_bytes_sent $request_time "$http_referer" "$http_user_agent" "$http_x_forwarded_for" "$upstream_addr"`
	grokpattern = `%{IPORHOST:remote_addr} - %{DATA:remote-2-user} \[%{HTTPDATE:time_local}\] "%{WORD:method} %{URIPATH:request} %{DATA:protocol}" %{NUMBER:status} %{NUMBER:body_bytes_sent} %{NUMBER:request_time} "%{DATA:http_referer}" "%{DATA:http_user_agent}" "%{DATA:http_x_forwarded_for}" "%{DATA:upstream_addr}"`
)

func Test_GonxGrokParse(t *testing.T) {
	gonx, err := NewGonxParser(gonxpattern, false)
	if err != nil {
		t.Fatal(err)
	}
	grok, err := NewGrokParser(grokpattern, false)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(gonx.Parse(text))
	fmt.Println(grok.Parse(text))
}

func Benchmark_ParserGonx(b *testing.B) {
	gonx, err := NewGonxParser(gonxpattern, false)
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		gonx.Parse(text)
	}
}

func Benchmark_ParserGrok(b *testing.B) {
	grok, err := NewGrokParser(grokpattern, false)
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		grok.Parse(text)
	}
}
