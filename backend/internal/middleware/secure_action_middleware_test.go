package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"pantheon-platform/backend/pkg/common"

	"github.com/gin-gonic/gin"
)

func TestSecureActionMiddlewareRejectsSessionMismatch(t *testing.T) {
	gin.SetMode(gin.TestMode)

	token, err := common.GenerateOperationToken(7, "session-a", "secure_action", time.Minute)
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

	token, err := common.GenerateOperationToken(7, "session-a", "secure_action", time.Minute)
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
