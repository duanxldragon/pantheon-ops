package tenant

import (
	"strconv"
	"strings"
	"testing"

	"github.com/duanxldragon/pantheon-base/backend/pkg/database"
	tenantctx "github.com/duanxldragon/pantheon-base/backend/pkg/tenant"

	"github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/model"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// These tests cover Service.EnsureTenantRolePolicies, the startup repair for a
// tenant whose Casbin domain policies went missing.
//
// Why the repair has to exist: ProvisionTenant writes `role:<key>@tenant:<id>`
// policies, but nothing re-derives them. `casbin_rule` has no foreign key to
// `tenants`, and the documented ops path for hard-deleting a tenant purges
// those rows by hand, so one over-broad purge leaves a tenant whose subjects
// authenticate but hold no permissions — the console signs in and then answers
// `403 permission.denied` on every page. That is the state tenant 253 was found
// in: roles 576/577 present, 97 permission keys bound, and zero Casbin rules.

const casbinModelForRepair = `
[request_definition]
r = sub, obj, act
[policy_definition]
p = sub, obj, act
[role_definition]
g = _, _
[policy_effect]
e = some(where (p.eft == allow))
[matchers]
m = (r.sub == p.sub || g(r.sub, p.sub)) && keyMatch2(r.obj, p.obj) && r.act == p.act
`

// installRepairEnforcer swaps database.Enforcer for an in-memory enforcer (no
// adapter: these tests assert the rule set, not persistence) and restores the
// previous value afterwards so no other test inherits it.
func installRepairEnforcer(t *testing.T) *casbin.SyncedEnforcer {
	t.Helper()

	previous := database.Enforcer
	t.Cleanup(func() { database.Enforcer = previous })

	m, err := model.NewModelFromString(casbinModelForRepair)
	require.NoError(t, err)
	enforcer, err := casbin.NewSyncedEnforcer(m)
	require.NoError(t, err)

	database.Enforcer = enforcer
	return enforcer
}

// seedRepairTenant inserts one tenant master row with an explicit id.
func seedRepairTenant(t *testing.T, db *gorm.DB, id uint64, code, status string) {
	t.Helper()
	require.NoError(t, db.Create(&Tenant{ID: id, Code: code, Name: code, Status: status}).Error)
}

// policyCountFor counts the enforcer's rules for one tenant domain.
func policyCountFor(t *testing.T, enforcer *casbin.SyncedEnforcer, tenantID uint64) int {
	t.Helper()
	policies, err := enforcer.GetPolicy()
	require.NoError(t, err)

	suffix := "@tenant:" + strconv.FormatUint(tenantID, 10)
	count := 0
	for _, rule := range policies {
		if len(rule) > 0 && strings.HasSuffix(rule[0], suffix) {
			count++
		}
	}
	return count
}

// The repair grants exactly what tenantRoleDefinitions defines, for every
// active tenant.
func TestEnsureTenantRolePolicies_RestoresPoliciesForActiveTenant(t *testing.T) {
	db := setupTestDB(t)
	enforcer := installRepairEnforcer(t)
	service := NewService(db)

	seedRepairTenant(t, db, 501, "repair-active", tenantctx.TenantStatusActive)
	require.Equal(t, 0, policyCountFor(t, enforcer, 501), "fixture must start with no tenant policies")

	require.NoError(t, service.EnsureTenantRolePolicies())

	for _, def := range tenantRoleDefinitions {
		subject := tenantctx.RoleSubject(def.Key, 501)
		for _, method := range def.Methods {
			ok, err := enforcer.HasPolicy(subject, "/api/v1/*", method)
			require.NoError(t, err)
			require.Truef(t, ok, "expected %s %s to be granted after repair", subject, method)
		}
	}
}

// Running the repair twice must not duplicate rules: the bootstrap runs on every
// process start, and AddPolicy is a no-op for a rule that already exists.
func TestEnsureTenantRolePolicies_IsIdempotent(t *testing.T) {
	db := setupTestDB(t)
	enforcer := installRepairEnforcer(t)
	service := NewService(db)

	seedRepairTenant(t, db, 502, "repair-idempotent", tenantctx.TenantStatusActive)

	require.NoError(t, service.EnsureTenantRolePolicies())
	first, err := enforcer.GetPolicy()
	require.NoError(t, err)
	require.NotEmpty(t, first)

	require.NoError(t, service.EnsureTenantRolePolicies())
	second, err := enforcer.GetPolicy()
	require.NoError(t, err)

	require.Len(t, second, len(first), "second repair must not add rules")
}

// Suspension is the supported way to lock a tenant out, so the repair must not
// hand policies to a tenant that is not active. Nothing here removes rules
// either — a suspended tenant simply keeps whatever it already had.
func TestEnsureTenantRolePolicies_SkipsNonActiveTenants(t *testing.T) {
	db := setupTestDB(t)
	enforcer := installRepairEnforcer(t)
	service := NewService(db)

	seedRepairTenant(t, db, 503, "repair-suspended", tenantctx.TenantStatusSuspended)
	seedRepairTenant(t, db, 504, "repair-archived", tenantctx.TenantStatusArchived)
	seedRepairTenant(t, db, 505, "repair-deleted", tenantStatusDeleted)
	seedRepairTenant(t, db, 506, "repair-active", tenantctx.TenantStatusActive)

	require.NoError(t, service.EnsureTenantRolePolicies())

	for _, id := range []uint64{503, 504, 505} {
		require.Equalf(t, 0, policyCountFor(t, enforcer, id), "tenant %d must not receive policies", id)
	}
	require.NotZero(t, policyCountFor(t, enforcer, 506), "the active tenant must be repaired")
}

// The reserved platform/global placeholder (id 0, archived) is not a tenant
// that hosts sessions, so it must never receive domain policies.
func TestEnsureTenantRolePolicies_NeverTouchesTheReservedPlaceholder(t *testing.T) {
	db := setupTestDB(t)
	enforcer := installRepairEnforcer(t)
	service := NewService(db)

	seedReservedPlaceholder(t, db)

	require.NoError(t, service.EnsureTenantRolePolicies())

	require.Equal(t, 0, policyCountFor(t, enforcer, tenantctx.PlatformGlobalTenantID))
}

// The repair is additive: an operator's own extra grant on a tenant role must
// survive, because the repair only restores what the role definitions define.
func TestEnsureTenantRolePolicies_KeepsOperatorPolicies(t *testing.T) {
	db := setupTestDB(t)
	enforcer := installRepairEnforcer(t)
	service := NewService(db)

	seedRepairTenant(t, db, 507, "repair-additive", tenantctx.TenantStatusActive)

	custom := tenantctx.RoleSubject(TenantRoleAdminKey, 507)
	_, err := enforcer.AddPolicy(custom, "/api/v1/system/tenant/*", "GET")
	require.NoError(t, err)

	require.NoError(t, service.EnsureTenantRolePolicies())

	ok, err := enforcer.HasPolicy(custom, "/api/v1/system/tenant/*", "GET")
	require.NoError(t, err)
	require.True(t, ok, "the repair must not remove an existing policy")
}

// A process that never reached InitCasbin (unit fixtures, or an early failure)
// must not make Bootstrap fail: there is nothing to repair against.
func TestEnsureTenantRolePolicies_WithoutEnforcerIsNoop(t *testing.T) {
	db := setupTestDB(t)

	previous := database.Enforcer
	database.Enforcer = nil
	t.Cleanup(func() { database.Enforcer = previous })

	seedRepairTenant(t, db, 508, "repair-no-enforcer", tenantctx.TenantStatusActive)

	require.NoError(t, NewService(db).EnsureTenantRolePolicies())
}
