package login

import (
	"context"
	"testing"

	"github.com/duanxldragon/pantheon-base/backend/pkg/authtoken"
	"github.com/duanxldragon/pantheon-base/backend/pkg/tenant"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// setupBlacklistTestDB uses the full login fixture: session creation touches
// system_user_session and the policy cache reads system_setting.
func setupBlacklistTestDB(t *testing.T) (*gorm.DB, *Runtime) {
	t.Helper()
	db := setupTestDB(t)
	if err := db.AutoMigrate(&tenant.Membership{}); err != nil {
		t.Fatalf("migrate tenant memberships: %v", err)
	}
	return db, NewRuntime(db, testCredentialRepo(db))
}

// setupBlacklistRedis isolates this file on its own Redis database: these
// tests assert that a key survives an operation, and the shared DB 1 is flushed
// by other packages running in parallel under `go test ./...`.
func setupBlacklistRedis(t *testing.T) *redis.Client {
	t.Helper()
	t.Setenv("PANTHEON_TEST_REDIS_DB", "9")
	return setupTestRedis(t)
}

func assertUserBlacklist(t *testing.T, rdb *redis.Client, userID uint64, want bool) {
	t.Helper()
	_, err := rdb.Get(context.Background(), authtoken.BlacklistUserKey(userID)).Result()
	exists := err == nil
	if err != redis.Nil && err != nil {
		t.Fatalf("read user blacklist: %v", err)
	}
	if exists != want {
		t.Fatalf("user blacklist exists = %v, want %v", exists, want)
	}
}

// F2: RevokeUserTokens writes blacklist:<userID> with TTL AccessTokenTTL+1m and
// login never cleared it, so a password reset or a disable/enable cycle locked
// the account out for ~16 minutes with 401 token.expired.force.
func TestSuccessfulLoginClearsUserBlacklist(t *testing.T) {
	db, rt := setupBlacklistTestDB(t)
	rdb := setupBlacklistRedis(t)

	user := createPasswordUser(t, db, "lockout_user", "123456")
	if err := authtoken.BlacklistUser(context.Background(), rdb, user.ID); err != nil {
		t.Fatalf("seed user blacklist: %v", err)
	}
	// Session-level revocation for an old session must survive the clear.
	oldSessionID := "old-session"
	if err := authtoken.BlacklistSession(context.Background(), rdb, oldSessionID); err != nil {
		t.Fatalf("seed session blacklist: %v", err)
	}

	if _, err := rt.Authenticate(&LoginReq{Username: "lockout_user", Password: "123456"}); err != nil {
		t.Fatalf("authenticate: %v", err)
	}
	if _, err := rt.CreateSessionForTenantWithContext(context.Background(), user.ID, nil, "127.0.0.1", "test", 0); err != nil {
		t.Fatalf("create session: %v", err)
	}

	assertUserBlacklist(t, rdb, user.ID, false)

	if _, err := rdb.Get(context.Background(), authtoken.BlacklistSessionKey(oldSessionID)).Result(); err != nil {
		t.Fatalf("session blacklist was cleared with the user marker: %v", err)
	}
}

// Clearing before the tenant gate would let a failed login erase a lockout that
// a disable action just imposed. The user here is tenant-scoped with no active
// membership, so issuance is denied and the marker must stay.
func TestFailedIssuanceKeepsUserBlacklist(t *testing.T) {
	db, rt := setupBlacklistTestDB(t)
	rdb := setupBlacklistRedis(t)
	seedSettings(t, db, map[string]string{tenant.FeatureFlagSettingKey: tenant.ModeMulti})
	if err := rt.ReloadSettings(); err != nil {
		t.Fatalf("reload settings: %v", err)
	}
	if err := db.Create(&tenant.Tenant{ID: 101, Code: "alpha", Name: "alpha", Status: tenant.TenantStatusActive}).Error; err != nil {
		t.Fatalf("seed tenant: %v", err)
	}
	user := createPasswordUser(t, db, "tenant_no_member", "123456")
	if err := db.Model(&user).Update("tenant_id", 101).Error; err != nil {
		t.Fatalf("tenant-scope user: %v", err)
	}

	if err := authtoken.BlacklistUser(context.Background(), rdb, user.ID); err != nil {
		t.Fatalf("seed user blacklist: %v", err)
	}

	if _, err := rt.CreateSessionForTenantWithContext(context.Background(), user.ID, nil, "127.0.0.1", "test", 0); err == nil {
		t.Fatal("expected issuance to fail for a tenant-scoped subject without membership")
	}
	assertUserBlacklist(t, rdb, user.ID, true)
}

// A bad password must not clear the marker either: only a completed login is
// proof that the new credential works.
func TestFailedCredentialKeepsUserBlacklist(t *testing.T) {
	db, rt := setupBlacklistTestDB(t)
	rdb := setupBlacklistRedis(t)

	user := createPasswordUser(t, db, "wrong_password_user", "123456")
	if err := authtoken.BlacklistUser(context.Background(), rdb, user.ID); err != nil {
		t.Fatalf("seed user blacklist: %v", err)
	}

	if _, err := rt.Authenticate(&LoginReq{Username: "wrong_password_user", Password: "nope"}); err == nil {
		t.Fatal("expected wrong password to fail")
	}
	assertUserBlacklist(t, rdb, user.ID, true)
}

// The real sequence from the acceptance run: disable (revoke tokens), enable,
// log in. The login must succeed instead of answering token.expired.force.
func TestDisableEnableCycleLetsTheUserLogInAgain(t *testing.T) {
	db, rt := setupBlacklistTestDB(t)
	rdb := setupBlacklistRedis(t)

	user := createPasswordUser(t, db, "toggled_user", "123456")
	if err := db.Model(&user).Update("status", 2).Error; err != nil {
		t.Fatalf("disable user: %v", err)
	}
	if err := authtoken.BlacklistUser(context.Background(), rdb, user.ID); err != nil {
		t.Fatalf("revoke tokens on disable: %v", err)
	}

	if _, err := rt.Authenticate(&LoginReq{Username: "toggled_user", Password: "123456"}); err == nil {
		t.Fatal("expected a disabled user to be rejected")
	}
	assertUserBlacklist(t, rdb, user.ID, true)

	if err := db.Model(&user).Update("status", 1).Error; err != nil {
		t.Fatalf("re-enable user: %v", err)
	}
	authenticated, err := rt.Authenticate(&LoginReq{Username: "toggled_user", Password: "123456"})
	if err != nil {
		t.Fatalf("authenticate after re-enable: %v", err)
	}
	if _, err := rt.CreateSessionForTenantWithContext(context.Background(), authenticated.ID, nil, "127.0.0.1", "test", 0); err != nil {
		t.Fatalf("create session after re-enable: %v", err)
	}
	assertUserBlacklist(t, rdb, user.ID, false)
}
