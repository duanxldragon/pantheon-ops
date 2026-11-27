package argocd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/duanxldragon/pantheon-base/backend/modules/business/shared/tenantfixture"
	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
	"github.com/duanxldragon/pantheon-base/backend/pkg/permissionroutes"
	"github.com/duanxldragon/pantheon-base/backend/pkg/tenant"
	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type argoCDTestMode struct {
	mode string
	err  error
}

func (m argoCDTestMode) ReadMode(context.Context) (string, error) { return m.mode, m.err }

func TestArgoCDTenantRegistryAndSync(t *testing.T) {
	f := tenantfixture.Open(t)
	svc := NewArgoCDService(f.DB)
	requireArgoCDSuccess(t, svc.Migrate())
	var creates, syncs atomic.Int32
	remote := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/applications":
			creates.Add(1)
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/applications/app-a/sync":
			syncs.Add(1)
		default:
			t.Errorf("unexpected provider request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusBadRequest)
		}
	}))
	defer remote.Close()
	svc.newClient = func(string, string, string) (*ArgoCDClient, error) {
		return &ArgoCDClient{endpoint: remote.URL, httpClient: remote.Client()}, nil
	}
	a := svc.WithTenantContext(context.Background(), f.TenantA)
	b := svc.WithTenantContext(context.Background(), f.TenantB)
	compat := svc.WithTenantContext(context.Background(), f.Compat)
	server, err := a.AddServer(AddServerRequest{Name: "server-a", Endpoint: "https://a.example.test"}, "fixture")
	requireArgoCDSuccess(t, err)
	_, err = b.AddServer(AddServerRequest{Name: "server-b"}, "fixture")
	requireArgoCDSuccess(t, err)
	legacy, err := compat.AddServer(AddServerRequest{Name: "legacy", ClusterID: 9}, "fixture")
	requireArgoCDSuccess(t, err)
	if _, err := a.AddServer(AddServerRequest{Name: "foreign-cluster", ClusterID: 9}, "fixture"); !errors.Is(err, common.ErrBadRequest) {
		t.Fatalf("unverified cluster reference accepted: %v", err)
	}
	for _, tc := range []struct {
		service *ArgoCDService
		name    string
	}{
		{a, "server-a"}, {b, "server-b"}, {compat, "legacy"}, {svc, "legacy"},
	} {
		servers, err := tc.service.ListServers()
		requireArgoCDSuccess(t, err)
		if len(servers.Items) != 1 || servers.Items[0].Name != tc.name {
			t.Fatalf("server scope for %s: %+v", tc.name, servers.Items)
		}
	}
	repoReq := AddRepositoryRequest{ServerID: server.ID, RepoURL: "https://git.example.test/app", Type: "git"}
	appReq := CreateApplicationRequest{ServerID: server.ID, Name: "app-a", SourceRepoURL: repoReq.RepoURL}
	repo, err := a.AddRepository(repoReq, "fixture")
	requireArgoCDSuccess(t, err)
	app, err := a.CreateApplication(appReq, "fixture")
	requireArgoCDSuccess(t, err)
	foreignHistory := ArgoCDSyncHistory{
		TenantID: f.TenantB.TenantID, ApplicationID: app.ID, Revision: "foreign",
		Status: syncHistoryRunning, StartedAt: time.Now().UTC(),
	}
	requireArgoCDSuccess(t, f.DB.Create(&foreignHistory).Error)
	pending, err := a.SyncApplication(app.ID, "main")
	requireArgoCDSuccess(t, err)
	history, err := a.GetSyncHistory(app.ID)
	requireArgoCDSuccess(t, err)
	if len(history) != 1 || history[0].ID != pending.ID {
		t.Fatalf("foreign history disclosed or selected during sync: %+v", history)
	}
	for _, service := range []*ArgoCDService{b, compat} {
		t.Run(fmt.Sprintf("foreign-%d", service.TenantOwnerID()), func(t *testing.T) {
			repos, err := service.ListRepositories(server.ID)
			requireArgoCDSuccess(t, err)
			apps, err := service.ListApplications(server.ID)
			requireArgoCDSuccess(t, err)
			history, err := service.GetSyncHistory(app.ID)
			requireArgoCDSuccess(t, err)
			if len(repos.Items) != 0 || len(apps.Items) != 0 || len(history) != 0 {
				t.Fatalf("foreign list returned rows: %+v %+v %+v", repos, apps, history)
			}
			_, getServerErr := service.GetServer(server.ID)
			_, probeErr := service.TestServerConnection(server.ID)
			_, getRepoErr := service.GetRepository(repo.ID)
			_, getAppErr := service.GetApplication(app.ID)
			_, addRepoErr := service.AddRepository(repoReq, "fixture")
			_, addAppErr := service.CreateApplication(appReq, "fixture")
			_, syncErr := service.SyncApplication(app.ID, "main")
			for action, err := range map[string]error{
				"get server": getServerErr, "probe": probeErr, "get repository": getRepoErr, "get application": getAppErr,
				"add repository with foreign server": addRepoErr, "add application with foreign server": addAppErr,
				"update server":      service.UpdateServer(server.ID, UpdateServerRequest{Name: "foreign"}),
				"update repository":  service.UpdateRepository(repo.ID, repoReq),
				"update application": service.UpdateApplication(app.ID, UpdateApplicationRequest{Project: "foreign"}),
				"delete server":      service.DeleteServer(server.ID), "delete repository": service.DeleteRepository(repo.ID),
				"delete application": service.DeleteApplication(app.ID),
				"sync":               syncErr, "refresh": service.RefreshApplication(app.ID),
			} {
				if !errors.Is(err, gorm.ErrRecordNotFound) {
					t.Errorf("%s did not reject foreign record: %v", action, err)
				}
			}
		})
	}
	if creates.Load() != 1 || syncs.Load() != 1 {
		t.Fatalf("provider calls: creates=%d syncs=%d", creates.Load(), syncs.Load())
	}
	for _, row := range []struct {
		model any
		id    uint64
		owner uint64
	}{
		{&ArgoCDServer{}, server.ID, f.TenantA.TenantID}, {&ArgoCDServer{}, legacy.ID, 0},
		{&ArgoCDRepository{}, repo.ID, f.TenantA.TenantID}, {&ArgoCDApplication{}, app.ID, f.TenantA.TenantID},
		{&ArgoCDSyncHistory{}, pending.ID, f.TenantA.TenantID},
	} {
		var owner uint64
		requireArgoCDSuccess(t, f.DB.Model(row.model).Select("tenant_id").Where(idWhereClause, row.id).First(&owner).Error)
		if owner != row.owner {
			t.Errorf("%T %d owner = %d, want %d", row.model, row.id, owner, row.owner)
		}
	}
	requireArgoCDSuccess(t, a.UpdateRepository(repo.ID, repoReq))
	requireArgoCDSuccess(t, a.DeleteRepository(repo.ID))
	requireArgoCDSuccess(t, a.UpdateApplication(app.ID, UpdateApplicationRequest{DestinationNS: "ops"}))
	updated, err := a.GetApplication(app.ID)
	requireArgoCDSuccess(t, err)
	if updated.DestinationNS != "ops" {
		t.Fatalf("tenant-owned namespace update was lost: %+v", updated)
	}
}

