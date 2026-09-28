package login

import (
	"testing"
	"time"

	"github.com/duanxldragon/pantheon-base/backend/modules/auth/security"
	user "github.com/duanxldragon/pantheon-base/backend/modules/system/iam/user"
	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
	"github.com/duanxldragon/pantheon-base/backend/pkg/testmysql"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// stubPolicyProvider returns a fixed runtime policy so throttle tests don't
// depend on the settings store.
type stubPolicyProvider struct {
	policy RuntimePolicy
}

func (s *stubPolicyProvider) GetRuntimePolicy() RuntimePolicy {
	return s.policy
}

// recordingSecurityEventRecorder captures emitted security events.
type recordingSecurityEventRecorder struct {
	events []security.SystemAuthSecurityEvent
}

func (s *recordingSecurityEventRecorder) RecordSecurityEvent(event security.SystemAuthSecurityEvent) {
	s.events = append(s.events, event)
}

func setupThrottleTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db := testmysql.Open(t)
	_ = db.AutoMigrate(&user.SystemUser{}, &SystemLoginThrottle{}, &SystemLogLogin{})
	return db
}

func seedThrottleUser(t *testing.T, db *gorm.DB, username, password string) user.SystemUser {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("hash test password: %v", err)
	}
	u := user.SystemUser{Username: username, Password: string(hash), Status: common.StatusEnabled}
	if err := db.Create(&u).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
	return u
}

func defaultThrottlePolicy() RuntimePolicy {
	return RuntimePolicy{
		MaxFailedAttempts:       5,
		LockMinutes:             15,
		SourceMaxFailedAttempts: 5,
		SourceWindowMinutes:     15,
		SourceLockMinutes:       15,
		SecurityEventEnabled:    true,
	}
}

func TestLoginService_RecordFailedLoginAttemptIncrementsCount(t *testing.T) {
	db := setupThrottleTestDB(t)
	u := seedThrottleUser(t, db, "counter_user", "rightpass")
	policy := defaultThrottlePolicy()

	locked, err := NewLoginService(db, testCredentialRepo(db), &stubPolicyProvider{policy: policy}, nil).recordFailedLoginAttempt(loadAuthUser(t, db, u.ID), policy)
	if err != nil {
		t.Fatalf("record failed attempt: %v", err)
	}
	if locked {
		t.Fatal("expected not locked after first failure")
	}

	var reloaded user.SystemUser
	if err := db.First(&reloaded, u.ID).Error; err != nil {
		t.Fatalf("reload user: %v", err)
	}
	if reloaded.FailedLoginAttempts != 1 {
		t.Fatalf("expected failed attempts 1, got %d", reloaded.FailedLoginAttempts)
	}
	if reloaded.LoginLockedUntil != nil {
		t.Fatalf("expected no lock yet, got %v", reloaded.LoginLockedUntil)
	}
}

func TestLoginService_RecordFailedLoginAttemptLocksAtThreshold(t *testing.T) {
	db := setupThrottleTestDB(t)
	u := seedThrottleUser(t, db, "lock_user", "rightpass")
	policy := defaultThrottlePolicy()
	svc := NewLoginService(db, testCredentialRepo(db), &stubPolicyProvider{policy: policy}, nil)

	// 4 failures: not yet locked (threshold 5)
	for i := 0; i < 4; i++ {
		locked, err := svc.recordFailedLoginAttempt(loadAuthUser(t, db, u.ID), policy)
		if err != nil {
			t.Fatalf("record failed attempt %d: %v", i+1, err)
		}
		if locked {
			t.Fatalf("expected not locked on attempt %d", i+1)
		}
	}

	// 5th failure: locked, counter reset
	locked, err := svc.recordFailedLoginAttempt(loadAuthUser(t, db, u.ID), policy)
	if err != nil {
		t.Fatalf("record 5th failed attempt: %v", err)
	}
	if !locked {
		t.Fatal("expected locked at threshold")
	}

	var reloaded user.SystemUser
	if err := db.First(&reloaded, u.ID).Error; err != nil {
		t.Fatalf("reload user: %v", err)
	}
	if reloaded.FailedLoginAttempts != 0 {
		t.Fatalf("expected counter reset on lock, got %d", reloaded.FailedLoginAttempts)
	}
	if reloaded.LoginLockedUntil == nil || !reloaded.LoginLockedUntil.After(time.Now()) {
		t.Fatalf("expected future lock time, got %v", reloaded.LoginLockedUntil)
	}
}

