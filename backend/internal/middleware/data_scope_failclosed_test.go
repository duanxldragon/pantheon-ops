package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
	"github.com/duanxldragon/pantheon-base/backend/pkg/database"
	"github.com/duanxldragon/pantheon-base/backend/pkg/testmysql"

	"github.com/gin-gonic/gin"
)

type dataScopeDenyProbe struct {
	ID     uint64 `gorm:"primaryKey"`
	DeptID uint64
}

func (dataScopeDenyProbe) TableName() string {
	return "data_scope_deny_probes"
}

// TestApplyRolePolicyFailureFailsClosedScope 直接注入策略读取失败，
// 验证 applyRoleDataScopePolicy 返回失败且 scope 不含任何部门（绝不退回 all），
// WithDataScope 对该 scope 拒绝返回任何行（F01 故障注入）。
func TestApplyRolePolicyFailureFailsClosedScope(t *testing.T) {
	db := testmysql.Open(t)
	if err := db.Exec("CREATE TABLE system_user (id BIGINT PRIMARY KEY, dept_id BIGINT)").Error; err != nil {
		t.Fatalf("create user table: %v", err)
	}
	// 结构损坏的策略表：注入 role_key 查询失败。
	if err := db.Exec("CREATE TABLE system_role_data_scope (unrelated_col BIGINT)").Error; err != nil {
		t.Fatalf("create broken policy table: %v", err)
	}
	storeCachedTableExistence(db, (&SystemRoleDataScope{}).TableName(), true)
	if err := db.AutoMigrate(&dataScopeDenyProbe{}); err != nil {
		t.Fatalf("migrate probe table: %v", err)
	}
	if err := db.Create(&dataScopeDenyProbe{ID: 1, DeptID: 42}).Error; err != nil {
		t.Fatalf("seed probe row: %v", err)
	}

	scope := &common.DataScopeReq{
		UserID:   7,
		RoleKeys: []string{"restricted_role"},
		Mode:     common.DataScopeModeAll,
	}
	failed := applyRoleDataScopePolicy(db, scope)
	if !failed {
		t.Fatalf("expected policy lookup failure to be reported")
	}
	if scope.Mode == common.DataScopeModeAll {
		t.Fatalf("policy failure must not fall back to mode=all")
	}
	if len(scope.DeptIDs) != 0 {
		t.Fatalf("expected empty dept ids under failure, got %+v", scope.DeptIDs)
	}

	var rows []dataScopeDenyProbe
	if err := db.Model(&dataScopeDenyProbe{}).Scopes(database.WithDataScope(scope)).Find(&rows).Error; err != nil {
		t.Fatalf("scoped query failed: %v", err)
	}
	if len(rows) != 0 {
		t.Fatalf("expected zero rows under failed policy lookup, got %+v", rows)
	}
}

