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

// Provisioned tenant-local role keys (contract §4/§7). They live in the
// tenant's own role namespace (system_role.tenant_id = <tenant>), so every
// tenant may own its own copy without colliding with the platform's roles.
//
// These are declared here, in the shared tenant package, so the middleware can
// recognise a tenant administrator without importing the tenant module.
const (
	// TenantRoleAdminKey is the tenant's own administrator: the full console,
	// with every query automatically confined to the tenant by the tenant
	// scope — not by the row-level data-scope policy, which is what the
	// platform `admin` key relaxes.
	TenantRoleAdminKey = "tenant_admin"
	// TenantRoleMemberKey is a read-only tenant member.
	TenantRoleMemberKey = "tenant_member"
)

// TenantScopedRoleKeys is the full set of tenant-local role keys provisioned by
// ProvisionTenant. Membership in this list is what marks a role as "tenant
// scoped": its row-level confinement is owned by the tenant scope (applied
// unconditionally downstream) and its read/write split by Casbin — NOT by the
// platform's dept-based data-scope policy, which is a global (tenant-agnostic)
// mechanism (contract §7 keeps the two role namespaces separate).
//
// DataScopeMiddleware consults this list to exempt tenant-local roles from the
// fail-closed policy lookup. Without the exemption a tenant member — which has
// no system_role_data_scope row — would fall through to custom + empty dept set
// and every request it makes would be rejected with a 500.
//
// A new tenant-local role must be registered here. The tenant module asserts at
// test time that its provisioning set is a subset of this list, so an omission
// fails CI instead of silently 500-ing every request from that role.
var TenantScopedRoleKeys = []string{
	TenantRoleAdminKey,
	TenantRoleMemberKey,
}

// IsTenantScopedRoleKey reports whether roleKey names a tenant-local role.
func IsTenantScopedRoleKey(roleKey string) bool {
	trimmed := strings.TrimSpace(roleKey)
	for _, key := range TenantScopedRoleKeys {
		if trimmed == key {
			return true
		}
	}
	return false
}

// ReadOnlyExportRoute binds a `:export` permission key to the route that
// implements it. Export endpoints are POST requests while a read-only role's
// Casbin surface is the GET wildcard, so the key alone would render an export
// action the middleware refuses (contract §4: the policy, not the key, is the
// authority). Provisioning and the startup repair grant exactly these routes to
// a role that holds the matching key; the workbench uses the same table to
// report a genuine missing policy instead of guessing from the key name.
type ReadOnlyExportRoute struct {
	PermissionKey string
	Path          string
}

// ReadOnlyExportRoutes is the single binding table for export actions. Add a
// route here when an export endpoint is introduced, so the workbench, tenant
// provisioning and the repair stay in sync.
var ReadOnlyExportRoutes = []ReadOnlyExportRoute{
	{PermissionKey: "system:user:export", Path: "/api/v1/system/user/export"},
	{PermissionKey: "system:role:export", Path: "/api/v1/system/role/export"},
	{PermissionKey: "system:permission:export", Path: "/api/v1/system/permission/export"},
	{PermissionKey: "system:dept:export", Path: "/api/v1/system/dept/export"},
	{PermissionKey: "system:dept:export", Path: "/api/v1/system/dept/governance/export"},
	{PermissionKey: "system:post:export", Path: "/api/v1/system/post/export"},
	{PermissionKey: "system:dict:export", Path: "/api/v1/system/dict/type/export"},
	{PermissionKey: "system:dict:export", Path: "/api/v1/system/dict/item/export"},
	{PermissionKey: "system:setting:export", Path: "/api/v1/system/setting/audit/export"},
	{PermissionKey: "system:operation-log:export", Path: "/api/v1/system/operation-log/export"},
	{PermissionKey: "system:login-log:export", Path: "/api/v1/system/login-log/export"},
	{PermissionKey: "system:session:export", Path: "/api/v1/system/session/export"},
	{PermissionKey: "system:security-event:export", Path: "/api/v1/system/security-event/export"},
	{PermissionKey: "system:i18n:export", Path: "/api/v1/system/i18n/export"},
}

// Tenant master status values (contract §2.1).
const (
	TenantStatusActive    = "active"
	TenantStatusSuspended = "suspended"
	TenantStatusArchived  = "archived"
)

// TenantStatusDeleted is the soft-delete marker, not a lifecycle state: it is
// absent from contract §2.1's active/suspended/archived vocabulary on purpose
// and only appears through an explicit status filter. It lives here because two
// modules need the same token for different reasons — the tenant module writes
// it, and the credential lookup has to treat a row that carries it as no longer
// able to host a session.
const TenantStatusDeleted = "deleted"

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