func TestLoginService_RecordFailedLoginAttemptClearsExpiredLock(t *testing.T) {
	db := setupThrottleTestDB(t)
	u := seedThrottleUser(t, db, "expired_lock_user", "rightpass")
	policy := defaultThrottlePolicy()

	// 3 prior attempts + an already-expired lock: the expired lock is cleared
	// and the counter keeps incrementing without re-locking.
	u.FailedLoginAttempts = 3
	expired := time.Now().Add(-time.Hour)
	u.LoginLockedUntil = &expired
	if err := db.Save(&u).Error; err != nil {
		t.Fatalf("seed expired lock: %v", err)
	}

	locked, err := NewLoginService(db, testCredentialRepo(db), &stubPolicyProvider{policy: policy}, nil).recordFailedLoginAttempt(loadAuthUser(t, db, u.ID), policy)
	if err != nil {
		t.Fatalf("record failed attempt: %v", err)
	}
	if locked {
		t.Fatal("expected expired lock to be cleared, not re-lock")
	}

	var reloaded user.SystemUser
	if err := db.First(&reloaded, u.ID).Error; err != nil {
		t.Fatalf("reload user: %v", err)
	}
	if reloaded.LoginLockedUntil != nil {
		t.Fatalf("expected expired lock cleared, got %v", reloaded.LoginLockedUntil)
	}
	if reloaded.FailedLoginAttempts != 4 {
		t.Fatalf("expected counter incremented to 4, got %d", reloaded.FailedLoginAttempts)
	}
}

func TestLoginService_RecordSourceFailureCreatesThrottleRow(t *testing.T) {
	db := setupThrottleTestDB(t)
	policy := defaultThrottlePolicy()
	svc := NewLoginService(db, testCredentialRepo(db), &stubPolicyProvider{policy: policy}, nil)

	blocked, err := svc.recordSourceFailure("ip:10.0.0.1", policy, time.Now())
	if err != nil {
		t.Fatalf("record source failure: %v", err)
	}
	if blocked {
		t.Fatal("expected not blocked on first failure")
	}

	var throttle SystemLoginThrottle
	if err := db.Where("source_key = ?", "ip:10.0.0.1").First(&throttle).Error; err != nil {
		t.Fatalf("load throttle: %v", err)
	}
	if throttle.FailureCount != 1 {
		t.Fatalf("expected failure count 1, got %d", throttle.FailureCount)
	}
	if throttle.WindowStartedAt == nil || throttle.LastAttemptAt == nil {
		t.Fatalf("expected window timestamps set, got %+v", throttle)
	}
	if throttle.BlockedUntil != nil {
		t.Fatalf("expected no block yet, got %v", throttle.BlockedUntil)
	}
}

func TestLoginService_RecordSourceFailureBlocksAtThreshold(t *testing.T) {
	db := setupThrottleTestDB(t)
	policy := defaultThrottlePolicy()
	svc := NewLoginService(db, testCredentialRepo(db), &stubPolicyProvider{policy: policy}, nil)

	for i := 0; i < 4; i++ {
		blocked, err := svc.recordSourceFailure("ip:10.0.0.2", policy, time.Now())
		if err != nil {
			t.Fatalf("record source failure %d: %v", i+1, err)
		}
		if blocked {
			t.Fatalf("expected not blocked on failure %d", i+1)
		}
	}

	blocked, err := svc.recordSourceFailure("ip:10.0.0.2", policy, time.Now())
	if err != nil {
		t.Fatalf("record 5th source failure: %v", err)
	}
	if !blocked {
		t.Fatal("expected source blocked at threshold")
	}

	var throttle SystemLoginThrottle
	if err := db.Where("source_key = ?", "ip:10.0.0.2").First(&throttle).Error; err != nil {
		t.Fatalf("load throttle: %v", err)
	}
	if throttle.FailureCount != 5 {
		t.Fatalf("expected failure count 5, got %d", throttle.FailureCount)
	}
	if throttle.BlockedUntil == nil || !throttle.BlockedUntil.After(time.Now()) {
		t.Fatalf("expected future block time, got %v", throttle.BlockedUntil)
	}
}

