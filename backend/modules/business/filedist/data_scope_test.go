package filedist

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/duanxldragon/pantheon-base/backend/modules/business/secrets"
	"github.com/duanxldragon/pantheon-base/backend/pkg/testmysql"

	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
	"github.com/gin-gonic/gin"
)

func TestFileDistRoutesRejectUnsupportedDataScope(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cases := []struct {
		name  string
		scope any
	}{
		{"missing", nil}, {"wrong-type", "all"},
		{"typed-nil", (*common.DataScopeReq)(nil)},
		{"empty", &common.DataScopeReq{}},
		{"whitespace", &common.DataScopeReq{Mode: " "}},
		{"self-numeric-username", &common.DataScopeReq{UserID: 1, Mode: common.DataScopeModeSelf}},
		{"dept", &common.DataScopeReq{DeptID: 1, Mode: common.DataScopeModeDept}},
		{"children", &common.DataScopeReq{DeptID: 1, DeptIDs: []uint64{1, 2}, Mode: common.DataScopeModeDeptAndChildren}},
		{"custom", &common.DataScopeReq{DeptIDs: []uint64{1, 2}, Mode: common.DataScopeModeCustom}},
		{"unknown", &common.DataScopeReq{Mode: "tenant"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			router := gin.New()
			router.Use(gin.RecoveryWithWriter(io.Discard))
			router.Use(func(c *gin.Context) {
				if tc.scope != nil {
					c.Set(common.DataScopeContextKey, tc.scope)
				}
				c.Set("username", "1")
				c.Next()
			})
			handler := NewFileDistHandler(nil, nil)
			handler.RegisterRoutes(router.Group("/filedist"))
			handler.RegisterRoutes(router.Group("/business/filedist"))
			// Enumerate the registered surface so newly added aliases/actions are
			// also protected. Nil services prove denial precedes any handler I/O.
			for _, route := range router.Routes() {
				t.Run(route.Method+route.Path, func(t *testing.T) {
					path := strings.ReplaceAll(route.Path, ":id", "1")
					request := httptest.NewRequest(route.Method, path+"?scope=all&isAdmin=true", strings.NewReader(`{"scope":"all","isAdmin":true}`))
					request.Header.Set("Content-Type", "application/json")
					request.Header.Set("X-Data-Scope", "all")
					request.Header.Set("X-Admin", "true")
					recorder := httptest.NewRecorder()
					router.ServeHTTP(recorder, request)
					var response struct {
						Code    int    `json:"code"`
						Message string `json:"message"`
					}
					if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
						t.Fatalf("expected denial envelope, HTTP %d: %s", recorder.Code, recorder.Body.String())
					}
					if response.Code != http.StatusForbidden || response.Message != "permission.denied" {
						t.Fatalf("unsupported scope reached handler: HTTP %d: %s", recorder.Code, recorder.Body.String())
					}
				})
			}
		})
	}
}

func TestFileDistExplicitUnrestrictedScope(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, scope := range []*common.DataScopeReq{
		{UserID: 1, Mode: common.DataScopeModeAll},
		{UserID: 1, Mode: " all "},
		{UserID: 1, IsAdmin: true, Mode: common.DataScopeModeSelf},
		{UserID: 1, IsAdmin: true},
	} {
		router := gin.New()
		calls := 0
		router.GET("/allowed", func(c *gin.Context) {
			c.Set(common.DataScopeContextKey, scope)
			c.Next()
		}, requireFileDistUnrestrictedScope, func(c *gin.Context) {
			calls++
			c.Status(http.StatusNoContent)
		})
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/allowed", nil))
		if recorder.Code != http.StatusNoContent || calls != 1 {
			t.Fatalf("trusted all/admin scope blocked: %+v HTTP %d calls %d", scope, recorder.Code, calls)
		}
	}
}

func TestFileDistScopeGateDoesNotAffectSiblingRoutes(t *testing.T) {
	router := gin.New()
	NewFileDistHandler(nil, nil).RegisterRoutes(router.Group("/filedist"))
	router.GET("/other", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/other", nil))
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("sibling route blocked: %d", recorder.Code)
	}
}

