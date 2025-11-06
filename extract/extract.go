package extract

import (
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/flashcatcloud/fc-stash/transform/field_format"
	"github.com/flashcatcloud/fc-stash/transform/text_parser"
	"github.com/flashcatcloud/fc-stash/types"
)

const (
	LogExtractModeJson = "json" // json日志
)

const (
	LogExtractRuleTypeAppend   = "append"   // 新增字段, 要求父节点都必须存在
	LogExtractRuleTypeSubMatch = "submatch" // 类型转化(含嫁接), 从源字段X提取出来, 嫁接到目标Y, 如果X==Y则不处理
	LogExtractRuleTypeRemove   = "remove"   // 删除
	LogExtractRuleTypeGraft    = "graft"    // 嫁接
)

const (
	DefaultLogExtractRegexp    = "(.*)"       // 全提取, 不需要编译正则
	DefaultLogAppendDateNow    = "NOW"        // 默认以当前时间填充
	DefaultDateFormatUnix      = "unix"       // unix时间戳
	DefaultDateFormatUnixMilli = "unix_milli" // unix milli时间戳
	DefaultDateFormatUnixMicro = "unix_micro" // unix micro时间戳
)

const (
	LogExtractValueTypeLong   = "long"
	LogExtractValueTypeFloat  = "float"
	LogExtractValueTypeText   = "text"
	LogExtractValueTypeDate   = "date"
	LogExtractValueTypeObject = "object"
	LogExtractValueTypeArray  = "array"
)

var (
	ErrorLogExtractEmptyFields           = errors.New("no fields was set for extracting")
	ErrorLogExtractRuleTypeNotSupported  = errors.New("extract rule type not supported")
	ErrorLogExtractValueTypeNotSupported = errors.New("extract value type not supported")
	ErrorJsonExtractValueTypeNotMatched  = errors.New("extract value type not matched")
	ErrorJsonExtractValueTypeNil         = errors.New("extract value type nil pointer")
	ErrorJsonExtractFieldNotMatched      = errors.New("extract field not matched")
	ErrorInvalidJsonString               = errors.New("invalid json string")
)

var (
	ErrorJsonExtractParamsMissingFn = func(param string) error {
		return fmt.Errorf("param[%v] missing for json extracting", param)
	}
	ErrorJsonExtractRequiredMissingFn = func(field string) error {
		return fmt.Errorf("field[%v] required, but not matched", field)
	}
)

// 复杂文本提取 textParser
// 支持JSON反序列化、geoip转换、正则表达式、gonx解析等
type JsonPreSourceExtract struct {
	Mode       string `json:"mode" yaml:"mode"`
	Field      string `json:"field" yaml:"field"`
	Format     string `json:"format" yaml:"format"`
	parser     types.TextParser
	isCompiled bool
}

// Deprecated, 旧版本依赖
// fc-stash在单个字段提取时的逻辑不够抽象, 很难灵活的加功能
// 单个字段的提取规则
type FieldExtract struct {
	RuleType       string      `json:"rule_type" yaml:"type"`        // sub_match/append
	OriginField    string      `json:"origin_field" yaml:"field"`    // 嵌套的json字段
	OriginValue    interface{} `json:"-"`                            // 内部处理时使用
	Key            string      `json:"key" yaml:"target"`            // 最终写入index的name
	ValueType      string      `json:"value_type" yaml:"value_type"` // long/float/text/date
	Extract        ExtractRule `json:"extention" yaml:"extention"`   //
	Required       bool        `json:"required"`                     // 是否必须, 如果匹配不到则丢弃
	SystemRequired bool        `json:"-"`                            // 系统必须, TO DELETE
	source         string      `json:"-"`
	targets        []string    `json:"-"`
}

