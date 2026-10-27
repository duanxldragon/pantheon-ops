package tenant

import (
	"testing"

	tenantctx "github.com/duanxldragon/pantheon-base/backend/pkg/tenant"

	"github.com/stretchr/testify/require"
)

// A tenant member holds `:export` permission keys (exports only read), but its
// wildcard grant is GET-only while every export endpoint is POST. Provisioning
// must therefore grant the exact POST routes behind the keys the member
// actually holds, otherwise the console renders an export button the middleware
// refuses.
func TestProvisionTenant_GrantsMemberExportPoliciesForHeldKeys(t *testing.T) {
	db := setupTestDB(t)
	enforcer := installRepairEnforcer(t)
	service := NewService(db)

	// Platform administrator: the source of the member's permission keys.
	require.NoError(t, db.Exec(
		"INSERT INTO system_role (id, tenant_id, role_key, role_name, status) VALUES (1, 0, 'admin', 'admin', 1)",
	).Error)
	require.NoError(t, db.Exec(
		"INSERT INTO system_role_permission (tenant_id, role_id, permission_key) VALUES (0, 1, 'system:user:export'), (0, 1, 'system:user:list')",
	).Error)

	seedRepairTenant(t, db, 615, "member-export", tenantctx.TenantStatusActive)

	_, err := service.ProvisionTenant(ProvisionInput{TenantID: 615})
	require.NoError(t, err)

	subject := tenantctx.RoleSubject(TenantRoleMemberKey, 615)
	ok, err := enforcer.HasPolicy(subject, "/api/v1/system/user/export", "POST")
	require.NoError(t, err)
	require.True(t, ok, "the member holds system:user:export, so the POST route must be granted")

	ok, err = enforcer.HasPolicy(subject, "/api/v1/system/role/export", "POST")
	require.NoError(t, err)
	require.False(t, ok, "a route whose key the member does not hold must stay ungranted")
}

// The startup repair runs for tenants provisioned before the export bindings
// existed. It must converge to the same rule set, and only for keys the member
// holds, because the repair is additive and never widens a role's permissions.
func TestEnsureTenantRolePolicies_RestoresMemberExportPolicies(t *testing.T) {
	db := setupTestDB(t)
	enforcer := installRepairEnforcer(t)
	service := NewService(db)

	seedRepairTenant(t, db, 616, "repair-member-export", tenantctx.TenantStatusActive)
	require.NoError(t, db.Exec(
		"INSERT INTO system_role (tenant_id, role_key, role_name, status) VALUES (?, ?, 'tenant member', 1)",
		616, TenantRoleMemberKey,
	).Error)
	var memberRoleID uint64
	require.NoError(t, db.Table("system_role").
		Where("tenant_id = ? AND role_key = ?", 616, TenantRoleMemberKey).
		Pluck("id", &memberRoleID).Error)
	require.NotZero(t, memberRoleID)
	require.NoError(t, db.Exec(
		"INSERT INTO system_role_permission (tenant_id, role_id, permission_key) VALUES (?, ?, 'system:dict:export')",
		616, memberRoleID,
	).Error)

	require.NoError(t, service.EnsureTenantRolePolicies())

	subject := tenantctx.RoleSubject(TenantRoleMemberKey, 616)
	for _, path := range []string{
		"/api/v1/system/dict/type/export",
		"/api/v1/system/dict/item/export",
	} {
		ok, err := enforcer.HasPolicy(subject, path, "POST")
		require.NoError(t, err)
		require.Truef(t, ok, "system:dict:export must grant %s", path)
	}

	ok, err := enforcer.HasPolicy(subject, "/api/v1/system/user/export", "POST")
	require.NoError(t, err)
	require.False(t, ok, "the repair must not grant an export the member was not given")
}

// Tenants provisioned before the read-only filter existed carry write keys on
// the member role (security-event:clear, session:delete, ...). The console then
// renders actions the GET-only Casbin surface refuses, and the governance board
// reports a gap that can never be closed without breaking read-only intent.
// Re-provisioning must converge the role back to the read-only key set.
func TestProvisionTenant_PrunesLegacyMemberWriteKeys(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(db)

	seedRepairTenant(t, db, 617, "member-legacy", tenantctx.TenantStatusActive)
	first, err := service.ProvisionTenant(ProvisionInput{TenantID: 617})
	require.NoError(t, err)
	memberID := first.RoleIDs[TenantRoleMemberKey]
	require.NotZero(t, memberID)

	require.NoError(t, db.Exec(
		"INSERT INTO system_role_permission (tenant_id, role_id, permission_key) VALUES (?, ?, 'system:security-event:clear'), (?, ?, 'system:security-event:list')",
		617, memberID, 617, memberID,
	).Error)

	_, err = service.ProvisionTenant(ProvisionInput{TenantID: 617})
	require.NoError(t, err)

	keys := tenantPermissionKeys(t, db, 617, memberID)
	require.NotContains(t, keys, "system:security-event:clear",
		"a write key must not survive on the read-only member role")
	require.Contains(t, keys, "system:security-event:list",
		"read-only keys must be preserved")
}
