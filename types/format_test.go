package types

import (
	"testing"
	"time"
)

func Test_Format(t *testing.T) {
	unix := time.Unix(1661307272, 0)

	input := map[string]string{
		"ds_d9efa_%{+@2006010215}": "ds_d9efa_2022082410",
		"ds_d9efa_%{+@20060102}":   "ds_d9efa_20220824",
		"ds_d9efa_%{+@200601}":     "ds_d9efa_202208",
		"ds_d9efa_%{+@weekly}":     "ds_d9efa_2022-08-w4",
		"ds_d9efa_%{+@quarterly}":  "ds_d9efa_2022-q3",
	}
	for k, v := range input {
		formatted := FormatWithEventTime(k, unix)
		if formatted != v {
			t.Fatal("invalid output:", formatted, "input:", k, "expected:", v)
		}
	}

}
