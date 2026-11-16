package cmdb

import (
	"strings"
	"testing"

	"github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/model"
	"github.com/casbin/casbin/v2/util"
	"github.com/duanxldragon/pantheon-base/backend/modules/business/cmdb/group"
	"github.com/duanxldragon/pantheon-base/backend/modules/business/cmdb/host"
	"github.com/duanxldragon/pantheon-base/backend/modules/business/cmdb/label"
	"github.com/duanxldragon/pantheon-base/backend/pkg/permissionroutes"
	"github.com/gin-gonic/gin"
)

func TestCMDBRegisteredActionsDoNotGrantOtherEndpoints(t *testing.T) {
	if err := registerHostPermissionRoutes(); err != nil {
		t.Fatal(err)
	}
	router := gin.New()
	routes := router.Group("/api/v1/business/cmdb")
	host.NewHostHandler(nil).RegisterRoutes(routes)
	group.NewGroupHandler(nil).RegisterRoutes(routes)
	label.NewLabelHandler(nil).RegisterRoutes(routes)
	actual := make(map[permissionroutes.Route]bool)
	for _, route := range router.Routes() {
		actual[permissionroutes.Route{Method: route.Method, Path: route.Path}] = true
	}
	registeredKeys := 0
	for _, seed := range hostMenuSeeds() {
		key := seed.Perms
		if key == "" {
			key = seed.PagePerm
		}
		policies := permissionroutes.Lookup(key)
		if key == "business:cmdb:host:detail" || key == "business:cmdb:group:detail" {
			if len(policies) != 0 {
				t.Fatalf("unsafe GET detail policy registered without released static guard: %s", key)
			}
			continue
		}
		if len(policies) == 0 {
			t.Fatalf("safe existing CMDB key has no API bindings: %s", key)
		}
		registeredKeys++
		t.Run(key, func(t *testing.T) {
			m, err := model.NewModelFromString(`[request_definition]
r = sub, obj, act
[policy_definition]
p = sub, obj, act
[policy_effect]
e = some(where (p.eft == allow))
[matchers]
m = r.sub == p.sub && keyMatch2(r.obj, p.obj) && r.act == p.act`)
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
			for route := range actual {
				path := strings.ReplaceAll(route.Path, ":id", "7")
				want := false
				for _, policy := range policies {
					want = want || route == policy
				}
				allowed, err := enforcer.Enforce("scoped-role", path, route.Method)
				if err != nil || allowed != want {
					t.Fatalf("key grants another action for %s %s: allowed=%v want=%v error=%v", route.Method, path, allowed, want, err)
				}
			}
			if _, err := enforcer.RemoveFilteredPolicy(0, "scoped-role"); err != nil {
				t.Fatal(err)
			}
			for _, policy := range policies {
				path := strings.ReplaceAll(policy.Path, ":id", "7")
				allowed, err := enforcer.Enforce("scoped-role", path, policy.Method)
				if err != nil || allowed {
					t.Fatalf("removed role grant still authorizes action: %+v (%v)", policy, err)
				}
				if _, err := enforcer.AddPolicy("manual-role", "/api/v1/business/cmdb/*", policy.Method); err != nil {
					t.Fatal(err)
				}
				allowed, err = enforcer.Enforce("manual-role", path, policy.Method)
				if err != nil || !allowed {
					t.Fatalf("existing explicit wildcard role policy changed: %+v (%v)", policy, err)
				}
			}
		})
	}
	if registeredKeys != 20 {
		t.Fatalf("unexpected CMDB active key coverage: %d", registeredKeys)
	}
}

func TestCMDBDetailBindingRemainsBlockedByReleasedMatcher(t *testing.T) {
	for _, prefix := range []string{"hosts", "groups"} {
		for _, action := range []string{"export", "import-template"} {
			if !util.KeyMatch2("/api/v1/business/cmdb/"+prefix+"/"+action, "/api/v1/business/cmdb/"+prefix+"/:id") {
				t.Fatal("released matcher changed; reevaluate Base-first detail dependency")
			}
		}
	}
}