// This exercises real Gin handlers and MySQL, with a trusted test context.
// Production JWT/Casbin/IAM policy resolution remains a separate acceptance gap.
func TestFileDistScopedHandlerPersistence(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv(secrets.BusinessSecretKeyEnv, "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef")
	db := testmysql.Open(t)
	if err := db.AutoMigrate(&FileStorage{}, &FileRecord{}, &DistributionTask{}, &DistributionTarget{}); err != nil {
		t.Fatal(err)
	}
	storage := FileStorage{Name: "scope-test", Type: "local", BasePath: t.TempDir(), Status: "active"}
	if err := db.Create(&storage).Error; err != nil {
		t.Fatal(err)
	}
	file := FileRecord{StorageID: storage.ID, Name: "payload.txt", Path: "payload.txt", Size: 1, UploadedBy: "owner", UploadedAt: time.Now()}
	if err := db.Create(&file).Error; err != nil {
		t.Fatal(err)
	}
	storageService := NewStorageService(db)
	service := NewDistributionService(db, storageService, NewTransferEngine(db, storageService, 1))
	handler := NewFileDistHandler(storageService, service)
	var scope *common.DataScopeReq
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(common.DataScopeContextKey, scope)
		c.Set("username", "owner")
		c.Next()
	})
	handler.RegisterRoutes(router.Group("/business/filedist"))
	scheduledAt := time.Now().Add(time.Hour)
	dto := DistributionTaskDTO{Name: "scheduled-scoped", FileID: file.ID, ScheduledAt: &scheduledAt, Targets: []TargetConfig{{Host: "127.0.0.1", TargetPath: "/tmp/payload.txt", Username: "fixture", HostFingerprint: "SHA256:fixture"}}}
	raw, err := json.Marshal(dto)
	if err != nil {
		t.Fatal(err)
	}
	request := func(method, path string, body io.Reader) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, "/business/filedist"+path, body)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		return rec
	}
	decode := func(rec *httptest.ResponseRecorder, expected int) json.RawMessage {
		t.Helper()
		var envelope struct {
			Code int             `json:"code"`
			Data json.RawMessage `json:"data"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil || envelope.Code != expected {
			t.Fatalf("expected code %d, got HTTP %d: %s", expected, rec.Code, rec.Body.String())
		}
		return envelope.Data
	}
	for _, admitted := range []*common.DataScopeReq{{UserID: 7, Mode: common.DataScopeModeAll}, {UserID: 7, IsAdmin: true, Mode: common.DataScopeModeSelf}} {
		scope = admitted
		data := decode(request(http.MethodPost, "/tasks", strings.NewReader(string(raw))), http.StatusOK)
		var created DistributionTaskResponse
		if err := json.Unmarshal(data, &created); err != nil || created.ID == 0 || created.Status != statusPending {
			t.Fatalf("invalid create: %s", data)
		}
		id := strconv.FormatUint(created.ID, 10)
		for _, path := range []string{"/storages", "/files", "/tasks", "/tasks/" + id, "/tasks/" + id + "/targets", "/tasks/" + id + "/hosts", "/tasks/" + id + "/events", "/tasks/" + id + "/progress"} {
			decode(request(http.MethodGet, path, nil), http.StatusOK)
		}
		scope = &common.DataScopeReq{UserID: 7, DeptID: 1, Mode: common.DataScopeModeSelf}
		decode(request(http.MethodPost, "/tasks", strings.NewReader(string(raw))), http.StatusForbidden)
		for _, path := range []string{"/tasks/" + id + "/start", "/tasks/" + id + "/cancel", "/tasks/" + id + "/retry"} {
			decode(request(http.MethodPost, path, nil), http.StatusForbidden)
		}
		decode(request(http.MethodDelete, "/tasks/"+id, nil), http.StatusForbidden)
		decode(request(http.MethodDelete, "/files/"+strconv.FormatUint(file.ID, 10), nil), http.StatusForbidden)
		decode(request(http.MethodDelete, "/storages/"+strconv.FormatUint(storage.ID, 10), nil), http.StatusForbidden)
		decode(request(http.MethodGet, "/files/"+strconv.FormatUint(file.ID, 10)+"/download", nil), http.StatusForbidden)
		var persisted DistributionTask
		if err := db.First(&persisted, created.ID).Error; err != nil || persisted.Status != statusPending || persisted.StartedAt != nil {
			t.Fatalf("denial changed task: %+v %v", persisted, err)
		}
	}
	for _, model := range []struct {
		value    any
		expected int64
	}{{&FileStorage{}, 1}, {&FileRecord{}, 1}, {&DistributionTask{}, 2}, {&DistributionTarget{}, 2}} {
		var count int64
		if err := db.Model(model.value).Count(&count).Error; err != nil || count != model.expected {
			t.Fatalf("denial changed rows for %T: count %d err %v", model.value, count, err)
		}
	}
}
