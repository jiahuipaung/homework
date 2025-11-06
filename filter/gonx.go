package filter

import (
	"errors"
	"regexp"
	"strings"

	"github.com/flashcatcloud/fc-stash/types"
	"github.com/satyrius/gonx"
)

var (
	PresetGonx = "gonx"
)

func init() {
	registerParser(PresetGonx, NewGonxParser)
}

type GonxParser struct {
	parser     *gonx.Parser
	fields     []string
	needOrigin bool
	Format     string `json:"format"`
}

func NewGonxParser(format string, needOrigin bool) (types.TextParser, error) {
	reFields := regexp.MustCompile(`\$([A-Za-z0-9_]+)`)
	fieldMatches := reFields.FindAllStringSubmatch(format, -1)
	if len(fieldMatches) < 1 {
		return nil, errors.New("no field found in format")
	}

	fields := make([]string, len(fieldMatches))
	for i, field := range fieldMatches {
		fields[i] = strings.TrimPrefix(field[0], "$")
	}
	return &GonxParser{
		parser:     gonx.NewParser(format),
		fields:     fields,
		needOrigin: needOrigin,
		Format:     format,
	}, nil
}

func (p *GonxParser) Name() string {
	return PresetGonx
}

func (p *GonxParser) Parse(origin string) (map[string]interface{}, error) {
	if p.parser == nil || len(origin) == 0 {
		return nil, nil
	}
	entry, err := p.parser.ParseString(origin)
	if err != nil {
		return nil, err
	}

	ret := make(map[string]interface{})
	for _, field := range p.fields {
		value, _ := entry.Field(field)
		ret[field] = value
	}
	if p.needOrigin {
		ret[types.LogParserOriginText] = origin
	}
	return ret, nil
}
