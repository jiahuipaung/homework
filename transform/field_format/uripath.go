package field_format

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"os"
	"regexp"
	"sort"
	"strings"
	"sync/atomic"
	"unicode"

	"github.com/flashcatcloud/fc-stash/types"
	"github.com/vjeantet/grok"
)

// URL归一化
const (
	PresetUriPath = "uripath"
)

var (
	uriPathPatternFile string // 默认使用grok自带的pattern
)

var (
	uuid0Regexp = regexp.MustCompile(`[A-Fa-f0-9]{8}-([A-Fa-f0-9]{4}-){3}[A-Fa-f0-9]{12}`)
)

const (
	StashUriPathReplacePatternFromFrontend = "StashUriPathReplacePattern"
)

var (
	globalReplacePatterns = atomic.Value{}
	globalReplaceValue    = ""
)

type UriPathExtract struct {
	match   string
	replace [][2]string
	grk     *grok.Grok
}

func init() {
	// 结构初始化
	globalReplacePatterns.Store([][2]string{})
	_, err := NewUriPathExtract()
	if err == nil {
		registerFieldParser(PresetUriPath, NewUriPathExtract)
	}
}

/*
  - value样例: json, "prefix_match":"pattern"
  - 样例说明: 以 /ws-oms 开头的url, 第二段如果都是数字则统一转化为":id", 第三段如果是字母和数字则统一转化为":uuid"

[

	{
	  "prefix": "/ws-oms",
	  "pattern": "/ws-oms/(?P<id>[0-9]+)/(?P<uuid>[a-z0-9]+)/.*",
	  "order": 0
	}

]
*/
func LoadUriPathReplacePatternsFromKeyValue(ctx context.Context, value string) error {
	// value为空, 不处理
	if len(value) == 0 {
		return nil
	}
	// 没有变化, 跳过
	if globalReplaceValue == value {
		return nil
	}
	var decodes []struct {
		Prefix  string `json:"prefix"`
		Pattern string `json:"pattern"`
		Order   int    `json:"order"`
	}
	if err := json.Unmarshal([]byte(value), &decodes); err != nil {
		return err
	}
	if len(decodes) > 1 {
		sort.Slice(decodes, func(i, j int) bool {
			return decodes[i].Order < decodes[j].Order
		})
	}

	patterns := make([][2]string, 0)
	for i := range decodes {
		prefix := decodes[i].Prefix
		pattern := decodes[i].Pattern
		if len(prefix) > 0 && len(pattern) > 0 {
			patterns = append(patterns, [2]string{prefix, pattern})
		}
	}

	globalReplaceValue = value
	globalReplacePatterns.Store(patterns)
	return nil
}

func ResetUriPathPatternFile(grok string, replace string) error {
	if len(grok) > 0 {
		if _, err := os.Open(grok); err != nil {
			return err
		}
		if _, err := newUriPathExtract(grok); err != nil {
			return err
		}
		uriPathPatternFile = grok
	}
	// TO DELETE
	// 目前只有高济这个环境在用, 完成替换后再去掉这段逻辑
	if len(replace) > 0 {
		file, err := os.Open(replace)
		if err != nil {
			return err
		}
		defer file.Close()
		scanner := bufio.NewScanner(bufio.NewReader(file))

		patterns := make([][2]string, 0)
		for scanner.Scan() {
			l := scanner.Text()
			// #"key":"value"
			if len(l) > 1 && l[0] == '#' && l[1] == '"' {
				slice := strings.SplitN(strings.TrimPrefix(l, "#"), ":", 2)
				if len(slice) == 2 {
					prefix := strings.TrimSuffix(strings.TrimPrefix(slice[0], "\""), "\"")
					pattern := strings.TrimSuffix(strings.TrimPrefix(slice[1], "\""), "\"")
					patterns = append(patterns, [2]string{prefix, pattern})
				}
			}
		}
		globalReplacePatterns.Store(patterns)
	}
	return nil
}

func NewUriPathExtract() (types.FieldParser, error) {
	return newUriPathExtract(uriPathPatternFile)
}

func newUriPathExtract(patternfile string) (*UriPathExtract, error) {
	// SkipDefaultPatterns, grok pkg默认支持pattern集合, 与文件中有差异
	g, err := grok.NewWithConfig(&grok.Config{
		NamedCapturesOnly:   true,
		RemoveEmptyValues:   true,
		SkipDefaultPatterns: false, // 加载默认pattern
	})
	if err != nil {
		return nil, err
	}
	// 以文件路径覆盖
	if len(patternfile) > 0 {
		if err := g.AddPatternsFromPath(patternfile); err != nil {
			return nil, err
		}
	}
	match := "%{URIPATH:uri}"
	// preload 触发g.compile()执行, 将待提取规则(match)加载到g.compiledPatterns结构
	// 否则在上层并发调用时, 会导致 g.compiledPatterns 的 concurrent map writes
	_, _ = g.Parse(match, "GET /api/v1 HTTP/2.0")

	return &UriPathExtract{
		match: match,
		grk:   g,
	}, nil
}

