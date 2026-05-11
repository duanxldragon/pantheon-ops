package auth

import "time"

type SystemAuthSecurityEvent struct {
	ID         uint64    `gorm:"primaryKey;autoIncrement"`
	UserID     uint64    `gorm:"index"`
	Username   string    `gorm:"size:64;index"`
	EventType  string    `gorm:"size:64;not null;index"`
	Severity   string    `gorm:"size:16;not null;index"`
	SourceKey  string    `gorm:"size:191;index"`
	IP         string    `gorm:"size:128"`
	UserAgent  string    `gorm:"size:255"`
	MessageKey string    `gorm:"size:128;not null"`
	Metadata   string    `gorm:"type:text"`
	CreatedAt  time.Time `gorm:"index"`
}

func (SystemAuthSecurityEvent) TableName() string {
	return "system_auth_security_event"
}

type SystemUserPasswordHistory struct {
	ID           uint64    `gorm:"primaryKey;autoIncrement"`
	UserID       uint64    `gorm:"not null;index:idx_user_password_history_user_changed,priority:1"`
	PasswordHash string    `gorm:"size:255;not null"`
	ChangedAt    time.Time `gorm:"index:idx_user_password_history_user_changed,priority:2"`
	CreatedAt    time.Time
}

func (SystemUserPasswordHistory) TableName() string {
	return "system_user_password_history"
}
