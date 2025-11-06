package pipeline_handler

import (
	"fmt"
	"testing"
	"time"

	"github.com/flashcatcloud/fc-stash/handler/logtask"
	"github.com/flashcatcloud/fc-stash/sink"
	"github.com/flashcatcloud/fc-stash/types"
	"github.com/xhit/go-str2duration/v2"
	"go.uber.org/zap"
)

func Test_Rollover(t *testing.T) {
	logger = zap.NewExample()
	AppendElasticRolloverTask(656,
		&sink.ElasticOutput{
			Servers:  []string{"http://10.99.1.105:9200"},
			Username: "elastic",
			Password: "MhxzKhl@FC",
		},
		&logtask.LogeventElasticSettings{
			IndexName:         "ds_retention_2_hours",
			IndexSuffixFormat: "default",
			RetentionDuration: "4h",
		},
	)

	time.Sleep(time.Minute * 20)
}

func Test_RolloverFunction(t *testing.T) {
	retention := "92d"
	if retention == logtask.DefaultElasticSettings {
		retention = "2d"
	}
	if len(retention) == 0 {
		return
	}
	suffixFormat := "daily"
	var suffixDuration time.Duration
	var lookbackMax int
	switch suffixFormat {
	case "hourly":
		suffixDuration = time.Hour
		lookbackMax = 24 * 7
	case "daily":
		suffixDuration = time.Hour * 24
		lookbackMax = 24
	case "weekly":
		suffixDuration = time.Hour * 24 * 7
		lookbackMax = 10
	case "monthly":
		suffixDuration = time.Hour * 24 * 30
		lookbackMax = 6 // 半年
	case "quarterly":
		suffixDuration = time.Hour * 24 * 90
		lookbackMax = 4 // 1年
	}

	if suffixDuration == 0 {
		return
	}

	indexname := "bx7_hshare_app"
	indexformat := logtask.FormatIndexFormat(indexname, suffixFormat)

	// 配置了分割逻辑和保留时间
	expired, err := str2duration.ParseDuration(retention)
	if err != nil {
		t.Fatal(err)
	}
	// 0 0s 代表永久保存, expired > 0 代表需要执行过期清理
	if expired > 0 {
		// 查找名称前缀是 {{自定义名称}}+"_",  "_" 代表存在分片规则
		indices := []string{
			"bx7_hshare_app_20250807",
			"bx7_hshare_app_20250806",
			"bx7_hshare_app_20250805",
			"bx7_hshare_app_20250804",
			"bx7_hshare_app_20250803",
			"bx7_hshare_app_20250802",
			"bx7_hshare_app_20250801",
			"bx7_hshare_app_20250701",
			"bx7_hshare_app_20250601",
			"bx7_hshare_app_20250622",
			"bx7_hshare_app_20250501",
			"bx7_hshare_app_20250521",
			"bx7_hshare_app_20250401",
		}
		if len(indices) == 0 {
			return
		}
		remainIndex := make(map[string]struct{})
		for i := range indices {
			remainIndex[indices[i]] = struct{}{}
		}
		todelete := make(map[string]struct{})
		// 计算N倍的时长, 避免有遗漏
		base := time.Now().Add(-1 * expired)
		for i := 1; i < lookbackMax; i++ {
			expiredTime := base.Add(suffixDuration * time.Duration(i) * -1)
			expiredIndex := types.FormatWithEventTime(indexformat, expiredTime)
			if _, found := remainIndex[expiredIndex]; found {
				todelete[expiredIndex] = struct{}{}
			}
		}
		fmt.Println(todelete)
	}
}
