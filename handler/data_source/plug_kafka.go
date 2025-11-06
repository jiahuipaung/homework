package data_source

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"log"
	"reflect"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/flashcatcloud/fc-stash/transform/filter"
	"github.com/flashcatcloud/fc-stash/utils"
	"github.com/mitchellh/mapstructure"
	"github.com/segmentio/kafka-go"
	"github.com/segmentio/kafka-go/sasl/plain"
)

// Ref. https://docs.confluent.io/platform/current/kafka/overview-authentication-methods.html
const (
	PlugLoggingKafkaAuthMethodmTLS  = "SSL/TLS"
	PlugLoggingKafkaAuthMethodPlain = "SASL/PLAIN"
)

type PlugLoggingKafka struct {
	Brokers        []string             `json:"kafka.brokers" mapstructure:"kafka.brokers"`
	Topic          string               `json:"kafka.topic" mapstructure:"kafka.topic"`
	Group          string               `json:"kafka.group" mapstructure:"kafka.group"`
	Authentication PlugLoggingKafkaAuth `json:"kafka.auth" mapstructure:"kafka.auth"`
	MessageSample  string               `json:"kafka.message.sample" mapstructure:"kafka.message.sample"`
}

type PlugLoggingKafkaAuth struct {
	Method string                   `json:"method" mapstructure:"method"`
	MTLS   PlugLoggingKafkaAuthmTLS `json:"mtls" mapstructure:"mtls"`
	SASL   PlugLoggingKafkaAuthSASL `json:"sasl" mapstructure:"sasl"`
}

type PlugLoggingKafkaAuthmTLS struct {
	ClientCert string `json:"client.cert" mapstructure:"client.cert"` // client.cert.pem
	ClientKey  string `json:"client.key" mapstructure:"client.key"`   // client.key.pem
	CACert     string `json:"ca.cert" mapstructure:"ca.cert"`         // ca.cert
	ServerName string `json:"server.name" mapstructure:"server.name"` // server.name 不知道什么用处
}

type PlugLoggingKafkaAuthSASL struct {
	Username string `json:"client.username" mapstructure:"client.username"`
	Password string `json:"client.password" mapstructure:"client.password"`
}

func NewPlugKafkaWithSettings(settings interface{}) (*PlugLoggingKafka, error) {
	newest := new(PlugLoggingKafka)

	settingsMap := map[string]interface{}{}
	if reflect.TypeOf(settings).Kind() == reflect.String {
		if err := utils.JsonDecodeString(settings.(string), &settingsMap); err != nil {
			return nil, err
		}
	} else {
		var assert bool
		settingsMap, assert = settings.(map[string]interface{})
		if !assert {
			return nil, errors.New("settings type invalid")
		}
	}
	if err := mapstructure.Decode(settingsMap, newest); err != nil {
		return nil, err
	}

	return newest, nil
}

func (p *PlugLoggingKafka) Validate(ctx context.Context) (err error) {
	if len(p.Brokers) == 0 {
		return errors.New("missing brokers")
	}
	if len(p.Topic) == 0 {
		return errors.New("topic")
	}
	if len(p.Group) == 0 {
		return errors.New("consumer_group")
	}
	if len(p.Authentication.Method) > 0 {
		switch p.Authentication.Method {
		case PlugLoggingKafkaAuthMethodmTLS:
			// broker不需要认证时, 任何证书都不需要
			if len(strings.TrimSpace(p.Authentication.MTLS.ClientCert)) > 0 {
				if len(p.Authentication.MTLS.CACert) == 0 {
					return errors.New("CA CERTIFICATE")
				}
				if len(p.Authentication.MTLS.ClientKey) == 0 {
					return errors.New("CLIENT PRIVATE KEY")
				}
				if len(p.Authentication.MTLS.ClientCert) == 0 {
					return errors.New("CLIENT CERTIFICATE")
				}
			}
		case PlugLoggingKafkaAuthMethodPlain:
			if len(p.Authentication.SASL.Username) == 0 {
				return errors.New("username")
			}

			if len(p.Authentication.SASL.Password) == 0 {
				return errors.New("password")
			}

		default:
			return errors.New("not support")
		}
	}

	return nil
}

