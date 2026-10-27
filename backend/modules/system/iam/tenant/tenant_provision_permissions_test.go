package tenant

import (
	"testing"

	tenantctx "github.com/duanxldragon/pantheon-base/backend/pkg/tenant"
	"github.com/stretchr/testify/require"
)

// TestProvisionTenant_PermissionGrantsAreIdempotent pins the grant write that
// used to be a read-then-insert loop. Provisioning is the repair entry point,
// so a second run must not fail on the rows the first one wrote, and the
// composite key (tenant_id, role_id, permission_key) must stay unique.
func TestProvisionTenant_PermissionGrantsAreIdempotent(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(db)

	require.NoError(t, db.Exec(
		"INSERT INTO system_role (id, tenant_id, role_key, role_name, status) VALUES (1, 0, 'admin', 'admin', 1)",
	).Error)
	readOnlyKeys := []string{"system:user:list", "system:login-log:list"}
	for _, key := range readOnlyKeys {
		require.NoError(t, db.Exec(
			"INSERT INTO system_role_permission (tenant_id, role_id, permission_key) VALUES (0, 1, ?)", key,
		).Error)
	}
	seedRepairTenant(t, db, 615, "permission-idempotent", tenantctx.TenantStatusActive)

	first, err := service.ProvisionTenant(ProvisionInput{TenantID: 615})
	require.NoError(t, err)

	countTenantRows := func() int64 {
		t.Helper()
		var count int64
		require.NoError(t, db.Table("system_role_permission").
			Where("tenant_id = ?", 615).
			Count(&count).Error)
		return count
	}
	adminKeys := tenantPermissionKeys(t, db, 615, first.RoleIDs[TenantRoleAdminKey])
	memberKeys := tenantPermissionKeys(t, db, 615, first.RoleIDs[TenantRoleMemberKey])
	require.ElementsMatch(t, readOnlyKeys, adminKeys)
	require.ElementsMatch(t, readOnlyKeys, memberKeys)
	before := countTenantRows()
	require.EqualValues(t, len(readOnlyKeys)*2, before)

	second, err := service.ProvisionTenant(ProvisionInput{TenantID: 615})
	require.NoError(t, err, "re-provisioning must not fail on existing grants")
	require.Equal(t, first.RoleIDs, second.RoleIDs)
	require.Equal(t, before, countTenantRows(), "re-provisioning must not add permission rows")
	require.ElementsMatch(t, adminKeys, tenantPermissionKeys(t, db, 615, second.RoleIDs[TenantRoleAdminKey]))
	require.ElementsMatch(t, memberKeys, tenantPermissionKeys(t, db, 615, second.RoleIDs[TenantRoleMemberKey]))

	// Provisioning also repairs a tenant whose grants were lost.
	require.NoError(t, db.Exec(
		"DELETE FROM system_role_permission WHERE tenant_id = ? AND role_id = ? AND permission_key = ?",
		615, second.RoleIDs[TenantRoleAdminKey], readOnlyKeys[0],
	).Error)
	_, err = service.ProvisionTenant(ProvisionInput{TenantID: 615})
	require.NoError(t, err)
	require.ElementsMatch(t, adminKeys, tenantPermissionKeys(t, db, 615, second.RoleIDs[TenantRoleAdminKey]),
		"re-provisioning must restore the missing grant")
	require.Equal(t, before, countTenantRows())
}
