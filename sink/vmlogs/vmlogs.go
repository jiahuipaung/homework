package victorialogs

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/flashcatcloud/fc-stash/types"
	"github.com/flashcatcloud/go-pkg/x/prom_exporter"
	"go.uber.org/zap"
)

const DefaultMessage = "default message"

type VMLogsOutput struct {
	Endpoints       []string          `json:"endpoints"`
	Username        string            `json:"username"`
	Password        string            `json:"password"`
	SkipTlsVerify   bool              `json:"skip_tls_verify"`
	Headers         map[string]string `json:"headers"`
	RequestTimeout  time.Duration     `json:"request_timeout"`
	MaxRetryBackoff time.Duration     `json:"max_retry_backoff"`
	Client          *http.Client      `json:"client"`

	// VMLogs fields 从配置中获取
	MsgField     string   `json:"msg_field"`
	TimeField    string   `json:"time_field"`
	StreamFields []string `json:"stream_fields"`

	BatchActions       int           `json:"batch_actions"`
	BatchSize          int           `json:"batch_size"`
	BatchFlushInterval time.Duration `json:"batch_flush_interval"`
	NumOfWorker        int           `json:"num_of_worker"`

	processor *BatchProcessor
	logger    *zap.Logger
	closed    bool
}

func (o *VMLogsOutput) Name() string {
	return "vmlogs"
}

// IsDiff 检查配置是否变化，是否需要重建
func (o *VMLogsOutput) IsDiff(other types.Output) bool {
	if other == nil {
		return true
	}
	newest, ok := other.(*VMLogsOutput)
	if !ok {
		return true
	}
	// Client层面的变化需要重建
	if o.RequestTimeout != newest.RequestTimeout ||
		o.SkipTlsVerify != newest.SkipTlsVerify ||
		strings.Join(o.Endpoints, ",") != strings.Join(newest.Endpoints, ",") {
		return true
	}
	return false
}

// Init initializes the VMLogs sink
func (o *VMLogsOutput) Init(ctx context.Context, worker int) error {
	if len(o.Endpoints) == 0 {
		return errors.New("vmlogs endpoints are not set")
	}

	// Bulk设置默认值
	if o.BatchActions <= 0 {
		o.BatchActions = 1000
	}
	if o.BatchSize <= 0 {
		o.BatchSize = 2 << 20 // 2MB
	}
	if o.BatchFlushInterval == 0 {
		o.BatchFlushInterval = time.Second * 10
	}
	if o.MaxRetryBackoff == 0 {
		o.MaxRetryBackoff = time.Duration(15) * time.Second
	}
	if worker <= 0 {
		worker = 2
	}
	o.NumOfWorker = worker

	if o.RequestTimeout <= 0 {
		o.RequestTimeout = time.Duration(10) * time.Second
	}
	o.Client = &http.Client{
		Timeout: o.RequestTimeout,
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			ForceAttemptHTTP2:     true,
			MaxIdleConns:          100 * len(o.Endpoints),
			MaxIdleConnsPerHost:   100,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: o.SkipTlsVerify,
			},
		},
	}

	return nil
}

// Start begins the sink process
func (o *VMLogsOutput) Start(ctx context.Context, logger *zap.Logger) error {
	o.logger = logger
	// Create BatchProcessor
	o.processor = NewBatchProcessor(o).
		BatchActions(o.BatchActions).
		BatchSize(o.BatchSize).
		FlushInterval(o.BatchFlushInterval).
		Works(o.NumOfWorker).
		MaxRetryBackoff(o.MaxRetryBackoff)

	if err := o.processor.Start(ctx); err != nil {
		return fmt.Errorf("failed to start processor: %w", err)
	}

	return nil
}

// Stop gracefully shuts down the sink
func (o *VMLogsOutput) Stop(ctx context.Context) {
	o.closed = true
	if o.processor != nil {
		err := o.processor.Stop()
		if err != nil {
			return
		}
	}
}

// Sink sends a log to the bulk processor
func (o *VMLogsOutput) Sink(ctx context.Context, ext types.ExtractedLog) error {
	if len(ext) == 0 {
		return errors.New("extracted nil pointer")
	}
	if o.processor == nil {
		return errors.New("vmlogs processor is nil")
	}
	if o.closed {
		prom_exporter.Inc("log_event_output_error", map[string]string{
			"code":   "sink_closed",
			"target": "vmlogs",
		})
		return errors.New("vmlogs sink was closed")
	}

	if o.MsgField == "_msg" { // 用户选择自动生成，设置 _msg: default_message
		ext["_msg"] = DefaultMessage
	}
	if o.TimeField != "" { // 用户自选时间字段，只处理time.Time类型
		if val, ok := ext[o.TimeField]; ok {
			if _, isTime := val.(time.Time); !isTime {
				if o.logger != nil {
					o.logger.Debug("Unsupported time format", zap.String("field", o.TimeField))
				}
			}
		} else {
			if o.logger != nil {
				o.logger.Debug("The time_field is not exist", zap.String("field", o.TimeField))
			}
		}
	}
	if len(o.StreamFields) != 0 { // 若用户配置的stream不存在，不影响log写入，相应的stream配置为空，这里仅做warn处理
		for _, field := range o.StreamFields {
			if _, ok := ext[field]; !ok {
				if o.logger != nil {
					o.logger.Debug("The stream_field is not exist", zap.String("field", field))
				}
			}
		}
	}

	o.processor.Add(ext)
	return nil
}
