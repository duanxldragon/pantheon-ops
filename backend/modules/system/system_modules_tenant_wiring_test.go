package system

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/duanxldragon/pantheon-base/backend/internal/middleware"
	audit "github.com/duanxldragon/pantheon-base/backend/modules/system/audit"
	setting "github.com/duanxldragon/pantheon-base/backend/modules/system/config/setting"
	"github.com/duanxldragon/pantheon-base/backend/pkg/authtoken"
	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
	"github.com/duanxldragon/pantheon-base/backend/pkg/database"
	"github.com/duanxldragon/pantheon-base/backend/pkg/tenant"
	"github.com/duanxldragon/pantheon-base/backend/pkg/testmysql"

	"github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/model"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// TestInitAuditModules_TenantContextWiring pins the audit module's route-group
// wiring (queue-6 live matrix finding #1, 2026-09-13): the protected group MUST
// run TenantContextMiddleware, otherwise tenant.FromGin returns nil and
// WithTenantScope no-ops — every tenant would read the full operation-log
// population in multi mode. The test registers the module through the
// production path (initAuditModules) and asserts a tenant-claimed subject sees
// only its own rows.
func TestInitAuditModules_TenantContextWiring(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cases := []struct {
		name          string
		tenantClaim   uint64
		wantCode      int
		wantTitlesAll bool // true = expect BOTH tenant rows (wiring regressed)
	}{
		{
			name:          "tenant 101 claim is scoped to tenant 101 rows",
			tenantClaim:   101,
			wantCode:      common.CodeSuccess,
			wantTitlesAll: false,
		},
		{
			name:          "tenant 202 claim is scoped to tenant 202 rows",
			tenantClaim:   202,
			wantCode:      common.CodeSuccess,
			wantTitlesAll: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			engine, db, token := newAuditWiringEngineWithToken(t, tc.tenantClaim)

			recorder := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/api/v1/system/operation-log/list", nil)
			req.Header.Set("Authorization", "Bearer "+token)
			engine.ServeHTTP(recorder, req)

			if recorder.Code != http.StatusOK {
				t.Fatalf("expected HTTP 200 envelope, got %d: %s", recorder.Code, recorder.Body.String())
			}
			response := decodeAuditWiringResponse(t, recorder)
			if response.Code != tc.wantCode {
				t.Fatalf("expected code %d, got %d (message=%s)", tc.wantCode, response.Code, response.Message)
			}
			assertAuditWiringIsolation(t, response, tc.tenantClaim)
			if got := countAuditRows(t, db); got != 2 {
				t.Fatalf("fixture corruption: expected 2 rows in table, got %d", got)
			}
		})
	}
}

// newAuditWiringEngine builds an engine with the audit module registered via
// initAuditModules against an isolated MySQL DB, with a stub of the token
// middleware that injects the same context keys as applyTokenContext —
// including the tenant claim — before the module's route chain runs.
func newAuditWiringEngineWithToken(t *testing.T, tenantClaim uint64) (*gin.Engine, *gorm.DB, string) {
	t.Helper()

	db := testmysql.Open(t)
	if err := db.AutoMigrate(&middleware.SystemLogOper{}, &setting.SystemSetting{}); err != nil {
		t.Fatalf("migrate system tables: %v", err)
	}
	if err := db.Exec("DELETE FROM system_log_oper").Error; err != nil {
		t.Fatalf("clear system_log_oper: %v", err)
	}
	seedAuditWiringRow(t, db, 101, "wiring-tenant-101")
	seedAuditWiringRow(t, db, 202, "wiring-tenant-202")

	deps := &systemModuleDependencies{
		db:                 db,
		refreshSyncSvc:     NewRefreshSyncService(db),
		refreshSyncHandler: NewRefreshSyncHandler(NewRefreshSyncService(db)),
		// Mode is pinned multi; the TTL is irrelevant (loadFn is constant).
		tenantModeLoader: tenant.NewModeLoader(func() string { return tenant.ModeMulti }, 0),
		auditSvc:         audit.NewAuditService(db),
		auditHandler:     audit.NewAuditHandler(audit.NewAuditService(db)),
		settingSvc:       setting.NewSettingService(db), // retention settings lookup
	}

	// Real Redis-backed token session (skip when the local Redis is down so
	// the wiring pin keeps running in DSN-less CI): TokenAuthMiddleware reads
	// database.RDB, so point it at the same store and seed a session with the
	// desired tenant claim — exactly the shape a real login produces.
	rdb := database.RDB
	if rdb == nil {
		addr := strings.TrimSpace(os.Getenv("PANTHEON_REDIS_ADDR"))
		if addr == "" {
			addr = "127.0.0.1:6379"
		}
		rdb = redis.NewClient(&redis.Options{
			Addr:     addr,
			Password: os.Getenv("PANTHEON_REDIS_PASSWORD"),
		})
		database.RDB = rdb
		t.Cleanup(func() { database.RDB = nil })
	}
	ctx := context.Background()
	if err := rdb.Ping(ctx).Err(); err != nil {
		t.Skipf("local redis unavailable: %v", err)
	}
	token := authtoken.NewAccessToken()
	if err := authtoken.StoreSession(ctx, rdb, token, &authtoken.SessionData{
		UserID:    1,
		Username:  "wiring-admin",
		RoleKeys:  []string{"admin"},
		SessionID: "wiring-session-" + uint64ToString(tenantClaim),
		TenantID:  tenantClaim,
	}, time.Minute); err != nil {
		t.Fatalf("store token session: %v", err)
	}

	// CasbinMiddleware requires a global enforcer; install a minimal one that
	// grants the admin role on the audit list route (subject matching only —
	// the wiring under test sits upstream of the policy check).
	installWiringEnforcer(t)

	engine := gin.New()
	api := engine.Group("/api/v1")
	for _, m := range initAuditModules(deps) {
		m.RegisterRoutes(api)
	}
	return engine, db, token
}

