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
	// TenantID is the subject's owning tenant (0 = platform population).
	//
	// Login needs it to tell a platform subject apart from a tenant-scoped one.
	// A tenant-scoped subject that resolves to no tenant claim in multi mode has
	// no usable session — every tenant-scoped route fails closed on it — so it
	// must be refused rather than issued a claim-less session.
	TenantID uint64
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
	//
	// Because usernames are unique only per tenant (contract §2.2), the same
	// username can exist in several tenants and the implementation must pick a
	// deterministic winner: the platform population (TenantID 0) first, then the
	// lowest id. A subject whose tenant has been soft-deleted is not a login
	// candidate and must be skipped — it can never host a session (its tenant is
	// gone, so no claim resolves), and letting it win the tie-break would shadow
	// a live tenant's same-named user and hand the caller a subject that fails
	// every subsequent request.
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

// TenantUserFinder is the optional extension of Repository for tenant-aware
// credential lookup.
//
// Under multi mode a username is unique only WITHIN a tenant (contract §2.2:
// the unique key is (tenant_id, username)), so a login that names a tenant must
// resolve the subject inside that tenant — otherwise `admin` in tenant 22 would
// resolve to the platform `admin` and report a password mismatch. The login
// service asserts this interface and falls back to the plain global lookup only
// when no tenant was named (compat mode, or the tenant-selection flow).
//
// The name follows the Go convention for a single-method interface (the method
// noun plus -er). TenantScopedRepository is kept below as a deprecated alias so
// downstream repositories that already name this contract keep compiling.
type TenantUserFinder interface {
	// FindByUsernameInTenant loads the credential record for a login subject
	// whose username is unique only within tenantID.
	FindByUsernameInTenant(ctx context.Context, tenantID uint64, username string) (*User, error)
}

// TenantScopedRepository is the former name of TenantUserFinder.
//
// Deprecated: use TenantUserFinder. The alias is an identical type, so
// satisfying one satisfies the other; it will be removed in the next major
// release.
type TenantScopedRepository = TenantUserFinder
