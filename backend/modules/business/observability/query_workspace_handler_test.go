package observability

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/duanxldragon/pantheon-base/backend/pkg/auditpolicy"
	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
	"github.com/duanxldragon/pantheon-base/backend/pkg/contracts/businessauth"
	"github.com/duanxldragon/pantheon-base/backend/pkg/database"
	"github.com/duanxldragon/pantheon-base/backend/pkg/tenant"
	"github.com/duanxldragon/pantheon-base/backend/pkg/testmysql"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type queryTestAuthority struct {
	mu                    sync.Mutex
	denied                bool
	mode                  string
	sourceDenied          bool
	denySourcesAtOwnCheck int
	ownChecks             int
}

func (r *queryTestAuthority) RefreshPrincipal(ctx context.Context, identity businessauth.Identity) (businessauth.Principal, error) {
	return businessauth.Principal{Identity: identity}, nil
}
func (r *queryTestAuthority) Authorize(ctx context.Context, input businessauth.Input) (businessauth.Authorization, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if input.Action.Permission == "" {
		r.ownChecks++
		if r.denySourcesAtOwnCheck != 0 && r.ownChecks >= r.denySourcesAtOwnCheck {
			r.sourceDenied = true
		}
	}
	if r.denied || (r.sourceDenied && input.Action.Permission != "") || ctx.Err() != nil {
		return businessauth.Authorization{}, businessauth.ErrDenied
	}
	return businessauth.Authorization{DataScope: &common.DataScopeReq{UserID: input.Identity.ActorID, IsAdmin: true}, Bindings: []businessauth.RoleBinding{{TenantID: input.ResourceTenantID, RoleKey: "query-reader"}}}, nil
}
func (r *queryTestAuthority) ReadMode(ctx context.Context) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if ctx.Err() != nil {
		return "", businessauth.ErrDenied
	}
	return r.mode, nil
}
func (r *queryTestAuthority) revoke() { r.mu.Lock(); defer r.mu.Unlock(); r.denied = true }
func newQueryTestRouter(t *testing.T, db *gorm.DB, service *DatasourceService, owner QueryOwner, authority *queryTestAuthority) *gin.Engine {
	t.Helper()
	if err := RegisterQueryPolicies(); err != nil {
		t.Fatal(err)
	}
	rdb := redis.NewClient(&redis.Options{Addr: "127.0.0.1:26316"})
	if rdb.Ping(context.Background()).Err() != nil {
		t.Fatal("required Redis unavailable")
	}
	old := database.RDB
	database.RDB = rdb
	t.Cleanup(func() { database.RDB = old; _ = rdb.Close() })
	if err := db.AutoMigrate(&QueryHistory{}, &SavedView{}, &QueryOwnerBudget{}); err != nil {
		t.Fatal(err)
	}
	router := gin.New()
	router.Use(func(c *gin.Context) {
		tc := tenant.Context{TenantID: owner.TenantID, Mode: tenant.ModeMulti, ResolvedBy: tenant.ResolvedBySubject}
		tenant.SetGin(c, &tc)
		c.Set("userID", owner.ActorID)
		input := businessauth.Input{Identity: businessauth.Identity{ActorID: owner.ActorID}, Resolution: tc, ResourceTenantID: owner.TenantID, DataScope: true,
			Action: businessauth.Action{Method: c.Request.Method, Path: c.Request.URL.Path, RouteTemplate: c.FullPath()}}
		businessauth.BindGin(c, authority, authority, input)
		c.Next()
	})
	handler := NewObservabilityHandler(service, nil, nil, nil)
	group := router.Group(observabilityAPIPrefix)
	handler.RegisterDatasourceRoutes(group)
	handler.RegisterWorkspaceRoutes(group)
	return router
}
func TestQueryResponseReleaseRevocationAndSourceRevision(t *testing.T) {
	for _, condition := range []string{"current-read-port-denied", "source-revision-changed"} {
		t.Run(condition, func(t *testing.T) {
			db := testmysql.Open(t)
			if err := db.AutoMigrate(&PrometheusDatasource{}); err != nil {
				t.Fatal(err)
			}
			entered, release := make(chan struct{}), make(chan struct{})
			provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost || r.URL.RawQuery != "" {
					t.Error("raw query generated provider URL")
				}
				close(entered)
				<-release
				_, _ = w.Write([]byte(`{"status":"success","data":{"result":[{"value":[1,"SECRET_PROVIDER_SENTINEL"]}]}}`))
			}))
			defer provider.Close()
			source := PrometheusDatasource{TenantID: 11, Name: "test-source", Type: "prometheus", URL: provider.URL}
			if err := db.Create(&source).Error; err != nil {
				t.Fatal(err)
			}
			ds := NewDatasourceService(db)
			ds.ssrfValidator = testDatasourceValidator()
			authority := &queryTestAuthority{mode: tenant.ModeMulti}
			router := newQueryTestRouter(t, db, ds, QueryOwner{TenantID: 11, ActorID: 31}, authority)
			recorder := httptest.NewRecorder()
			done := make(chan struct{})
			go func() {
				router.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, observabilityAPIPrefix+"/datasources/"+strconv.FormatUint(source.ID, 10)+"/query",
					strings.NewReader(`{"rawQuery":"SECRET_RAW_SENTINEL","time":{"mode":"relative","duration":"15m"},"limit":100,"resultMode":"table"}`)))
				close(done)
			}()
			select {
			case <-entered:
			case <-time.After(3 * time.Second):
				close(release)
				t.Fatal("provider never entered")
			}
			if condition == "current-read-port-denied" {
				authority.revoke()
			} else {
				if err := db.Model(&source).Update("revision", source.Revision+1).Error; err != nil {
					close(release)
					t.Fatal(err)
				}
			}
			close(release)
			select {
			case <-done:
			case <-time.After(3 * time.Second):
				t.Fatal("provider result outlived bounded work")
			}
			if recorder.Code == http.StatusOK || strings.Contains(recorder.Body.String(), "SECRET_") {
				t.Fatal("revoked provider result leaked")
			}
			var count int64
			db.Model(&QueryHistory{}).Count(&count)
			if count != 0 {
				t.Fatal("revoked provider result metadata persisted")
			}
			if n := database.RDB.ZCard(context.Background(), "pantheon:o4:permit:{11}:actor:31").Val(); n != 0 {
				t.Fatal("stopped worker kept actor permit")
			}
		})
	}
}
func TestQueryPrivacyDenialAndRestoreNeverExecutes(t *testing.T) {
	db := testmysql.Open(t)
	if err := db.AutoMigrate(&PrometheusDatasource{}); err != nil {
		t.Fatal(err)
	}
	source := PrometheusDatasource{TenantID: 11, Name: "test-source", Type: "prometheus", URL: "http://127.0.0.1:1"}
	if err := db.Create(&source).Error; err != nil {
		t.Fatal(err)
	}
	ds := NewDatasourceService(db)
	authority := &queryTestAuthority{mode: tenant.ModeMulti}
	router := newQueryTestRouter(t, db, ds, QueryOwner{TenantID: 11, ActorID: 31}, authority)
	service, _ := NewQueryWorkspaceService(db, QueryOwner{TenantID: 11, ActorID: 31}, &common.DataScopeReq{IsAdmin: true})
	view, err := service.Save(context.Background(), source.ID, 0, 0, safeMetricRecipe())
	if err != nil {
		t.Fatal(err)
	}
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, observabilityAPIPrefix+"/saved-views/"+strconv.FormatUint(view.ID, 10)+"/restore", nil))
	if recorder.Code != http.StatusOK || strings.Contains(recorder.Body.String(), "expression") {
		t.Fatal("restore executed or exposed raw expression")
	}
	authority.revoke()
	for _, path := range []string{"/datasources/1/query", "/datasources/1/query-fields", "/saved-views/1/restore"} {
		recorder := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, observabilityAPIPrefix+path, strings.NewReader(`{"rawQuery":"SECRET_BODY_SENTINEL"}`))
		req.Header.Set("X-Request-ID", "SECRET_REQUEST_SENTINEL")
		router.ServeHTTP(recorder, req)
		if recorder.Code == http.StatusOK || strings.Contains(recorder.Body.String(), "SECRET_") {
			t.Fatal("denial leaked user input")
		}
	}
	// All adopted route declarations are exact. Nearby unmatched routes retain
	// their existing audit/idempotency strategy.
	router.GET("/api/v1/business/observability/datasources/:id/query-extra", func(c *gin.Context) {
		if IsProtectedQueryRoute(c) {
			t.Error("prefix matcher broadened protected slice")
		}
		c.Status(http.StatusNoContent)
	})
}
func TestProtectedRouteDeclarationsCoverRegisteredSlice(t *testing.T) {
	router := gin.New()
	handler := NewObservabilityHandler(nil, nil, nil, nil)
	for _, prefix := range []string{observabilityAPIPrefix, "/api/v1/observability"} {
		handler.RegisterDatasourceRoutes(router.Group(prefix))
	}
	handler.RegisterWorkspaceRoutes(router.Group(observabilityAPIPrefix))
	actual := map[auditpolicy.Route]bool{}
	for _, route := range router.Routes() {
		actual[auditpolicy.Route{Method: route.Method, Template: route.Path}] = true
	}
	for _, route := range ProtectedQueryRoutes() {
		if !actual[route] {
			t.Fatalf("declared route missing: %+v", route)
		}
	}
}
