package database

import (
	"context"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/duanxldragon/pantheon-base/backend/pkg/metrics"
	"github.com/redis/go-redis/v9"
)

var RDB *redis.Client

// RequireRedis reports whether Redis is a mandatory startup dependency:
// always in production, or explicitly via PANTHEON_REDIS_REQUIRED=true (the
// dev/test opt-out that keeps non-production startup compatible when Redis
// is genuinely absent).
func RequireRedis() bool {
	if os.Getenv("PANTHEON_ENV") == "production" {
		return true
	}
	return strings.EqualFold(strings.TrimSpace(os.Getenv("PANTHEON_REDIS_REQUIRED")), "true")
}

// InitRedis 初始化 Redis 连接。Redis 承载令牌会话存储与吊销黑名单——缺失时
// 所有认证请求都会失败（事实上的硬依赖）。连接失败时：生产环境（或
// PANTHEON_REDIS_REQUIRED=true）fail-fast 退出；否则保持兼容行为降级为 nil。
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
		slog.Warn("failed to connect redis (authenticated requests will fail: Redis stores token sessions and the revocation blacklist)", "error", err)

		// Fail-fast: Redis is a hard dependency for token sessions in
		// production and for any deployment that opts in explicitly.
		if RequireRedis() {
			slog.Error("Redis connection is required (production mode or PANTHEON_REDIS_REQUIRED=true)", "error", err)
			os.Exit(1)
		}

		RDB = nil
		return
	}

	// 启动后台协程采集 Redis 连接池指标
	go func() {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("redis metrics goroutine panic", "panic", r)
			}
		}()
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
