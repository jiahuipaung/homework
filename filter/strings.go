package filter

import (
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"strings"
)

var (
	RFCHeaderGzip = "1f8b"
	RFCHeaderZlib = "789c"
)

var (
	//TODO KEY VAlue配对
	GlobalFuncStringMatch       = "match"        // 匹配正则
	GlobalFuncStringNotMatch    = "not_match"    // 不匹配正则
	GlobalFuncStringContains    = "contains"     // 包含子串
	GlobalFuncStringNotContains = "not_contains" // 不包含子串
	GlobalFuncKeyValueMatch     = "match_key_value"
	GlobalFuncKeyValueNotMatch  = "not_match_key_value"
)

var (
	ErrorJsonExtractParamsMissingFn = func(param string) error {
		return fmt.Errorf("param[%v] missing for json extracting", param)
	}
)

// 日志原文的前置处理
type StringPreFunction struct {
	Mode       string         `json:"mode" yaml:"mode"`     // 与preExtract保持一致
	Format     string         `json:"format" yaml:"format"` // 函数的参数, 非必须
	regexp     *regexp.Regexp // 正则模式下必须
	kvMatch    *KvMatch
	isCompiled bool
}

type KvMatch struct {
	index      map[string]int
	subParts   []string
	matchValue string
}

func (pfunc *StringPreFunction) IsPrePipelineInputHandle() bool {
	return pfunc.Mode == GlobalFuncDecompress ||
		pfunc.Mode == GlobalFuncSplitJsonArray
}

// input环节的前置处理: 解压缩、json数组处理
func StringPrePipelineInputHandle(old string, funcs []*StringPreFunction,
	forceTostr ...bool) ([]interface{}, error) {
	if len(funcs) == 0 {
		return []interface{}{old}, nil
	}
	decompressIdx := -1
	splitJsonIdx := -1
	stripAnsiIdx := -1
	hasFilter := false
	for i, pfunc := range funcs {
		if pfunc.Mode == GlobalFuncDecompress {
			decompressIdx = i
		} else if pfunc.Mode == GlobalFuncSplitJsonArray {
			splitJsonIdx = i
		} else if pfunc.Mode == GlobalFuncStripAnsi {
			stripAnsiIdx = i
		} else {
			hasFilter = true
		}
	}
	// 不需要处理, 上层完全可以处理
	if decompressIdx == -1 && splitJsonIdx == -1 {
		return []interface{}{old}, nil
	}
	var newest string
	if decompressIdx != -1 {
		if format, ok := checkLogCompress([]byte(old)); ok {
			decompress, err := Decompress(old, format)
			if err != nil {
				return []interface{}{}, err
			}
			newest = decompress
		}
	}
	// 只有解压缩操作时, 不需要额外执行strip_ansi, 交给下游做
	if splitJsonIdx == -1 {
		return []interface{}{newest}, nil
	}
	if len(newest) == 0 {
		newest = old
	}
	// 否则, 先执行strip_ansi
	if stripAnsiIdx != -1 {
		newest = RemoveAnsiEscape(newest)
	}
	// 再执行json split
	slice, err := SplitJsonArray(newest)
	if err != nil {
		return []interface{}{}, nil
	}
	// split后的结果, 如果需要额外的过滤操作, 则全部转化为[]string{}
	if hasFilter || (len(forceTostr) > 0 && forceTostr[0]) {
		var strs []interface{}
		for i := range slice {
			if slice[i] == nil {
				continue
			}
			if reflect.TypeOf(slice[i]).Kind() == reflect.String {
				strs = append(strs, slice[i])
			} else {
				jsonstr, err := json.Marshal(slice[i])
				if err != nil {
					// 暂时不报错
					continue
				}
				strs = append(strs, string(jsonstr))
			}
		}
		return strs, nil
	}

	return slice, nil
}

// newest, isValid
// 日志提取的前置处理: 移除特殊字符、包含/排除
func StringPreExtractHandle(old string, funcs []*StringPreFunction, enableKvMatch bool) (string, bool) {
	hasConvert := false
	for _, pfunc := range funcs {
		if pfunc.Mode == GlobalFuncStripAnsi ||
			pfunc.Mode == GlobalFuncFormatToJson {
			hasConvert = true
			continue
		}
		if pfunc.IsFiltered(old) {
			return old, false
		}
	}

	if enableKvMatch {
		//原始日志反序列化至level0
		level0 := make(map[string]interface{})
		if err := json.Unmarshal([]byte(old), &level0); err != nil {
			return old, false
		}
		//Key-Value对的匹配与排除
		if len(funcs) > 0 {
			for _, field := range funcs {
				if field.Mode == GlobalFuncKeyValueMatch {
					if !field.KvMatchOrNotMatch(level0) {
						return old, false
					}
				} else if field.Mode == GlobalFuncKeyValueNotMatch {
					if field.KvMatchOrNotMatch(level0) {
						return old, false
					}
				}
			}
		}
	}

	if !hasConvert {
		return old, true
	}
	newest := old
	for _, pfunc := range funcs {
		if pfunc.Mode == GlobalFuncStripAnsi ||
			pfunc.Mode == GlobalFuncFormatToJson {
			newest = pfunc.Convert(newest)
		}
	}
	return newest, true
}

