package iam

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/duanxldragon/pantheon-base/backend/pkg/common"

	"gorm.io/gorm"
)

func seedCredentialUser(t *testing.T, db *gorm.DB, username string) SystemUser {
	t.Helper()
	user := SystemUser{
		Username:       username,
		Password:       "old-hash",
		Nickname:       "Credential Subject",
		Email:          username + "@example.com",
		Phone:          "13800000000",
		PreferenceJSON: `{"theme":"light"}`,
		Status:         common.StatusEnabled,
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
	return user
}

func TestCredentialRepositoryFindByUsernameProjectsSystemUser(t *testing.T) {
	db := setupUserTestDB(t)
	repo := NewCredentialRepository(db)
	seeded := seedCredentialUser(t, db, "credential_lookup_user")

	got, err := repo.FindByUsername(context.Background(), seeded.Username)
	if err != nil {
		t.Fatalf("find by username: %v", err)
	}
	if got.ID != seeded.ID || got.Username != seeded.Username || got.Password != "old-hash" {
		t.Fatalf("unexpected projection: %+v", got)
	}
	if got.Status != common.StatusEnabled || got.PreferenceJSON != `{"theme":"light"}` {
		t.Fatalf("projection dropped credential/state fields: %+v", got)
	}
}

func TestCredentialRepositoryFindByUsernameMissingUserIsRecordNotFound(t *testing.T) {
	db := setupUserTestDB(t)
	repo := NewCredentialRepository(db)

	_, err := repo.FindByUsername(context.Background(), "credential_missing_user")
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("expected gorm.ErrRecordNotFound so callers keep using errors.Is, got %v", err)
	}
}

func TestCredentialRepositoryClearFailedLoginStateResetsLockAndCounter(t *testing.T) {
	db := setupUserTestDB(t)
	repo := NewCredentialRepository(db)
	seeded := seedCredentialUser(t, db, "credential_clear_user")

	lockedUntil := time.Now().Add(10 * time.Minute)
	if err := db.Model(&SystemUser{}).Where("id = ?", seeded.ID).Updates(map[string]any{
		"failed_login_attempts": 4,
		"login_locked_until":    &lockedUntil,
	}).Error; err != nil {
		t.Fatalf("seed failure state: %v", err)
	}

	if err := repo.ClearFailedLoginState(context.Background(), seeded.ID); err != nil {
		t.Fatalf("clear failed login state: %v", err)
	}

	var reloaded SystemUser
	if err := db.First(&reloaded, seeded.ID).Error; err != nil {
		t.Fatalf("reload user: %v", err)
	}
	if reloaded.FailedLoginAttempts != 0 || reloaded.LoginLockedUntil != nil {
		t.Fatalf("expected cleared throttle state, got attempts=%d locked=%v",
			reloaded.FailedLoginAttempts, reloaded.LoginLockedUntil)
	}
}

func TestCredentialRepositoryUpdateFailedLoginStateWritesAndClearsLock(t *testing.T) {
	db := setupUserTestDB(t)
	repo := NewCredentialRepository(db)
	seeded := seedCredentialUser(t, db, "credential_throttle_user")

	lockUntil := time.Now().Add(15 * time.Minute).Truncate(time.Second)
	if err := repo.UpdateFailedLoginState(context.Background(), seeded.ID, 0, &lockUntil); err != nil {
		t.Fatalf("lock account: %v", err)
	}
	var locked SystemUser
	if err := db.First(&locked, seeded.ID).Error; err != nil {
		t.Fatalf("reload locked user: %v", err)
	}
	if locked.FailedLoginAttempts != 0 || locked.LoginLockedUntil == nil {
		t.Fatalf("expected stored lock, got attempts=%d locked=%v",
			locked.FailedLoginAttempts, locked.LoginLockedUntil)
	}

	// nil lockedUntil must clear the stored lock (expired-lock path).
	if err := repo.UpdateFailedLoginState(context.Background(), seeded.ID, 3, nil); err != nil {
		t.Fatalf("clear lock: %v", err)
	}
	var cleared SystemUser
	if err := db.First(&cleared, seeded.ID).Error; err != nil {
		t.Fatalf("reload cleared user: %v", err)
	}
	if cleared.FailedLoginAttempts != 3 || cleared.LoginLockedUntil != nil {
		t.Fatalf("expected attempt counter only, got attempts=%d locked=%v",
			cleared.FailedLoginAttempts, cleared.LoginLockedUntil)
	}
}

