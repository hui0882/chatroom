package redis

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"

	"github.com/hui0882/chatroom/pkg/config"
)

// Init 初始化 Redis 客户端并验证连通性
func Init(cfg *config.RedisConfig) (*redis.Client, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
		PoolSize: cfg.PoolSize,
	})

	if err := rdb.Ping(context.Background()).Err(); err != nil {
		return nil, fmt.Errorf("redis ping: %w", err)
	}

	return rdb, nil
}
