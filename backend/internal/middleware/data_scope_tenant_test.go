package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
	"github.com/duanxldragon/pantheon-base/backend/pkg/tenant"
	"github.com/duanxldragon/pantheon-base/backend/pkg/testmysql"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// 三个租户共用同一个 role_key。这正是验收 F10 的场景：`role_key` 只在一个租户
// 内唯一，所以 `WHERE role_key IN (...)` 会把所有租户的策略一起读出来，而
// resolveDataScopeMode 取最宽的一条 —— 谁的策略最宽，别人就看到最多行。
const (
	dsTenantSelfOnly uint64 = 101 // 自己那条是 self
	dsTenantDept     uint64 = 202 // 自己那条是 dept
	dsTenantNoPolicy uint64 = 303 // 自己没有任何策略行
	dsSharedRoleKey         = "ops"
	dsPolicyLessRole        = "viewer"
)

// seedDataScopeFixtures 建出一个「同一 role_key 被多个租户持有」的最小环境：
// 平台行（tenant 0）是 all，租户 101 是 self，租户 202 是 dept，租户 303 没有行。
// 修复前三个租户都会读到 all。
func seedDataScopeFixtures(t *testing.T) *gorm.DB {
	t.Helper()
	db := testmysql.Open(t)
	if err := db.Exec("CREATE TABLE system_user (id BIGINT PRIMARY KEY, dept_id BIGINT)").Error; err != nil {
		t.Fatalf("create user table: %v", err)
	}
	if err := MigrateDataScopePolicy(db); err != nil {
		t.Fatalf("migrate data scope policy: %v", err)
	}
	if err := db.Exec("INSERT INTO system_user (id, dept_id) VALUES (7, 42)").Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
	rows := []SystemRoleDataScope{
		{TenantID: tenant.PlatformGlobalTenantID, RoleKey: dsSharedRoleKey, Mode: common.DataScopeModeAll},
		{TenantID: dsTenantSelfOnly, RoleKey: dsSharedRoleKey, Mode: common.DataScopeModeSelf},
		{TenantID: dsTenantDept, RoleKey: dsSharedRoleKey, Mode: common.DataScopeModeDept},
	}
	for i := range rows {
		if err := db.Create(&rows[i]).Error; err != nil {
			t.Fatalf("seed data scope policy %+v: %v", rows[i], err)
		}
	}
	return db
}

// scopedRequest 跑一次经过 DataScopeMiddleware 的请求，返回处理器是否被执行、
// 落到 gin 上下文里的数据范围模式，以及原始响应体。
func scopedRequest(db *gorm.DB, tenantID uint64, roleKeys []string) (bool, string, string) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(func(c *gin.Context) {
		c.Set("userId", uint64(7))
		c.Set("roleKeys", roleKeys)
		if tenantID != 0 {
			tenant.SetGin(c, &tenant.Context{TenantID: tenantID, Mode: tenant.ModeMulti})
		}
		c.Next()
	})
	engine.Use(DataScopeMiddleware(db))

	reached := false
	mode := ""
	engine.GET("/scoped", func(c *gin.Context) {
		reached = true
		if scope := common.GetDataScope(c); scope != nil {
			mode = scope.Mode
		}
		common.Success(c, gin.H{"mode": mode})
	})

	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/scoped", nil))
	return reached, mode, recorder.Body.String()
}

// dataScopeTenantCase is one row of the F10 matrix: the tenant under test and
// the policy its shared role key must resolve to.
type dataScopeTenantCase struct {
	name     string
	tenantID uint64
	want     string
	wantRows int
}

// TestLoadRoleDataScopePoliciesIsTenantScoped 是 F10 的最小复现：同一 role_key
// 在多个租户各有一条策略时，读取必须只返回调用方租户的那一条。
func TestLoadRoleDataScopePoliciesIsTenantScoped(t *testing.T) {
	db := seedDataScopeFixtures(t)
	cases := []dataScopeTenantCase{
		{name: "self-only tenant", tenantID: dsTenantSelfOnly, want: common.DataScopeModeSelf, wantRows: 1},
		{name: "dept tenant", tenantID: dsTenantDept, want: common.DataScopeModeDept, wantRows: 1},
		{name: "tenant without policy", tenantID: dsTenantNoPolicy, want: "", wantRows: 0},
		{name: "platform scope", tenantID: tenant.PlatformGlobalTenantID, want: common.DataScopeModeAll, wantRows: 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assertTenantResolvesOwnPolicy(t, db, tc)
		})
	}
}

