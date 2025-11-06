package utils

import (
	"bytes"
	"encoding/json"
	"reflect"
	"strings"
	"unsafe"
)

func MustToJsonString(src interface{}) string {
	if IsNil(src) {
		return "{}"
	}
	if data, err := ToJsonString(src); err != nil {
		panic(err)
	} else {
		return data
	}
}

func ToJsonString(src interface{}) (string, error) {
	if IsNil(src) {
		return "{}", nil
	}
	data, err := JsonEncode(src)
	return strings.TrimRight(BytesToString(data), "\n"), err
}

func IsNil(src interface{}) bool {
	if src == nil {
		return true
	}
	switch reflect.TypeOf(src).Kind() {
	case reflect.Ptr, reflect.Map, reflect.Interface:
		if reflect.ValueOf(src).IsNil() {
			return true
		}
	case reflect.Slice, reflect.Array:
		if reflect.ValueOf(src).IsNil() {
			return true
		}
	}
	return false
}

func JsonEncode(obj interface{}) ([]byte, error) {
	buf := bytes.NewBuffer(nil)
	encoder := json.NewEncoder(buf)
	err := encoder.Encode(obj)
	return buf.Bytes(), err
}

func BytesToString(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}

func StringArrayContains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

func JsonDecodeString(data string, obj interface{}) error {
	buf := bytes.NewBufferString(data)
	decoder := json.NewDecoder(buf)
	decoder.UseNumber()
	return decoder.Decode(obj)
}

// /xxx/ 至少有3个字符, 以'/'开始和结束
func TrimRegexpPattern(old string) (bool, string) {
	if len(old) > 2 &&
		strings.HasPrefix(old, "/") &&
		strings.HasSuffix(old, "/") {
		return true, strings.TrimSuffix(strings.TrimPrefix(old, "/"), "/")
	}
	return false, old
}
