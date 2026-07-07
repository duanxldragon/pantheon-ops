package http

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestGetRequestIDReturnsEmptyForNilContext(t *testing.T) {
	if id := GetRequestID(nil); id != "" {
		t.Fatalf("expected empty string for nil context, got %s", id)
	}
}

func TestGetRequestIDFromHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = &http.Request{
		Header: http.Header{},
	}
	c.Request.Header.Set(HeaderRequestID, "req-abc-123")

	id := GetRequestID(c)
	if id != "req-abc-123" {
		t.Fatalf("expected req-abc-123, got %s", id)
	}
}

func TestGetRequestIDFromContextKey(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)

	c.Set(ContextKeyRequestID, "ctx-req-456")

	id := GetRequestID(c)
	if id != "ctx-req-456" {
		t.Fatalf("expected ctx-req-456, got %s", id)
	}
}

func TestGetRequestIDContextKeyTakesPrecedence(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = &http.Request{
		Header: http.Header{},
	}
	c.Request.Header.Set(HeaderRequestID, "from-header")
	c.Set(ContextKeyRequestID, "from-context")

	id := GetRequestID(c)
	if id != "from-context" {
		t.Fatalf("expected from-context (key precedence), got %s", id)
	}
}

func TestGetRequestIDNoHeaderNoContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = &http.Request{
		Header: http.Header{},
	}

	id := GetRequestID(c)
	if id != "" {
		t.Fatalf("expected empty string, got %s", id)
	}
}
