package platform

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"pantheon-base/internal/middleware"
	"pantheon-base/pkg/database"
	"pantheon-base/pkg/logging"
	"pantheon-base/pkg/testmysql"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

type healthTestResponse struct {
	Data struct {
		Status       string `json:"status"`
		Dependencies map[string]struct {
			Status  string `json:"status"`
			Message string `json:"message"`
		} `json:"dependencies"`
	} `json:"data"`
}

func decodeHealthTestResponse(t *testing.T, recorder *httptest.ResponseRecorder) healthTestResponse {
	t.Helper()
	var resp healthTestResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal health response: %v", err)
	}
	return resp
}

func TestSanitizeHealthErrorDoesNotExposeDependencyDetails(t *testing.T) {
	raw := errors.New("dial tcp 10.0.0.8:3306: connect: connection refused")
	if got := sanitizeHealthError(raw, "database.unavailable"); got != "database.unavailable" {
		t.Fatalf("expected stable health error key, got %q", got)
	}
	if got := sanitizeHealthError(nil, "database.unavailable"); got != "" {
		t.Fatalf("expected empty message without an error, got %q", got)
	}
}

func TestLogHealthDependencyFailureSamplesAndOmitsRawError(t *testing.T) {
	core, observed := observer.New(zap.ErrorLevel)
	previousLogger := logging.Logger
	logging.Logger = zap.New(core)
	healthFailureLogState.Lock()
	previousLogTimes := healthFailureLogState.lastLogged
	healthFailureLogState.lastLogged = make(map[string]time.Time)
	healthFailureLogState.Unlock()
	t.Cleanup(func() {
		logging.Logger = previousLogger
		healthFailureLogState.Lock()
		healthFailureLogState.lastLogged = previousLogTimes
		healthFailureLogState.Unlock()
	})

	raw := &net.OpError{
		Op:  "dial",
		Net: "tcp",
		Err: errors.New("user:secret@10.0.0.8:3306: connection refused"),
	}
	logHealthDependencyFailure("database", "req-health-log", raw)
	logHealthDependencyFailure("database", "req-health-log-repeat", raw)

	entries := observed.All()
	if len(entries) != 1 {
		t.Fatalf("expected one sampled health failure log, got %d", len(entries))
	}
	context := entries[0].ContextMap()
	if context["requestId"] != "req-health-log" || context["dependency"] != "database" {
		t.Fatalf("unexpected health log context: %+v", context)
	}
	if context["reason"] != "network" {
		t.Fatalf("expected safe network reason, got %+v", context)
	}
	if strings.Contains(entries[0].Message+fmt.Sprint(context), raw.Error()) {
		t.Fatalf("health failure log exposed raw dependency error: %+v", entries[0])
	}
}

func TestClassifyHealthDependencyErrorUsesSafeCategories(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{name: "timeout", err: context.DeadlineExceeded, want: "timeout"},
		{name: "closed connection", err: sql.ErrConnDone, want: "connection_closed"},
		{name: "unusable connection", err: driver.ErrBadConn, want: "connection_unusable"},
		{name: "unknown", err: errors.New("credential=secret"), want: "unavailable"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := classifyHealthDependencyError(tt.err); got != tt.want {
				t.Fatalf("expected %q, got %q", tt.want, got)
			}
		})
	}
}

func TestRegisterHealthRoutes_DatabaseFailureUsesStablePublicMessage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := testmysql.Open(t)
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get sql database: %v", err)
	}
	if err := sqlDB.Close(); err != nil {
		t.Fatalf("close sql database: %v", err)
	}

	engine := gin.New()
	RegisterHealthRoutes(engine.Group("/api/v1"), db)
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/health", nil))

	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("unexpected status code: %d", recorder.Code)
	}
	resp := decodeHealthTestResponse(t, recorder)
	dependency := resp.Data.Dependencies["database"]
	if dependency.Message != "database.unavailable" {
		t.Fatalf("expected stable database message, got %q", dependency.Message)
	}
	if strings.Contains(recorder.Body.String(), "sql: database is closed") {
		t.Fatalf("health response exposed raw database error: %s", recorder.Body.String())
	}
}

func TestRegisterHealthRoutes_RedisFailureUsesStablePublicMessage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	previousRedis := database.RDB
	t.Cleanup(func() { database.RDB = previousRedis })

	client := redis.NewClient(&redis.Options{
		Addr:        "127.0.0.1:0",
		DialTimeout: 50 * time.Millisecond,
		MaxRetries:  -1,
	})
	t.Cleanup(func() { _ = client.Close() })
	database.RDB = client

	engine := gin.New()
	RegisterHealthRoutes(engine.Group("/api/v1"), testmysql.Open(t))
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/health", nil))

	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("unexpected status code: %d", recorder.Code)
	}
	resp := decodeHealthTestResponse(t, recorder)
	dependency := resp.Data.Dependencies["redis"]
	if dependency.Message != "redis.unavailable" {
		t.Fatalf("expected stable redis message, got %q", dependency.Message)
	}
	if strings.Contains(recorder.Body.String(), "127.0.0.1:0") {
		t.Fatalf("health response exposed raw redis endpoint: %s", recorder.Body.String())
	}
}

func TestRegisterHealthRoutes_ReturnsDependencyState(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := testmysql.Open(t)

	engine := gin.New()
	engine.Use(middleware.RequestContextMiddleware())
	api := engine.Group("/api/v1")
	RegisterHealthRoutes(api, db)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	req.Header.Set("X-Request-ID", "req-health-001")
	recorder := httptest.NewRecorder()

	engine.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected status code: %d", recorder.Code)
	}

	var resp struct {
		Code int `json:"code"`
		Data struct {
			Status       string `json:"status"`
			RequestID    string `json:"requestId"`
			Dependencies map[string]struct {
				Status string `json:"status"`
			} `json:"dependencies"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal health response: %v", err)
	}
	if resp.Code != 200 {
		t.Fatalf("expected common success code, got %d", resp.Code)
	}
	if resp.Data.Status != "ok" {
		t.Fatalf("expected ok health, got %s", resp.Data.Status)
	}
	if resp.Data.RequestID != "req-health-001" {
		t.Fatalf("expected request id in response, got %s", resp.Data.RequestID)
	}
	if resp.Data.Dependencies["database"].Status != "ok" {
		t.Fatalf("expected database dependency ok, got %+v", resp.Data.Dependencies["database"])
	}
}

func TestRegisterHealthRoutes_UsesCommonEnvelopeForDegradedState(t *testing.T) {
	gin.SetMode(gin.TestMode)

	engine := gin.New()
	engine.Use(middleware.RequestContextMiddleware())
	api := engine.Group("/api/v1")
	RegisterHealthRoutes(api, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	recorder := httptest.NewRecorder()

	engine.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("unexpected status code: %d", recorder.Code)
	}

	var resp struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    struct {
			Status       string `json:"status"`
			Dependencies map[string]struct {
				Status  string `json:"status"`
				Message string `json:"message"`
			} `json:"dependencies"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal health response: %v", err)
	}
	if resp.Code != 200 || resp.Message != "success" {
		t.Fatalf("expected common success envelope, got code=%d message=%q", resp.Code, resp.Message)
	}
	if resp.Data.Status != "degraded" {
		t.Fatalf("expected degraded health status, got %s", resp.Data.Status)
	}
	if resp.Data.Dependencies["database"].Status != "down" {
		t.Fatalf("expected database dependency down, got %+v", resp.Data.Dependencies["database"])
	}
}
