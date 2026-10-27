package tenant

import (
	"testing"

	tenantctx "github.com/duanxldragon/pantheon-base/backend/pkg/tenant"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// These tests cover the tenant console navigation grants that provisioning
// seeds for the tenant administrator.
//
// Why this has to exist: role-menu grants are tenant-scoped
// (system_role_menu.tenant_id), and the navigation resolver joins that table by
// role id. A newly provisioned tenant's role ids therefore have no rows at all,
// so before this seeding existed the tenant administrator logged in, the shell
// rendered, and every page was unreachable because the menu tree was empty. The
// tenant smoke suite caught it on main right after the grants became
// tenant-scoped; this test pins the fix at the provisioning boundary instead of
// leaving it to a browser run.

const tenantMenuTestDDL = `
CREATE TABLE IF NOT EXISTS system_menu (
	id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
	parent_id BIGINT UNSIGNED NOT NULL DEFAULT 0,
	title_key VARCHAR(191) NOT NULL DEFAULT '',
	path VARCHAR(191) NOT NULL DEFAULT '',
	component VARCHAR(191) NOT NULL DEFAULT '',
	page_perm VARCHAR(191) NOT NULL DEFAULT '',
	perms VARCHAR(191) NOT NULL DEFAULT '',
	type VARCHAR(8) NOT NULL DEFAULT 'C',
	module VARCHAR(64) NULL,
	sort INT NOT NULL DEFAULT 0,
	is_visible TINYINT NOT NULL DEFAULT 1,
	PRIMARY KEY (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;`

const tenantRoleMenuTestDDL = `
CREATE TABLE IF NOT EXISTS system_role_menu (
	tenant_id BIGINT UNSIGNED NOT NULL DEFAULT 0,
	role_id BIGINT UNSIGNED NOT NULL,
	menu_id BIGINT UNSIGNED NOT NULL,
	PRIMARY KEY (tenant_id, role_id, menu_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;`

// seedMenu inserts one system_menu row and returns its id.
func seedMenu(t *testing.T, db *gorm.DB, menuType, module, path string, visible int) uint64 {
	t.Helper()
	row := struct {
		Type      string
		Module    *string
		Path      string
		IsVisible int
		Sort      int
	}{Type: menuType, Path: path, IsVisible: visible, Sort: 1}
	if module != "" {
		row.Module = &module
	}
	require.NoError(t, db.Table("system_menu").Create(&row).Error)
	var id uint64
	require.NoError(t, db.Table("system_menu").Select("MAX(id)").Scan(&id).Error)
	return id
}

// grantedMenuIDs returns the menu ids granted to roleID inside tenantID.
func grantedMenuIDs(t *testing.T, db *gorm.DB, tenantID, roleID uint64) map[uint64]bool {
	t.Helper()
	var ids []uint64
	require.NoError(t, db.Table("system_role_menu").
		Where("tenant_id = ? AND role_id = ?", tenantID, roleID).
		Pluck("menu_id", &ids).Error)
	granted := make(map[uint64]bool, len(ids))
	for _, id := range ids {
		granted[id] = true
	}
	return granted
}

// A provisioned tenant administrator must receive the tenant console
// navigation, and only the navigation: platform-only pages, hidden pages and
// button-type permission rows stay ungranted.
func TestProvisionTenant_SeedsTenantAdminMenus(t *testing.T) {
	db := setupTestDB(t)
	require.NoError(t, db.Exec(tenantMenuTestDDL).Error)
	require.NoError(t, db.Exec(tenantRoleMenuTestDDL).Error)
	service := NewService(db)

	seedRepairTenant(t, db, 611, "menu-provision", tenantctx.TenantStatusActive)

	visible := seedMenu(t, db, "C", "system.iam", "/system/user", 1)
	hidden := seedMenu(t, db, "C", "system.iam", "/system/user-hidden", 0)
	button := seedMenu(t, db, "F", "system.iam", "", 1)
	platformOnly := seedMenu(t, db, "C", tenantPlatformMenuModule, "/system/tenant", 1)
	menuAuthoring := seedMenu(t, db, "C", "system.iam", "/system/menu", 1)
	i18n := seedMenu(t, db, "C", "system.iam", "/system/i18n", 1)

	result, err := service.ProvisionTenant(ProvisionInput{TenantID: 611})
	require.NoError(t, err)

	adminID := result.RoleIDs[TenantRoleAdminKey]
	require.NotZero(t, adminID)

	granted := grantedMenuIDs(t, db, 611, adminID)
	require.True(t, granted[visible], "tenant admin must see the tenant console navigation")
	require.False(t, granted[hidden], "hidden menus must not be granted")
	require.False(t, granted[button], "button-type rows are permissions, not navigation")
	require.False(t, granted[platformOnly], "platform tenancy menus belong to the platform console")
	require.False(t, granted[menuAuthoring], "menu authoring is platform-capability gated")
	require.False(t, granted[i18n], "i18n management is platform-capability gated")

	memberID := result.RoleIDs[TenantRoleMemberKey]
	require.NotZero(t, memberID)
	require.Empty(t, grantedMenuIDs(t, db, 611, memberID), "provisioning must not grant navigation to members")
}

// Provisioning doubles as a repair entry point, so re-running it must not
// duplicate grants and must backfill a tenant whose grants are missing.
func TestProvisionTenant_SeedsTenantAdminMenusIdempotently(t *testing.T) {
	db := setupTestDB(t)
	require.NoError(t, db.Exec(tenantMenuTestDDL).Error)
	require.NoError(t, db.Exec(tenantRoleMenuTestDDL).Error)
	service := NewService(db)

	seedRepairTenant(t, db, 612, "menu-idempotent", tenantctx.TenantStatusActive)
	visible := seedMenu(t, db, "C", "system.iam", "/system/user", 1)

	first, err := service.ProvisionTenant(ProvisionInput{TenantID: 612})
	require.NoError(t, err)
	// Simulate the broken state this fix repairs: grants lost, roles intact.
	require.NoError(t, db.Table("system_role_menu").Where("tenant_id = ?", 612).Delete(nil).Error)

	second, err := service.ProvisionTenant(ProvisionInput{TenantID: 612})
	require.NoError(t, err)
	require.Equal(t, first.RoleIDs[TenantRoleAdminKey], second.RoleIDs[TenantRoleAdminKey])

	granted := grantedMenuIDs(t, db, 612, second.RoleIDs[TenantRoleAdminKey])
	require.True(t, granted[visible], "re-provisioning must restore the missing grants")

	var count int64
	require.NoError(t, db.Table("system_role_menu").
		Where("tenant_id = ? AND role_id = ? AND menu_id = ?", 612, second.RoleIDs[TenantRoleAdminKey], visible).
		Count(&count).Error)
	require.EqualValues(t, 1, count, "grants must not be duplicated")
}

// A tenant administrator must inherit the platform administrator's *permission
// keys*, not just the navigation rows.
//
// Why this has to exist: the console shell resolves the sidebar from
// `system_role_menu` but the route guard authorises on the key set resolved from
// `system_role_permission`. Provisioning copied only the keys the platform
// administrator actually held, so a menu seeded without its permission half
// (the auth module's login-log / session / security-event pages) produced a
// visible sidebar entry that answered 403. The tenant smoke suite caught it as
// "403 你没有访问当前页面的权限" on /system/login-log.
func TestProvisionTenant_CopiesPlatformAdminPermissionKeys(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(db)

	// Platform administrator: the role whose key set provisioning mirrors.
	require.NoError(t, db.Exec(
		"INSERT INTO system_role (id, tenant_id, role_key, role_name, status) VALUES (1, 0, 'admin', 'admin', 1)",
	).Error)
	platformKeys := []string{
		"system:user:list",
		"system:login-log:list",
		"system:login-log:export",
		"system:security-event:list",
		"system:session:list",
		"system:menu:list",
	}
	// Platform-only keys sit on the platform admin role but every tenant route
	// rejects them with tenant.platformOnly, so they must not be copied.
	platformOnlyKeys := []string{
		"system:tenant:list",
		"system:tenant:create",
		"system:i18n:list",
		"system:menu:create",
	}
	for _, key := range append(append([]string{}, platformKeys...), platformOnlyKeys...) {
		require.NoError(t, db.Exec(
			"INSERT INTO system_role_permission (tenant_id, role_id, permission_key) VALUES (0, 1, ?)", key,
		).Error)
	}

	// The platform administrator's menu grants are copied through a second path
	// (readPlatformAdminBindings), so pin that path too.
	normalMenu := seedMenu(t, db, "C", "system.iam", "/system/login-log", 1)
	platformTenantMenu := seedMenu(t, db, "C", tenantPlatformMenuModule, "/system/tenant", 1)
	platformMenuAuthoring := seedMenu(t, db, "C", "system.iam", "/system/menu", 1)
	for _, menuID := range []uint64{normalMenu, platformTenantMenu, platformMenuAuthoring} {
		require.NoError(t, db.Exec(
			"INSERT INTO system_role_menu (tenant_id, role_id, menu_id) VALUES (0, 1, ?)", menuID,
		).Error)
	}

	seedRepairTenant(t, db, 613, "permission-copy", tenantctx.TenantStatusActive)

	result, err := service.ProvisionTenant(ProvisionInput{TenantID: 613})
	require.NoError(t, err)
	adminID := result.RoleIDs[TenantRoleAdminKey]
	require.NotZero(t, adminID)

	require.ElementsMatch(t, platformKeys, tenantPermissionKeys(t, db, 613, adminID),
		"the tenant administrator must hold every assignable platform administrator key")

	granted := grantedMenuIDs(t, db, 613, adminID)
	require.True(t, granted[normalMenu], "assignable platform menus must be copied")
	require.False(t, granted[platformTenantMenu], "platform-only menus must not be copied")
	require.False(t, granted[platformMenuAuthoring], "platform-only menus must not be copied")

	// The member role stays read-only: list/detail/view/tree/export only.
	memberID := result.RoleIDs[TenantRoleMemberKey]
	require.NotZero(t, memberID)
	require.ElementsMatch(t, []string{
		"system:user:list",
		"system:login-log:list",
		"system:login-log:export",
		"system:security-event:list",
		"system:session:list",
		"system:menu:list",
	}, tenantPermissionKeys(t, db, 613, memberID))
}

// tenantPermissionKeys returns the permission keys granted to roleID inside
// tenantID, read back through the tenant scope the login loader uses.
func tenantPermissionKeys(t *testing.T, db *gorm.DB, tenantID, roleID uint64) []string {
	t.Helper()
	var keys []string
	require.NoError(t, db.Table("system_role_permission").
		Where("tenant_id = ? AND role_id = ?", tenantID, roleID).
		Order("permission_key asc").
		Pluck("permission_key", &keys).Error)
	return keys
}

// RepairTenantPlatformGrants is the upgrade path for tenants provisioned before
// the binding copy was filtered: it must clear their platform-only grants while
// leaving the platform namespace and the assignable grants untouched.
func TestRepairTenantPlatformGrants(t *testing.T) {
	db := setupTestDB(t)
	require.NoError(t, db.Exec(tenantRoleMenuTestDDL).Error)
	service := NewService(db)

	seedRepairTenant(t, db, 614, "repair-platform-grants", tenantctx.TenantStatusActive)
	require.NoError(t, db.Exec(
		"INSERT INTO system_role (id, tenant_id, role_key, role_name, status) VALUES (1, 0, 'admin', 'admin', 1), (50, 614, 'tenant_admin', 'tenant_admin', 1)",
	).Error)

	normalMenu := seedMenu(t, db, "C", "system.iam", "/system/user", 1)
	platformTenantMenu := seedMenu(t, db, "C", tenantPlatformMenuModule, "/system/tenant", 1)
	platformMenuAuthoring := seedMenu(t, db, "C", "system.iam", "/system/menu", 1)
	platformButton := seedMenu(t, db, "F", "system.iam", "", 1)
	require.NoError(t, db.Exec("UPDATE system_menu SET parent_id = ? WHERE id = ?", platformTenantMenu, platformButton).Error)

	for _, menuID := range []uint64{normalMenu, platformTenantMenu, platformMenuAuthoring, platformButton} {
		require.NoError(t, db.Exec(
			"INSERT INTO system_role_menu (tenant_id, role_id, menu_id) VALUES (614, 50, ?)", menuID,
		).Error)
		require.NoError(t, db.Exec(
			"INSERT INTO system_role_menu (tenant_id, role_id, menu_id) VALUES (0, 1, ?)", menuID,
		).Error)
	}
	tenantKeys := []string{
		"system:user:list",
		"system:login-log:list",
		"system:menu:list",
		"system:tenant:list",
		"system:i18n:list",
		"system:menu:create",
	}
	for _, key := range tenantKeys {
		require.NoError(t, db.Exec(
			"INSERT INTO system_role_permission (tenant_id, role_id, permission_key) VALUES (614, 50, ?)", key,
		).Error)
	}
	for _, key := range []string{"system:tenant:list", "system:menu:create"} {
		require.NoError(t, db.Exec(
			"INSERT INTO system_role_permission (tenant_id, role_id, permission_key) VALUES (0, 1, ?)", key,
		).Error)
	}

	require.NoError(t, service.RepairTenantPlatformGrants())

	require.Equal(t, map[uint64]bool{normalMenu: true}, grantedMenuIDs(t, db, 614, 50),
		"tenant roles keep assignable menus only")
	require.ElementsMatch(t, []string{
		"system:login-log:list",
		"system:menu:list",
		"system:user:list",
	}, tenantPermissionKeys(t, db, 614, 50), "tenant roles keep assignable permission keys only")

	require.Equal(t, map[uint64]bool{
		normalMenu:            true,
		platformTenantMenu:    true,
		platformMenuAuthoring: true,
		platformButton:        true,
	}, grantedMenuIDs(t, db, 0, 1), "the platform namespace is never repaired away")
	require.ElementsMatch(t, []string{"system:menu:create", "system:tenant:list"},
		tenantPermissionKeys(t, db, 0, 1))
}
