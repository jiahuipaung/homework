package text_parser

import (
	"encoding/json"
	"errors"
	"regexp"
	"strconv"
	"strings"

	"github.com/flashcatcloud/fc-stash/types"
)

var (
	PresetSplit = "split"
)

func init() {
	registerParser(PresetSplit, NewSplitParser)
}

type SplitParser struct {
	isRegexp    bool
	re          *regexp.Regexp
	needOrigin  bool
	remainIndex map[int]struct{} // 需要保留的下标, 如果没有依赖, 不返回对应的下标
	args        SplitArgument
}

type SplitArgument struct {
	Delimiter      string `json:"delimiter"` // delimiter
	NumOfSubstring int    `json:"num_of_substring"`
}

func NewSplitParser(format string, needOrigin bool) (types.TextParser, error) {
	if len(format) == 0 {
		return nil, errors.New("empty delimiter")
	}
	var args SplitArgument
	if len(format) > 0 && json.Valid([]byte(format)) {
		// 参数解析失败, 不影响json反序列化的执行
		if err := json.Unmarshal([]byte(format), &args); err != nil {
			return nil, errors.New("invalid args:" + err.Error())
		}
		if len(args.Delimiter) == 0 {
			return nil, errors.New("empty delimiter")
		}
	} else {
		args.Delimiter = format
	}
	parser := &SplitParser{}
	parser.args = args
	if args.Delimiter == " " {
		return parser, nil
	}
	// @2023-2-3 不确认为什么有trimspace的行为, 暂时决定去掉该逻辑, 用户输入什么就以什么为准
	// parser.args.Delimiter = strings.TrimSpace(args.Delimiter)
	parser.needOrigin = needOrigin
	// sep: regexp /..../
	// 至少有两个符号加一个分割字符串
	if len(args.Delimiter) > 2 &&
		strings.HasPrefix(args.Delimiter, "/") && strings.HasSuffix(args.Delimiter, "/") {
		parser.isRegexp = true
		re, err := regexp.Compile(strings.TrimSuffix(strings.TrimPrefix(args.Delimiter, "/"), "/"))
		if err != nil {
			return nil, err
		}
		parser.re = re
	}
	return parser, nil
}

func (p *SplitParser) Name() string {
	return PresetSplit
}

func (p *SplitParser) SetRemainIndex(index []int) {
	if len(index) > 0 {
		p.remainIndex = make(map[int]struct{})
		for _, idx := range index {
			p.remainIndex[idx] = struct{}{}
		}
	}
}

func (p *SplitParser) Parse(origin string) (map[string]interface{}, error) {
	if len(origin) == 0 {
		return map[string]interface{}{}, nil
	}
	ret := make(map[string]interface{})
	var splited []string
	if p.isRegexp {
		if p.re == nil {
			return map[string]interface{}{}, nil
		}
		if p.args.NumOfSubstring > 0 {
			splited = p.re.Split(origin, p.args.NumOfSubstring)
		} else {
			splited = p.re.Split(origin, -1)
		}

	} else {
		if p.args.NumOfSubstring > 0 {
			splited = strings.SplitN(origin, p.args.Delimiter, p.args.NumOfSubstring)
		} else {
			splited = strings.Split(origin, p.args.Delimiter)
		}
	}
	for i, substr := range splited {
		if len(p.remainIndex) > 0 {
			if _, found := p.remainIndex[i]; found {
				ret[strconv.Itoa(i)] = substr
			}
		} else {
			ret[strconv.Itoa(i)] = substr
		}
	}
	if p.needOrigin {
		ret[types.LogParserOriginText] = origin
	}

	return ret, nil
}
