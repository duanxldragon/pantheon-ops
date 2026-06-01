package auth

import (
	"testing"
	"time"

	"pantheon-ops/backend/internal/middleware"
	auditmod "pantheon-ops/backend/modules/system/audit"
	settingmod "pantheon-ops/backend/modules/system/config/setting"
	user "pantheon-ops/backend/modules/system/iam/user"
	"pantheon-ops/backend/pkg/common"
	"pantheon-ops/backend/pkg/contracts"
	"pantheon-ops/backend/pkg/testmysql"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db := testmysql.Open(t)

	// 迁移模型
	_ = db.AutoMigrate(&user.SystemUser{}, &SystemUserSession{}, &SystemLogLogin{}, &SystemLoginThrottle{}, &SystemAuthFactor{}, &SystemAuthMFAChallenge{}, &SystemAuthSecurityEvent{}, &SystemUserPasswordHistory{})
	_ = db.Exec("CREATE TABLE IF NOT EXISTS system_setting (setting_key TEXT PRIMARY KEY, setting_value TEXT)")
	_ = db.Exec("CREATE TABLE IF NOT EXISTS system_role (id INTEGER PRIMARY KEY AUTOINCREMENT, role_key TEXT, status INTEGER)")
	_ = db.Exec("CREATE TABLE IF NOT EXISTS system_user_role (user_id INTEGER, role_id INTEGER)")
	_ = db.Exec("CREATE TABLE IF NOT EXISTS system_role_permission (id INTEGER PRIMARY KEY AUTOINCREMENT, role_id INTEGER, permission_key TEXT)")
	return db
}

func boolPtr(value bool) *bool {
	return &value
}

func TestAuthService_MFAChallengeSetupAndVerify(t *testing.T) {
	db := setupTestDB(t)
	s := NewAuthService(db)

	hash, _ := bcrypt.GenerateFromPassword([]byte("123456"), bcrypt.DefaultCost)
	testUser := user.SystemUser{
		Username: "mfa_user",
		Password: string(hash),
		Status:   1,
	}
	db.Create(&testUser)
	_ = db.Exec("INSERT INTO system_setting (setting_key, setting_value) VALUES ('login.mfa_enabled', 'true')")
	_ = s.ReloadSettings()

	authenticated, err := s.Authenticate(&LoginReq{Username: "mfa_user", Password: "123456"})
	if err != nil {
		t.Fatalf("authenticate before mfa: %v", err)
	}

	challenge, err := s.CreateMFAChallenge(authenticated)
	if err != nil {
		t.Fatalf("create mfa challenge: %v", err)
	}
	if !challenge.MFARequired || !challenge.SetupRequired || challenge.TOTPSecret == "" || challenge.TOTPProvisionURI == "" {
		t.Fatalf("expected setup challenge with provisioning data, got %+v", challenge)
	}

	code := generateTOTPCode(challenge.TOTPSecret, time.Now().Unix()/totpPeriod)
	resp, err := s.VerifyMFAChallenge(&MFAVerifyReq{ChallengeID: challenge.ChallengeID, Code: code}, "127.0.0.1", "test-agent")
	if err != nil {
		t.Fatalf("verify mfa challenge: %v", err)
	}
	if resp.AccessToken == "" || resp.RefreshToken == "" || resp.User == nil || resp.User.Username != "mfa_user" {
		t.Fatalf("expected token response after mfa, got %+v", resp)
	}

	var factorCount int64
	if err := db.Model(&SystemAuthFactor{}).Where("user_id = ? AND enabled = ?", testUser.ID, 1).Count(&factorCount).Error; err != nil {
		t.Fatalf("count factor: %v", err)
	}
	if factorCount != 1 {
		t.Fatalf("expected one enabled factor, got %d", factorCount)
	}

	_, err = s.VerifyMFAChallenge(&MFAVerifyReq{ChallengeID: challenge.ChallengeID, Code: code}, "127.0.0.1", "test-agent")
	if err == nil || err.Error() != "auth.mfa.challenge_expired" {
		t.Fatalf("expected consumed challenge to fail, got %v", err)
	}
}

func TestAuthService_MFARejectsInvalidCode(t *testing.T) {
	db := setupTestDB(t)
	s := NewAuthService(db)

	hash, _ := bcrypt.GenerateFromPassword([]byte("123456"), bcrypt.DefaultCost)
	testUser := user.SystemUser{
		Username: "mfa_invalid_user",
		Password: string(hash),
		Status:   1,
	}
	db.Create(&testUser)
	_ = db.Exec("INSERT INTO system_setting (setting_key, setting_value) VALUES ('login.mfa_enabled', 'true')")
	_ = s.ReloadSettings()

	challenge, err := s.CreateMFAChallenge(&testUser)
	if err != nil {
		t.Fatalf("create mfa challenge: %v", err)
	}
	_, err = s.VerifyMFAChallenge(&MFAVerifyReq{ChallengeID: challenge.ChallengeID, Code: "000000"}, "127.0.0.1", "test-agent")
	if err == nil || err.Error() != "auth.mfa.code_invalid" {
		t.Fatalf("expected invalid code error, got %v", err)
	}
}

func TestAuthService_ReloadsMFASettingAfterSettingUpdate(t *testing.T) {
	db := testmysql.Open(t)
	_ = db.AutoMigrate(&user.SystemUser{}, &SystemUserSession{}, &SystemLogLogin{}, &SystemLoginThrottle{}, &SystemAuthFactor{}, &SystemAuthMFAChallenge{})
	settingSvc := settingmod.NewSettingService(db)
	if err := settingSvc.Migrate(); err != nil {
		t.Fatalf("migrate setting: %v", err)
	}

	authSvc := NewAuthService(db)
	if authSvc.getAuthRuntimePolicy().MFAEnabled {
		t.Fatalf("expected MFA to start disabled")
	}
	unregister := contracts.RegisterRuntimeSettingReloader("test/system-auth-mfa", authSvc.ReloadSettings)
	defer unregister()

	if _, err := settingSvc.UpdateGroup("login", &settingmod.SettingGroupUpdateReq{Items: []settingmod.SettingUpdateItemReq{
		{SettingKey: "login.mfa_enabled", SettingValue: "true"},
	}}); err != nil {
		t.Fatalf("update MFA setting: %v", err)
	}

	if !authSvc.getAuthRuntimePolicy().MFAEnabled {
		t.Fatalf("expected MFA cache to reload after setting update")
	}
}

