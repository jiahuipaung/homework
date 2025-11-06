package text_parser

import (
	"encoding/json"
	"errors"
	"regexp"
	"strings"

	"github.com/flashcatcloud/fc-stash/types"
)

var (
	PresetKeyValue = "key_value"
)

const (
	PresetKeyValueModeTable            = "table"          // 表结构, key在表头
	PresetKeyValueModeVerticalTable    = "vertical_table" // 纵向表结构, key在某列
	PresetKeyValueModeDelimiterInValue = "delimiter_in_value"
)

func init() {
	registerParser(PresetKeyValue, NewKeyValueParser)
}

type KeyValueParser struct {
	args       KeyValueArgument
	needOrigin bool
}

// 暂时不支持正则
type KeyValueArgument struct {
	FieldDelimiter    string         `json:"field_delimiter"`     // Default: ","
	KeyValueDelimiter string         `json:"key_value_delimiter"` // Default: "="
	KeyPattern        string         `json:"key_pattern"`         // key的正则,默认不存在
	keyPatternReg     *regexp.Regexp `json:"-"`                   //

	// 特殊的key/value格式处理
	Mode string `json:"mode"`
	// 表格通用
	TableRowDelimiter     string `json:"table_row_delimiter"` // 每行的分隔符
	TableKeyTrimPattern   string `json:"table_key_trim"`      // 去掉非预期的前后缀
	TableValueTrimPattern string `json:"table_value_trim"`    // 去掉非预期的前后缀
	// 横向表格专用
	TableKeyDelimiter    string         `json:"table_key_delimiter"`   // 表头的列分隔符
	TableValueDelimiter  string         `json:"table_value_delimiter"` // 数据的列分隔符
	TableValueMultiline  bool           `json:"table_value_multiline"` // 多行, 都要处理
	TableValuePattern    string         `json:"table_value_pattern"`   // 数据列的匹配, 用于排除无效的行
	tableValuePatternReg *regexp.Regexp `json:"-"`                     // 内部使用

	// 纵向表格专用
	TableColumnDelimiter  string `json:"table_column_delimiter"` // 纵向表格的列分隔符
	TableKeyColumnIndex   int    `json:"table_key_index"`        // 纵向表格的
	TableValueColumnIndex int    `json:"table_value_index"`      // 纵向表格的
}

func NewKeyValueParser(format string, needOrigin bool) (types.TextParser, error) {
	parser := &KeyValueParser{}
	var args KeyValueArgument
	if len(format) > 0 {
		if err := json.Unmarshal([]byte(format), &args); err != nil {
			return nil, err
		}
	}
	// 设置默认分隔符
	if len(args.FieldDelimiter) == 0 {
		args.FieldDelimiter = ","
	}
	if len(args.KeyValueDelimiter) == 0 {
		args.KeyValueDelimiter = "="
	}
	if len(args.KeyPattern) > 0 {
		var err error
		// 自动补齐 ^{}$, 确保强匹配
		if !strings.HasPrefix(args.KeyPattern, "^") {
			args.KeyPattern = "^" + args.KeyPattern
		}
		if !strings.HasSuffix(args.KeyPattern, "$") {
			args.KeyPattern = args.KeyPattern + "$"
		}
		args.keyPatternReg, err = regexp.Compile(args.KeyPattern)
		if err != nil {
			return nil, err
		}
	}
	if len(args.Mode) > 0 {
		if args.Mode == PresetKeyValueModeTable {
			// 设置默认分隔符
			if len(args.TableRowDelimiter) == 0 {
				args.TableRowDelimiter = "\n"
			}
			if len(args.TableKeyDelimiter) == 0 {
				args.TableKeyDelimiter = "|"
			}
			if len(args.TableValueDelimiter) == 0 {
				args.TableValueDelimiter = "|"
			}
			if args.TableValueMultiline && len(args.TableValuePattern) > 0 {
				var err error
				args.tableValuePatternReg, err = regexp.Compile(args.TableValuePattern)
				if err != nil {
					return nil, err
				}
			}
		}
		if args.Mode == PresetKeyValueModeVerticalTable {
			// 设置默认分隔符
			if len(args.TableRowDelimiter) == 0 {
				args.TableRowDelimiter = "\n"
			}
			if len(args.TableColumnDelimiter) == 0 {
				args.TableColumnDelimiter = "\t"
			}
			if args.TableKeyColumnIndex == 0 && args.TableValueColumnIndex == 0 {
				args.TableKeyColumnIndex = 0
				args.TableValueColumnIndex = 1
			}
			if args.TableKeyColumnIndex == args.TableValueColumnIndex ||
				args.TableKeyColumnIndex < 0 || args.TableValueColumnIndex < 0 {
				return nil, errors.New("key/value的index参数错误")
			}
		}
	}
	parser.args = args
	parser.needOrigin = needOrigin
	return parser, nil
}

func (p *KeyValueParser) Name() string {
	return PresetKeyValue
}

