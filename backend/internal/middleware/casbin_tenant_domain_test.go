package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/duanxldragon/pantheon-base/backend/pkg/tenant"
	"github.com/gin-gonic/gin"
)

// ─────────────────────────────────────────────────────────────
// Casbin tenant-domain enforcement (contract §4, slice 2 wiring)
// ─────────────────────────────────────────────────────────────

func newTenantContext(c *gin.Context, tenantID uint64, mode string) {
	tenant.SetGin(c, &tenant.Context{TenantID: tenantID, Mode: mode, ResolvedBy: "subject"})
}

// Tenant-scoped policy grants the subject only inside its tenant context.
func TestCasbinMiddleware_TenantDomainPolicyGrants(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setTestEnforcer(t, newTestEnforcer(t,
		[]string{"role:editor@tenant:101", "/api/v1/system/users", http.MethodGet},
	))

	engine := gin.New()
	engine.Use(func(c *gin.Context) {
		c.Set("roleKeys", []string{"editor"})
		newTenantContext(c, 101, tenant.ModeMulti)
		c.Next()
	})
	engine.GET("/api/v1/system/users", CasbinMiddleware(), func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/system/users", nil)
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("expected tenant-domain policy to authorize request, got %d", recorder.Code)
	}
}

// The same role key with NO tenant policy stays denied in multi mode when the
// global policy doesn't cover the path — the global namespace is consulted
// first, then the domain namespace; neither silently widens the other.
func TestCasbinMiddleware_TenantContextWithoutDomainPolicyDenied(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setTestEnforcer(t, newTestEnforcer(t)) // no policies at all

	reachedHandler := false
	engine := gin.New()
	engine.Use(func(c *gin.Context) {
		c.Set("roleKeys", []string{"editor"})
		newTenantContext(c, 101, tenant.ModeMulti)
		c.Next()
	})
	engine.GET("/api/v1/system/users", CasbinMiddleware(), func(c *gin.Context) {
		reachedHandler = true
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/system/users", nil)
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, req)

	assertForbiddenResponse(t, recorder, "permission.denied")
	if reachedHandler {
		t.Fatal("expected denied request to abort before reaching protected handler")
	}
}

// Cross-tenant leak attempt: tenant 101's policy must NOT authorize tenant
// 202's request (the domain subject is tenant-specific).
func TestCasbinMiddleware_CrossTenantPolicyDoesNotLeak(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setTestEnforcer(t, newTestEnforcer(t,
		[]string{"role:editor@tenant:101", "/api/v1/system/users", http.MethodGet},
	))

	reachedHandler := false
	engine := gin.New()
	engine.Use(func(c *gin.Context) {
		c.Set("roleKeys", []string{"editor"})
		newTenantContext(c, 202, tenant.ModeMulti) // different tenant
		c.Next()
	})
	engine.GET("/api/v1/system/users", CasbinMiddleware(), func(c *gin.Context) {
		reachedHandler = true
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/system/users", nil)
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, req)

	assertForbiddenResponse(t, recorder, "permission.denied")
	if reachedHandler {
		t.Fatal("tenant 202 must not be authorized by tenant 101's domain policy")
	}
}

// Compat context (flag off / global) consults ONLY the global subject: a
// tenant-domain policy never grants a compat request (no backwards leak).
func TestCasbinMiddleware_CompatContextIgnoresTenantPolicies(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setTestEnforcer(t, newTestEnforcer(t,
		[]string{"role:editor@tenant:101", "/api/v1/system/users", http.MethodGet},
	))

	reachedHandler := false
	engine := gin.New()
	engine.Use(func(c *gin.Context) {
		c.Set("roleKeys", []string{"editor"})
		newTenantContext(c, tenant.PlatformGlobalTenantID, tenant.ModeCompat)
		c.Next()
	})
	engine.GET("/api/v1/system/users", CasbinMiddleware(), func(c *gin.Context) {
		reachedHandler = true
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/system/users", nil)
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, req)

	assertForbiddenResponse(t, recorder, "permission.denied")
	if reachedHandler {
		t.Fatal("compat request must not be authorized by a tenant-domain policy")
	}
}

// Global policy still authorizes a multi-mode tenant request (global-first
// order, contract §4: platform capabilities are not lost inside a tenant).
func TestCasbinMiddleware_GlobalPolicyStillAppliesInTenantContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setTestEnforcer(t, newTestEnforcer(t,
		[]string{"editor", "/api/v1/system/users", http.MethodGet},
	))

	engine := gin.New()
	engine.Use(func(c *gin.Context) {
		c.Set("roleKeys", []string{"editor"})
		newTenantContext(c, 101, tenant.ModeMulti)
		c.Next()
	})
	engine.GET("/api/v1/system/users", CasbinMiddleware(), func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/system/users", nil)
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("expected global policy to authorize in-tenant request, got %d", recorder.Code)
	}
}

// Nil tenant context (e.g. route group without the tenant middleware) behaves
// exactly like the legacy global-only check.
func TestCasbinMiddleware_NilTenantContextGlobalOnly(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setTestEnforcer(t, newTestEnforcer(t,
		[]string{"role:editor@tenant:101", "/api/v1/system/users", http.MethodGet},
	))

	reachedHandler := false
	engine := gin.New()
	engine.Use(func(c *gin.Context) {
		c.Set("roleKeys", []string{"editor"})
		// no tenant context set
		c.Next()
	})
	engine.GET("/api/v1/system/users", CasbinMiddleware(), func(c *gin.Context) {
		reachedHandler = true
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/system/users", nil)
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, req)

	assertForbiddenResponse(t, recorder, "permission.denied")
	if reachedHandler {
		t.Fatal("nil tenant context must consult global subjects only")
	}
}