func TestArgoCDTenantMissingResourceHTTPContract(t *testing.T) {
	f := tenantfixture.Open(t)
	svc := NewArgoCDService(f.DB)
	requireArgoCDSuccess(t, svc.Migrate())
	svc.newClient = func(string, string, string) (*ArgoCDClient, error) {
		t.Error("a missing resource must be rejected before constructing a provider client")
		return nil, errors.New("unexpected provider call")
	}
	server := ArgoCDServer{TenantID: f.TenantA.TenantID, Name: "server"}
	requireArgoCDSuccess(t, f.DB.Create(&server).Error)
	legacy := ArgoCDServer{TenantID: f.Compat.TenantID, Name: "legacy"}
	requireArgoCDSuccess(t, f.DB.Create(&legacy).Error)
	repo := ArgoCDRepository{TenantID: f.TenantA.TenantID, ServerID: server.ID, RepoURL: "https://git.example.test/repo"}
	requireArgoCDSuccess(t, f.DB.Create(&repo).Error)
	app := ArgoCDApplication{TenantID: f.TenantA.TenantID, ServerID: server.ID, Name: "app"}
	requireArgoCDSuccess(t, f.DB.Create(&app).Error)

	router := func(ctx *tenant.Context) *gin.Engine {
		r := gin.New()
		group := r.Group("/business", func(c *gin.Context) { tenant.SetGin(c, ctx) })
		NewArgoCDHandler(svc).RegisterRoutes(group)
		return r
	}
	call := func(t *testing.T, r *gin.Engine, method, path, body string) common.Response {
		t.Helper()
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(method, "/business/cicd/argocd"+path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(rec, req)
		var response common.Response
		if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil || rec.Code != http.StatusOK {
			t.Fatalf("%s %s: HTTP %d body %s, decode %v", method, path, rec.Code, rec.Body.String(), err)
		}
		return response
	}
	notFound := common.Response{Code: common.CodeNotFound, Message: "common.notFound"}
	cases := []struct {
		name, method, path, body string
		id                       uint64
	}{
		{"server detail", http.MethodGet, "/servers/%d", "", server.ID},
		{"server update", http.MethodPut, "/servers/%d", `{"name":"changed","endpoint":"https://example.test"}`, server.ID},
		{"server delete", http.MethodDelete, "/servers/%d", "", server.ID},
		{"server probe", http.MethodPost, "/servers/%d/test", "", server.ID},
		{"repository detail", http.MethodGet, "/repositories/%d", "", repo.ID},
		{"repository update", http.MethodPut, "/repositories/%d", `{"serverId":1,"repoUrl":"https://git.example.test/new","type":"git"}`, repo.ID},
		{"repository delete", http.MethodDelete, "/repositories/%d", "", repo.ID},
		{"application detail", http.MethodGet, "/applications/%d", "", app.ID},
		{"application update", http.MethodPut, "/applications/%d", `{"project":"changed"}`, app.ID},
		{"application delete", http.MethodDelete, "/applications/%d", "", app.ID},
		{"application sync", http.MethodPost, "/applications/%d/sync", `{"revision":"main"}`, app.ID},
		{"application refresh", http.MethodPost, "/applications/%d/refresh", "", app.ID},
		{"application rollback", http.MethodPost, "/applications/%d/rollback", `{"revision":"main"}`, app.ID},
	}
	b := router(f.TenantB)
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			for _, id := range []uint64{tc.id, tc.id + 1000} {
				got := call(t, b, tc.method, fmt.Sprintf(tc.path, id), tc.body)
				if got.Code != notFound.Code || got.Message != notFound.Message || got.Data != nil {
					t.Errorf("ID %d: response %+v, want %+v", id, got, notFound)
				}
			}
		})
	}
	for _, path := range []string{"/repositories", "/applications"} {
		for _, id := range []uint64{server.ID, server.ID + 1000} {
			body := fmt.Sprintf(`{"serverId":%d,"repoUrl":"https://git.example.test/repo","type":"git","name":"new","sourceRepoUrl":"https://git.example.test/repo"}`, id)
			got := call(t, b, http.MethodPost, path, body)
			if got.Code != notFound.Code || got.Message != notFound.Message || got.Data != nil {
				t.Errorf("%s parent %d: response %+v, want %+v", path, id, got, notFound)
			}
		}
	}
	compat := router(f.Compat)
	if got := call(t, compat, http.MethodGet, fmt.Sprintf("/servers/%d", legacy.ID), ""); got.Code != common.CodeSuccess || got.Data == nil {
		t.Fatalf("compat's own record is missing: %+v", got)
	}
	if got := call(t, compat, http.MethodGet, fmt.Sprintf("/servers/%d", server.ID), ""); got.Code != notFound.Code || got.Message != notFound.Message || got.Data != nil {
		t.Fatalf("compat lookup did not mask a tenant record: %+v", got)
	}
}