func TestAuthService_Authenticate(t *testing.T) {
	db := setupTestDB(t)
	s := NewAuthService(db)

	password := "123456"
	hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)

	// 创建测试用户
	testUser := user.SystemUser{
		Username: "testuser",
		Password: string(hash),
		Status:   1,
	}
	db.Create(&testUser)

	// 1. 成功登录
	u, err := s.Authenticate(&LoginReq{
		Username: "testuser",
		Password: password,
	})
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if u.Username != "testuser" {
		t.Errorf("expected username testuser, got %s", u.Username)
	}

	// 2. 密码错误
	_, err = s.Authenticate(&LoginReq{
		Username: "testuser",
		Password: "wrongpassword",
	})
	if err == nil || err.Error() != "user.login.error.password_wrong" {
		t.Errorf("expected password wrong error, got %v", err)
	}

	// 3. 用户不存在
	_, err = s.Authenticate(&LoginReq{
		Username: "nonexistent",
		Password: password,
	})
	if err == nil || err.Error() != "user.login.error.not_found" {
		t.Errorf("expected user not found error, got %v", err)
	}

	// 4. 用户被禁用
	db.Model(&testUser).Update("status", 2)
	_, err = s.Authenticate(&LoginReq{
		Username: "testuser",
		Password: password,
	})
	if err == nil || err.Error() != "user.login.error.disabled" {
		t.Errorf("expected user disabled error, got %v", err)
	}
}

func TestAuthService_AuthenticateTrimsUsername(t *testing.T) {
	db := setupTestDB(t)
	s := NewAuthService(db)

	hash, _ := bcrypt.GenerateFromPassword([]byte("123456"), bcrypt.DefaultCost)
	testUser := user.SystemUser{
		Username: "trim_user",
		Password: string(hash),
		Status:   1,
	}
	db.Create(&testUser)

	u, err := s.Authenticate(&LoginReq{
		Username: "  trim_user  ",
		Password: "123456",
	})
	if err != nil {
		t.Fatalf("expected trimmed username login to succeed, got %v", err)
	}
	if u.Username != "trim_user" {
		t.Fatalf("expected trim_user, got %s", u.Username)
	}
}

func TestAuthService_VerifyPasswordForOperationBindsSession(t *testing.T) {
	db := setupTestDB(t)
	s := NewAuthService(db)

	hash, _ := bcrypt.GenerateFromPassword([]byte("123456"), bcrypt.DefaultCost)
	testUser := user.SystemUser{
		Username: "verify_user",
		Password: string(hash),
		Status:   1,
	}
	db.Create(&testUser)

	token, err := s.VerifyPasswordForOperation(testUser.ID, "session-verify-1", "123456")
	if err != nil {
		t.Fatalf("verify password for operation: %v", err)
	}
	claims, err := common.ParseOperationToken(token)
	if err != nil {
		t.Fatalf("parse operation token: %v", err)
	}
	if claims.SessionID != "session-verify-1" {
		t.Fatalf("expected bound session id, got %s", claims.SessionID)
	}
	if claims.OperationScope != "secure_action" {
		t.Fatalf("expected secure_action scope, got %s", claims.OperationScope)
	}
}

func TestAuthService_UpdatePassword(t *testing.T) {
	db := setupTestDB(t)
	s := NewAuthService(db)

	oldPassword := "oldpassword"
	newPassword := "newpassword"
	hash, _ := bcrypt.GenerateFromPassword([]byte(oldPassword), bcrypt.DefaultCost)

	testUser := user.SystemUser{
		Username: "testpwd",
		Password: string(hash),
		Status:   1,
	}
	db.Create(&testUser)

	// 1. 成功修改密码
	err := s.UpdatePassword(testUser.ID, "session123", &PasswordUpdateReq{
		OldPassword: oldPassword,
		NewPassword: newPassword,
	})
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	// 验证新密码
	var updatedUser user.SystemUser
	db.First(&updatedUser, testUser.ID)
	err = bcrypt.CompareHashAndPassword([]byte(updatedUser.Password), []byte(newPassword))
	if err != nil {
		t.Errorf("new password verification failed: %v", err)
	}

	// 2. 旧密码错误
	err = s.UpdatePassword(testUser.ID, "session123", &PasswordUpdateReq{
		OldPassword: "wrongpassword",
		NewPassword: "somepassword",
	})
	if err == nil || err.Error() != "user.password.error.old_password_invalid" {
		t.Errorf("expected old password invalid error, got %v", err)
	}

	// 3. 新旧密码相同
	err = s.UpdatePassword(testUser.ID, "session123", &PasswordUpdateReq{
		OldPassword: newPassword,
		NewPassword: newPassword,
	})
	if err == nil || err.Error() != "user.password.error.same_as_old" {
		t.Errorf("expected same as old error, got %v", err)
	}

	// 4. 密码太短
	err = s.UpdatePassword(testUser.ID, "session123", &PasswordUpdateReq{
		OldPassword: newPassword,
		NewPassword: "123",
	})
	if err == nil || err.Error() != "user.update.error.password_too_short" {
		t.Errorf("expected password too short error, got %v", err)
	}
}

func TestAuthService_AuthenticateLocksUserByConfiguredPolicy(t *testing.T) {
	db := setupTestDB(t)
	s := NewAuthService(db)

	hash, _ := bcrypt.GenerateFromPassword([]byte("123456"), bcrypt.DefaultCost)
	testUser := user.SystemUser{
		Username: "locked_user",
		Password: string(hash),
		Status:   1,
	}
	db.Create(&testUser)
	_ = db.Exec("INSERT INTO system_setting (setting_key, setting_value) VALUES ('login.max_failed_attempts', '2'), ('login.lock_minutes', '10')")
	_ = s.ReloadSettings()

	_, err := s.Authenticate(&LoginReq{Username: "locked_user", Password: "wrong"})
	if err == nil || err.Error() != "user.login.error.password_wrong" {
		t.Fatalf("expected password wrong on first failure, got %v", err)
	}
	_, err = s.Authenticate(&LoginReq{Username: "locked_user", Password: "wrong"})
	if err == nil || err.Error() != "user.login.error.locked" {
		t.Fatalf("expected locked error on second failure, got %v", err)
	}

	var updated user.SystemUser
	if err := db.First(&updated, testUser.ID).Error; err != nil {
		t.Fatalf("reload locked user: %v", err)
	}
	if updated.LoginLockedUntil == nil || !updated.LoginLockedUntil.After(time.Now()) {
		t.Fatalf("expected login_locked_until to be set, got %+v", updated.LoginLockedUntil)
	}
}

func TestAuthService_LoginWithSourceBlocksSourceAfterConfiguredFailures(t *testing.T) {
	db := setupTestDB(t)
	s := NewAuthService(db)

	hash, _ := bcrypt.GenerateFromPassword([]byte("123456"), bcrypt.DefaultCost)
	testUser := user.SystemUser{
		Username: "source_locked_user",
		Password: string(hash),
		Status:   1,
	}
	db.Create(&testUser)
	_ = db.Exec("INSERT INTO system_setting (setting_key, setting_value) VALUES ('login.source_max_failed_attempts', '2'), ('login.source_window_minutes', '15'), ('login.source_lock_minutes', '10')")
	_ = s.ReloadSettings()

	_, err := s.LoginWithSource(&LoginReq{Username: "source_locked_user", Password: "wrong"}, "ip:10.0.0.1")
	if err == nil || err.Error() != "user.login.error.password_wrong" {
		t.Fatalf("expected password wrong on first failure, got %v", err)
	}
	_, err = s.LoginWithSource(&LoginReq{Username: "source_locked_user", Password: "wrong"}, "ip:10.0.0.1")
	if err == nil || err.Error() != "auth.login.error.source_blocked" {
		t.Fatalf("expected source blocked error on second failure, got %v", err)
	}
	_, err = s.LoginWithSource(&LoginReq{Username: "source_locked_user", Password: "123456"}, "ip:10.0.0.1")
	if err == nil || err.Error() != "auth.login.error.source_blocked" {
		t.Fatalf("expected source to remain blocked, got %v", err)
	}
}