func TestLoginService_CheckSourceThrottleBlocksWhenBlocked(t *testing.T) {
	db := setupThrottleTestDB(t)
	policy := defaultThrottlePolicy()

	blockedUntil := time.Now().Add(10 * time.Minute)
	if err := db.Create(&SystemLoginThrottle{
		SourceKey:    "ip:10.0.0.3",
		FailureCount: 5,
		BlockedUntil: &blockedUntil,
	}).Error; err != nil {
		t.Fatalf("seed throttle: %v", err)
	}

	blocked, err := NewLoginService(db, testCredentialRepo(db), &stubPolicyProvider{policy: policy}, nil).checkSourceThrottle("ip:10.0.0.3", policy, time.Now())
	if err != nil {
		t.Fatalf("check source throttle: %v", err)
	}
	if !blocked {
		t.Fatal("expected source to be blocked")
	}
}

func TestLoginService_CheckSourceThrottleResetsExpiredBlock(t *testing.T) {
	db := setupThrottleTestDB(t)
	policy := defaultThrottlePolicy()

	expired := time.Now().Add(-time.Minute)
	if err := db.Create(&SystemLoginThrottle{
		SourceKey:    "ip:10.0.0.4",
		FailureCount: 5,
		BlockedUntil: &expired,
	}).Error; err != nil {
		t.Fatalf("seed throttle: %v", err)
	}

	blocked, err := NewLoginService(db, testCredentialRepo(db), &stubPolicyProvider{policy: policy}, nil).checkSourceThrottle("ip:10.0.0.4", policy, time.Now())
	if err != nil {
		t.Fatalf("check source throttle: %v", err)
	}
	if blocked {
		t.Fatal("expected expired block to be cleared")
	}

	var throttle SystemLoginThrottle
	if err := db.Where("source_key = ?", "ip:10.0.0.4").First(&throttle).Error; err != nil {
		t.Fatalf("load throttle: %v", err)
	}
	if throttle.FailureCount != 0 || throttle.BlockedUntil != nil {
		t.Fatalf("expected throttle reset, got count=%d blocked=%v", throttle.FailureCount, throttle.BlockedUntil)
	}
}

func TestLoginService_CheckSourceThrottleEmptyKeyOrDisabledPolicyNeverBlocks(t *testing.T) {
	db := setupThrottleTestDB(t)
	policy := defaultThrottlePolicy()

	// Empty source key: never blocks.
	blocked, err := NewLoginService(db, testCredentialRepo(db), &stubPolicyProvider{policy: policy}, nil).checkSourceThrottle("  ", policy, time.Now())
	if err != nil {
		t.Fatalf("check empty source: %v", err)
	}
	if blocked {
		t.Fatal("expected empty source key to never block")
	}

	// Disabled source throttle (SourceMaxFailedAttempts <= 0): never blocks.
	disabledPolicy := defaultThrottlePolicy()
	disabledPolicy.SourceMaxFailedAttempts = 0
	blocked, err = NewLoginService(db, testCredentialRepo(db), &stubPolicyProvider{policy: disabledPolicy}, nil).checkSourceThrottle("ip:10.0.0.5", disabledPolicy, time.Now())
	if err != nil {
		t.Fatalf("check disabled source throttle: %v", err)
	}
	if blocked {
		t.Fatal("expected disabled source throttle to never block")
	}
}

func TestLoginService_AuthenticateWithSourcePreBlockedSourceReturnsError(t *testing.T) {
	db := setupThrottleTestDB(t)
	u := seedThrottleUser(t, db, "blocked_user", "rightpass")
	policy := defaultThrottlePolicy()
	svc := NewLoginService(db, testCredentialRepo(db), &stubPolicyProvider{policy: policy}, nil)

	// Pre-block the source so ensureSourceThrottleAllowed fails immediately.
	blockedUntil := time.Now().Add(10 * time.Minute)
	if err := db.Create(&SystemLoginThrottle{
		SourceKey:    "ip:10.0.0.6",
		FailureCount: 5,
		BlockedUntil: &blockedUntil,
	}).Error; err != nil {
		t.Fatalf("seed throttle: %v", err)
	}

	_, err := svc.AuthenticateWithSource(&LoginReq{Username: u.Username, Password: "rightpass"}, "ip:10.0.0.6")
	if err == nil || err.Error() != "auth.login.error.source_blocked" {
		t.Fatalf("expected source blocked error, got %v", err)
	}
}

