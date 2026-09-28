package tenant

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/duanxldragon/pantheon-base/backend/pkg/authtoken"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// Membership roles (contract §2.2). Tenant-internal roles are namespaced
// separately from global IAM roles; no implicit mapping (contract §7).
const (
	MembershipRoleOwner  = "owner"
	MembershipRoleAdmin  = "admin"
	MembershipRoleMember = "member"
)

// Tenant master status values (contract §2.1).
const (
	TenantStatusActive    = "active"
	TenantStatusSuspended = "suspended"
	TenantStatusArchived  = "archived"
)

// RoleDomainPrefix decorates a global role key into the tenant-scoped Casbin
// subject `role:<key>@tenant:<id>` (contract §4). Global roles never carry a
// domain; the two namespaces stay separate (contract §7).
const RoleDomainPrefix = "role:"

// RoleSubject builds `role:<key>@tenant:<id>` for tenant-scoped policies.
func RoleSubject(roleKey string, tenantID uint64) string {
	return RoleDomainPrefix + strings.TrimSpace(roleKey) + "@tenant:" + strconv.FormatUint(tenantID, 10)
}

// GlobalRoleSubject is the plain global role key used by existing policies.
func GlobalRoleSubject(roleKey string) string {
	return strings.TrimSpace(roleKey)
}

// Tenant is the read-side projection of the `tenants` master table
// (contract §2.1). The master data is owned by system/org; auth only reads
// status here to decide loginability (contract §7 ownership boundaries).
type Tenant struct {
	ID     uint64 `gorm:"primaryKey"`
	Code   string `gorm:"size:64;not null"`
	Name   string `gorm:"size:128;not null"`
	Status string `gorm:"size:16;not null;default:active"`
}

// TableName pins the read-side projection to the `tenants` master table.
func (Tenant) TableName() string { return "tenants" }

// LoginDiscovery is the result of resolving a user's login tenancy in multi
// mode (contract §3.1 source 2: the subject claim issued at login).
type LoginDiscovery struct {
	TenantID uint64
	Resolved bool
}

// DiscoverDefaultTenant returns the tenant a user logs into under multi mode.
// Contract §3.1: login resolves tenancy from membership — a single active
// membership resolves deterministically; zero memberships means the subject
// logs in without a tenant claim (compat population / platform-only users);
// multiple active memberships require an explicit choice by the caller and
// deny-by-default for now (ambiguous context is never silently narrowed).
// Missing table or DB error also denies without leaking which part failed.
func DiscoverDefaultTenant(db *gorm.DB, userID uint64) (LoginDiscovery, error) {
	if db == nil || userID == 0 {
		return LoginDiscovery{}, nil
	}
	var memberships []Membership
	if err := db.Model(&Membership{}).
		Where("user_id = ? AND status = ?", userID, MembershipActive).
		Order("tenant_id asc").
		Find(&memberships).Error; err != nil {
		return LoginDiscovery{}, ErrTenantContextMissing
	}
	if len(memberships) > 1 {
		return LoginDiscovery{}, ErrTenantForbidden
	}
	if len(memberships) == 0 {
		return LoginDiscovery{}, nil
	}
	if err := EnsureTenantLoginable(db, memberships[0].TenantID); err != nil {
		return LoginDiscovery{}, err
	}
	return LoginDiscovery{TenantID: memberships[0].TenantID, Resolved: true}, nil
}

// IssuanceCheckInput carries everything the session-issuance gate needs.
type IssuanceCheckInput struct {
	Mode     string
	UserID   uint64
	TenantID uint64 // proposed tenant claim (0 = none / compat)
}

// GateSessionIssuance validates that a token pair may carry the proposed
// tenant claim (contract §2.2/§5). Compat mode never stamps a claim.
// Multi mode requires an active membership AND an active tenant master row;
// suspended/archived tenants deny (contract §5).
func GateSessionIssuance(db *gorm.DB, in IssuanceCheckInput) (uint64, error) {
	if NormalizeMode(in.Mode) != ModeMulti {
		return 0, nil
	}
	if in.TenantID == PlatformGlobalTenantID {
		return 0, nil
	}
	if in.TenantID != 0 {
		if !HasActiveMembership(db, in.TenantID, in.UserID) {
			return 0, ErrTenantForbidden
		}
		if err := EnsureTenantLoginable(db, in.TenantID); err != nil {
			return 0, err
		}
		return in.TenantID, nil
	}
	// No proposed claim: keep 0 (compat population / platform-only user).
	return 0, nil
}

// EnsureTenantLoginable denies login/issuance against suspended/archived
// tenants (contract §2.1/§5). Missing tenant row denies as well.
func EnsureTenantLoginable(db *gorm.DB, tenantID uint64) error {
	if db == nil || tenantID == 0 {
		return ErrTenantForbidden
	}
	var row Tenant
	if err := db.Select("id", "status").Where("id = ?", tenantID).First(&row).Error; err != nil {
		return ErrTenantForbidden
	}
	if row.Status != TenantStatusActive {
		if row.Status == TenantStatusSuspended {
			return ErrTenantSuspended
		}
		return ErrTenantArchived
	}
	return nil
}

// RefreshCheckInput carries the refresh-time re-validation inputs.
type RefreshCheckInput struct {
	Mode     string
	UserID   uint64
	TenantID uint64 // claim carried by the session being refreshed
}

