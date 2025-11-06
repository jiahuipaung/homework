package filter

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"reflect"
	"testing"

	"github.com/flashcatcloud/fc-stash/utils"
)

func Test_StringArraySplit(t *testing.T) {
	message := make(map[string]interface{})
	if err := json.Unmarshal([]byte(origin), &message); err != nil {
		t.Fatal(err)
	}
	line := utils.MustToJsonString(message)
	var lines []string
	for i := 0; i < rand.Intn(2)+1; i++ {
		lines = append(lines, line)
	}
	origin := utils.MustToJsonString(lines)

	ret, err := SplitJsonArray(origin)
	if err != nil {
		t.Fatal(err)
	}
	for i := range ret {
		fmt.Println(reflect.TypeOf(ret[i]).Kind().String(), ret[i])
	}
}

func Test_MapArraySplit(t *testing.T) {
	message := make(map[string]interface{})
	if err := json.Unmarshal([]byte(origin), &message); err != nil {
		t.Fatal(err)
	}
	var lines []interface{}
	for i := 0; i < rand.Intn(2)+1; i++ {
		lines = append(lines, message)
	}
	origin := utils.MustToJsonString(lines)

	ret, err := SplitJsonArray(origin)
	if err != nil {
		t.Fatal(err)
	}
	for i := range ret {
		fmt.Println(reflect.TypeOf(ret[i]).Kind().String(), ret[i])
	}
}