func installWiringEnforcer(t *testing.T) {
	t.Helper()
	m, err := model.NewModelFromString(`
		[request_definition]
		r = sub, obj, act
		[policy_definition]
		p = sub, obj, act
		[role_definition]
		g = _, _
		[policy_effect]
		e = some(where (p.eft == allow))
		[matchers]
		m = (r.sub == p.sub || g(r.sub, p.sub)) && keyMatch2(r.obj, p.obj) && r.act == p.act
	`)
	if err != nil {
		t.Fatalf("create casbin model: %v", err)
	}
	enforcer, err := casbin.NewSyncedEnforcer(m)
	if err != nil {
		t.Fatalf("create casbin enforcer: %v", err)
	}
	if _, err := enforcer.AddPolicy("admin", "/api/v1/system/operation-log/*", "GET"); err != nil {
		t.Fatalf("add casbin policy: %v", err)
	}
	original := database.Enforcer
	database.Enforcer = enforcer
	t.Cleanup(func() { database.Enforcer = original })
}

func seedAuditWiringRow(t *testing.T, db *gorm.DB, tenantID uint64, title string) {
	t.Helper()
	row := middleware.SystemLogOper{
		Title:    title,
		TenantID: tenantID,
		OperTime: time.Now(),
	}
	if err := db.Create(&row).Error; err != nil {
		t.Fatalf("seed audit row tenant %d: %v", tenantID, err)
	}
}

// decodeAuditWiringTitles extracts the operation-log titles from the page
// response data (OperationLogPageResp.List[].Title).
// decodeAuditWiringResponse asserts the HTTP envelope and decodes the body.
func decodeAuditWiringResponse(t *testing.T, recorder *httptest.ResponseRecorder) common.Response {
	t.Helper()
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected HTTP 200 envelope, got %d: %s", recorder.Code, recorder.Body.String())
	}
	var response common.Response
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response body: %v", err)
	}
	return response
}

// assertAuditWiringIsolation verifies the subject sees its own row and no row
// from the other tenant (the wiring-regression tenancy-leak assertion).
func assertAuditWiringIsolation(t *testing.T, response common.Response, tenantClaim uint64) {
	t.Helper()
	dataJSON, err := json.Marshal(response.Data)
	if err != nil {
		t.Fatalf("re-encode response data: %v", err)
	}
	visible := decodeAuditWiringTitles(t, dataJSON)
	own := "wiring-tenant-" + uint64ToString(tenantClaim)
	other := "wiring-tenant-" + uint64ToString(otherTenantID(tenantClaim))
	if !containsString(visible, own) {
		t.Fatalf("expected subject's own row %q in %v", own, visible)
	}
	if containsString(visible, other) {
		t.Fatalf("TENANT LEAK: tenant %d saw row %q from tenant %d — TenantContextMiddleware is missing from the audit route group (rows: %v)",
			tenantClaim, other, otherTenantID(tenantClaim), visible)
	}
}

func decodeAuditWiringTitles(t *testing.T, data json.RawMessage) []string {
	t.Helper()
	var page struct {
		Items []struct {
			Title string `json:"title"`
		} `json:"items"`
	}
	if len(data) > 0 {
		if err := json.Unmarshal(data, &page); err != nil {
			t.Fatalf("decode page data: %v", err)
		}
	}
	titles := make([]string, 0, len(page.Items))
	for _, item := range page.Items {
		titles = append(titles, item.Title)
	}
	return titles
}

func countAuditRows(t *testing.T, db *gorm.DB) int64 {
	t.Helper()
	var n int64
	if err := db.Model(&middleware.SystemLogOper{}).Count(&n).Error; err != nil {
		t.Fatalf("count audit rows: %v", err)
	}
	return n
}

func otherTenantID(claim uint64) uint64 {
	if claim == 101 {
		return 202
	}
	return 101
}

func uint64ToString(v uint64) string {
	if v == 0 {
		return "0"
	}
	digits := make([]byte, 0, 20)
	for v > 0 {
		digits = append(digits, byte('0'+v%10))
		v /= 10
	}
	for i, j := 0, len(digits)-1; i < j; i, j = i+1, j-1 {
		digits[i], digits[j] = digits[j], digits[i]
	}
	return string(digits)
}

func containsString(list []string, want string) bool {
	for _, item := range list {
		if item == want {
			return true
		}
	}
	return false
}
