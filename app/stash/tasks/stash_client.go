package tasks

import (
	"errors"
	"math/rand"
	"net/http"
	"time"

	"github.com/flashcatcloud/fc-stash/config"
	"github.com/flashcatcloud/fc-stash/handler/logtask"
	"github.com/flashcatcloud/go-pkg/srv"
	"github.com/flashcatcloud/go-pkg/x/gorequest"
	"go.uber.org/zap"
)

// stash从两个途径获取任务
// 1. fc-insight的数据库: 数据源通过fc-insight/logic/data_source的函数额外获取
// 2. n9e-plus的接口: 数据源在同一个接口中
// 两份任务都拿到以后, 直接合并, 但是task.ID 在两边是可能冲突的, 因此做一个掩码处理
type LogeventTaskList struct {
	DataSources   map[int64]*logtask.DatasourceSettings  `json:"data_sources"`
	Tasks         []*logtask.LogeventTask                `json:"tasks"`
	LabelMappings map[int64]map[string]map[string]string `json:"label_mappings"`
}

// mask=10000
func GetLogeventTaskListFromInsightService(ctx *srv.Context,
) (dsmap map[int64]*logtask.DatasourceSettings, tasks []*logtask.LogeventTask, err error) {
	var (
		url = "/api/v2/dimensions/logevent/tasks"
	)
	// 未配置远端地址, 不执行
	if len(config.C.Stash.InsightService.Webapis) == 0 {
		return
	}
	servers := randomServers(config.C.Stash.InsightService.Webapis)
	for i := range servers {
		result := struct {
			Data struct {
				DataSources map[int64]*logtask.DatasourceSettings `json:"data_sources"`
				Tasks       []*logtask.LogeventTask               `json:"tasks"`
			} `json:"data"`
		}{}

		resp, _, errs := gorequest.New(ctx).Timeout(time.Second * 5).
			Get(servers[i] + url).
			EndStruct(&result)

		if len(errs) > 0 {
			ctx.Logger().Warn("get logevent task list from insight service warning",
				zap.String("insight_server", servers[i]), zap.Error(errs[0]))
			continue
		}
		if resp.StatusCode != http.StatusOK {
			ctx.Logger().Warn("get logevent task list from insight service warning",
				zap.String("insight_server", servers[i]), zap.String("http_code", resp.Status))
			continue
		}
		return result.Data.DataSources, result.Data.Tasks, nil
	}
	return nil, nil, errors.New("all insight service api failed")
}

// mask=0
// 根据配置文件中是否有远端地址判断
func GetLogeventTaskListFromN9eService(ctx *srv.Context,
) (task *LogeventTaskList, err error) {
	// 未配置远端地址, 不执行
	if len(config.C.Stash.N9eService.Webapis) == 0 {
		return
	}
	var (
		url = "/v1/n9e-plus/logevent/tasks"
	)
	servers := randomServers(config.C.Stash.N9eService.Webapis)
	for i := range servers {
		result := struct {
			Data LogeventTaskList `json:"data"`
		}{}

		resp, _, errs := gorequest.New(ctx).Timeout(time.Second*5).
			Get(servers[i]+url).
			SetBasicAuth(config.C.Stash.N9eService.Username, config.C.Stash.N9eService.Password).
			EndStruct(&result)

		if len(errs) > 0 {
			ctx.Logger().Warn("get logevent task list from n9e-plus warning",
				zap.String("n9e_server", servers[i]), zap.Error(errs[0]))
			continue
		}
		if resp.StatusCode != http.StatusOK {
			ctx.Logger().Warn("get logevent task list from n9e-plus warning",
				zap.String("n9e_server", servers[i]), zap.String("http_code", resp.Status))
			continue
		}
		return &result.Data, nil
	}
	return nil, errors.New("all n9e api failed")
}

func randomServers(old []string) []string {
	if len(old) == 1 || len(old) == 0 {
		return old
	}
	first := rand.Intn(len(old))
	newest := make([]string, len(old))
	for i := 0; i < len(old); i++ {
		newest[i] = old[(i+first)%len(old)]
	}
	return newest
}
