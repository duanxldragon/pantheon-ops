package permission

import (
	"strings"
	"testing"

	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
	"github.com/duanxldragon/pantheon-base/backend/pkg/tenant"
	"github.com/duanxldragon/pantheon-base/backend/pkg/testmysql"

	"gorm.io/gorm"
)

// 数据权限控制台的租户维度（验收 F7/F8/F9/F11）。
//
// role_key 只在租户内唯一，所以两个租户可以各有一个 `ops`。修复前
// ListDataScopePolicies 列出全部租户的角色、UpdateDataScopePolicy 以 role_key
// 为唯一键 upsert 且 tenant_id 恒为 0 —— 于是一个租户管理员既能看到别人的角色，
// 又能改写别人角色对应的数据范围。
const (
	dsScopeTenantA     uint64 = 101
	dsScopeTenantB     uint64 = 202
	dsScopeShared             = "ops"         // 两个租户都有的角色
	dsScopeTenantBOnly        = "b_only"      // 只有租户 B 有的角色
	dsScopePlatform           = "platform_op" // 只有平台（tenant 0）有的角色
)

type dsPolicyRow struct {
	TenantID uint64 `gorm:"column:tenant_id"`
	RoleKey  string `gorm:"column:role_key"`
	Mode     string `gorm:"column:mode"`
	DeptIDs  string `gorm:"column:dept_ids"`
}

func (dsPolicyRow) TableName() string { return "system_role_data_scope" }

// setupDataScopeTenantDB 建一个「同一 role_key 被两个租户 + 平台同时持有」的环境。
// 这里手写 DDL 而不用 setupPermissionTestDB：那个夹具的 system_role.role_key 是
// 全局唯一索引，无法表达「两个租户各有一个 ops」。
func setupDataScopeTenantDB(t *testing.T) *gorm.DB {
	t.Helper()
	db := testmysql.Open(t)

	for _, stmt := range []string{
		`CREATE TABLE system_role (
			id BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
			tenant_id BIGINT UNSIGNED NOT NULL DEFAULT 0,
			role_name VARCHAR(64) NOT NULL,
			role_key VARCHAR(64) NOT NULL,
			sort INT NOT NULL DEFAULT 0,
			status INT NOT NULL DEFAULT 1,
			deleted_at DATETIME(3) NULL,
			UNIQUE KEY uk_system_role_tenant_key (tenant_id, role_key)
		)`,
		`CREATE TABLE system_dept (
			id BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
			tenant_id BIGINT UNSIGNED NOT NULL DEFAULT 0,
			dept_name VARCHAR(64) NOT NULL DEFAULT '',
			deleted_at DATETIME(3) NULL
		)`,
	} {
		if err := db.Exec(stmt).Error; err != nil {
			t.Fatalf("create fixture table: %v", err)
		}
	}
	if err := db.AutoMigrate(&PermissionRoleDataScopePolicy{}); err != nil {
		t.Fatalf("migrate data scope policy table: %v", err)
	}

	roles := []struct {
		tenantID uint64
		roleKey  string
	}{
		{tenant.PlatformGlobalTenantID, dsScopePlatform},
		{dsScopeTenantA, dsScopeShared},
		{dsScopeTenantA, "billing"},
		{dsScopeTenantB, dsScopeShared},
		{dsScopeTenantB, dsScopeTenantBOnly},
	}
	for _, role := range roles {
		if err := db.Exec(
			"INSERT INTO system_role (tenant_id, role_name, role_key, sort, status) VALUES (?, ?, ?, 1, 1)",
			role.tenantID, role.roleKey, role.roleKey,
		).Error; err != nil {
			t.Fatalf("seed role %s@%d: %v", role.roleKey, role.tenantID, err)
		}
	}

	policies := []dsPolicyRow{
		{TenantID: tenant.PlatformGlobalTenantID, RoleKey: dsScopeShared, Mode: common.DataScopeModeAll},
		{TenantID: dsScopeTenantA, RoleKey: dsScopeShared, Mode: common.DataScopeModeSelf},
		{TenantID: dsScopeTenantB, RoleKey: dsScopeShared, Mode: common.DataScopeModeSelf},
	}
	for i := range policies {
		if err := db.Create(&policies[i]).Error; err != nil {
			t.Fatalf("seed policy %+v: %v", policies[i], err)
		}
	}

	// 部门 id 是全局序列：10 属于租户 A，20 属于租户 B。
	for _, dept := range []struct {
		id       uint64
		tenantID uint64
	}{{10, dsScopeTenantA}, {20, dsScopeTenantB}} {
		if err := db.Exec("INSERT INTO system_dept (id, tenant_id, dept_name) VALUES (?, ?, ?)",
			dept.id, dept.tenantID, "dept").Error; err != nil {
			t.Fatalf("seed dept %d: %v", dept.id, err)
		}
	}

	return db
}

