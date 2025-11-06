package text_parser

import (
	"fmt"
	"testing"

	"github.com/flashcatcloud/fc-stash/utils"
	"github.com/oschwald/geoip2-golang"
)

func Test_GeoIP(t *testing.T) {
	fmt.Println(geoipMMDBFile)
	reader, err := geoip2.Open("/Users/hujter/Documents/go/src/github.com/flashcatcloud/dev.poc.p0/fc-insight/scripts/dict/GeoLite2-City.mmdb")
	if err != nil {
		t.Fatal(err)
	}
	geoipReader = reader
	geoipP, err := NewGeoIPParser("", false)
	if err != nil {
		t.Fatal(err)
	}
	ret, err := geoipP.Parse("183.199.242.240")
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(utils.MustToJsonString(ret))
}