func TestAuthService_LoginWithSourceRecordsSecurityEventWhenSourceBlocked(t *testing.T) {
	db := setupTestDB(t)
	s := NewAuthService(db)

	hash, _ := bcrypt.GenerateFromPassword([]byte("123456"), bcrypt.DefaultCost)
	testUser := user.SystemUser{
		Username: "risk_user",
		Password: string(hash),
		Status:   1,
	}
	db.Create(&testUser)
	_ = db.Exec("INSERT INTO system_setting (setting_key, setting_value) VALUES ('login.source_max_failed_attempts', '1'), ('login.source_window_minutes', '15'), ('login.source_lock_minutes', '10')")
	_ = s.ReloadSettings()

	_, err := s.LoginWithSource(&LoginReq{Username: "risk_user", Password: "wrong"}, "ip:10.0.0.9")
	if err == nil || err.Error() != "auth.login.error.source_blocked" {
		t.Fatalf("expected source blocked error, got %v", err)
	}

	events, err := s.ListSecurityEvents(&SecurityEventQuery{Username: "risk_user", EventType: "source_blocked", Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("list security events: %v", err)
	}
	if events.Total != 1 || len(events.Items) != 1 {
		t.Fatalf("expected one security event, got %+v", events)
	}
	if events.Items[0].Severity != "high" || events.Items[0].SourceKey != "ip:10.0.0.9" || events.Items[0].MessageKey != "auth.security.event.source_blocked" {
		t.Fatalf("unexpected security event: %+v", events.Items[0])
	}
}

func TestAuthService_LoginWithSourceRecordsSecurityEventWhenPasswordWrong(t *testing.T) {
	db := setupTestDB(t)
	s := NewAuthService(db)

	hash, _ := bcrypt.GenerateFromPassword([]byte("123456"), bcrypt.DefaultCost)
	testUser := user.SystemUser{
		Username: "wrong_password_user",
		Password: string(hash),
		Status:   1,
	}
	db.Create(&testUser)
	_ = s.ReloadSettings()

	_, err := s.LoginWithSource(&LoginReq{Username: "wrong_password_user", Password: "wrong"}, "ip:10.0.0.8")
	if err == nil || err.Error() != "user.login.error.password_wrong" {
		t.Fatalf("expected password wrong error, got %v", err)
	}

	events, err := s.ListSecurityEvents(&SecurityEventQuery{
		Username:  "wrong_password_user",
		EventType: "password_wrong",
		Page:      1,
		PageSize:  10,
	})
	if err != nil {
		t.Fatalf("list security events: %v", err)
	}
	if events.Total != 1 || len(events.Items) != 1 {
		t.Fatalf("expected one wrong-password security event, got %+v", events)
	}
	if events.Items[0].Severity != "medium" ||
		events.Items[0].SourceKey != "ip:10.0.0.8" ||
		events.Items[0].IP != "10.0.0.8" ||
		events.Items[0].MessageKey != "auth.security.event.password_wrong" {
		t.Fatalf("unexpected wrong-password security event: %+v", events.Items[0])
	}
}

func TestAuthService_AcknowledgeSecurityEventPersistsAuditFields(t *testing.T) {
	db := setupTestDB(t)
	s := NewAuthService(db)

	event := SystemAuthSecurityEvent{
		UserID:     11,
		Username:   "risk_user",
		EventType:  "source_blocked",
		Severity:   "high",
		SourceKey:  "ip:10.0.0.9",
		MessageKey: "auth.security.event.source_blocked",
		CreatedAt:  time.Now().UTC(),
	}
	if err := db.Create(&event).Error; err != nil {
		t.Fatalf("seed security event: %v", err)
	}

	if err := s.AcknowledgeSecurityEvent(event.ID, 9001, "auditor", "confirmed and user notified"); err != nil {
		t.Fatalf("acknowledge security event: %v", err)
	}

	events, err := s.ListSecurityEvents(&SecurityEventQuery{
		Username:     "risk_user",
		Acknowledged: boolPtr(true),
		Page:         1,
		PageSize:     10,
	})
	if err != nil {
		t.Fatalf("list acknowledged security events: %v", err)
	}
	if events.Total != 1 || len(events.Items) != 1 {
		t.Fatalf("expected one acknowledged security event, got %+v", events)
	}
	if events.Items[0].AcknowledgedAt == nil || events.Items[0].AcknowledgedByUser != "auditor" || events.Items[0].AcknowledgementNote != "confirmed and user notified" {
		t.Fatalf("unexpected acknowledged security event payload: %+v", events.Items[0])
	}
}

func TestAuthService_UpdatePasswordUsesConfiguredMinLength(t *testing.T) {
	db := setupTestDB(t)
	s := NewAuthService(db)

	hash, _ := bcrypt.GenerateFromPassword([]byte("oldpassword"), bcrypt.DefaultCost)
	testUser := user.SystemUser{
		Username: "policy_user",
		Password: string(hash),
		Status:   1,
	}
	db.Create(&testUser)
	_ = db.Exec("INSERT INTO system_setting (setting_key, setting_value) VALUES ('security.password_min_length', '8')")
	_ = s.ReloadSettings()

	err := s.UpdatePassword(testUser.ID, "session123", &PasswordUpdateReq{
		OldPassword: "oldpassword",
		NewPassword: "1234567",
	})
	if err == nil || err.Error() != "user.update.error.password_too_short" {
		t.Fatalf("expected configured min length error, got %v", err)
	}
}

func TestAuthService_UpdatePasswordUsesConfiguredComplexity(t *testing.T) {
	db := setupTestDB(t)
	s := NewAuthService(db)

	hash, _ := bcrypt.GenerateFromPassword([]byte("oldpassword"), bcrypt.DefaultCost)
	testUser := user.SystemUser{
		Username: "complexity_user",
		Password: string(hash),
		Status:   1,
	}
	db.Create(&testUser)
	_ = db.Exec("INSERT INTO system_setting (setting_key, setting_value) VALUES ('security.password_require_digit', 'true'), ('security.password_require_uppercase', 'true')")
	_ = s.ReloadSettings()

	err := s.UpdatePassword(testUser.ID, "session123", &PasswordUpdateReq{
		OldPassword: "oldpassword",
		NewPassword: "longenough",
	})
	if err == nil || err.Error() != "user.update.error.password_weak" {
		t.Fatalf("expected weak password error without digit and uppercase, got %v", err)
	}

	err = s.UpdatePassword(testUser.ID, "session123", &PasswordUpdateReq{
		OldPassword: "oldpassword",
		NewPassword: "Longenough1",
	})
	if err != nil {
		t.Fatalf("expected complex password to pass, got %v", err)
	}
}

func TestAuthService_UpdatePasswordRejectsRecentPasswordReuse(t *testing.T) {
	db := setupTestDB(t)
	s := NewAuthService(db)

	hash, _ := bcrypt.GenerateFromPassword([]byte("oldpassword"), bcrypt.DefaultCost)
	testUser := user.SystemUser{
		Username: "history_user",
		Password: string(hash),
		Status:   1,
	}
	db.Create(&testUser)
	_ = db.Exec("INSERT INTO system_setting (setting_key, setting_value) VALUES ('security.password_history_limit', '2')")
	_ = s.ReloadSettings()

	if err := s.UpdatePassword(testUser.ID, "session123", &PasswordUpdateReq{OldPassword: "oldpassword", NewPassword: "newpassword1"}); err != nil {
		t.Fatalf("first password update: %v", err)
	}
	if err := s.UpdatePassword(testUser.ID, "session123", &PasswordUpdateReq{OldPassword: "newpassword1", NewPassword: "newpassword2"}); err != nil {
		t.Fatalf("second password update: %v", err)
	}

	err := s.UpdatePassword(testUser.ID, "session123", &PasswordUpdateReq{OldPassword: "newpassword2", NewPassword: "oldpassword"})
	if err == nil || err.Error() != "user.password.error.reused" {
		t.Fatalf("expected reused password error, got %v", err)
	}
}

func TestAuthService_UpdateCurrentUserPreferencesReturnsNormalizedPayload(t *testing.T) {
	db := setupTestDB(t)
	s := NewAuthService(db)

	hash, _ := bcrypt.GenerateFromPassword([]byte("123456"), bcrypt.DefaultCost)
	testUser := user.SystemUser{
		Username:       "preference_user",
		Password:       string(hash),
		Status:         1,
		PreferenceJSON: `{"theme":"emerald","layout":"vertical","lang":"zh-CN"}`,
	}
	if err := db.Create(&testUser).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}

	result, err := s.UpdateCurrentUserPreferences(testUser.ID, &UserPlatformPreferenceUpdateReq{
		Theme:       "slate",
		Language:    "en-US",
		LayoutMode:  "horizontal",
		DensityMode: "compact",
	})
	if err != nil {
		t.Fatalf("update preferences: %v", err)
	}
	if result.User == nil || result.User.Preferences == nil {
		t.Fatalf("expected returned user preferences")
	}
	if result.Previous == nil || result.Previous.Theme != "emerald" || result.Previous.LayoutMode != "vertical" || result.Previous.Language != "zh-CN" {
		t.Fatalf("unexpected previous preferences: %+v", result.Previous)
	}
	if result.Current == nil || result.Current.Theme != "slate" || result.Current.Language != "en-US" || result.Current.LayoutMode != "horizontal" || result.Current.DensityMode != "compact" {
		t.Fatalf("unexpected current preferences: %+v", result.Current)
	}

	var updated user.SystemUser
	if err := db.First(&updated, testUser.ID).Error; err != nil {
		t.Fatalf("reload user: %v", err)
	}
	if updated.PreferenceJSON != `{"theme":"slate","language":"en-US","layoutMode":"horizontal","densityMode":"compact"}` {
		t.Fatalf("unexpected persisted preference json: %s", updated.PreferenceJSON)
	}
}

