// Package rbacbind centralizes the idempotent RBAC join-table binding SQL
// (system_role_menu, system_user_role) that was previously copy-pasted across
// the auth, iam/role, iam/menu, iam/user, and seed packages. Keeping the raw
// SQL in one place means a schema change to these join tables is a single edit.
package rbacbind

import "gorm.io/gorm"

const (
	insertRoleMenuSQL = "INSERT INTO system_role_menu (role_id, menu_id) VALUES (?, ?)"
	insertUserRoleSQL = "INSERT INTO system_user_role (user_id, role_id) VALUES (?, ?)"
	defaultBatchSize  = 100
)

type roleMenuBinding struct {
	RoleID uint64 `gorm:"column:role_id"`
	MenuID uint64 `gorm:"column:menu_id"`
}

type userRoleBinding struct {
	UserID uint64 `gorm:"column:user_id"`
	RoleID uint64 `gorm:"column:role_id"`
}

// InsertRoleMenu inserts a role→menu binding unconditionally. Callers that have
// already cleared existing rows (e.g. replaceRoleMenus) use this directly.
func InsertRoleMenu(db *gorm.DB, roleID, menuID uint64) error {
	return db.Exec(insertRoleMenuSQL, roleID, menuID).Error
}

// InsertUserRole inserts a user→role binding unconditionally.
func InsertUserRole(db *gorm.DB, userID, roleID uint64) error {
	return db.Exec(insertUserRoleSQL, userID, roleID).Error
}

// InsertRoleMenus inserts role→menu bindings in batches. Callers must preserve
// the same uniqueness semantics they need before invoking this helper.
func InsertRoleMenus(db *gorm.DB, roleID uint64, menuIDs []uint64) error {
	if len(menuIDs) == 0 {
		return nil
	}
	rows := make([]roleMenuBinding, 0, len(menuIDs))
	for _, menuID := range menuIDs {
		rows = append(rows, roleMenuBinding{RoleID: roleID, MenuID: menuID})
	}
	return db.Table("system_role_menu").CreateInBatches(rows, defaultBatchSize).Error
}

// InsertUserRoles inserts user→role bindings in batches. Callers must preserve
// the same uniqueness semantics they need before invoking this helper.
func InsertUserRoles(db *gorm.DB, userIDs []uint64, roleID uint64) error {
	if len(userIDs) == 0 {
		return nil
	}
	rows := make([]userRoleBinding, 0, len(userIDs))
	for _, userID := range userIDs {
		rows = append(rows, userRoleBinding{UserID: userID, RoleID: roleID})
	}
	return db.Table("system_user_role").CreateInBatches(rows, defaultBatchSize).Error
}

// EnsureRoleMenu binds a menu to a role only when the binding does not already
// exist, making repeated bootstrap/seed runs idempotent.
func EnsureRoleMenu(db *gorm.DB, roleID, menuID uint64) error {
	var count int64
	if err := db.Table("system_role_menu").
		Where("role_id = ? AND menu_id = ?", roleID, menuID).
		Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	return InsertRoleMenu(db, roleID, menuID)
}

// EnsureUserRole binds a role to a user only when the binding does not already
// exist, making repeated bootstrap/seed runs idempotent.
func EnsureUserRole(db *gorm.DB, userID, roleID uint64) error {
	var count int64
	if err := db.Table("system_user_role").
		Where("user_id = ? AND role_id = ?", userID, roleID).
		Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	return InsertUserRole(db, userID, roleID)
}