// 处理规则
type ExtractRule struct {
	Regexp       string            `json:"regexp,omitempty" yaml:"regexp"`               // 提取规则
	regexpJSON   *regexp.Regexp    `json:"-"`                                            // 内部使用
	PresetFilter string            `json:"preset_filter,omitempty" yaml:"preset_filter"` // 预置的方法, 优先级高于 regexp
	filter       types.FieldParser `json:"-"`
	DefaultValue string            `json:"default_value"` // 默认填充

	DateFormat       string         `json:"date_format,omitempty" yaml:"date_format"`     // 不包含时区的字符串
	DateLocation     string         `json:"date_location,omitempty" yaml:"date_location"` // 解析时生效, 默认用UTC时间
	DateFixUTCOffset int            `json:"date_utc_offset,omitempty"`                    // 时区转化
	loc              *time.Location `json:"-"`
	fixedLoc         *time.Location `json:"-"`

	AppendValue string `json:"append_value,omitempty"`
}

func (field *FieldExtract) Compile() error {
	if field.Key == types.LogEventMessageKey {
		return errors.New(types.LogEventMessageKey + " is system keyword")
	}
	if field.RuleType == LogExtractRuleTypeAppend {
		if len(field.ValueType) == 0 {
			return ErrorJsonExtractParamsMissingFn("field.extract.value_type")
		}
		if len(field.Extract.AppendValue) == 0 {
			return ErrorJsonExtractParamsMissingFn("field.extract.append_value")
		}
		if len(field.Key) == 0 {
			return ErrorJsonExtractParamsMissingFn("field.extract.append_key")
		}
		if field.ValueType == LogExtractValueTypeDate { // append 新增时间默认用local时区
			field.Extract.loc = time.Local
		}

		// 目标是__root_.xxx, 实际按照xxx处理即可
		// 目标是 __root__, 需要在addByPath中处理
		keys := strings.Split(strings.TrimPrefix(field.Key, types.LogExtractJsonRoot+"."), ".")
		field.targets = keys[len(keys)-1:] // 最后一段
		if len(keys) > 1 {                 // 父节点
			field.source = strings.Join(keys[:len(keys)-1], ".")
		}
		// 适配 以复用 rule.ExtractSubMatch 逻辑
		field.Extract.Regexp = DefaultLogExtractRegexp
		return nil
	}
	// 源字段
	field.source = field.OriginField
	// 对于类型转换, 如果目标与源不一致, 需要嫁接
	// 对于嫁接, 如果目标与源一致, 则不操作
	if field.RuleType == LogExtractRuleTypeSubMatch ||
		field.RuleType == LogExtractRuleTypeGraft {
		fieldkey := strings.TrimPrefix(field.Key, types.LogExtractJsonRoot+".")
		if fieldkey != field.OriginField {
			// 目标是__root_.xxx, 实际按照xxx处理即可
			// 目标是 __root__, 需要在addByPath中处理
			field.targets = strings.Split(fieldkey, ".")
		}
	}
	if len(field.Extract.Regexp) > 0 {
		regexpStr := strings.TrimSpace(field.Extract.Regexp)
		// 全匹配, 不需要编译正则
		if regexpStr != DefaultLogExtractRegexp {
			regexp, err := regexp.Compile(regexpStr)
			if err != nil {
				return err
			}
			field.Extract.regexpJSON = regexp
		}
	}
	if len(field.Extract.PresetFilter) > 0 {
		preset, err := field_format.NewPresetFieldParser(field.Extract.PresetFilter)
		if err != nil {
			return err
		}
		field.Extract.filter = preset
	}
	if field.ValueType == LogExtractValueTypeDate {
		// submatch 必须提供有效的时间格式
		if field.RuleType == LogExtractRuleTypeSubMatch {
			if len(field.Extract.DateFormat) == 0 {
				return ErrorJsonExtractParamsMissingFn("field.extract.date_format")
			}
		}
		if len(field.Extract.DateLocation) > 0 {
			var err error
			field.Extract.loc, err = time.LoadLocation(field.Extract.DateLocation)
			if err != nil {
				return errors.New("field[" + field.Key + "] illegal date_location:" + field.Extract.DateLocation)
			}
		} else {
			field.Extract.loc = time.UTC // submatch 解析时间默认用UTC时区
		}
		if field.Extract.DateFixUTCOffset != 0 {
			field.Extract.fixedLoc = time.FixedZone("UTC", field.Extract.DateFixUTCOffset)
		}
	}
	return nil
}

