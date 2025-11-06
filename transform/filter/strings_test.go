package filter

import (
	"fmt"
	"testing"
)

func Test_StringsFunction(t *testing.T) {
	var filters []*StringPreFunction
	filters = append(filters, &StringPreFunction{
		// Mode:   GlobalFuncStringNotMatch,
		Mode:   GlobalFuncStringContains,
		Format: "from_channel",
	})
	filters = append(filters, &StringPreFunction{
		Mode:   GlobalFuncStringNotMatch,
		Format: "200",
	})
	filters = append(filters, &StringPreFunction{
		Mode: GlobalFuncFormatToJson,
	})

	for i := range filters {
		if err := filters[i].Compile(); err != nil {
			t.Fatal(err)
		}
	}
	msg, ok := StringPreExtractHandle("from_channel,20", filters, false)
	fmt.Println(ok)
	fmt.Println(msg)
}
