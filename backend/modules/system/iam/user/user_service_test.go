package iam

import (
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"pantheon-platform/backend/pkg/testmysql"
)

func setupUserTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db := testmysql.Open(t)
	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(1)

	// 迁移模型
	_ = db.AutoMigrate(&SystemUser{}, &SystemUserProfileExt{})
	_ = db.Exec("CREATE TABLE IF NOT EXISTS system_role (id BIGINT PRIMARY KEY, role_key VARCHAR(64), role_name VARCHAR(128), status INT, deleted_at DATETIME NULL)")
	_ = db.Exec("CREATE TABLE IF NOT EXISTS system_user_role (user_id BIGINT, role_id BIGINT)")
	_ = db.Exec("CREATE TABLE IF NOT EXISTS system_user_session (session_id VARCHAR(128), user_id BIGINT, revoked_at DATETIME NULL)")
	_ = db.Exec("CREATE TABLE IF NOT EXISTS system_dept (id BIGINT PRIMARY KEY, parent_id BIGINT, dept_name VARCHAR(128))")
	_ = db.Exec("CREATE TABLE IF NOT EXISTS system_post (id BIGINT PRIMARY KEY, post_code VARCHAR(64), post_name VARCHAR(128), dept_id BIGINT)")
	_ = db.Exec("CREATE TABLE IF NOT EXISTS system_log_login (id BIGINT PRIMARY KEY AUTO_INCREMENT, username VARCHAR(64), status INT, login_time DATETIME)")
	_ = db.Exec("CREATE TABLE IF NOT EXISTS system_setting (setting_key VARCHAR(128) PRIMARY KEY, setting_value TEXT)")

	// 插入基础数据
	_ = db.Exec("INSERT INTO system_role (id, role_key, role_name, status) VALUES (1, 'admin', '管理员', 1)")
	_ = db.Exec("INSERT INTO system_role (id, role_key, role_name, status) VALUES (2, 'test', '测试角色', 1)")

	return db
}

