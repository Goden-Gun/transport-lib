package bootstrap

import (
	"context"

	"github.com/redis/go-redis/v9"
	log "github.com/sirupsen/logrus"

	"github.com/Goden-Gun/transport-lib/pkg/config"
)

// InitRedis 初始化 Redis 客户端并测试连接
func InitRedis(ctx context.Context, cfg config.RedisConfig) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Username: cfg.Username,
		Password: cfg.Password,
		DB:       cfg.Db,
	})

	if err := client.Ping(ctx).Err(); err != nil {
		log.Errorf("redis初始化失败: %v", err)
		return nil, err
	}

	log.Info("redis initialized successfully")
	return client, nil
}
