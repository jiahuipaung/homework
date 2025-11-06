package doris

import (
	"context"
	"testing"

	"github.com/flashcatcloud/fc-stash/types"
)

var testCli *Client

func init() {
	output := &DorisOutput{
		Host:     "10.99.1.214:9030",
		FeHost:   "10.99.1.214:8030",
		Database: "log_analysis",
		Table:    "fc_insight",
		Username: "root",
		Password: "",
	}
	cli, err := output.NewClient(context.Background())
	if err != nil {
		panic(err)
	}
	testCli = cli
}

func TestClient_LoadData(t *testing.T) {
	testCli.WithLoader(&SqlLoader{client: testCli})
	if err := testCli.loader.Write([]types.ExtractedLog{
		map[string]interface{}{
			"k": "1111111",
			"v": map[string]interface{}{
				"before":        "abb58cc0db673a0bd5190000d2ff9c53bb51d04d",
				"distinct_size": 4,
				"head":          "91edd3c8c98c214155191feb852831ec535580ba",
				"push_id":       int64(6027092734),
				"ref":           "refs/heads/master",
				"size":          4,
			},
		},
		map[string]interface{}{
			"k": "222",
			"v": map[string]interface{}{
				"before":        "abb58cc0db673a0bd5190000d2ff9c53bb51d04d",
				"distinct_size": 4,
				"head":          "91edd3c8c98c214155191feb852831ec535580ba",
				"push_id":       int64(6027092734),
				"ref":           "refs/heads/master",
				"size":          4,
			},
		},
	}); err != nil {
		t.Fatal(err)
	}
}
