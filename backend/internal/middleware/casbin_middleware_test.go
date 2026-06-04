package middleware

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"pantheon-platform/backend/pkg/common"
	"pantheon-platform/backend/pkg/database"

	"github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/model"
	"github.com/gin-gonic/gin"
)

func TestIsSelfServiceRouteBySignature_MenuTreeScopeBoundary(t *testing.T) {
	tests := []struct {
		name     string
		fullPath string
		method   string
		scope    string
		want     bool
	}{
		{name: "system logout", fullPath: "/api/v1/system/logout", method: http.MethodPost, want: true},
		{name: "auth logout", fullPath: "/api/v1/auth/logout", method: http.MethodPost, want: true},
		{name: "auth activity", fullPath: "/api/v1/auth/activity", method: http.MethodPost, want: true},
		{name: "system user info", fullPath: "/api/v1/system/user/info", method: http.MethodGet, want: true},
		{name: "auth me", fullPath: "/api/v1/auth/me", method: http.MethodGet, want: true},
		{name: "auth security", fullPath: "/api/v1/auth/security", method: http.MethodGet, want: true},
		{name: "system profile get", fullPath: "/api/v1/system/profile", method: http.MethodGet, want: true},
		{name: "system profile put", fullPath: "/api/v1/system/profile", method: http.MethodPut, want: true},
		{name: "system profile password", fullPath: "/api/v1/system/profile/password", method: http.MethodPut, want: true},
		{name: "system menu tree nav scope", fullPath: "/api/v1/system/menu/tree", method: http.MethodGet, scope: "nav", want: true},
		{name: "system menu tree empty scope", fullPath: "/api/v1/system/menu/tree", method: http.MethodGet, scope: "", want: true},
		{name: "system menu tree manage scope", fullPath: "/api/v1/system/menu/tree", method: http.MethodGet, scope: "manage", want: false},
		{name: "auth password", fullPath: "/api/v1/auth/password", method: http.MethodPut, want: true},
		{name: "auth sessions list", fullPath: "/api/v1/auth/sessions", method: http.MethodGet, want: true},
		{name: "auth session delete", fullPath: "/api/v1/auth/sessions/:id", method: http.MethodDelete, want: true},
		{name: "auth login logs", fullPath: "/api/v1/auth/login-logs", method: http.MethodGet, want: true},
		{name: "wrong method", fullPath: "/api/v1/system/logout", method: http.MethodGet, want: false},
		{name: "unknown route", fullPath: "/api/v1/system/roles", method: http.MethodGet, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isSelfServiceRouteBySignature(tt.fullPath, tt.method, tt.scope)
			if got != tt.want {
				t.Fatalf("expected %v for %s %s scope=%q, got %v", tt.want, tt.method, tt.fullPath, tt.scope, got)
			}
		})
	}
}

func TestReadRoleKeysFromContext_PrefersRoleKeysSlice(t *testing.T) {
	c, _ := gin.CreateTestContext(nil)
	c.Set("roleKeys", []string{"admin", "auditor"})
	c.Set("roleKey", "guest")

	roleKeys := readRoleKeysFromContext(c)
	if len(roleKeys) != 2 || roleKeys[0] != "admin" || roleKeys[1] != "auditor" {
		t.Fatalf("expected roleKeys slice from context, got %#v", roleKeys)
	}
}

func TestReadRoleKeysFromContext_FallsBackToSingleRoleKey(t *testing.T) {
	c, _ := gin.CreateTestContext(nil)
	c.Set("roleKey", "operator")

	roleKeys := readRoleKeysFromContext(c)
	if len(roleKeys) != 1 || roleKeys[0] != "operator" {
		t.Fatalf("expected single roleKey fallback, got %#v", roleKeys)
	}
}

