package iam

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
	commonsecurity "github.com/duanxldragon/pantheon-base/backend/pkg/common/security"
	"github.com/duanxldragon/pantheon-base/backend/pkg/rbacbind"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

const deletedUsernamePrefix = "__deleted_user_"

const condIDEquals = "id = ?"

const (
	defaultConfiguredPasswordMinLength = 6
	defaultDevInitialAdminPassword     = "123456"
	productionInitialAdminMinLength    = 12
)

func (s *UserService) normalizeUserPreferenceJSON() error {
	if s.db == nil {
		return common.ErrDatabaseNotInitialized
	}

	type userPreferenceRow struct {
		ID             uint64 `gorm:"column:id"`
		PreferenceJSON string `gorm:"column:preference_json"`
	}

	var rows []userPreferenceRow
	if err := s.db.Unscoped().
		Model(&SystemUser{}).
		Select("id", "preference_json").
		Where("preference_json <> ''").
		Find(&rows).Error; err != nil {
		return err
	}

	for _, row := range rows {
		normalized, err := MarshalUserPlatformPreferences(ParseUserPlatformPreferences(row.PreferenceJSON))
		if err != nil {
			return err
		}
		if normalized == row.PreferenceJSON {
			continue
		}
		if err := s.db.Unscoped().
			Model(&SystemUser{}).
			Where(condIDEquals, row.ID).
			Update("preference_json", normalized).Error; err != nil {
			return err
		}
	}

	return nil
}

func (s *UserService) ensureAdminUserSeed() error {
	var count int64
	if err := s.db.Model(&SystemUser{}).Where(condIDEquals, 1).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	initialPassword, err := resolveInitialAdminPassword()
	if err != nil {
		return err
	}
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(initialPassword), commonsecurity.GetBcryptCost())
	if err != nil {
		return err
	}
	admin := SystemUser{
		Username: "admin",
		Password: string(passwordHash),
		Nickname: "Administrator",
		Status:   common.StatusEnabled,
	}
	admin.ID = 1
	return s.db.Create(&admin).Error
}

func resolveInitialAdminPassword() (string, error) {
	password := strings.TrimSpace(os.Getenv("PANTHEON_INITIAL_ADMIN_PASSWORD"))
	if !commonsecurity.IsProductionEnv() {
		if password != "" {
			return password, nil
		}
		return defaultDevInitialAdminPassword, nil
	}
	if password == "" {
		return "", common.NewBadRequest("admin.initial_password_required")
	}
	if len(password) < productionInitialAdminMinLength {
		return "", common.NewBadRequest("admin.initial_password_too_short")
	}
	return password, nil
}

func (s *UserService) ensureAdminRoleBinding() error {
	if !s.db.Migrator().HasTable("system_user_role") || !s.db.Migrator().HasTable("system_role") {
		return nil
	}

	var adminRoleID uint64
	if err := s.db.Table("system_role").Select("id").Where("role_key = ?", "admin").Limit(1).Pluck("id", &adminRoleID).Error; err != nil {
		return err
	}
	if adminRoleID == 0 {
		return nil
	}

	return rbacbind.EnsureUserRole(s.db, 1, adminRoleID)
}

func (s *UserService) releaseDeletedUsernames() error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		var deletedUsers []SystemUser
		if err := tx.Unscoped().
			Where("deleted_at IS NOT NULL").
			Find(&deletedUsers).Error; err != nil {
			return err
		}

		for _, user := range deletedUsers {
			if strings.HasPrefix(user.Username, deletedUsernamePrefix) {
				continue
			}
			deletedUsername, err := s.allocateDeletedUsername(tx, user.ID)
			if err != nil {
				return err
			}
			if err := tx.Unscoped().
				Model(&SystemUser{}).
				Where(condIDEquals, user.ID).
				Update("username", deletedUsername).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *UserService) allocateDeletedUsername(tx *gorm.DB, userID uint64) (string, error) {
	for attempt := 0; attempt < 5; attempt++ {
		candidate := fmt.Sprintf("%s%d", deletedUsernamePrefix, userID)
		if attempt > 0 {
			candidate = fmt.Sprintf("%s%d_%d", deletedUsernamePrefix, userID, time.Now().UnixNano())
		}

		var count int64
		if err := tx.Unscoped().
			Model(&SystemUser{}).
			Where("username = ? AND id <> ?", candidate, userID).
			Count(&count).Error; err != nil {
			return "", err
		}
		if count == 0 {
			return candidate, nil
		}
	}

	return "", common.NewConflict("user.delete.error.archive_username_conflict")
}
