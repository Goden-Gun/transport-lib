package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisRefreshStore stores refresh tokens as one-time keys.
type RedisRefreshStore struct {
	client redis.Cmdable
	prefix string
}

func NewRedisRefreshStore(client redis.Cmdable, prefix string) *RedisRefreshStore {
	if client == nil {
		return nil
	}
	if prefix == "" {
		prefix = DefaultRefreshStorePrefix
	}
	return &RedisRefreshStore{client: client, prefix: prefix}
}

func (s *RedisRefreshStore) Save(ctx context.Context, jti string, meta RefreshMetadata, ttl time.Duration) error {
	if s == nil || jti == "" {
		return fmt.Errorf("refresh store not configured")
	}
	data, err := json.Marshal(meta)
	if err != nil {
		return err
	}
	return s.client.Set(ctx, s.key(jti), data, ttl).Err()
}

func (s *RedisRefreshStore) Consume(ctx context.Context, jti string) (*RefreshMetadata, error) {
	if s == nil || jti == "" {
		return nil, fmt.Errorf("refresh store not configured")
	}
	key := s.key(jti)
	val, err := s.client.GetDel(ctx, key).Result()
	if err == redis.Nil {
		return nil, fmt.Errorf("refresh token not found")
	}
	if err != nil {
		return nil, err
	}
	var meta RefreshMetadata
	if err := json.Unmarshal([]byte(val), &meta); err != nil {
		return nil, err
	}
	return &meta, nil
}

func (s *RedisRefreshStore) key(jti string) string {
	return s.prefix + jti
}

// RedisAccessBlocklist stores revoked access JTI with TTL.
type RedisAccessBlocklist struct {
	client redis.Cmdable
	prefix string
}

func NewRedisAccessBlocklist(client redis.Cmdable, prefix string) *RedisAccessBlocklist {
	if client == nil {
		return nil
	}
	if prefix == "" {
		prefix = DefaultAccessBlocklistPrefix
	}
	return &RedisAccessBlocklist{client: client, prefix: prefix}
}

func (b *RedisAccessBlocklist) Block(ctx context.Context, jti string, ttl time.Duration) error {
	if b == nil || jti == "" {
		return fmt.Errorf("blocklist not configured")
	}
	return b.client.Set(ctx, b.key(jti), "1", ttl).Err()
}

func (b *RedisAccessBlocklist) IsBlocked(ctx context.Context, jti string) (bool, error) {
	if b == nil || jti == "" {
		return false, nil
	}
	res, err := b.client.Exists(ctx, b.key(jti)).Result()
	if err != nil {
		return false, err
	}
	return res > 0, nil
}

func (b *RedisAccessBlocklist) key(jti string) string {
	return b.prefix + jti
}

// SessionVersionStore keeps track of latest session version per user.
type SessionVersionStore interface {
	Get(ctx context.Context, userID int64) (int64, error)
	Incr(ctx context.Context, userID int64) (int64, error)
}

type RedisSessionVersionStore struct {
	client redis.Cmdable
	prefix string
}

func NewRedisSessionVersionStore(client redis.Cmdable, prefix string) *RedisSessionVersionStore {
	if client == nil {
		return nil
	}
	if prefix == "" {
		prefix = DefaultSessionVersionPrefix
	}
	return &RedisSessionVersionStore{client: client, prefix: prefix}
}

func (s *RedisSessionVersionStore) Get(ctx context.Context, userID int64) (int64, error) {
	if s == nil {
		return 0, fmt.Errorf("session version store not configured")
	}
	return s.client.Get(ctx, s.key(userID)).Int64()
}

func (s *RedisSessionVersionStore) Incr(ctx context.Context, userID int64) (int64, error) {
	if s == nil {
		return 0, fmt.Errorf("session version store not configured")
	}
	return s.client.Incr(ctx, s.key(userID)).Result()
}

func (s *RedisSessionVersionStore) key(userID int64) string {
	return fmt.Sprintf("%s%d", s.prefix, userID)
}

// RedisTokenVersionStore implements TokenVersionStore using Redis.
// Each user has only ONE key, minimizing Redis storage.
type RedisTokenVersionStore struct {
	client redis.Cmdable
	prefix string
	ttl    time.Duration // optional TTL for version keys
}

// RedisTokenVersionStoreOption configures RedisTokenVersionStore
type RedisTokenVersionStoreOption func(*RedisTokenVersionStore)

// WithTokenVersionTTL sets TTL for version keys (cleanup inactive users)
func WithTokenVersionTTL(ttl time.Duration) RedisTokenVersionStoreOption {
	return func(s *RedisTokenVersionStore) {
		s.ttl = ttl
	}
}

// NewRedisTokenVersionStore creates a new TokenVersionStore backed by Redis.
func NewRedisTokenVersionStore(client redis.Cmdable, opts ...RedisTokenVersionStoreOption) *RedisTokenVersionStore {
	if client == nil {
		return nil
	}
	s := &RedisTokenVersionStore{
		client: client,
		prefix: DefaultTokenVersionPrefix,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// IncrVersion increments the token version for a user (called on login).
// This automatically invalidates all previous tokens for this user.
func (s *RedisTokenVersionStore) IncrVersion(ctx context.Context, userID int64) (int64, error) {
	if s == nil {
		return 0, fmt.Errorf("token version store not configured")
	}
	key := s.key(userID)
	ver, err := s.client.Incr(ctx, key).Result()
	if err != nil {
		return 0, err
	}
	// Set TTL if configured (refreshes on each login)
	if s.ttl > 0 {
		s.client.Expire(ctx, key, s.ttl)
	}
	return ver, nil
}

// GetVersion returns the current token version for a user.
// Returns 0 if the user has never logged in.
func (s *RedisTokenVersionStore) GetVersion(ctx context.Context, userID int64) (int64, error) {
	if s == nil {
		return 0, fmt.Errorf("token version store not configured")
	}
	ver, err := s.client.Get(ctx, s.key(userID)).Int64()
	if err == redis.Nil {
		return 0, nil // user never logged in, version is 0
	}
	return ver, err
}

func (s *RedisTokenVersionStore) key(userID int64) string {
	return fmt.Sprintf("%s%d", s.prefix, userID)
}