func TestAuthorizeRoleKeys_AllowsWhenAnyRoleSucceeds(t *testing.T) {
	allowed, err := authorizeRoleKeys([]string{"guest", "admin"}, "/api/v1/system/users", "GET", func(roleKey, obj, act string) (bool, error) {
		return roleKey == "admin", nil
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if !allowed {
		t.Fatal("expected access to be allowed when one role matches")
	}
}

func TestAuthorizeRoleKeys_DeniesWhenNoRoleMatches(t *testing.T) {
	allowed, err := authorizeRoleKeys([]string{"guest", "operator"}, "/api/v1/system/users", "GET", func(roleKey, obj, act string) (bool, error) {
		return false, nil
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if allowed {
		t.Fatal("expected access to be denied when no roles match")
	}
}

func TestAuthorizeRoleKeys_StopsOnEnforcerError(t *testing.T) {
	expectedErr := errors.New("enforce failed")
	allowed, err := authorizeRoleKeys([]string{"admin"}, "/api/v1/system/users", "GET", func(roleKey, obj, act string) (bool, error) {
		return false, expectedErr
	})
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected %v, got %v", expectedErr, err)
	}
	if allowed {
		t.Fatal("expected access to be denied when enforcer returns error")
	}
}

func TestCasbinMiddleware_BypassesSelfServiceRoute(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setTestEnforcer(t, nil)

	engine := gin.New()
	engine.GET("/api/v1/auth/me", CasbinMiddleware(), func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("expected self-service route to bypass authorization, got %d", recorder.Code)
	}
}

func TestCasbinMiddleware_RejectsWhenEnforcerIsNotInitialized(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setTestEnforcer(t, nil)

	reachedHandler := false
	engine := gin.New()
	engine.Use(func(c *gin.Context) {
		c.Set("roleKey", "admin")
		c.Next()
	})
	engine.GET("/api/v1/system/users", CasbinMiddleware(), func(c *gin.Context) {
		reachedHandler = true
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/system/users", nil)
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, req)

	assertForbiddenResponse(t, recorder, "permission.engine.not_initialized")
	if reachedHandler {
		t.Fatal("expected request to abort before reaching protected handler")
	}
}

func TestCasbinMiddleware_UsesGuestRoleFallback(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setTestEnforcer(t, newTestEnforcer(t, []string{"guest", "/api/v1/system/users", http.MethodGet}))

	engine := gin.New()
	engine.GET("/api/v1/system/users", CasbinMiddleware(), func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/system/users", nil)
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("expected guest fallback to authorize request, got %d", recorder.Code)
	}
}

func TestCasbinMiddleware_RejectsDeniedRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setTestEnforcer(t, newTestEnforcer(t))

	reachedHandler := false
	engine := gin.New()
	engine.Use(func(c *gin.Context) {
		c.Set("roleKey", "operator")
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

func setTestEnforcer(t *testing.T, enforcer *casbin.SyncedEnforcer) {
	t.Helper()
	original := database.Enforcer
	database.Enforcer = enforcer
	t.Cleanup(func() {
		database.Enforcer = original
	})
}

func newTestEnforcer(t *testing.T, policies ...[]string) *casbin.SyncedEnforcer {
	t.Helper()

	m, err := model.NewModelFromString(`
		[request_definition]
		r = sub, obj, act
		[policy_definition]
		p = sub, obj, act
		[role_definition]
		g = _, _
		[policy_effect]
		e = some(where (p.eft == allow))
		[matchers]
		m = (r.sub == p.sub || g(r.sub, p.sub)) && keyMatch2(r.obj, p.obj) && r.act == p.act
	`)
	if err != nil {
		t.Fatalf("create casbin model: %v", err)
	}

	enforcer, err := casbin.NewSyncedEnforcer(m)
	if err != nil {
		t.Fatalf("create casbin enforcer: %v", err)
	}

	for _, policy := range policies {
		args := make([]interface{}, 0, len(policy))
		for _, item := range policy {
			args = append(args, item)
		}
		if _, err := enforcer.AddPolicy(args...); err != nil {
			t.Fatalf("add casbin policy %v: %v", policy, err)
		}
	}

	return enforcer
}

func assertForbiddenResponse(t *testing.T, recorder *httptest.ResponseRecorder, message string) {
	t.Helper()

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected HTTP 200 envelope, got %d", recorder.Code)
	}

	var response common.Response
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response body: %v", err)
	}
	if response.Code != common.CodeForbidden {
		t.Fatalf("expected response code %d, got %d", common.CodeForbidden, response.Code)
	}
	if response.Message != message {
		t.Fatalf("expected response message %q, got %q", message, response.Message)
	}
}
