package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TestRequestLogFieldsUseRouteTemplate(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var encoded map[string]interface{}
	router := gin.New()
	router.GET("/users/:id", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
		encoded = encodeFields(requestLogFields(c, 250*time.Millisecond))
	})

	req := httptest.NewRequest(http.MethodGet, "/users/42?expand=roles", nil)
	req.Header.Set("User-Agent", "pantheon-tests")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	if got := encoded["method"]; got != http.MethodGet {
		t.Fatalf("method = %v, want %q", got, http.MethodGet)
	}
	if got := encoded["route"]; got != "/users/:id" {
		t.Fatalf("route = %v, want %q", got, "/users/:id")
	}
	if got := encoded["query_present"]; got != true {
		t.Fatalf("query_present = %v, want true", got)
	}
	if got := encoded["client_ip_present"]; got != true {
		t.Fatalf("client_ip_present = %v, want true", got)
	}
	if got := encoded["user_agent_present"]; got != true {
		t.Fatalf("user_agent_present = %v, want true", got)
	}
}

func TestRequestRouteLabelFallsBackForUnmatchedRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var encoded map[string]interface{}
	router := gin.New()
	router.NoRoute(func(c *gin.Context) {
		c.Status(http.StatusNotFound)
		encoded = encodeFields(requestLogFields(c, 100*time.Millisecond))
	})

	req := httptest.NewRequest(http.MethodGet, "/missing?foo=bar", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	if got := encoded["route"]; got != "unmatched" {
		t.Fatalf("route = %v, want %q", got, "unmatched")
	}
	if got := encoded["query_present"]; got != true {
		t.Fatalf("query_present = %v, want true", got)
	}
}

func encodeFields(fields []zap.Field) map[string]interface{} {
	encoder := zapcore.NewMapObjectEncoder()
	for _, field := range fields {
		field.AddTo(encoder)
	}
	return encoder.Fields
}
