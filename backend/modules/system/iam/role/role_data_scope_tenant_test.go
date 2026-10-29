package role

import (
	"testing"

	"github.com/duanxldragon/pantheon-base/backend/modules/system/iam/permission"
	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
	"github.com/duanxldragon/pantheon-base/backend/pkg/tenant"
)

// 角色列表回显的 dataScope 必须来自调用方租户自己的策略行（验收 F13）。
//
// 修复前 loadRoleDataScopes 只按 role_key 查，于是两个租户的同名角色共享一份策略，
// 且「最后扫到的那行」决定所有人看到的值——租户 A 的角色列表会显示租户 B（或平台行）
// 的数据范围。这条测试把「同名角色 + 三种模式」摆在一起，断言每个租户只看到自己那份。
func TestListRolesEchoesOnlyTheCallersDataScope(t *testing.T) {
	db := setupRoleTestDB(t)

	roles := []SystemRole{
		{TenantID: 101, RoleName: "A 运营", RoleKey: "ops", Status: 1, Sort: 1},
		{TenantID: 202, RoleName: "B 运营", RoleKey: "ops", Status: 1, Sort: 1},
	}
	if err := db.Create(&roles).Error; err != nil {
		t.Fatalf("seed roles: %v", err)
	}

	policies := []permission.PermissionRoleDataScopePolicy{
		{TenantID: tenant.PlatformGlobalTenantID, RoleKey: "ops", Mode: common.DataScopeModeAll},
		{TenantID: 101, RoleKey: "ops", Mode: common.DataScopeModeSelf},
		{TenantID: 202, RoleKey: "ops", Mode: common.DataScopeModeDept},
	}
	if err := db.Create(&policies).Error; err != nil {
		t.Fatalf("seed data scope policies: %v", err)
	}

	for _, tc := range []struct {
		name     string
		tenantID uint64
		want     string
	}{
		{name: "tenant A echoes its own self", tenantID: 101, want: common.DataScopeModeSelf},
		{name: "tenant B echoes its own dept", tenantID: 202, want: common.DataScopeModeDept},
	} {
		t.Run(tc.name, func(t *testing.T) {
			service := NewRoleService(db).WithTenantContext(&tenant.Context{TenantID: tc.tenantID, Mode: tenant.ModeMulti})
			page, err := service.ListRoles(nil)
			if err != nil {
				t.Fatalf("list roles: %v", err)
			}
			if page.Total != 1 || len(page.Items) != 1 {
				t.Fatalf("tenant %d owns 1 role, expected 1 row, got total=%d items=%+v",
					tc.tenantID, page.Total, page.Items)
			}
			if got := page.Items[0]; got.RoleKey != "ops" || got.DataScope != tc.want {
				t.Fatalf("tenant %d must echo %q for its own role, got %+v", tc.tenantID, tc.want, got)
			}
		})
	}
}

// 兼容基线：没有租户上下文（compat / 平台调用）时行为不变，读到的是平台行。
func TestListRolesDataScopeKeepsCompatBaseline(t *testing.T) {
	db := setupRoleTestDB(t)

	if err := db.Create(&SystemRole{TenantID: tenant.PlatformGlobalTenantID, RoleName: "平台运营", RoleKey: "ops", Status: 1, Sort: 1}).Error; err != nil {
		t.Fatalf("seed platform role: %v", err)
	}
	if err := db.Create(&permission.PermissionRoleDataScopePolicy{TenantID: tenant.PlatformGlobalTenantID, RoleKey: "ops", Mode: common.DataScopeModeDeptAndChildren}).Error; err != nil {
		t.Fatalf("seed platform policy: %v", err)
	}

	page, err := NewRoleService(db).ListRoles(nil)
	if err != nil {
		t.Fatalf("list roles: %v", err)
	}
	if page.Total != 1 || len(page.Items) != 1 {
		t.Fatalf("expected the platform role only, got total=%d items=%+v", page.Total, page.Items)
	}
	if got := page.Items[0].DataScope; got != common.DataScopeModeDeptAndChildren {
		t.Fatalf("compat callers must keep reading the platform row, got %q", got)
	}
}
