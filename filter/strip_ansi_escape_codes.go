package filter

import (
	"regexp"
	"strings"
)

var (
	reAnsi = regexp.MustCompile(
		`(\\u001b)(8|7|H|>|\[(\?\d+(h|l)|[0-2]?(K|J)|\d*(A|B|C|D\D|E|F|G|g|i|m|n|S|s|T|u)|1000D\d+|\d*;\d*(f|H|r|m)|\d+;\d+;\d+m))`)
)

var (
	GlobalFuncStripAnsi = "strip_ansi"
)

func RemoveAnsiEscape(input string) (result string) {
	result = input
	matches := reAnsi.FindAllStringSubmatch(input, -1)
	for _, submatches := range matches {
		result = strings.Replace(result, submatches[0], "", -1)
	}
	return
}
