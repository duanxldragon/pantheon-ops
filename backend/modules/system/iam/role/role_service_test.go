package iam

import (
	"strings"
	"testing"
	"time"

	"pantheon-ops/backend/pkg/common"
	"pantheon-ops/backend/pkg/database"
	"pantheon-ops/backend/pkg/testmysql"

	"gorm.io/gorm"
)

func setupRoleTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db := testmysql.Open(t)

	// 迁移模型
	_ = db.AutoMigrate(&SystemRole{}, &SystemRolePermission{}, &database.CasbinRule{})
	_ = db.Exec("CREATE TABLE IF NOT EXISTS system_menu (id INTEGER PRIMARY KEY, page_perm TEXT, perms TEXT, type TEXT)")
	_ = db.Exec("CREATE TABLE IF NOT EXISTS system_role_menu (role_id INTEGER, menu_id INTEGER)")
	_ = db.Exec("CREATE TABLE IF NOT EXISTS system_user_role (user_id INTEGER, role_id INTEGER)")
	_ = db.Exec("CREATE TABLE IF NOT EXISTS system_user (id INTEGER PRIMARY KEY, username TEXT, nickname TEXT, dept_id INTEGER, post_id INTEGER, status INTEGER, created_at DATETIME, updated_at DATETIME, deleted_at DATETIME)")
	_ = db.Exec("CREATE TABLE IF NOT EXISTS system_dept (id INTEGER PRIMARY KEY, dept_name TEXT)")
	_ = db.Exec("CREATE TABLE IF NOT EXISTS system_post (id INTEGER PRIMARY KEY, post_name TEXT)")

	// 插入菜单测试数据
	_ = db.Exec("INSERT INTO system_menu (id, page_perm, perms, type) VALUES (1, 'sys:user:list', '', 'C')")
	_ = db.Exec("INSERT INTO system_menu (id, page_perm, perms, type) VALUES (2, '', 'sys:user:create', 'F')")

	return db
}

