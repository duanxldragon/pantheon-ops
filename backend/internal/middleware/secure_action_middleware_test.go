package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/duanxldragon/pantheon-base/backend/pkg/authtoken"
	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
	"github.com/duanxldragon/pantheon-base/backend/pkg/database"
	"github.com/duanxldragon/pantheon-base/backend/pkg/testredis"

	"github.com/gin-gonic/gin"
)

// setupSecureActionRedis wires the test Redis into the global database.RDB
// that SecureActionMiddleware reads, restoring the previous value on cleanup.
func setupSecureActionRedis(t *testing.T) {
	t.Helper()
	rdb := testredis.Open(t)
	previousRDB := database.RDB
	database.RDB = rdb
	t.Cleanup(func() {
		database.RDB = previousRDB
	})
}

func assertSecureActionStatus(t *testing.T, token string, userID uint64, sessionID string, wantStatus int) {
	t.Helper()
	engine := gin.New()
	engine.Use(func(c *gin.Context) {
		c.Set("userId", userID)
		c.Set("sessionId", sessionID)
		c.Next()
	})
	engine.POST("/secure", SecureActionMiddleware(), func(c *gin.Context) {
		common.Success(c, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodPost, "/secure", nil)
	if token != "" {
		req.Header.Set("X-Operation-Token", token)
	}
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, req)
	if recorder.Code != wantStatus {
		t.Fatalf("expected status %d, got %d", wantStatus, recorder.Code)
	}
}

func TestSecureActionMiddlewareRejectsSessionMismatch(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setupSecureActionRedis(t)

	token, err := authtoken.GenerateOperationToken(7, "session-a", "secure_action", authtoken.DefaultAccessTokenTTL, database.RDB)
	if err != nil {
		t.Fatalf("generate operation token: %v", err)
	}

	assertSecureActionStatus(t, token, 7, "session-b", http.StatusForbidden)
}

func TestSecureActionMiddlewareAllowsMatchingSession(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setupSecureActionRedis(t)

	token, err := authtoken.GenerateOperationToken(7, "session-a", "secure_action", authtoken.DefaultAccessTokenTTL, database.RDB)
	if err != nil {
		t.Fatalf("generate operation token: %v", err)
	}

	assertSecureActionStatus(t, token, 7, "session-a", http.StatusOK)
}

func TestSecureActionMiddlewareRejectsMissingToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	assertSecureActionStatus(t, "", 7, "session-a", http.StatusForbidden)
}

func TestSecureActionMiddlewareRejectsUserMismatch(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setupSecureActionRedis(t)

	token, err := authtoken.GenerateOperationToken(7, "session-a", "secure_action", authtoken.DefaultAccessTokenTTL, database.RDB)
	if err != nil {
		t.Fatalf("generate operation token: %v", err)
	}

	assertSecureActionStatus(t, token, 99, "session-a", http.StatusForbidden)
}

func TestSecureActionMiddlewareRejectsWrongScope(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setupSecureActionRedis(t)

	token, err := authtoken.GenerateOperationToken(7, "session-a", "other_scope", authtoken.DefaultAccessTokenTTL, database.RDB)
	if err != nil {
		t.Fatalf("generate operation token: %v", err)
	}

	assertSecureActionStatus(t, token, 7, "session-a", http.StatusForbidden)
}
