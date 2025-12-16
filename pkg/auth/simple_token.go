package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// SimpleTokenConfig 简化版 Token 配置
// 只需要一个 Access Token，通过版本号控制失效
type SimpleTokenConfig struct {
	Secret    string        // JWT 签名密钥
	TTL       time.Duration // Token 有效期
	ClockSkew time.Duration // 时钟偏差容忍
}

// Defaults 填充默认值
func (c *SimpleTokenConfig) Defaults() {
	if c.TTL <= 0 {
		c.TTL = 7 * 24 * time.Hour // 默认7天
	}
	if c.ClockSkew < 0 {
		c.ClockSkew = 0
	}
}

// SimpleTokenClaims 简化版 Token Claims
type SimpleTokenClaims struct {
	UserID   int64  `json:"uid"`
	Sequence string `json:"seq"`
	Version  int64  `json:"ver"` // 版本号，必须等于 Redis 中的版本
	jwt.RegisteredClaims
}

// SimpleTokenResult 生成 Token 的返回结果
type SimpleTokenResult struct {
	Token     string    // Token 字符串
	ExpiresAt time.Time // 过期时间
	ExpiresIn int64     // 剩余秒数
	Version   int64     // Token 版本号
}

// TokenVersionStore Token 版本存储接口
// Redis 中每用户仅存储一个 key: auth:token:ver:{user_id}
type TokenVersionStore interface {
	// IncrVersion 递增版本号并返回新版本（登录时调用）
	IncrVersion(ctx context.Context, userID int64) (int64, error)
	// GetVersion 获取当前版本号（验证时调用）
	GetVersion(ctx context.Context, userID int64) (int64, error)
}

// GenerateSimpleToken 生成简化版 Access Token
// 登录时调用，自动递增版本号，旧 Token 立即失效
func GenerateSimpleToken(ctx context.Context, userID int64, sequence string, cfg SimpleTokenConfig, store TokenVersionStore) (*SimpleTokenResult, error) {
	cfg.Defaults()
	if cfg.Secret == "" {
		return nil, errors.New("jwt secret is empty")
	}
	if store == nil {
		return nil, errors.New("token version store is required")
	}

	// 递增版本号，使旧 token 失效
	version, err := store.IncrVersion(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to increment token version: %w", err)
	}

	now := time.Now()
	expiresAt := now.Add(cfg.TTL)

	claims := SimpleTokenClaims{
		UserID:   userID,
		Sequence: sequence,
		Version:  version,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   fmt.Sprintf("%d", userID),
			ID:        uuid.NewString(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString([]byte(cfg.Secret))
	if err != nil {
		return nil, fmt.Errorf("failed to sign token: %w", err)
	}

	return &SimpleTokenResult{
		Token:     tokenStr,
		ExpiresAt: expiresAt,
		ExpiresIn: int64(cfg.TTL.Seconds()),
		Version:   version,
	}, nil
}

// VerifySimpleToken 验证简化版 Access Token
// 验证签名、过期时间、以及版本号是否与 Redis 中一致
func VerifySimpleToken(ctx context.Context, tokenStr string, cfg SimpleTokenConfig, store TokenVersionStore) (*SimpleTokenClaims, error) {
	cfg.Defaults()
	if cfg.Secret == "" {
		return nil, errors.New("jwt secret is empty")
	}
	if store == nil {
		return nil, errors.New("token version store is required")
	}

	claims := &SimpleTokenClaims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(cfg.Secret), nil
	}, jwt.WithLeeway(cfg.ClockSkew))

	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, errors.New("invalid token")
	}

	// 验证版本号
	currentVersion, err := store.GetVersion(ctx, claims.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to get token version: %w", err)
	}

	if claims.Version != currentVersion {
		return nil, errors.New("token invalidated (logged in elsewhere)")
	}

	return claims, nil
}

// RevokeSimpleToken 强制使用户的所有 Token 失效
// 只需递增版本号即可，无需存储任何 blocklist
func RevokeSimpleToken(ctx context.Context, userID int64, store TokenVersionStore) error {
	if store == nil {
		return errors.New("token version store is required")
	}
	_, err := store.IncrVersion(ctx, userID)
	return err
}

// ParseSimpleTokenUnverified 解析 Token 但不验证（用于调试或获取 claims）
func ParseSimpleTokenUnverified(tokenStr string) (*SimpleTokenClaims, error) {
	claims := &SimpleTokenClaims{}
	parser := jwt.NewParser()
	_, _, err := parser.ParseUnverified(tokenStr, claims)
	if err != nil {
		return nil, err
	}
	return claims, nil
}
