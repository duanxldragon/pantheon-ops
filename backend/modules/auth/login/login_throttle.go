package login

import "time"

type SystemLoginThrottle struct {
	ID              uint64 `gorm:"primaryKey;autoIncrement"`
	SourceKey       string `gorm:"size:191;uniqueIndex:idx_system_login_throttle_source_key;not null"`
	FailureCount    int    `gorm:"default:0"`
	WindowStartedAt *time.Time
	LastAttemptAt   *time.Time
	BlockedUntil    *time.Time `gorm:"index"`
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

func (SystemLoginThrottle) TableName() string {
	return "system_login_throttle"
}
