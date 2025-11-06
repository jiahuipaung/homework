package filter

import (
	"errors"

	"github.com/flashcatcloud/fc-stash/types"
)

// 目前只支持对字符串的预处理
var (
	presetFilters = map[string]newPresetFilterFn{}
)

type newPresetFilterFn func() (types.FieldParser, error)

func registerFilter(method string, fn newPresetFilterFn) {
	if _, found := presetFilters[method]; found {
		return
	}
	presetFilters[method] = fn
}

func NewPresetFilter(method string) (types.FieldParser, error) {
	fn, found := presetFilters[method]
	if !found {
		return nil, errors.New(method + " not supported")
	}
	return fn()
}
