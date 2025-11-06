package filter

import (
	"bytes"
	"sync"
)

// Deprecated, 不再依赖, 后续删除

var (
	bytesPool = sync.Pool{
		New: func() interface{} { return new(bytes.Buffer) },
	}
)
