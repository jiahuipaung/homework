package filter

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/vjeantet/grok"
)

func Test_UriPath(t *testing.T) {
	_ = ResetUriPathPatternFile("", "../../../scripts/templates/patterns/replace-patterns")
	uri, err := newUriPathExtract("../../../scripts/templates/patterns/grok-patterns")
	if err != nil {
		t.Fatal(err)
	}

	samples := map[string]string{
		"POST /api/v1/frontendPerfData HTTP/2.0": "/api/v1/frontendPerfData",
		"POST /api/v1/frontend.png HTTP/2.0":     "/api/v1/frontend",
		"POST /mobile-pos/api/posorder/getShoppingCart?businessId=24147&topOrgId=198&storeId=1718411624147&extendCode=AD1M HTTP/2.0": "/mobile-pos/api/posorder/getShoppingCart",
		"POST storeId=1718411624147&extendCode=AD1M HTTP2.0":                                                                         "",
		"POST /api/v1/124/store?Id=1718411624147&extendCode=AD1M HTTP2.0":                                                            "/api/v1/:id/store",
		"POST /synchrony-proxy/v1/data/Synchrony-7c9bd9f4-a74d-31c7-94b3-3586ac756780/confluence-19918338 HTTP2.0":                   "/synchrony-proxy/v1/data/Synchrony-:uuid/confluence-19918338",
		"GET ///// HTTP/1.1":                 "/",
		"GET / / / // HTTP/1.1":              "/",
		"/ws-oms/403/dxejcod1/xhr_streaming": "/ws-oms/:id/:uuid/xhr_streaming",
	}
	for key := range samples {
		matched, err := uri.ParseString(key)
		fmt.Println(matched, err)
	}
}

func Test_UriReplace(t *testing.T) {
	_ = ResetUriPathPatternFile("", "../../../scripts/templates/patterns/replace-patterns")
	uri, err := newUriPathExtract("../../../scripts/templates/patterns/grok-patterns")
	if err != nil {
		t.Fatal(err)
	}

	samples := map[string]string{
		"/hsy/us558q9nhcc6/3gszc2cpwz/data/grid/CustVendor.list":                              "",
		"/hsy/us558q9nhcc6/3gszc2cpwz/amount/voucherPaidAmount/GoodsReceipt/2053095269073484": "",
	}
	for key := range samples {
		matched, err := uri.ParseString(key)
		fmt.Println(matched, err)
	}
}

func Test_UriDetect(t *testing.T) {
	uri, err := newUriPathExtract("../../../scripts/templates/patterns/grok-patterns")
	if err != nil {
		t.Fatal(err)
	}
	samples := map[string]bool{
		"POST /mobile-pos/api/posorder/getShoppingCart?businessId=24147&topOrgId=198&storeId=1718411624147&extendCode=AD1M HTTP/2.0": true,
		"POST /synchrony-proxy/v1/data/Synchrony-7c9bd9f4-a74d-31c7-94b3-3586ac756780/confluence-19918338":                           true,
		"/mobile-pos/api/posorder/getShoppingCart?businessId=24147&topOrgId=198&storeId=1718411624147&extendCode=AD1M":               true,
		"storeId=1718411624147&extendCode=AD1M": false,
	}
	for key := range samples {
		fmt.Println(uri.UriPathFormatDetect(key))
	}
}

func Test_UriGrok(t *testing.T) {
	g, err := grok.New()
	if err != nil {
		t.Fatal(err)
	}
	/*
		result, _ := g.ParseTyped(`(?P<hispath>(/his/api/inquiry/further))/.*`, "/his/api/inquiry/further/2559081262241301/27479839/latest")
		fmt.Println(result)
	*/
	result, _ := g.Parse(`(/ws-oms/(?P<id>[0-9]+)/(?P<uuid>[a-z0-9]+)/.*)`, "/ws-oms/403/dxejcod1/xhr_streaming")
	for k, v := range result {
		fmt.Println(k, v)
	}
	result2, _ := g.ParseTyped(`(/ws-oms/(?P<id>[0-9]+)/(?P<uuid>[a-z0-9]+)/.*)`, "/ws-oms/403/dxejcod1/xhr_streaming")
	for k, v := range result2 {
		fmt.Println(k, v, reflect.TypeOf(v).Name())
	}
}

func Test_Static(t *testing.T) {
	samples := []string{
		"/api/v1/northstar/incident/bubble/list",
		"index.0ad6980b.js",
		"logo-s.svg",
		"avatar1.png",
		"avatar1.",
	}
	for _, origin := range samples {
		fmt.Println(origin, replaceStaticFile(origin))
	}
}

func Test_UriAutoLoad(t *testing.T) {
	ctx := context.TODO()
	value := `
[
  {
    "prefix": "/ws-oms",
    "pattern": "/ws-oms/(?P<id>[0-9]+)/(?P<uuid>[a-z0-9]+)/.*",
    "order": 0
  }
]
`
	if err := LoadUriPathReplacePatternsFromKeyValue(ctx, value); err != nil {
		t.Fatal(err)
	}
	uripath, err := NewUriPathExtract()
	if err != nil {
		t.Fatal(err)
	}
	samples := map[string]string{
		"/ws-oms/12850123/3gszc2cpwz/data/grid/CustVendor.list":                            "",
		"/ws-oms/129680/3gszc2cpwz/amount/voucherPaidAmount/GoodsReceipt/2053095269073484": "",
	}
	for key := range samples {
		matched, err := uripath.ParseString(key)
		fmt.Println(matched, err)
	}
}
