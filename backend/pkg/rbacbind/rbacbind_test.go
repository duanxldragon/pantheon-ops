package rbacbind

import (
	"testing"

	"github.com/duanxldragon/pantheon-base/backend/pkg/testmysql"
)

// systemUserRoleDDL mirrors the real `system_user_role` join table: the tenant
// namespace is part of the key, so a binding written without it is invisible to
// tenant-scoped reads.
const systemUserRoleDDL = "CREATE TABLE system_user_role (" +
	"tenant_id BIGINT UNSIGNED NOT NULL DEFAULT 0, " +
	"user_id BIGINT UNSIGNED NOT NULL, " +
	"role_id BIGINT UNSIGNED NOT NULL, " +
	"PRIMARY KEY (tenant_id, user_id, role_id))"

// systemRoleMenuDDL mirrors the real `system_role_menu` join table after
// 000020: the tenant namespace is part of the primary key, so a grant written
// without it is invisible to the tenant-scoped read path (acceptance F3).
const systemRoleMenuDDL = "CREATE TABLE system_role_menu (" +
	"tenant_id BIGINT UNSIGNED NOT NULL DEFAULT 0, " +
	"role_id BIGINT UNSIGNED NOT NULL, " +
	"menu_id BIGINT UNSIGNED NOT NULL, " +
	"PRIMARY KEY (tenant_id, role_id, menu_id))"

// TestEnsureBindingsAreIdempotent verifies the Ensure* helpers insert once and
// are safe to call repeatedly (the property the copy-pasted call sites relied on).
// Skips automatically when no test DSN is configured.
func TestEnsureBindingsAreIdempotent(t *testing.T) {
	db := testmysql.Open(t)
	if err := db.Exec(systemRoleMenuDDL).Error; err != nil {
		t.Fatalf("create system_role_menu failed: %v", err)
	}
	if err := db.Exec(systemUserRoleDDL).Error; err != nil {
		t.Fatalf("create system_user_role failed: %v", err)
	}

	assertCount := func(table string, want int64) {
		t.Helper()
		var got int64
		if err := db.Table(table).Count(&got).Error; err != nil {
			t.Fatalf("count %s failed: %v", table, err)
		}
		if got != want {
			t.Fatalf("%s row count = %d, want %d", table, got, want)
		}
	}

	for i := 0; i < 3; i++ {
		if err := EnsureRoleMenu(db, 7, 1, 100); err != nil {
			t.Fatalf("EnsureRoleMenu call %d failed: %v", i, err)
		}
		if err := EnsureUserRole(db, 7, 1, 1); err != nil {
			t.Fatalf("EnsureUserRole call %d failed: %v", i, err)
		}
	}
	assertCount("system_role_menu", 1)
	assertCount("system_user_role", 1)

	// A distinct binding is inserted independently.
	if err := EnsureRoleMenu(db, 7, 1, 200); err != nil {
		t.Fatalf("EnsureRoleMenu distinct failed: %v", err)
	}
	assertCount("system_role_menu", 2)

	// The same role/menu pair in another tenant is a distinct grant: the tenant
	// namespace is part of the identity, not decoration.
	if err := EnsureRoleMenu(db, 8, 1, 100); err != nil {
		t.Fatalf("EnsureRoleMenu other tenant failed: %v", err)
	}
	assertCount("system_role_menu", 3)

	var wrongTenant int64
	if err := db.Table("system_role_menu").Where("tenant_id = ?", 0).Count(&wrongTenant).Error; err != nil {
		t.Fatalf("count tenant-0 grants failed: %v", err)
	}
	if wrongTenant != 0 {
		t.Fatalf("system_role_menu rows in tenant 0 = %d, want 0", wrongTenant)
	}

	// The same (user, role) pair in another tenant is a distinct binding: the
	// tenant namespace is part of the identity, not decoration.
	if err := EnsureUserRole(db, 8, 1, 1); err != nil {
		t.Fatalf("EnsureUserRole other tenant failed: %v", err)
	}
	assertCount("system_user_role", 2)
}

func TestInsertBindingsBatch(t *testing.T) {
	db := testmysql.Open(t)
	if err := db.Exec(systemRoleMenuDDL).Error; err != nil {
		t.Fatalf("create system_role_menu failed: %v", err)
	}
	if err := db.Exec(systemUserRoleDDL).Error; err != nil {
		t.Fatalf("create system_user_role failed: %v", err)
	}

	if err := InsertRoleMenus(db, 42, 1, []uint64{10, 20, 30}); err != nil {
		t.Fatalf("InsertRoleMenus failed: %v", err)
	}
	if err := InsertUserRoles(db, 42, []uint64{100, 200, 300}, 1); err != nil {
		t.Fatalf("InsertUserRoles failed: %v", err)
	}

	var roleMenuCount int64
	if err := db.Table("system_role_menu").Where("role_id = ?", 1).Count(&roleMenuCount).Error; err != nil {
		t.Fatalf("count system_role_menu failed: %v", err)
	}
	if roleMenuCount != 3 {
		t.Fatalf("system_role_menu row count = %d, want 3", roleMenuCount)
	}

	var menuWrongTenant int64
	if err := db.Table("system_role_menu").Where("role_id = ? AND tenant_id <> ?", 1, 42).Count(&menuWrongTenant).Error; err != nil {
		t.Fatalf("count mismatched menu tenant failed: %v", err)
	}
	if menuWrongTenant != 0 {
		t.Fatalf("system_role_menu rows with wrong tenant_id = %d, want 0", menuWrongTenant)
	}

	var userRoleCount int64
	if err := db.Table("system_user_role").Where("role_id = ?", 1).Count(&userRoleCount).Error; err != nil {
		t.Fatalf("count system_user_role failed: %v", err)
	}
	if userRoleCount != 3 {
		t.Fatalf("system_user_role row count = %d, want 3", userRoleCount)
	}

	// Every row must carry the requested tenant namespace — a row landing in
	// tenant 0 would be invisible to the tenant-scoped login role loader.
	var wrongTenant int64
	if err := db.Table("system_user_role").
		Where("role_id = ? AND tenant_id <> ?", 1, 42).
		Count(&wrongTenant).Error; err != nil {
		t.Fatalf("count mismatched tenant failed: %v", err)
	}
	if wrongTenant != 0 {
		t.Fatalf("system_user_role rows with wrong tenant_id = %d, want 0", wrongTenant)
	}
}
