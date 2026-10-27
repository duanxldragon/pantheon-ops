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

// IsPlatformPopulationUser reports whether the user belongs to no tenant at
// all — the "platform population / platform-only user" the issuance gate
// resolves to a claim-less session (contract §3.1 source 2). It mirrors
// DiscoverDefaultTenant's zero-membership branch so the request side and the
// login side agree on who is a non-tenant subject.
//
// failClosed is true whenever the answer cannot be established (nil DB, zero
// user id, query error): the caller must then refuse the platform-global
// fallback rather than assume the subject is tenant-less.
func IsPlatformPopulationUser(db *gorm.DB, userID uint64) (tenantLess bool, failClosed bool) {
	if db == nil || userID == 0 {
		return false, true
	}
	var count int64
	if err := db.Model(&Membership{}).
		Where("user_id = ? AND status = ?", userID, MembershipActive).
		Count(&count).Error; err != nil {
		return false, true
	}
	return count == 0, false
}

// ListActiveMemberships returns every active membership a user holds, ordered
// by tenant id. It backs the tenant switcher: the subject may only ever move
// between tenants it is an active member of (contract §2.2/§3.1).
func ListActiveMemberships(db *gorm.DB, userID uint64) ([]Membership, error) {
	if db == nil || userID == 0 {
		return nil, ErrTenantForbidden
	}
	var rows []Membership
	if err := db.Model(&Membership{}).
		Where("user_id = ? AND status = ?", userID, MembershipActive).
		Order("tenant_id asc").
		Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

// ValidateExplicitTenant checks that an explicit tenant override
// (`X-Tenant-Id`) names a real, active tenant (contract §3.1 source 1, §5).
//
// Tenant 0 is the platform-global scope and is always valid: it is a
// first-class scope in both modes (contract §2.3/§4), not a tenant row.
// Anything else must exist and be active. Fail-closed: an unavailable lookup
// denies rather than widening scope.
func ValidateExplicitTenant(db *gorm.DB, tenantID uint64) error {
	if tenantID == PlatformGlobalTenantID {
		return nil
	}
	if db == nil || !db.Migrator().HasTable("tenants") {
		return ErrTenantForbidden
	}
	return EnsureTenantLoginable(db, tenantID)
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
