package k8s

import (
	"net/http"
	"strings"
	"testing"

	"github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/model"
	"github.com/duanxldragon/pantheon-base/backend/pkg/permissionroutes"
	"github.com/duanxldragon/pantheon-base/backend/pkg/testmysql"
	"github.com/gin-gonic/gin"
)

func TestK8sPermissionRoutesMatchActualActions(t *testing.T) {
	db := testmysql.Open(t)
	router := gin.New()
	InitK8sModule(router.Group("/api/v1"), db, nil)
	actual := make(map[permissionroutes.Route]bool)
	for _, route := range router.Routes() {
		actual[permissionroutes.Route{Method: route.Method, Path: route.Path}] = true
	}
	expectedMethod := map[string][]string{
		"list":      {http.MethodGet},
		"view":      {http.MethodGet},
		"logs":      {http.MethodGet},
		"nodes":     {http.MethodGet},
		"create":    {http.MethodPost},
		"sync":      {http.MethodPost},
		"scale":     {http.MethodPost},
		"restart":   {http.MethodPost},
		"rollback":  {http.MethodPost},
		"reconcile": {http.MethodPost},
		"update":    {http.MethodPut, http.MethodPost},
		"delete":    {http.MethodDelete},
	}
	for key, routes := range permissionroutes.Snapshot() {
		if !strings.HasPrefix(key, "business:k8s:") {
			continue
		}
		action := key[strings.LastIndex(key, ":")+1:]
		allowed, known := expectedMethod[action]
		if !known {
			t.Fatalf("unregistered action suffix for %s", key)
		}
		if len(routes) == 0 {
			t.Fatalf("empty binding for %s", key)
		}
		for _, route := range routes {
			if !actual[route] {
				t.Fatalf("%s binds a route the router does not expose: %+v", key, route)
			}
			matched := false
			for _, method := range allowed {
				if route.Method == method {
					matched = true
					break
				}
			}
			if !matched {
				t.Fatalf("%s binds %s %s; allowed methods %v", key, route.Method, route.Path, allowed)
			}
		}
	}
}

func TestK8sServicePermissionRoutesMatchActualActions(t *testing.T) {
	db := testmysql.Open(t)
	router := gin.New()
	InitK8sModule(router.Group("/api/v1"), db, nil)
	actual := make(map[permissionroutes.Route]bool)
	for _, route := range router.Routes() {
		actual[permissionroutes.Route{Method: route.Method, Path: route.Path}] = true
	}
	const definitionText = `[request_definition]
r = sub, obj, act
[policy_definition]
p = sub, obj, act
[policy_effect]
e = some(where (p.eft == allow))
[matchers]
m = r.sub == p.sub && keyMatch2(r.obj, p.obj) && r.act == p.act`
	for _, tc := range []struct{ suffix, method string }{
		{"list", http.MethodGet}, {"view", http.MethodGet}, {"create", http.MethodPost}, {"update", http.MethodPut}, {"delete", http.MethodDelete},
	} {
		t.Run(tc.suffix, func(t *testing.T) {
			policies := permissionroutes.Lookup("business:k8s:service:" + tc.suffix)
			if len(policies) == 0 {
				t.Fatal("missing provider-owned Service action binding")
			}
			definition, err := model.NewModelFromString(definitionText)
			if err != nil {
				t.Fatal(err)
			}
			enforcer, err := casbin.NewEnforcer(definition)
			if err != nil {
				t.Fatal(err)
			}
			for _, policy := range policies {
				if !actual[policy] || policy.Method != tc.method || !strings.Contains(policy.Path, "/services") {
					t.Fatalf("invalid action route: %+v", policy)
				}
				if _, err := enforcer.AddPolicy("service-actor", policy.Path, policy.Method); err != nil {
					t.Fatal(err)
				}
			}
			for _, route := range []permissionroutes.Route{
				{Method: http.MethodGet, Path: "/api/v1/business/k8s/clusters/:id/services"},
				{Method: http.MethodGet, Path: "/api/v1/business/k8s/clusters/:id/services/:name"},
				{Method: http.MethodPost, Path: "/api/v1/business/k8s/clusters/:id/services"},
				{Method: http.MethodPut, Path: "/api/v1/business/k8s/clusters/:id/services/:name"},
				{Method: http.MethodDelete, Path: "/api/v1/business/k8s/clusters/:id/services/:name"},
			} {
				path := strings.ReplaceAll(strings.ReplaceAll(route.Path, ":id", "7"), ":name", "catalog-api")
				allowed, err := enforcer.Enforce("service-actor", path, route.Method)
				if err != nil || allowed != (route.Method == tc.method) {
					t.Fatalf("action policy mismatch: %+v allowed=%v error=%v", route, allowed, err)
				}
			}
			if allowed, err := enforcer.Enforce("service-actor", "/api/v1/business/k8s/clusters/7/nodes", http.MethodGet); err != nil || allowed {
				t.Fatalf("Service permission widened to cluster nodes: %v %v", allowed, err)
			}
		})
	}
}