// TestPolicyFailureRequestRejected 端到端验证：策略读取失败时中间件
// 中止请求，返回 fail-closed 错误 key，处理函数不会执行。
func TestPolicyFailureRequestRejected(t *testing.T) {
	db := testmysql.Open(t)
	if err := db.Exec("CREATE TABLE system_user (id BIGINT PRIMARY KEY, dept_id BIGINT)").Error; err != nil {
		t.Fatalf("create user table: %v", err)
	}
	if err := db.Exec("CREATE TABLE system_role_data_scope (unrelated_col BIGINT)").Error; err != nil {
		t.Fatalf("create broken policy table: %v", err)
	}
	storeCachedTableExistence(db, (&SystemRoleDataScope{}).TableName(), true)

	gin.SetMode(gin.TestMode)
	engine := gin.New()
	handlerReached := false
	engine.Use(func(c *gin.Context) {
		c.Set("userId", uint64(7))
		c.Set("roleKeys", []string{"restricted_role"})
		c.Next()
	})
	engine.Use(DataScopeMiddleware(db))
	engine.GET("/scoped", func(c *gin.Context) {
		handlerReached = true
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/scoped", nil)
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, req)

	if handlerReached {
		t.Fatalf("handler must not run when data scope policy lookup fails")
	}
	body := recorder.Body.String()
	if !strings.Contains(body, "common.serverError") {
		t.Fatalf("expected fail-closed error key in body, got %s", body)
	}
}

// TestAdminBypassesPolicyLookup 验证管理员在策略读取失败时仍可访问
// （管理员不走角色数据范围策略，属既有合同行为）。
func TestAdminBypassesPolicyLookup(t *testing.T) {
	db := testmysql.Open(t)
	if err := db.Exec("CREATE TABLE system_user (id BIGINT PRIMARY KEY, dept_id BIGINT)").Error; err != nil {
		t.Fatalf("create user table: %v", err)
	}
	if err := db.Exec("CREATE TABLE system_role_data_scope (unrelated_col BIGINT)").Error; err != nil {
		t.Fatalf("create broken policy table: %v", err)
	}
	storeCachedTableExistence(db, (&SystemRoleDataScope{}).TableName(), true)

	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(func(c *gin.Context) {
		c.Set("userId", uint64(1))
		c.Set("roleKeys", []string{"admin"})
		c.Next()
	})
	engine.Use(DataScopeMiddleware(db))
	engine.GET("/scoped", func(c *gin.Context) {
		scope := common.GetDataScope(c)
		if scope == nil || !scope.IsAdmin || scope.Mode != common.DataScopeModeAll {
			t.Errorf("expected admin scope mode all, got %+v", scope)
		}
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/scoped", nil)
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected admin request to pass, got %d", recorder.Code)
	}
}

func TestEmptyCustomPolicyDoesNotBecomeAll(t *testing.T) {
	policies := []SystemRoleDataScope{{RoleKey: "restricted", Mode: common.DataScopeModeCustom}}
	if got := resolveDataScopeMode(policies); got != common.DataScopeModeCustom {
		t.Fatalf("empty custom policy must remain custom, got %q", got)
	}
}

func TestEmptyCustomPolicyReturnsNoRows(t *testing.T) {
	db := testmysql.Open(t)
	if err := db.AutoMigrate(&SystemRoleDataScope{}, &dataScopeDenyProbe{}); err != nil {
		t.Fatalf("migrate policy and probe tables: %v", err)
	}
	if err := db.Create(&SystemRoleDataScope{RoleKey: "restricted", Mode: common.DataScopeModeCustom}).Error; err != nil {
		t.Fatalf("create empty custom policy: %v", err)
	}
	if err := db.Create(&dataScopeDenyProbe{DeptID: 42}).Error; err != nil {
		t.Fatalf("create probe row: %v", err)
	}

	scope := &common.DataScopeReq{UserID: 7, RoleKeys: []string{"restricted"}, Mode: common.DataScopeModeAll}
	if applyRoleDataScopePolicy(db, scope) {
		t.Fatal("policy lookup unexpectedly failed")
	}
	if scope.Mode != common.DataScopeModeCustom || len(scope.DeptIDs) != 0 {
		t.Fatalf("empty custom policy must remain restrictive, got %+v", scope)
	}
	var rows []dataScopeDenyProbe
	if err := db.Scopes(database.WithDataScope(scope)).Find(&rows).Error; err != nil {
		t.Fatalf("scoped query failed: %v", err)
	}
	if len(rows) != 0 {
		t.Fatalf("empty custom policy returned %d rows", len(rows))
	}
}

func TestMissingPolicyTableRejectsRequest(t *testing.T) {
	db := testmysql.Open(t)
	if err := db.Exec("CREATE TABLE system_user (id BIGINT PRIMARY KEY, dept_id BIGINT)").Error; err != nil {
		t.Fatalf("create user table: %v", err)
	}

	gin.SetMode(gin.TestMode)
	engine := gin.New()
	handlerReached := false
	engine.Use(func(c *gin.Context) {
		c.Set("userId", uint64(7))
		c.Set("roleKeys", []string{"restricted_role"})
		c.Next()
	})
	engine.Use(DataScopeMiddleware(db))
	engine.GET("/scoped", func(c *gin.Context) {
		handlerReached = true
		c.Status(http.StatusOK)
	})

	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/scoped", nil))
	if handlerReached {
		t.Fatalf("missing policy table must reject request, status=%d reached=%t", recorder.Code, handlerReached)
	}
	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), `"code":500`) || !strings.Contains(recorder.Body.String(), "common.serverError") {
		t.Fatalf("missing policy table must return fail-closed business error, status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestPolicyCoverageRejectsMissingRole(t *testing.T) {
	policies := []SystemRoleDataScope{{RoleKey: "known", Mode: common.DataScopeModeAll}}
	if rolePoliciesCoverAll([]string{"known", "missing"}, policies) {
		t.Fatal("a missing role policy must not inherit all scope")
	}
	if !rolePoliciesCoverAll([]string{"known", "known"}, policies) {
		t.Fatal("duplicate role keys should not require duplicate rows")
	}
}

func TestMigrateDataScopePolicyDoesNotBackfillExistingRoles(t *testing.T) {
	db := testmysql.Open(t)
	if err := db.Exec(`CREATE TABLE system_role (
		id BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
		role_key VARCHAR(64) NOT NULL,
		deleted_at DATETIME NULL
	)`).Error; err != nil {
		t.Fatalf("create role table: %v", err)
	}
	if err := db.Exec(`INSERT INTO system_role (role_key, deleted_at) VALUES
		('legacy', NULL), ('existing', NULL), ('archived', NOW())`).Error; err != nil {
		t.Fatalf("seed roles: %v", err)
	}
	if err := db.AutoMigrate(&SystemRoleDataScope{}); err != nil {
		t.Fatalf("migrate policy table: %v", err)
	}
	if err := db.Create(&SystemRoleDataScope{RoleKey: "existing", Mode: common.DataScopeModeCustom, DeptIDs: "42"}).Error; err != nil {
		t.Fatalf("seed existing policy: %v", err)
	}
	if err := MigrateDataScopePolicy(db); err != nil {
		t.Fatalf("migrate policies: %v", err)
	}
	if err := MigrateDataScopePolicy(db); err != nil {
		t.Fatalf("repeat migration: %v", err)
	}
	var policies []SystemRoleDataScope
	if err := db.Order("role_key").Find(&policies).Error; err != nil {
		t.Fatalf("load policies: %v", err)
	}
	if len(policies) != 1 || policies[0].RoleKey != "existing" || policies[0].Mode != common.DataScopeModeCustom || policies[0].DeptIDs != "42" {
		t.Fatalf("runtime migration must preserve existing policies without backfilling: %+v", policies)
	}
}

func TestRoleDataScopePolicyReadSeesCommittedUpdate(t *testing.T) {
	db := testmysql.Open(t)
	if err := db.AutoMigrate(&SystemRoleDataScope{}); err != nil {
		t.Fatalf("migrate policy table: %v", err)
	}
	if err := db.Create(&SystemRoleDataScope{RoleKey: "restricted", Mode: common.DataScopeModeAll}).Error; err != nil {
		t.Fatalf("seed policy: %v", err)
	}
	first, ok := loadRoleDataScopePolicies(db, []string{"restricted"})
	if !ok || len(first) != 1 || first[0].Mode != common.DataScopeModeAll {
		t.Fatalf("unexpected initial policy: %+v, ok=%t", first, ok)
	}
	if err := db.Model(&SystemRoleDataScope{}).Where("role_key = ?", "restricted").Update("mode", common.DataScopeModeSelf).Error; err != nil {
		t.Fatalf("update policy: %v", err)
	}
	second, ok := loadRoleDataScopePolicies(db, []string{"restricted"})
	if !ok || len(second) != 1 || second[0].Mode != common.DataScopeModeSelf {
		t.Fatalf("policy read must see committed restriction: %+v, ok=%t", second, ok)
	}
}
