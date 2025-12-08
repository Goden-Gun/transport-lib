package auth

import "time"

const (
	// DefaultAccessBlocklistPrefix is the Redis key prefix for revoked access JTIs.
	DefaultAccessBlocklistPrefix = "auth:access:block:"
	// DefaultRefreshStorePrefix is the Redis key prefix for refresh token JTIs.
	DefaultRefreshStorePrefix = "auth:refresh:"
)

// Config controls JWT signing and validation.
// Secret: shared HS key; Alg currently supports HS256.
// Blocklist/Refresh prefixes are used by Redis implementations.
type Config struct {
	Secret             string
	Alg                string
	AccessTTL          time.Duration
	RefreshTTL         time.Duration
	ClockSkew          time.Duration
	BlocklistPrefix    string
	RefreshStorePrefix string
}

// Defaults fills zero values.
func (c *Config) Defaults() {
	if c.Alg == "" {
		c.Alg = "HS256"
	}
	if c.AccessTTL <= 0 {
		c.AccessTTL = 30 * time.Minute
	}
	if c.RefreshTTL <= 0 {
		c.RefreshTTL = 72 * time.Hour
	}
	if c.ClockSkew < 0 {
		c.ClockSkew = 0
	}
	if c.BlocklistPrefix == "" {
		c.BlocklistPrefix = DefaultAccessBlocklistPrefix
	}
	if c.RefreshStorePrefix == "" {
		c.RefreshStorePrefix = DefaultRefreshStorePrefix
	}
}
