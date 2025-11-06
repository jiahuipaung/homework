package map_iterator

import (
	"fmt"
	"reflect"
	"strconv"
	"time"
)

// map[s]

// 很多类型没有处理, 实际上经过json.Unmarshal()一般不会出现
func GetString(origin map[string]interface{}, target string) (string, bool) {
	value, found := FindByField(origin, target)
	if !found || value == nil {
		return "", false
	}
	switch reflect.TypeOf(value).Kind() {
	case reflect.Array, reflect.Map, reflect.Struct, reflect.Pointer:
		return "", false

	case reflect.Float64:
		return strconv.FormatFloat(value.(float64), 'f', -1, 64), true

	case reflect.Int64:
		return strconv.FormatInt(value.(int64), 10), true

	case reflect.Bool:
		return strconv.FormatBool(value.(bool)), true
	}

	// default
	return fmt.Sprintf("%v", value), true
}

func GetFloat64(origin map[string]interface{}, target string) (float64, bool) {
	value, found := FindByField(origin, target)
	if !found || value == nil {
		return 0.0, false
	}

	switch reflect.TypeOf(value).Kind() {
	case reflect.Array, reflect.Map, reflect.Struct, reflect.Pointer:
		return 0.0, false

	case reflect.String:
		fvalue, err := strconv.ParseFloat(value.(string), 64)
		if err == nil {
			return fvalue, true
		}

	case reflect.Float64:
		return value.(float64), true

	case reflect.Int64:
		return float64(value.(int64)), true
	}

	// default
	return 0.0, false
}

func GetInt64(origin map[string]interface{}, target string) (int64, bool) {
	value, found := FindByField(origin, target)
	if !found || value == nil {
		return 0, false
	}

	switch reflect.TypeOf(value).Kind() {
	case reflect.Array, reflect.Map, reflect.Struct, reflect.Pointer:
		return 0, false

	case reflect.String:
		fvalue, err := strconv.ParseInt(value.(string), 10, 64)
		if err == nil {
			return fvalue, true
		}

	case reflect.Float64:
		return int64(value.(float64)), true

	case reflect.Int64:
		return value.(int64), true
	}

	// default
	return 0, false
}

func GetTimestamp(origin map[string]interface{}, target string, formats ...string) (int64, bool) {
	value, found := FindByField(origin, target)
	if !found || value == nil {
		return 0, false
	}
	// time.Time类型, 由stash阶段解析得到的
	if vdate, ok := value.(time.Time); ok {
		return vdate.Unix(), true
	}
	vts, found := GetString(origin, target)
	if !found {
		return 0, false
	}
	ts := AutoDetectTimestamp(vts, formats...)
	return ts, ts > 0
}
