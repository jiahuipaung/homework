package esv6

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/flashcatcloud/fc-stash/types"
	"github.com/flashcatcloud/go-pkg/x/prom_exporter"
	"github.com/olivere/elastic/v6"
	"go.uber.org/zap"
)

var (
	pool = new(sync.Map)
)

func init() {
	prom_exporter.SetHistogramBuckets("log_event_output_elastic_bulk_milliseconds",
		[]float64{10, 100, 1000, 2000, 5000, 10000, 15000, 25000},
	)
}

type ElasticOutput struct {
	Index               string   `json:"index"`
	TimestampIndexField string   `json:"timestamp_index_field"`
	Servers             []string `json:"servers"`
	Username            string   `json:"username"`
	Password            string   `json:"password"`
	SkipTlsVerify       bool     `json:"skip_tls_verify"`

	BulkActions       int           `json:"bulk_actions,omitempty"`
	BulkSize          int           `json:"bulk_size,omitempty"`
	BulkFlushInterval time.Duration `json:"bulk_flush_interval"`
	client            *elastic.Client
	processor         *elastic.BulkProcessor // 没有做过优化的版本, 多worker时会有发送延迟
	logger            *zap.Logger
	closed            bool
	labelIndex        string
	numOfWorker       int
}

func (output *ElasticOutput) Name() string {
	return "elasticsearch_" + output.Index
}

// elasticsearch 的server端配置发生了变化, 需要重建
// elasticsearch 的index发生了变化, 不需要重建
func (output *ElasticOutput) IsDiff(other types.Output) bool {
	if other == nil {
		return true
	}
	newest, ok := other.(*ElasticOutput)
	if !ok {
		return true
	}
	if output.Username != newest.Username {
		return true
	}
	if output.Password != newest.Password {
		return true
	}
	if output.SkipTlsVerify != newest.SkipTlsVerify {
		return true
	}
	if strings.Join(output.Servers, ",") != strings.Join(newest.Servers, ",") {
		return true
	}
	return false
}

func (output *ElasticOutput) NewClient(ctx context.Context) (*elastic.Client, error) {
	var keys []string
	keys = append(keys, output.Servers...)
	keys = append(keys, output.Username)
	keys = append(keys, output.Password)
	keys = append(keys, "skip_tls_verify:"+strconv.FormatBool(output.SkipTlsVerify))
	sort.Strings(keys)
	key := strings.Join(keys, ":")
	var escli *elastic.Client
	// es client long-lived and shared
	cli, ok := pool.Load(key)
	if ok {
		escli = cli.(*elastic.Client)
	} else {
		newtransport := &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			ForceAttemptHTTP2:     true,
			MaxIdleConns:          100 * len(output.Servers),
			MaxIdleConnsPerHost:   100, // 默认值是 2, 提高连接复用
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		}
		// https skip verify
		if strings.Contains(output.Servers[0], "https") {
			tlsConfig := &tls.Config{
				InsecureSkipVerify: output.SkipTlsVerify,
			}
			newtransport.TLSClientConfig = tlsConfig
		}
		var err error
		options := []elastic.ClientOptionFunc{
			elastic.SetHttpClient(&http.Client{Transport: newtransport}),
			elastic.SetURL(output.Servers...),
			elastic.SetBasicAuth(output.Username, output.Password),
			elastic.SetSniff(false),
		}
		if len(output.Servers) == 1 {
			options = append(options, elastic.SetHealthcheck(false))
		}
		escli, err = elastic.NewClient(options...)
		if err != nil {
			return nil, err
		}
		pool.Store(key, escli)
	}
	return escli, nil
}

func (output *ElasticOutput) Init(ctx context.Context, worker int) error {
	if len(output.Servers) == 0 {
		return errors.New("elastic servers empty")
	}
	if len(output.Index) == 0 {
		return errors.New("elastic index not set")
	}
	// set default
	// 默认关闭 bulk_actions
	if output.BulkActions <= 0 {
		output.BulkActions = -1
	}
	// 以bulk_size + interval 优先
	if output.BulkSize == 0 {
		output.BulkSize = 5 << 20 // 5 MB
	}
	if output.BulkFlushInterval == 0 {
		output.BulkFlushInterval = time.Second * 10
	}
	if worker <= 0 {
		worker = 1
	}
	output.numOfWorker = worker
	// es health check
	escli, err := output.NewClient(ctx)
	if err != nil {
		return err
	}
	_, code, err := escli.Ping(output.Servers[0]).Do(ctx)
	if err != nil {
		return err
	}
	if code != http.StatusOK {
		return fmt.Errorf("got code %v from elastic server", code)
	}
	output.client = escli
	return nil
}

