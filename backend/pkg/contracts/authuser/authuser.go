// Package authuser defines the credential port the auth domain uses to reach
// system user state.
//
// Contract source: docs/designs/REPOSITORY_LAYOUT.md §8.2 — `auth` may consume
// `system/*` only through a public contract or a composition-root adapter.
// The implementation lives in modules/system/iam/user
// (NewCredentialRepository) and is injected at the composition root
// (backend/cmd/server/main.go), so the auth module never imports
// modules/system/iam/user. scripts/harness/check-boundaries.mjs enforces that.
package authuser

import (
	"context"
	"time"

	"gorm.io/gorm"
)

// User is the auth-facing projection of the system user credential record. It
// deliberately carries only the fields auth flows need; modules/system/iam/user
// stays the owner of the full model and of the system_user table.
type User struct {
	ID                  uint64
	Username            string
	Password            string
	Nickname            string
	Avatar              string
	Email               string
	Phone               string
	Status              int
	PreferenceJSON      string
	FailedLoginAttempts int
	LoginLockedUntil    *time.Time
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

// TxFunc runs inside the same database transaction as a Repository write. tx is
// the caller's transaction handle, so the caller can keep writes to its own
// tables (password history, session revocation) in the same unit of work.
type TxFunc func(tx *gorm.DB) error

// Repository is the read/write port over user credential state.
//
// Implementations must return gorm.ErrRecordNotFound (or an error wrapping it)
// for a missing user so callers keep working with errors.Is.
type Repository interface {
	// FindByUsername loads the credential record for a login subject.
	FindByUsername(ctx context.Context, username string) (*User, error)
	// FindByID loads the credential record for an authenticated subject.
	FindByID(ctx context.Context, userID uint64) (*User, error)
	// ClearFailedLoginState resets the throttle counters after a good password.
	ClearFailedLoginState(ctx context.Context, userID uint64) error
	// UpdateFailedLoginState persists a failed-attempt counter; a nil
	// lockedUntil clears any stored lock.
	UpdateFailedLoginState(ctx context.Context, userID uint64, attempts int, lockedUntil *time.Time) error
	// UpdatePreferenceJSON persists the user's opaque platform preference blob.
	UpdatePreferenceJSON(ctx context.Context, userID uint64, preferenceJSON string) error
	// RotatePassword atomically replaces the stored password hash and runs
	// inTx inside the same transaction.
	RotatePassword(ctx context.Context, userID uint64, newHash string, inTx TxFunc) error
}