func TestCredentialRepositoryUpdatePreferenceJSONPersists(t *testing.T) {
	db := setupUserTestDB(t)
	repo := NewCredentialRepository(db)
	seeded := seedCredentialUser(t, db, "credential_preference_user")

	if err := repo.UpdatePreferenceJSON(context.Background(), seeded.ID, `{"theme":"dark"}`); err != nil {
		t.Fatalf("update preference: %v", err)
	}

	var reloaded SystemUser
	if err := db.First(&reloaded, seeded.ID).Error; err != nil {
		t.Fatalf("reload user: %v", err)
	}
	if reloaded.PreferenceJSON != `{"theme":"dark"}` {
		t.Fatalf("expected persisted preference, got %q", reloaded.PreferenceJSON)
	}
}

func TestCredentialRepositoryRotatePasswordRunsInTxAndCommits(t *testing.T) {
	db := setupUserTestDB(t)
	repo := NewCredentialRepository(db)
	seeded := seedCredentialUser(t, db, "credential_rotate_user")

	inTxRan := false
	err := repo.RotatePassword(context.Background(), seeded.ID, "new-hash", func(tx *gorm.DB) error {
		inTxRan = true
		// A write to a different table must land in the same transaction.
		return tx.Exec("INSERT INTO system_user_role (user_id, role_id) VALUES (?, ?)", seeded.ID, 42).Error
	})
	if err != nil {
		t.Fatalf("rotate password: %v", err)
	}
	if !inTxRan {
		t.Fatal("expected the caller callback to run inside the rotation transaction")
	}

	var reloaded SystemUser
	if err := db.First(&reloaded, seeded.ID).Error; err != nil {
		t.Fatalf("reload user: %v", err)
	}
	if reloaded.Password != "new-hash" {
		t.Fatalf("expected rotated hash, got %q", reloaded.Password)
	}
	var roleRows int64
	if err := db.Table("system_user_role").Where("user_id = ? AND role_id = ?", seeded.ID, 42).Count(&roleRows).Error; err != nil {
		t.Fatalf("count in-tx rows: %v", err)
	}
	if roleRows != 1 {
		t.Fatalf("expected in-tx write to commit, got %d rows", roleRows)
	}
}

func TestCredentialRepositoryRotatePasswordRollsBackOnCallbackFailure(t *testing.T) {
	db := setupUserTestDB(t)
	repo := NewCredentialRepository(db)
	seeded := seedCredentialUser(t, db, "credential_rollback_user")

	boom := errors.New("password history write failed")
	err := repo.RotatePassword(context.Background(), seeded.ID, "rolled-back-hash", func(tx *gorm.DB) error {
		if execErr := tx.Exec("INSERT INTO system_user_role (user_id, role_id) VALUES (?, ?)", seeded.ID, 43).Error; execErr != nil {
			return execErr
		}
		return boom
	})
	if !errors.Is(err, boom) {
		t.Fatalf("expected callback error to surface, got %v", err)
	}

	var reloaded SystemUser
	if err := db.First(&reloaded, seeded.ID).Error; err != nil {
		t.Fatalf("reload user: %v", err)
	}
	if reloaded.Password != "old-hash" {
		t.Fatalf("expected password hash to be rolled back, got %q", reloaded.Password)
	}
	var roleRows int64
	if err := db.Table("system_user_role").Where("user_id = ? AND role_id = ?", seeded.ID, 43).Count(&roleRows).Error; err != nil {
		t.Fatalf("count in-tx rows: %v", err)
	}
	if roleRows != 0 {
		t.Fatalf("expected in-tx write to roll back, got %d rows", roleRows)
	}
}

// The port is what auth depends on; the compile-time implementation assertion
// lives next to the adapter (auth_repository.go). This pins the factory side of
// the same contract: the composition root must receive the concrete adapter.
func TestCredentialRepositoryImplementsAuthUserPort(t *testing.T) {
	t.Helper()
	repo := NewCredentialRepository(nil)
	if _, ok := repo.(*credentialRepository); !ok {
		t.Fatalf("factory must return the concrete credentialRepository port, got %T", repo)
	}
}
