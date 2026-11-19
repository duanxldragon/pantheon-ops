package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/model"
	bizcap "github.com/duanxldragon/pantheon-base/backend/modules/business/capability"
	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
	"github.com/duanxldragon/pantheon-base/backend/pkg/permissionroutes"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func TestServiceCatalogDetailRegisteredAndScoped(t *testing.T) {
	m, instance, scope := serviceInstanceUpdateFixture(t)
	for _, tc := range []struct {
		name  string
		scope *common.DataScopeReq
		code  int
	}{
		{"allowed", scope, common.CodeSuccess},
		{"other-department", &common.DataScopeReq{Mode: common.DataScopeModeDept, DeptID: 8}, common.CodeNotFound},
	} {
		t.Run(tc.name, func(t *testing.T) {
			router := gin.New()
			group := router.Group("/api/v1/business/service")
			group.Use(func(c *gin.Context) { c.Set(common.DataScopeContextKey, tc.scope) })
			NewHandler(m).RegisterRoutes(group)
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/business/service/services/%d", instance.ServiceID), nil))
			if recorder.Code != http.StatusOK {
				t.Fatalf("service detail route is unavailable: HTTP %d, %s", recorder.Code, recorder.Body.String())
			}
			var response common.Response
			if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
				t.Fatal(err)
			}
			if response.Code != tc.code {
				t.Fatalf("service detail scoped response: %+v", response)
			}
			if tc.code != common.CodeSuccess {
				if response.Data != nil {
					t.Fatalf("out-of-scope service detail leaked data: %+v", response)
				}
				return
			}
			data, ok := response.Data.(map[string]any)
			if !ok || data["runtimeType"] != RuntimeTypeHybrid || data["businessScopeId"] != float64(10) || data["instanceCount"] != float64(1) || data["applicationId"] != float64(instance.ApplicationID) {
				t.Fatalf("service detail does not expose the catalog DTO: %+v", response)
			}
			if _, rawModel := data["RuntimeType"]; rawModel {
				t.Fatal("service detail exposed a Go model instead of camelCase response fields")
			}
		})
	}
}

const serviceCatalogReadCasbinModel = `[request_definition]
r = sub, obj, act
[policy_definition]
p = sub, obj, act
[policy_effect]
e = some(where (p.eft == allow))
[matchers]
m = r.sub == p.sub && keyMatch2(r.obj, p.obj) && r.act == p.act`

func TestServiceCatalogReadGrantOnlyActualGETRoutes(t *testing.T) {
	m, instance, scope := serviceInstanceUpdateFixture(t)
	startup := gin.New()
	InitServiceModule(startup.Group("/api/v1"), m.db, Dependencies{BizScopeReader: m.bizScope, CMDBReader: m.cmdbReader, K8sReader: m.k8sReader})
	actual := make(map[permissionroutes.Route]bool)
	for _, route := range startup.Routes() {
		actual[permissionroutes.Route{Method: route.Method, Path: route.Path}] = true
	}
	for _, permission := range []string{"business:service:list", "business:service:view"} {
		t.Run(permission, func(t *testing.T) {
			policies := permissionroutes.Lookup(permission)
			if len(policies) == 0 {
				t.Fatalf("service read permission has no owner route bindings: %s", permission)
			}
			definition, err := model.NewModelFromString(serviceCatalogReadCasbinModel)
			if err != nil {
				t.Fatal(err)
			}
			enforcer, err := casbin.NewEnforcer(definition)
			if err != nil {
				t.Fatal(err)
			}
			for _, policy := range policies {
				if policy.Method != http.MethodGet || !actual[policy] {
					t.Fatalf("read grant maps a mutation or nonexistent handler: %+v", policy)
				}
				if _, err := enforcer.AddPolicy("catalog-reader", policy.Path, policy.Method); err != nil {
					t.Fatal(err)
				}
			}
			for route := range actual {
				path := strings.ReplaceAll(route.Path, ":id", "7")
				allowed, err := enforcer.Enforce("catalog-reader", path, route.Method)
				if err != nil || allowed != (route.Method == http.MethodGet) {
					t.Fatalf("read grant mismatch for %s %s: allowed=%v, error=%v", route.Method, path, allowed, err)
				}
			}
			router := gin.New()
			group := router.Group("/api/v1/business/service")
			group.Use(func(c *gin.Context) {
				allowed, err := enforcer.Enforce("catalog-reader", c.Request.URL.Path, c.Request.Method)
				if err != nil || !allowed {
					common.Fail(c, common.CodeForbidden, "permission.denied")
					c.Abort()
					return
				}
				c.Set(common.DataScopeContextKey, scope)
			})
			NewHandler(m).RegisterRoutes(group)
			before := instanceUpdateFacts(t, m, instance.ID)
			for _, method := range []string{http.MethodGet, http.MethodPut} {
				recorder := httptest.NewRecorder()
				router.ServeHTTP(recorder, httptest.NewRequest(method, fmt.Sprintf("/api/v1/business/service/services/%d", instance.ServiceID), nil))
				var response common.Response
				if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
					t.Fatal(err)
				}
				want := common.CodeSuccess
				if method == http.MethodPut {
					want = common.CodeForbidden
				}
				if response.Code != want {
					t.Fatalf("read role HTTP %s returned %+v", method, response)
				}
			}
			if instanceUpdateFacts(t, m, instance.ID) != before {
				t.Fatal("read role request changed instance facts")
			}
		})
	}
}

type catalogRuntimeContextKey struct{}

type blockedCatalogK8sReader struct {
	entered chan struct{}
	release chan struct{}
	once    sync.Once
}

