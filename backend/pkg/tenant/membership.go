package tenant

import (
	"strings"

	"gorm.io/gorm"
)

// Membership is the read-side projection of `tenant_memberships`
// (contract §2.2). The canary only needs existence + active status; full
// membership management belongs to system/org (a later task). The table is
// created by Migrate so the canary fixture stays self-contained.
type Membership struct {
	ID       uint64 `gorm:"primaryKey;autoIncrement"`
	TenantID uint64 `gorm:"not null;uniqueIndex:uk_tenant_membership,priority:1"`
	UserID   uint64 `gorm:"not null;uniqueIndex:uk_tenant_membership,priority:2"`
	Role     string `gorm:"size:32;not null;default:member"`
	Status   string `gorm:"size:16;not null;default:active"`
}

// TableName pins the projection to the `tenant_memberships` table.
func (Membership) TableName() string { return "tenant_memberships" }

// Membership status values (contract §2.2).
const (
	MembershipActive   = "active"
	MembershipDisabled = "disabled"
)

// HasActiveMembership reports whether the user is an active member of the
// tenant. Missing table or row => false (deny-by-default).
func HasActiveMembership(db *gorm.DB, tenantID, userID uint64) bool {
	if db == nil || tenantID == 0 || userID == 0 {
		return false
	}
	var count int64
	if err := db.Model(&Membership{}).
		Where("tenant_id = ? AND user_id = ? AND status = ?", tenantID, userID, MembershipActive).
		Count(&count).Error; err != nil {
		return false
	}
	return count > 0
}

// SettingFlagReader is the minimal read interface the mode loader binds to
// (satisfied by *config.SettingService.GetByKey and test stubs).
type SettingFlagReader interface {
	GetByKey(settingKey string) (string, error)
}

// FeatureFlagSettingKey is the system/config flag (contract §6):
// "compat" (default) | "multi".
const FeatureFlagSettingKey = "platform.tenant_mode"

// NewSettingModeLoader builds a ModeLoader backed by the settings service.
func NewSettingModeLoader(reader SettingFlagReader, ttlNano int64) *ModeLoader {
	return NewModeLoader(func() string {
		if reader == nil {
			return ModeCompat
		}
		value, err := reader.GetByKey(FeatureFlagSettingKey)
		if err != nil {
			// flag missing => compat (fail-safe, flag-off = today's behavior)
			return ModeCompat
		}
		return value
	}, ttlNano)
}

// NormalizeMode maps any unknown/empty value to compat.
func NormalizeMode(value string) string {
	if strings.TrimSpace(value) == ModeMulti {
		return ModeMulti
	}
	return ModeCompat
}
