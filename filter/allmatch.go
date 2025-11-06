package filter

import (
	"strings"

	"github.com/flashcatcloud/fc-stash/types"
)

// 预定义一组通用的提取规则

// allmatch: 全匹配, 简化用户的使用成本
// 默认去掉前后的空格
var (
	PresetAllmatch = "allmatch"
)

var (
	PresetDesensitize = "desensitize"
)

type AllmatchExtract struct {
}

func init() {
	registerFilter(PresetAllmatch, NewAllmatchExtract)
	registerFilter(PresetDesensitize, NewDesensitizeExtract)
}

func NewAllmatchExtract() (types.FieldParser, error) {
	return &AllmatchExtract{}, nil
}

func (all *AllmatchExtract) ParseString(origin string) (string, error) {
	return strings.TrimSpace(origin), nil
}

type DesensitizeExtract struct{}

func NewDesensitizeExtract() (types.FieldParser, error) {
	return &DesensitizeExtract{}, nil
}

func (all *DesensitizeExtract) ParseString(origin string) (string, error) {
	return "********", nil
}