func TestAuthService_CleanupLoginLogsUsesConfiguredRetentionOptions(t *testing.T) {
	db := setupTestDB(t)
	s := NewAuthService(db)

	now := time.Now().UTC()
	if err := db.Create(&[]SystemLogLogin{
		{Username: "u1", Status: 1, LoginTime: now.AddDate(0, 0, -12)},
		{Username: "u2", Status: 1, LoginTime: now.AddDate(0, 0, -3)},
	}).Error; err != nil {
		t.Fatalf("seed login logs: %v", err)
	}
	if err := db.Exec("INSERT INTO system_setting (setting_key, setting_value) VALUES ('audit.login_log_retention_options', '[3,10]')").Error; err != nil {
		t.Fatalf("seed audit retention setting: %v", err)
	}

	clearedCount, err := s.CleanupLoginLogs(10, "", "")
	if err != nil {
		t.Fatalf("cleanup login logs with configured option: %v", err)
	}
	if clearedCount != 1 {
		t.Fatalf("expected to clean 1 login log, got %d", clearedCount)
	}

	_, err = s.CleanupLoginLogs(7, "", "")
	if err == nil || err.Error() != "auth.login_log.cleanup.days_invalid" {
		t.Fatalf("expected invalid retention days error, got %v", err)
	}
}

func TestAuthService_CleanupLoginLogsSupportsExplicitTimeRange(t *testing.T) {
	db := setupTestDB(t)
	s := NewAuthService(db)

	now := time.Now().UTC()
	inRange := now.Add(-4 * time.Hour)
	outOfRange := now.Add(-30 * time.Hour)
	if err := db.Create(&[]SystemLogLogin{
		{Username: "inside", Status: 1, LoginTime: inRange},
		{Username: "outside", Status: 1, LoginTime: outOfRange},
	}).Error; err != nil {
		t.Fatalf("seed login logs: %v", err)
	}

	clearedCount, err := s.CleanupLoginLogs(0, now.Add(-12*time.Hour).Format(time.RFC3339), now.Format(time.RFC3339))
	if err != nil {
		t.Fatalf("cleanup login logs by explicit range: %v", err)
	}
	if clearedCount != 1 {
		t.Fatalf("expected to clean 1 login log in range, got %d", clearedCount)
	}
}

