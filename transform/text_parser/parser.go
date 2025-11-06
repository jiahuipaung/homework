package text_parser

import (
	"errors"

	"github.com/flashcatcloud/fc-stash/types"
)

var (
	presetParsers = map[string]newPresetParserFn{}
)

type newPresetParserFn func(format string, needOrigin bool) (types.TextParser, error)

func registerParser(method string, fn newPresetParserFn) {
	if _, found := presetParsers[method]; found {
		return
	}
	presetParsers[method] = fn
}

func NewPresetTextParser(method string, format string, needOrigin bool) (types.TextParser, error) {
	fn, found := presetParsers[method]
	if !found {
		return nil, errors.New(method + " not supported")
	}
	return fn(format, needOrigin)
}
