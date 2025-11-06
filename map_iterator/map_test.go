package map_iterator

import (
	"encoding/json"
	"fmt"
	"testing"
)

func Test_FindByPath(t *testing.T) {
	input := `
	{
		"message": {
			"status": 200,
			"domain": "flashcat.cloud",
			"request_time": 1.1
		},
		"timestamp": "1234567890",
		"agent_host": "localhost",
		"level": "info"
	}
	`
	origin := make(map[string]interface{})
	if err := json.Unmarshal([]byte(input), &origin); err != nil {
		t.Fatal(err)
	}
	fmt.Println(FindByField(origin, "timestamp"))
	fmt.Println(FindByField(origin, "message"))
	fmt.Println(FindByField(origin, "message.status"))
	fmt.Println(FindByField(origin, "not exist"))
}

func Test_SetByPath(t *testing.T) {
	origin := make(map[string]interface{})
	err := SetByPath(origin, []string{"message", "status"}, 200)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(origin)
	err = SetByPath(origin, []string{"message", "data", "status"}, 200)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(origin)
}