func TestArgoCDTenantReconciliation(t *testing.T) {
	f := tenantfixture.Open(t)
	requireArgoCDSuccess(t, NewArgoCDService(f.DB).Migrate())
	requireArgoCDSuccess(t, f.DB.AutoMigrate(&tenant.Tenant{}))
	requireArgoCDSuccess(t, f.DB.Create(&[]tenant.Tenant{
		{ID: f.TenantA.TenantID, Code: "a", Name: "A", Status: tenant.TenantStatusActive},
		{ID: f.TenantB.TenantID, Code: "b", Name: "B", Status: tenant.TenantStatusSuspended},
	}).Error)
	servers := []ArgoCDServer{
		{TenantID: f.TenantA.TenantID, Name: "active", Endpoint: "active"},
		{TenantID: f.TenantB.TenantID, Name: "suspended", Endpoint: "suspended"},
		{Name: "compat", Endpoint: "compat"},
	}
	requireArgoCDSuccess(t, f.DB.Create(&servers).Error)
	apps := []ArgoCDApplication{
		{TenantID: f.TenantA.TenantID, ServerID: servers[0].ID, Name: "owned"},
		{TenantID: f.TenantB.TenantID, ServerID: servers[0].ID, Name: "foreign"},
		{ServerID: servers[2].ID, Name: "legacy"},
	}
	requireArgoCDSuccess(t, f.DB.Create(&apps).Error)
	started := time.Now().UTC().Add(-time.Minute).Truncate(time.Second)
	histories := []ArgoCDSyncHistory{
		{TenantID: f.TenantA.TenantID, ApplicationID: apps[0].ID, Revision: "main", Status: syncHistoryRunning, StartedAt: started},
		{TenantID: f.TenantB.TenantID, ApplicationID: apps[0].ID, Revision: "other", Status: syncHistoryRunning, StartedAt: started},
	}
	requireArgoCDSuccess(t, f.DB.Create(&histories).Error)
	remote := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/version":
		case "/api/v1/applications/owned", "/api/v1/applications/legacy":
			_, _ = fmt.Fprintf(w, `{"status":{"sync":{"status":"Synced"},"health":{"status":"Healthy"},"operationState":{"phase":"Succeeded","startedAt":%q,"operation":{"sync":{"revision":"main"}}}}}`, started.Add(time.Second).Format(time.RFC3339))
		default:
			t.Errorf("reconciler queried foreign application: %s", r.URL.Path)
		}
	}))
	defer remote.Close()
	r := NewReconciler(f.DB, time.Second)
	var probed []string
	r.newClient = func(endpoint, _, _ string) (*ArgoCDClient, error) {
		probed = append(probed, endpoint)
		return &ArgoCDClient{endpoint: remote.URL, httpClient: remote.Client()}, nil
	}
	r.modeReader = argoCDTestMode{mode: tenant.ModeCompat}
	requireArgoCDSuccess(t, r.reconcile(context.Background()))
	if !slices.Equal(probed, []string{"compat"}) {
		t.Fatalf("compat probed private servers: %v", probed)
	}
	r.serverCursor = 0
	for _, reader := range []tenant.ModeReader{nil, argoCDTestMode{err: tenant.ErrModeUnavailable}} {
		r.modeReader = reader
		if err := r.reconcile(context.Background()); !errors.Is(err, tenant.ErrModeUnavailable) || len(probed) != 1 {
			t.Fatalf("unavailable mode probed servers: %v, %v", probed, err)
		}
	}
	r.modeReader = argoCDTestMode{mode: tenant.ModeMulti}
	requireArgoCDSuccess(t, r.reconcile(context.Background()))
	if !slices.Equal(probed, []string{"compat", "active", "compat"}) {
		t.Fatalf("multi probed unexpected servers: %v", probed)
	}
	for i, want := range []string{syncHistorySucceeded, syncHistoryRunning} {
		requireArgoCDSuccess(t, f.DB.First(&histories[i], histories[i].ID).Error)
		if histories[i].Status != want {
			t.Errorf("history owner %d status = %q, want %q", histories[i].TenantID, histories[i].Status, want)
		}
	}
	requireArgoCDSuccess(t, f.DB.First(&apps[1], apps[1].ID).Error)
	if apps[1].HealthStatus != "" {
		t.Fatalf("foreign application modified: %+v", apps[1])
	}
}

