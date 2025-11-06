package doris

import "github.com/flashcatcloud/fc-stash/types"

type Loader interface {
	Write(batch []types.ExtractedLog) error
}