func TestAuthService_ListLoginLogsAppliesAutomaticRetention(t *testing.T) {
	db := setupTestDB(t)
	s := NewAuthService(db)

	if err := db.Exec("CREATE TABLE IF NOT EXISTS system_setting (setting_key VARCHAR(191) PRIMARY KEY, setting_value TEXT)").Error; err != nil {
		t.Fatalf("create system_setting table: %v", err)
	}
	if err := db.Exec("INSERT INTO system_setting (setting_key, setting_value) VALUES ('audit.login_log_retention_days', '3')").Error; err != nil {
		t.Fatalf("seed login log retention days: %v", err)
	}
	_ = s.ReloadSettings()

	now := time.Now().UTC()
	if err := db.Create(&[]SystemLogLogin{
		{Username: "legacy-user", Status: 1, LoginTime: now.AddDate(0, 0, -10)},
		{Username: "recent-user", Status: 1, LoginTime: now.AddDate(0, 0, -1)},
	}).Error; err != nil {
		t.Fatalf("seed login logs: %v", err)
	}

	resp, err := s.ListLoginLogs(&LoginLogQuery{Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("list login logs: %v", err)
	}
	if resp.Total != 1 || len(resp.Items) != 1 || resp.Items[0].Username != "recent-user" {
		t.Fatalf("expected only retained login log, got %+v", resp)
	}
}

func TestAuthService_CleanupHistoricSessionsUsesConfiguredRetentionOptions(t *testing.T) {
	db := setupTestDB(t)
	s := NewAuthService(db)
	now := time.Now().UTC()
	oldRevokedAt := now.AddDate(0, 0, -12)
	recentRevokedAt := now.AddDate(0, 0, -2)

	testUser := user.SystemUser{Username: "cleanup-policy-user", Status: 1}
	if err := db.Create(&testUser).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
	if err := db.Create(&[]SystemUserSession{
		{
			SessionID:        "old-revoked",
			UserID:           testUser.ID,
			RefreshJTI:       "old-jti",
			RefreshExpiresAt: now.AddDate(0, 0, -20),
			RevokedAt:        &oldRevokedAt,
			CreatedAt:        now.AddDate(0, 0, -20),
		},
		{
			SessionID:        "recent-revoked",
			UserID:           testUser.ID,
			RefreshJTI:       "recent-jti",
			RefreshExpiresAt: now.AddDate(0, 0, -4),
			RevokedAt:        &recentRevokedAt,
			CreatedAt:        now.AddDate(0, 0, -4),
		},
	}).Error; err != nil {
		t.Fatalf("seed sessions: %v", err)
	}
	if err := db.Exec("INSERT INTO system_setting (setting_key, setting_value) VALUES ('audit.session_cleanup_retention_options', '[3,10]')").Error; err != nil {
		t.Fatalf("seed session cleanup retention setting: %v", err)
	}

	clearedCount, err := s.CleanupHistoricSessions(10, "", "")
	if err != nil {
		t.Fatalf("cleanup historic sessions with configured option: %v", err)
	}
	if clearedCount != 1 {
		t.Fatalf("expected to clean 1 historic session, got %d", clearedCount)
	}

	_, err = s.CleanupHistoricSessions(7, "", "")
	if err == nil || err.Error() != "auth.session.cleanup.days_invalid" {
		t.Fatalf("expected invalid historic-session retention days error, got %v", err)
	}
}

func TestAuthService_CleanupHistoricSessionsSupportsExplicitTimeRange(t *testing.T) {
	db := setupTestDB(t)
	s := NewAuthService(db)
	now := time.Now().UTC()
	oldRevokedAt := now.Add(-48 * time.Hour)
	recentRevokedAt := now.Add(-2 * time.Hour)

	testUser := user.SystemUser{Username: "session-range-user", Status: 1}
	if err := db.Create(&testUser).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
	if err := db.Create(&[]SystemUserSession{
		{
			SessionID:        "range-old",
			UserID:           testUser.ID,
			RefreshJTI:       "range-old-jti",
			RefreshExpiresAt: now.AddDate(0, 0, -10),
			RevokedAt:        &oldRevokedAt,
			CreatedAt:        now.AddDate(0, 0, -10),
		},
		{
			SessionID:        "range-recent",
			UserID:           testUser.ID,
			RefreshJTI:       "range-recent-jti",
			RefreshExpiresAt: now.AddDate(0, 0, -1),
			RevokedAt:        &recentRevokedAt,
			CreatedAt:        now.AddDate(0, 0, -1),
		},
	}).Error; err != nil {
		t.Fatalf("seed sessions: %v", err)
	}

	clearedCount, err := s.CleanupHistoricSessions(0, now.Add(-12*time.Hour).Format(time.RFC3339), now.Format(time.RFC3339))
	if err != nil {
		t.Fatalf("cleanup sessions by explicit range: %v", err)
	}
	if clearedCount != 1 {
		t.Fatalf("expected to clean 1 revoked session in range, got %d", clearedCount)
	}
}

func TestAuthService_BatchRevokeSessionsSkipsCurrentSessionBoundary(t *testing.T) {
	db := setupTestDB(t)
	s := NewAuthService(db)
	now := time.Now().UTC()
	testUser := user.SystemUser{Username: "batch-revoke-user", Status: 1}
	if err := db.Create(&testUser).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}

	sessions := []SystemUserSession{
		{
			SessionID:        "session-a",
			UserID:           testUser.ID,
			RefreshJTI:       "jti-a",
			RefreshExpiresAt: now.Add(24 * time.Hour),
			CreatedAt:        now.Add(-2 * time.Hour),
		},
		{
			SessionID:        "session-b",
			UserID:           testUser.ID,
			RefreshJTI:       "jti-b",
			RefreshExpiresAt: now.Add(24 * time.Hour),
			CreatedAt:        now.Add(-time.Hour),
		},
	}
	if err := db.Create(&sessions).Error; err != nil {
		t.Fatalf("seed sessions: %v", err)
	}

	revokedCount, err := s.BatchRevokeSessions("current-session", []string{"session-a", "session-b", "session-a"})
	if err != nil {
		t.Fatalf("batch revoke sessions: %v", err)
	}
	if revokedCount != 2 {
		t.Fatalf("expected to revoke 2 sessions, got %d", revokedCount)
	}

	var revokedTotal int64
	if err := db.Model(&SystemUserSession{}).Where("revoked_at IS NOT NULL").Count(&revokedTotal).Error; err != nil {
		t.Fatalf("count revoked sessions: %v", err)
	}
	if revokedTotal != 2 {
		t.Fatalf("expected 2 revoked sessions, got %d", revokedTotal)
	}

	_, err = s.BatchRevokeSessions("session-a", []string{"session-a", "session-b"})
	if err == nil || err.Error() != "auth.session.current_revoke_forbidden" {
		t.Fatalf("expected current-session protection error, got %v", err)
	}
}

func TestAuthService_ListSessionsOnlyReturnsActiveSessions(t *testing.T) {
	db := setupTestDB(t)
	s := NewAuthService(db)
	now := time.Now()
	revokedAt := now.Add(-time.Hour)

	sessions := []SystemUserSession{
		{
			SessionID:        "other-active",
			UserID:           7,
			RefreshJTI:       "jti-other-active",
			RefreshExpiresAt: now.Add(24 * time.Hour),
			CreatedAt:        now.Add(-time.Hour),
		},
		{
			SessionID:        "current-active",
			UserID:           7,
			RefreshJTI:       "jti-current-active",
			RefreshExpiresAt: now.Add(24 * time.Hour),
			CreatedAt:        now.Add(-2 * time.Hour),
		},
		{
			SessionID:        "revoked",
			UserID:           7,
			RefreshJTI:       "jti-revoked",
			RefreshExpiresAt: now.Add(24 * time.Hour),
			RevokedAt:        &revokedAt,
			CreatedAt:        now,
		},
		{
			SessionID:        "expired",
			UserID:           7,
			RefreshJTI:       "jti-expired",
			RefreshExpiresAt: now.Add(-time.Hour),
			CreatedAt:        now,
		},
	}

	if err := db.Create(&sessions).Error; err != nil {
		t.Fatalf("seed sessions: %v", err)
	}

	result, err := s.ListSessions(7, "current-active")
	if err != nil {
		t.Fatalf("list sessions: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 active sessions, got %d: %+v", len(result), result)
	}
	if result[0].SessionID != "current-active" || !result[0].IsCurrent {
		t.Fatalf("expected current active session first, got %+v", result[0])
	}
	if result[1].SessionID != "other-active" {
		t.Fatalf("expected other active session second, got %+v", result[1])
	}
}

