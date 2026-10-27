package role

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// backfillRoleFixture is the three-role graph both backfill tests need: a
// platform role, a tenant-700 role and a soft-deleted tenant-700 role. The role
// keys are suffixed per test because the unique key is (tenant_id, role_key).
type backfillRoleFixture struct {
	platformRole SystemRole
	tenantRole   SystemRole
	deletedRole  SystemRole
}

func seedBackfillRoles(t *testing.T, db *gorm.DB, suffix string) backfillRoleFixture {
	t.Helper()
	fixture := backfillRoleFixture{
		platformRole: SystemRole{TenantID: 0, RoleName: "Platform", RoleKey: "platform_" + suffix, Status: 1},
		tenantRole:   SystemRole{TenantID: 700, RoleName: "Tenant", RoleKey: "tenant_" + suffix, Status: 1},
		deletedRole: SystemRole{
			TenantID: 700,
			RoleName: "Deleted",
			RoleKey:  "deleted_" + suffix,
			Status:   1,
			DeletedAt: gorm.DeletedAt{
				Time:  time.Now(),
				Valid: true,
			},
		},
	}
	for _, role := range []*SystemRole{&fixture.platformRole, &fixture.tenantRole, &fixture.deletedRole} {
		require.NoError(t, db.Create(role).Error)
	}
	return fixture
}

type rolePermissionRow struct {
	TenantID      uint64 `gorm:"column:tenant_id"`
	RoleID        uint64 `gorm:"column:role_id"`
	PermissionKey string `gorm:"column:permission_key"`
}

func loadRolePermissionRows(t *testing.T, db *gorm.DB) []rolePermissionRow {
	t.Helper()
	var rows []rolePermissionRow
	require.NoError(t, db.Table("system_role_permission").
		Select("tenant_id, role_id, permission_key").
		Order("tenant_id asc, role_id asc, permission_key asc").
		Scan(&rows).Error)
	return rows
}

// TestBackfillRolePermissionsStampsTenantAndDeduplicates pins the bootstrap
// backfill: permission rows are keyed by (tenant_id, role_id, permission_key),
// but the select used to read role menus without joining the owning role, so
// every backfilled row landed in tenant 0. Two menus exposing the same key also
// produced duplicate seeds. The backfill must stamp the role's tenant, keep one
// row per natural key and stay idempotent across restarts.
func TestBackfillRolePermissionsStampsTenantAndDeduplicates(t *testing.T) {
	db := setupRoleTestDB(t)
	service := NewRoleService(db)

	fixture := seedBackfillRoles(t, db, "backfill")

	// Menu 1 and the extra menu 3 expose the same permission key, so the tenant
	// role would seed it twice.
	require.NoError(t, db.Exec(
		"INSERT INTO system_menu (id, page_perm, perms, type) VALUES (3, 'sys:user:list', '', 'F')",
	).Error)
	grants := []SystemRoleMenu{
		{TenantID: 0, RoleID: fixture.platformRole.ID, MenuID: 1},
		{TenantID: 700, RoleID: fixture.tenantRole.ID, MenuID: 1},
		{TenantID: 700, RoleID: fixture.tenantRole.ID, MenuID: 2},
		{TenantID: 700, RoleID: fixture.tenantRole.ID, MenuID: 3},
		{TenantID: 700, RoleID: fixture.deletedRole.ID, MenuID: 1},
	}
	for _, grant := range grants {
		require.NoError(t, db.Create(&grant).Error)
	}

	require.NoError(t, service.backfillRolePermissions())

	rows := loadRolePermissionRows(t, db)
	assert.ElementsMatch(t, []rolePermissionRow{
		{TenantID: 0, RoleID: fixture.platformRole.ID, PermissionKey: "sys:user:list"},
		{TenantID: 700, RoleID: fixture.tenantRole.ID, PermissionKey: "sys:user:create"},
		{TenantID: 700, RoleID: fixture.tenantRole.ID, PermissionKey: "sys:user:list"},
	}, rows, "backfill must stamp the owning tenant, deduplicate keys and skip deleted roles")

	// Re-running bootstrap must not insert a second copy.
	require.NoError(t, service.backfillRolePermissions())
	assert.Equal(t, rows, loadRolePermissionRows(t, db), "backfill must be idempotent")
}

// TestBackfillRolePermissionsRepairsPlatformNamespaceGrants pins the repair for
// grants a pre-fix backfill stamped with tenant_id = 0 even though the role is
// tenant-owned. Two situations must be handled in this order: a stale row whose
// key the role already owns under its real tenant is a duplicate and must be
// deleted (updating it would violate uk_role_permission_tenant), while a stale
// row with no owner must be re-stamped. Platform grants and grants of
// soft-deleted roles are out of scope and must not move.
func TestBackfillRolePermissionsRepairsPlatformNamespaceGrants(t *testing.T) {
	db := setupRoleTestDB(t)
	service := NewRoleService(db)

	fixture := seedBackfillRoles(t, db, "repair")

	grants := []SystemRolePermission{
		{TenantID: 0, RoleID: fixture.tenantRole.ID, PermissionKey: "sys:tenant:list"},
		{TenantID: 0, RoleID: fixture.tenantRole.ID, PermissionKey: "sys:user:list"},
		{TenantID: 700, RoleID: fixture.tenantRole.ID, PermissionKey: "sys:user:list"},
		{TenantID: 0, RoleID: fixture.platformRole.ID, PermissionKey: "sys:tenant:list"},
		{TenantID: 0, RoleID: fixture.deletedRole.ID, PermissionKey: "sys:tenant:list"},
	}
	for i := range grants {
		require.NoError(t, db.Create(&grants[i]).Error)
	}

	require.NoError(t, service.backfillRolePermissions())

	rows := loadRolePermissionRows(t, db)
	assert.ElementsMatch(t, []rolePermissionRow{
		{TenantID: 700, RoleID: fixture.tenantRole.ID, PermissionKey: "sys:tenant:list"},
		{TenantID: 700, RoleID: fixture.tenantRole.ID, PermissionKey: "sys:user:list"},
		{TenantID: 0, RoleID: fixture.platformRole.ID, PermissionKey: "sys:tenant:list"},
		{TenantID: 0, RoleID: fixture.deletedRole.ID, PermissionKey: "sys:tenant:list"},
	}, rows, "stale tenant grants must be re-stamped or deduplicated, platform and deleted-role grants untouched")

	require.NoError(t, service.backfillRolePermissions())
	assert.Equal(t, rows, loadRolePermissionRows(t, db), "the repair must be idempotent")
}