func (p *KeyValueParser) ParseModeTable(origin string) (valueIndex map[string][]string, ordered []string) {
	if p.args.Mode != PresetKeyValueModeTable {
		return
	}
	lines := strings.Split(origin, p.args.TableRowDelimiter)
	if len(lines) < 2 { // 分割出错不用报错
		return
	}
	// 多行的处理
	valueIndex = make(map[string][]string)
	var keys []string
	if p.args.TableValueMultiline {
		keys = strings.Split(lines[0], p.args.TableKeyDelimiter)
		for i := range keys {
			if len(keys[i]) == 0 {
				continue
			}
			var key string = keys[i]
			if len(p.args.TableKeyTrimPattern) > 0 {
				key = strings.Trim(key, p.args.TableKeyTrimPattern)
			}
			if len(key) == 0 {
				keys[i] = ""
				continue
			}
			keys[i] = key
		}
		for i := 1; i < len(lines); i++ {
			if len(lines[i]) == 0 {
				continue
			}
			if p.args.tableValuePatternReg != nil &&
				!p.args.tableValuePatternReg.MatchString(lines[i]) {
				continue
			}
			values := strings.Split(lines[i], p.args.TableValueDelimiter)
			if len(values) == 0 {
				continue
			}
			for j := range keys {
				key := keys[j]
				if len(key) == 0 {
					continue
				}
				if j < len(values) {
					var value string = values[j]
					if len(p.args.TableValueTrimPattern) > 0 {
						value = strings.Trim(value, p.args.TableValueTrimPattern)
					}
					valueIndex[key] = append(valueIndex[key], value)
				} else {
					valueIndex[key] = append(valueIndex[key], "")
				}
			}
		}
	} else {
		// 默认只解析一行数据
		keys = strings.Split(lines[0], p.args.TableKeyDelimiter)
		values := strings.Split(lines[1], p.args.TableValueDelimiter)
		for i := range keys {
			if len(keys[i]) == 0 {
				continue
			}
			var key string = keys[i]
			if len(p.args.TableKeyTrimPattern) > 0 {
				key = strings.Trim(key, p.args.TableKeyTrimPattern)
			}
			if len(key) == 0 {
				continue
			}
			if i < len(values) {
				var value string = values[i]
				if len(p.args.TableValueTrimPattern) > 0 {
					value = strings.Trim(value, p.args.TableValueTrimPattern)
				}
				valueIndex[key] = []string{value}
			} else {
				valueIndex[key] = []string{""}
			}
		}
	}
	for _, key := range keys {
		if len(key) == 0 {
			continue
		}
		if values, found := valueIndex[key]; found && len(values) > 0 {
			ordered = append(ordered, key)
		}
	}
	return
}

func (p *KeyValueParser) Parse(origin string) (map[string]interface{}, error) {
	if len(origin) == 0 {
		return nil, nil
	}
	ret := make(map[string]interface{})
	if p.needOrigin {
		ret[types.LogParserOriginText] = origin // 原始日志
	}
	if len(p.args.Mode) > 0 {
		if p.args.Mode == PresetKeyValueModeTable {
			valueIndex, _ := p.ParseModeTable(origin)
			if len(valueIndex) > 0 {
				for key, values := range valueIndex {
					if len(key) == 0 || len(values) == 0 {
						continue
					}
					if p.args.TableValueMultiline {
						ret[key] = values
					} else {
						ret[key] = values[0]
					}
				}
			}
		}
		if p.args.Mode == PresetKeyValueModeVerticalTable {
			lines := strings.Split(origin, p.args.TableRowDelimiter)
			for _, line := range lines {
				columns := strings.Split(line, p.args.TableColumnDelimiter)
				if len(columns) <= p.args.TableKeyColumnIndex ||
					len(columns) <= p.args.TableValueColumnIndex {
					continue
				}
				key := columns[p.args.TableKeyColumnIndex]
				value := columns[p.args.TableValueColumnIndex]
				if len(p.args.TableKeyTrimPattern) > 0 {
					key = strings.Trim(key, p.args.TableKeyTrimPattern)
				}
				if len(key) == 0 {
					continue
				}
				if len(p.args.TableValueTrimPattern) > 0 {
					value = strings.Trim(value, p.args.TableValueTrimPattern)
				}
				ret[key] = value
			}
		}
		return ret, nil
	}

	if p.args.keyPatternReg != nil {
		items, err := p.ParseWithKeyPattern(origin)
		if err != nil {
			return nil, err
		}
		return items, nil
	}

	err := SimpleParseKeyValue(ret, origin, p.args.FieldDelimiter, p.args.KeyValueDelimiter)
	if err != nil {
		return nil, err
	}

	return ret, nil
}

func (p *KeyValueParser) ParseWithKeyPattern(origin string) (map[string]interface{}, error) {
	parts := strings.Split(origin, p.args.FieldDelimiter)
	result := make([][2]string, 0)
	for _, part := range parts {
		values := strings.SplitN(part, p.args.KeyValueDelimiter, 2)
		if len(values) == 2 {
			if p.args.keyPatternReg.MatchString(values[0]) {
				result = append(result, [2]string{strings.TrimSpace(values[0]), values[1]})
				continue
			}
		}
		// 匹配失败, 整体追加到上一条记录的 .value 字段
		lastidx := len(result) - 1
		if lastidx >= 0 {
			result[lastidx][1] = strings.Join([]string{result[lastidx][1], part}, p.args.FieldDelimiter)
		}
	}
	ret := make(map[string]interface{})
	if p.needOrigin {
		ret[types.LogParserOriginText] = origin // 原始日志
	}
	for i := range result {
		if len(result[i][0]) > 0 {
			ret[result[i][0]] = result[i][1]
		}
	}

	return ret, nil
}

// 用于处理简单的情况
func SimpleParseKeyValue(ret map[string]interface{}, origin, fieldDelimiter, keyValueDelimiter string) error {
	for _, field := range strings.Split(origin, fieldDelimiter) {
		values := strings.SplitN(field, keyValueDelimiter, 2)
		if len(values) == 2 {
			ret[strings.TrimSpace(values[0])] = values[1]
		}
	}
	return nil
}