// 获取一条消息样例
// 不指定consumergroup, 指定partition, 消费最新的一条数据
// 从最新开始读, 如果读不到, 就算了, 不会从最旧再回溯一条
func (p *PlugLoggingKafka) GetMessageSample(ctx context.Context,
	funcs []*filter.StringPreFunction, fromBeginning ...bool) (string, error) {
	// p.Group = "sample_by_flashcat" // 避免validate()报错

	for i := range funcs {
		// 再校验一次
		if err := funcs[i].Compile(); err != nil {
			return "", err
		}
	}
	dialer, err := p.NewDialer(ctx)
	if err != nil {
		return "", err
	}

	conn, err := p.NewConn(ctx)
	if err != nil {
		return "", err
	}
	// 获取分片属性
	partitions, err := conn.ReadPartitions(p.Topic)
	if err != nil {
		return "", err
	}
	ready := make(chan struct{}, 1)
	readyClosed := false
	readyMutex := &sync.Mutex{}
	var result string
	wg := sync.WaitGroup{}
	for i := range partitions {
		wg.Add(1)
		go func(partition int, ctx context.Context) {
			defer wg.Done()
			reader := kafka.NewReader(kafka.ReaderConfig{
				Brokers:   p.Brokers,
				Topic:     p.Topic,
				Partition: partition,
				MaxBytes:  10e6, // 10MB
				Dialer:    dialer,
			})

			defer reader.Close()

			// 默认from latest, 除非显示的标注
			if !(len(fromBeginning) > 0 && fromBeginning[0]) {
				reader.SetOffset(kafka.LastOffset)
			}

			// 一旦某个分片读到了数据则其他的都退出
			msgch := make(chan string, 1)
			msgchClosed := false
			readerch := make(chan struct{})
			go func(ctx context.Context) {
				// ctx timeout, 15秒
				timeout, cancel := context.WithTimeout(ctx, time.Second*15)
				defer func() {
					if errP := recover(); errP != nil {
						log.Printf("go routine run error: %+v, stack: %s\n",
							errP, utils.Stack(3))
					}
				}()
				defer close(readerch)
				defer cancel() // cancle() 执行后, 关掉readerch

				for {
					var msg kafka.Message
					msg, err = reader.ReadMessage(timeout)
					if err != nil { // 超时 或 其他错误
						return
					}
					if len(msg.Value) > 0 {
						// 外层已退出, 这里也可以直接退出
						if msgchClosed {
							return
						}
						select {
						case msgch <- string(msg.Value):
						case <-time.After(time.Second * 2): // 消费者已关闭, msgch无法再写入
							return
						}
					}
				}
			}(ctx)
			for {
				select {
				case message := <-msgch:
					converts, err := filter.StringPrePipelineInputHandle(message, funcs, true)
					if err == nil && len(converts) > 0 {
						for cidx := range converts {
							mstr, ok := converts[cidx].(string)
							if ok && len(mstr) > 0 {
								converted, match := filter.StringPreExtractHandle(mstr, funcs, true)
								if match {
									readyMutex.Lock() // 避免重复锁
									if !readyClosed {
										readyClosed = true
										close(ready)
									}
									readyMutex.Unlock()
									msgchClosed = true
									result = converted
									return
								}
							}
						}
					}
				case <-readerch: // reader已关闭, 超时或当前partition读到目标样例
					msgchClosed = true
					return
				case <-ready: // ready已关闭, 代表其他partition已读到目标样例
					msgchClosed = true
					return
				}
			}
		}(partitions[i].ID, ctx)
	}
	wg.Wait()

	return result, nil
}

type KafkaPartitionDescribe struct {
	Topic         string `json:"topic"`
	Group         string `json:"group"`
	Partition     int    `json:"partition"`
	CurrentOffset int64  `json:"current_offset"`
	LogEndOffset  int64  `json:"log_end_offset"`
	Lag           int64  `json:"lag"`
	ClientID      string `json:"client_id"`
}

