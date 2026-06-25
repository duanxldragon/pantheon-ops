package iam

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"pantheon-ops/backend/pkg/common"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

const deletedUsernamePrefix = "__deleted_user_"

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
			Where("id = ?", row.ID).
			Update("preference_json", normalized).Error; err != nil {
			return err
		}
	}

	return nil
}

func (s *UserService) ensureAdminUserSeed() error {
	var count int64
	if err := s.db.Model(&SystemUser{}).Where("id = ?", 1).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	initialPassword, err := resolveInitialAdminPassword()
	if err != nil {
		return err
	}
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(initialPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	admin := SystemUser{
		Username: "admin",
		Password: string(passwordHash),
		Nickname: "Administrator",
		Status:   1,
	}
	admin.ID = 1
	return s.db.Create(&admin).Error
}

func resolveInitialAdminPassword() (string, error) {
	password := strings.TrimSpace(os.Getenv("PANTHEON_INITIAL_ADMIN_PASSWORD"))
	if !common.IsProductionEnv() {
		if password != "" {
			return password, nil
		}
		return defaultDevInitialAdminPassword, nil
	}
	if password == "" {
		return "", errors.New("admin.initial_password_required")
	}
	if len(password) < productionInitialAdminMinLength {
		return "", errors.New("admin.initial_password_too_short")
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

	var count int64
	if err := s.db.Table("system_user_role").Where("user_id = ? AND role_id = ?", 1, adminRoleID).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	return s.db.Exec("INSERT INTO system_user_role (user_id, role_id) VALUES (?, ?)", 1, adminRoleID).Error
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
				Where("id = ?", user.ID).
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

	return "", errors.New("user.delete.error.archive_username_conflict")
}
