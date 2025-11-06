package http_handler

import (
	"fmt"
	"testing"

	"github.com/flashcatcloud/fc-stash/transform/text_parser"
	"github.com/flashcatcloud/fc-stash/utils"
)

func Test_KvPair(t *testing.T) {
	item := GetLogEventSystemKeyPairs("zh_CN")
	fmt.Println(utils.MustToJsonString(item))
}
func Test_Dict(t *testing.T) {
	lang := "zh_CN"
	item := FormatKeyvaluePairByLang(lang, "logx_"+text_parser.PresetOTELTraceSpans)
	fmt.Println(item)
}
