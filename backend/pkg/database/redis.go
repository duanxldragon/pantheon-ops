package database

import (
	"context"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
	"pantheon-ops/backend/pkg/metrics"
)

var RDB *redis.Client

// InitRedis 初始化 Redis 连接（可选依赖，连接失败不阻止服务启动）
func InitRedis(addr string, password string, db int) {
	RDB = redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password, // 如果没有密码则留空
		DB:       db,       // 默认使用 DB 0
	})

	// 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := RDB.Ping(ctx).Result()
	if err != nil {
		slog.Warn("failed to connect redis (token blacklist will be disabled)", "error", err)
		RDB = nil
		return
	}

	// 启动后台协程采集 Redis 连接池指标
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			stats := RDB.PoolStats()
			metrics.RedisConnectionsActive.Set(float64(stats.TotalConns - stats.IdleConns))
			metrics.RedisConnectionsIdle.Set(float64(stats.IdleConns))
		}
	}()

	slog.Info("Redis connection successful")
}

// SetEx 设置带过期时间的缓存 (对底座后续业务很有用)
func SetEx(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	return RDB.Set(ctx, key, value, expiration).Err()
}

// Get 获取缓存
func Get(ctx context.Context, key string) (string, error) {
	return RDB.Get(ctx, key).Result()
}