func (f *StringPreFunction) Compile() error {
	if f.isCompiled {
		return nil
	}
	if f.Mode == GlobalFuncFormatToJson ||
		f.Mode == GlobalFuncStripAnsi ||
		f.Mode == GlobalFuncSplitJsonArray {
		f.isCompiled = true
		return nil
	}
	if len(f.Format) == 0 {
		return ErrorJsonExtractParamsMissingFn("function.format")
	}
	if f.Mode == GlobalFuncStringMatch || f.Mode == GlobalFuncStringNotMatch {
		var err error
		f.regexp, err = regexp.Compile(f.Format)
		if err != nil {
			return err
		}
	}
	if f.Mode == GlobalFuncKeyValueMatch || f.Mode == GlobalFuncKeyValueNotMatch {
		err := f.KvMatchPreProcess()
		if err != nil {
			return err
		}
	}

	f.isCompiled = true
	return nil
}

// 是否被过滤掉
func (f *StringPreFunction) IsFiltered(old string) bool {
	if !f.isCompiled {
		return false
	}
	switch f.Mode {
	case GlobalFuncStringMatch:
		return !f.regexp.MatchString(old)

	case GlobalFuncStringNotMatch:
		return f.regexp.MatchString(old)

	case GlobalFuncStringContains:
		return !strings.Contains(old, f.Format)

	case GlobalFuncStringNotContains:
		return strings.Contains(old, f.Format)
	}
	return false
}

func (f *StringPreFunction) Convert(old string) string {
	if f.Mode == GlobalFuncStripAnsi && len(old) > 0 {
		return RemoveAnsiEscape(old)
	}
	if f.Mode == GlobalFuncFormatToJson && len(old) > 0 {
		return FormatToJson(old)
	}
	return old
}

func (f *StringPreFunction) KvMatchOrNotMatch(old map[string]interface{}) bool {
	if f.kvMatch == nil {
		if f.Mode == GlobalFuncKeyValueMatch {
			return false
		} else if f.Mode == GlobalFuncKeyValueNotMatch {
			return true
		}
	}
	stack := []map[string]interface{}{old}
	//level为当前层数
	level := -1
	for len(stack) > 0 {
		level++
		data := stack[len(stack)-1]
		stack = stack[:len(stack)-1]

		for key, value := range data {
			switch v := value.(type) {
			// 当前v是map类型，说明还有下一层
			case map[string]interface{}:
				//当前层数 == 需匹配路径key的层数 且 当前key == 所需key
				if level == f.kvMatch.index[key] && f.kvMatch.subParts[level] == key {
					stack = append(stack, v)
				}
			default:
				if level == f.kvMatch.index[key] && f.kvMatch.subParts[level] == key {
					if f.kvMatch.matchValue == fmt.Sprintf("%v", v) {
						return true
					} else {
						return false
					}
				} else {
					continue
				}
			}
		}
	}
	return false
}

func (f *StringPreFunction) KvMatchPreProcess() error {
	format := f.Format
	parts := strings.Split(format, "=")

	//保证输入格式合法
	//输入格式为xxx.xx=x
	if len(parts) != 2 {
		return ErrorJsonExtractParamsMissingFn("function.format")
	}

	parts[0] = strings.TrimSpace(parts[0])
	parts[1] = strings.TrimSpace(parts[1])

	//将匹配路径分割
	//xxx.xx -> xxx xx
	subParts := strings.Split(parts[0], ".")
	index := make(map[string]int)
	for i := 0; i < len(subParts); i++ {
		subParts[i] = strings.TrimSpace(subParts[i])
		index[subParts[i]] = i
	}
	f.kvMatch = &KvMatch{
		index:      index,
		subParts:   subParts,
		matchValue: parts[1],
	}
	return nil
}

func checkLogCompress(oldLog []byte) (string, bool) {
	old := fmt.Sprintf("%x", oldLog)
	if strings.HasPrefix(old, RFCHeaderGzip) {
		return "gzip", true
	}
	if strings.HasPrefix(old, RFCHeaderZlib) {
		return "zlib", true
	}
	return "", false
}
