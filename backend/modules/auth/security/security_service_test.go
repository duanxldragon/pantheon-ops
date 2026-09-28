package security

import (
	"testing"
	"time"

	"github.com/duanxldragon/pantheon-base/backend/pkg/testmysql"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// TestPasswordMatchesComplexity covers every combination of the require-digit /
// require-upper policy branches, which previously had no direct test.
func TestPasswordMatchesComplexity(t *testing.T) {
	cases := []struct {
		name         string
		password     string
		requireDigit bool
		requireUpper bool
		want         bool
	}{
		{"no policy accepts anything", "abc", false, false, true},
		{"require digit, has digit", "abc1", true, false, true},
		{"require digit, missing digit", "abcdef", true, false, false},
		{"require upper, has upper", "abcD", false, true, true},
		{"require upper, missing upper", "abcdef", false, true, false},
		{"require both, satisfies both", "Abc1", true, true, true},
		{"require both, missing digit", "Abcdef", true, true, false},
		{"require both, missing upper", "abc123", true, true, false},
		{"require both, empty password", "", true, true, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			policy := AuthRuntimePolicy{
				PasswordRequireDigit: tc.requireDigit,
				PasswordRequireUpper: tc.requireUpper,
			}
			if got := passwordMatchesComplexity(tc.password, policy); got != tc.want {
				t.Fatalf("passwordMatchesComplexity(%q, digit=%v upper=%v) = %v, want %v",
					tc.password, tc.requireDigit, tc.requireUpper, got, tc.want)
			}
		})
	}
}

// TestEnsurePasswordNotRecentlyUsed_NoDBBranches covers the branches that resolve
// before any database access: history disabled, and reuse of the current password.
func TestEnsurePasswordNotRecentlyUsed_NoDBBranches(t *testing.T) {
	svc := &Service{} // db intentionally nil: these branches must not touch it

	t.Run("history limit disabled returns nil", func(t *testing.T) {
		policy := AuthRuntimePolicy{PasswordHistoryLimit: 0}
		if err := svc.ensurePasswordNotRecentlyUsed(1, "NewPass123", "irrelevant-hash", policy); err != nil {
			t.Fatalf("expected nil when history disabled, got %v", err)
		}
	})

	t.Run("reusing current password is rejected", func(t *testing.T) {
		newPassword := "NewPass123"
		currentHash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
		if err != nil {
			t.Fatalf("hash setup failed: %v", err)
		}
		policy := AuthRuntimePolicy{PasswordHistoryLimit: 3}
		err = svc.ensurePasswordNotRecentlyUsed(1, newPassword, string(currentHash), policy)
		if err == nil || err.Error() != "user.password.error.reused" {
			t.Fatalf("expected user.password.error.reused, got %v", err)
		}
	})
}

// TestEnsurePasswordNotRecentlyUsed_History exercises the DB-backed history lookup
// branch: a password matching an older history row must be rejected, and a fresh
// password must pass. Skips automatically when no test DSN is configured.
func TestEnsurePasswordNotRecentlyUsed_History(t *testing.T) {
	db := testmysql.Open(t)
	if err := db.AutoMigrate(&SystemUserPasswordHistory{}); err != nil {
		t.Fatalf("automigrate failed: %v", err)
	}
	svc := &Service{db: db}
	policy := AuthRuntimePolicy{PasswordHistoryLimit: 3}

	const userID = uint64(42)
	oldPassword := "OldPass123"
	oldHash, err := bcrypt.GenerateFromPassword([]byte(oldPassword), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("hash setup failed: %v", err)
	}
	if err := db.Create(&SystemUserPasswordHistory{
		UserID:       userID,
		PasswordHash: string(oldHash),
		ChangedAt:    time.Now(),
	}).Error; err != nil {
		t.Fatalf("seed history failed: %v", err)
	}

	// current hash is some unrelated value so only the history row can match
	currentHash, _ := bcrypt.GenerateFromPassword([]byte("CurrentPass9"), bcrypt.DefaultCost)

	t.Run("password matching a history row is rejected", func(t *testing.T) {
		err := svc.ensurePasswordNotRecentlyUsed(userID, oldPassword, string(currentHash), policy)
		if err == nil || err.Error() != "user.password.error.reused" {
			t.Fatalf("expected reuse rejection from history, got %v", err)
		}
	})

	t.Run("fresh password passes", func(t *testing.T) {
		if err := svc.ensurePasswordNotRecentlyUsed(userID, "BrandNew456", string(currentHash), policy); err != nil {
			t.Fatalf("expected fresh password to pass, got %v", err)
		}
	})
}

// TestCleanupSecurityEvents_OnlyAcknowledged confirms the cleanup deletes only
// acknowledged (resolved) events and preserves pending ones, and that retention
// days outside the configured options are rejected.
func TestCleanupSecurityEvents_OnlyAcknowledged(t *testing.T) {
	db := testmysql.Open(t)
	if err := db.AutoMigrate(&SystemAuthSecurityEvent{}); err != nil {
		t.Fatalf("automigrate failed: %v", err)
	}
	svc := &Service{db: db}
	seedSecurityEvents(t, db)

	t.Run("retention days not in options is rejected", func(t *testing.T) {
		assertUnsupportedSecurityEventRetention(t, svc)
	})

	t.Run("retention cleanup removes only old acknowledged events", func(t *testing.T) {
		assertSecurityEventCleanup(t, svc)
	})
}

func seedSecurityEvents(t *testing.T, db interface{ Create(value any) *gorm.DB }) {
	t.Helper()
	now := time.Now()
	ackedAt := now.AddDate(0, 0, -40)
	seed := []SystemAuthSecurityEvent{
		{EventType: "password_wrong", Severity: "medium", MessageKey: "m", CreatedAt: now.AddDate(0, 0, -40), AcknowledgedAt: &ackedAt},
		{EventType: "password_wrong", Severity: "medium", MessageKey: "m", CreatedAt: now.AddDate(0, 0, -40)},
		{EventType: "password_wrong", Severity: "medium", MessageKey: "m", CreatedAt: now.AddDate(0, 0, -1), AcknowledgedAt: &now},
	}
	for i := range seed {
		if err := db.Create(&seed[i]).Error; err != nil {
			t.Fatalf("seed event %d failed: %v", i, err)
		}
	}
}

func assertUnsupportedSecurityEventRetention(t *testing.T, svc *Service) {
	t.Helper()
	if _, err := svc.CleanupSecurityEvents(3, "", ""); err == nil {
		t.Fatal("expected days_invalid error for unsupported retention")
	}
}

func assertSecurityEventCleanup(t *testing.T, svc *Service) {
	t.Helper()
	cleared, err := svc.CleanupSecurityEvents(30, "", "")
	if err != nil {
		t.Fatalf("cleanup failed: %v", err)
	}
	if cleared != 1 {
		t.Fatalf("expected 1 cleared event, got %d", cleared)
	}
	assertSecurityEventCount(t, svc.db, "", 2)
	assertSecurityEventCount(t, svc.db, "acknowledged_at IS NULL", 1)
}

func assertSecurityEventCount(t *testing.T, db *gorm.DB, condition string, want int64) {
	t.Helper()
	query := db.Model(&SystemAuthSecurityEvent{})
	if condition != "" {
		query = query.Where(condition)
	}
	var got int64
	if err := query.Count(&got).Error; err != nil {
		t.Fatalf("count security events: %v", err)
	}
	if got != want {
		t.Fatalf("expected %d security events for %q, got %d", want, condition, got)
	}
}
