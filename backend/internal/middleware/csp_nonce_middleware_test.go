package middleware

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestCSPNonceGeneration(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(CSPNonceMiddleware())
	router.GET("/test", func(c *gin.Context) {
		nonce := GetCSPNonce(c)
		c.String(http.StatusOK, nonce)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	nonce := w.Body.String()
	if nonce == "" {
		t.Fatal("nonce should not be empty")
	}
	if len(nonce) < 16 {
		t.Errorf("nonce too short: %d bytes", len(nonce))
	}
}

func TestCSPNonceUniqueness(t *testing.T) {
	// 生成100个 nonce，确保全部唯一
	nonces := make(map[string]bool)
	for i := 0; i < 100; i++ {
		nonce := generateNonce()
		if nonces[nonce] {
			t.Fatalf("duplicate nonce generated: %s", nonce)
		}
		nonces[nonce] = true
	}
}

func TestGetCSPNonceFromContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(CSPNonceMiddleware())

	var capturedNonce string
	router.GET("/test", func(c *gin.Context) {
		capturedNonce = GetCSPNonce(c)
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if capturedNonce == "" {
		t.Fatal("captured nonce should not be empty")
	}
}

func TestGetCSPNonceWithoutMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())

	nonce := GetCSPNonce(c)
	if nonce != "" {
		t.Errorf("expected empty nonce without middleware, got %q", nonce)
	}
}

func TestCSPNonceBase64Encoding(t *testing.T) {
	nonce := generateNonce()

	// 验证 nonce 是有效的 base64
	if _, err := base64.StdEncoding.DecodeString(nonce); err != nil {
		t.Errorf("nonce is not valid base64: %s, error: %v", nonce, err)
	}

	// 验证解码后的长度
	decoded, _ := base64.StdEncoding.DecodeString(nonce)
	if len(decoded) < nonceLength && !strings.Contains(nonce, "=") {
		t.Errorf("decoded nonce length unexpected: %d bytes (expected >= %d)", len(decoded), nonceLength)
	}
}