func tenantScopedPermissionService(db *gorm.DB, tenantID uint64) *PermissionService {
	return NewPermissionService(db).WithTenantContext(&tenant.Context{TenantID: tenantID, Mode: tenant.ModeMulti})
}

func loadPolicyRows(t *testing.T, db *gorm.DB, roleKey string) []dsPolicyRow {
	t.Helper()
	var rows []dsPolicyRow
	if err := db.Where("role_key = ?", roleKey).Order("tenant_id asc").Find(&rows).Error; err != nil {
		t.Fatalf("load policy rows for %q: %v", roleKey, err)
	}
	return rows
}

// TestListDataScopePoliciesIsTenantScoped 是 F7 的回归：租户管理员只应看到自己
// 租户的角色，且同名角色不会出现两行。
func TestListDataScopePoliciesIsTenantScoped(t *testing.T) {
	db := setupDataScopeTenantDB(t)

	list, err := tenantScopedPermissionService(db, dsScopeTenantA).ListDataScopePolicies(nil)
	if err != nil {
		t.Fatalf("list data scope policies: %v", err)
	}
	if list.Total != 2 || len(list.Items) != 2 {
		t.Fatalf("tenant %d owns 2 roles, expected 2 rows, got %d: %+v", dsScopeTenantA, list.Total, list.Items)
	}

	seen := make(map[string]PermissionDataScopePolicyResp, len(list.Items))
	for _, item := range list.Items {
		if _, dup := seen[item.RoleKey]; dup {
			t.Fatalf("duplicate role key %q in tenant-scoped list", item.RoleKey)
		}
		seen[item.RoleKey] = item
	}
	for _, foreign := range []string{dsScopePlatform, dsScopeTenantBOnly} {
		if _, ok := seen[foreign]; ok {
			t.Fatalf("tenant %d must not see foreign role %q", dsScopeTenantA, foreign)
		}
	}
	if got := seen[dsScopeShared]; got.Mode != common.DataScopeModeSelf || !got.PolicyExists {
		t.Fatalf("expected tenant %d's own %q policy, got %+v", dsScopeTenantA, common.DataScopeModeSelf, got)
	}
	if got := seen["billing"]; got.Mode != common.DataScopeModeAll || got.PolicyExists {
		t.Fatalf("expected the default all mode for a policy-less role, got %+v", got)
	}
}

// TestUpdateDataScopePolicyWritesToCallersTenant 是 F8/F9 的回归：写操作必须落在
// 调用方租户上，既不能覆盖平台行，也不能覆盖另一个租户的同名角色。
func TestUpdateDataScopePolicyWritesToCallersTenant(t *testing.T) {
	db := setupDataScopeTenantDB(t)

	if _, err := tenantScopedPermissionService(db, dsScopeTenantA).UpdateDataScopePolicy(dsScopeShared, &PermissionDataScopePolicyUpdateReq{
		Mode: common.DataScopeModeDept,
	}); err != nil {
		t.Fatalf("update data scope policy: %v", err)
	}

	rows := loadPolicyRows(t, db, dsScopeShared)
	if len(rows) != 3 {
		t.Fatalf("expected exactly 3 policy rows (platform + 2 tenants), got %d: %+v", len(rows), rows)
	}
	got := make(map[uint64]string, len(rows))
	for _, row := range rows {
		got[row.TenantID] = row.Mode
	}
	if got[dsScopeTenantA] != common.DataScopeModeDept {
		t.Fatalf("expected tenant %d's row to be updated, got %q", dsScopeTenantA, got[dsScopeTenantA])
	}
	if got[dsScopeTenantB] != common.DataScopeModeSelf {
		t.Fatalf("tenant %d's row must not be touched, got %q", dsScopeTenantB, got[dsScopeTenantB])
	}
	if got[tenant.PlatformGlobalTenantID] != common.DataScopeModeAll {
		t.Fatalf("the platform row must not be touched, got %q", got[tenant.PlatformGlobalTenantID])
	}
}

