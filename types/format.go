package types

import (
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	LogExtractValueTypeLong   = "long"   // 整数
	LogExtractValueTypeFloat  = "float"  // 浮点数
	LogExtractValueTypeText   = "text"   // 文本
	LogExtractValueTypeDate   = "date"   // 日期
	LogExtractValueTypeObject = "object" // 对象
	LogExtractValueTypeArray  = "array"  // 数组
)

const (
	LogEventMessageKey = "__message__"
)

const (
	LogParserOriginText = "__origin__" // 系统保留字
)

const (
	LogExtractJsonRoot = "__root__" // json的根节点, 用于移动
)

const (
	ConstantEventTimeWeekly    = "weekly"    // 2006-01-w1  固定样式, 不允许更改
	ConstantEventTimeQuarterly = "quarterly" // 2006-q1 固定样式, 不允许更改
)

var (
	reEventTime = regexp.MustCompile(`%{\+@([^}]+)}`)
)

// 统一用Local时间
func FormatWithEventTime(text string, evevtTime time.Time, loc ...*time.Location) (result string) {
	loc = append(loc, time.Local)
	local := evevtTime.In(loc[0])
	result = text

	matches := reEventTime.FindAllStringSubmatch(result, -1)
	for _, submatches := range matches {
		var value string
		if submatches[1] == ConstantEventTimeWeekly {
			value = formatEventTimeByWeek(local)
		} else if submatches[1] == ConstantEventTimeQuarterly {
			value = formatEventTimeByQuarter(local)
		} else {
			value = local.Format(submatches[1])
		}
		result = strings.Replace(result, submatches[0], value, -1)
	}
	return
}

// index的分片规则统一替换为 .*, 做全匹配查询
func FormatToWildMatch(text string) (result string) {
	result = text
	matches := reEventTime.FindAllStringSubmatch(result, -1)
	for _, submatches := range matches {
		result = strings.Replace(result, submatches[0], "*", -1)
	}
	return
}

// 固定格式: 2006-01-w1
// 规则: monday是一周的第一天, sunday是一周的最后一天
// 当月跨周的情况下, 计入上一个月
// 实际效果是: 计算当前本月的第几个周一
func formatEventTimeByWeek(t time.Time) (result string) {
	prefix := t.Format("2006-01-w")
	day := t.Day()              // 比如 8.23日, 结果是 23
	weekday := int(t.Weekday()) // 星期二,  即 2
	if weekday == 0 {
		weekday = 7
	}
	numOfWeek := (day-weekday)/7 + 1
	result = prefix + strconv.Itoa(numOfWeek)
	return
}

// 固定格式: 2006-q1
func formatEventTimeByQuarter(t time.Time) (result string) {
	prefix := t.Format("2006-q")
	month := t.Month()
	numOfQuarter := int(t.Month()) / 3
	if month%3 != 0 {
		numOfQuarter += 1
	}
	result = prefix + strconv.Itoa(numOfQuarter)
	return
}