func (r *blockedCatalogK8sReader) ResolveTarget(ctx context.Context, _ bizcap.K8sTargetQuery) (bizcap.K8sTargetRef, error) {
	r.once.Do(func() { close(r.entered) })
	select {
	case <-r.release:
		return bizcap.K8sTargetRef{ClusterID: 99, Namespace: "prod", WorkloadKind: "Deployment", WorkloadName: "api", BusinessScopeID: 10}, nil
	case <-ctx.Done():
		return bizcap.K8sTargetRef{}, ctx.Err()
	}
}

func TestServiceCatalogRuntimeEditSerializesWithRebind(t *testing.T) {
	m, instance, scope := serviceInstanceUpdateFixture(t)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	targetReader := &blockedCatalogK8sReader{entered: make(chan struct{}), release: make(chan struct{})}
	m.k8sReader = targetReader
	runtimeReady := make(chan struct{})
	var once sync.Once
	notifyRuntime := func(db *gorm.DB) {
		if db.Statement.Context.Value(catalogRuntimeContextKey{}) != true || db.Statement.Table != "biz_service" {
			return
		}
		if lock, ok := db.Statement.Clauses["FOR"].Expression.(clause.Locking); ok && lock.Strength == "UPDATE" {
			once.Do(func() { close(runtimeReady) })
		}
	}
	if err := m.db.Callback().Query().Before("gorm:query").Register("catalog_runtime_lock_wait", notifyRuntime); err != nil {
		t.Fatal(err)
	}
	if err := m.db.Callback().Update().Before("gorm:update").Register("catalog_runtime_write_wait", func(db *gorm.DB) {
		if db.Statement.Context.Value(catalogRuntimeContextKey{}) == true && db.Statement.Table == "biz_service" {
			once.Do(func() { close(runtimeReady) })
		}
	}); err != nil {
		t.Fatal(err)
	}
	defer m.db.Callback().Query().Remove("catalog_runtime_lock_wait")
	defer m.db.Callback().Update().Remove("catalog_runtime_write_wait")
	rebindDone := make(chan error, 1)
	go func() {
		_, err := m.UpdateInstanceWithContext(ctx, instance.ID, decodeCatalogRebind(instance.LifecycleVersion), "2", scope)
		rebindDone <- err
	}()
	select {
	case <-targetReader.entered:
	case <-ctx.Done():
		t.Fatal("target rebind did not reach its owner-read gate")
	}
	runtimeOwner := *m
	runtimeOwner.db = m.db.WithContext(context.WithValue(ctx, catalogRuntimeContextKey{}, true))
	runtimeDone := make(chan error, 1)
	go func() {
		runtimeType := RuntimeTypeVM
		_, err := runtimeOwner.UpdateService(instance.ServiceID, UpdateServiceRequest{RuntimeType: &runtimeType}, "3", scope)
		runtimeDone <- err
	}()
	select {
	case <-runtimeReady:
	case <-ctx.Done():
		t.Fatal("runtime edit did not reach its Service lock/write gate")
	}
	close(targetReader.release)
	if err := <-rebindDone; err != nil {
		t.Fatalf("controlled rebind failed: %v", err)
	}
	if err := <-runtimeDone; err == nil || err.Error() != errTargetInvalid {
		t.Fatalf("runtime edit admitted an incompatible instance after waiting: %v", err)
	}
	service, err := m.GetService(context.Background(), instance.ServiceID, scope)
	if err != nil || service.RuntimeType != RuntimeTypeHybrid {
		t.Fatalf("runtime refusal did not preserve compatible service: %+v, %v", service, err)
	}
	stored, err := m.GetInstance(context.Background(), instance.ID, scope)
	if err != nil || stored.TargetType != TargetTypeK8s || stored.LifecycleVersion != 1 {
		t.Fatalf("runtime refusal damaged the accepted rebind: %+v, %v", stored, err)
	}
}

func decodeCatalogRebind(version uint64) UpdateInstanceRequest {
	return UpdateInstanceRequest{ExpectedLifecycleVersion: &version, Target: &InstanceTargetRequest{TargetType: TargetTypeK8s, K8sClusterID: 99, Namespace: "prod", WorkloadKind: "Deployment", WorkloadName: "api"}}
}

func TestServiceCatalogK8sKnownNotFoundAndFaultCause(t *testing.T) {
	fault := errors.New("injected K8s owner connection failure")
	for _, tc := range []struct {
		name string
		err  error
		code int
	}{
		{"known-not-found", errors.New("business.service.k8s_target_not_found"), common.CodeNotFound},
		{"owner-fault", fmt.Errorf("owner lookup failed: %w", fault), common.CodeError},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m, instance, scope := serviceInstanceUpdateFixture(t)
			m.k8sReader = instanceUpdateK8sReader{err: tc.err}
			before := instanceUpdateFacts(t, m, instance.ID)
			_, err := m.UpdateInstance(instance.ID, decodeCatalogRebind(0), "2", scope)
			if tc.code == common.CodeNotFound {
				if err == nil || err.Error() != errTargetNotFound {
					t.Fatalf("known K8s not-found did not become canonical target refusal: %v", err)
				}
			} else if !errors.Is(err, fault) {
				t.Fatalf("unexpected K8s owner failure lost its cause: %v", err)
			}
			response := serviceInstanceUpdateHTTP(context.Background(), t, m, instance.ID, map[string]any{
				"expectedLifecycleVersion": 0, "desiredVersion": "must-not-persist",
				"target": map[string]any{"targetType": TargetTypeK8s, "k8sClusterId": 99, "namespace": "prod", "workloadKind": "Deployment", "workloadName": "api"},
			})
			if response.Code != tc.code {
				t.Fatalf("K8s owner HTTP refusal was misclassified: %+v", response)
			}
			if instanceUpdateFacts(t, m, instance.ID) != before {
				t.Fatal("K8s owner refusal changed persisted facts")
			}
		})
	}
}