// TestUpdateDataScopePolicyRejectsForeignRoleKey 是 F8 的另一半：租户不能通过
// 猜 role_key 去给别人的角色建策略。
func TestUpdateDataScopePolicyRejectsForeignRoleKey(t *testing.T) {
	db := setupDataScopeTenantDB(t)

	_, err := tenantScopedPermissionService(db, dsScopeTenantA).UpdateDataScopePolicy(dsScopeTenantBOnly, &PermissionDataScopePolicyUpdateReq{
		Mode: common.DataScopeModeAll,
	})
	if err == nil || !strings.Contains(err.Error(), "permission.role.invalid") {
		t.Fatalf("expected permission.role.invalid for a foreign role key, got %v", err)
	}
	if rows := loadPolicyRows(t, db, dsScopeTenantBOnly); len(rows) != 0 {
		t.Fatalf("a rejected update must not create a policy row, got %+v", rows)
	}
}

// TestUpdateDataScopePolicyRejectsForeignDeptIDs 是 F11 的回归：部门 id 是全局
// 序列，同一串数字在不同租户指向不同部门，所以自定义范围只能引用本租户的部门。
func TestUpdateDataScopePolicyRejectsForeignDeptIDs(t *testing.T) {
	db := setupDataScopeTenantDB(t)
	service := tenantScopedPermissionService(db, dsScopeTenantA)

	_, err := service.UpdateDataScopePolicy(dsScopeShared, &PermissionDataScopePolicyUpdateReq{
		Mode:    common.DataScopeModeCustom,
		DeptIDs: []uint64{20}, // 属于租户 B
	})
	if err == nil || !strings.Contains(err.Error(), "permission.data_scope.dept_invalid") {
		t.Fatalf("expected permission.data_scope.dept_invalid for a foreign dept id, got %v", err)
	}

	updated, err := service.UpdateDataScopePolicy(dsScopeShared, &PermissionDataScopePolicyUpdateReq{
		Mode:    common.DataScopeModeCustom,
		DeptIDs: []uint64{10}, // 属于租户 A
	})
	if err != nil {
		t.Fatalf("update with an own-tenant dept id must succeed: %v", err)
	}
	if updated.Mode != common.DataScopeModeCustom || len(updated.DeptIDs) != 1 || updated.DeptIDs[0] != 10 {
		t.Fatalf("expected custom scope [10], got %+v", updated)
	}
}

// TestWithTenantContextKeepsCompatBaseline 固定回退语义：没有租户上下文（平台内部调用）
// 时服务行为不变；compat 上下文把作用域钉在平台全局 0，写入同样落在 tenant_id=0。
func TestWithTenantContextKeepsCompatBaseline(t *testing.T) {
	db := setupDataScopeTenantDB(t)
	base := NewPermissionService(db)

	if base.WithTenantContext(nil) != base {
		t.Fatalf("a nil context must return the receiver unchanged")
	}
	compat := base.WithTenantContext(&tenant.Context{TenantID: tenant.PlatformGlobalTenantID, Mode: tenant.ModeCompat})
	if compat == base {
		t.Fatalf("a compat context must produce a bound view")
	}
	if compat.TenantOwnerID() != tenant.PlatformGlobalTenantID {
		t.Fatalf("compat writes must stay on the platform scope, got tenant %d", compat.TenantOwnerID())
	}

	if _, err := compat.UpdateDataScopePolicy(dsScopePlatform, &PermissionDataScopePolicyUpdateReq{
		Mode: common.DataScopeModeDept,
	}); err != nil {
		t.Fatalf("platform update: %v", err)
	}
	rows := loadPolicyRows(t, db, dsScopePlatform)
	if len(rows) != 1 || rows[0].TenantID != tenant.PlatformGlobalTenantID {
		t.Fatalf("compat writes must stay on the platform scope, got %+v", rows)
	}
}
