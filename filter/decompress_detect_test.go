package filter

import (
	"bytes"
	"compress/gzip"
	"compress/zlib"
	"fmt"
	"log"
	"testing"
)

type testDecompressDetect struct {
	logs       string
	want       bool
	wantString string

	result       bool
	resultString string
}

func Test_Decompress_Detect(t *testing.T) {
	for i, data := range decompressTests {
		testLog := ""
		//fmt.Println(data.logs)
		var err error
		if data.wantString == "gzip" {
			testLog, err = compressStringGzip(data.logs)
			if err != nil {
				log.Fatal("Error compressing with gzip: ", err)
			}
		} else if data.wantString == "zlib" {
			testLog, err = compressStringZlib(data.logs)
			if err != nil {
				log.Fatal("Error compressing with zlib: ", err)
			}
		} else if data.wantString == "" {
			testLog = data.logs
		}
		//fmt.Println(i, string(testLog))
		data.resultString, data.result = checkLogCompress([]byte(testLog))
		fmt.Println(i, data.want, data.result, data.wantString, data.resultString)
	}
}

// 压缩字符串使用 gzip
func compressStringGzip(s string) (string, error) {
	var buf bytes.Buffer

	gzipWriter := gzip.NewWriter(&buf)

	_, err := gzipWriter.Write([]byte(s))
	if err != nil {
		return "", err
	}

	err = gzipWriter.Close()
	if err != nil {
		return "", err
	}
	compData := buf.Bytes()
	return string(compData), nil
}

func compressStringZlib(s string) (string, error) {
	var buf bytes.Buffer

	zlibWriter, _ := zlib.NewWriterLevel(&buf, zlib.DefaultCompression)

	_, err := zlibWriter.Write([]byte(s))
	if err != nil {
		return "", err
	}

	err = zlibWriter.Close()
	if err != nil {
		return "", err
	}
	compData := buf.Bytes()
	return string(compData), nil
}

var decompressTests = []testDecompressDetect{
	{
		logs:       "\"@metadata\": {\n\t  \"beat\": \"filebeat\",\n\t  \"type\": \"doc\",\n\t  \"version\": \"6.2.4\",\n\t  \"topic\": \"ELK_NGINX\"\n\t}",
		want:       true,
		wantString: "gzip",
	},
	{
		logs:       "\"@metadata\": {\n\t  \"beat\": \"filebeat\",\n\t  \"type\": \"doc\",\n\t  \"version\": \"6.2.4\",\n\t  \"topic\": \"ELK_NGINX\"\n\t}",
		want:       true,
		wantString: "zlib",
	},
	{
		logs:       "\"@metadata\": {\n\t  \"beat\": \"filebeat\",\n\t  \"type\": \"doc\",\n\t  \"version\": \"6.2.4\",\n\t  \"topic\": \"ELK_NGINX\"\n\t}",
		want:       false,
		wantString: "",
	},
	{
		logs:       "fortest",
		want:       true,
		wantString: "gzip",
	},
	{
		logs:       "fortest",
		want:       true,
		wantString: "zlib",
	},
	{
		logs:       "fortest",
		want:       false,
		wantString: "",
	},
}
