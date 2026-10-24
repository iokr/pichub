package bootstrap

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/pichub/pichub/internal/config"
	"github.com/pichub/pichub/internal/logger"
)

// NewRedis 创建 Redis 客户端并检查连通性。
func NewRedis(cfg *config.Config, log *logger.Logger) (*redis.Client, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:         cfg.Redis.Addr,
		Password:     cfg.Redis.Password,
		DB:           cfg.Redis.DB,
		PoolSize:     cfg.Redis.PoolSize,
		MinIdleConns: cfg.Redis.MinIdleConns,
		DialTimeout:  cfg.Redis.DialTimeout,
		ReadTimeout:  cfg.Redis.ReadTimeout,
		WriteTimeout: cfg.Redis.WriteTimeout,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err != nil {
		_ = rdb.Close()
		return nil, fmt.Errorf("connect redis %s: %w", cfg.Redis.Addr, err)
	}
	if log != nil {
		log.Info("redis connected", logger.String("addr", cfg.Redis.Addr), logger.Int("db", cfg.Redis.DB))
	}
	return rdb, nil
}
