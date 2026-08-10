package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"pantheon-ops/backend/pkg/authtoken"
	"pantheon-ops/backend/pkg/common"
	"pantheon-ops/backend/pkg/testredis"

	"github.com/gin-gonic/gin"
)

func TestSecureActionMiddlewareRejectsSessionMismatch(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rdb := testredis.Open(t)

	token, err := authtoken.GenerateOperationToken(7, "session-a", "secure_action", authtoken.DefaultAccessTokenTTL, rdb)
	if err != nil {
		t.Fatalf("generate operation token: %v", err)
	}

	engine := gin.New()
	engine.Use(func(c *gin.Context) {
		c.Set("userId", uint64(7))
		c.Set("sessionId", "session-b")
		c.Next()
	})
	engine.POST("/secure", SecureActionMiddleware(), func(c *gin.Context) {
		common.Success(c, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodPost, "/secure", nil)
	req.Header.Set("X-Operation-Token", token)
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("expected session mismatch to be rejected, got %d", recorder.Code)
	}
}

func TestSecureActionMiddlewareAllowsMatchingSession(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rdb := testredis.Open(t)

	token, err := authtoken.GenerateOperationToken(7, "session-a", "secure_action", authtoken.DefaultAccessTokenTTL, rdb)
	if err != nil {
		t.Fatalf("generate operation token: %v", err)
	}

	engine := gin.New()
	engine.Use(func(c *gin.Context) {
		c.Set("userId", uint64(7))
		c.Set("sessionId", "session-a")
		c.Next()
	})
	engine.POST("/secure", SecureActionMiddleware(), func(c *gin.Context) {
		common.Success(c, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodPost, "/secure", nil)
	req.Header.Set("X-Operation-Token", token)
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected matching session to pass, got %d", recorder.Code)
	}
}

func TestSecureActionMiddlewareRejectsMissingToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	engine := gin.New()
	engine.Use(func(c *gin.Context) {
		c.Set("userId", uint64(7))
		c.Set("sessionId", "session-a")
		c.Next()
	})
	engine.POST("/secure", SecureActionMiddleware(), func(c *gin.Context) {
		common.Success(c, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodPost, "/secure", nil)
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("expected missing token to be rejected, got %d", recorder.Code)
	}
}

func TestSecureActionMiddlewareRejectsUserMismatch(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rdb := testredis.Open(t)

	token, err := authtoken.GenerateOperationToken(7, "session-a", "secure_action", authtoken.DefaultAccessTokenTTL, rdb)
	if err != nil {
		t.Fatalf("generate operation token: %v", err)
	}

	engine := gin.New()
	engine.Use(func(c *gin.Context) {
		c.Set("userId", uint64(99)) // Different user
		c.Set("sessionId", "session-a")
		c.Next()
	})
	engine.POST("/secure", SecureActionMiddleware(), func(c *gin.Context) {
		common.Success(c, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodPost, "/secure", nil)
	req.Header.Set("X-Operation-Token", token)
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("expected user mismatch to be rejected, got %d", recorder.Code)
	}
}

func TestSecureActionMiddlewareRejectsWrongScope(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rdb := testredis.Open(t)

	token, err := authtoken.GenerateOperationToken(7, "session-a", "other_scope", authtoken.DefaultAccessTokenTTL, rdb)
	if err != nil {
		t.Fatalf("generate operation token: %v", err)
	}

	engine := gin.New()
	engine.Use(func(c *gin.Context) {
		c.Set("userId", uint64(7))
		c.Set("sessionId", "session-a")
		c.Next()
	})
	engine.POST("/secure", SecureActionMiddleware(), func(c *gin.Context) {
		common.Success(c, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodPost, "/secure", nil)
	req.Header.Set("X-Operation-Token", token)
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("expected wrong scope to be rejected, got %d", recorder.Code)
	}
}