func (output *ElasticOutput) Start(ctx context.Context, logger *zap.Logger) error {
	if output.client == nil {
		return errors.New("nil pointer of elastic output client")
	}

	var err error
	output.logger = logger
	output.processor, err = output.client.BulkProcessor().
		BulkActions(output.BulkActions).
		BulkSize(output.BulkSize).
		FlushInterval(output.BulkFlushInterval).
		After(output.BulkAfter). // ingore
		Workers(output.numOfWorker).
		Do(ctx)
	if err != nil {
		return err
	}

	if idx := strings.Index(output.Index, "_%"); idx != -1 {
		output.labelIndex = output.Index[0:idx]
	} else {
		output.labelIndex = output.Index
	}
	return nil
}

func (output *ElasticOutput) Stop(ctx context.Context) {
	output.closed = true
	output.processor.Close()
	output.client.Stop()
}

func (output *ElasticOutput) Sink(ctx context.Context, ext types.ExtractedLog) error {
	if len(ext) == 0 {
		return errors.New("extracted nil pointer")
	}
	if output.processor == nil {
		return errors.New("elastic output handler not started")
	}
	index := output.Index
	if len(output.TimestampIndexField) > 0 {
		t, found := ext[output.TimestampIndexField]
		if found {
			date, ok := t.(time.Time)
			if ok {
				index = types.FormatWithEventTime(index, date)
			}
		}
	}
	// elastic index name should be lowercase
	index = strings.ToLower(index)
	if output.closed {
		prom_exporter.Inc("log_event_output_error", map[string]string{
			"code":   "bulk_processor_closed",
			"target": "elasticsearch",
			"index":  output.labelIndex,
		})
		return errors.New("elastic output handler was closed")
	}

	prom_exporter.Inc("log_event_output", map[string]string{
		"target": "elasticsearch",
		"index":  output.labelIndex,
	})
	indexRequest := elastic.NewBulkIndexRequest().
		Index(index).
		Type("_doc").
		RetryOnConflict(1).
		Doc(ext)
	output.processor.Add(indexRequest)
	return nil
}

func (t *ElasticOutput) BulkAfter(executionID int64, requests []elastic.BulkableRequest, response *elastic.BulkResponse, err error) {
	// 请求出错
	if err != nil {
		t.logger.Warn("elastic bulk request warning", zap.String("index", t.Index), zap.Error(err))
		prom_exporter.Inc("log_event_output_elastic_bulk_error",
			map[string]string{
				"index": t.labelIndex,
				"code":  "http_request_failed",
			},
		)
		return
	}

	// bulk action请求成功, 对返回元素进行分析
	prom_exporter.Histogram("log_event_output_elastic_bulk_milliseconds", float64(response.Took),
		map[string]string{
			"index": t.labelIndex,
		},
	)
	// metric it
	if response.Errors {
		prom_exporter.Inc("log_event_output_elastic_bulk_error",
			map[string]string{
				"index": t.labelIndex,
				"code":  "bulk_action_failed",
			},
		)
	}
	succeed := len(response.Succeeded())
	fails := response.Failed()
	if succeed > 0 {
		prom_exporter.IncN("log_event_output_elastic_bulk_items", succeed,
			map[string]string{
				"index": t.labelIndex,
				"code":  "ok",
			},
		)
	}
	if len(fails) > 0 {
		// log first only
		if fails[0].Error != nil {
			t.logger.Warn("elastic bulk request failed", zap.String("index", t.Index),
				zap.Reflect("error", fails[0].Error))
		}
		prom_exporter.IncN("log_event_output_elastic_bulk_items", len(fails),
			map[string]string{
				"index": t.labelIndex,
				"code":  "failed",
			},
		)
	}
}