func TestAuthService_GetSecurityOverviewIncludesRuntimePolicy(t *testing.T) {
	db := setupTestDB(t)
	s := NewAuthService(db)

	hash, _ := bcrypt.GenerateFromPassword([]byte("12345678"), bcrypt.DefaultCost)
	testUser := user.SystemUser{
		Username: "security_user",
		Nickname: "Security User",
		Password: string(hash),
		Status:   1,
	}
	db.Create(&testUser)

	now := time.Now()
	if err := db.Create(&SystemUserSession{
		SessionID:        "current-session",
		UserID:           testUser.ID,
		RefreshJTI:       "refresh-jti",
		RefreshExpiresAt: now.Add(24 * time.Hour),
		LastRefreshAt:    timePtr(now.Add(-5 * time.Minute)),
		LastActivityAt:   timePtr(now.Add(-2 * time.Minute)),
		LastIP:           "127.0.0.1",
		UserAgent:        "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/122.0 Safari/537.36",
		CreatedAt:        now.Add(-time.Hour),
	}).Error; err != nil {
		t.Fatalf("seed session: %v", err)
	}
	if err := db.Create(&SystemLogLogin{
		Username:  testUser.Username,
		Status:    1,
		LoginTime: now.Add(-30 * time.Minute),
	}).Error; err != nil {
		t.Fatalf("seed login log: %v", err)
	}
	_ = db.Exec("INSERT INTO system_setting (setting_key, setting_value) VALUES ('security.password_min_length', '10'), ('login.max_failed_attempts', '3'), ('login.lock_minutes', '12'), ('login.source_max_failed_attempts', '9'), ('login.source_window_minutes', '20'), ('login.source_lock_minutes', '30'), ('login.captcha_enabled', 'true'), ('login.mfa_enabled', 'false'), ('login.sso_enabled', 'true'), ('login.session_idle_minutes', '45'), ('login.max_active_sessions_per_user', '1'), ('audit.session_retention_days', '90')")
	_ = s.ReloadSettings()

	resp, err := s.GetSecurityOverview(testUser.ID, testUser.Username, "current-session")
	if err != nil {
		t.Fatalf("get security overview: %v", err)
	}
	if resp.Policy.PasswordMinLength != 10 || resp.Policy.MaxFailedAttempts != 3 || resp.Policy.LockMinutes != 12 || resp.Policy.SourceMaxFailedAttempts != 9 || resp.Policy.SourceWindowMinutes != 20 || resp.Policy.SourceLockMinutes != 30 || resp.Policy.SessionIdleMinutes != 45 || resp.Policy.MaxActiveSessions != 1 || resp.Policy.SessionRetentionDays != 90 || !resp.Policy.CaptchaEnabled || resp.Policy.MFAEnabled || !resp.Policy.SSOEnabled {
		t.Fatalf("unexpected security policy: %+v", resp.Policy)
	}
	if resp.CurrentSession == nil || resp.CurrentSession.SessionID != "current-session" {
		t.Fatalf("expected current session to be returned, got %+v", resp.CurrentSession)
	}
	if resp.ActiveSessionCount != 1 {
		t.Fatalf("expected one active session, got %d", resp.ActiveSessionCount)
	}
}

func TestAuthService_GetSecurityOverviewReportsPasswordExpiration(t *testing.T) {
	db := setupTestDB(t)
	s := NewAuthService(db)

	hash, _ := bcrypt.GenerateFromPassword([]byte("12345678"), bcrypt.DefaultCost)
	testUser := user.SystemUser{
		Username: "expired_password_user",
		Password: string(hash),
		Status:   1,
	}
	db.Create(&testUser)
	changedAt := time.Now().AddDate(0, 0, -40)
	if err := db.Create(&SystemUserPasswordHistory{
		UserID:       testUser.ID,
		PasswordHash: "previous-hash",
		ChangedAt:    changedAt,
	}).Error; err != nil {
		t.Fatalf("seed password history: %v", err)
	}
	_ = db.Exec("INSERT INTO system_setting (setting_key, setting_value) VALUES ('security.password_expire_days', '30')")
	_ = s.ReloadSettings()

	resp, err := s.GetSecurityOverview(testUser.ID, testUser.Username, "")
	if err != nil {
		t.Fatalf("get security overview: %v", err)
	}
	if !resp.PasswordExpired || resp.PasswordExpiresAt == nil {
		t.Fatalf("expected expired password details, got %+v", resp)
	}
	if resp.Policy.PasswordExpireDays != 30 {
		t.Fatalf("expected policy to expose password expire days, got %+v", resp.Policy)
	}
}

func TestAuthService_ListAllSessionsSupportsAdminFilters(t *testing.T) {
	db := setupTestDB(t)
	s := NewAuthService(db)
	now := time.Now()
	revokedAt := now.Add(-30 * time.Minute)

	users := []user.SystemUser{
		{Username: "alice", Nickname: "Alice", Status: 1},
		{Username: "bob", Nickname: "Bob", Status: 1},
	}
	if err := db.Create(&users).Error; err != nil {
		t.Fatalf("seed users: %v", err)
	}

	sessions := []SystemUserSession{
		{
			SessionID:        "alice-active",
			UserID:           users[0].ID,
			RefreshJTI:       "alice-active-jti",
			RefreshExpiresAt: now.Add(24 * time.Hour),
			LastRefreshAt:    timePtr(now.Add(-10 * time.Minute)),
			LastActivityAt:   timePtr(now.Add(-5 * time.Minute)),
			LastIP:           "10.0.0.1",
			UserAgent:        "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/122.0 Safari/537.36",
			CreatedAt:        now.Add(-2 * time.Hour),
		},
		{
			SessionID:        "bob-revoked",
			UserID:           users[1].ID,
			RefreshJTI:       "bob-revoked-jti",
			RefreshExpiresAt: now.Add(24 * time.Hour),
			LastRefreshAt:    timePtr(now.Add(-40 * time.Minute)),
			LastActivityAt:   timePtr(now.Add(-35 * time.Minute)),
			LastIP:           "10.0.0.2",
			UserAgent:        "Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) AppleWebKit/605.1.15 Version/17.0 Mobile/15E148 Safari/604.1",
			RevokedAt:        &revokedAt,
			CreatedAt:        now.Add(-3 * time.Hour),
		},
	}
	if err := db.Create(&sessions).Error; err != nil {
		t.Fatalf("seed sessions: %v", err)
	}

	resp, err := s.ListAllSessions(&AdminSessionQuery{
		Username: "alice",
		LastIP:   "10.0.0",
		Browser:  "Chrome",
		OS:       "Windows",
		Device:   "Desktop",
		Status:   intPtr(1),
		Page:     1,
		PageSize: 20,
	})
	if err != nil {
		t.Fatalf("list filtered sessions: %v", err)
	}
	if resp.Total != 1 || len(resp.Items) != 1 {
		t.Fatalf("expected one filtered session, got total=%d len=%d", resp.Total, len(resp.Items))
	}
	if resp.Items[0].SessionID != "alice-active" {
		t.Fatalf("expected alice-active, got %+v", resp.Items[0])
	}
	if resp.Items[0].LastActivityAt == nil {
		t.Fatalf("expected last activity time to be populated")
	}

	revokedResp, err := s.ListAllSessions(&AdminSessionQuery{
		Status:   intPtr(2),
		Page:     1,
		PageSize: 20,
	})
	if err != nil {
		t.Fatalf("list revoked sessions: %v", err)
	}
	if revokedResp.Total != 1 || len(revokedResp.Items) != 1 || revokedResp.Items[0].SessionID != "bob-revoked" {
		t.Fatalf("expected one revoked session, got %+v", revokedResp.Items)
	}
}

