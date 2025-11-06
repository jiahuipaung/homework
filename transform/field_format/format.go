package field_format

import (
	"errors"
	"reflect"
	"regexp"
	"strconv"
	"time"

	"github.com/flashcatcloud/fc-stash/types"
)

const (
	ConvertToUTC8Offset = 28800 // UTC时间转化为UTC+8时间, 8小时
)

const (
	DefaultDateFormatUnix      = "unix"       // unix时间戳
	DefaultDateFormatUnixMilli = "unix_milli" // unix milli时间戳
	DefaultDateFormatUnixMicro = "unix_micro" // unix micro时间戳
)

type FieldFormatSettings struct {
	FormatType    string                `json:"format_type,omitempty"` // 全部提取/正则提取/URL归一化/数据脱敏/正则映射
	TimeFormat    TimeFormatSettings    `json:"time_format"`           // 时间格式化
	Regexp        string                `json:"regexp,omitempty"`      // 正则提取的参数
	Desensitize   DesensitizeSettings   `json:"desensitize"`
	RegexpMapping RegexpMappingSettings `json:"regexp_mapping"`
}

// 时间格式化
type TimeFormatSettings struct {
	DateFormat       string `json:"date_format,omitempty"`     // 不包含时区的字符串
	DateLocation     string `json:"date_location,omitempty"`   // 解析时生效, 默认用UTC时间
	ConvertToUTC8    bool   `json:"convert_to_utc8,omitempty"` // 是否转换为UTC+8时间
	DateFixUTCOffset int    `json:"-"`                         // 时区转化
}

// 字段格式化, 根据参数执行对应的操作
type FieldFormat struct {
	Settings      FieldFormatSettings
	TargetType    string
	isAllMatch    bool
	regexp        *regexp.Regexp  // 正则提取的表达式
	uripath       *UriPathExtract // URI路径的执行器
	desensitize   *Desensitize    // 数据脱敏的执行器
	regexpMapping *RegexpMapping  // 正则映射的执行器
	dateFormat    string          // 时间格式化, 用于解析时间字符串, timeFormat保存的是linux语法, 转换成golang语法
	loc           *time.Location  // time_format.DateLocation的解析结果(如果有)
	fixedLoc      *time.Location  // time_format.DateFixUTCOffset的解析结果(如果有)
}

var (
	ErrorLogFormatRuleTypeNotSupported   = errors.New("format type not supported")
	ErrorLogFormatOriginTypeNotSupported = errors.New("origin value type not supported")
	ErrorLogFormatTargetTypeNotSupported = errors.New("target value type not supported")
	ErrorLogFormatValueToStringFailed    = errors.New("convert value to string failed")
	ErrorLogFormatValueToFloatFailed     = errors.New("convert value to float failed")
	ErrorLogFormatValueTypeNil           = errors.New("value nil pointer")
	ErrorLogFormatValueEmpty             = errors.New("value was empty")
)

func NewFieldFormat(settings FieldFormatSettings, targetType string) (*FieldFormat, error) {
	formatter := &FieldFormat{
		Settings:   settings,
		TargetType: targetType,
	}
	if len(targetType) == 0 {
		return nil, errors.New("target type is required")
	}
	var err error
	switch settings.FormatType {
	case PresetAllmatch:
		formatter.isAllMatch = true
	case PresetRegexp:
		formatter.regexp, err = regexp.Compile(settings.Regexp)
		if err != nil {
			return nil, err
		}

	case PresetUriPath:
		formatter.uripath, err = newUriPathExtract(uriPathPatternFile)
		if err != nil {
			return nil, err
		}

	case PresetDesensitize:
		formatter.desensitize, err = NewDesensitize(settings.Desensitize)
		if err != nil {
			return nil, err
		}

	case PresetRegexpMapping:
		formatter.regexpMapping, err = NewRegexpMapping(settings.RegexpMapping)
		if err != nil {
			return nil, err
		}
	}
	// 如果valueType是日期类型, 则需要解析时间格式
	if targetType == types.LogExtractValueTypeDate {
		if len(settings.TimeFormat.DateFormat) == 0 {
			return nil, errors.New("date_format is required")
		}
		formatter.dateFormat = ConvertLinuxDateFormatToGolang(settings.TimeFormat.DateFormat)
		if len(settings.TimeFormat.DateLocation) > 0 {
			formatter.loc, err = time.LoadLocation(settings.TimeFormat.DateLocation)
			if err != nil {
				return nil, err
			}
		} else {
			formatter.loc = time.UTC // 解析时间默认用UTC时区
		}
		if settings.TimeFormat.ConvertToUTC8 {
			formatter.fixedLoc = time.FixedZone("UTC", ConvertToUTC8Offset)
		}
	}
	return formatter, nil
}

