package session

import (
	"context"
	"testing"
	"time"

	"pantheon-base/pkg/authtoken"
	"pantheon-base/pkg/database"
	"pantheon-base/pkg/testmysql"
	"pantheon-base/pkg/testredis"
)

func setupLifecycleTestDB(t *testing.T) *LifecycleService {
	t.Helper()
	db := testmysql.Open(t)
	if err := db.AutoMigrate(&SystemUserSession{}); err != nil {
		t.Fatalf("migrate sessions: %v", err)
	}
	return NewLifecycleService(db)
}

func TestLifecycleService_RevokeUserSessions(t *testing.T) {
	service := setupLifecycleTestDB(t)
	now := time.Date(2026, 6, 23, 12, 0, 0, 0, time.UTC)

	if err := service.db.Create(&[]SystemUserSession{
		{SessionID: "active-1", UserID: 42, RefreshJTI: "r1", RefreshExpiresAt: now.Add(time.Hour)},
		{SessionID: "active-2", UserID: 42, RefreshJTI: "r2", RefreshExpiresAt: now.Add(time.Hour)},
		{SessionID: "other-user", UserID: 99, RefreshJTI: "r3", RefreshExpiresAt: now.Add(time.Hour)},
	}).Error; err != nil {
		t.Fatalf("seed sessions: %v", err)
	}

	revoked, err := service.RevokeUserSessions(42, now)
	if err != nil {
		t.Fatalf("revoke sessions: %v", err)
	}
	if revoked != 2 {
		t.Fatalf("expected 2 revoked sessions, got %d", revoked)
	}

	var revokedCount int64
	if err := service.db.Model(&SystemUserSession{}).
		Where("user_id = ? AND revoked_at IS NOT NULL", 42).
		Count(&revokedCount).Error; err != nil {
		t.Fatalf("count revoked sessions: %v", err)
	}
	if revokedCount != 2 {
		t.Fatalf("expected 2 revoked rows, got %d", revokedCount)
	}

	var otherRevokedCount int64
	if err := service.db.Model(&SystemUserSession{}).
		Where("user_id = ? AND revoked_at IS NOT NULL", 99).
		Count(&otherRevokedCount).Error; err != nil {
		t.Fatalf("count other user sessions: %v", err)
	}
	if otherRevokedCount != 0 {
		t.Fatalf("expected other user session to remain active, got %d revoked", otherRevokedCount)
	}
}

func TestLifecycleService_DeleteUserSessions(t *testing.T) {
	service := setupLifecycleTestDB(t)
	now := time.Date(2026, 6, 23, 12, 0, 0, 0, time.UTC)

	if err := service.db.Create(&[]SystemUserSession{
		{SessionID: "delete-1", UserID: 42, RefreshJTI: "r1", RefreshExpiresAt: now.Add(time.Hour)},
		{SessionID: "delete-2", UserID: 42, RefreshJTI: "r2", RefreshExpiresAt: now.Add(time.Hour)},
		{SessionID: "keep-1", UserID: 99, RefreshJTI: "r3", RefreshExpiresAt: now.Add(time.Hour)},
	}).Error; err != nil {
		t.Fatalf("seed sessions: %v", err)
	}

	if err := service.DeleteUserSessions(42); err != nil {
		t.Fatalf("delete sessions: %v", err)
	}

	var remainingForUser int64
	if err := service.db.Model(&SystemUserSession{}).
		Where("user_id = ?", 42).
		Count(&remainingForUser).Error; err != nil {
		t.Fatalf("count deleted user sessions: %v", err)
	}
	if remainingForUser != 0 {
		t.Fatalf("expected user sessions to be deleted, got %d", remainingForUser)
	}

	var remainingOther int64
	if err := service.db.Model(&SystemUserSession{}).
		Where("user_id = ?", 99).
		Count(&remainingOther).Error; err != nil {
		t.Fatalf("count other user sessions: %v", err)
	}
	if remainingOther != 1 {
		t.Fatalf("expected other user session to remain, got %d", remainingOther)
	}
}

