package filter

import (
	"encoding/json"
	"errors"
	"strings"

	"github.com/flashcatcloud/fc-stash/utils"
)

const (
	GlobalFuncFormatToJson = "format_to_json"
)

const (
	GlobalFuncSplitJsonArray = "split_json_array"
)

// 封装成json结构
func FormatToJson(input string) string {
	// 本身是有效的json不额外处理
	if strings.Contains(input, "{") && json.Valid([]byte(input)) {
		return input
	}
	// 否则封装一个固定的key
	ret := make(map[string]string)
	ret["message"] = input
	return utils.MustToJsonString(ret)
}

// json数组拆分, json.Unmarshal()后的元素 只支持 string和map[string]interface{}
func SplitJsonArray(input string) ([]interface{}, error) {
	if !strings.Contains(input, "[") || !json.Valid([]byte(input)) {
		return nil, errors.New("invalid json array")
	}
	var array []interface{}
	if err := json.Unmarshal([]byte(input), &array); err != nil {
		return nil, err
	}
	var ret []interface{}
	for i := range array {
		switch _v := array[i].(type) {
		case map[string]interface{}:
			if len(_v) == 0 { // 过滤掉空结果
				continue
			}
			ret = append(ret, _v)

		case string:
			if len(_v) == 0 { // 过滤掉空字符串
				continue
			}
			ret = append(ret, _v)

		}
	}
	return ret, nil
}