// 对value进行格式化, 根据valueType, 转换成对应的类型
func (f *FieldFormat) Format(value interface{}) (interface{}, error) {
	// 不支持空值
	if value == nil {
		return nil, ErrorLogFormatValueTypeNil
	}
	// 原始值只支持两种类型: string和float64
	// KAFKA的消息经过golang json反序列化后, 会变成map[string]interface{}
	// value的类型主要是string和float64, array/map不再这里支持

	vtype := reflect.TypeOf(value).Kind()
	// 如果原始值是float64, 并且format_type是PresetAllmatch, 则直接处理
	if vtype == reflect.Float64 && f.Settings.FormatType == PresetAllmatch {
		num, ok := value.(float64)
		if !ok {
			return nil, ErrorLogFormatValueToFloatFailed
		}

		switch f.TargetType {
		case types.LogExtractValueTypeLong:
			return int64(num), nil

		case types.LogExtractValueTypeFloat:
			return num, nil

		case types.LogExtractValueTypeText:
			return strconv.FormatFloat(num, 'f', -1, 64), nil

		case types.LogExtractValueTypeDate:
			var t time.Time
			if f.dateFormat == DefaultDateFormatUnix {
				t = time.Unix(int64(num), 0)
			} else if f.dateFormat == DefaultDateFormatUnixMilli {
				t = time.UnixMilli(int64(num))
			} else if f.dateFormat == DefaultDateFormatUnixMicro {
				t = time.UnixMicro(int64(num))
			} else {
				return nil, errors.New("unknown unix date format")
			}
			if f.fixedLoc != nil {
				return t.In(f.fixedLoc), nil
			} else if f.loc != nil {
				return t.In(f.loc), nil
			}
			return t, nil
		}
		return nil, ErrorLogFormatTargetTypeNotSupported
	}

	// 如果原始值是string, 则需要根据format_type, 执行对应的操作
	// 如果原始值是float64, 且format_type不是PresetAllmatch, 则需要转换成string, 再执行对应的操作
	if vtype == reflect.String || vtype == reflect.Float64 {
		var formatted string
		var ok bool
		if vtype == reflect.String {
			formatted, ok = value.(string)
			if !ok {
				return nil, ErrorLogFormatValueToStringFailed
			}
		} else {
			num, ok := value.(float64)
			if !ok {
				return nil, ErrorLogFormatValueToFloatFailed
			}
			formatted = strconv.FormatFloat(num, 'f', -1, 64)
		}

		var err error
		// 根据format_type, 执行对应的操作
		switch f.Settings.FormatType {
		case PresetAllmatch: // 不处理

		case PresetRegexp:
			submatch := f.regexp.FindStringSubmatch(formatted)
			if len(submatch) > 1 {
				formatted = submatch[1]
			}

		case PresetUriPath:
			formatted, err = f.uripath.ParseString(formatted)
			if err != nil {
				return nil, err
			}

		case PresetDesensitize:
			formatted, err = f.desensitize.ParseString(formatted)
			if err != nil {
				return nil, err
			}

		case PresetRegexpMapping:
			formatted, err = f.regexpMapping.ParseString(formatted)
			if err != nil {
				return nil, err
			}

		default:
			return nil, ErrorLogFormatRuleTypeNotSupported
		}

		switch f.TargetType {
		case types.LogExtractValueTypeText:
			// 字符串类型允许为空
			return formatted, nil

		case types.LogExtractValueTypeLong:
			// 整型的默认值
			if len(formatted) == 0 {
				return 0, nil
			}
			// 整型解析失败, 按照0值处理
			v, err := strconv.ParseInt(formatted, 10, 64)
			if err != nil {
				return 0, nil
			}
			return v, nil

		case types.LogExtractValueTypeFloat:
			// 浮点数的默认值
			if len(formatted) == 0 {
				return 0.0, nil
			}
			// 浮点数解析失败, 按照0.0值处理
			v, err := strconv.ParseFloat(formatted, 64)
			if err != nil {
				return 0.0, nil
			}
			return v, nil

		case types.LogExtractValueTypeDate:
			// 时间类型不允许为空
			if len(formatted) == 0 {
				return nil, ErrorLogFormatValueEmpty
			}
			if len(f.Settings.TimeFormat.DateFormat) == 0 {
				return nil, errors.New("date_format is required")
			}

			var t time.Time
			var err error
			if f.dateFormat == DefaultDateFormatUnix {
				sec, err := strconv.ParseInt(formatted, 10, 64)
				if err != nil {
					return nil, errors.New("invalid unix sec interger")
				}
				t = time.Unix(sec, 0)

			} else if f.dateFormat == DefaultDateFormatUnixMilli {
				msec, err := strconv.ParseInt(formatted, 10, 64)
				if err != nil {
					return nil, errors.New("invalid unix_milli sec interger")
				}
				t = time.UnixMilli(msec)

			} else if f.dateFormat == DefaultDateFormatUnixMicro {
				micro, err := strconv.ParseInt(formatted, 10, 64)
				if err != nil {
					return nil, errors.New("invalid unix_micro sec interger")
				}
				t = time.UnixMicro(micro)

			} else if f.loc != nil {
				t, err = time.ParseInLocation(
					f.dateFormat, formatted, f.loc)
			} else {
				t, err = time.Parse(f.dateFormat, formatted)
			}
			if err != nil {
				return nil, errors.New("invalid date string")
			}
			if f.fixedLoc != nil {
				return t.In(f.fixedLoc), nil
			}
			return t, nil
		}
		return nil, ErrorLogFormatTargetTypeNotSupported
	}
	return nil, ErrorLogFormatOriginTypeNotSupported
}
