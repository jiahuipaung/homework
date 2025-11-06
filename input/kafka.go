package input

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"time"

	"github.com/flashcatcloud/fc-stash/transform/filter"
	"github.com/flashcatcloud/fc-stash/types"
	"github.com/flashcatcloud/fc-stash/utils"
	"github.com/flashcatcloud/go-pkg/x/prom_exporter"
	"github.com/segmentio/kafka-go"
	"github.com/segmentio/kafka-go/sasl/plain"
	"go.uber.org/zap"
)

type InputKafka struct {
	Brokers     []string                    `json:"brokers"`
	Topic       string                      `json:"topic"`
	GroupID     string                      `json:"group_id"`
	AuthMethod  string                      `json:"auth_method"`
	StartOffset int64                       `json:"start_offset"`
	QueueSize   int                         `json:"queue_size"`     // 缓存的消息数量
	MTLS        *InputKafkaAuthmTLS         `json:"mtls,omitempty"` // if not empty
	SASL        *InputKafkaAuthSASL         `json:"sasl,omitempty"`
	PreFunction []*filter.StringPreFunction `json:"pre_function"` // 前置处理
	Stopped     bool                        `json:"-"`            // 依赖上层重启
	hasPreFunc  bool
	reader      *kafka.Reader
	logger      *zap.Logger
}

type InputKafkaAuthmTLS struct {
	ClientCert string `json:"client.cert"` // client.cert.pem
	ClientKey  string `json:"client.key"`  // client.key.pem
	CACert     string `json:"ca.cert"`     // ca.cert
}

type InputKafkaAuthSASL struct {
	Username string `json:"client.username" mapstructure:"client.username"`
	Password string `json:"client.password" mapstructure:"client.password"`
}

func (k *InputKafka) Init(ctx context.Context) error {
	if len(k.Brokers) == 0 {
		return errors.New("kafka brokers empty")
	}
	if len(k.Topic) == 0 {
		return errors.New("kafka topic empty")
	}
	if len(k.GroupID) == 0 {
		return errors.New("consumer group empty")
	}
	// 第一次启动消费时, 从最新开始, 避免消费无效数据
	if k.StartOffset == 0 {
		k.StartOffset = kafka.LastOffset
	}
	if k.QueueSize <= 0 {
		k.QueueSize = 2000 // set defautl to pipeline.chsize * 2
	}
	k.hasPreFunc = false
	for _, pfunc := range k.PreFunction {
		if pfunc.IsPrePipelineInputHandle() {
			k.hasPreFunc = true
		}
	}
	return nil
}

func (k *InputKafka) getDialer() (*kafka.Dialer, error) {

	if k.SASL == nil && k.MTLS == nil {
		return nil, nil
	}

	d := kafka.Dialer{
		Timeout:   5 * time.Second,
		DualStack: true,
	}

	tlsConfig := tls.Config{}
	// 不验证服务器的证书(避免非权威CA签名的问题)
	// broker不需要认证时, 跳过验证
	tlsConfig.InsecureSkipVerify = true
	if k.MTLS != nil {
		if len(strings.TrimSpace(k.MTLS.ClientCert)) > 0 {
			// Load client cert
			cert, err := tls.X509KeyPair([]byte(strings.TrimSpace(k.MTLS.ClientCert)),
				[]byte(strings.TrimSpace(k.MTLS.ClientKey)))
			if err != nil {
				return nil, errors.New("加载证书出错:" + err.Error())
			}
			tlsConfig.Certificates = []tls.Certificate{cert}

			// Load CA cert
			caCertPool := x509.NewCertPool()
			caCertPool.AppendCertsFromPEM([]byte(strings.TrimSpace(k.MTLS.CACert)))
			tlsConfig.RootCAs = caCertPool
		}
		d.TLS = &tlsConfig
	}

	// SASL PLAIN
	if k.SASL != nil {
		d.SASLMechanism = plain.Mechanism{
			Username: k.SASL.Username,
			Password: k.SASL.Password,
		}
	}

	return &d, nil
}

