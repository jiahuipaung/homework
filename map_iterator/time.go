package map_iterator

import (
	"strconv"
	"time"
)

var logDateFormatDetect = [][2]string{
	{"2006-01-02T15:04:05.999-07:00"},  // 默认的时间格式
	{"2006-01-02 15:04:05", "Local"},   // 格式一
	{"2006-01-02T15:04:05", "Local"},   // 格式二
	{"02/Jan/2006:15:04:05", "Local"},  // 格式三
	{"02/Jan/2006 15:04:05", "Local"},  // 格式三.一
	{"02-Jan-2006:15:04:05", "Local"},  // 格式四
	{"02-Jan-2006 15:04:05", "Local"},  // 格式四.一
	{"2006/01/02 15:04:05", "Local"},   // 格式五.一
	{"2006/01/02 - 15:04:05", "Local"}, // 格式五.一
}

// 自动推导
func AutoDetectTimestamp(origin string, formats ...string) int64 {
	// 配置了时间格式和时区
	if len(formats) > 0 && len(formats[0]) > 0 {
		format := formats[0]
		var loc *time.Location = time.Local
		if len(formats) > 1 && len(formats[1]) > 0 {
			_loc, _ := time.LoadLocation(formats[1])
			if _loc != nil {
				loc = _loc
			}
		}
		t0, err := time.ParseInLocation(format, origin, loc)
		if err != nil {
			return 0
		}
		return t0.Unix()
	}

	// 没有配置, 自动推导
	// 可能是Unix秒时间戳 或 Unix毫秒时间戳
	if len(origin) == 10 || len(origin) == 13 {
		if v, err := strconv.ParseInt(origin, 10, 64); err == nil {
			if len(origin) == 10 && v > 1e9 {
				return v
			}
			if len(origin) == 13 && v > 1e12 {
				return v / 1000
			}
		}
	}

	for _, format := range logDateFormatDetect {
		if len(format[1]) == 0 {
			if t0, err := time.Parse(format[0], origin); err == nil {
				return t0.Unix()
			}
		} else {
			var loc *time.Location
			if format[1] == "Local" {
				loc = time.Local
			} else if format[1] == "UTC" {
				loc = time.UTC
			}
			if loc != nil {
				if t0, err := time.ParseInLocation(format[0], origin, loc); err == nil {
					return t0.Unix()
				}
			}
		}
	}

	return 0
}
