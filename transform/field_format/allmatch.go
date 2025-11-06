package field_format

import (
	"strings"

	"github.com/flashcatcloud/fc-stash/types"
)

// 全部提取

const (
	PresetAllmatch = "allmatch"
)

type AllmatchExtract struct {
}

func init() {
	registerFieldParser(PresetAllmatch, NewAllmatchExtract)
}

func NewAllmatchExtract() (types.FieldParser, error) {
	return &AllmatchExtract{}, nil
}

func (all *AllmatchExtract) ParseString(origin string) (string, error) {
	return strings.TrimSpace(origin), nil
}
