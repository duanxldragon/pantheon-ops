package bizscope

import (
	"testing"

	"github.com/duanxldragon/pantheon-base/backend/pkg/permissionroutes"
)

func TestEnvironmentPermissionBindingsStayInsideProviderAndSeparateActions(t *testing.T) {
	if err := registerBizScopePermissionRoutes(); err != nil {
		t.Fatal(err)
	}
	if err := registerBizScopePermissionRoutes(); err != nil {
		t.Fatalf("repeated registration: %v", err)
	}
	for _, test := range []struct {
		key      string
		expected []permissionroutes.Route
	}{
		{"host:view", []permissionroutes.Route{{Method: "GET", Path: "/api/v1/business/bizscope/options"}}},
		{"environment:list", []permissionroutes.Route{{Method: "GET", Path: "/api/v1/business/bizscope/environments"}, {Method: "GET", Path: "/api/v1/business/bizscope/options"}}},
		{"environment:view", []permissionroutes.Route{{Method: "GET", Path: "/api/v1/business/bizscope/environments/:id"}}},
		{"environment:create", []permissionroutes.Route{{Method: "POST", Path: "/api/v1/business/bizscope/environments"}, {Method: "GET", Path: "/api/v1/business/bizscope/options"}}},
		{"environment:update", []permissionroutes.Route{{Method: "PUT", Path: "/api/v1/business/bizscope/environments/:id"}, {Method: "GET", Path: "/api/v1/business/bizscope/options"}}},
		{"environment:delete", []permissionroutes.Route{{Method: "DELETE", Path: "/api/v1/business/bizscope/environments/:id"}}},
	} {
		t.Run(test.key, func(t *testing.T) {
			got := permissionroutes.Lookup("business:bizscope:" + test.key)
			if len(got) != len(test.expected) {
				t.Fatalf("bindings: %+v want %+v", got, test.expected)
			}
			want := map[permissionroutes.Route]bool{}
			for _, route := range test.expected {
				want[route] = true
			}
			for _, route := range got {
				if !want[route] {
					t.Fatalf("unexpected privilege: %+v", route)
				}
			}
		})
	}
	if routes := permissionroutes.Lookup("business:bizscope:view"); len(routes) != 0 {
		t.Fatalf("introduced unsafe dynamic root policy: %+v", routes)
	}
}
