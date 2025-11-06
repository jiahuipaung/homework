package text_parser

import (
	"encoding/json"
	"errors"
	"net"
	"os"
	"path/filepath"

	"github.com/flashcatcloud/fc-stash/types"
	"github.com/oschwald/geoip2-golang"
)

// 词表文件不在git仓库, 需要在打包的环境中单独处理
var (
	PresetGeoIP   = "geoip"
	geoipMMDBFile = filepath.Join(filepath.Dir(os.Args[0]), "../scripts/dict/GeoLite2-City.mmdb")
	geoipReader   *geoip2.Reader
)

func init() {
	registerParser(PresetGeoIP, NewGeoIPParser)
}

type GeoIPParser struct {
	GeoIPArgument
}

type GeoIPArgument struct {
	Lang string `json:"lang"` // 默认是en
}

func InitGeoIPParser(file string) error {
	var reader *geoip2.Reader
	var err error
	if len(file) > 0 {
		reader, err = geoip2.Open(file)
	} else {
		reader, err = geoip2.Open(geoipMMDBFile)
	}
	if err != nil {
		return err
	}
	geoipReader = reader
	return nil
}

func NewGeoIPParser(format string, needOrigin bool) (types.TextParser, error) {
	if geoipReader == nil {
		return nil, errors.New("nil geoip reader")
	}
	var args GeoIPArgument
	args.Lang = "en"
	if len(format) > 0 && json.Valid([]byte(format)) {
		// 参数解析失败, 不影响json反序列化的执行
		_ = json.Unmarshal([]byte(format), &args)
	}
	return &GeoIPParser{
		GeoIPArgument: args,
	}, nil
}

func (p *GeoIPParser) Name() string {
	return PresetGeoIP
}

// https://www.elastic.co/guide/en/logstash/current/plugins-filters-geoip.html
func (p *GeoIPParser) Parse(origin string) (map[string]interface{}, error) {
	ip := net.ParseIP(origin)
	if ip.IsPrivate() || ip.IsLoopback() {
		return nil, errors.New("must be public ip address")
	}
	record, err := geoipReader.City(ip)
	if err != nil {
		return nil, err
	}
	ret := make(map[string]interface{})
	ret["ip"] = origin
	ret["city_name"] = record.City.Names[p.Lang]
	ret["continent_code"] = record.Continent.Code
	ret["continent_name"] = record.Continent.Names[p.Lang]
	ret["country_name"] = record.Country.Names[p.Lang]
	ret["country_code2"] = record.Country.IsoCode
	ret["country_code3"] = record.RegisteredCountry.IsoCode // 不太懂
	ret["postal_code"] = record.Postal.Code
	ret["location"] = map[string]interface{}{
		"lat": record.Location.Latitude,
		"lon": record.Location.Longitude,
	}
	ret["latitude"] = record.Location.Latitude
	ret["longitude"] = record.Location.Longitude
	ret["timezone"] = record.Location.TimeZone
	if len(record.Subdivisions) > 0 {
		ret["region_code"] = record.Subdivisions[0].IsoCode
		ret["region_name"] = record.Subdivisions[0].Names[p.Lang]
	}

	return map[string]interface{}{"geoip": ret}, nil
}
