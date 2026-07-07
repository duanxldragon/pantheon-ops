package ratelimit

import (
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/ulule/limiter/v3"
)

func TestIPKeyFunc(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(nil)
	c.Request = &http.Request{}
	c.Request.RemoteAddr = "192.168.1.1:12345"

	key := IPKeyFunc(c)
	expected := "ip:192.168.1.1"
	if key != expected {
		t.Errorf("IPKeyFunc() = %v, want %v", key, expected)
	}
}

func TestUserKeyFunc(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("with user ID", func(t *testing.T) {
		c, _ := gin.CreateTestContext(nil)
		c.Set("userID", uint64(123))
		c.Request = &http.Request{}
		c.Request.RemoteAddr = "192.168.1.1:12345"

		key := UserKeyFunc(c)
		expected := "user:123"
		if key != expected {
			t.Errorf("UserKeyFunc() = %v, want %v", key, expected)
		}
	})

	t.Run("without user ID falls back to IP", func(t *testing.T) {
		c, _ := gin.CreateTestContext(nil)
		c.Request = &http.Request{}
		c.Request.RemoteAddr = "192.168.1.1:12345"

		key := UserKeyFunc(c)
		expected := "ip:192.168.1.1"
		if key != expected {
			t.Errorf("UserKeyFunc() = %v, want %v", key, expected)
		}
	})
}

func TestPathIPKeyFunc(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(nil)
	c.Request = &http.Request{}
	c.Request.RemoteAddr = "192.168.1.1:12345"

	key := PathIPKeyFunc(c)
	// FullPath() 在测试中返回空字符串
	expected := "path::ip:192.168.1.1"
	if key != expected {
		t.Errorf("PathIPKeyFunc() = %v, want %v", key, expected)
	}
}

func TestNewRateLimitMiddlewareWithNilRedis(t *testing.T) {
	// 测试 Redis 为 nil 时返回空中间件
	middleware := NewRateLimitMiddleware(nil, RateLimitConfig{
		Rate: limiter.Rate{Period: 60, Limit: 10},
	})

	if middleware == nil {
		t.Error("NewRateLimitMiddleware() returned nil")
	}

	// 空中间件应该正常执行
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(nil)
	middleware(c)
}

func TestRateLimitConfig(t *testing.T) {
	config := RateLimitConfig{
		Rate: limiter.Rate{
			Period: 60,
			Limit:  10,
		},
		KeyFunc: IPKeyFunc,
	}

	if config.Rate.Period != 60 {
		t.Errorf("Rate.Period = %v, want 60", config.Rate.Period)
	}

	if config.Rate.Limit != 10 {
		t.Errorf("Rate.Limit = %v, want 10", config.Rate.Limit)
	}

	if config.KeyFunc == nil {
		t.Error("KeyFunc is nil")
	}
}
