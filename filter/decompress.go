package filter

import (
	"bytes"
	"compress/gzip"
	"compress/zlib"
	"errors"
	"io/ioutil"
)

var (
	GlobalFuncDecompress = "decompress"
)

func Decompress(input string, alg string) (result string, err error) {
	if alg == "gzip" {
		return gzipDecompress(input)
	}
	if alg == "zlib" {
		return zlibDecompress(input)
	}
	return "", errors.New("not suppored")
}

func gzipDecompress(input string) (string, error) {
	reader, err := gzip.NewReader(bytes.NewBufferString(input))
	if err != nil {
		return "", err
	}
	defer reader.Close()
	out, err := ioutil.ReadAll(reader)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func zlibDecompress(input string) (string, error) {
	reader, err := zlib.NewReader(bytes.NewBufferString(input))
	if err != nil {
		return "", err
	}
	defer reader.Close()
	out, err := ioutil.ReadAll(reader)
	if err != nil {
		return "", err
	}
	return string(out), nil
}
