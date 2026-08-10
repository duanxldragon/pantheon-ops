package middleware

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestRateLimiter_ScopedToPublicRoutesOnly(t *testing.T) {
	gin.SetMode(gin.TestMode)

	limiter := RateLimiter(RateLimiterConfig{
		MaxRequests: 1,
		Window:      time.Minute,
	})

	engine := gin.New()

	publicAuth := engine.Group("/api/v1/auth").Use(limiter)
	publicAuth.POST("/login", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	engine.GET("/api/v1/system/dept/tree", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	firstLogin := httptest.NewRecorder()
	engine.ServeHTTP(firstLogin, httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", nil))
	if firstLogin.Code != http.StatusNoContent {
		t.Fatalf("expected first login request to pass, got %d", firstLogin.Code)
	}

	secondLogin := httptest.NewRecorder()
	engine.ServeHTTP(secondLogin, httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", nil))
	if secondLogin.Code != http.StatusTooManyRequests {
		t.Fatalf("expected second login request to be rate limited, got %d", secondLogin.Code)
	}

	protectedPageData := httptest.NewRecorder()
	engine.ServeHTTP(protectedPageData, httptest.NewRequest(http.MethodGet, "/api/v1/system/dept/tree", nil))
	if protectedPageData.Code != http.StatusNoContent {
		t.Fatalf("expected protected page data request to bypass public auth limiter, got %d", protectedPageData.Code)
	}
}

func TestMemoryRateLimitStore_AllowAndEvict(t *testing.T) {
	store := newMemoryRateLimitStore()
	window := time.Minute
	now := time.Now()

	for i := 0; i < 2; i++ {
		allowed, err := store.Allow(t.Context(), "key-a", 2, window)
		if err != nil || !allowed {
			t.Fatalf("expected request %d allowed, got allowed=%v err=%v", i+1, allowed, err)
		}
	}
	allowed, err := store.Allow(t.Context(), "key-a", 2, window)
	if err != nil || allowed {
		t.Fatalf("expected third request rate-limited, got allowed=%v err=%v", allowed, err)
	}

	// 淘汰：塞入超阈值的过期条目后，新写入触发清理
	for i := 0; i < 10001; i++ {
		store.entries[fmt.Sprintf("stale-%d", i)] = &rateLimiterEntry{count: 1, lastSeen: now.Add(-2 * window)}
	}
	if _, err := store.Allow(t.Context(), "fresh-key", 2, window); err != nil {
		t.Fatalf("allow fresh key: %v", err)
	}
	if len(store.entries) > 100 {
		t.Fatalf("expected stale entries evicted, got %d entries", len(store.entries))
	}
}
