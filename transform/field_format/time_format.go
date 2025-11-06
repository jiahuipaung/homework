package field_format

import (
	"strings"
)

// 旧版本的日志格式转换保存的是golang的格式, 但是前端暴露给用户的是linux的格式, 所以需要进行转换
// 新版本统一改为linux的格式, transformer执行前自己做转换

var (
	// 2006-01-02T15:04:05Z07:00
	linuxGolangDateFormatMapping = [][2]string{
		{"%s%6N", DefaultDateFormatUnixMicro},
		{"%s%3N", DefaultDateFormatUnixMilli},
		{"%s", DefaultDateFormatUnix},
		{"%Y", "2006"},
		{"%m", "01"},
		{"%b", "Jan"},     // Jan
		{"%B", "January"}, // January
		{"%d", "02"},
		{"%H", "15"},
		{"%M", "04"},
		{"%S", "05"},
		{"%z", "0700"},
		{"%:z", "07:00"},
		{"%N", "999999999"},
		{"%3N", "999"}, // mapping, 不是contain判断, 不会与999999999发生冲突
	}
)

func ConvertLinuxDateFormatToGolang(origin string) string {
	for _, mapping := range linuxGolangDateFormatMapping {
		origin = strings.Replace(origin, mapping[0], mapping[1], 1)
	}
	return origin
}

func ConvertGolangDateFormatToLinux(origin string) string {
	for _, mapping := range linuxGolangDateFormatMapping {
		origin = strings.Replace(origin, mapping[1], mapping[0], 1)
	}
	return origin
}
