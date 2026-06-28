package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/ulule/limiter/v3"
	"pantheon-ops/backend/pkg/ratelimit"
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

// GeneralAPIRateLimitMiddleware 一般API速率限制：每用户每秒最多50次
func GeneralAPIRateLimitMiddleware(rdb *redis.Client) gin.HandlerFunc {
	return ratelimit.NewRateLimitMiddleware(rdb, ratelimit.RateLimitConfig{
		Rate: limiter.Rate{
			Period: time.Second,
			Limit:  50,
		},
		KeyFunc: ratelimit.UserKeyFunc,
	})
}
