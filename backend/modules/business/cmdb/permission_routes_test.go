package cmdb

import (
	"errors"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/model"
	"github.com/casbin/casbin/v2/util"
	"github.com/duanxldragon/pantheon-base/backend/modules/business/cmdb/host"
	"github.com/duanxldragon/pantheon-base/backend/pkg/permissionroutes"
	"github.com/gin-gonic/gin"
)

func TestHostPermissionRoutesKeepListAndCSVActionsSeparate(t *testing.T) {
	for i := 0; i < 2; i++ {
		if err := registerHostPermissionRoutes(); err != nil {
			t.Fatal(err)
		}
	}
	engine := gin.New()
	host.NewHostHandler(nil).RegisterRoutes(engine.Group("/api/v1/business/cmdb"))
	actualRoutes := make(map[permissionroutes.Route]bool)
	for _, route := range engine.Routes() {
		actualRoutes[permissionroutes.Route{Method: route.Method, Path: route.Path}] = true
	}
	seedKeys := make(map[string]bool)
	for _, seed := range hostMenuSeeds() {
		seedKeys[seed.PagePerm] = true
		seedKeys[seed.Perms] = true
	}
	for _, test := range []struct {
		key     string
		allowed []permissionroutes.Route
	}{
		{"business:cmdb:host:view", []permissionroutes.Route{{Method: "GET", Path: "/api/v1/business/cmdb/hosts"}}},
		{"business:cmdb:host:create", []permissionroutes.Route{{Method: "POST", Path: "/api/v1/business/cmdb/hosts"}}},
		{"business:cmdb:host:export", []permissionroutes.Route{{Method: "GET", Path: "/api/v1/business/cmdb/hosts/export"}}},
		{"business:cmdb:host:import", []permissionroutes.Route{{Method: "GET", Path: "/api/v1/business/cmdb/hosts/import-template"}, {Method: "POST", Path: "/api/v1/business/cmdb/hosts/import"}}},
	} {
		t.Run(test.key, func(t *testing.T) {
			if !seedKeys[test.key] {
				t.Fatal("registration invented a permission absent from the current seeds")
			}
			policies := permissionroutes.Lookup(test.key)
			if len(policies) != len(test.allowed) {
				t.Fatalf("unexpected policy count: %+v", policies)
			}
			for _, policy := range policies {
				if !actualRoutes[policy] || strings.ContainsAny(policy.Path, "*:") {
					t.Fatalf("registered noncanonical or wildcard host route: %+v", policy)
				}
			}
			casbinModel, err := model.NewModelFromString(`
[request_definition]
r = sub, obj, act
[policy_definition]
p = sub, obj, act
[policy_effect]
e = some(where (p.eft == allow))
[matchers]
m = r.sub == p.sub && keyMatch2(r.obj, p.obj) && r.act == p.act
`)
			if err != nil {
				t.Fatal(err)
			}
			enforcer, err := casbin.NewEnforcer(casbinModel)
			if err != nil {
				t.Fatal(err)
			}
			for _, policy := range policies {
				if _, err := enforcer.AddPolicy("host-probe", policy.Path, policy.Method); err != nil {
					t.Fatal(err)
				}
			}
			allows := func(request permissionroutes.Route) bool {
				allowed, err := enforcer.Enforce("host-probe", request.Path, request.Method)
				if err != nil {
					t.Fatal(err)
				}
				return allowed
			}
			for _, request := range test.allowed {
				if !allows(request) {
					t.Fatalf("required host action denied: %+v", request)
				}
			}
			for request := range actualRoutes {
				want := false
				for _, allowed := range test.allowed {
					want = want || request == allowed
				}
				if allows(request) != want {
					t.Fatalf("permission crossed action boundary for %+v", request)
				}
			}
			for _, allowed := range test.allowed {
				for _, method := range []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"} {
					if method != allowed.Method && allows(permissionroutes.Route{Method: method, Path: allowed.Path}) {
						t.Fatalf("permission widened HTTP method to %s", method)
					}
				}
				for _, path := range []string{allowed.Path + "-evil", allowed.Path + "/child", strings.Replace(allowed.Path, "/business/cmdb/", "/cmdb/", 1), "/api/v1/business/cmdb/hosts/7", "/api/v1/business/cmdb/groups", "/api/v1/system/user/list"} {
					if allows(permissionroutes.Route{Method: allowed.Method, Path: path}) {
						t.Fatalf("permission widened path to %s", path)
					}
				}
			}
		})
	}
	// This is the current matcher collision: the provider must not add the
	// dynamic detail policy while that policy would authorize CSV endpoints.
	for _, path := range []string{"/api/v1/business/cmdb/hosts/export", "/api/v1/business/cmdb/hosts/import-template"} {
		if !util.KeyMatch2(path, "/api/v1/business/cmdb/hosts/:id") {
			t.Fatal("shared matcher changed; re-evaluate the documented detail gap")
		}
	}
	for _, key := range []string{"business:cmdb:host:viewer", "business:cmdb:host:export-evil", "business:cmdb:host:import-evil", "business:cmdb:host:detail", "business:cmdb:group:detail"} {
		if len(permissionroutes.Lookup(key)) != 0 {
			t.Fatalf("registration invented an adjacent permission: %s", key)
		}
	}
}

func TestHostPermissionRegistrationFailureStopsModuleInitialization(t *testing.T) {
	const flag = "PANTHEON_TEST_CMDB_PERMISSION_CONFLICT"
	if os.Getenv(flag) == "1" {
		if err := permissionroutes.Register("business:cmdb", []permissionroutes.Binding{{Permission: "business:cmdb:host:view", Routes: []permissionroutes.Route{{Method: "GET", Path: "/api/v1/business/cmdb/hosts/export"}}}}); err != nil {
			t.Fatal(err)
		}
		defer func() {
			err, ok := recover().(error)
			if !ok || !errors.Is(err, permissionroutes.ErrConflict) {
				t.Fatalf("module did not stop on the registration conflict: %v", err)
			}
		}()
		InitCmdbModule(nil, nil)
		t.Fatal("module proceeded after a conflicting permission registration")
		return
	}
	command := exec.Command(os.Args[0], "-test.run=^TestHostPermissionRegistrationFailureStopsModuleInitialization$")
	command.Env = append(os.Environ(), flag+"=1")
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("isolated startup conflict test failed: %v\n%s", err, output)
	}
}