func TestLoginService_FailLoginSourceBlockedEmitsSecurityEvent(t *testing.T) {
	db := setupThrottleTestDB(t)
	u := seedThrottleUser(t, db, "event_user", "rightpass")
	policy := defaultThrottlePolicy()
	recorder := &recordingSecurityEventRecorder{}
	svc := NewLoginService(db, testCredentialRepo(db), &stubPolicyProvider{policy: policy}, recorder)

	// Pre-block the source so recordSourceFailure reports blocked immediately.
	blockedUntil := time.Now().Add(10 * time.Minute)
	if err := db.Create(&SystemLoginThrottle{
		SourceKey:    "ip:10.0.0.7",
		FailureCount: 5,
		BlockedUntil: &blockedUntil,
	}).Error; err != nil {
		t.Fatalf("seed throttle: %v", err)
	}

	err := svc.failLoginSourceBlocked(loadAuthUser(t, db, u.ID), "ip:10.0.0.7", policy, time.Now())
	if err == nil || err.Error() != "auth.login.error.source_blocked" {
		t.Fatalf("expected source blocked error, got %v", err)
	}
	if len(recorder.events) != 1 {
		t.Fatalf("expected one security event, got %d", len(recorder.events))
	}
	if recorder.events[0].EventType != "source_blocked" || recorder.events[0].Severity != "high" || recorder.events[0].SourceKey != "ip:10.0.0.7" {
		t.Fatalf("unexpected security event: %+v", recorder.events[0])
	}
	if recorder.events[0].UserID != u.ID || recorder.events[0].Username != u.Username {
		t.Fatalf("expected user identity in event, got %+v", recorder.events[0])
	}
}

func TestLoginService_EmitSecurityEventSkippedWhenDisabledOrNoRecorder(t *testing.T) {
	db := setupThrottleTestDB(t)
	u := seedThrottleUser(t, db, "quiet_user", "rightpass")

	// Security events disabled in policy: recorder must not be called.
	disabledPolicy := defaultThrottlePolicy()
	disabledPolicy.SecurityEventEnabled = false
	recorder := &recordingSecurityEventRecorder{}
	NewLoginService(db, testCredentialRepo(db), &stubPolicyProvider{policy: disabledPolicy}, recorder).emitSecurityEvent(loadAuthUser(t, db, u.ID), "password_wrong", "medium", "ip:1.1.1.1", "key", "1.1.1.1")
	if len(recorder.events) != 0 {
		t.Fatalf("expected no events when disabled, got %d", len(recorder.events))
	}

	// Enabled policy but nil recorder: must not panic.
	NewLoginService(db, testCredentialRepo(db), &stubPolicyProvider{policy: defaultThrottlePolicy()}, nil).emitSecurityEvent(loadAuthUser(t, db, u.ID), "password_wrong", "medium", "ip:1.1.1.1", "key", "1.1.1.1")
}

func TestLoginService_AuthenticateWithSourcePasswordMismatchRecordsFailures(t *testing.T) {
	db := setupThrottleTestDB(t)
	u := seedThrottleUser(t, db, "mismatch_user", "rightpass")
	policy := defaultThrottlePolicy()
	recorder := &recordingSecurityEventRecorder{}
	svc := NewLoginService(db, testCredentialRepo(db), &stubPolicyProvider{policy: policy}, recorder)

	_, err := svc.AuthenticateWithSource(&LoginReq{Username: u.Username, Password: "wrong"}, "ip:10.0.0.8")
	if err == nil || err.Error() != "user.login.error.password_wrong" {
		t.Fatalf("expected password wrong error, got %v", err)
	}

	var reloaded user.SystemUser
	if err := db.First(&reloaded, u.ID).Error; err != nil {
		t.Fatalf("reload user: %v", err)
	}
	if reloaded.FailedLoginAttempts != 1 {
		t.Fatalf("expected user failed attempts 1, got %d", reloaded.FailedLoginAttempts)
	}

	var throttle SystemLoginThrottle
	if err := db.Where("source_key = ?", "ip:10.0.0.8").First(&throttle).Error; err != nil {
		t.Fatalf("load throttle: %v", err)
	}
	if throttle.FailureCount != 1 {
		t.Fatalf("expected source failure count 1, got %d", throttle.FailureCount)
	}

	if len(recorder.events) != 1 || recorder.events[0].EventType != "password_wrong" {
		t.Fatalf("expected password_wrong security event, got %+v", recorder.events)
	}
}