// GateSessionRefresh re-validates the tenant claim on refresh (task packet
// risk node: "stale membership"). If the membership was revoked/disabled
// since issuance the refresh is denied — a session cannot outlive its
// membership. Compat mode and claim-less sessions pass unchanged.
func GateSessionRefresh(db *gorm.DB, in RefreshCheckInput) error {
	if NormalizeMode(in.Mode) != ModeMulti {
		return nil
	}
	if in.TenantID == PlatformGlobalTenantID {
		return nil
	}
	if in.TenantID == 0 {
		return nil
	}
	if !HasActiveMembership(db, in.TenantID, in.UserID) {
		return ErrTenantForbidden
	}
	return EnsureTenantLoginable(db, in.TenantID)
}

// RevokeUserSessionsInTenant force-expires every live session a user holds
// after a membership change (implementation note: "Membership changes
// invalidate affected sessions"). The per-user blacklist key kills live
// access tokens for their remaining TTL window; refresh tokens bound to the
// user's open sessions are cascade-deleted; the session rows are marked
// revoked. Returns the number of session rows revoked.
func RevokeUserSessionsInTenant(ctx context.Context, rdb *redis.Client, db *gorm.DB, userID uint64) (int64, error) {
	if db == nil || userID == 0 {
		return 0, nil
	}
	var sessionIDs []string
	if err := db.Table("system_user_session").
		Where("user_id = ? AND revoked_at IS NULL", userID).
		Pluck("session_id", &sessionIDs).Error; err != nil {
		return 0, err
	}
	if err := authtoken.BlacklistUser(ctx, rdb, userID); err != nil {
		return 0, err
	}
	for _, sid := range sessionIDs {
		_ = authtoken.RevokeSessionRefresh(ctx, rdb, sid)
	}
	var now = time.Now()
	result := db.Exec(
		"UPDATE system_user_session SET revoked_at = ? WHERE user_id = ? AND revoked_at IS NULL",
		&now, userID,
	)
	return result.RowsAffected, result.Error
}

// IsTenantGateError reports whether err came from a tenant gate sentinel
// (forbidden/suspended/archived/context-missing). Callers use it to map the
// error message to its i18n key directly (contract §5 error codes).
func IsTenantGateError(err error) bool {
	return errors.Is(err, ErrTenantForbidden) ||
		errors.Is(err, ErrTenantSuspended) ||
		errors.Is(err, ErrTenantArchived) ||
		errors.Is(err, ErrTenantContextMissing)
}

// LoginTenantCandidate is one selectable tenant for the login picker: only
// active memberships in loginable (active) tenants are listed. Master-data
// reads stay in pkg/tenant; the caller never touches the tenants table
// directly (contract §7).
type LoginTenantCandidate struct {
	TenantID uint64 `json:"tenantId"`
	Code     string `json:"code"`
	Name     string `json:"name"`
	Role     string `json:"role"`
}

// ListActiveMembershipsForUser returns the login-able tenant candidates for a
// user: active membership ∧ active tenant master. Missing tables or query
// errors return an error so callers can fail closed.
func ListActiveMembershipsForUser(db *gorm.DB, userID uint64) ([]LoginTenantCandidate, error) {
	if db == nil || userID == 0 {
		return nil, nil
	}
	var rows []struct {
		TenantID uint64
		Code     string
		Name     string
		Role     string
	}
	err := db.Table("tenant_memberships").
		Select("tenant_memberships.tenant_id AS tenant_id, tenants.code AS code, tenants.name AS name, tenant_memberships.role AS role").
		Joins("JOIN tenants ON tenants.id = tenant_memberships.tenant_id").
		Where("tenant_memberships.user_id = ? AND tenant_memberships.status = ? AND tenants.status = ?",
			userID, MembershipActive, TenantStatusActive).
		Order("tenant_memberships.tenant_id asc").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	candidates := make([]LoginTenantCandidate, 0, len(rows))
	for _, row := range rows {
		candidates = append(candidates, LoginTenantCandidate{
			TenantID: row.TenantID,
			Code:     row.Code,
			Name:     row.Name,
			Role:     row.Role,
		})
	}
	return candidates, nil
}

// FeatureFlagSettingKeyReader reads the raw `platform.tenant_mode` value
// directly from system_setting (used by callers that have a raw *gorm.DB but
// no SettingService handle, e.g. the auth runtime). Fail-safe: missing table
// or row => compat.
func FeatureFlagSettingKeyReader(db *gorm.DB) string {
	if db == nil {
		return ModeCompat
	}
	var raw string
	if err := db.Table("system_setting").
		Select("setting_value").
		Where("setting_key = ?", FeatureFlagSettingKey).
		Limit(1).
		Pluck("setting_value", &raw).Error; err != nil {
		return ModeCompat
	}
	return raw
}

// SessionTenantClaim normalizes a session row's tenant claim for re-stamping
// into a rotated token pair (0 = no claim).
func SessionTenantClaim(raw uint64) uint64 {
	if raw == PlatformGlobalTenantID {
		return 0
	}
	return raw
}

// CasbinDomainPolicySubjects expands a role key into the Casbin subjects that
// must be consulted for a request in the given tenant context (contract §4:
// global role first, then domain role; never merged). Compat / global
// contexts consult only the global subject.
func CasbinDomainPolicySubjects(roleKey string, ctx *Context) []string {
	key := strings.TrimSpace(roleKey)
	if ctx == nil || !ctx.IsMulti() || ctx.TenantID == PlatformGlobalTenantID {
		return []string{key}
	}
	return []string{
		GlobalRoleSubject(key),
		RoleSubject(key, ctx.TenantID),
	}
}
