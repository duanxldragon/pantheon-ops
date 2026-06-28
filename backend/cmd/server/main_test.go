package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestShouldExposeMetricsRequiresExplicitProductionGate(t *testing.T) {
	t.Setenv("PANTHEON_METRICS_ENABLED", "")
	t.Setenv("PANTHEON_METRICS_BEARER_TOKEN", "")
	t.Setenv("PANTHEON_METRICS_PUBLIC", "")

	if shouldExposeMetrics("production") {
		t.Fatal("expected production metrics to be disabled without token or public opt-in")
	}
	if !shouldExposeMetrics("development") {
		t.Fatal("expected non-production metrics to be exposed by default")
	}

	t.Setenv("PANTHEON_METRICS_BEARER_TOKEN", "secret")
	if !shouldExposeMetrics("production") {
		t.Fatal("expected production metrics to be exposed when token is configured")
	}
}

func TestMetricsAccessMiddlewareRequiresBearerTokenWhenConfigured(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("PANTHEON_METRICS_BEARER_TOKEN", "secret")

	engine := gin.New()
	engine.GET("/metrics", metricsAccessMiddleware(), func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected unauthorized without bearer token, got %d", recorder.Code)
	}

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	req.Header.Set("Authorization", "Bearer secret")
	recorder = httptest.NewRecorder()
	engine.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("expected authorized metrics request, got %d", recorder.Code)
	}
}
