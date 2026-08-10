package mfa

import "time"

type SystemAuthFactor struct {
	ID              uint64     `gorm:"primaryKey;autoIncrement;column:id"`
	UserID          uint64     `gorm:"uniqueIndex;not null;column:user_id"`
	FactorType      string     `gorm:"size:32;not null;default:totp;column:factor_type"`
	SecretEncrypted string     `gorm:"size:512;not null;column:secret_encrypted"`
	Enabled         int        `gorm:"not null;default:1;column:enabled"`
	ConfirmedAt     *time.Time `gorm:"column:confirmed_at"`
	CreatedAt       time.Time  `gorm:"column:created_at"`
	UpdatedAt       time.Time  `gorm:"column:updated_at"`
}

func (SystemAuthFactor) TableName() string {
	return "system_auth_factor"
}

type SystemAuthMFAChallenge struct {
	ID              uint64     `gorm:"primaryKey;autoIncrement;column:id"`
	ChallengeID     string     `gorm:"uniqueIndex;size:64;not null;column:challenge_id"`
	UserID          uint64     `gorm:"index;not null;column:user_id"`
	Purpose         string     `gorm:"size:32;not null;column:purpose"`
	SecretEncrypted string     `gorm:"size:512;column:secret_encrypted"`
	SetupRequired   int        `gorm:"not null;default:0;column:setup_required"`
	ExpiresAt       time.Time  `gorm:"index;not null;column:expires_at"`
	ConsumedAt      *time.Time `gorm:"column:consumed_at"`
	CreatedAt       time.Time  `gorm:"column:created_at"`
	UpdatedAt       time.Time  `gorm:"column:updated_at"`
}

func (SystemAuthMFAChallenge) TableName() string {
	return "system_auth_mfa_challenge"
}