// assertTenantResolvesOwnPolicy asserts that the shared role key resolves to
// exactly one policy row, owned by the case's tenant, with the expected mode.
func assertTenantResolvesOwnPolicy(t *testing.T, db *gorm.DB, tc dataScopeTenantCase) {
	t.Helper()
	policies, ok := loadRoleDataScopePolicies(db, tc.tenantID, []string{dsSharedRoleKey})
	if !ok {
		t.Fatalf("policy lookup must succeed")
	}
	if len(policies) != tc.wantRows {
		t.Fatalf("expected %d policy row(s) for tenant %d, got %d: %+v",
			tc.wantRows, tc.tenantID, len(policies), policies)
	}
	if tc.want == "" {
		return
	}
	if policies[0].Mode != tc.want {
		t.Fatalf("expected mode %q for tenant %d, got %q", tc.want, tc.tenantID, policies[0].Mode)
	}
	if policies[0].TenantID != tc.tenantID {
		t.Fatalf("expected policy row owned by tenant %d, got %d", tc.tenantID, policies[0].TenantID)
	}
}

// TestDataScopeMiddlewareAppliesOnlyTheCallersPolicy 断言端到端行为：租户 101
// 只能拿到自己那条 self，绝不会被平台行（all）或租户 202 的行放宽。
func TestDataScopeMiddlewareAppliesOnlyTheCallersPolicy(t *testing.T) {
	db := seedDataScopeFixtures(t)

	reached, mode, body := scopedRequest(db, dsTenantSelfOnly, []string{dsSharedRoleKey})
	if !reached {
		t.Fatalf("request must reach the handler, body=%s", body)
	}
	if mode != common.DataScopeModeSelf {
		t.Fatalf("tenant %d must be limited to its own %q policy, got %q (platform/peer rows leaked)",
			dsTenantSelfOnly, common.DataScopeModeSelf, mode)
	}

	reached, mode, body = scopedRequest(db, dsTenantDept, []string{dsSharedRoleKey})
	if !reached {
		t.Fatalf("request must reach the handler, body=%s", body)
	}
	if mode != common.DataScopeModeDept {
		t.Fatalf("tenant %d must be limited to its own %q policy, got %q",
			dsTenantDept, common.DataScopeModeDept, mode)
	}
}

// TestDataScopeMiddlewareIgnoresOtherTenantsPolicy 断言反方向：租户 303 自己
// 没有策略行时，平台行（all）不会被当成它的策略 —— 必须 fail-closed。
// 这正是「A 租户的数据权限决定 B 租户看到多少行」的那条链路。
func TestDataScopeMiddlewareIgnoresOtherTenantsPolicy(t *testing.T) {
	db := seedDataScopeFixtures(t)

	reached, mode, body := scopedRequest(db, dsTenantNoPolicy, []string{dsSharedRoleKey})
	if reached {
		t.Fatalf("handler must not run when the caller's tenant owns no policy row, got mode=%q", mode)
	}
	if !strings.Contains(body, `"code":500`) || !strings.Contains(body, "common.serverError") {
		t.Fatalf("expected the fail-closed envelope, status body=%s", body)
	}

	// 同一条链路对「完全没有策略的普通角色」也必须 fail-closed，防止上面的断言
	// 被一个更宽的放行分支意外满足。
	reached, _, body = scopedRequest(db, dsTenantSelfOnly, []string{dsPolicyLessRole})
	if reached {
		t.Fatalf("handler must not run for a role without any policy row")
	}
	if !strings.Contains(body, `"code":500`) {
		t.Fatalf("expected the fail-closed envelope, body=%s", body)
	}
}

// TestResolvePolicyTenantID 固定上下文解析规则：multi 模式用请求租户，
// compat / 缺失上下文回落到平台全局（0），而不是去合并所有租户。
func TestResolvePolicyTenantID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	if got := resolvePolicyTenantID(c); got != tenant.PlatformGlobalTenantID {
		t.Fatalf("missing tenant context must resolve to the platform scope, got %d", got)
	}

	tenant.SetGin(c, &tenant.Context{TenantID: 0, Mode: tenant.ModeCompat})
	if got := resolvePolicyTenantID(c); got != tenant.PlatformGlobalTenantID {
		t.Fatalf("compat mode must resolve to the platform scope, got %d", got)
	}

	tenant.SetGin(c, &tenant.Context{TenantID: dsTenantDept, Mode: tenant.ModeMulti})
	if got := resolvePolicyTenantID(c); got != dsTenantDept {
		t.Fatalf("multi mode must resolve to the request tenant %d, got %d", dsTenantDept, got)
	}
}