func (p *PlugLoggingKafka) GetKafkaConsumerCurrentLag(ctx context.Context) ([]KafkaPartitionDescribe, error) {
	conn, err := p.NewConn(ctx)
	if err != nil {
		return nil, err
	}
	// 获取分片属性
	partitions, err := conn.ReadPartitions(p.Topic)
	if err != nil {
		return nil, err
	}
	client, err := p.NewClient(ctx)
	if err != nil {
		return nil, err
	}
	var offsetReqs []kafka.OffsetRequest
	var offsetFetchPts []int
	for _, partition := range partitions {
		offsetReqs = append(offsetReqs, kafka.LastOffsetOf(partition.ID)) // log-end-offset
		offsetFetchPts = append(offsetFetchPts, partition.ID)
	}
	// 最新的offset
	latest, err := client.ListOffsets(ctx, &kafka.ListOffsetsRequest{
		Addr:   kafka.TCP(p.Brokers...),
		Topics: map[string][]kafka.OffsetRequest{p.Topic: offsetReqs},
	})
	if err != nil {
		return nil, err
	}
	// 最新消费的offset
	current, err := client.OffsetFetch(ctx, &kafka.OffsetFetchRequest{
		Addr:    kafka.TCP(p.Brokers...),
		GroupID: p.Group,
		Topics:  map[string][]int{p.Topic: offsetFetchPts},
	})
	if err != nil {
		return nil, err
	}
	// 消费者列表
	members, err := client.DescribeGroups(ctx, &kafka.DescribeGroupsRequest{
		Addr:     kafka.TCP(p.Brokers...),
		GroupIDs: []string{p.Group},
	})
	if err != nil {
		return nil, err
	}
	var ret []KafkaPartitionDescribe
	memberMatch := func(member kafka.DescribeGroupsResponseMember, topic string, partition int) bool {
		for _, target := range member.MemberAssignments.Topics {
			if target.Topic == topic {
				for _, p := range target.Partitions {
					if p == partition {
						return true
					}
				}
			}
		}
		return false

	}
	for i := range partitions {
		ID := partitions[i].ID
		desc := KafkaPartitionDescribe{Topic: p.Topic, Group: p.Group, Partition: ID}
		for _, offset := range latest.Topics[p.Topic] {
			if offset.Partition == ID {
				desc.LogEndOffset = offset.LastOffset
			}
		}
		for _, offset := range current.Topics[p.Topic] {
			if offset.Partition == ID {
				desc.CurrentOffset = offset.CommittedOffset
			}
		}
		desc.Lag = desc.LogEndOffset - desc.CurrentOffset
		if len(members.Groups) > 0 {
			for _, member := range members.Groups[0].Members {
				if memberMatch(member, p.Topic, ID) {
					desc.ClientID = member.ClientID
				}
			}
		}
		ret = append(ret, desc)
	}
	if len(ret) > 0 {
		sort.Slice(ret, func(i, j int) bool {
			return ret[i].Partition < ret[j].Partition
		})
	}
	return ret, nil
}

func (p *PlugLoggingKafka) ResetOffset(ctx context.Context, offset int64) error {
	if offset <= 0 && offset != kafka.LastOffset {
		return errors.New("invalid offset")
	}
	conn, err := p.NewConn(ctx)
	if err != nil {
		return err
	}
	partitions, err := conn.ReadPartitions(p.Topic)
	if err != nil {
		return err
	}

	client, err := p.NewClient(ctx)
	if err != nil {
		return err
	}
	// 校验是否所有消费者已停止
	members, err := client.DescribeGroups(ctx, &kafka.DescribeGroupsRequest{
		Addr:     kafka.TCP(p.Brokers...),
		GroupIDs: []string{p.Group},
	})
	if err != nil {
		return err
	}
	if len(members.Groups) > 0 && len(members.Groups[0].Members) > 0 {
		return errors.New("active consumer exist")
	}
	var offsetReqs []kafka.OffsetRequest
	for _, partition := range partitions {
		offsetReqs = append(offsetReqs, kafka.LastOffsetOf(partition.ID))  // log-end-offset
		offsetReqs = append(offsetReqs, kafka.FirstOffsetOf(partition.ID)) // beginning
	}
	// 最新的offset
	latest, err := client.ListOffsets(ctx, &kafka.ListOffsetsRequest{
		Addr:   kafka.TCP(p.Brokers...),
		Topics: map[string][]kafka.OffsetRequest{p.Topic: offsetReqs},
	})
	if err != nil {
		return err
	}

	dialer, err := p.NewDialer(ctx)
	if err != nil {
		return err
	}
	group, err := kafka.NewConsumerGroup(kafka.ConsumerGroupConfig{
		ID:      p.Group,
		Brokers: p.Brokers,
		Topics:  []string{p.Topic},
		Dialer:  dialer,
	})
	if err != nil {
		return err
	}
	defer group.Close()

	gen, err := group.Next(ctx)
	if err != nil {
		return err
	}
	target := make(map[string]map[int]int64)
	target[p.Topic] = make(map[int]int64)
	for i := range partitions {
		for _, current := range latest.Topics[p.Topic] {
			if current.Partition == partitions[i].ID {
				var value int64
				if offset == kafka.LastOffset || offset > current.LastOffset {
					value = current.LastOffset

				} else if offset < current.FirstOffset {
					value = current.FirstOffset

				} else {
					value = offset
				}
				if value > 0 {
					target[p.Topic][partitions[i].ID] = value
				}
			}
		}
	}
	if len(target[p.Topic]) > 0 {
		err = gen.CommitOffsets(target)
		if err != nil {
			return err
		}
	}
	return nil
}