func (k *InputKafka) Start(ctx context.Context, logger *zap.Logger, queue chan<- *types.LogEvent) error {
	// 启动前校验, 端口是否存活
	checkpass := false
	for i := range k.Brokers {
		conn, err := net.DialTimeout("tcp", k.Brokers[i], time.Second*2)
		if err == nil && conn != nil {
			checkpass = true
			conn.Close()
		}
	}
	if !checkpass {
		// 没有可达的broker
		return errors.New("no available brokers")
	}

	d, err := k.getDialer()
	if err != nil {
		return err
	}
	k.logger = logger

	// offset 默认是 FirstOffset, 即 oldest
	k.reader = kafka.NewReader(kafka.ReaderConfig{
		Brokers:               k.Brokers,
		GroupID:               k.GroupID,
		Topic:                 k.Topic,
		MaxBytes:              10485760, // 10M
		CommitInterval:        time.Second * 5,
		StartOffset:           k.StartOffset,
		QueueCapacity:         k.QueueSize,
		Logger:                newKqLogger(logger),
		WatchPartitionChanges: true,
		Dialer:                d,
	})

	go func(reader *kafka.Reader) {
		defer func() {
			_ = reader.Close()
			k.Stopped = true
		}()

		for {
			msg, err := reader.ReadMessage(ctx)
			// io.EOF means consumer closed
			// io.ErrClosedPipe means committing messages on the consumer,
			// kafka will refire the messages on uncommitted messages, ignore
			if err != nil {
				if err == io.EOF || err == io.ErrClosedPipe {
					k.logger.Sugar().Errorf("topic:[%s] group[%s] conn closed, got %v", k.Topic, k.GroupID, err)
					return
				}
				if err.Error() == "fetching message: EOF" {
					k.logger.Sugar().Errorf("topic:[%s] group[%s] conn closed, got %v", k.Topic, k.GroupID, err)
					return
				}
				k.logger.Error("kafka read message failed", zap.Error(err))
				continue
			}
			// 空字符串过滤
			if len(msg.Value) == 1 && msg.Value[0] == '\n' {
				continue
			}
			prom_exporter.Inc("log_event_input", map[string]string{
				"source": "kafka",
				"topic":  k.Topic,
				"group":  k.GroupID,
			})
			prom_exporter.IncN("log_event_in_bytes", len(msg.Value), map[string]string{
				"topic": k.Topic,
			})

			// 与之前的逻辑一致
			if !k.hasPreFunc {
				queue <- &types.LogEvent{
					Source:  "kafka:" + k.Topic,
					Message: utils.BytesToString(msg.Value),
				}
			} else {
				oldmsg := utils.BytesToString(msg.Value)
				newmsgs, err := filter.StringPrePipelineInputHandle(oldmsg, k.PreFunction)
				if err != nil {
					k.logger.Error("kafka pre function failed",
						zap.String("message", oldmsg),
						zap.String("pre_funcs", utils.MustToJsonString(k.PreFunction)),
						zap.Error(err))
					continue
				}
				for i := range newmsgs {
					queue <- &types.LogEvent{
						Source:  "kafka:" + k.Topic,
						Message: newmsgs[i],
					}
				}
			}
		}
	}(k.reader)
	return nil
}

func (k *InputKafka) Stop(ctx context.Context) {
	if k.reader != nil {
		k.reader.Close()
	}
	k.Stopped = true
}

type kqLogger struct {
	logger *zap.Logger
}

func newKqLogger(logger *zap.Logger) *kqLogger {
	return &kqLogger{logger: logger}
}

func (l *kqLogger) Printf(format string, args ...interface{}) {
	l.logger.Info(strings.ReplaceAll(fmt.Sprintf(format, args...), "\n", " "))
}
