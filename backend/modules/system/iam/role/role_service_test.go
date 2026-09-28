package iam

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
	"github.com/duanxldragon/pantheon-base/backend/pkg/database"
	"github.com/duanxldragon/pantheon-base/backend/pkg/impexp"
	"github.com/duanxldragon/pantheon-base/backend/pkg/testmysql"

	"gorm.io/gorm"
)

func setupRoleTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db := testmysql.Open(t)

	// 迁移模型
	_ = db.AutoMigrate(&SystemRole{}, &SystemRolePermission{}, &SystemRoleMenu{}, &roleDataScopePolicy{}, &database.CasbinRule{})
	_ = db.Exec("CREATE TABLE IF NOT EXISTS system_menu (id INTEGER PRIMARY KEY, page_perm TEXT, perms TEXT, type TEXT)")
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

func TestNormalizeRoleDataScopeDefaultsToAll(t *testing.T) {
	for _, value := range []string{"", " ", "invalid"} {
		if got := normalizeRoleDataScope(value); got != common.DataScopeModeAll {
			t.Fatalf("expected %q to normalize to all, got %q", value, got)
		}
	}
	if got := normalizeRoleDataScope(common.DataScopeModeSelf); got != common.DataScopeModeSelf {
		t.Fatalf("expected self to remain self, got %q", got)
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

	createReq := &RoleCreateReq{
		RoleName: "Delete Me",
		RoleKey:  "delete_me",
		Status:   1,
	}
	assertRoleDeletionCleansAssociations(t, db, s, createReq)
	assertProtectedRoleDeletion(t, db, s)
	assertRoleWithUsersDeletion(t, db, s)
}

func assertRoleDeletionCleansAssociations(t *testing.T, db *gorm.DB, s *RoleService, createReq *RoleCreateReq) {
	t.Helper()
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

	// 预置 custom 数据权限策略，验证删除会清理而不是被同名新角色继承
	if err := db.Exec("INSERT INTO system_role_data_scope (role_key, mode) VALUES ('delete_me','custom') ON DUPLICATE KEY UPDATE mode='custom'").Error; err != nil {
		t.Fatalf("failed to seed data scope policy: %v", err)
	}

	if err := s.DeleteRole(roleResp.ID); err != nil {
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
	var scopeCount int64
	if err := db.Model(&roleDataScopePolicy{}).Where(condRoleKeyEquals, "delete_me").Count(&scopeCount).Error; err != nil {
		t.Fatalf("failed to count data scope policies: %v", err)
	}
	if scopeCount != 0 {
		t.Fatalf("expected data scope policies to be removed, got %d", scopeCount)
	}
	if _, err := s.CreateRole(createReq); err != nil {
		t.Fatalf("expected role key to be reusable after delete, got %v", err)
	}
	var recreatedScope roleDataScopePolicy
	if err := db.Where(condRoleKeyEquals, "delete_me").First(&recreatedScope).Error; err == nil {
		if recreatedScope.Mode == "custom" {
			t.Fatalf("re-created role inherited dead role's custom data scope")
		}
	}
}

func assertProtectedRoleDeletion(t *testing.T, db *gorm.DB, s *RoleService) {
	t.Helper()
	adminRole := SystemRole{RoleName: "Admin", RoleKey: "admin"}
	db.Create(&adminRole)
	err := s.DeleteRole(adminRole.ID)
	if err == nil || common.ErrMessage(err) != "role.delete.error.protected" {
		t.Errorf("expected protected error for admin role, got %v", err)
	}
}

func assertRoleWithUsersDeletion(t *testing.T, db *gorm.DB, s *RoleService) {
	t.Helper()
	roleWithUser := SystemRole{RoleName: "Has User", RoleKey: "has_user"}
	db.Create(&roleWithUser)
	_ = db.Exec("INSERT INTO system_user_role (user_id, role_id) VALUES (1, ?)", roleWithUser.ID)

	err := s.DeleteRole(roleWithUser.ID)
	if err == nil || common.ErrMessage(err) != "role.delete.error.has_users" {
		t.Errorf("expected has_users error, got %v", err)
	}
}

func TestRoleService_ExportRolesHonorsRowCap(t *testing.T) {
	db := setupRoleTestDB(t)
	s := NewRoleService(db)

	oldCap := maxRoleExportRows
	maxRoleExportRows = 2
	defer func() { maxRoleExportRows = oldCap }()

	for _, key := range []string{"cap_role_a", "cap_role_b", "cap_role_c"} {
		if err := db.Create(&SystemRole{RoleName: key, RoleKey: key, Status: 1}).Error; err != nil {
			t.Fatalf("seed role %s: %v", key, err)
		}
	}

	exported, err := s.ExportRoles(context.Background(), &RoleListQuery{RoleKey: "cap_role_"})
	if err != nil {
		t.Fatalf("export roles: %v", err)
	}
	if len(exported.Rows) != 2 {
		t.Fatalf("expected export capped at 2 rows, got %d", len(exported.Rows))
	}
}

func TestRoleService_DeleteRoleRollsBackOnFailure(t *testing.T) {
	db := setupRoleTestDB(t)
	s := NewRoleService(db)

	roleResp, err := s.CreateRole(&RoleCreateReq{
		RoleName: "Rollback Me",
		RoleKey:  "rollback_me",
		Status:   1,
	})
	if err != nil {
		t.Fatalf("failed to create role: %v", err)
	}

	// 删除 system_role_menu 表制造事务中途失败，验证角色未被半删除。
	if err := db.Exec("DROP TABLE system_role_menu").Error; err != nil {
		t.Fatalf("failed to drop table for failure injection: %v", err)
	}

	if err := s.DeleteRole(roleResp.ID); err == nil {
		t.Fatalf("expected delete to fail after dropping system_role_menu")
	}

	var role SystemRole
	if err := db.First(&role, roleResp.ID).Error; err != nil {
		t.Fatalf("expected role to survive failed delete, got %v", err)
	}
	if role.RoleKey != "rollback_me" {
		t.Fatalf("expected role key unchanged after rollback, got %s", role.RoleKey)
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

	exported, err := s.ExportRoles(context.Background(), &RoleListQuery{RoleKey: "editor"})
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

	assertRoleMemberPage(t, s, role.ID, 2, "")
	assertRoleCandidatePage(t, s, role.ID, "", 0, "")

	removedCount, err := s.RemoveRoleMembers(role.ID, []uint64{11})
	if err != nil {
		t.Fatalf("remove role member: %v", err)
	}
	if removedCount != 1 {
		t.Fatalf("expected 1 member removed, got %d", removedCount)
	}

	assertRoleMemberPage(t, s, role.ID, 1, "bob")
	assertRoleCandidatePage(t, s, role.ID, "alice", 1, "alice")
}

func assertRoleMemberPage(t *testing.T, s *RoleService, roleID uint64, wantTotal int64, wantUsername string) {
	t.Helper()
	members, err := s.ListRoleMembers(roleID, &RoleMemberQuery{Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("list role members: %v", err)
	}
	if members.Total != wantTotal || len(members.Items) != int(wantTotal) {
		t.Fatalf("unexpected role members: %+v", members)
	}
	if wantUsername != "" && members.Items[0].Username != wantUsername {
		t.Fatalf("expected role member %s, got %+v", wantUsername, members.Items)
	}
}

func assertRoleCandidatePage(t *testing.T, s *RoleService, roleID uint64, keyword string, wantTotal int64, wantUsername string) {
	t.Helper()
	candidates, err := s.ListAssignableUsers(roleID, &RoleMemberQuery{Keyword: keyword, Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("list role candidates: %v", err)
	}
	if candidates.Total != wantTotal || len(candidates.Items) != int(wantTotal) {
		t.Fatalf("unexpected role candidates: %+v", candidates)
	}
	if wantUsername != "" && candidates.Items[0].Username != wantUsername {
		t.Fatalf("expected role candidate %s, got %+v", wantUsername, candidates.Items)
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

// roleImportHeader 与 role_export.go 中 BuildRoleImportTemplate 的表头保持一致。
var roleImportHeader = []string{"roleName", "roleKey", "sort", "status", "menuIds", "permissionKeys"}

func hasImportErrorMessage(errors []impexp.ImportError, msg string) bool {
	for _, e := range errors {
		if e.Message == msg {
			return true
		}
	}
	return false
}

func TestRoleService_ImportRolesEmptyFile(t *testing.T) {
	db := setupRoleTestDB(t)
	s := NewRoleService(db)

	result, err := s.ImportRoles([][]string{})
	if err != nil {
		t.Fatalf("unexpected error on empty records: %v", err)
	}
	if result.Applied {
		t.Fatalf("expected applied=false for empty file")
	}
	if result.Failed != 1 {
		t.Fatalf("expected failed=1 for empty file, got %d", result.Failed)
	}
	if !hasImportErrorMessage(result.Errors, "import.file.empty") {
		t.Fatalf("expected import.file.empty error, got %+v", result.Errors)
	}
}

func TestRoleService_ImportRolesCreateAndUpdate(t *testing.T) {
	db := setupRoleTestDB(t)
	s := NewRoleService(db)

	createRecords := [][]string{
		roleImportHeader,
		{"Imported A", "imported_a", "10", "1", "1", ""},
		{"Imported B", "imported_b", "20", "2", "1", ""},
	}
	created, err := s.ImportRoles(createRecords)
	if err != nil {
		t.Fatalf("import create: %v", err)
	}
	if !created.Applied || created.Created != 2 || created.Updated != 0 {
		t.Fatalf("expected 2 created, got %+v", created)
	}

	updateRecords := [][]string{
		roleImportHeader,
		{"Imported A Edit", "imported_a", "11", "2", "1", ""},
	}
	updated, err := s.ImportRoles(updateRecords)
	if err != nil {
		t.Fatalf("import update: %v", err)
	}
	if !updated.Applied || updated.Updated != 1 || updated.Created != 0 {
		t.Fatalf("expected 1 updated, got %+v", updated)
	}

	var role SystemRole
	if err := db.Where("role_key = ?", "imported_a").First(&role).Error; err != nil {
		t.Fatalf("load updated role: %v", err)
	}
	if role.Status != 2 || role.Sort != 11 || role.RoleName != "Imported A Edit" {
		t.Fatalf("unexpected updated role: status=%d sort=%d name=%s", role.Status, role.Sort, role.RoleName)
	}

	var menuCount int64
	if err := db.Table("system_role_menu").Where("role_id = ?", role.ID).Count(&menuCount).Error; err != nil {
		t.Fatalf("count menus: %v", err)
	}
	if menuCount != 1 {
		t.Fatalf("expected 1 menu binding after update, got %d", menuCount)
	}
}

func TestRoleService_ImportRowsDuplicateDetection(t *testing.T) {
	db := setupRoleTestDB(t)
	s := NewRoleService(db)

	records := [][]string{
		roleImportHeader,
		{"Dup One", "dup_role", "1", "1", "", ""},
		{"Dup Two", "dup_role", "2", "1", "", ""},
	}
	result, err := s.ImportRoles(records)
	if err != nil {
		t.Fatalf("import dup: %v", err)
	}
	if result.Applied {
		t.Fatalf("expected applied=false for duplicate rows")
	}
	if result.Failed == 0 {
		t.Fatalf("expected at least one failure for duplicate rows")
	}
	if !hasImportErrorMessage(result.Errors, "import.duplicate.row.2") {
		t.Fatalf("expected duplicate-row error, got %+v", result.Errors)
	}
}

func TestRoleService_ImportRowsRejectInvalidReferences(t *testing.T) {
	testCases := []struct {
		name        string
		record      []string
		wantMessage string
	}{
		{
			name:        "missing role key",
			record:      []string{"No Key", "", "1", "1", "", ""},
			wantMessage: "role.roleKey.required",
		},
		{
			name:        "missing role name",
			record:      []string{"", "no_name", "1", "1", "", ""},
			wantMessage: "role.roleName.required",
		},
		{
			name:        "invalid menu id",
			record:      []string{"Bad Menu", "bad_menu", "1", "1", "99", ""},
			wantMessage: "role.menu.not_found",
		},
		{
			name:        "invalid permission",
			record:      []string{"Bad Perm", "bad_perm", "1", "1", "", "bogus:perm"},
			wantMessage: "role.permission.not_found",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			db := setupRoleTestDB(t)
			result, err := NewRoleService(db).ImportRoles([][]string{roleImportHeader, testCase.record})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result.Applied || result.Failed == 0 {
				t.Fatalf("expected import validation failure, got %+v", result)
			}
			if !hasImportErrorMessage(result.Errors, testCase.wantMessage) {
				t.Fatalf("expected %s, got %+v", testCase.wantMessage, result.Errors)
			}
		})
	}
}
