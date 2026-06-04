package platform

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"pantheon-platform/backend/internal/middleware"
	"pantheon-platform/backend/pkg/testmysql"

	"github.com/gin-gonic/gin"
)

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