func TestAuthService_ListAllSessionsCleansExpiredAndIdleSessions(t *testing.T) {
	db := setupTestDB(t)
	s := NewAuthService(db)
	now := time.Now()

	if err := db.Exec("INSERT INTO system_setting (setting_key, setting_value) VALUES ('login.session_idle_minutes', '30')").Error; err != nil {
		t.Fatalf("seed session idle setting: %v", err)
	}
	if err := s.ReloadSettings(); err != nil {
		t.Fatalf("reload settings: %v", err)
	}

	users := []user.SystemUser{
		{Username: "active-user", Nickname: "Active", Status: 1},
		{Username: "expired-user", Nickname: "Expired", Status: 1},
		{Username: "idle-user", Nickname: "Idle", Status: 1},
	}
	if err := db.Create(&users).Error; err != nil {
		t.Fatalf("seed users: %v", err)
	}

	sessions := []SystemUserSession{
		{
			SessionID:        "active-session",
			UserID:           users[0].ID,
			RefreshJTI:       "active-jti",
			RefreshExpiresAt: now.Add(24 * time.Hour),
			LastRefreshAt:    timePtr(now.Add(-15 * time.Minute)),
			LastActivityAt:   timePtr(now.Add(-5 * time.Minute)),
			CreatedAt:        now.Add(-2 * time.Hour),
		},
		{
			SessionID:        "expired-session",
			UserID:           users[1].ID,
			RefreshJTI:       "expired-jti",
			RefreshExpiresAt: now.Add(-5 * time.Minute),
			LastRefreshAt:    timePtr(now.Add(-10 * time.Minute)),
			LastActivityAt:   timePtr(now.Add(-5 * time.Minute)),
			CreatedAt:        now.Add(-3 * time.Hour),
		},
		{
			SessionID:        "idle-session",
			UserID:           users[2].ID,
			RefreshJTI:       "idle-jti",
			RefreshExpiresAt: now.Add(24 * time.Hour),
			LastRefreshAt:    timePtr(now.Add(-2 * time.Hour)),
			LastActivityAt:   timePtr(now.Add(-40 * time.Minute)),
			CreatedAt:        now.Add(-4 * time.Hour),
		},
	}
	if err := db.Create(&sessions).Error; err != nil {
		t.Fatalf("seed sessions: %v", err)
	}

	resp, err := s.ListAllSessions(&AdminSessionQuery{Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("list all sessions: %v", err)
	}
	if resp.ActiveCount != 1 || resp.RevokedCount != 2 {
		t.Fatalf("expected active=1 revoked=2, got active=%d revoked=%d", resp.ActiveCount, resp.RevokedCount)
	}

	var revokedRows int64
	if err := db.Model(&SystemUserSession{}).
		Where("session_id IN ? AND revoked_at IS NOT NULL", []string{"expired-session", "idle-session"}).
		Count(&revokedRows).Error; err != nil {
		t.Fatalf("count cleaned sessions: %v", err)
	}
	if revokedRows != 2 {
		t.Fatalf("expected cleaned sessions to be revoked, got %d", revokedRows)
	}
}

func TestAuthService_TouchSessionActivityStoresUserAgentAsValue(t *testing.T) {
	db := setupTestDB(t)
	s := NewAuthService(db)
	now := time.Now()

	testUser := user.SystemUser{Username: "agent-safety-user", Status: 1}
	if err := db.Create(&testUser).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
	session := SystemUserSession{
		SessionID:        "agent-safety-session",
		UserID:           testUser.ID,
		RefreshJTI:       "agent-safety-jti",
		RefreshExpiresAt: now.Add(24 * time.Hour),
		LastActivityAt:   timePtr(now.Add(-2 * time.Minute)),
		CreatedAt:        now.Add(-time.Hour),
	}
	if err := db.Create(&session).Error; err != nil {
		t.Fatalf("seed session: %v", err)
	}

	maliciousAgent := "Mozilla/5.0', revoked_at = CURRENT_TIMESTAMP --"
	if err := s.TouchSessionActivity(session.SessionID, testUser.ID, "127.0.0.1", maliciousAgent); err != nil {
		t.Fatalf("touch session activity: %v", err)
	}

	var stored SystemUserSession
	if err := db.Where("session_id = ?", session.SessionID).First(&stored).Error; err != nil {
		t.Fatalf("load session: %v", err)
	}
	if stored.RevokedAt != nil {
		t.Fatalf("expected malicious user agent to be stored as data, not executed as SQL")
	}
	if stored.UserAgent != maliciousAgent {
		t.Fatalf("expected user agent value %q, got %q", maliciousAgent, stored.UserAgent)
	}
	if stored.LastIP != "127.0.0.1" {
		t.Fatalf("expected last ip to update, got %s", stored.LastIP)
	}
}

func TestAuthService_CreateSessionRevokesOlderActiveSessionsByConfiguredLimit(t *testing.T) {
	db := setupTestDB(t)
	s := NewAuthService(db)
	now := time.Now()

	hash, _ := bcrypt.GenerateFromPassword([]byte("12345678"), bcrypt.DefaultCost)
	testUser := user.SystemUser{
		Username: "single-session-user",
		Password: string(hash),
		Status:   1,
	}
	if err := db.Create(&testUser).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
	if err := db.Exec("INSERT INTO system_setting (setting_key, setting_value) VALUES ('login.max_active_sessions_per_user', '1'), ('audit.session_retention_days', '90')").Error; err != nil {
		t.Fatalf("seed session settings: %v", err)
	}
	if err := s.ReloadSettings(); err != nil {
		t.Fatalf("reload settings: %v", err)
	}

	sessions := []SystemUserSession{
		{
			SessionID:        "older-active",
			UserID:           testUser.ID,
			RefreshJTI:       "older-jti",
			RefreshExpiresAt: now.Add(24 * time.Hour),
			LastActivityAt:   timePtr(now.Add(-20 * time.Minute)),
			CreatedAt:        now.Add(-2 * time.Hour),
		},
		{
			SessionID:        "newer-active",
			UserID:           testUser.ID,
			RefreshJTI:       "newer-jti",
			RefreshExpiresAt: now.Add(24 * time.Hour),
			LastActivityAt:   timePtr(now.Add(-5 * time.Minute)),
			CreatedAt:        now.Add(-time.Hour),
		},
	}
	if err := db.Create(&sessions).Error; err != nil {
		t.Fatalf("seed sessions: %v", err)
	}

	pair, err := s.CreateSession(&testUser, nil, "127.0.0.1", "Mozilla/5.0")
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	var activeCount int64
	if err := db.Model(&SystemUserSession{}).
		Where("user_id = ? AND revoked_at IS NULL", testUser.ID).
		Count(&activeCount).Error; err != nil {
		t.Fatalf("count active sessions: %v", err)
	}
	if activeCount != 1 {
		t.Fatalf("expected only one active session after login, got %d", activeCount)
	}

	var current SystemUserSession
	if err := db.Where("session_id = ?", pair.SessionID).First(&current).Error; err != nil {
		t.Fatalf("load current session: %v", err)
	}
	if current.RevokedAt != nil {
		t.Fatalf("expected new session to stay active")
	}
}

func TestAuthService_ListAllSessionsPurgesHistoricSessions(t *testing.T) {
	db := setupTestDB(t)
	s := NewAuthService(db)
	now := time.Now()
	oldRevokedAt := now.AddDate(0, 0, -120)

	if err := db.Exec("INSERT INTO system_setting (setting_key, setting_value) VALUES ('audit.session_retention_days', '90')").Error; err != nil {
		t.Fatalf("seed retention setting: %v", err)
	}
	if err := s.ReloadSettings(); err != nil {
		t.Fatalf("reload settings: %v", err)
	}

	testUser := user.SystemUser{Username: "cleanup-user", Status: 1}
	if err := db.Create(&testUser).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
	if err := db.Create(&[]SystemUserSession{
		{
			SessionID:        "historic-revoked",
			UserID:           testUser.ID,
			RefreshJTI:       "historic-jti",
			RefreshExpiresAt: now.AddDate(0, 0, -110),
			RevokedAt:        &oldRevokedAt,
			CreatedAt:        now.AddDate(0, 0, -150),
		},
		{
			SessionID:        "current-active",
			UserID:           testUser.ID,
			RefreshJTI:       "current-jti",
			RefreshExpiresAt: now.Add(24 * time.Hour),
			LastActivityAt:   timePtr(now.Add(-2 * time.Minute)),
			CreatedAt:        now.Add(-time.Hour),
		},
	}).Error; err != nil {
		t.Fatalf("seed sessions: %v", err)
	}

	resp, err := s.ListAllSessions(&AdminSessionQuery{Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("list sessions: %v", err)
	}
	if resp.Total != 1 || resp.ActiveCount != 1 {
		t.Fatalf("expected only current active session to remain, got total=%d active=%d", resp.Total, resp.ActiveCount)
	}

	var historicCount int64
	if err := db.Model(&SystemUserSession{}).
		Where("session_id = ?", "historic-revoked").
		Count(&historicCount).Error; err != nil {
		t.Fatalf("count historic session: %v", err)
	}
	if historicCount != 0 {
		t.Fatalf("expected historic revoked session to be purged, got %d", historicCount)
	}
}

func TestAuthService_CleanupHistoricSessionsDeletesRevokedHistoryOnly(t *testing.T) {
	db := setupTestDB(t)
	s := NewAuthService(db)
	now := time.Now()
	revokedAt := now.Add(-time.Hour)

	testUser := user.SystemUser{Username: "session-clean-user", Status: 1}
	if err := db.Create(&testUser).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
	if err := db.Create(&[]SystemUserSession{
		{
			SessionID:        "revoked-history",
			UserID:           testUser.ID,
			RefreshJTI:       "revoked-jti",
			RefreshExpiresAt: now.AddDate(0, 0, -1),
			RevokedAt:        &revokedAt,
			CreatedAt:        now.AddDate(0, 0, -10),
		},
		{
			SessionID:        "active-session",
			UserID:           testUser.ID,
			RefreshJTI:       "active-jti",
			RefreshExpiresAt: now.Add(24 * time.Hour),
			LastActivityAt:   timePtr(now.Add(-2 * time.Minute)),
			CreatedAt:        now.Add(-time.Hour),
		},
	}).Error; err != nil {
		t.Fatalf("seed sessions: %v", err)
	}

	clearedCount, err := s.CleanupHistoricSessions(1, "", "")
	if err != nil {
		t.Fatalf("cleanup historic sessions: %v", err)
	}
	if clearedCount != 1 {
		t.Fatalf("expected 1 cleared session, got %d", clearedCount)
	}

	var remaining []SystemUserSession
	if err := db.Order("session_id asc").Find(&remaining).Error; err != nil {
		t.Fatalf("load remaining sessions: %v", err)
	}
	if len(remaining) != 1 || remaining[0].SessionID != "active-session" {
		t.Fatalf("expected only active-session to remain, got %+v", remaining)
	}
}

func TestAuthService_ExportLoginLogs(t *testing.T) {
	db := setupTestDB(t)
	s := NewAuthService(db)

	if err := db.Create(&SystemLogLogin{
		Username:  "tester",
		Ipaddr:    "127.0.0.1",
		Browser:   "Chrome",
		Os:        "Windows",
		Status:    1,
		Msg:       "auth.loginSuccess",
		LoginTime: time.Now(),
	}).Error; err != nil {
		t.Fatalf("seed login log: %v", err)
	}

	exported, err := s.ExportLoginLogs(&LoginLogQuery{Username: "tester"})
	if err != nil {
		t.Fatalf("export login logs: %v", err)
	}
	if len(exported.Rows) != 1 || exported.Rows[0][0] != "tester" {
		t.Fatalf("unexpected exported login log rows: %+v", exported.Rows)
	}
}

func TestAuditService_ExportOperationLogs(t *testing.T) {
	db := setupTestDB(t)
	if err := db.AutoMigrate(&middleware.SystemLogOper{}); err != nil {
		t.Fatalf("migrate operation log: %v", err)
	}
	s := auditmod.NewAuditService(db)

	if err := db.Create(&middleware.SystemLogOper{
		Title:        "导出用户",
		BusinessType: 5,
		Method:       "POST",
		OperName:     "admin",
		OperURL:      "/api/v1/system/user/export",
		OperIP:       "127.0.0.1",
		Status:       1,
		OperTime:     time.Now(),
		CostTime:     12,
	}).Error; err != nil {
		t.Fatalf("seed operation log: %v", err)
	}

	exported, err := s.ExportOperationLogs(&auditmod.OperationLogQuery{Title: "导出用户"})
	if err != nil {
		t.Fatalf("export operation logs: %v", err)
	}
	if len(exported.Rows) != 1 || exported.Rows[0][0] != "导出用户" {
		t.Fatalf("unexpected exported operation log rows: %+v", exported.Rows)
	}
}

func timePtr(value time.Time) *time.Time {
	return &value
}

func intPtr(value int) *int {
	return &value
}
