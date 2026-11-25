package cmdb

import (
	"strings"
	"testing"

	"github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/model"
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
			// Base enforces against the route template; a :param policy segment
			// covers only :param route segments, so detail never reaches export.
			for route := range actual {
				want, got := false, false
				for _, policy := range policies {
					want = want || route == policy
					got = got || (policy.Method == route.Method && permissionroutes.Covers(route.Path, policy.Path))
				}
				if got != want {
					t.Fatalf("key grants another action for %s %s: allowed=%v want=%v", route.Method, route.Path, got, want)
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
	if registeredKeys != 22 {
		t.Fatalf("unexpected CMDB active key coverage: %d", registeredKeys)
	}
}

func TestCMDBDetailPoliciesStaySeparateFromStaticSiblings(t *testing.T) {
	for _, prefix := range []string{"hosts", "groups"} {
		base := "/api/v1/business/cmdb/" + prefix
		for _, action := range []string{"export", "import-template"} {
			if permissionroutes.Covers(base+"/"+action, base+"/:id") {
				t.Fatalf("%s detail policy widened to %s; the Base static-route guard regressed", prefix, action)
			}
			if permissionroutes.Covers(base+"/:id", base+"/"+action) {
				t.Fatalf("%s %s policy widened to the detail route", prefix, action)
			}
		}
		if !permissionroutes.Covers(base+"/:id", base+"/:id") {
			t.Fatalf("%s detail policy no longer covers its own route", prefix)
		}
	}
}
