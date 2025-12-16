package kafka

import (
	"context"
	"crypto/tls"
	"errors"
	"strings"
	"sync"
	"time"
	
	"github.com/IBM/sarama"
	"github.com/xdg-go/scram"
	"go.opentelemetry.io/otel"
)

// Config defines Kafka connection and producer defaults.
//
// It is intentionally infrastructure-only: topics and consumer groups can be
// decided by each service.
type Config struct {
	Brokers       []string `yaml:"brokers" mapstructure:"brokers"`
	Topic         string   `yaml:"topic" mapstructure:"topic"`
	ClientID      string   `yaml:"client_id" mapstructure:"client_id"`
	Username      string   `yaml:"username" mapstructure:"username"`
	Password      string   `yaml:"password" mapstructure:"password"`
	SASLMechanism string   `yaml:"sasl_mechanism" mapstructure:"sasl_mechanism"`
	TLSEnabled    bool     `yaml:"tls_enabled" mapstructure:"tls_enabled"`
	
	// RequiredAcks supports: "none" | "one" | "all" (default: all).
	RequiredAcks string `yaml:"required_acks" mapstructure:"required_acks"`
	// MaxAttempts controls producer retry max attempts (default: 3).
	MaxAttempts int `yaml:"max_attempts" mapstructure:"max_attempts"`
}

// PublishObserver is an optional hook to observe publish latency and errors.
//
// It is intentionally metrics-backend agnostic (no Prometheus dependency) so each
// service can map it to its own metrics and labels.
type PublishObserver interface {
	ObservePublish(topic string, duration time.Duration, err error)
}

// ConsumeObserver is an optional hook to observe message processing latency and errors.
//
// "Consume" in this context means a service has received a Kafka message and attempted
// to process it. The service decides whether "success" is only after business processing.
type ConsumeObserver interface {
	ObserveConsume(topic, group, eventType string, duration time.Duration, err error)
}

// Manager manages a shared Kafka sync producer and a base sarama config for consumers.
type Manager struct {
	cfg      Config
	producer sarama.SyncProducer
	baseConf *sarama.Config
	
	observerMu      sync.RWMutex
	publishObserver PublishObserver
	consumeObserver ConsumeObserver
	
	closeOnce sync.Once
}

// kafkaHeadersCarrier implements propagation.TextMapCarrier for Kafka headers.
type kafkaHeadersCarrier []sarama.RecordHeader

func (c *kafkaHeadersCarrier) Get(key string) string {
	for _, h := range *c {
		if string(h.Key) == key {
			return string(h.Value)
		}
	}
	return ""
}

func (c *kafkaHeadersCarrier) Set(key, value string) {
	*c = append(*c, sarama.RecordHeader{
		Key:   []byte(key),
		Value: []byte(value),
	})
}

func (c *kafkaHeadersCarrier) Keys() []string {
	keys := make([]string, 0, len(*c))
	for _, h := range *c {
		keys = append(keys, string(h.Key))
	}
	return keys
}

// NewManager builds a Kafka manager using the provided config.
func NewManager(cfg Config) (*Manager, error) {
	
	if len(cfg.Brokers) == 0 {
		return nil, errors.New("kafka brokers empty")
	}
	base := sarama.NewConfig()
	base.Version = sarama.V2_1_0_0
	if cfg.ClientID != "" {
		base.ClientID = cfg.ClientID
	}
	
	base.Producer.Return.Successes = true
	base.Producer.Retry.Max = max(cfg.MaxAttempts, 3)
	base.Producer.RequiredAcks = parseRequiredAcks(cfg.RequiredAcks)
	base.Producer.Idempotent = false
	
	if cfg.TLSEnabled {
		base.Net.TLS.Enable = true
		base.Net.TLS.Config = &tls.Config{MinVersion: tls.VersionTLS12}
	}
	
	if cfg.Username != "" {
		base.Net.SASL.Enable = true
		base.Net.SASL.User = cfg.Username
		base.Net.SASL.Password = cfg.Password
		mech := strings.ToUpper(strings.TrimSpace(cfg.SASLMechanism))
		switch mech {
		case "SCRAM-SHA-512":
			base.Net.SASL.Mechanism = sarama.SASLTypeSCRAMSHA512
			base.Net.SASL.SCRAMClientGeneratorFunc = func() sarama.SCRAMClient {
				return newSCRAMClient(scram.SHA512)
			}
		case "SCRAM-SHA-256":
			base.Net.SASL.Mechanism = sarama.SASLTypeSCRAMSHA256
			base.Net.SASL.SCRAMClientGeneratorFunc = func() sarama.SCRAMClient {
				return newSCRAMClient(scram.SHA256)
			}
		case "PLAIN", "":
			base.Net.SASL.Mechanism = sarama.SASLTypePlaintext
		default:
			base.Net.SASL.Mechanism = sarama.SASLTypePlaintext
		}
	}
	
	producer, err := sarama.NewSyncProducer(cfg.Brokers, base)
	if err != nil {
		return nil, err
	}
	return &Manager{cfg: cfg, producer: producer, baseConf: base}, nil
}

