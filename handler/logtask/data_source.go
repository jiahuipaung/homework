package logtask

import (
	"github.com/flashcatcloud/fc-stash/config"
	"github.com/flashcatcloud/fc-stash/handler/data_source"
	"github.com/flashcatcloud/fc-stash/utils"
	"github.com/toolkits/pkg/slice"
)

// 数据源配置
type DatasourceSettings struct {
	Type     string      `json:"type"`
	Settings interface{} `json:"settings"`
}

func DecodeDatasourceIndex(input map[int64]*DatasourceSettings) map[int64]interface{} {
	output := make(map[int64]interface{})
	for dsID, item := range input {
		switch item.Type {
		case data_source.DataSourceTypeElasticsearch:
			pluges, err := data_source.NewPlugESWithSettings(item.Settings)
			if err != nil {
				continue
			}
			if !shouldHandle(pluges.ClusterName) {
				utils.Logger.Sugar().Infof("target cluster [%s] of datasource_id [%d](datasource type:es) not Belongs to %v or [default], skip...", pluges.ClusterName, dsID, config.C.GetRelatedClusters())
				continue
			}

			output[dsID] = pluges
		case data_source.DataSourceTypeDoris:
			plugDoris, err := data_source.NewPlugDorisWithSettings(item.Settings)
			if err != nil {
				continue
			}
			if !shouldHandle(plugDoris.ClusterName) {
				utils.Logger.Sugar().Infof("target cluster [%s] of datasource_id [%d](datasource type:doris) not Belongs to %v or [default], skip...", plugDoris.ClusterName, dsID, config.C.GetRelatedClusters())
				continue
			}

			output[dsID] = plugDoris

		case data_source.DataSourceTypeKafka:
			plugKafka, err := data_source.NewPlugKafkaWithSettings(item.Settings)
			if err != nil {
				continue
			}
			output[dsID] = plugKafka

		case data_source.DataSourceTypeVictoriaLogs:
			plugVictoriaLogs, err := data_source.NewPlugVMLogsWithSettings(item.Settings)
			if err != nil {
				continue
			}
			if !shouldHandle(plugVictoriaLogs.ClusterName) {
				utils.Logger.Sugar().Infof("target cluster [%s] of datasource_id [%d](datasource type:VictoriaLogs) not Belongs to %v or [default], skip...", plugVictoriaLogs.ClusterName, dsID, config.C.GetRelatedClusters())
			}
			output[dsID] = plugVictoriaLogs
		}
	}
	return output
}

func shouldHandle(clusterName string) bool {
	// 如果指定了target_clusters, 处理指定的
	relatedClusters := config.C.GetRelatedClusters()
	// 配置文件未指定，处理所有规则
	if len(relatedClusters) == 0 {
		return true
	}

	if !slice.ContainsString(relatedClusters, clusterName) {
		return false
	}
	return true
}