// HasRoleBindingInTenant reports whether the user holds at least one
// system_user_role binding stamped for (user, tenant). W4 (W3 decision A,
// docs/reviews/TENANT_MULTI_MEMBERSHIP_DECISION_2026-10-27.md): a membership
// only resolves when the subject owns a role binding in that tenant, because
// bindings are stamped from the user's owning tenant — a membership elsewhere
// resolves to an empty role set (the TC-01 blank console). The table is
// addressed by name, not by model import: pkg/tenant must not depend on the
// IAM user module, and the binding table has no status column, so existence is
// the predicate. Missing table or query error fails closed (false) — a
// deployment that cannot prove the binding cannot hand out the context either.
func HasRoleBindingInTenant(db *gorm.DB, userID, tenantID uint64) bool {
	if db == nil || userID == 0 || tenantID == 0 {
		return false
	}
	if !db.Migrator().HasTable("system_user_role") {
		return false
	}
	var count int64
	if err := db.Table("system_user_role").
		Where("user_id = ? AND tenant_id = ?", userID, tenantID).
		Count(&count).Error; err != nil {
		return false
	}
	return count > 0
}

// DiscoverDefaultTenant returns the tenant a user logs into under multi mode.
// Contract §3.1: login resolves tenancy from membership — a single active
// membership resolves deterministically; zero memberships means the subject
// logs in without a tenant claim (compat population / platform-only users);
// multiple active memberships require an explicit choice by the caller and
// deny-by-default for now (ambiguous context is never silently narrowed).
// Missing table or DB error also denies without leaking which part failed.
//
// W4 (W3 decision A): memberships without a role binding are filtered out
// before the count logic, so only a membership the subject can actually use
// (at least one system_user_role row) participates in resolution.
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
	usable := make([]Membership, 0, len(memberships))
	for _, m := range memberships {
		if HasRoleBindingInTenant(db, userID, m.TenantID) {
			usable = append(usable, m)
		}
	}
	if len(usable) > 1 {
		return LoginDiscovery{}, ErrTenantForbidden
	}
	if len(usable) == 0 {
		return LoginDiscovery{}, nil
	}
	if err := EnsureTenantLoginable(db, usable[0].TenantID); err != nil {
		return LoginDiscovery{}, err
	}
	return LoginDiscovery{TenantID: usable[0].TenantID, Resolved: true}, nil
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
		// W4 (W3 decision A): the same predicate the discovery and picker paths
		// apply. An explicit choice of a tenant where the subject holds no role
		// binding would stamp an unusable claim — refuse it at the gate so both
		// login paths agree.
		if !HasRoleBindingInTenant(db, in.UserID, in.TenantID) {
			return 0, ErrTenantForbidden
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
// user: active membership ∧ active tenant master ∧ the W4 role-presence
// predicate. The predicate is HasRoleBindingInTenant itself — the same single
// implementation discovery and the switchable list apply — so the three read
// points cannot drift. Missing tables or query errors return an error so
// callers can fail closed.
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
		if !HasRoleBindingInTenant(db, userID, row.TenantID) {
			continue
		}
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
//
// The flag is platform-scoped (contract §6/§7: one mode per deployment), so
// the read is pinned to the platform-global row (tenant_id = 0). Reading
// without that filter is non-deterministic once tenants exist: provisioning
// copies platform settings into every tenant, so an unfiltered lookup could
// resolve the flag from an arbitrary tenant's copy and silently degrade the
// whole deployment to compat.
func FeatureFlagSettingKeyReader(db *gorm.DB) string {
	if db == nil {
		return ModeCompat
	}
	var raw string
	if err := db.Table("system_setting").
		Select("setting_value").
		Where("setting_key = ? AND tenant_id = ?", FeatureFlagSettingKey, PlatformGlobalTenantID).
		Limit(1).
		Pluck("setting_value", &raw).Error; err != nil {
		return ModeCompat
	}
	// Pluck leaves raw empty (and returns no error) when the row is absent.
	if strings.TrimSpace(raw) == "" {
		return ModeCompat
	}
	return NormalizeFlagValue(raw)
}

// NormalizeFlagValue trims a stored flag value and strips a single layer of
// surrounding double quotes. Bootstrap historically persisted the JSON-encoded
// form (`"compat"`) while the settings API persists the bare form (`compat`),
// so both spellings must resolve identically.
func NormalizeFlagValue(raw string) string {
	value := strings.TrimSpace(raw)
	if len(value) >= 2 && value[0] == '"' && value[len(value)-1] == '"' {
		value = strings.TrimSpace(value[1 : len(value)-1])
	}
	return value
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