func TestRoleService_CreateRole(t *testing.T) {
	db := setupRoleTestDB(t)
	s := NewRoleService(db)

	req := &RoleCreateReq{
		RoleName:       "Test Role",
		RoleKey:        "test_role",
		Sort:           1,
		Status:         1,
		MenuIDs:        []uint64{1},
		PermissionKeys: []string{"sys:user:list"},
	}

	// 1. 成功创建
	resp, err := s.CreateRole(req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resp.RoleName != "Test Role" {
		t.Errorf("expected role name Test Role, got %s", resp.RoleName)
	}

	// 2. RoleKey 重复
	_, err = s.CreateRole(req)
	if err == nil || common.ErrMessage(err) != "role.key.exists" {
		t.Errorf("expected role key exists error, got %v", err)
	}

	// 3. 无效权限 Key
	req.RoleKey = "invalid_perm"
	req.PermissionKeys = []string{"nonexistent:perm"}
	_, err = s.CreateRole(req)
	if err == nil || common.ErrMessage(err) != "role.permission.invalid" {
		t.Errorf("expected permission invalid error, got %v", err)
	}
}

func TestRoleService_ListRolesExcludesActionMenuBindings(t *testing.T) {
	db := setupRoleTestDB(t)
	s := NewRoleService(db)

	role, err := s.CreateRole(&RoleCreateReq{
		RoleName:       "Navigator",
		RoleKey:        "navigator",
		Status:         1,
		MenuIDs:        []uint64{1},
		PermissionKeys: []string{"sys:user:list", "sys:user:create"},
	})
	if err != nil {
		t.Fatalf("create role: %v", err)
	}
	if err := db.Exec("INSERT INTO system_role_menu (role_id, menu_id) VALUES (?, ?)", role.ID, 2).Error; err != nil {
		t.Fatalf("seed legacy action menu binding: %v", err)
	}

	resp, err := s.ListRoles(&RoleListQuery{RoleKey: "navigator", Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("list roles: %v", err)
	}
	if len(resp.Items) != 1 {
		t.Fatalf("expected one role, got %d", len(resp.Items))
	}
	if len(resp.Items[0].MenuIDs) != 1 || resp.Items[0].MenuIDs[0] != 1 {
		t.Fatalf("expected only navigation menu id 1, got %+v", resp.Items[0].MenuIDs)
	}
}

func TestRoleService_MigrateSeedsAdminRoleAndBinding(t *testing.T) {
	db := setupRoleTestDB(t)
	if err := db.Exec("CREATE TABLE IF NOT EXISTS system_user (id INTEGER PRIMARY KEY, username TEXT)").Error; err != nil {
		t.Fatalf("create user table: %v", err)
	}
	if err := db.Exec("INSERT INTO system_user (id, username) VALUES (1, 'admin')").Error; err != nil {
		t.Fatalf("seed admin user: %v", err)
	}

	s := NewRoleService(db)
	if err := s.Migrate(); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	var adminRole SystemRole
	if err := db.Where("role_key = ?", "admin").First(&adminRole).Error; err != nil {
		t.Fatalf("load admin role: %v", err)
	}
	if adminRole.Status != 1 {
		t.Fatalf("expected admin role status 1, got %d", adminRole.Status)
	}

	var bindingCount int64
	if err := db.Table("system_user_role").Where("user_id = ? AND role_id = ?", 1, adminRole.ID).Count(&bindingCount).Error; err != nil {
		t.Fatalf("count admin binding: %v", err)
	}
	if bindingCount != 1 {
		t.Fatalf("expected admin binding count 1, got %d", bindingCount)
	}
}

func TestRoleService_BootstrapSeedsAdminRoleAndBinding(t *testing.T) {
	db := setupRoleTestDB(t)
	if err := db.Exec("INSERT INTO system_user (id, username) VALUES (1, 'admin')").Error; err != nil {
		t.Fatalf("seed admin user: %v", err)
	}

	s := NewRoleService(db)
	if err := s.Bootstrap(); err != nil {
		t.Fatalf("bootstrap: %v", err)
	}

	var adminRole SystemRole
	if err := db.Where("role_key = ?", "admin").First(&adminRole).Error; err != nil {
		t.Fatalf("load admin role: %v", err)
	}
	if adminRole.Status != 1 {
		t.Fatalf("expected admin role status 1, got %d", adminRole.Status)
	}

	var bindingCount int64
	if err := db.Table("system_user_role").Where("user_id = ? AND role_id = ?", 1, adminRole.ID).Count(&bindingCount).Error; err != nil {
		t.Fatalf("count admin binding: %v", err)
	}
	if bindingCount != 1 {
		t.Fatalf("expected admin binding count 1, got %d", bindingCount)
	}
}

func TestRoleService_DeleteRole(t *testing.T) {
	db := setupRoleTestDB(t)
	s := NewRoleService(db)

	// 创建占位角色占用 ID 1
	db.Create(&SystemRole{RoleName: "Placeholder", RoleKey: "placeholder"})

	// 创建初始角色
	createReq := &RoleCreateReq{
		RoleName: "Delete Me",
		RoleKey:  "delete_me",
		Status:   1,
	}
	roleResp, err := s.CreateRole(createReq)
	if err != nil {
		t.Fatalf("failed to create role: %v", err)
	}
	if err := db.Create(&database.CasbinRule{
		PType: "p",
		V0:    "delete_me",
		V1:    "/api/v1/system/test",
		V2:    "GET",
	}).Error; err != nil {
		t.Fatalf("failed to seed casbin policy: %v", err)
	}

	// 1. 成功删除
	err = s.DeleteRole(roleResp.ID)
	if err != nil {
		t.Errorf("expected no error for ID %d, got %v", roleResp.ID, err)
	}
	var deletedRole SystemRole
	if err := db.Unscoped().First(&deletedRole, roleResp.ID).Error; err != nil {
		t.Fatalf("failed to load deleted role: %v", err)
	}
	if !strings.HasPrefix(deletedRole.RoleKey, deletedRoleKeyPrefix) {
		t.Fatalf("expected archived role key, got %s", deletedRole.RoleKey)
	}
	var policyCount int64
	if err := db.Model(&database.CasbinRule{}).Where("ptype = ? AND v0 = ?", "p", "delete_me").Count(&policyCount).Error; err != nil {
		t.Fatalf("failed to count casbin policy: %v", err)
	}
	if policyCount != 0 {
		t.Fatalf("expected casbin policies to be removed, got %d", policyCount)
	}
	if _, err := s.CreateRole(createReq); err != nil {
		t.Fatalf("expected role key to be reusable after delete, got %v", err)
	}

	// 2. 删除超级管理员 (admin)
	adminRole := SystemRole{RoleName: "Admin", RoleKey: "admin"}
	db.Create(&adminRole)
	err = s.DeleteRole(adminRole.ID)
	if err == nil || common.ErrMessage(err) != "role.delete.error.protected" {
		t.Errorf("expected protected error for admin role, got %v", err)
	}

	// 3. 删除有用户的角色
	roleWithUser := SystemRole{RoleName: "Has User", RoleKey: "has_user"}
	db.Create(&roleWithUser)
	_ = db.Exec("INSERT INTO system_user_role (user_id, role_id) VALUES (1, ?)", roleWithUser.ID)

	err = s.DeleteRole(roleWithUser.ID)
	if err == nil || common.ErrMessage(err) != "role.delete.error.has_users" {
		t.Errorf("expected has_users error, got %v", err)
	}
}

func TestRoleService_MigrateReleasesLegacyDeletedRoleKey(t *testing.T) {
	db := setupRoleTestDB(t)
	s := NewRoleService(db)

	legacy := SystemRole{
		RoleName: "Legacy Role",
		RoleKey:  "legacy_role",
		Status:   1,
	}
	if err := db.Create(&legacy).Error; err != nil {
		t.Fatalf("failed to seed legacy role: %v", err)
	}
	if err := db.Model(&legacy).Update("deleted_at", time.Now()).Error; err != nil {
		t.Fatalf("failed to soft delete legacy role: %v", err)
	}

	if err := s.Migrate(); err != nil {
		t.Fatalf("expected migrate to succeed, got %v", err)
	}
	var repaired SystemRole
	if err := db.Unscoped().First(&repaired, legacy.ID).Error; err != nil {
		t.Fatalf("failed to load repaired role: %v", err)
	}
	if !strings.HasPrefix(repaired.RoleKey, deletedRoleKeyPrefix) {
		t.Fatalf("expected archived legacy role key, got %s", repaired.RoleKey)
	}

	if _, err := s.CreateRole(&RoleCreateReq{RoleName: "Legacy Role", RoleKey: "legacy_role", Status: 1}); err != nil {
		t.Fatalf("expected legacy role key to be reusable after migrate, got %v", err)
	}
}

func TestRoleService_ExportAndBatchStatus(t *testing.T) {
	db := setupRoleTestDB(t)
	s := NewRoleService(db)

	adminRole := SystemRole{ID: 1, RoleName: "Admin", RoleKey: "admin", Status: 1, Sort: 1}
	editorRole := SystemRole{ID: 2, RoleName: "Editor", RoleKey: "editor", Status: 1, Sort: 2}
	if err := db.Create(&adminRole).Error; err != nil {
		t.Fatalf("seed admin role: %v", err)
	}
	if err := db.Create(&editorRole).Error; err != nil {
		t.Fatalf("seed editor role: %v", err)
	}

	exported, err := s.ExportRoles(&RoleListQuery{RoleKey: "editor"})
	if err != nil {
		t.Fatalf("export roles: %v", err)
	}
	if len(exported.Rows) != 1 || exported.Rows[0][1] != "editor" {
		t.Fatalf("unexpected export rows: %+v", exported.Rows)
	}

	updated, err := s.BatchUpdateRoleStatus([]uint64{editorRole.ID}, 2)
	if err != nil {
		t.Fatalf("batch disable role: %v", err)
	}
	if updated != 1 {
		t.Fatalf("expected 1 updated role, got %d", updated)
	}
	var disabled SystemRole
	if err := db.First(&disabled, editorRole.ID).Error; err != nil {
		t.Fatalf("load disabled role: %v", err)
	}
	if disabled.Status != 2 {
		t.Fatalf("expected role status 2, got %d", disabled.Status)
	}

	if _, err := s.BatchUpdateRoleStatus([]uint64{adminRole.ID}, 2); err == nil || common.ErrMessage(err) != "role.update.error.protected" {
		t.Fatalf("expected protected error for admin batch disable, got %v", err)
	}
}

func TestRoleService_RoleMembersLifecycle(t *testing.T) {
	db := setupRoleTestDB(t)
	s := NewRoleService(db)

	role, err := s.CreateRole(&RoleCreateReq{
		RoleName: "Reviewer",
		RoleKey:  "reviewer",
		Status:   1,
	})
	if err != nil {
		t.Fatalf("create role: %v", err)
	}
	if err := db.Exec(
		"INSERT INTO system_user (id, username, nickname, status, created_at, updated_at) VALUES (11, 'alice', 'Alice', 1, NOW(), NOW()), (12, 'bob', 'Bob', 1, NOW(), NOW())",
	).Error; err != nil {
		t.Fatalf("seed users: %v", err)
	}

	addedCount, err := s.AddRoleMembers(role.ID, []uint64{11, 12})
	if err != nil {
		t.Fatalf("add role members: %v", err)
	}
	if addedCount != 2 {
		t.Fatalf("expected 2 members added, got %d", addedCount)
	}

	members, err := s.ListRoleMembers(role.ID, &RoleMemberQuery{Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("list role members: %v", err)
	}
	if members.Total != 2 || len(members.Items) != 2 {
		t.Fatalf("expected 2 bound members, got %+v", members)
	}

	candidates, err := s.ListAssignableUsers(role.ID, &RoleMemberQuery{Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("list role candidates: %v", err)
	}
	if candidates.Total != 0 {
		t.Fatalf("expected no remaining candidates, got %+v", candidates.Items)
	}

	removedCount, err := s.RemoveRoleMembers(role.ID, []uint64{11})
	if err != nil {
		t.Fatalf("remove role member: %v", err)
	}
	if removedCount != 1 {
		t.Fatalf("expected 1 member removed, got %d", removedCount)
	}

	members, err = s.ListRoleMembers(role.ID, &RoleMemberQuery{Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("list role members after remove: %v", err)
	}
	if members.Total != 1 || len(members.Items) != 1 || members.Items[0].Username != "bob" {
		t.Fatalf("expected only bob to remain bound, got %+v", members.Items)
	}

	candidates, err = s.ListAssignableUsers(role.ID, &RoleMemberQuery{Keyword: "alice", Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("list candidates after remove: %v", err)
	}
	if candidates.Total != 1 || len(candidates.Items) != 1 || candidates.Items[0].Username != "alice" {
		t.Fatalf("expected alice to return to candidates, got %+v", candidates.Items)
	}
}

func TestRoleService_RemoveAdminMemberProtection(t *testing.T) {
	db := setupRoleTestDB(t)
	s := NewRoleService(db)

	if err := db.Exec(
		"INSERT INTO system_user (id, username, nickname, status, created_at, updated_at) VALUES (1, 'admin', 'Administrator', 1, NOW(), NOW())",
	).Error; err != nil {
		t.Fatalf("seed admin user: %v", err)
	}

	adminRole := SystemRole{ID: 1, RoleName: "Admin", RoleKey: "admin", Status: 1, Sort: 1}
	if err := db.Create(&adminRole).Error; err != nil {
		t.Fatalf("seed admin role: %v", err)
	}
	if err := db.Exec("INSERT INTO system_user_role (user_id, role_id) VALUES (1, 1)").Error; err != nil {
		t.Fatalf("seed admin binding: %v", err)
	}

	if _, err := s.RemoveRoleMembers(adminRole.ID, []uint64{1}); err == nil || common.ErrMessage(err) != "user.update.error.protected" {
		t.Fatalf("expected protected error when removing built-in admin, got %v", err)
	}
}
