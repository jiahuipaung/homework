package text_parser

import (
	"github.com/flashcatcloud/fc-stash/types"
	"github.com/vjeantet/grok"
)

var (
	PresetGrok = "grok"
)

func init() {
	registerParser(PresetGrok, NewGrokParser)
}

type GrokParser struct {
	parser     *grok.Grok
	needOrigin bool
	Format     string `json:"format"`
}

func NewGrokParser(format string, needOrigin bool) (types.TextParser, error) {
	g, err := grok.NewWithConfig(&grok.Config{NamedCapturesOnly: true})
	if err != nil {
		return nil, err
	}

	// compile before parallel parse
	_, err = g.Parse(format, "EMPTY logs")
	if err != nil {
		return nil, err
	}

	return &GrokParser{
		parser:     g,
		needOrigin: needOrigin,
		Format:     format,
	}, nil
}

func (p *GrokParser) Name() string {
	return PresetGrok
}

func (p *GrokParser) Parse(origin string) (map[string]interface{}, error) {
	if p.parser == nil || len(origin) == 0 {
		return nil, nil
	}
	values, err := p.parser.ParseTyped(p.Format, origin)
	if err != nil {
		return nil, err
	}

	ret := make(map[string]interface{})
	for k, v := range values {
		ret[k] = v
	}
	if p.needOrigin {
		ret[types.LogParserOriginText] = origin
	}
	return ret, nil
}