func (field *FieldExtract) ExtractSubMatch(v interface{}) (interface{}, error) {
	var useDefaultIfNil bool
	if v == nil {
		// 设置了默认值, 则按照默认值继续处理
		if len(field.Extract.DefaultValue) > 0 {
			useDefaultIfNil = true
			v = "" // set to empty string
		} else {
			// 必须字段, 则报错missing
			if field.Required {
				return nil, ErrorJsonExtractRequiredMissingFn(field.Key)
			}
			// 否则报一个value nil的错误, 交给上层处理
			return nil, ErrorJsonExtractValueTypeNil
		}
	}
	// 字符串类型需要提取
	vtype := reflect.TypeOf(v).Kind()
	if vtype == reflect.String {
		str, ok := v.(string)
		if !ok {
			return nil, ErrorJsonExtractValueTypeNotMatched
		}
		var matched string
		if useDefaultIfNil {
			matched = field.Extract.DefaultValue
		} else if field.Extract.filter != nil {
			var err error
			matched, err = field.Extract.filter.ParseString(str)
			if err != nil {
				return nil, errors.New("invalid filter[" + field.Extract.PresetFilter + "] string")
			}
		} else if field.Extract.Regexp == DefaultLogExtractRegexp {
			matched = str
		} else if field.Extract.regexpJSON != nil {
			submatch := field.Extract.regexpJSON.FindStringSubmatch(str)
			if len(submatch) > 1 {
				matched = submatch[1]
			}
		}
		if len(matched) == 0 && len(field.Extract.DefaultValue) > 0 {
			matched = field.Extract.DefaultValue
		}
		if len(matched) == 0 && field.Required {
			return nil, ErrorJsonExtractRequiredMissingFn(field.Key)
		}
		switch field.ValueType {
		case LogExtractValueTypeText:
			// 字符串类型允许为空
			return matched, nil

		case LogExtractValueTypeLong:
			// 整型的默认值
			if len(matched) == 0 {
				return 0, nil
			}
			// 整型解析失败, 按照0值处理
			v, err := strconv.ParseInt(matched, 10, 64)
			if err != nil {
				return 0, nil
			}
			return v, nil

		case LogExtractValueTypeFloat:
			// 浮点数的默认值
			if len(matched) == 0 {
				return 0.0, nil
			}
			// 浮点数解析失败, 按照0.0值处理
			v, err := strconv.ParseFloat(matched, 64)
			if err != nil {
				return 0.0, nil
			}
			return v, nil

		case LogExtractValueTypeDate:
			// 时间类型不允许为空
			if len(matched) == 0 {
				return nil, ErrorJsonExtractFieldNotMatched
			}
			if len(field.Extract.DateFormat) == 0 {
				return nil, ErrorJsonExtractParamsMissingFn("field.extract.date_format")
			}
			if matched == DefaultLogAppendDateNow {
				return time.Now(), nil
			}
			var t time.Time
			var err error
			if field.Extract.DateFormat == DefaultDateFormatUnix {
				sec, err := strconv.ParseInt(matched, 10, 64)
				if err != nil {
					return nil, errors.New("invalid unix sec interger")
				}
				t = time.Unix(sec, 0)

			} else if field.Extract.DateFormat == DefaultDateFormatUnixMilli {
				msec, err := strconv.ParseInt(matched, 10, 64)
				if err != nil {
					return nil, errors.New("invalid unix_milli sec interger")
				}
				t = time.UnixMilli(msec)

			} else if field.Extract.DateFormat == DefaultDateFormatUnixMicro {
				micro, err := strconv.ParseInt(matched, 10, 64)
				if err != nil {
					return nil, errors.New("invalid unix_micro sec interger")
				}
				t = time.UnixMicro(micro)

			} else if field.Extract.loc != nil {
				t, err = time.ParseInLocation(
					field.Extract.DateFormat, matched, field.Extract.loc)
			} else {
				t, err = time.Parse(field.Extract.DateFormat, matched)
			}
			if err != nil {
				return nil, errors.New("invalid date string")
			}
			if field.Extract.fixedLoc != nil {
				return t.In(field.Extract.fixedLoc), nil
			}
			return t, nil
		}
		return nil, ErrorLogExtractValueTypeNotSupported
	}
	// 数字类型直接转换
	if vtype == reflect.Float64 {
		num, ok := v.(float64)
		if !ok {
			return nil, ErrorJsonExtractValueTypeNotMatched
		}

		if field.ValueType == LogExtractValueTypeLong {
			return int64(num), nil
		}
		if field.ValueType == LogExtractValueTypeFloat {
			return num, nil
		}
		if field.ValueType == LogExtractValueTypeText {
			return strconv.FormatFloat(num, 'f', -1, 64), nil
		}
		if field.ValueType == LogExtractValueTypeDate {
			var t time.Time
			if field.Extract.DateFormat == DefaultDateFormatUnix {
				t = time.Unix(int64(num), 0)
			} else if field.Extract.DateFormat == DefaultDateFormatUnixMilli {
				t = time.UnixMilli(int64(num))
			} else if field.Extract.DateFormat == DefaultDateFormatUnixMicro {
				t = time.UnixMicro(int64(num))
			} else {
				return nil, errors.New("unknown unix date format")
			}
			if field.Extract.fixedLoc != nil {
				return t.In(field.Extract.fixedLoc), nil
			} else if field.Extract.loc != nil {
				return t.In(field.Extract.loc), nil
			}
			return t, nil
		}
	}
	return nil, ErrorLogExtractValueTypeNotSupported
}

