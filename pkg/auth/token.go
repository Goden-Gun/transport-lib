package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// AccessClaims represents access token claims.
type AccessClaims struct {
	UserID   int64  `json:"user_id"`
	Sequence string `json:"sequence"`
	// SessionVersion is a monotonic number; tokens with lower versions are invalid once a higher version is issued.
	SessionVersion int64  `json:"session_version,omitempty"`
	TokenType      string `json:"type"`
	jwt.RegisteredClaims
}

// RefreshClaims represents refresh token claims.
type RefreshClaims struct {
	UserID    int64  `json:"user_id"`
	Sequence  string `json:"sequence"`
	TokenType string `json:"type"`
	jwt.RegisteredClaims
}

// TokenPair bundles access/refresh tokens and expiry metadata.
type TokenPair struct {
	AccessToken              string
	RefreshToken             string
	AccessTokenExpiresAt     time.Time
	RefreshTokenExpiresAt    time.Time
	AccessTokenExpiresInSec  int64
	RefreshTokenExpiresInSec int64
	AccessTokenJTI           string
	RefreshTokenJTI          string
}

// RefreshTokenStore abstracts refresh token persistence (one-time use).
type RefreshTokenStore interface {
	Save(ctx context.Context, jti string, meta RefreshMetadata, ttl time.Duration) error
	Consume(ctx context.Context, jti string) (*RefreshMetadata, error)
}

// RefreshMetadata stored alongside a refresh token.
type RefreshMetadata struct {
	UserID   int64  `json:"user_id"`
	Sequence string `json:"sequence"`
}

// AccessTokenBlocklist abstracts revoked access tokens.
type AccessTokenBlocklist interface {
	Block(ctx context.Context, jti string, ttl time.Duration) error
	IsBlocked(ctx context.Context, jti string) (bool, error)
}

// GenerateTokenPair issues access + refresh tokens and stores refresh JTI if store provided.
// Deprecated: prefer GenerateTokenPairWithVersion to include session versioning.
func GenerateTokenPair(ctx context.Context, userID int64, sequence string, cfg Config, store RefreshTokenStore) (*TokenPair, error) {
	return GenerateTokenPairWithVersion(ctx, userID, sequence, 0, cfg, store)
}

// GenerateTokenPairWithVersion issues tokens with explicit sessionVersion.
func GenerateTokenPairWithVersion(ctx context.Context, userID int64, sequence string, sessionVersion int64, cfg Config, store RefreshTokenStore) (*TokenPair, error) {
	cfg.Defaults()
	if cfg.Secret == "" {
		return nil, errors.New("jwt secret is empty")
	}
	now := time.Now()
	accessJTI := uuid.NewString()
	refreshJTI := uuid.NewString()

	accessClaims := AccessClaims{
		UserID:         userID,
		Sequence:       sequence,
		SessionVersion: sessionVersion,
		TokenType:      "access",
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   stringFromInt64(userID),
			ID:        accessJTI,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(cfg.AccessTTL)),
		},
	}
	refreshClaims := RefreshClaims{
		UserID:    userID,
		Sequence:  sequence,
		TokenType: "refresh",
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   stringFromInt64(userID),
			ID:        refreshJTI,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(cfg.RefreshTTL)),
		},
	}
	accessToken, err := signClaims(accessClaims, cfg)
	if err != nil {
		return nil, err
	}
	refreshToken, err := signClaims(refreshClaims, cfg)
	if err != nil {
		return nil, err
	}
	if store != nil {
		meta := RefreshMetadata{UserID: userID, Sequence: sequence}
		if err := store.Save(ctx, refreshJTI, meta, cfg.RefreshTTL); err != nil {
			return nil, err
		}
	}
	return &TokenPair{
		AccessToken:              accessToken,
		RefreshToken:             refreshToken,
		AccessTokenExpiresAt:     accessClaims.ExpiresAt.Time,
		RefreshTokenExpiresAt:    refreshClaims.ExpiresAt.Time,
		AccessTokenExpiresInSec:  int64(cfg.AccessTTL.Seconds()),
		RefreshTokenExpiresInSec: int64(cfg.RefreshTTL.Seconds()),
		AccessTokenJTI:           accessJTI,
		RefreshTokenJTI:          refreshJTI,
	}, nil
}

