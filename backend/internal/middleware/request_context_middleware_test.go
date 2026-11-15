package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/duanxldragon/pantheon-base/backend/pkg/auditpolicy"
	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
	"github.com/duanxldragon/pantheon-base/backend/pkg/logging"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
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

func TestMetadataAwareLoggerWritesSanitizedSingleLineRouteFields(t *testing.T) {
	gin.SetMode(gin.TestMode)

	route := auditpolicy.Route{Method: http.MethodGet, Template: "/api/v1/log-sanitize/:id"}
	if err := auditpolicy.RegisterMetadataOnly([]auditpolicy.Route{route}); err != nil {
		t.Fatal(err)
	}
	core, observed := observer.New(zap.DebugLevel)
	previous := logging.Logger
	logging.Logger = zap.New(core)
	defer func() { logging.Logger = previous }()

	engine := gin.New()
	engine.Use(MetadataAwareLogger())
	engine.GET(route.Template, func(c *gin.Context) { c.Status(http.StatusNoContent) })

	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/log-sanitize/7", nil))

	entries := observed.FilterMessage("business request").All()
	if len(entries) != 1 {
		t.Fatalf("expected one business request entry, got %d", len(entries))
	}
	fields := entries[0].ContextMap()
	for key, want := range map[string]string{"method": route.Method, "route": route.Template} {
		got, ok := fields[key].(string)
		if !ok {
			t.Fatalf("field %q missing or not a string: %#v", key, fields[key])
		}
		if got != want {
			t.Fatalf("field %q = %q, want registry value %q", key, got, want)
		}
		if strings.ContainsAny(got, "\n\r") {
			t.Fatalf("field %q contains a control character: %q", key, got)
		}
	}
}
