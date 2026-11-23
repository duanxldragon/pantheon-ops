package deploy

import (
	"strings"
	"testing"

	"github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/model"
	"github.com/casbin/casbin/v2/util"
	"github.com/duanxldragon/pantheon-base/backend/pkg/permissionroutes"
	"github.com/gin-gonic/gin"
)

// deployPermKeys is the deploy permission set bound by
// registerDeployPermissionRoutes. It mirrors the seeded F menu perms in
// deploy_seed.go (§seedDeployMenus), excluding two keys that are seeded but
// intentionally unbound: business:deploy:task:export (log_download.go
// registers it separately) and business:deploy:task:detail (the GET detail
// route stays denied until Base ships the keyMatch2 static-route guard).
// A drift here fails the test instead of leaving a page/API silent gap.
var deployPermKeys = []string{
	"business:deploy:package:view",
	"business:deploy:package:create",
	"business:deploy:package:update",
	"business:deploy:package:delete",
	"business:deploy:package:export",
	"business:deploy:package:import",
	"business:deploy:template:list",
	"business:deploy:template:create",
	"business:deploy:template:update",
	"business:deploy:template:delete",
	"business:deploy:template:export",
	"business:deploy:template:import",
	"business:deploy:task:view",
	"business:deploy:task:create",
	"business:deploy:task:update",
	"business:deploy:task:delete",
	"business:deploy:task:start",
	"business:deploy:task:pause",
	"business:deploy:task:resume",
	"business:deploy:task:retry",
	"business:deploy:task:cancel",
	"business:deploy:task:mark-result",
	"business:deploy:credential:list",
	"business:deploy:credential:create",
	"business:deploy:credential:update",
	"business:deploy:credential:delete",
}

// casbinModel matches pkg/database/casbin.go's released model exactly. Testing
// against this public matcher contract (not constants/helpers) is what locks in
// the allowed/denied/nonexistent/cross-permission matrix.
const casbinModel = `[request_definition]
r = sub, obj, act
[policy_definition]
p = sub, obj, act
[policy_effect]
e = some(where (p.eft == allow))
[matchers]
m = r.sub == p.sub && keyMatch2(r.obj, p.obj) && r.act == p.act`

func TestDeployRegisteredActionsDoNotGrantOtherEndpoints(t *testing.T) {
	if err := registerDeployPermissionRoutes(); err != nil {
		t.Fatal(err)
	}
	router := gin.New()
	routes := router.Group("/api/v1/business/deploy")
	NewDeployHandler(nil).RegisterRoutes(routes)
	actual := make(map[permissionroutes.Route]bool)
	for _, route := range router.Routes() {
		actual[permissionroutes.Route{Method: route.Method, Path: route.Path}] = true
	}
	registeredKeys := 0
	for _, key := range deployPermKeys {
		policies := permissionroutes.Lookup(key)
		if len(policies) == 0 {
			t.Fatalf("safe existing deploy key has no API bindings: %s", key)
		}
		registeredKeys++
		t.Run(key, func(t *testing.T) {
			m, err := model.NewModelFromString(casbinModel)
			if err != nil {
				t.Fatal(err)
			}
			enforcer, err := casbin.NewEnforcer(m)
			if err != nil {
				t.Fatal(err)
			}
			for _, policy := range policies {
				if !actual[policy] {
					t.Fatalf("key maps a route absent from the actual handlers: %+v", policy)
				}
				if _, err := enforcer.AddPolicy("scoped-role", policy.Path, policy.Method); err != nil {
					t.Fatal(err)
				}
			}
			// This key's policy set must grant exactly its own endpoints and no
			// other registered route (the keyMatch2 anti-widening check).
			for route := range actual {
				path := strings.ReplaceAll(route.Path, ":id", "7")
				want := false
				for _, policy := range policies {
					want = want || route == policy
				}
				allowed, err := enforcer.Enforce("scoped-role", path, route.Method)
				if err != nil || allowed != want {
					t.Fatalf("key %s grants another action for %s %s: allowed=%v want=%v error=%v", key, route.Method, path, allowed, want, err)
				}
			}
			// Role revocation removes the grant; a pre-existing explicit wildcard
			// role policy must be untouched.
			if _, err := enforcer.RemoveFilteredPolicy(0, "scoped-role"); err != nil {
				t.Fatal(err)
			}
			for _, policy := range policies {
				path := strings.ReplaceAll(policy.Path, ":id", "7")
				allowed, err := enforcer.Enforce("scoped-role", path, policy.Method)
				if err != nil || allowed {
					t.Fatalf("removed role grant still authorizes action: %+v (%v)", policy, err)
				}
				if _, err := enforcer.AddPolicy("manual-role", "/api/v1/business/deploy/*", policy.Method); err != nil {
					t.Fatal(err)
				}
			}
		})
	}
	if registeredKeys != len(deployPermKeys) {
		t.Fatalf("unexpected deploy key coverage: %d", registeredKeys)
	}
}

// TestDeployPackageDetailRemainsUnregisteredForView locks in the preserved
// gap: `GET /packages/:id` is intentionally not bound to the view key because
// the released keyMatch2 matcher would widen it to `GET /packages/export`
// (CSV export), granting export to every package viewer. Until Base ships a
// static-route guard this detail route stays denied for scoped roles.
func TestDeployPackageDetailRemainsUnregisteredForView(t *testing.T) {
	if err := registerDeployPermissionRoutes(); err != nil {
		t.Fatal(err)
	}
	view := permissionroutes.Lookup("business:deploy:package:view")
	for _, policy := range view {
		if strings.HasSuffix(policy.Path, "/:id") {
			t.Fatalf("unsafe GET detail policy registered without released static guard: %s", policy.Path)
		}
		// Confirms the Base-first dependency actually holds on the released matcher.
		if !util.KeyMatch2("/api/v1/business/deploy/packages/export", policy.Path) {
			continue
		}
		t.Fatalf("view policy %s would widen to /packages/export; do NOT register", policy.Path)
	}
	m, err := model.NewModelFromString(casbinModel)
	if err != nil {
		t.Fatal(err)
	}
	enforcer, err := casbin.NewEnforcer(m)
	if err != nil {
		t.Fatal(err)
	}
	for _, policy := range view {
		if _, err := enforcer.AddPolicy("scoped-role", policy.Path, policy.Method); err != nil {
			t.Fatal(err)
		}
	}
	// Detail route must stay denied (preserved gap) and export must NOT be
	// reachable via the view grant.
	for _, tc := range []struct{ method, path string }{
		{"GET", "/api/v1/business/deploy/packages/7"},
		{"GET", "/api/v1/business/deploy/packages/export"},
		{"GET", "/api/v1/business/deploy/packages/7/nonexistent"},
	} {
		allowed, err := enforcer.Enforce("scoped-role", tc.path, tc.method)
		if err != nil || allowed {
			t.Fatalf("view grant must NOT authorize %s %s: allowed=%v err=%v", tc.method, tc.path, allowed, err)
		}
	}
	if allowed, err := enforcer.Enforce("scoped-role", "/api/v1/business/deploy/packages", "GET"); err != nil || !allowed {
		t.Fatalf("view grant must authorize list GET /packages: allowed=%v err=%v", allowed, err)
	}
}
