package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"pantheon-base/pkg/common"
	"pantheon-base/pkg/logging"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func TestRequestContextMiddlewareStoresRequestIDInRequestContext(t *testing.T) {
	gin.SetMode(gin.TestMode)

	engine := gin.New()
	engine.Use(RequestContextMiddleware())

	var loggerFromContext *zap.Logger

	engine.GET("/health", func(c *gin.Context) {
		loggerFromContext = logging.LogFromContext(c.Request.Context())
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	req.Header.Set(common.HeaderRequestID, "req-abc-123")
	rec := httptest.NewRecorder()

	engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected no content response, got %d", rec.Code)
	}
	if requestID := rec.Header().Get(common.HeaderRequestID); requestID != "req-abc-123" {
		t.Fatalf("expected request ID header to be set, got %q", requestID)
	}
	if traceID := rec.Header().Get(common.HeaderTraceID); traceID != "req-abc-123" {
		t.Fatalf("expected trace ID header to be set, got %q", traceID)
	}
	if loggerFromContext == nil {
		t.Fatal("expected logger from context to be set")
	}
	if loggerFromContext == logging.Logger {
		t.Fatal("expected request context logger to be enriched with a request ID")
	}
}