// SetPublishObserver installs or replaces the publish observer. It is safe to call
// before the manager is used concurrently.
func (m *Manager) SetPublishObserver(observer PublishObserver) {
	if m == nil {
		return
	}
	m.observerMu.Lock()
	m.publishObserver = observer
	m.observerMu.Unlock()
}

func (m *Manager) publishObserverSnapshot() PublishObserver {
	if m == nil {
		return nil
	}
	m.observerMu.RLock()
	observer := m.publishObserver
	m.observerMu.RUnlock()
	return observer
}

// SetConsumeObserver installs or replaces the consume observer. It is safe to call
// before the manager is used concurrently.
func (m *Manager) SetConsumeObserver(observer ConsumeObserver) {
	if m == nil {
		return
	}
	m.observerMu.Lock()
	m.consumeObserver = observer
	m.observerMu.Unlock()
}

func (m *Manager) consumeObserverSnapshot() ConsumeObserver {
	if m == nil {
		return nil
	}
	m.observerMu.RLock()
	observer := m.consumeObserver
	m.observerMu.RUnlock()
	return observer
}

// Publish sends a message to the given topic (falls back to cfg.Topic).
// Automatically injects trace context into Kafka headers for distributed tracing.
func (m *Manager) Publish(ctx context.Context, topic string, key, value []byte) (err error) {
	if m == nil {
		return errors.New("kafka manager nil")
	}
	start := time.Now()
	defer func() {
		if observer := m.publishObserverSnapshot(); observer != nil {
			observer.ObservePublish(topic, time.Since(start), err)
		}
	}()
	if topic == "" {
		topic = m.cfg.Topic
	}
	if topic == "" {
		return errors.New("kafka topic empty")
	}
	
	var headers kafkaHeadersCarrier
	propagator := otel.GetTextMapPropagator()
	propagator.Inject(ctx, &headers)
	
	msg := &sarama.ProducerMessage{Topic: topic}
	if len(key) > 0 {
		msg.Key = sarama.ByteEncoder(key)
	}
	if len(value) > 0 {
		msg.Value = sarama.ByteEncoder(value)
	}
	for _, h := range headers {
		msg.Headers = append(msg.Headers, h)
	}
	
	select {
	case <-ctx.Done():
		err = ctx.Err()
		return err
	default:
	}
	_, _, err = m.producer.SendMessage(msg)
	return err
}

// ObserveConsume triggers consume observer if installed.
func (m *Manager) ObserveConsume(topic, group, eventType string, duration time.Duration, err error) {
	if observer := m.consumeObserverSnapshot(); observer != nil {
		observer.ObserveConsume(topic, group, eventType, duration, err)
	}
}

// NewConsumerGroup returns a consumer group using shared base config.
func (m *Manager) NewConsumerGroup(group string) (sarama.ConsumerGroup, error) {
	if m == nil {
		return nil, errors.New("kafka manager nil")
	}
	if group == "" {
		return nil, errors.New("kafka consumer group empty")
	}
	cfg := *m.baseConf
	cfg.Consumer.Return.Errors = true
	cfg.Consumer.Offsets.Initial = sarama.OffsetNewest
	return sarama.NewConsumerGroup(m.cfg.Brokers, group, &cfg)
}

// Close shuts down producer.
func (m *Manager) Close() error {
	if m == nil {
		return nil
	}
	var err error
	m.closeOnce.Do(func() {
		if m.producer != nil {
			err = m.producer.Close()
		}
	})
	return err
}

func parseRequiredAcks(v string) sarama.RequiredAcks {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "none":
		return sarama.NoResponse
	case "one":
		return sarama.WaitForLocal
	case "all", "":
		return sarama.WaitForAll
	default:
		return sarama.WaitForAll
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

type scramClient struct {
	*scram.Client
	*scram.ClientConversation
	hash scram.HashGeneratorFcn
}

func newSCRAMClient(hash scram.HashGeneratorFcn) sarama.SCRAMClient {
	return &scramClient{hash: hash}
}

func (c *scramClient) Begin(userName, password, authzID string) error {
	client, err := c.hash.NewClient(userName, password, authzID)
	if err != nil {
		return err
	}
	c.Client = client
	c.ClientConversation = client.NewConversation()
	return nil
}

func (c *scramClient) Step(challenge string) (string, error) {
	return c.ClientConversation.Step(challenge)
}

func (c *scramClient) Done() bool {
	return c.ClientConversation.Done()
}
