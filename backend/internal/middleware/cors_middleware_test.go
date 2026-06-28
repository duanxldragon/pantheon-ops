package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestCORSMiddlewareDoesNotGrantUnknownOrigin(t *testing.T) {
	t.Setenv("PANTHEON_ALLOWED_ORIGINS", "https://console.example.com")
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(CORSMiddleware())
	router.GET("/ping", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	req.Header.Set("Origin", "https://evil.example.com")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, req)

	if got := recorder.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("expected unknown origin to be rejected, got %q", got)
	}
	if got := recorder.Header().Get("Access-Control-Allow-Credentials"); got != "" {
		t.Fatalf("expected unknown origin to omit credentials header, got %q", got)
	}
}

func TestCORSMiddlewareAllowsConfiguredOrigin(t *testing.T) {
	t.Setenv("PANTHEON_ALLOWED_ORIGINS", "https://console.example.com, https://ops.example.com")
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(CORSMiddleware())
	router.GET("/ping", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	req.Header.Set("Origin", "https://ops.example.com")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, req)

	if got := recorder.Header().Get("Access-Control-Allow-Origin"); got != "https://ops.example.com" {
		t.Fatalf("expected allowed origin to be echoed back, got %q", got)
	}
	if got := recorder.Header().Get("Access-Control-Allow-Credentials"); got != "true" {
		t.Fatalf("expected credentials header for allowed origin, got %q", got)
	}
	if got := recorder.Header().Get("Access-Control-Expose-Headers"); got == "" {
		t.Fatal("expected expose headers to remain configured")
	}
}
