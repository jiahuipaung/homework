package doris

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/flashcatcloud/fc-stash/types"
	"github.com/flashcatcloud/go-pkg/x/prom_exporter"
	"go.uber.org/zap"
)

const dorisDatetimeFormat = "2006-01-02 15:04:05.999999"

var (
	pool = new(sync.Map)
)

func init() {
	prom_exporter.SetHistogramBuckets("log_event_output_doris_bulk_milliseconds",
		[]float64{10, 100, 1000, 2000, 5000, 10000, 15000, 25000},
	)
}

type DorisOutput struct {
	// go中没有jdbc提供的负载均衡能力，无法直接写多个fe地址
	// 因此这里只写一个实例地址，如果需要负载均衡， doris维护者可以自行使用nginx或者HAProxy等搭建proxy
	// 参考 https://doris.apache.org/zh-CN/docs/admin-manual/cluster-management/load-balancing/
	Host     string `json:"host"`    // be node
	FeHost   string `json:"fe_host"` // fe node
	Database string `json:"database"`
	Table    string `json:"table"`
	Username string `json:"username"`
	Password string `json:"password"`

	// 表结构相关配置
	SortKey           []string      `json:"sort_keys"`       // 排序键
	PartitionField    string        `json:"partition_field"` // 分区字段(通常是时间戳字段)
	BulkActions       int           `json:"bulk_actions"`
	BulkFlushInterval time.Duration `json:"bulk_flush_interval"`

	client    *Client
	processor *BatchProcessor
	logger    *zap.Logger
	closed    bool
}

var _ types.Output = &DorisOutput{}

func (output *DorisOutput) Name() string {
	return "doris_" + output.Database + output.Table
}

func (output *DorisOutput) Init(ctx context.Context, worker int) error {
	if len(output.Host) == 0 {
		return errors.New("doris hosts empty")
	}
	if len(output.Database) == 0 {
		return errors.New("doris database not set")
	}
	if len(output.Table) == 0 {
		return errors.New("doris table not set")
	}

	// 设置默认值
	if output.BulkActions == 0 {
		// 默认缓冲大小为1000, 即内存中可以存放1000条日志
		output.BulkActions = 1000
	}
	if output.BulkFlushInterval == 0 {
		// 默认5s刷新一次
		output.BulkFlushInterval = time.Second * 5
	}

	// 初始化客户端
	client, err := output.NewClient(ctx)
	if err != nil {
		return err
	}
	client.WithLoader(&StreamLoader{client: client})
	output.client = client
	return nil
}

func (output *DorisOutput) Start(ctx context.Context, logger *zap.Logger) error {
	if output.client == nil {
		return errors.New("nil pointer of doris output client")
	}

	output.logger = logger
	output.processor = NewBatchProcessor(
		output.BulkActions,
		output.BulkFlushInterval,
		output.flush,
	)
	return nil
}

func (output *DorisOutput) Stop(ctx context.Context) {
	output.closed = true
	if output.processor != nil {
		output.processor.Close()
	}
	if output.client != nil {
		pool.Delete(output.client)
		output.client.Close()
	}
}

func (output *DorisOutput) Sink(ctx context.Context, ext types.ExtractedLog) error {
	if len(ext) == 0 {
		return errors.New("extracted nil pointer")
	}
	if output.processor == nil {
		return errors.New("doris output handler not started")
	}
	if output.closed {
		return errors.New("doris output handler was closed")
	}

	prom_exporter.Inc("log_event_output", map[string]string{
		"target": "doris",
		"table":  output.Table,
	})
	for k, v := range ext {
		if t, ok := v.(time.Time); ok {
			ext[k] = t.Format(dorisDatetimeFormat)
		}
	}

	return output.processor.Add(ext)
}

func (output *DorisOutput) IsDiff(other types.Output) bool {
	return false
}

func (output *DorisOutput) flush(batch []types.ExtractedLog) error {
	// 实现批量写入Doris的逻辑
	start := time.Now()
	err := output.client.loader.Write(batch)

	if err != nil {
		prom_exporter.Inc("log_event_output_doris_bulk_error", map[string]string{
			"table": output.Table,
			"code":  "load_failed",
		})
		output.logger.Error("write doris failed:", zap.Error(err))
		return err
	}

	prom_exporter.Histogram("log_event_output_doris_bulk_milliseconds",
		float64(time.Since(start).Milliseconds()),
		map[string]string{
			"database": output.Database,
			"table":    output.Table,
		},
	)

	prom_exporter.IncN("log_event_output_doris_bulk_items", len(batch),
		map[string]string{
			"database": output.Database,
			"table":    output.Table,
			"code":     "ok",
		},
	)

	return nil
}
