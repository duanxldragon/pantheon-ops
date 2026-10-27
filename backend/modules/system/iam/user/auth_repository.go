package user

import (
	"context"
	"time"

	"github.com/duanxldragon/pantheon-base/backend/pkg/contracts/authuser"
	"github.com/duanxldragon/pantheon-base/backend/pkg/tenant"

	"gorm.io/gorm"
)

// credentialRepository is the system module's implementation of the
// authuser.Repository port (pkg/contracts/authuser).
//
// Boundary rule: REPOSITORY_LAYOUT.md §8.2 — auth may consume system/* only
// through a public contract. Publishing the port from the package that owns the
// system_user table keeps SystemUser the single owner of that model while
// letting auth reach credential state without importing this package. The
// composition root (backend/cmd/server/main.go) is the only place allowed to
// wire the two together; scripts/harness/check-boundaries.mjs fails if the auth
// module imports this package directly.
type credentialRepository struct {
	db *gorm.DB
}

var _ authuser.Repository = (*credentialRepository)(nil)
var _ authuser.TenantUserFinder = (*credentialRepository)(nil)

// NewCredentialRepository publishes the system user credential port for the
// composition root to inject into the auth module.
func NewCredentialRepository(db *gorm.DB) authuser.Repository {
	return &credentialRepository{db: db}
}

// FindByUsername loads a login subject by username alone.
//
// Since the tenant-isolation work made usernames unique only per tenant, the
// same username can exist in several tenants, so this global lookup needs a
// deterministic winner. Platform-population subjects (tenant_id = 0) sort
// first: they are the pre-multi-tenant population and the only subjects that
// legitimately have no tenant to name at login.
//
// A tenant-named login resolves through FindByUsernameInTenant first, so this
// method is reached for a named tenant only when that tenant owns no such
// credential row — the picker case, where the choice names a session context
// rather than the subject's home tenant. The issuance gate then decides whether
// the subject may hold a claim for the named tenant, so picking the wrong
// subject here cannot by itself grant access to the named tenant.
//
// Subjects whose tenant has been deleted are excluded outright. Deleting a
// tenant does not remove the `system_user` rows it provisioned, so those rows
// survive as orphans; because the tie-break below is the lowest id, an orphan
// used to shadow a live tenant's user with the same username. Login then
// succeeded as the orphan — whose tenant is gone, so no claim could be resolved
// — and the resulting claim-less session failed every subsequent request with
// `tenant.context.missing`. Such a row cannot host a session at all, so it is
// not a login candidate.
func (r *credentialRepository) FindByUsername(ctx context.Context, username string) (*authuser.User, error) {
	var model SystemUser
	liveTenants := r.db.WithContext(ctx).Table("tenants").
		Select("id").
		Where("status <> ?", tenant.TenantStatusDeleted)
	if err := r.db.WithContext(ctx).
		Where("username = ?", username).
		Where("tenant_id = ? OR tenant_id IN (?)", tenant.PlatformGlobalTenantID, liveTenants).
		Order("CASE WHEN tenant_id = 0 THEN 0 ELSE 1 END, id ASC").
		First(&model).Error; err != nil {
		return nil, err
	}
	return toAuthUser(&model), nil
}

// FindByUsernameInTenant loads a login subject whose username is unique only
// within tenantID (contract §2.2).
func (r *credentialRepository) FindByUsernameInTenant(ctx context.Context, tenantID uint64, username string) (*authuser.User, error) {
	var model SystemUser
	if err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND username = ?", tenantID, username).
		First(&model).Error; err != nil {
		return nil, err
	}
	return toAuthUser(&model), nil
}

func (r *credentialRepository) FindByID(ctx context.Context, userID uint64) (*authuser.User, error) {
	var model SystemUser
	if err := r.db.WithContext(ctx).First(&model, userID).Error; err != nil {
		return nil, err
	}
	return toAuthUser(&model), nil
}

func (r *credentialRepository) ClearFailedLoginState(ctx context.Context, userID uint64) error {
	return r.db.WithContext(ctx).Model(&SystemUser{}).
		Where("id = ? AND (failed_login_attempts <> 0 OR login_locked_until IS NOT NULL)", userID).
		Updates(map[string]any{
			"failed_login_attempts": 0,
			"login_locked_until":    nil,
		}).Error
}

func (r *credentialRepository) UpdateFailedLoginState(ctx context.Context, userID uint64, attempts int, lockedUntil *time.Time) error {
	return r.db.WithContext(ctx).Model(&SystemUser{}).
		Where(condIDEquals, userID).
		Updates(map[string]any{
			"failed_login_attempts": attempts,
			"login_locked_until":    lockedUntil,
		}).Error
}

func (r *credentialRepository) UpdatePreferenceJSON(ctx context.Context, userID uint64, preferenceJSON string) error {
	return r.db.WithContext(ctx).Model(&SystemUser{}).
		Where(condIDEquals, userID).
		Update("preference_json", preferenceJSON).Error
}

func (r *credentialRepository) RotatePassword(ctx context.Context, userID uint64, newHash string, inTx authuser.TxFunc) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&SystemUser{}).Where(condIDEquals, userID).Update("password", newHash).Error; err != nil {
			return err
		}
		if inTx == nil {
			return nil
		}
		return inTx(tx)
	})
}

func toAuthUser(model *SystemUser) *authuser.User {
	return &authuser.User{
		ID:                  model.ID,
		Username:            model.Username,
		Password:            model.Password,
		Nickname:            model.Nickname,
		Avatar:              model.Avatar,
		Email:               model.Email,
		Phone:               model.Phone,
		Status:              model.Status,
		PreferenceJSON:      model.PreferenceJSON,
		FailedLoginAttempts: model.FailedLoginAttempts,
		LoginLockedUntil:    model.LoginLockedUntil,
		CreatedAt:           model.CreatedAt,
		UpdatedAt:           model.UpdatedAt,
		TenantID:            model.TenantID,
	}
}