func TestArgoCDTenantWebhookProcessing(t *testing.T) {
	f := tenantfixture.Open(t)
	requireArgoCDSuccess(t, f.DB.AutoMigrate(&ArgoCDApplication{}, &WebhookEvent{}))
	const repo = "https://git.example.test/shared"
	apps := []ArgoCDApplication{
		{TenantID: f.TenantA.TenantID, ServerID: 1, Name: "shared", SourceRepoURL: repo, SyncPolicy: datatypes.JSON(`{"automated":{}}`)},
		{TenantID: f.TenantB.TenantID, ServerID: 2, Name: "shared", SourceRepoURL: repo, SyncPolicy: datatypes.JSON(`{"automated":{}}`)},
		{ServerID: 3, Name: "shared", SourceRepoURL: repo, SyncPolicy: datatypes.JSON(`{"automated":{}}`)},
	}
	requireArgoCDSuccess(t, f.DB.Create(&apps).Error)
	event := WebhookEvent{TenantID: f.TenantB.TenantID, Source: sourceGitHub, EventType: "push", Repository: repo}
	requireArgoCDSuccess(t, f.DB.Create(&event).Error)
	syncer := &recordingSyncer{}
	h := &WebhookHandler{db: f.DB, sync: syncer.syncApplicationForOwner}
	h.processGitEvent(&event)
	if len(syncer.calls) != 1 || syncer.calls[0].ownerID != f.TenantB.TenantID || syncer.calls[0].appID != apps[1].ID {
		t.Fatalf("git callback did not use event owner: %+v", syncer.calls)
	}
	argoEvent := WebhookEvent{
		TenantID: f.TenantB.TenantID, Source: sourceArgoCD, EventType: "app-sync", Repository: "shared",
		Payload: `{"status":{"sync":{"status":"Synced"},"health":{"status":"Healthy"}}}`,
	}
	requireArgoCDSuccess(t, f.DB.Create(&argoEvent).Error)
	h.processArgoCDEvent(&argoEvent)
	for i, want := range []string{"", HealthHealthy, ""} {
		requireArgoCDSuccess(t, f.DB.First(&apps[i], apps[i].ID).Error)
		if apps[i].HealthStatus != want {
			t.Errorf("Argo CD callback health for owner %d = %q, want %q", apps[i].TenantID, apps[i].HealthStatus, want)
		}
	}
	for _, reader := range []tenant.ModeReader{nil, argoCDTestMode{mode: tenant.ModeMulti}, argoCDTestMode{err: tenant.ErrModeUnavailable}} {
		h.modeReader = reader
		for _, handler := range []http.HandlerFunc{h.HandleGitWebhook, h.HandleArgoCDWebhook} {
			response := httptest.NewRecorder()
			handler(response, httptest.NewRequest(http.MethodPost, "/webhook", nil))
			if response.Code != http.StatusForbidden {
				t.Fatalf("legacy ingress did not fail closed: %d", response.Code)
			}
		}
	}
	visible, err := h.ListEvents("", 10)
	requireArgoCDSuccess(t, err)
	if len(visible) != 0 {
		t.Fatalf("compat webhook listing contains tenant rows: %+v", visible)
	}
}

