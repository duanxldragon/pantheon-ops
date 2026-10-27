package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func setupSecurityHeadersRouter(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(SecurityHeadersMiddleware())
	router.GET("/ping", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	return router
}

func TestSecurityHeadersMiddlewareSetsMinimalHeaders(t *testing.T) {
	router := setupSecurityHeadersRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, req)

	if got := recorder.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Fatalf("expected nosniff header, got %q", got)
	}
	if got := recorder.Header().Get("X-Frame-Options"); got != "DENY" {
		t.Fatalf("expected frame deny header, got %q", got)
	}
	if got := recorder.Header().Get("Referrer-Policy"); got != "strict-origin-when-cross-origin" {
		t.Fatalf("expected referrer policy header, got %q", got)
	}
}

func TestSecurityHeadersMiddlewareSetsHSTS(t *testing.T) {
	router := setupSecurityHeadersRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	got := recorder.Header().Get("Strict-Transport-Security")
	if got != "max-age=31536000; includeSubDomains" {
		t.Fatalf("expected HSTS max-age=31536000; includeSubDomains, got %q", got)
	}
}

func TestSecurityHeadersMiddlewareSetsPermissionsPolicy(t *testing.T) {
	router := setupSecurityHeadersRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	got := recorder.Header().Get("Permissions-Policy")
	if got != "camera=(), microphone=(), geolocation=()" {
		t.Fatalf("expected permissions policy header, got %q", got)
	}
}

func TestSecurityHeadersMiddlewareDoesNotSetCSP(t *testing.T) {
	// CSP 由 CSPMiddleware 单一来源负责，SecurityHeadersMiddleware 不得重复设置。
	router := setupSecurityHeadersRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	if got := recorder.Header().Get("Content-Security-Policy"); got != "" {
		t.Fatalf("SecurityHeadersMiddleware must not set CSP (owned by CSPMiddleware), got %q", got)
	}
}
