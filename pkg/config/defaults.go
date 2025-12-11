package config

// ==================== SlotConfig 默认值 ====================

// ApplyDefaults 应用 Slot 配置默认值
func (s *SlotConfig) ApplyDefaults() {
	if s.TotalSlots <= 0 {
		s.TotalSlots = 512
	}
	if s.RedisPrefix == "" {
		s.RedisPrefix = "gga"
	}
	if s.LeaseTTLSeconds <= 0 {
		s.LeaseTTLSeconds = 120
	}
	if s.RouteTTLSeconds <= 0 {
		s.RouteTTLSeconds = 60
	}
	if s.RenewIntervalSeconds <= 0 {
		s.RenewIntervalSeconds = 30
	}
	if s.MaxRetry <= 0 {
		s.MaxRetry = 3
	}
}

// ==================== MetricsConfig 默认值 ====================

// ApplyDefaults 应用 Metrics 配置默认值
func (m *MetricsConfig) ApplyDefaults() {
	if m.Addr == "" {
		m.Addr = ":9090"
	}
}

// ==================== BridgeClientConfig 默认值 ====================

// ApplyDefaults 应用 Bridge 客户端配置默认值
func (b *BridgeClientConfig) ApplyDefaults() {
	if b.DialTimeoutSeconds <= 0 {
		b.DialTimeoutSeconds = 5
	}
	if b.HeartbeatIntervalSeconds <= 0 {
		b.HeartbeatIntervalSeconds = 15
	}
	if b.ReconnectBaseSeconds <= 0 {
		b.ReconnectBaseSeconds = 1
	}
	if b.ReconnectMaxSeconds <= 0 {
		b.ReconnectMaxSeconds = 15
	}
	if b.PendingAckTimeoutSeconds <= 0 {
		b.PendingAckTimeoutSeconds = 15
	}
}

// ==================== BridgeServerConfig 默认值 ====================

// ApplyDefaults 应用 Bridge 服务端配置默认值
func (b *BridgeServerConfig) ApplyDefaults() {
	if b.DeliverBuffer <= 0 {
		b.DeliverBuffer = 128
	}
	if b.HeartbeatIntervalSeconds <= 0 {
		b.HeartbeatIntervalSeconds = 15
	}
	if b.ReconnectInitialSeconds <= 0 {
		b.ReconnectInitialSeconds = 1
	}
	if b.ReconnectMaxSeconds <= 0 {
		b.ReconnectMaxSeconds = 30
	}
	if b.PendingAckTimeoutSeconds <= 0 {
		b.PendingAckTimeoutSeconds = 15
	}
}

// ==================== TracingConfig 默认值 ====================

// ApplyDefaults 应用 Tracing 配置默认值
func (t *TracingConfig) ApplyDefaults() {
	if t.Exporter == "" {
		t.Exporter = "stdout"
	}
	if t.SampleRatio <= 0 {
		t.SampleRatio = 1.0
	}
}

// ==================== PostgresConfig 默认值 ====================

// ApplyDefaults 应用 Postgres 配置默认值
func (p *PostgresConfig) ApplyDefaults() {
	if p.MaxOpenConns <= 0 {
		p.MaxOpenConns = 10
	}
	if p.MaxIdleConns <= 0 {
		p.MaxIdleConns = 5
	}
	if p.ConnMaxLifetimeSeconds <= 0 {
		p.ConnMaxLifetimeSeconds = 3600
	}
}
