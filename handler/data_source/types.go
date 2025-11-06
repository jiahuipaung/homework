package data_source

import "errors"

const (
	DataSourceTypeElasticsearch = "elasticsearch"
	DataSourceTypeDoris         = "doris"
	DataSourceTypeKafka         = "kafka"
	DataSourceTypeVictoriaLogs  = "victoria_logs"
)

func CacheGetDataSourcByID(dsID int64) (interface{}, error) {
	return nil, errors.New("not found")
}
