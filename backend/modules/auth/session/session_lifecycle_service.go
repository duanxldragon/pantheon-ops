package session

import (
	"time"

	"gorm.io/gorm"
)

// LifecycleService owns system_user_session lifecycle mutations for auth/session.
type LifecycleService struct {
	db *gorm.DB
}

func NewLifecycleService(db *gorm.DB) *LifecycleService {
	return &LifecycleService{db: db}
}

func (s *LifecycleService) WithDB(db *gorm.DB) *LifecycleService {
	return &LifecycleService{db: db}
}

func (s *LifecycleService) RevokeUserSessions(userID uint64, now time.Time) (int64, error) {
	if s == nil || s.db == nil || userID == 0 {
		return 0, nil
	}

	result := s.db.Model(&SystemUserSession{}).
		Where("user_id = ? AND revoked_at IS NULL", userID).
		UpdateColumn("revoked_at", now)
	return result.RowsAffected, result.Error
}

func (s *LifecycleService) DeleteUserSessions(userID uint64) error {
	if s == nil || s.db == nil || userID == 0 {
		return nil
	}
	return s.db.Where("user_id = ?", userID).Delete(&SystemUserSession{}).Error
}
