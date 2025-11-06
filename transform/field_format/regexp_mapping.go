package field_format

import "regexp"

// 正则映射
const (
	PresetRegexpMapping = "regexp_mapping"
)

// 正则映射
type RegexpMappingSettings struct {
	Rules []RegexpMappingRule `json:"rules,omitempty"`
}

type RegexpMappingRule struct {
	Regexp  string `json:"regexp"`
	Mapping string `json:"mapping"`
}

type RegexpMapping struct {
	regexps  []*regexp.Regexp // 正则表达式, 二者一一对应
	mappings []string         // 映射规则
}

func NewRegexpMapping(settings RegexpMappingSettings) (*RegexpMapping, error) {
	regexps := make([]*regexp.Regexp, len(settings.Rules))
	mappings := make([]string, len(settings.Rules))
	for i, rule := range settings.Rules {
		var err error
		regexps[i], err = regexp.Compile(rule.Regexp)
		if err != nil {
			return nil, err
		}
		mappings[i] = rule.Mapping
	}
	return &RegexpMapping{regexps: regexps, mappings: mappings}, nil
}

func (r *RegexpMapping) ParseString(origin string) (string, error) {
	for i, regexp := range r.regexps {
		if regexp.MatchString(origin) {
			return r.mappings[i], nil
		}
	}
	return origin, nil
}
