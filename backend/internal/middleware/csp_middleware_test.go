package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func setupCSPRouter(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(CSPNonceMiddleware())
	router.Use(CSPMiddleware())
	router.GET("/ping", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	return router
}

func TestCSPMiddlewareSetsHeader(t *testing.T) {
	router := setupCSPRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	got := recorder.Header().Get("Content-Security-Policy")
	if got == "" {
		t.Fatal("expected Content-Security-Policy header, got empty")
	}
}

func TestCSPMiddlewareBaseDirectives(t *testing.T) {
	router := setupCSPRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	policy := recorder.Header().Get("Content-Security-Policy")

	requiredDirectives := []string{
		"default-src 'self'",
		"style-src 'self' 'unsafe-inline'",
		"img-src 'self' data:",
		"connect-src 'self'",
		"frame-ancestors 'none'",
		"base-uri 'self'",
		"form-action 'self'",
	}
	for _, directive := range requiredDirectives {
		if !strings.Contains(policy, directive) {
			t.Errorf("CSP policy missing required directive %q: %q", directive, policy)
		}
	}
}

func TestBuildCSPPolicyScriptSrcByEnv(t *testing.T) {
	testNonce := "test-nonce-abc123"

	t.Setenv("PANTHEON_ENV", "development")
	devPolicy := buildCSPPolicy(testNonce)
	if !strings.Contains(devPolicy, "script-src 'self' 'nonce-"+testNonce+"'") {
		t.Errorf("development policy should contain nonce, got %q", devPolicy)
	}
	if !strings.Contains(devPolicy, "'unsafe-inline' 'unsafe-eval'") {
		t.Errorf("development policy should allow unsafe-inline and unsafe-eval for Vite HMR, got %q", devPolicy)
	}

	t.Setenv("PANTHEON_ENV", "production")
	prodPolicy := buildCSPPolicy(testNonce)
	if strings.Contains(prodPolicy, "'unsafe-eval'") {
		t.Errorf("production policy must not contain 'unsafe-eval', got %q", prodPolicy)
	}
	if !strings.Contains(prodPolicy, "'nonce-"+testNonce+"'") {
		t.Errorf("production policy should contain nonce, got %q", prodPolicy)
	}
	// 生产环境有 nonce 时，script-src 不应包含 unsafe-inline
	scriptSrcStart := strings.Index(prodPolicy, "script-src")
	if scriptSrcStart != -1 {
		scriptSrcPart := prodPolicy[scriptSrcStart:]
		nextDirective := strings.Index(scriptSrcPart, ";")
		if nextDirective != -1 {
			scriptSrcPart = scriptSrcPart[:nextDirective]
		}
		if strings.Contains(scriptSrcPart, "'unsafe-inline'") {
			t.Errorf("production script-src with nonce should not contain 'unsafe-inline', got %q", scriptSrcPart)
		}
	}
}

func TestBuildCSPPolicyReportURI(t *testing.T) {
	t.Setenv("CSP_REPORT_URI", "https://csp.example.com/report")
	policy := buildCSPPolicy("test-nonce")
	if !strings.Contains(policy, "report-uri https://csp.example.com/report") {
		t.Errorf("expected report-uri directive, got %q", policy)
	}
}

func TestCSPPolicyWithNonce(t *testing.T) {
	testNonce := "AbCdEf123456=="

	t.Setenv("PANTHEON_ENV", "production")
	policy := buildCSPPolicy(testNonce)

	// 验证包含 nonce
	expectedNonce := "'nonce-" + testNonce + "'"
	if !strings.Contains(policy, expectedNonce) {
		t.Errorf("policy should contain %s, got %q", expectedNonce, policy)
	}

	// 验证 script-src 不包含 unsafe-inline（生产环境 + nonce）
	scriptSrcStart := strings.Index(policy, "script-src")
	if scriptSrcStart != -1 {
		scriptSrcPart := policy[scriptSrcStart:]
		nextDirective := strings.Index(scriptSrcPart, ";")
		if nextDirective != -1 {
			scriptSrcPart = scriptSrcPart[:nextDirective]
		}
		if strings.Contains(scriptSrcPart, "'unsafe-inline'") {
			t.Errorf("production script-src with nonce should not contain 'unsafe-inline', got %q", scriptSrcPart)
		}
	}
}

func TestCSPPolicyWithoutNonce(t *testing.T) {
	t.Setenv("PANTHEON_ENV", "production")
	policy := buildCSPPolicy("")

	// 没有 nonce 时应该降级到 unsafe-inline
	if !strings.Contains(policy, "'unsafe-inline'") {
		t.Errorf("policy without nonce should contain 'unsafe-inline' as fallback, got %q", policy)
	}
}