// TODO: 替换, 包括数据脱敏和旧数据处理
func (le *FieldExtract) ExtractAppend() (interface{}, error) {
	if len(le.Extract.AppendValue) == 0 {
		return nil, ErrorJsonExtractParamsMissingFn(le.Key + ".append_value")
	}
	v := le.Extract.AppendValue
	if le.ValueType == LogExtractValueTypeText {
		return v, nil
	}
	// 新增时间字段, 取值是最新时刻
	if le.ValueType == LogExtractValueTypeDate &&
		le.Extract.AppendValue == DefaultLogAppendDateNow {
		return time.Now(), nil
	}
	// 非text的提取
	// 复用提取的逻辑, le.Compile() 已确保该步骤不会报错
	return le.ExtractSubMatch(v)
}

func (pre *JsonPreSourceExtract) Compile(forceOrigin bool) error {
	if pre.isCompiled {
		return nil
	}
	parser, err := text_parser.NewPresetTextParser(pre.Mode, pre.Format, forceOrigin)
	if err != nil {
		return errors.New("field[" + pre.Field + "] invalid format:" + err.Error())
	}
	pre.parser = parser
	pre.isCompiled = true
	return nil
}

// map结构按照json path打平, 主要用于网页处理
func JsonFlattenFromMap(m map[string]interface{}) (fields []string, values []interface{}, types []reflect.Kind) {
	for key, value := range m {
		if value == nil {
			continue
		}
		switch reflect.TypeOf(value).Kind() {
		case reflect.Map:
			fields = append(fields, key)
			values = append(values, map[string]interface{}{}) // set to nil
			types = append(types, reflect.Map)

			subFields, subValues, subTypes := JsonFlattenFromMap(value.(map[string]interface{}))

			if len(subFields) > 0 && len(subFields) == len(subValues) {
				for i, field := range subFields {
					fields = append(fields, key+"."+field)
					values = append(values, subValues[i])
					types = append(types, subTypes[i])
				}
			}
		default:
			fields = append(fields, key)
			values = append(values, value)
			types = append(types, reflect.TypeOf(value).Kind())
		}
	}
	return
}

func JsonFlattenTypeString(origin reflect.Kind) string {
	name := origin.String()
	if strings.Contains(name, "int") {
		return LogExtractValueTypeLong
	} else if strings.Contains(name, "float") {
		return LogExtractValueTypeFloat
	} else if name == "string" {
		return LogExtractValueTypeText
	} else if name == "slice" || name == "array" {
		return LogExtractValueTypeArray
	}
	// map/struct/bool etc.
	return LogExtractValueTypeObject
}
