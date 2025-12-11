package config

// ==================== 基础配置 (所有服务都需要) ====================

// AppConfig 应用基础配置
type AppConfig struct {
	Env    string `yaml:"env" mapstructure:"env"`
	Port   int    `yaml:"port" mapstructure:"port"`
	NodeID string `yaml:"node_id" mapstructure:"node_id"`
}

// LogConfig 日志配置
type LogConfig struct {
	Format       string `yaml:"format" mapstructure:"format"`
	Level        string `yaml:"level" mapstructure:"level"`
	ReportCaller bool   `yaml:"report_caller" mapstructure:"report_caller"`
}

// ==================== 基础设施配置 ====================

// RedisConfig Redis 连接配置
type RedisConfig struct {
	Addr     string `yaml:"addr" mapstructure:"addr"`
	Username string `yaml:"username" mapstructure:"username"`
	Password string `yaml:"password" mapstructure:"password"`
	Db       int    `yaml:"db" mapstructure:"db"`
}

// PostgresConfig PostgreSQL 配置
type PostgresConfig struct {
	DSN                    string `yaml:"dsn" mapstructure:"dsn"`
	MaxOpenConns           int    `yaml:"max_open_conns" mapstructure:"max_open_conns"`
	MaxIdleConns           int    `yaml:"max_idle_conns" mapstructure:"max_idle_conns"`
	ConnMaxLifetimeSeconds int    `yaml:"conn_max_lifetime_seconds" mapstructure:"conn_max_lifetime_seconds"`
}

// KafkaConfig Kafka 配置
type KafkaConfig struct {
	Enabled       bool     `yaml:"enabled" mapstructure:"enabled"`
	Brokers       []string `yaml:"brokers" mapstructure:"brokers"`
	Topic         string   `yaml:"topic" mapstructure:"topic"`
	ConsumerGroup string   `yaml:"consumer_group" mapstructure:"consumer_group"`
	ClientID      string   `yaml:"client_id" mapstructure:"client_id"`
	Username      string   `yaml:"username" mapstructure:"username"`
	Password      string   `yaml:"password" mapstructure:"password"`
	SASLMechanism string   `yaml:"sasl_mechanism" mapstructure:"sasl_mechanism"`
	TLSEnabled    bool     `yaml:"tls_enabled" mapstructure:"tls_enabled"`
}

// ==================== 认证配置 ====================

// JWTConfig JWT 认证配置
type JWTConfig struct {
	SecretKey       string   `yaml:"secret_key" mapstructure:"secret_key"`
	AccessTokenTTL  Duration `yaml:"access_token_ttl" mapstructure:"access_token_ttl"`
	RefreshTokenTTL Duration `yaml:"refresh_token_ttl" mapstructure:"refresh_token_ttl"`
	ClockSkew       Duration `yaml:"clock_skew" mapstructure:"clock_skew"`
}

// ==================== Slot 路由配置 (SideCar & Worker 共用) ====================

// SlotConfig Slot 路由配置
type SlotConfig struct {
	Enabled              bool   `yaml:"enabled" mapstructure:"enabled"`
	TotalSlots           int    `yaml:"total_slots" mapstructure:"total_slots"`
	RedisPrefix          string `yaml:"redis_prefix" mapstructure:"redis_prefix"`
	LeaseTTLSeconds      int    `yaml:"lease_ttl_seconds" mapstructure:"lease_ttl_seconds"`
	RouteTTLSeconds      int    `yaml:"route_ttl_seconds" mapstructure:"route_ttl_seconds"`           // SideCar 使用
	RenewIntervalSeconds int    `yaml:"renew_interval_seconds" mapstructure:"renew_interval_seconds"` // Worker 使用
	MaxRetry             int    `yaml:"max_retry" mapstructure:"max_retry"`
}

// ==================== Bridge 配置 ====================

// BridgeClientConfig gRPC Bridge 客户端配置 (SideCar 使用)
type BridgeClientConfig struct {
	Address                  string            `yaml:"address" mapstructure:"address"`
	Insecure                 bool              `yaml:"insecure" mapstructure:"insecure"`
	Headers                  map[string]string `yaml:"headers" mapstructure:"headers"`
	DialTimeoutSeconds       int               `yaml:"dial_timeout_seconds" mapstructure:"dial_timeout_seconds"`
	HeartbeatIntervalSeconds int               `yaml:"heartbeat_interval_seconds" mapstructure:"heartbeat_interval_seconds"`
	ReconnectBaseSeconds     int               `yaml:"reconnect_base_seconds" mapstructure:"reconnect_base_seconds"`
	ReconnectMaxSeconds      int               `yaml:"reconnect_max_seconds" mapstructure:"reconnect_max_seconds"`
	EnableBackpressure       bool              `yaml:"enable_backpressure" mapstructure:"enable_backpressure"`
	MaxInFlightDeliver       int               `yaml:"max_inflight_deliver" mapstructure:"max_inflight_deliver"`
	PendingAckTimeoutSeconds int               `yaml:"pending_ack_timeout_seconds" mapstructure:"pending_ack_timeout_seconds"`
}

// BridgeServerConfig gRPC Bridge 服务端配置 (Worker 使用)
type BridgeServerConfig struct {
	ListenAddr               string   `yaml:"listen_addr" mapstructure:"listen_addr"`
	Namespace                string   `yaml:"namespace" mapstructure:"namespace"`
	Actions                  []string `yaml:"actions" mapstructure:"actions"`
	TLSCertFile              string   `yaml:"tls_cert_file" mapstructure:"tls_cert_file"`
	TLSKeyFile               string   `yaml:"tls_key_file" mapstructure:"tls_key_file"`
	DeliverBuffer            int      `yaml:"deliver_buffer" mapstructure:"deliver_buffer"`
	HeartbeatIntervalSeconds int      `yaml:"heartbeat_interval_seconds" mapstructure:"heartbeat_interval_seconds"`
	ReconnectInitialSeconds  int      `yaml:"reconnect_initial_seconds" mapstructure:"reconnect_initial_seconds"`
	ReconnectMaxSeconds      int      `yaml:"reconnect_max_seconds" mapstructure:"reconnect_max_seconds"`
	PendingAckTimeoutSeconds int      `yaml:"pending_ack_timeout_seconds" mapstructure:"pending_ack_timeout_seconds"`
	MaxInFlightDeliver       int      `yaml:"max_inflight_deliver" mapstructure:"max_inflight_deliver"`
}

// ==================== 可观测性配置 ====================

// TracingConfig 分布式追踪配置
type TracingConfig struct {
	Exporter     string            `yaml:"exporter" mapstructure:"exporter"`
	Endpoint     string            `yaml:"endpoint" mapstructure:"endpoint"`
	ServiceName  string            `yaml:"service_name" mapstructure:"service_name"`
	Insecure     bool              `yaml:"insecure" mapstructure:"insecure"`
	Headers      map[string]string `yaml:"headers" mapstructure:"headers"`
	SampleRatio  float64           `yaml:"sample_ratio" mapstructure:"sample_ratio"`
	ResourceTags map[string]string `yaml:"resource_tags" mapstructure:"resource_tags"`
}

// MetricsConfig 指标暴露配置
type MetricsConfig struct {
	Addr string `yaml:"addr" mapstructure:"addr"`
}
