package utils

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"sync"
)

var (
	bytesPool = sync.Pool{
		New: func() interface{} { return new(bytes.Buffer) },
	}
)

func EncodeUUIDWithSortedKeys(keys []string) string {
	if len(keys) == 0 {
		return ""
	}
	encoder := bytesPool.Get().(*bytes.Buffer)
	encoder.Reset()
	defer bytesPool.Put(encoder)

	for _, k := range keys {
		encoder.WriteString(k)
		encoder.WriteByte('/')
	}
	hash := md5.Sum(encoder.Bytes())
	return hex.EncodeToString(hash[:])
}
