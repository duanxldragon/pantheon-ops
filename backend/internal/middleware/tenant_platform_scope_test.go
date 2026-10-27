package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
	"github.com/duanxldragon/pantheon-base/backend/pkg/tenant"
	"github.com/duanxldragon/pantheon-base/backend/pkg/testmysql"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// The platform-scope variant (contract §3.3 "平台管理 API（显式声明，要求平台权限，
// 落审计）") adds exactly one resolution branch on top of the strict middleware:
// a platform-capable subject with no tenant claim resolves to the
// platform-global context (tenant 0) instead of being denied. That branch is
// what lets a tenant-less platform operator and tenant subjects share one
// multi-mode deployment.
//
// These tests pin the branch's two conditions — the route must have declared
// the platform scope AND the subject must carry a platform-level grant — and
// pin that it never outranks the higher-priority sources.
func TestPlatformScopeTenantContextMiddleware_ResolutionMatrix(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// The explicit-override branch validates the header against the master
	// table (contract §3.1 source 1, §5), so seed a real active tenant 202.
	// The implicit-fallback branch requires the subject to belong to no tenant,
	// so the seeded membership table is deliberately empty and every request
	// carries a tenant-less user id.
	db := setupTenantContextTestDB(t)
	const tenantLessUserID uint64 = 8

	tests := []struct {
		name           string
		platformScoped bool
		mode           string
		subjectTenant  string
		headerTenant   string
		roles          []string
		wantCode       int
		wantTenantID   uint64
		wantMode       string
	}{
		{
			name:           "declared platform scope lets a tenant-less operator reach tenant 0",
			platformScoped: true, mode: tenant.ModeMulti, roles: []string{"platform_ops"},
			wantCode: common.CodeSuccess, wantTenantID: tenant.PlatformGlobalTenantID, wantMode: tenant.ModeMulti,
		},
		{
			name:           "capability is still required on a declared route",
			platformScoped: true, mode: tenant.ModeMulti,
			wantCode: common.CodeError,
		},
		{
			name:           "undeclared route denies the same operator",
			platformScoped: false, mode: tenant.ModeMulti, roles: []string{"platform_ops"},
			wantCode: common.CodeError,
		},
		{
			name:           "subject claim outranks the platform fallback",
			platformScoped: true, mode: tenant.ModeMulti, subjectTenant: "101", roles: []string{"platform_ops"},
			wantCode: common.CodeSuccess, wantTenantID: 101, wantMode: tenant.ModeMulti,
		},
		{
			name:           "explicit header outranks the platform fallback",
			platformScoped: true, mode: tenant.ModeMulti, headerTenant: "202", roles: []string{"platform_ops"},
			wantCode: common.CodeSuccess, wantTenantID: 202, wantMode: tenant.ModeMulti,
		},
		{
			name:           "explicit header may pin the platform-global scope",
			platformScoped: true, mode: tenant.ModeMulti, headerTenant: "0", roles: []string{"platform_ops"},
			wantCode: common.CodeSuccess, wantTenantID: tenant.PlatformGlobalTenantID, wantMode: tenant.ModeMulti,
		},
		{
			name:           "header without capability is still forbidden",
			platformScoped: true, mode: tenant.ModeMulti, headerTenant: "202",
			wantCode: common.CodeForbidden,
		},
		{
			name:           "compat keeps the compat fallback for every subject",
			platformScoped: true, mode: tenant.ModeCompat,
			wantCode: common.CodeSuccess, wantTenantID: tenant.PlatformGlobalTenantID, wantMode: tenant.ModeCompat,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := runTenantScopeRequest(t, tenantScopeRequest{
				platformScoped: tt.platformScoped, mode: tt.mode, subjectTenant: tt.subjectTenant,
				headerTenant: tt.headerTenant, roles: tt.roles, membershipDB: db, userID: tenantLessUserID,
			})
			assertTenantSmokeResponse(t, recorder, http.StatusOK, tt.wantCode, tt.wantTenantID, tt.wantMode)
		})
	}
}