func TestLoginService_AuthenticateWithSourceDisabledUserFails(t *testing.T) {
	db := setupThrottleTestDB(t)
	u := seedThrottleUser(t, db, "disabled_user", "rightpass")
	policy := defaultThrottlePolicy()

	u.Status = common.StatusDisabled
	if err := db.Save(&u).Error; err != nil {
		t.Fatalf("disable user: %v", err)
	}

	_, err := NewLoginService(db, testCredentialRepo(db), &stubPolicyProvider{policy: policy}, nil).AuthenticateWithSource(
		&LoginReq{Username: u.Username, Password: "rightpass"},
		"ip:10.0.0.9",
	)
	if err == nil || err.Error() != "user.login.error.disabled" {
		t.Fatalf("expected disabled error, got %v", err)
	}
}

func TestLoginService_AuthenticateWithSourceLockedUserFails(t *testing.T) {
	db := setupThrottleTestDB(t)
	u := seedThrottleUser(t, db, "locked_user_2", "rightpass")
	policy := defaultThrottlePolicy()

	lockUntil := time.Now().Add(10 * time.Minute)
	u.LoginLockedUntil = &lockUntil
	if err := db.Save(&u).Error; err != nil {
		t.Fatalf("lock user: %v", err)
	}

	_, err := NewLoginService(db, testCredentialRepo(db), &stubPolicyProvider{policy: policy}, nil).AuthenticateWithSource(
		&LoginReq{Username: u.Username, Password: "rightpass"},
		"ip:10.0.0.10",
	)
	if err == nil || err.Error() != "user.login.error.locked" {
		t.Fatalf("expected locked error, got %v", err)
	}
}

func TestLoginService_AuthenticateWithSourceSuccessClearsFailedState(t *testing.T) {
	db := setupThrottleTestDB(t)
	u := seedThrottleUser(t, db, "recovery_user", "rightpass")
	policy := defaultThrottlePolicy()

	// Seed prior failed state: counter at 3 with an expired lock.
	u.FailedLoginAttempts = 3
	expired := time.Now().Add(-time.Hour)
	u.LoginLockedUntil = &expired
	if err := db.Save(&u).Error; err != nil {
		t.Fatalf("seed failed state: %v", err)
	}

	current, err := NewLoginService(db, testCredentialRepo(db), &stubPolicyProvider{policy: policy}, nil).AuthenticateWithSource(
		&LoginReq{Username: u.Username, Password: "rightpass"},
		"",
	)
	if err != nil {
		t.Fatalf("expected successful authentication, got %v", err)
	}
	if current.ID != u.ID || current.Username != u.Username {
		t.Fatalf("unexpected authenticated user: %+v", current)
	}

	var reloaded user.SystemUser
	if err := db.First(&reloaded, u.ID).Error; err != nil {
		t.Fatalf("reload user: %v", err)
	}
	if reloaded.FailedLoginAttempts != 0 {
		t.Fatalf("expected failed attempts cleared, got %d", reloaded.FailedLoginAttempts)
	}
	if reloaded.LoginLockedUntil != nil {
		t.Fatalf("expected lock cleared, got %v", reloaded.LoginLockedUntil)
	}
}

func TestLoginService_AuthenticateWithSourceEmptyUsernameCountsSourceFailure(t *testing.T) {
	db := setupThrottleTestDB(t)
	policy := defaultThrottlePolicy()
	recorder := &recordingSecurityEventRecorder{}
	svc := NewLoginService(db, testCredentialRepo(db), &stubPolicyProvider{policy: policy}, recorder)

	_, err := svc.AuthenticateWithSource(&LoginReq{Username: "   ", Password: "x"}, "ip:10.0.0.11")
	if err == nil || err.Error() != "user.login.error.not_found" {
		t.Fatalf("expected not_found error for empty username, got %v", err)
	}

	var throttle SystemLoginThrottle
	if err := db.Where("source_key = ?", "ip:10.0.0.11").First(&throttle).Error; err != nil {
		t.Fatalf("load throttle: %v", err)
	}
	if throttle.FailureCount != 1 {
		t.Fatalf("expected source failure recorded for empty username, got %d", throttle.FailureCount)
	}
}

func TestLoginService_AuthenticateWithSourceNilDBReturnsError(t *testing.T) {
	_, err := NewLoginService(nil, testCredentialRepo(nil), &stubPolicyProvider{policy: defaultThrottlePolicy()}, nil).AuthenticateWithSource(
		&LoginReq{Username: "x", Password: "y"},
		"ip:10.0.0.12",
	)
	if err == nil || err.Error() != "database.not_initialized" {
		t.Fatalf("expected database.not_initialized error, got %v", err)
	}
}
