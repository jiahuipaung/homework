package filter

import (
	"errors"
	"fmt"
	"regexp"

	"github.com/flashcatcloud/fc-stash/types"
)

var (
	PresetRegexp = "regexp"
)

func init() {
	registerParser(PresetRegexp, NewRegexpParser)
}

type RegexpParser struct {
	re         *regexp.Regexp
	fields     []string
	needOrigin bool
	Format     string `json:"format"`
}

func NewRegexpParser(format string, needOrigin bool) (types.TextParser, error) {
	re, err := regexp.Compile(format)
	if err != nil {
		return nil, err
	}
	var fields []string
	for i, name := range re.SubexpNames() {
		if i == 0 {
			continue
		}
		fields = append(fields, name)
	}
	if len(fields) == 0 {
		return nil, errors.New(`key should set like (?P<key>\w+):\s+(?P<value>\w+)$`)
	}
	return &RegexpParser{
		re:         re,
		fields:     fields,
		needOrigin: needOrigin,
		Format:     format,
	}, nil
}

func (p *RegexpParser) Name() string {
	return PresetRegexp
}

func (p *RegexpParser) Parse(origin string) (map[string]interface{}, error) {
	if p.re == nil || len(origin) == 0 {
		return nil, nil
	}
	fields := p.re.FindStringSubmatch(origin)
	if len(fields) == 0 {
		return nil, fmt.Errorf("log line '%v' does not match given format '%v'", origin, p.Format)
	}
	// fields[0] 就是 origin
	ret := make(map[string]interface{})
	if len(fields) == len(p.fields)+1 {
		for i := range p.fields {
			ret[p.fields[i]] = fields[i+1]
		}
	}
	if p.needOrigin {
		ret[types.LogParserOriginText] = origin
	}

	return ret, nil
}