// The implicit platform fallback is additionally gated on the subject holding
// no membership at all, so it can never shadow a tenant membership. A
// membership-bearing subject still reaches the global scope, but only through
// the explicit header override.
// A claim-less platform subject that holds only narrow (non-platform) roles
// must still be able to bootstrap: self-service routes and the tenant switcher
// are authorized by the login state alone (PERMISSION_MODEL §10.3, contract
// §2.2), so no global Casbin policy exists for them. They resolve the
// platform-global context instead of answering tenant.context.missing, while
// data routes keep denying the same subject and the explicit header override
// keeps requiring a real platform grant.
func TestPlatformScopeTenantContextMiddleware_LoginScopedRoutesResolveForPlatformSubject(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupTenantMembershipTestDB(t)
	const tenantLessUserID uint64 = 8

	tests := []struct {
		name         string
		path         string
		headerTenant string
		wantCode     int
		wantTenantID uint64
	}{
		{
			name:     "self-service route resolves the platform scope",
			path:     "/api/v1/auth/me",
			wantCode: common.CodeSuccess, wantTenantID: tenant.PlatformGlobalTenantID,
		},
		{
			name:     "tenant switcher current context resolves the platform scope",
			path:     "/api/v1/auth/tenants/current",
			wantCode: common.CodeSuccess, wantTenantID: tenant.PlatformGlobalTenantID,
		},
		{
			name:     "tenant switcher list resolves the platform scope",
			path:     "/api/v1/auth/tenants/switchable",
			wantCode: common.CodeSuccess, wantTenantID: tenant.PlatformGlobalTenantID,
		},
		{
			name:     "data route still requires a platform policy",
			path:     "/api/v1/system/i18n/list",
			wantCode: common.CodeError,
		},
		{
			name: "header override still requires a platform grant",
			path: "/api/v1/auth/me", headerTenant: "0",
			wantCode: common.CodeForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := runTenantScopeRequest(t, tenantScopeRequest{
				platformScoped: true, mode: tenant.ModeMulti, headerTenant: tt.headerTenant,
				membershipDB: db, userID: tenantLessUserID, path: tt.path,
			})
			assertTenantSmokeResponse(t, recorder, http.StatusOK, tt.wantCode, tt.wantTenantID, tenant.ModeMulti)
		})
	}
}

// Relaxing the fallback for login-scoped routes must not relax its
// membership condition: a subject that already belongs to a tenant cannot
// reach the platform-global scope implicitly, not even on /auth/me.
func TestPlatformScopeTenantContextMiddleware_MembershipBlocksLoginScopedFallback(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupTenantMembershipTestDB(t)
	if err := db.Create(&tenant.Membership{
		TenantID: 5, UserID: 7, Role: tenant.MembershipRoleMember, Status: tenant.MembershipActive,
	}).Error; err != nil {
		t.Fatalf("seed membership: %v", err)
	}

	recorder := runTenantScopeRequest(t, tenantScopeRequest{
		platformScoped: true, mode: tenant.ModeMulti,
		membershipDB: db, userID: 7, path: "/api/v1/auth/me",
	})
	assertTenantSmokeResponse(t, recorder, http.StatusOK, common.CodeError, 0, "")
}

func TestPlatformScopeTenantContextMiddleware_MembershipBlocksImplicitFallback(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupTenantMembershipTestDB(t)

	if err := db.Create(&tenant.Membership{
		TenantID: 5, UserID: 7, Role: tenant.MembershipRoleMember, Status: tenant.MembershipActive,
	}).Error; err != nil {
		t.Fatalf("seed membership: %v", err)
	}

	tests := []struct {
		name         string
		userID       uint64
		headerTenant string
		wantCode     int
		wantTenantID uint64
	}{
		{
			name:   "member without a claim cannot fall back implicitly",
			userID: 7, wantCode: common.CodeError,
		},
		{
			name:   "member can still pin the platform-global scope explicitly",
			userID: 7, headerTenant: "0",
			wantCode: common.CodeSuccess, wantTenantID: tenant.PlatformGlobalTenantID,
		},
		{
			name:     "tenant-less operator falls back implicitly",
			userID:   8,
			wantCode: common.CodeSuccess, wantTenantID: tenant.PlatformGlobalTenantID,
		},
		{
			name:   "unknown subject fails closed",
			userID: 0, wantCode: common.CodeError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := runTenantScopeRequest(t, tenantScopeRequest{
				platformScoped: true, mode: tenant.ModeMulti, headerTenant: tt.headerTenant,
				roles: []string{"platform_ops"}, membershipDB: db, userID: tt.userID,
			})
			assertTenantSmokeResponse(t, recorder, http.StatusOK, tt.wantCode, tt.wantTenantID, tenant.ModeMulti)
		})
	}
}

func setupTenantMembershipTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db := testmysql.Open(t)
	if err := db.AutoMigrate(&tenant.Membership{}); err != nil {
		t.Fatalf("migrate tenant memberships: %v", err)
	}
	return db
}

// setupTenantContextTestDB opens a fresh schema with both the tenant master and
// the membership projection, and seeds an active tenant 202 — the id the
// resolution matrices pin through X-Tenant-Id. The membership table is left
// empty so a caller-supplied user id reads as tenant-less and the implicit
// platform fallback stays reachable.
func setupTenantContextTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db := testmysql.Open(t)
	if err := db.AutoMigrate(&tenant.Tenant{}, &tenant.Membership{}); err != nil {
		t.Fatalf("migrate tenant context tables: %v", err)
	}
	if err := db.Create(&tenant.Tenant{
		ID: 202, Code: "tenant-202", Name: "Tenant 202", Status: tenant.TenantStatusActive,
	}).Error; err != nil {
		t.Fatalf("seed tenant 202: %v", err)
	}
	return db
}

// runTenantScopeRequest issues the protected request through either the strict
// or the platform-scoped tenant context middleware, with the same stub context
// keys the hostile matrix uses.
// tenantScopeRequest is the input of one protected request: which middleware
// variant runs, the subject's stub context, and the database the membership
// lookup reads. Grouping the fields keeps the helper's signature readable and
// makes each call site name what it is varying.
type tenantScopeRequest struct {
	platformScoped bool
	mode           string
	subjectTenant  string
	headerTenant   string
	roles          []string
	membershipDB   *gorm.DB
	userID         uint64
	path           string
}

func runTenantScopeRequest(t *testing.T, spec tenantScopeRequest) *httptest.ResponseRecorder {
	t.Helper()
	loader := tenant.NewModeLoader(func() string { return spec.mode }, 0)
	engine := gin.New()
	engine.Use(func(c *gin.Context) {
		if spec.subjectTenant != "" {
			c.Set("tenantId", spec.subjectTenant)
		}
		if len(spec.roles) > 0 {
			c.Set("roleKeys", spec.roles)
		}
		if spec.userID != 0 {
			c.Set("userID", spec.userID)
		}
	})
	if spec.platformScoped {
		engine.Use(PlatformScopeTenantContextMiddleware(loader, spec.membershipDB))
	} else {
		engine.Use(TenantContextMiddleware(loader, spec.membershipDB))
	}
	path := spec.path
	if path == "" {
		path = "/protected"
	}
	engine.GET(path, func(c *gin.Context) {
		ctx := tenant.FromGin(c)
		if ctx == nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": common.CodeError})
			return
		}
		c.JSON(http.StatusOK, gin.H{"code": common.CodeSuccess, "data": tenantSmokeResponse{
			TenantID: ctx.TenantID, Mode: ctx.Mode,
		}})
	})

	req := httptest.NewRequest(http.MethodGet, path, nil)
	if spec.headerTenant != "" {
		req.Header.Set(tenant.HeaderTenantID, spec.headerTenant)
	}
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, req)
	return recorder
}

// The failure envelopes must stay distinguishable: a missing context is a
// configuration error (§5, 500) while a rejected override is an authorization
// error (403).
func TestPlatformScopeTenantContextMiddleware_FailureEnvelopes(t *testing.T) {
	gin.SetMode(gin.TestMode)

	recorder := runTenantScopeRequest(t, tenantScopeRequest{platformScoped: true, mode: tenant.ModeMulti})
	var payload struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Code != common.CodeError || payload.Message != tenant.ErrTenantContextMissing.Error() {
		t.Fatalf("missing context must report %q with code %d, got %+v",
			tenant.ErrTenantContextMissing.Error(), common.CodeError, payload)
	}

	recorder = runTenantScopeRequest(t, tenantScopeRequest{platformScoped: true, mode: tenant.ModeMulti, headerTenant: "202"})
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Code != common.CodeForbidden {
		t.Fatalf("rejected override must report code %d, got %+v", common.CodeForbidden, payload)
	}
}
