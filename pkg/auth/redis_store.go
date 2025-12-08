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
