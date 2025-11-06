package field_format

import (
	"errors"

	"github.com/flashcatcloud/fc-stash/types"
)

// 目前只支持对字符串的预处理
var (
	presetFieldParsers = map[string]newPresetFieldParserFn{}
)

type newPresetFieldParserFn func() (types.FieldParser, error)

func registerFieldParser(method string, fn newPresetFieldParserFn) {
	if _, found := presetFieldParsers[method]; found {
		return
	}
	presetFieldParsers[method] = fn
}

func NewPresetFieldParser(method string) (types.FieldParser, error) {
	fn, found := presetFieldParsers[method]
	if !found {
		return nil, errors.New(method + " not supported")
	}
	return fn()
}
