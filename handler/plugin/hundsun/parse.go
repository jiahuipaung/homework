package hundsun

import (
	"strings"

	"github.com/flashcatcloud/fc-stash/transform/text_parser"
)

const (
	FormatModeCheckTab = "tab_table"
	FormatModeMultiTab = "multi_table"
)

var (
	parser *text_parser.KeyValueParser
)

type MultilineData struct {
	Mode   string     `json:"mode,omitempty"`
	Keys   []string   `json:"keys,omitempty"`
	Values [][]string `json:"values,omitempty"`
}

func init() {
	_parser, _ := text_parser.NewKeyValueParser(
		`{
			"mode": "table",
			"table_row_delimiter": "\n",
			"table_key_delimiter": "|",
			"table_value_delimiter": "|",
			"table_value_multiline": true
		 }`,
		false,
	)
	if _parser != nil {
		parser = _parser.(*text_parser.KeyValueParser)
	}
}

// request_data/response_data格式化
func (log *RequestMergedLog) FormatData() {
	if parser == nil {
		return
	}
	if len(log.RequestData) > 0 {
		if strings.Contains(log.RequestData, "check_tab_data") {
			log.RequestFormat.Mode = FormatModeCheckTab
		} else {
			valueIndex, ordered := parser.ParseModeTable(log.RequestData)
			log.RequestFormat.Mode = FormatModeMultiTab
			log.RequestFormat.Keys, log.RequestFormat.Values = formatMultilineData(valueIndex, ordered)
		}
	}
	if len(log.ResponseData) > 0 {
		if strings.Contains(log.ResponseData, "check_tab_data") {
			log.ResponseFormat.Mode = FormatModeCheckTab
		} else {
			valueIndex, ordered := parser.ParseModeTable(log.ResponseData)
			log.ResponseFormat.Mode = FormatModeMultiTab
			log.ResponseFormat.Keys, log.ResponseFormat.Values = formatMultilineData(valueIndex, ordered)
		}
	}
}

func formatMultilineData(valueIndex map[string][]string, ordered []string,
) (keys []string, values [][]string) {
	if len(valueIndex) == 0 || len(ordered) == 0 {
		return
	}
	numOfValue := 0
	for _, index := range valueIndex {
		if len(index) == 0 {
			return
		}
		if numOfValue == 0 {
			numOfValue = len(index)
		}
		if numOfValue != len(index) {
			return
		}
	}
	for i := 0; i < numOfValue; i++ {
		values = make([][]string, numOfValue)
	}
	for _, key := range ordered {
		if len(key) == 0 {
			continue
		}
		index, found := valueIndex[key]
		if !found || len(index) == 0 {
			continue
		}
		keys = append(keys, key)
		for i := range index {
			values[i] = append(values[i], index[i])
		}
	}
	return
}
