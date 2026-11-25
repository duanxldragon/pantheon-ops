package cmdb

import (
	"errors"
	"os"
	"os/exec"
	"strings"
	"testing"

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
		{"business:cmdb:host:detail", []permissionroutes.Route{{Method: "GET", Path: "/api/v1/business/cmdb/hosts/:id"}}},
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
				if !actualRoutes[policy] || strings.Contains(policy.Path, "*") {
					t.Fatalf("registered noncanonical or wildcard host route: %+v", policy)
				}
			}
			// Base enforces business routes against the Gin route template, where a
			// :param policy segment covers only :param route segments.
			allows := func(request permissionroutes.Route) bool {
				for _, policy := range policies {
					if policy.Method == request.Method && permissionroutes.Covers(request.Path, policy.Path) {
						return true
					}
				}
				return false
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
					if strings.Contains(allowed.Path, ":") && strings.HasPrefix(path, allowed.Path) {
						continue // ":id-evil" is not a real route template
					}
					if allows(permissionroutes.Route{Method: allowed.Method, Path: path}) {
						t.Fatalf("permission widened path to %s", path)
					}
				}
			}
		})
	}
	// Base's static-route guard: a dynamic detail policy must not authorize the
	// CSV endpoints, nor an export policy the detail route.
	for _, path := range []string{"/api/v1/business/cmdb/hosts/export", "/api/v1/business/cmdb/hosts/import-template"} {
		if permissionroutes.Covers(path, "/api/v1/business/cmdb/hosts/:id") {
			t.Fatal("detail policy widened to a static sibling; the Base static-route guard regressed")
		}
	}
	if permissionroutes.Covers("/api/v1/business/cmdb/hosts/:id", "/api/v1/business/cmdb/hosts/export") {
		t.Fatal("export policy widened to the detail route")
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