func TestArgoCDPermissionRoutesCoverMountedMethods(t *testing.T) {
	requireArgoCDSuccess(t, RegisterPermissionRoutes())
	router := gin.New()
	NewArgoCDHandler(nil).RegisterRoutes(router.Group("/api/v1/business"))
	want := make(map[permissionroutes.Route]bool)
	for _, route := range router.Routes() {
		want[permissionroutes.Route{Method: route.Method, Path: route.Path}] = true
	}
	bindings := map[string][]string{
		"list": {http.MethodGet}, "view": {http.MethodGet},
		"create": {http.MethodPost, http.MethodPut}, "delete": {http.MethodDelete},
	}
	for key, methods := range bindings {
		routes := permissionroutes.Lookup("business:cicd:argocd:" + key)
		if len(routes) == 0 {
			t.Errorf("permission %s has no routes", key)
		}
		for _, route := range routes {
			if _, mounted := want[route]; !mounted {
				t.Errorf("%s permission binds an unmounted route: %+v", key, route)
			}
			if !slices.Contains(methods, route.Method) {
				t.Errorf("%s permission binds unexpected method: %+v", key, route)
			}
			want[route] = false
		}
	}
	for route, missing := range want {
		if missing {
			t.Errorf("mounted route lacks a permission: %+v", route)
		}
	}
}

func requireArgoCDSuccess(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}