func TestLifecycleService_RevokeUserTokens(t *testing.T) {
	service := setupLifecycleTestDB(t)
	rdb := testredis.Open(t)
	oldRDB := database.RDB
	database.RDB = rdb
	t.Cleanup(func() { database.RDB = oldRDB })

	now := time.Date(2026, 7, 26, 12, 0, 0, 0, time.UTC)
	if err := service.db.Create(&[]SystemUserSession{
		{SessionID: "revoke-a", UserID: 42, RefreshJTI: "r1", RefreshExpiresAt: now.Add(time.Hour)},
		{SessionID: "revoke-b", UserID: 42, RefreshJTI: "r2", RefreshExpiresAt: now.Add(time.Hour)},
		{SessionID: "keep-c", UserID: 99, RefreshJTI: "r3", RefreshExpiresAt: now.Add(time.Hour)},
	}).Error; err != nil {
		t.Fatalf("seed sessions: %v", err)
	}

	ctx := context.Background()
	for _, seed := range []struct {
		token, sid string
		uid        uint64
	}{
		{token: "rt-a", sid: "revoke-a", uid: 42},
		{token: "rt-b", sid: "revoke-b", uid: 42},
		{token: "rt-c", sid: "keep-c", uid: 99},
	} {
		if err := authtoken.StoreRefresh(ctx, rdb, seed.token, seed.uid, seed.sid, time.Hour); err != nil {
			t.Fatalf("seed refresh token %s: %v", seed.token, err)
		}
	}

	if err := service.RevokeUserTokens(42); err != nil {
		t.Fatalf("revoke user tokens: %v", err)
	}

	blacklistVal, err := rdb.Get(ctx, authtoken.BlacklistUserKey(42)).Result()
	if err != nil || blacklistVal == "" {
		t.Fatalf("expected blacklist key for user 42, got val=%q err=%v", blacklistVal, err)
	}
	ttl, err := rdb.TTL(ctx, authtoken.BlacklistUserKey(42)).Result()
	if err != nil || ttl < authtoken.AccessTokenTTL {
		t.Fatalf("expected blacklist TTL >= access token TTL, got %v err=%v", ttl, err)
	}
	for _, revoked := range []string{"rt-a", "rt-b"} {
		if _, _, err := authtoken.ValidateRefresh(ctx, rdb, revoked); err == nil {
			t.Fatalf("expected refresh token %s revoked", revoked)
		}
	}
	if _, _, err := authtoken.ValidateRefresh(ctx, rdb, "rt-c"); err != nil {
		t.Fatalf("expected other user's refresh token kept, got %v", err)
	}
	if val, err := rdb.Get(ctx, authtoken.BlacklistUserKey(99)).Result(); err == nil {
		t.Fatalf("expected no blacklist for user 99, got %q", val)
	}
}

func TestLifecycleService_PurgeUserAuthArtifacts(t *testing.T) {
	service := setupLifecycleTestDB(t)

	execLifecycleStatements(t, service, []string{
		"CREATE TABLE IF NOT EXISTS system_user_password_history (id BIGINT PRIMARY KEY AUTO_INCREMENT, user_id BIGINT, password_hash VARCHAR(255))",
		"CREATE TABLE IF NOT EXISTS system_auth_factor (id BIGINT PRIMARY KEY AUTO_INCREMENT, user_id BIGINT, factor_type VARCHAR(32), secret VARCHAR(255))",
		"CREATE TABLE IF NOT EXISTS system_auth_mfa_challenge (id BIGINT PRIMARY KEY AUTO_INCREMENT, user_id BIGINT, challenge_id VARCHAR(64))",
	}, "create table")
	execLifecycleStatements(t, service, []string{
		"INSERT INTO system_user_password_history (user_id, password_hash) VALUES (42, 'h1'), (99, 'h2')",
		"INSERT INTO system_auth_factor (user_id, factor_type, secret) VALUES (42, 'totp', 's1'), (99, 'totp', 's2')",
		"INSERT INTO system_auth_mfa_challenge (user_id, challenge_id) VALUES (42, 'c1'), (99, 'c2')",
	}, "seed rows")

	if err := service.PurgeUserAuthArtifacts(42); err != nil {
		t.Fatalf("purge artifacts: %v", err)
	}

	for _, table := range []string{"system_user_password_history", "system_auth_factor", "system_auth_mfa_challenge"} {
		assertLifecycleTableUserRows(t, service, table, 42, 0)
		assertLifecycleTableUserRows(t, service, table, 99, 1)
	}

	// 表不存在时应静默跳过（HasTable 守卫）
	if err := service.db.Exec("DROP TABLE system_auth_mfa_challenge").Error; err != nil {
		t.Fatalf("drop table: %v", err)
	}
	if err := service.PurgeUserAuthArtifacts(99); err != nil {
		t.Fatalf("expected purge to skip missing table, got %v", err)
	}
}

func execLifecycleStatements(t *testing.T, service *LifecycleService, statements []string, action string) {
	t.Helper()
	for _, stmt := range statements {
		if err := service.db.Exec(stmt).Error; err != nil {
			t.Fatalf("%s: %v", action, err)
		}
	}
}

func assertLifecycleTableUserRows(t *testing.T, service *LifecycleService, table string, userID uint64, want int64) {
	t.Helper()
	var got int64
	if err := service.db.Table(table).Where("user_id = ?", userID).Count(&got).Error; err != nil {
		t.Fatalf("count %s for user %d: %v", table, userID, err)
	}
	if got != want {
		t.Fatalf("expected %s rows for user %d to be %d, got %d", table, userID, want, got)
	}
}

func TestLifecycleService_NilAndZeroInputsAreNoops(t *testing.T) {
	var nilService *LifecycleService
	if revoked, err := nilService.RevokeUserSessions(42, time.Now()); err != nil || revoked != 0 {
		t.Fatalf("expected nil service revoke noop, got revoked=%d err=%v", revoked, err)
	}
	if err := nilService.DeleteUserSessions(42); err != nil {
		t.Fatalf("expected nil service delete noop, got %v", err)
	}
	if err := nilService.PurgeUserAuthArtifacts(42); err != nil {
		t.Fatalf("expected nil service purge noop, got %v", err)
	}
	if err := nilService.RevokeUserTokens(42); err != nil {
		t.Fatalf("expected nil service token revoke noop, got %v", err)
	}

	service := NewLifecycleService(nil)
	if revoked, err := service.RevokeUserSessions(42, time.Now()); err != nil || revoked != 0 {
		t.Fatalf("expected nil db revoke noop, got revoked=%d err=%v", revoked, err)
	}
	if err := service.DeleteUserSessions(42); err != nil {
		t.Fatalf("expected nil db delete noop, got %v", err)
	}
}
