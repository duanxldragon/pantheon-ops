package rbacbind

import (
	"testing"

	"pantheon-base/pkg/testmysql"
)

// TestEnsureBindingsAreIdempotent verifies the Ensure* helpers insert once and
// are safe to call repeatedly (the property the copy-pasted call sites relied on).
// Skips automatically when no test DSN is configured.
func TestEnsureBindingsAreIdempotent(t *testing.T) {
	db := testmysql.Open(t)
	if err := db.Exec("CREATE TABLE system_role_menu (role_id BIGINT UNSIGNED NOT NULL, menu_id BIGINT UNSIGNED NOT NULL, PRIMARY KEY (role_id, menu_id))").Error; err != nil {
		t.Fatalf("create system_role_menu failed: %v", err)
	}
	if err := db.Exec("CREATE TABLE system_user_role (user_id BIGINT UNSIGNED NOT NULL, role_id BIGINT UNSIGNED NOT NULL, PRIMARY KEY (user_id, role_id))").Error; err != nil {
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
		if err := EnsureRoleMenu(db, 1, 100); err != nil {
			t.Fatalf("EnsureRoleMenu call %d failed: %v", i, err)
		}
		if err := EnsureUserRole(db, 1, 1); err != nil {
			t.Fatalf("EnsureUserRole call %d failed: %v", i, err)
		}
	}
	assertCount("system_role_menu", 1)
	assertCount("system_user_role", 1)

	// A distinct binding is inserted independently.
	if err := EnsureRoleMenu(db, 1, 200); err != nil {
		t.Fatalf("EnsureRoleMenu distinct failed: %v", err)
	}
	assertCount("system_role_menu", 2)
}

func TestInsertBindingsBatch(t *testing.T) {
	db := testmysql.Open(t)
	if err := db.Exec("CREATE TABLE system_role_menu (role_id BIGINT UNSIGNED NOT NULL, menu_id BIGINT UNSIGNED NOT NULL, PRIMARY KEY (role_id, menu_id))").Error; err != nil {
		t.Fatalf("create system_role_menu failed: %v", err)
	}
	if err := db.Exec("CREATE TABLE system_user_role (user_id BIGINT UNSIGNED NOT NULL, role_id BIGINT UNSIGNED NOT NULL, PRIMARY KEY (user_id, role_id))").Error; err != nil {
		t.Fatalf("create system_user_role failed: %v", err)
	}

	if err := InsertRoleMenus(db, 1, []uint64{10, 20, 30}); err != nil {
		t.Fatalf("InsertRoleMenus failed: %v", err)
	}
	if err := InsertUserRoles(db, []uint64{100, 200, 300}, 1); err != nil {
		t.Fatalf("InsertUserRoles failed: %v", err)
	}

	var roleMenuCount int64
	if err := db.Table("system_role_menu").Where("role_id = ?", 1).Count(&roleMenuCount).Error; err != nil {
		t.Fatalf("count system_role_menu failed: %v", err)
	}
	if roleMenuCount != 3 {
		t.Fatalf("system_role_menu row count = %d, want 3", roleMenuCount)
	}

	var userRoleCount int64
	if err := db.Table("system_user_role").Where("role_id = ?", 1).Count(&userRoleCount).Error; err != nil {
		t.Fatalf("count system_user_role failed: %v", err)
	}
	if userRoleCount != 3 {
		t.Fatalf("system_user_role row count = %d, want 3", userRoleCount)
	}
}
