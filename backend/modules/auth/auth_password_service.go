package auth

import (
	"errors"
	"strings"
	"time"

	user "pantheon-ops/backend/modules/system/iam/user"
	"pantheon-ops/backend/pkg/common"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type authPasswordService struct {
	auth *AuthService
}

func newAuthPasswordService(auth *AuthService) *authPasswordService {
	return &authPasswordService{auth: auth}
}

func (s *authPasswordService) VerifyPasswordForOperation(userID uint64, sessionID, password string) (string, error) {
	if s.auth.db == nil {
		return "", common.ErrDatabaseNotInitialized
	}
	if strings.TrimSpace(sessionID) == "" {
		return "", errors.New("auth.operation.verification_mismatch")
	}

	var currentUser user.SystemUser
	if err := s.auth.db.First(&currentUser, userID).Error; err != nil {
		return "", err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(currentUser.Password), []byte(password)); err != nil {
		return "", errors.New("auth.password.verify_failed")
	}

	token, err := common.GenerateOperationToken(userID, sessionID, "secure_action", 5*time.Minute)
	if err != nil {
		return "", err
	}
	return token, nil
}

func (s *authPasswordService) UpdatePassword(userID uint64, currentSessionID string, req *PasswordUpdateReq) error {
	if s.auth.db == nil {
		return common.ErrDatabaseNotInitialized
	}

	oldPassword := strings.TrimSpace(req.OldPassword)
	newPassword := strings.TrimSpace(req.NewPassword)
	policy := s.auth.getAuthRuntimePolicy()
	if len(newPassword) < policy.PasswordMinLength {
		return errors.New("user.update.error.password_too_short")
	}
	if !passwordMatchesComplexity(newPassword, policy) {
		return errors.New("user.update.error.password_weak")
	}

	var currentUser user.SystemUser
	if err := s.auth.db.First(&currentUser, userID).Error; err != nil {
		return err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(currentUser.Password), []byte(oldPassword)); err != nil {
		return errors.New("user.password.error.old_password_invalid")
	}
	if oldPassword == newPassword {
		return errors.New("user.password.error.same_as_old")
	}
	if err := s.ensurePasswordNotRecentlyUsed(currentUser.ID, newPassword, currentUser.Password, policy); err != nil {
		return err
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	return s.persistPasswordUpdate(currentUser, userID, currentSessionID, string(passwordHash), policy.PasswordHistoryLimit > 0)
}

func (s *authPasswordService) ensurePasswordNotRecentlyUsed(userID uint64, newPassword, currentPasswordHash string, policy authRuntimePolicy) error {
	if policy.PasswordHistoryLimit <= 0 {
		return nil
	}
	if bcrypt.CompareHashAndPassword([]byte(currentPasswordHash), []byte(newPassword)) == nil {
		return errors.New("user.password.error.reused")
	}

	var rows []SystemUserPasswordHistory
	if err := s.auth.db.Where(userIDWhereClause, userID).
		Order("changed_at desc, id desc").
		Limit(policy.PasswordHistoryLimit).
		Find(&rows).Error; err != nil {
		return err
	}
	for _, row := range rows {
		if bcrypt.CompareHashAndPassword([]byte(row.PasswordHash), []byte(newPassword)) == nil {
			return errors.New("user.password.error.reused")
		}
	}
	return nil
}

func (s *authPasswordService) passwordExpiresAt(userID uint64, policy authRuntimePolicy) *string {
	if policy.PasswordExpireDays <= 0 {
		return nil
	}
	changedAt := s.passwordLastChangedAt(userID)
	if changedAt.IsZero() {
		return nil
	}
	expiresAt := changedAt.AddDate(0, 0, policy.PasswordExpireDays).Format(time.RFC3339)
	return &expiresAt
}

func (s *authPasswordService) isPasswordExpired(userID uint64, policy authRuntimePolicy, now time.Time) bool {
	expiresAt := s.passwordExpiresAt(userID, policy)
	if expiresAt == nil {
		return false
	}
	parsed, err := time.Parse(time.RFC3339, *expiresAt)
	if err != nil {
		return false
	}
	return !parsed.After(now)
}

func (s *authPasswordService) passwordLastChangedAt(userID uint64) time.Time {
	var row SystemUserPasswordHistory
	if err := s.auth.db.Where(userIDWhereClause, userID).Order("changed_at desc, id desc").First(&row).Error; err == nil {
		return row.ChangedAt
	}
	var currentUser user.SystemUser
	if err := s.auth.db.First(&currentUser, userID).Error; err == nil {
		if !currentUser.UpdatedAt.IsZero() {
			return currentUser.UpdatedAt
		}
		return currentUser.CreatedAt
	}
	return time.Time{}
}

func (s *authPasswordService) persistPasswordUpdate(currentUser user.SystemUser, userID uint64, currentSessionID, passwordHash string, keepHistory bool) error {
	return s.auth.db.Transaction(func(tx *gorm.DB) error {
		if keepHistory {
			if err := tx.Create(&SystemUserPasswordHistory{
				UserID:       currentUser.ID,
				PasswordHash: currentUser.Password,
				ChangedAt:    time.Now(),
			}).Error; err != nil {
				return err
			}
		}
		if err := tx.Model(&currentUser).Update("password", passwordHash).Error; err != nil {
			return err
		}
		return revokeOtherUserSessions(tx, userID, currentSessionID)
	})
}
