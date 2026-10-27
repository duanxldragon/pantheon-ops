package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// PlatformCapabilityMiddleware (contract §3.3) gates the platform-global
// write surfaces (tenant console, menu authoring, i18n catalogue). These
// tests pin the capability signals without any database:
//   - compat-mode platform admin: a global (domain-less) Casbin policy for
//     the bare role key passes;
//   - platform_ops role key: passes on any route;
//   - tenant_admin whose only grant is a tenant-domain subject
//     (`role:tenant_admin@tenant:<id>`): denied 403, because the bare-key
//     global check cannot be satisfied by a tenant-domain policy.
func runPlatformCapabilityRequest(t *testing.T, roles []string, policies [][]string) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	setTestEnforcer(t, newTestEnforcer(t, policies...))

	reachedHandler := false
	engine := gin.New()
	engine.Use(func(c *gin.Context) {
		if len(roles) > 0 {
			c.Set("roleKeys", roles)
		}
		c.Next()
	})
	engine.POST("/api/v1/system/menu", PlatformCapabilityMiddleware(), func(c *gin.Context) {
		reachedHandler = true
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/system/menu", nil)
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, req)

	if recorder.Code == http.StatusNoContent && !reachedHandler {
		t.Fatal("success response must have reached the protected handler")
	}
	return recorder
}

func TestPlatformCapabilityMiddleware_CompatAdminGlobalPolicyPasses(t *testing.T) {
	// Compat-mode admin: its authority is the bare global role key matched by
	// a domain-less policy — exactly the grant this gate accepts.
	recorder := runPlatformCapabilityRequest(t,
		[]string{"admin"},
		[][]string{{"admin", "/api/v1/system/menu", http.MethodPost}},
	)
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("compat admin with a global policy must pass the platform gate, got %d", recorder.Code)
	}
}

func TestPlatformCapabilityMiddleware_PlatformOpsRolePasses(t *testing.T) {
	recorder := runPlatformCapabilityRequest(t, []string{platformOpsRoleKey}, nil)
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("platform_ops must pass the platform gate, got %d", recorder.Code)
	}
}

func TestPlatformCapabilityMiddleware_TenantAdminDomainGrantDenied(t *testing.T) {
	// A tenant administrator in multi mode holds tenant-domain grants (and,
	// per contract §4, no domain-less global policy for this endpoint). The
	// bare-key global check must not be satisfied by the domain subject.
	recorder := runPlatformCapabilityRequest(t,
		[]string{"tenant_admin"},
		[][]string{{"role:tenant_admin@tenant:101", "/api/v1/system/menu", http.MethodPost}},
	)
	assertForbiddenResponse(t, recorder, "tenant.platformOnly")
}

func TestPlatformCapabilityMiddleware_AnonymousSubjectDenied(t *testing.T) {
	recorder := runPlatformCapabilityRequest(t, nil, nil)
	assertForbiddenResponse(t, recorder, "tenant.platformOnly")
}
