package tasks

import (
	"fmt"
	"testing"

	"github.com/flashcatcloud/fc-stash/config"
	"github.com/flashcatcloud/go-pkg/srv"
	"github.com/flashcatcloud/go-pkg/utils"
	"github.com/flashcatcloud/go-pkg/x/log"
)

func Test_N9eAPI(t *testing.T) {
	config.MustLoad("../../../etc/", "./logs/")

	ctx := srv.NewCtx(log.NewTestLogger(1111))
	task, err := GetLogeventTaskListFromN9eService(ctx)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(utils.ShouldToJsonString(task.DataSources))
	fmt.Println(utils.ShouldToJsonString(task.Tasks))
	fmt.Println(utils.ShouldToJsonString(task.LabelMappings))
}

func Test_InsightAPI(t *testing.T) {
	config.MustLoad("../../../etc/", "./logs/")

	ctx := srv.NewCtx(log.NewTestLogger(1111))
	ds, task, err := GetLogeventTaskListFromInsightService(ctx)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(utils.ShouldToJsonString(ds))
	fmt.Println(utils.ShouldToJsonString(task))
}
