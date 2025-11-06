package field_format

import (
	"crypto/md5"
	"encoding/hex"
	"errors"
	"regexp"
)

// 数据脱敏
const (
	PresetDesensitize = "desensitize"
)

// 内置的脱敏方法
const (
	DesensitizeMethodPhone     = "phone"      // 手机号脱敏
	DesensitizeMethodEmail     = "email"      // 邮箱脱敏
	DesensitizeMethodIP        = "ip"         // IP脱敏
	DesensitizeMethodBankCard  = "bank_card"  // 银行卡号脱敏
	DesensitizeMethodIDCard    = "id_card"    // 身份证号脱敏
	DesensitizeMethodAccessKey = "access_key" // 访问密钥脱敏
	// 静态方法
	DesensitizeMethodKeepFirstLast = "keep_first_last" // 首尾各保留一个字符
	DesensitizeMethodKeepLastFour  = "keep_last_four"  // 结尾保留四个字符
	DesensitizeMethodMd5           = "md5"             // md5脱敏
)
const (
	DesensitizeMethodCustom = "custom" // 自定义脱敏, 自定义表达式
)

// 数据脱敏
type DesensitizeSettings struct {
	Method  string `json:"method,omitempty"`  // 脱敏方法
	Regexp  string `json:"regexp,omitempty"`  // 自定义正则表达式
	Replace string `json:"replace,omitempty"` // 替换规则
}

type Desensitize struct {
	method  string         // 脱敏方法
	regexp  *regexp.Regexp // 脱敏的正则表达式
	replace string         // 替换规则
}

var (
	builtinDesensitize = make(map[string]*Desensitize) // 内置的脱敏方法
)

func init() {
	builtinDesensitize[DesensitizeMethodPhone] = &Desensitize{
		regexp:  regexp.MustCompile(`([\+86]*1[3-9]{1}\d{1})\d{4}(\d{4,11})`),
		replace: `$1****$2`,
	}
	builtinDesensitize[DesensitizeMethodEmail] = &Desensitize{
		regexp:  regexp.MustCompile(`[A-Za-z\d]+([-_.][A-Za-z\d]+)*(@([A-Za-z\d]+[-.])+[A-Za-z\d]{2,4})`),
		replace: `****$2`,
	}
	builtinDesensitize[DesensitizeMethodIP] = &Desensitize{
		regexp:  regexp.MustCompile(`(\d{1,3}).\d{1,3}.\d{1,3}.\d{1,3}`),
		replace: `$1.****`,
	}
	builtinDesensitize[DesensitizeMethodBankCard] = &Desensitize{
		regexp:  regexp.MustCompile(`([1-9]{1})(\d{11}|\d{13}|\d{14})(\d{4})`),
		replace: `****$3`,
	}
	builtinDesensitize[DesensitizeMethodIDCard] = &Desensitize{
		regexp:  regexp.MustCompile(`([\d]{4})[\d]{11}([\d]{2}[\d|Xx])`),
		replace: `$1****`,
	}
	builtinDesensitize[DesensitizeMethodAccessKey] = &Desensitize{
		regexp:  regexp.MustCompile(`([a-zA-Z0-9]{4})(([a-zA-Z0-9]{26})|([a-zA-Z0-9]{12}))`),
		replace: `$1****`,
	}
}

func NewDesensitize(settings DesensitizeSettings) (*Desensitize, error) {
	switch settings.Method {
	case DesensitizeMethodMd5, DesensitizeMethodKeepFirstLast, DesensitizeMethodKeepLastFour:
		return &Desensitize{
			method: settings.Method,
		}, nil

	case DesensitizeMethodCustom:
		reg, err := regexp.Compile(settings.Regexp)
		if err != nil {
			return nil, err
		}
		return &Desensitize{
			regexp:  reg,
			replace: settings.Replace,
		}, nil

	default:
		// 内置的脱敏方法
		desensitize, found := builtinDesensitize[settings.Method]
		if !found {
			return nil, errors.New("built-in desensitize method not found: " + settings.Method)
		}
		return desensitize, nil
	}
}

func (d *Desensitize) ParseString(origin string) (string, error) {
	switch d.method {
	case DesensitizeMethodMd5:
		hash := md5.Sum([]byte(origin))
		return hex.EncodeToString(hash[:]), nil

	case DesensitizeMethodKeepFirstLast:
		length := len(origin)
		if length <= 2 {
			return origin, nil
		}
		return origin[:1] + "****" + origin[length-1:], nil

	case DesensitizeMethodKeepLastFour:
		length := len(origin)
		if length <= 4 {
			return origin, nil
		}
		return "****" + origin[length-4:], nil

	default:
		if d.regexp == nil {
			return origin, nil
		}
		return d.regexp.ReplaceAllString(origin, d.replace), nil
	}
}
