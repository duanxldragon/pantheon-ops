// Package rbacbind centralizes the idempotent RBAC join-table binding SQL
// (system_role_menu, system_user_role) that was previously copy-pasted across
// the auth, iam/role, iam/menu, iam/user, and seed packages. Keeping the raw
// SQL in one place means a schema change to these join tables is a single edit.
package rbacbind

import (
	"strings"

	"gorm.io/gorm"
)

const (
	// insertRoleMenuSQL carries tenant_id explicitly. `system_role_menu` is
	// keyed by (tenant_id, role_id, menu_id) and every read goes through the
	// tenant scope, so a grant written without it lands in tenant 0 and is
	// invisible the moment it is saved (acceptance F3).
	insertRoleMenuSQL = "INSERT INTO system_role_menu (tenant_id, role_id, menu_id) VALUES (?, ?, ?)"
	// insertRolePermissionSQL carries tenant_id explicitly for the same reason:
	// the login permission loader filters on
	// `system_role_permission.tenant_id = <login tenant>`, so a grant written
	// without it lands in tenant 0 and a tenant subject never sees the key.
	insertRolePermissionSQL = "INSERT INTO system_role_permission (tenant_id, role_id, permission_key) VALUES (?, ?, ?)"
	// insertUserRoleSQL carries tenant_id explicitly. `system_user_role` is
	// keyed by (tenant_id, user_id, role_id): the tenant namespace is what the
	// login role loader filters on (system_user_role.tenant_id = <login
	// tenant>), so a binding written without it lands in tenant 0 and is
	// invisible to every tenant-scoped read — the user then logs in with no
	// roles and is denied everywhere.
	insertUserRoleSQL = "INSERT INTO system_user_role (tenant_id, user_id, role_id) VALUES (?, ?, ?)"
	defaultBatchSize  = 100
)

type roleMenuBinding struct {
	TenantID uint64 `gorm:"column:tenant_id"`
	RoleID   uint64 `gorm:"column:role_id"`
	MenuID   uint64 `gorm:"column:menu_id"`
}

type userRoleBinding struct {
	TenantID uint64 `gorm:"column:tenant_id"`
	UserID   uint64 `gorm:"column:user_id"`
	RoleID   uint64 `gorm:"column:role_id"`
}

// InsertRoleMenu inserts a role→menu binding unconditionally into the tenant
// namespace that owns the role (0 = platform/global). Callers that have already
// cleared existing rows (e.g. replaceRoleMenus) use this directly.
func InsertRoleMenu(db *gorm.DB, tenantID, roleID, menuID uint64) error {
	return db.Exec(insertRoleMenuSQL, tenantID, roleID, menuID).Error
}

// InsertUserRole inserts a user→role binding unconditionally into the given
// tenant namespace (0 = platform/global).
func InsertUserRole(db *gorm.DB, tenantID, userID, roleID uint64) error {
	return db.Exec(insertUserRoleSQL, tenantID, userID, roleID).Error
}

// InsertRoleMenus inserts role→menu bindings in batches, all inside the tenant
// namespace that owns the role. Callers must preserve the same uniqueness
// semantics they need before invoking this helper.
func InsertRoleMenus(db *gorm.DB, tenantID, roleID uint64, menuIDs []uint64) error {
	if len(menuIDs) == 0 {
		return nil
	}
	rows := make([]roleMenuBinding, 0, len(menuIDs))
	for _, menuID := range menuIDs {
		rows = append(rows, roleMenuBinding{TenantID: tenantID, RoleID: roleID, MenuID: menuID})
	}
	return db.Table("system_role_menu").CreateInBatches(rows, defaultBatchSize).Error
}

// InsertUserRoles inserts user→role bindings in batches, all inside one tenant
// namespace. Callers must preserve the same uniqueness semantics they need
// before invoking this helper.
func InsertUserRoles(db *gorm.DB, tenantID uint64, userIDs []uint64, roleID uint64) error {
	if len(userIDs) == 0 {
		return nil
	}
	rows := make([]userRoleBinding, 0, len(userIDs))
	for _, userID := range userIDs {
		rows = append(rows, userRoleBinding{TenantID: tenantID, UserID: userID, RoleID: roleID})
	}
	return db.Table("system_user_role").CreateInBatches(rows, defaultBatchSize).Error
}

// EnsureRoleMenu binds a menu to a role only when the binding does not already
// exist in the given tenant namespace, making repeated bootstrap/seed runs
// idempotent.
func EnsureRoleMenu(db *gorm.DB, tenantID, roleID, menuID uint64) error {
	var count int64
	if err := db.Table("system_role_menu").
		Where("tenant_id = ? AND role_id = ? AND menu_id = ?", tenantID, roleID, menuID).
		Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	return InsertRoleMenu(db, tenantID, roleID, menuID)
}

// EnsureUserRole binds a role to a user in the given tenant namespace only when
// the binding does not already exist, making repeated bootstrap/seed runs
// idempotent.
func EnsureUserRole(db *gorm.DB, tenantID, userID, roleID uint64) error {
	var count int64
	if err := db.Table("system_user_role").
		Where("tenant_id = ? AND user_id = ? AND role_id = ?", tenantID, userID, roleID).
		Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	return InsertUserRole(db, tenantID, userID, roleID)
}

// EnsureRolePermission grants one permission key to a role in the given tenant
// namespace, skipping the write when the grant already exists.
//
// This is the permission half of a menu grant and must be written next to
// EnsureRoleMenu, not instead of it: the navigation tree is resolved from
// `system_role_menu`, while the console's route guard authorises on the key set
// resolved from `system_role_permission`. Granting only the menu renders a
// sidebar entry that answers 403 when it is opened.
func EnsureRolePermission(db *gorm.DB, tenantID, roleID uint64, permissionKey string) error {
	permissionKey = strings.TrimSpace(permissionKey)
	if permissionKey == "" {
		return nil
	}
	var count int64
	if err := db.Table("system_role_permission").
		Where("tenant_id = ? AND role_id = ? AND permission_key = ?", tenantID, roleID, permissionKey).
		Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	return db.Exec(insertRolePermissionSQL, tenantID, roleID, permissionKey).Error
}
