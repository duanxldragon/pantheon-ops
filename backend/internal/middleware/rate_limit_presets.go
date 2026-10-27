package middleware

import (
	"time"

	"github.com/duanxldragon/pantheon-base/backend/pkg/ratelimit"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/ulule/limiter/v3"
)

// LoginRateLimitMiddleware 登录端点速率限制：每IP每分钟最多5次
func LoginRateLimitMiddleware(rdb *redis.Client) gin.HandlerFunc {
	return ratelimit.NewRateLimitMiddleware(rdb, ratelimit.RateLimitConfig{
		Rate: limiter.Rate{
			Period: time.Minute,
			Limit:  5,
		},
		KeyFunc: ratelimit.PathIPKeyFunc,
	})
}

// RefreshTokenRateLimitMiddleware Token刷新速率限制：每用户每分钟最多10次
func RefreshTokenRateLimitMiddleware(rdb *redis.Client) gin.HandlerFunc {
	return ratelimit.NewRateLimitMiddleware(rdb, ratelimit.RateLimitConfig{
		Rate: limiter.Rate{
			Period: time.Minute,
			Limit:  10,
		},
		KeyFunc: ratelimit.UserKeyFunc,
	})
}

// GeneralAPIRateLimitMiddleware 一般API速率限制：每用户每秒最多50次。
// 注意：底座未注册本预设（全局限流走 cmd/server 注册的 RateLimiter，
// 按 IP 计数——组级中间件先于 TokenAuthMiddleware 执行，取不到 userId）。
// 保留导出供下游仓库在路由级(认证后)按用户限流使用。
func GeneralAPIRateLimitMiddleware(rdb *redis.Client) gin.HandlerFunc {
	return ratelimit.NewRateLimitMiddleware(rdb, ratelimit.RateLimitConfig{
		Rate: limiter.Rate{
			Period: time.Second,
			Limit:  50,
		},
		KeyFunc: ratelimit.UserKeyFunc,
	})
}