func TestUserService_CreateUser(t *testing.T) {
	db := setupUserTestDB(t)
	s := NewUserService(db)

	req := &UserCreateReq{
		Username: "admin_test",
		Password: "password123",
		Nickname: "Admin Test",
		Status:   1,
		RoleIDs:  []uint64{1},
	}

	// 1. 成功创建
	resp, err := s.CreateUser(req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resp.Username != "admin_test" {
		t.Errorf("expected username admin_test, got %s", resp.Username)
	}
	if resp.DeptID != 0 || resp.PostID != 0 {
		t.Fatalf("expected user without organization binding, got dept=%d post=%d", resp.DeptID, resp.PostID)
	}

	// 2. 用户名重复
	_, err = s.CreateUser(req)
	if err == nil || err.Error() != "user.create.error.username_exists" {
		t.Errorf("expected username exists error, got %v", err)
	}

	// 3. 角色无效
	req.Username = "invalid_role"
	req.RoleIDs = []uint64{99}
	_, err = s.CreateUser(req)
	if err == nil || err.Error() != "user.role.invalid" {
		t.Errorf("expected role invalid error, got %v", err)
	}
}

func TestUserService_CreateUserAllowsEmptyRoles(t *testing.T) {
	db := setupUserTestDB(t)
	s := NewUserService(db)

	resp, err := s.CreateUser(&UserCreateReq{
		Username: "role_optional_user",
		Password: "password123",
		Status:   1,
		RoleIDs:  []uint64{},
	})
	if err != nil {
		t.Fatalf("expected empty-role user creation to succeed, got %v", err)
	}
	if len(resp.RoleIDs) != 0 {
		t.Fatalf("expected no bound roles, got %+v", resp.RoleIDs)
	}
	detail, err := s.GetUserDetail(resp.ID)
	if err != nil {
		t.Fatalf("expected detail lookup to succeed, got %v", err)
	}
	if detail.RoleIDs == nil || detail.RoleKeys == nil || detail.RoleNames == nil {
		t.Fatalf("expected empty role collections, got %+v / %+v / %+v", detail.RoleIDs, detail.RoleKeys, detail.RoleNames)
	}
	if len(detail.RoleIDs) != 0 || len(detail.RoleKeys) != 0 || len(detail.RoleNames) != 0 {
		t.Fatalf("expected no role bindings in detail, got %+v / %+v / %+v", detail.RoleIDs, detail.RoleKeys, detail.RoleNames)
	}
}

func TestUserService_UpdateUser(t *testing.T) {
	db := setupUserTestDB(t)
	s := NewUserService(db)

	// 创建初始用户
	createReq := &UserCreateReq{
		Username: "update_test",
		Password: "password123",
		RoleIDs:  []uint64{2},
		Status:   1,
	}
	userResp, _ := s.CreateUser(createReq)

	// 1. 成功更新
	updateReq := &UserUpdateReq{
		Nickname: "Updated Nickname",
		RoleIDs:  []uint64{1, 2},
		Status:   1,
	}
	resp, err := s.UpdateUser(userResp.ID, updateReq)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resp.Nickname != "Updated Nickname" {
		t.Errorf("expected nickname Updated Nickname, got %s", resp.Nickname)
	}

	// 2. 更新不存在的用户
	_, err = s.UpdateUser(999, updateReq)
	if err == nil {
		t.Error("expected error for non-existent user, got nil")
	}
}

func TestUserService_UserProfileExtLifecycle(t *testing.T) {
	db := setupUserTestDB(t)
	s := NewUserService(db)

	userResp, err := s.CreateUser(&UserCreateReq{
		Username: "consumer_user",
		Password: "password123",
		Nickname: "Consumer",
		RoleIDs:  []uint64{2},
		Status:   1,
		ProfileExt: map[string]interface{}{
			"gender": "unknown",
			"vip":    true,
		},
	})
	if err != nil {
		t.Fatalf("create consumer user: %v", err)
	}

	detail, err := s.GetUserDetail(userResp.ID)
	if err != nil {
		t.Fatalf("get user detail: %v", err)
	}
	if detail.ProfileExt["gender"] != "unknown" || detail.ProfileExt["vip"] != true {
		t.Fatalf("unexpected created profile ext: %+v", detail.ProfileExt)
	}

	updated, err := s.UpdateProfile(userResp.ID, &UserProfileUpdateReq{
		Nickname: "Consumer Updated",
		ProfileExt: map[string]interface{}{
			"gender": "female",
			"source": "app",
		},
	})
	if err != nil {
		t.Fatalf("update profile ext: %v", err)
	}
	if updated.ProfileExt["gender"] != "female" || updated.ProfileExt["source"] != "app" {
		t.Fatalf("unexpected updated profile ext: %+v", updated.ProfileExt)
	}
	if updated.DeptID != 0 || updated.PostID != 0 {
		t.Fatalf("expected consumer profile without org binding, got dept=%d post=%d", updated.DeptID, updated.PostID)
	}
}

func TestUserService_BatchUpdateUserStatus(t *testing.T) {
	db := setupUserTestDB(t)
	s := NewUserService(db)

	if err := db.Create(&SystemUser{ID: 1, Username: "admin", Status: 1}).Error; err != nil {
		t.Fatalf("seed admin user: %v", err)
	}
	userResp, err := s.CreateUser(&UserCreateReq{
		Username: "batch_user",
		Password: "password123",
		RoleIDs:  []uint64{2},
		Status:   1,
	})
	if err != nil {
		t.Fatalf("create batch user: %v", err)
	}

	updated, err := s.BatchUpdateUserStatus([]uint64{userResp.ID, userResp.ID}, 2)
	if err != nil {
		t.Fatalf("batch disable user: %v", err)
	}
	if updated != 1 {
		t.Fatalf("expected 1 updated user, got %d", updated)
	}
	var disabled SystemUser
	if err := db.First(&disabled, userResp.ID).Error; err != nil {
		t.Fatalf("load disabled user: %v", err)
	}
	if disabled.Status != 2 {
		t.Fatalf("expected user status 2, got %d", disabled.Status)
	}

	if _, err := s.BatchUpdateUserStatus([]uint64{1}, 2); err == nil || err.Error() != "user.update.error.protected" {
		t.Fatalf("expected protected admin error, got %v", err)
	}
}

func TestUserService_MigrateCreatesUserRoleTableAndAdminBinding(t *testing.T) {
	t.Setenv("PANTHEON_ENV", "development")
	db := testmysql.Open(t)
	if err := db.Exec("CREATE TABLE IF NOT EXISTS system_role (id BIGINT PRIMARY KEY, role_key VARCHAR(64), status INT, deleted_at DATETIME NULL)").Error; err != nil {
		t.Fatalf("create role table: %v", err)
	}
	if err := db.Exec("INSERT INTO system_role (id, role_key, status) VALUES (1, 'admin', 1)").Error; err != nil {
		t.Fatalf("seed admin role: %v", err)
	}

	s := NewUserService(db)
	if err := s.Migrate(); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if !db.Migrator().HasTable("system_user_role") {
		t.Fatal("expected system_user_role table to exist")
	}

	var bindingCount int64
	if err := db.Table("system_user_role").Where("user_id = ? AND role_id = ?", 1, 1).Count(&bindingCount).Error; err != nil {
		t.Fatalf("count admin binding: %v", err)
	}
	if bindingCount != 1 {
		t.Fatalf("expected admin user-role binding, got %d", bindingCount)
	}
}

func TestUserService_MigrateRejectsMissingInitialAdminPasswordInProduction(t *testing.T) {
	t.Setenv("PANTHEON_ENV", "production")
	t.Setenv("PANTHEON_INITIAL_ADMIN_PASSWORD", "")
	db := testmysql.Open(t)

	s := NewUserService(db)
	if err := s.Migrate(); err == nil || err.Error() != "admin.initial_password_required" {
		t.Fatalf("expected production initial admin password guard, got %v", err)
	}
}

func TestUserService_MigrateUsesConfiguredInitialAdminPasswordInProduction(t *testing.T) {
	t.Setenv("PANTHEON_ENV", "production")
	t.Setenv("PANTHEON_INITIAL_ADMIN_PASSWORD", "initial-admin-password-2026")
	db := testmysql.Open(t)

	s := NewUserService(db)
	if err := s.Migrate(); err != nil {
		t.Fatalf("migrate with configured admin password: %v", err)
	}

	var admin SystemUser
	if err := db.First(&admin, 1).Error; err != nil {
		t.Fatalf("load admin user: %v", err)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(admin.Password), []byte("initial-admin-password-2026")); err != nil {
		t.Fatalf("expected configured initial admin password hash to match: %v", err)
	}
}

func TestResolveInitialAdminPasswordRequiresProductionOverride(t *testing.T) {
	t.Setenv("PANTHEON_ENV", "production")
	t.Setenv("PANTHEON_INITIAL_ADMIN_PASSWORD", "")

	_, err := resolveInitialAdminPassword()
	if err == nil || err.Error() != "admin.initial_password_required" {
		t.Fatalf("expected production guard error, got %v", err)
	}
}

func TestResolveInitialAdminPasswordUsesProductionOverride(t *testing.T) {
	t.Setenv("PANTHEON_ENV", "production")
	t.Setenv("PANTHEON_INITIAL_ADMIN_PASSWORD", "initial-admin-password-2026")

	password, err := resolveInitialAdminPassword()
	if err != nil {
		t.Fatalf("resolve production password: %v", err)
	}
	if password != "initial-admin-password-2026" {
		t.Fatalf("expected configured password, got %q", password)
	}
}

func TestResolveInitialAdminPasswordRejectsWeakProductionOverride(t *testing.T) {
	t.Setenv("PANTHEON_ENV", "production")
	t.Setenv("PANTHEON_INITIAL_ADMIN_PASSWORD", "123456")

	_, err := resolveInitialAdminPassword()
	if err == nil || err.Error() != "admin.initial_password_too_short" {
		t.Fatalf("expected weak production password error, got %v", err)
	}
}

func TestResolveInitialAdminPasswordKeepsDevFallback(t *testing.T) {
	t.Setenv("PANTHEON_ENV", "development")
	t.Setenv("PANTHEON_INITIAL_ADMIN_PASSWORD", "")

	password, err := resolveInitialAdminPassword()
	if err != nil {
		t.Fatalf("resolve development password: %v", err)
	}
	if password != "123456" {
		t.Fatalf("expected development fallback password, got %q", password)
	}
}

func TestUserService_ListUsersByDeptAndPost(t *testing.T) {
	db := setupUserTestDB(t)
	s := NewUserService(db)

	_ = db.Exec("INSERT INTO system_dept (id, dept_name) VALUES (10, '研发部'), (20, '财务部')")
	_ = db.Exec("INSERT INTO system_post (id, post_name, dept_id) VALUES (100, '研发工程师', 10), (200, '会计', 20)")

	if _, err := s.CreateUser(&UserCreateReq{
		Username: "rd_user",
		Password: "password123",
		DeptID:   10,
		PostID:   100,
		Status:   1,
		RoleIDs:  []uint64{2},
	}); err != nil {
		t.Fatalf("create rd user: %v", err)
	}
	if _, err := s.CreateUser(&UserCreateReq{
		Username: "finance_user",
		Password: "password123",
		DeptID:   20,
		PostID:   200,
		Status:   1,
		RoleIDs:  []uint64{2},
	}); err != nil {
		t.Fatalf("create finance user: %v", err)
	}

	deptList, err := s.ListUsers(&UserListQuery{DeptID: 10, Page: 1, PageSize: 20}, nil)
	if err != nil {
		t.Fatalf("list by dept: %v", err)
	}
	if deptList.Total != 1 || len(deptList.Items) != 1 || deptList.Items[0].Username != "rd_user" {
		t.Fatalf("expected only rd_user for dept 10, got %+v", deptList.Items)
	}

	postList, err := s.ListUsers(&UserListQuery{PostID: 200, Page: 1, PageSize: 20}, nil)
	if err != nil {
		t.Fatalf("list by post: %v", err)
	}
	if postList.Total != 1 || len(postList.Items) != 1 || postList.Items[0].Username != "finance_user" {
		t.Fatalf("expected only finance_user for post 200, got %+v", postList.Items)
	}
}

func TestUserService_DeleteUser(t *testing.T) {
	db := setupUserTestDB(t)
	s := NewUserService(db)

	// 先创建一个占位用户，占用 ID 1
	db.Create(&SystemUser{Username: "admin_placeholder"})

	createReq := &UserCreateReq{
		Username: "delete_test",
		Password: "password123",
		RoleIDs:  []uint64{2},
		Status:   1,
	}
	userResp, err := s.CreateUser(createReq)
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	// 1. 成功删除
	err = s.DeleteUser(userResp.ID)
	if err != nil {
		t.Errorf("expected no error for ID %d, got %v", userResp.ID, err)
	}

	var deletedUser SystemUser
	if err := db.Unscoped().First(&deletedUser, userResp.ID).Error; err != nil {
		t.Fatalf("failed to load deleted user: %v", err)
	}
	if deletedUser.DeletedAt.Time.IsZero() {
		t.Fatal("expected deleted_at to be populated")
	}
	if !strings.HasPrefix(deletedUser.Username, deletedUsernamePrefix) {
		t.Fatalf("expected deleted username to be archived, got %s", deletedUser.Username)
	}

	recreated, err := s.CreateUser(createReq)
	if err != nil {
		t.Fatalf("expected username to be reusable after delete, got %v", err)
	}
	if recreated.Username != createReq.Username {
		t.Fatalf("expected recreated username %s, got %s", createReq.Username, recreated.Username)
	}

	// 2. 删除超级管理员 (ID=1)
	err = s.DeleteUser(1)
	if err == nil || err.Error() != "user.delete.error.protected" {
		t.Errorf("expected protected error for admin, got %v", err)
	}
}

func TestUserService_MigrateReleasesLegacyDeletedUsername(t *testing.T) {
	db := setupUserTestDB(t)

	if err := db.Create(&SystemUser{
		Username: "admin_seed",
		Password: "hashed",
		Status:   1,
	}).Error; err != nil {
		t.Fatalf("failed to seed admin placeholder: %v", err)
	}

	legacyUser := SystemUser{
		Username: "legacy_deleted",
		Password: "hashed",
		Status:   1,
	}
	if err := db.Create(&legacyUser).Error; err != nil {
		t.Fatalf("failed to seed legacy user: %v", err)
	}
	if err := db.Model(&legacyUser).Update("deleted_at", time.Now()).Error; err != nil {
		t.Fatalf("failed to soft delete legacy user: %v", err)
	}

	s := NewUserService(db)
	if err := s.Migrate(); err != nil {
		t.Fatalf("expected migrate to succeed, got %v", err)
	}

	var repaired SystemUser
	if err := db.Unscoped().First(&repaired, legacyUser.ID).Error; err != nil {
		t.Fatalf("failed to reload repaired user: %v", err)
	}
	if !strings.HasPrefix(repaired.Username, deletedUsernamePrefix) {
		t.Fatalf("expected repaired username to be archived, got %s", repaired.Username)
	}

	created, err := s.CreateUser(&UserCreateReq{
		Username: "legacy_deleted",
		Password: "password123",
		RoleIDs:  []uint64{2},
		Status:   1,
	})
	if err != nil {
		t.Fatalf("expected legacy username to be reusable after migrate, got %v", err)
	}
	if created.Username != "legacy_deleted" {
		t.Fatalf("expected recreated username legacy_deleted, got %s", created.Username)
	}
}

func TestUserService_MigrateNormalizesLegacyPreferenceJSON(t *testing.T) {
	db := setupUserTestDB(t)

	if err := db.Create(&SystemUser{
		Username:       "legacy_preference_user",
		Password:       "hashed",
		Status:         1,
		PreferenceJSON: `{"theme":"emerald","layout":"horizontal","density":"compact","lang":"en-US","extra":"ignored"}`,
	}).Error; err != nil {
		t.Fatalf("seed legacy preference user: %v", err)
	}

	s := NewUserService(db)
	if err := s.Migrate(); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	var repaired SystemUser
	if err := db.Where("username = ?", "legacy_preference_user").First(&repaired).Error; err != nil {
		t.Fatalf("reload repaired user: %v", err)
	}
	if repaired.PreferenceJSON != `{"theme":"emerald","language":"en-US","layoutMode":"horizontal","densityMode":"compact"}` {
		t.Fatalf("unexpected normalized preference json: %s", repaired.PreferenceJSON)
	}
}

func TestUserService_ResetPassword(t *testing.T) {
	db := setupUserTestDB(t)
	s := NewUserService(db)

	createReq := &UserCreateReq{
		Username: "reset_test",
		Password: "password123",
		RoleIDs:  []uint64{2},
		Status:   1,
	}
	userResp, err := s.CreateUser(createReq)
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	_ = db.Exec("INSERT INTO system_user_session (session_id, user_id, revoked_at) VALUES ('s-1', ?, NULL)", userResp.ID)
	_ = db.Exec("INSERT INTO system_user_session (session_id, user_id, revoked_at) VALUES ('s-2', ?, NULL)", userResp.ID)

	revokedCount, err := s.ResetPassword(userResp.ID, "newpassword456")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if revokedCount != 2 {
		t.Fatalf("expected 2 revoked sessions, got %d", revokedCount)
	}

	var updatedUser SystemUser
	if err := db.First(&updatedUser, userResp.ID).Error; err != nil {
		t.Fatalf("failed to load updated user: %v", err)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(updatedUser.Password), []byte("newpassword456")); err != nil {
		t.Fatalf("expected password to be updated: %v", err)
	}

	var revokedRows int64
	if err := db.Table("system_user_session").Where("user_id = ? AND revoked_at IS NOT NULL", userResp.ID).Count(&revokedRows).Error; err != nil {
		t.Fatalf("failed to count revoked sessions: %v", err)
	}
	if revokedRows != 2 {
		t.Fatalf("expected 2 revoked session rows, got %d", revokedRows)
	}
}

func TestUserService_CreateAndResetPasswordUseConfiguredMinLength(t *testing.T) {
	db := setupUserTestDB(t)
	s := NewUserService(db)
	_ = db.Exec("INSERT INTO system_setting (setting_key, setting_value) VALUES ('security.password_min_length', '8')")

	_, err := s.CreateUser(&UserCreateReq{
		Username: "short_policy_user",
		Password: "1234567",
		RoleIDs:  []uint64{2},
		Status:   1,
	})
	if err == nil || err.Error() != "user.update.error.password_too_short" {
		t.Fatalf("expected create user to respect configured min length, got %v", err)
	}

	userResp, err := s.CreateUser(&UserCreateReq{
		Username: "long_policy_user",
		Password: "12345678",
		RoleIDs:  []uint64{2},
		Status:   1,
	})
	if err != nil {
		t.Fatalf("create long policy user: %v", err)
	}

	if _, err := s.ResetPassword(userResp.ID, "1234567"); err == nil || err.Error() != "user.update.error.password_too_short" {
		t.Fatalf("expected reset password to respect configured min length, got %v", err)
	}
}

func TestUserService_GetUserDetail(t *testing.T) {
	db := setupUserTestDB(t)
	s := NewUserService(db)

	_ = db.Exec("INSERT INTO system_dept (id, dept_name) VALUES (10, '研发部')")
	_ = db.Exec("INSERT INTO system_post (id, post_name, dept_id) VALUES (20, '架构师', 10)")

	userResp, err := s.CreateUser(&UserCreateReq{
		Username: "detail_test",
		Password: "password123",
		Nickname: "Detail User",
		Avatar:   "https://example.com/avatar.png",
		Email:    "detail@example.com",
		Phone:    "13800138000",
		DeptID:   10,
		PostID:   20,
		Status:   1,
		RoleIDs:  []uint64{1, 2},
	})
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}
	loginTime := time.Date(2026, 5, 16, 17, 4, 9, 0, time.UTC)
	if err := db.Exec(
		"INSERT INTO system_log_login (username, status, login_time) VALUES (?, ?, ?)",
		"detail_test",
		1,
		loginTime,
	).Error; err != nil {
		t.Fatalf("seed login log: %v", err)
	}

	detail, err := s.GetUserDetail(userResp.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if detail.Username != "detail_test" {
		t.Fatalf("expected username detail_test, got %s", detail.Username)
	}
	if detail.Avatar != "https://example.com/avatar.png" {
		t.Fatalf("expected avatar to be populated, got %s", detail.Avatar)
	}
	if detail.DeptName != "研发部" {
		t.Fatalf("expected dept name 研发部, got %s", detail.DeptName)
	}
	if detail.PostName != "架构师" {
		t.Fatalf("expected post name 架构师, got %s", detail.PostName)
	}
	if len(detail.RoleIDs) != 2 || len(detail.RoleKeys) != 2 || len(detail.RoleNames) != 2 {
		t.Fatalf(
			"expected role ids, keys, and names to be loaded, got %v / %v / %v",
			detail.RoleIDs,
			detail.RoleKeys,
			detail.RoleNames,
		)
	}
	if detail.UpdatedAt == "" {
		t.Fatal("expected updatedAt to be populated")
	}
	if detail.LastLoginAt == nil || *detail.LastLoginAt != loginTime.Format(time.RFC3339) {
		t.Fatalf("expected lastLoginAt to be populated, got %v", detail.LastLoginAt)
	}

	if _, err := s.GetUserDetail(999); err == nil {
		t.Fatal("expected error for missing user, got nil")
	}
}

func TestUserService_ImportTemplateAndExport(t *testing.T) {
	db := setupUserTestDB(t)
	s := NewUserService(db)

	template := s.BuildUserImportTemplate()
	if len(template.Rows) == 0 || !strings.HasPrefix(template.Rows[0][0], "#") {
		t.Fatalf("expected template to include ignored instruction rows, got %+v", template.Rows)
	}
	templateResult, err := s.ImportUsers(append([][]string{template.Headers}, template.Rows...))
	if err != nil {
		t.Fatalf("import template comments: %v", err)
	}
	if !templateResult.Applied || templateResult.Created != 0 || templateResult.Failed != 0 {
		t.Fatalf("expected template comments to be ignored, got %+v", templateResult)
	}

	result, err := s.ImportUsers([][]string{
		template.Headers,
		{"sample_user", "ChangeMe123", "示例用户", "sample@example.com", "13800138000", "", "", "1", "test"},
	})
	if err != nil {
		t.Fatalf("import user: %v", err)
	}
	if !result.Applied || result.Created != 1 || result.Failed != 0 {
		t.Fatalf("unexpected import result: %+v", result)
	}

	exported, err := s.ExportUsers(&UserListQuery{Username: "sample_user"})
	if err != nil {
		t.Fatalf("export user: %v", err)
	}
	if len(exported.Rows) != 1 || exported.Rows[0][0] != "sample_user" || exported.Rows[0][2] != "示例用户" {
		t.Fatalf("unexpected export rows: %+v", exported.Rows)
	}
}