// VerifyAccessToken parses, validates signature/type/exp, and checks blocklist.
func VerifyAccessToken(tokenStr string, cfg Config, blocklist AccessTokenBlocklist) (*AccessClaims, error) {
	cfg.Defaults()
	if cfg.Secret == "" {
		return nil, errors.New("jwt secret is empty")
	}
	claims := &AccessClaims{}
	parsed, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(cfg.Secret), nil
	}, jwt.WithLeeway(cfg.ClockSkew))
	if err != nil {
		return nil, err
	}
	if !parsed.Valid {
		return nil, errors.New("invalid token")
	}
	if claims.TokenType != "access" {
		return nil, errors.New("invalid token type")
	}
	if blocklist != nil && claims.ID != "" {
		blocked, err := blocklist.IsBlocked(context.Background(), claims.ID)
		if err != nil {
			return nil, err
		}
		if blocked {
			return nil, errors.New("token revoked")
		}
	}
	return claims, nil
}

// VerifyAccessTokenWithVersion additionally enforces session version equality when store is provided.
func VerifyAccessTokenWithVersion(tokenStr string, cfg Config, blocklist AccessTokenBlocklist, versionStore SessionVersionStore) (*AccessClaims, error) {
	claims, err := VerifyAccessToken(tokenStr, cfg, blocklist)
	if err != nil {
		return nil, err
	}
	if versionStore != nil {
		current, err := versionStore.Get(context.Background(), claims.UserID)
		if err != nil && !strings.Contains(err.Error(), "nil") { // tolerate missing key as version 0
			return nil, err
		}
		if current > 0 && claims.SessionVersion != current {
			return nil, errors.New("access token superseded")
		}
	}
	return claims, nil
}

// ConsumeRefreshToken verifies and consumes a refresh token (one-time use).
func ConsumeRefreshToken(ctx context.Context, tokenStr string, cfg Config, store RefreshTokenStore) (*RefreshClaims, error) {
	cfg.Defaults()
	if cfg.Secret == "" {
		return nil, errors.New("jwt secret is empty")
	}
	claims := &RefreshClaims{}
	parsed, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (any, error) {
		return []byte(cfg.Secret), nil
	}, jwt.WithLeeway(cfg.ClockSkew))
	if err != nil {
		return nil, err
	}
	if !parsed.Valid {
		return nil, errors.New("invalid token")
	}
	if claims.TokenType != "refresh" {
		return nil, errors.New("invalid token type")
	}
	if claims.ID == "" {
		return nil, errors.New("missing refresh jti")
	}
	if store == nil {
		return nil, errors.New("refresh store not configured")
	}
	meta, err := store.Consume(ctx, claims.ID)
	if err != nil {
		return nil, err
	}
	// Optional sequence/user check
	if meta != nil {
		if meta.UserID != 0 && meta.UserID != claims.UserID {
			return nil, errors.New("refresh token user mismatch")
		}
		if meta.Sequence != "" && meta.Sequence != claims.Sequence {
			return nil, errors.New("refresh token sequence mismatch")
		}
	}
	return claims, nil
}

// RevokeAccessToken writes the access JTI to blocklist with remaining TTL.
func RevokeAccessToken(ctx context.Context, claims *AccessClaims, cfg Config, blocklist AccessTokenBlocklist) error {
	if claims == nil || claims.ID == "" {
		return errors.New("missing access claims")
	}
	if blocklist == nil {
		return errors.New("blocklist not configured")
	}
	ttl := cfg.AccessTTL
	if claims.ExpiresAt != nil {
		ttl = time.Until(claims.ExpiresAt.Time)
	}
	if ttl <= 0 {
		ttl = time.Second
	}
	return blocklist.Block(ctx, claims.ID, ttl)
}

func signClaims(claims jwt.Claims, cfg Config) (string, error) {
	method := jwt.SigningMethodHS256
	token := jwt.NewWithClaims(method, claims)
	return token.SignedString([]byte(cfg.Secret))
}

func stringFromInt64(v int64) string {
	return fmt.Sprintf("%d", v)
}