func (p *PlugLoggingKafka) NewReader(ctx context.Context) (*kafka.Reader, error) {
	dialer, err := p.NewDialer(ctx)
	if err != nil {
		return nil, err
	}
	return kafka.NewReader(kafka.ReaderConfig{
		Brokers:   p.Brokers,
		Topic:     p.Topic,
		Partition: 0,
		MaxBytes:  10e6, // 10MB
		Dialer:    dialer,
	}), nil
}

func (p *PlugLoggingKafka) NewDialer(ctx context.Context) (*kafka.Dialer, error) {
	if err := p.Validate(ctx); err != nil {
		return nil, err
	}
	if len(p.Authentication.Method) == 0 {
		return kafka.DefaultDialer, nil
	}
	if p.Authentication.Method == PlugLoggingKafkaAuthMethodmTLS {
		tlsConfig, err := p.newTlsConfig()
		if err != nil {
			return nil, err
		}
		return &kafka.Dialer{
			Timeout:   5 * time.Second,
			DualStack: true,
			TLS:       tlsConfig,
		}, nil
	}

	if p.Authentication.Method == PlugLoggingKafkaAuthMethodPlain {
		user, pass, err := p.newSASLConfig()
		if err != nil {
			return nil, err
		}
		return &kafka.Dialer{
			Timeout:   5 * time.Second,
			DualStack: true,
			SASLMechanism: plain.Mechanism{
				Username: user,
				Password: pass,
			},
		}, nil
	}

	return nil, errors.New("认证类型不匹配")
}

func (p *PlugLoggingKafka) NewConn(ctx context.Context) (*kafka.Conn, error) {
	dialer, err := p.NewDialer(ctx)
	if err != nil {
		return nil, err
	}
	return dialer.Dial("tcp", p.Brokers[0])
}

func (p *PlugLoggingKafka) NewClient(ctx context.Context) (*kafka.Client, error) {
	if err := p.Validate(ctx); err != nil {
		return nil, err
	}
	if len(p.Authentication.Method) == 0 {
		return &kafka.Client{}, nil
	}
	if p.Authentication.Method == PlugLoggingKafkaAuthMethodmTLS {
		tlsConfig, err := p.newTlsConfig()
		if err != nil {
			return nil, err
		}
		client := new(kafka.Client)
		client.Transport = &kafka.Transport{TLS: tlsConfig}
		return client, nil
	}

	if p.Authentication.Method == PlugLoggingKafkaAuthMethodPlain {
		user, pass, err := p.newSASLConfig()
		if err != nil {
			return nil, err
		}
		client := new(kafka.Client)
		client.Transport = &kafka.Transport{SASL: plain.Mechanism{
			Username: user,
			Password: pass,
		}}

		return client, nil
	}
	return nil, errors.New("认证类型不匹配")
}

func (p *PlugLoggingKafka) newTlsConfig() (*tls.Config, error) {
	if p.Authentication.Method != PlugLoggingKafkaAuthMethodmTLS {
		return nil, errors.New("认证类型不匹配")
	}
	tlsConfig := tls.Config{}
	// 不验证服务器的证书(避免非权威CA签名的问题)
	// broker不需要认证时, 跳过验证
	tlsConfig.InsecureSkipVerify = true

	if len(strings.TrimSpace(p.Authentication.MTLS.ClientCert)) > 0 {
		// Load client cert
		cert, err := tls.X509KeyPair([]byte(strings.TrimSpace(p.Authentication.MTLS.ClientCert)),
			[]byte(strings.TrimSpace(p.Authentication.MTLS.ClientKey)))
		if err != nil {
			return nil, errors.New("加载证书出错:" + err.Error())
		}
		tlsConfig.Certificates = []tls.Certificate{cert}

		// Load CA cert
		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM([]byte(strings.TrimSpace(p.Authentication.MTLS.CACert)))
		tlsConfig.RootCAs = caCertPool
	}
	return &tlsConfig, nil
}

func (p *PlugLoggingKafka) newSASLConfig() (string, string, error) {
	if p.Authentication.Method != PlugLoggingKafkaAuthMethodPlain {
		return "", "", errors.New("认证类型不匹配")
	}

	password := p.Authentication.SASL.Password
	return p.Authentication.SASL.Username, password, nil
}
