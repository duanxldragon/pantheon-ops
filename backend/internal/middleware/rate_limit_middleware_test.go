package middleware

import (
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
