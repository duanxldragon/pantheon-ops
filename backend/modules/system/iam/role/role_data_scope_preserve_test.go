package role

import (
	"testing"

	"github.com/duanxldragon/pantheon-base/backend/modules/system/iam/permission"
	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
)

// TestRoleService_UpdateRolePreservesCustomDataScopeDeptIDs 验证（F06）：
// 配置自定义数据范围 → 编辑角色常规字段 → 部门集合与生效范围保持不变。
func TestRoleService_UpdateRolePreservesCustomDataScopeDeptIDs(t *testing.T) {
	db := setupRoleTestDB(t)
	s := NewRoleService(db)

	role, err := s.CreateRole(&RoleCreateReq{
		RoleName:  "Scoped Role",
		RoleKey:   "scoped_role",
		Status:    1,
		DataScope: common.DataScopeModeAll,
	})
	if err != nil {
		t.Fatalf("create role: %v", err)
	}

	// 迁移 permission 侧的策略表并配置自定义部门集合。
	p := permission.NewPermissionService(db)
	if err := db.AutoMigrate(&permission.PermissionRoleDataScopePolicy{}); err != nil {
		t.Fatalf("migrate permission policy table: %v", err)
	}
	if _, err := p.UpdateDataScopePolicy("scoped_role", &permission.PermissionDataScopePolicyUpdateReq{
		Mode:    common.DataScopeModeCustom,
		DeptIDs: []uint64{10, 20, 30},
	}); err != nil {
		t.Fatalf("configure custom data scope: %v", err)
	}

	// 角色常规编辑：角色列表返回的 dataScope 快照是当前策略模式 custom，
	// 前端表单不携带部门集合，仅回传模式值。
	updated, err := s.UpdateRole(role.ID, &RoleUpdateReq{
		RoleName:       "Scoped Role Renamed",
		RoleKey:        "scoped_role",
		Sort:           3,
		Status:         1,
		MenuIDs:        []uint64{1},
		PermissionKeys: []string{"sys:user:list"},
		DataScope:      common.DataScopeModeCustom,
	})
	if err != nil {
		t.Fatalf("update role: %v", err)
	}
	if updated.RoleName != "Scoped Role Renamed" {
		t.Fatalf("expected role rename applied, got %s", updated.RoleName)
	}

	// 编辑后数据范围策略必须仍是 custom 且保留全部部门 ID。
	policies, err := p.ListDataScopePolicies(&permission.PermissionDataScopeQuery{RoleKey: "scoped_role"})
	if err != nil {
		t.Fatalf("list data scope policies: %v", err)
	}
	if len(policies.Items) != 1 {
		t.Fatalf("expected one policy, got %d", len(policies.Items))
	}
	pol := policies.Items[0]
	if pol.Mode != common.DataScopeModeCustom {
		t.Fatalf("expected custom mode preserved, got %q", pol.Mode)
	}
	if len(pol.DeptIDs) != 3 || pol.DeptIDs[0] != 10 || pol.DeptIDs[1] != 20 || pol.DeptIDs[2] != 30 {
		t.Fatalf("expected dept ids [10 20 30] preserved, got %+v", pol.DeptIDs)
	}
	if _, err := s.UpdateRole(role.ID, &RoleUpdateReq{
		RoleName:  "Scoped Role Renamed",
		RoleKey:   "scoped_role_new",
		Status:    1,
		DataScope: common.DataScopeModeCustom,
	}); err != nil {
		t.Fatalf("rename role key: %v", err)
	}
	policies, err = p.ListDataScopePolicies(&permission.PermissionDataScopeQuery{RoleKey: "scoped_role_new"})
	if err != nil || len(policies.Items) != 1 {
		t.Fatalf("load policy after role key change: policies=%+v err=%v", policies, err)
	}
	if got := policies.Items[0].DeptIDs; len(got) != 3 || got[0] != 10 || got[1] != 20 || got[2] != 30 {
		t.Fatalf("role key change lost custom departments: %+v", got)
	}
	if policies.Items[0].ID != pol.ID {
		t.Fatalf("role key change created a new policy row: old=%d new=%d", pol.ID, policies.Items[0].ID)
	}
	var oldCount int64
	if err := db.Model(&permission.PermissionRoleDataScopePolicy{}).Where("role_key = ?", "scoped_role").Count(&oldCount).Error; err != nil || oldCount != 0 {
		t.Fatalf("old role policy remains after rename: count=%d err=%v", oldCount, err)
	}
}

// TestRoleService_UpdateRoleSwitchToCustomWithoutPolicyKeepsEmpty 验证从 all
// 切换到 custom 且从未配置部门时，策略落库为空集合（等待数据范围入口配置），
// 查询侧按空集合拒绝而不是放大权限。
func TestRoleService_UpdateRoleSwitchToCustomWithoutPolicyKeepsEmpty(t *testing.T) {
	db := setupRoleTestDB(t)
	s := NewRoleService(db)

	role, err := s.CreateRole(&RoleCreateReq{
		RoleName:  "Plain Role",
		RoleKey:   "plain_role",
		Status:    1,
		DataScope: common.DataScopeModeAll,
	})
	if err != nil {
		t.Fatalf("create role: %v", err)
	}

	if _, err := s.UpdateRole(role.ID, &RoleUpdateReq{
		RoleName:  "Plain Role",
		RoleKey:   "plain_role",
		Status:    1,
		DataScope: common.DataScopeModeCustom,
	}); err != nil {
		t.Fatalf("update role to custom: %v", err)
	}

	var policy permission.PermissionRoleDataScopePolicy
	err = db.Where("role_key = ?", "plain_role").Limit(1).Find(&policy).Error
	if err != nil {
		t.Fatalf("load policy: %v", err)
	}
	if policy.Mode != common.DataScopeModeCustom {
		t.Fatalf("expected custom mode, got %q", policy.Mode)
	}
	if policy.DeptIDs != "" {
		t.Fatalf("expected empty dept ids until configured, got %q", policy.DeptIDs)
	}
}
