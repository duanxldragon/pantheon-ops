package deploy

import (
	"strings"
	"testing"

	"github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/model"
	"github.com/duanxldragon/pantheon-base/backend/pkg/permissionroutes"
	"github.com/gin-gonic/gin"
)

// deployPermKeys is the deploy permission set bound by
// registerDeployPermissionRoutes. It mirrors the seeded F menu perms in
// deploy_seed.go (§seedDeployMenus), excluding business:deploy:task:export,
// which log_download.go registers separately. A drift here fails the test
// instead of leaving a page/API silent gap.
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
	"business:deploy:task:detail",
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
			// other registered route. Base enforces against the route template,
			// where a :param policy segment covers only :param route segments.
			for route := range actual {
				want, got := false, false
				for _, policy := range policies {
					want = want || route == policy
					got = got || (policy.Method == route.Method && permissionroutes.Covers(route.Path, policy.Path))
				}
				if got != want {
					t.Fatalf("key %s grants another action for %s %s: allowed=%v want=%v", key, route.Method, route.Path, got, want)
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

// TestDeployDetailPoliciesStaySeparateFromStaticSiblings pins the Base
// static-route guard for deploy: a detail grant rides on its view key but never
// reaches the CSV export sibling, and the export grant never reaches the detail.
func TestDeployDetailPoliciesStaySeparateFromStaticSiblings(t *testing.T) {
	if err := registerDeployPermissionRoutes(); err != nil {
		t.Fatal(err)
	}
	const base = "/api/v1/business/deploy/"
	grants := func(key, method, route string) bool {
		for _, policy := range permissionroutes.Lookup(key) {
			if policy.Method == method && permissionroutes.Covers(route, policy.Path) {
				return true
			}
		}
		return false
	}
	for _, tc := range []struct {
		key, method, route string
		want               bool
	}{
		{"business:deploy:package:view", "GET", base + "packages", true},
		{"business:deploy:package:view", "GET", base + "packages/:id", true},
		{"business:deploy:package:view", "GET", base + "packages/export", false},
		{"business:deploy:package:export", "GET", base + "packages/:id", false},
		{"business:deploy:template:list", "GET", base + "templates/:id", true},
		{"business:deploy:template:list", "GET", base + "templates/export", false},
		{"business:deploy:template:export", "GET", base + "templates/:id", false},
		{"business:deploy:task:detail", "GET", base + "tasks/:id", true},
		{"business:deploy:task:detail", "GET", base + "tasks/:id/logs", true},
		{"business:deploy:task:detail", "GET", base + "tasks/:id/events", true},
		{"business:deploy:task:detail", "GET", base + "tasks/:id/events/history", true},
		{"business:deploy:task:detail", "GET", base + "tasks/:id/rollback", true},
		{"business:deploy:task:detail", "GET", base + "tasks/export", false},
		{"business:deploy:task:detail", "POST", base + "tasks/:id/rollback", false},
		{"business:deploy:task:view", "GET", base + "tasks/:id", false},
	} {
		if got := grants(tc.key, tc.method, tc.route); got != tc.want {
			t.Fatalf("%s on %s %s: allowed=%v want=%v", tc.key, tc.method, tc.route, got, tc.want)
		}
	}
}
