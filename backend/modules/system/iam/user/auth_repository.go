package iam

import (
	"context"
	"time"

	"github.com/duanxldragon/pantheon-base/backend/pkg/contracts/authuser"

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

// NewCredentialRepository publishes the system user credential port for the
// composition root to inject into the auth module.
func NewCredentialRepository(db *gorm.DB) authuser.Repository {
	return &credentialRepository{db: db}
}

func (r *credentialRepository) FindByUsername(ctx context.Context, username string) (*authuser.User, error) {
	var model SystemUser
	if err := r.db.WithContext(ctx).Where("username = ?", username).First(&model).Error; err != nil {
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
	}
}