func (uri *UriPathExtract) UriPathFormatDetect(origin string) bool {
	match := "%{URIPATHPARAM:target}"
	if uri.grk == nil {
		return false
	}
	values, err := uri.grk.ParseTyped(match, origin)
	if err == nil && len(values) > 0 {
		return true
	}
	return false
}

// TODO: 后续按需再升级
func (uri *UriPathExtract) ParseString(origin string) (string, error) {
	// grok提取
	if uri.grk == nil || len(uri.match) == 0 {
		return "", errors.New("uripath extractor not valid")
	}
	// 这里会并发执行
	values, err := uri.grk.ParseTyped(uri.match, origin)
	var uripath string
	if err == nil && len(values) > 0 {
		for key, value := range values {
			if key == "uri" {
				switch v := value.(type) {
				case string:
					uripath = v
				}
			}
			if len(uripath) > 0 {
				break
			}
		}
	}
	// set default value
	if len(uripath) == 0 {
		return "-", nil
	}
	replaced, newest := uri.replaceByPattern(uripath)
	if replaced {
		uripath = newest
	}
	slice := strings.Split(uripath, "/")
	start := 0
	firstNonEmpty := false // 去掉空的前缀
	end := len(slice)
	// restful参数处理
	for i := 0; i < end; i++ {
		if !firstNonEmpty && len(slice[i]) == 0 {
			start = i
		} else {
			firstNonEmpty = true
		}
		// 如果执行过replace逻辑, 该段有标签, 则不处理
		if replaced && strings.HasPrefix(slice[i], ":") {
			continue
		}
		// 否则继续进行数字或UUID的检测
		slice[i] = replaceDigits(slice[i])
		slice[i] = replaceUUID(slice[i])
	}
	// 全部都是 "/"
	if start == end-1 {
		return "/", nil
	}
	// 静态文件处理
	slice[end-1] = replaceStaticFile(slice[end-1])

	return strings.Join(slice[start:end], "/"), nil
}

func (uri *UriPathExtract) replaceByPattern(uripath string) (replaced bool, newest string) {
	// 自定义的
	patterns := uri.replace
	if len(patterns) == 0 {
		// 系统默认的
		_patterns := globalReplacePatterns.Load().([][2]string)
		if len(_patterns) > 0 {
			patterns = _patterns
		}
	}
	if len(patterns) == 0 {
		return
	}
	for _, pattern := range patterns {
		if strings.HasPrefix(uripath, pattern[0]) {
			kvpairs, err := uri.grk.Parse(pattern[1], uripath)
			if err == nil && len(kvpairs) > 0 {
				replaced = true
				for key, value := range kvpairs {
					newest = strings.Replace(uripath, value, ":"+key, -1)
				}
				return
			}
		}
	}
	return
}

func replaceDigits(str string) string {
	runes := []rune(str)
	isAllDigit := len(str) > 0
	digitCount := 0
	for i := range runes {
		if !unicode.IsDigit(runes[i]) {
			isAllDigit = false
		} else {
			digitCount++
		}
	}
	if isAllDigit {
		return ":id"
	}
	// 整体长度大于8，数字占比超过一半 或大于6个
	if len(str) > 8 && (digitCount > 6 || digitCount > len(str)/2) {
		return ":uuid"
	}
	return str
}

// TODO: 优化UUID的识别算法
func replaceUUID(str string) string {
	if len(str) >= 32 {
		if uuid0Regexp.MatchString(str) {
			return ":uuid"
		}
	}
	return str
}

// 保留静态文件类型
func replaceStaticFile(str string) string {
	static := map[string]int{ // xxxx.{type}, xxxx长度小于等于n时保留完整的
		"gif":  0,
		"jpg":  0,
		"jpeg": 0,
		"png":  0,
		"svg":  0,
		"css":  0,
		"js":   0,
		"ico":  0,
		"html": 5,
	}
	index := strings.LastIndex(str, ".")
	if index > 0 {
		prefix := str[index+1:]
		if remain, found := static[prefix]; found {
			if remain <= 0 || len(str)-len(prefix)-1 > remain {
				return ":static." + prefix
			}
		}
	}
	return str
}
