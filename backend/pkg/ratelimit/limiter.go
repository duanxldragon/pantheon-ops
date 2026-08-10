package ratelimit

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/ulule/limiter/v3"
	limitergin "github.com/ulule/limiter/v3/drivers/middleware/gin"
	limiterredis "github.com/ulule/limiter/v3/drivers/store/redis"
)

// RateLimitConfig 速率限制配置
type RateLimitConfig struct {
	Rate    limiter.Rate              // 例如: limiter.Rate{Period: time.Minute, Limit: 10}
	KeyFunc func(*gin.Context) string // 获取限流键的函数
}

// NewRateLimitMiddleware 创建速率限制中间件
func NewRateLimitMiddleware(rdb *redis.Client, config RateLimitConfig) gin.HandlerFunc {
	// 如果 Redis 不可用，返回空中间件（不限流）
	if rdb == nil {
		return func(c *gin.Context) {
			c.Next()
		}
	}

	store, err := limiterredis.NewStoreWithOptions(rdb, limiter.StoreOptions{
		Prefix:   "ratelimit",
		MaxRetry: 3,
	})
	if err != nil {
		panic(fmt.Sprintf("Failed to create rate limiter store: %v", err))
	}

	rate := config.Rate
	instance := limiter.New(store, rate)

	middleware := limitergin.NewMiddleware(instance, limitergin.WithKeyGetter(func(c *gin.Context) string {
		if config.KeyFunc != nil {
			return config.KeyFunc(c)
		}
		// 默认使用 IP
		return c.ClientIP()
	}))

	return func(c *gin.Context) {
		middleware(c)
		if c.IsAborted() {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"code":    429,
				"message": "ratelimit.exceeded",
				"data":    nil,
			})
		}
	}
}

// IPKeyFunc 按 IP 限流
func IPKeyFunc(c *gin.Context) string {
	return fmt.Sprintf("ip:%s", c.ClientIP())
}

// UserKeyFunc 按用户限流
func UserKeyFunc(c *gin.Context) string {
	userID, exists := c.Get("userID")
	if !exists {
		return IPKeyFunc(c) // 未登录用户按 IP 限流
	}
	return fmt.Sprintf("user:%v", userID)
}

// PathIPKeyFunc 按路径+IP限流
func PathIPKeyFunc(c *gin.Context) string {
	return fmt.Sprintf("path:%s:ip:%s", c.FullPath(), c.ClientIP())
}
