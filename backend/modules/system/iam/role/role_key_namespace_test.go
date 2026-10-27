package role

import (
	"testing"

	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
	tenantctx "github.com/duanxldragon/pantheon-base/backend/pkg/tenant"
)

// FIX-5 (acceptance B6): the role-key namespace mutex must work in both
// directions. Casbin resolves the GLOBAL subject first (contract §4), so a
// platform role keyed like a live tenant role would silently grant that
// tenant's users platform policies — the mirror image of the tenant→platform
// escape already closed by the forward check.

func TestRoleKeyBlockingNamespace(t *testing.T) {
	cases := []struct {
		ownerID      uint64
		wantOperator string
		wantValue    uint64
	}{
		// A platform operator is blocked by any tenant-owned role.
		{tenantctx.PlatformGlobalTenantID, "<>", tenantctx.PlatformGlobalTenantID},
		// A tenant operator is blocked by platform roles only.
		{101, "=", tenantctx.PlatformGlobalTenantID},
		{0, "<>", tenantctx.PlatformGlobalTenantID},
	}
	for _, tc := range cases {
		operator, value := roleKeyBlockingNamespace(tc.ownerID)
		if operator != tc.wantOperator || value != tc.wantValue {
			t.Fatalf("roleKeyBlockingNamespace(%d) = (%q, %d), want (%q, %d)",
				tc.ownerID, operator, value, tc.wantOperator, tc.wantValue)
		}
	}
}

func TestEnsureRoleKeyAllowedEarlyReturns(t *testing.T) {
	// The reserved-name and blank-key guards run before any query, so a
	// nil-db service is enough to pin their behaviour.
	tenantSvc := NewRoleService(nil).WithTenantContext(
		&tenantctx.Context{TenantID: 101, Mode: tenantctx.ModeMulti})
	if err := tenantSvc.ensureRoleKeyAllowed(" platform_ops ", 0); err == nil ||
		common.ErrMessage(err) != "role.key.reserved" {
		t.Fatalf("tenant operator taking platform_ops: err = %v, want role.key.reserved", err)
	}
	if err := tenantSvc.ensureRoleKeyAllowed("  ", 0); err != nil {
		t.Fatalf("blank role key must be left to the validator: %v", err)
	}
	platformSvc := NewRoleService(nil)
	if err := platformSvc.ensureRoleKeyAllowed("", 0); err != nil {
		t.Fatalf("blank key for platform operator: %v", err)
	}
}

func TestEnsureRoleKeyAllowed_PlatformBlockedByTenantKeys(t *testing.T) {
	db := setupRoleTestDB(t)

	if err := db.Create(&SystemRole{ID: 2, TenantID: 101, RoleName: "tenant ops", RoleKey: "custom_ops", Status: 1}).Error; err != nil {
		t.Fatalf("seed tenant role: %v", err)
	}
	if err := db.Create(&SystemRole{ID: 1, TenantID: 0, RoleName: "admin", RoleKey: "admin", Status: 1}).Error; err != nil {
		t.Fatalf("seed platform admin: %v", err)
	}

	platform := NewRoleService(db)
	if err := platform.ensureRoleKeyAllowed("custom_ops", 0); err == nil ||
		common.ErrMessage(err) != "role.key.reserved" {
		t.Fatalf("platform key colliding with live tenant role: err = %v, want role.key.reserved", err)
	}
	if err := platform.ensureRoleKeyAllowed("brand_new_platform_key", 0); err != nil {
		t.Fatalf("unclaimed key must stay allowed for the platform: %v", err)
	}

	// The forward direction is unchanged: tenants still cannot take platform keys.
	tenantScoped := tenantScopedRoleService(db, 101)
	if err := tenantScoped.ensureRoleKeyAllowed("admin", 0); err == nil ||
		common.ErrMessage(err) != "role.key.reserved" {
		t.Fatalf("tenant key colliding with platform role: err = %v, want role.key.reserved", err)
	}

	// A soft-deleted tenant role releases its key: re-use by the platform is
	// legitimate because the login loader only ever binds live roles.
	if err := db.Delete(&SystemRole{}, 2).Error; err != nil {
		t.Fatalf("soft-delete tenant role: %v", err)
	}
	var deleted SystemRole
	if err := db.Unscoped().First(&deleted, 2).Error; err != nil {
		t.Fatalf("tenant role row must still exist unscoped: %v", err)
	}
	if deleted.DeletedAt.Valid == false {
		t.Fatalf("expected soft delete, got a hard-delete row")
	}
	if err := platform.ensureRoleKeyAllowed("custom_ops", 0); err != nil {
		t.Fatalf("soft-deleted role must not block the released key: %v", err)
	}
}

func TestCreateRole_PlatformRejectsTenantOwnedKey(t *testing.T) {
	db := setupRoleTestDB(t)
	if err := db.Create(&SystemRole{ID: 2, TenantID: 101, RoleName: "tenant ops", RoleKey: "custom_ops", Status: 1}).Error; err != nil {
		t.Fatalf("seed tenant role: %v", err)
	}

	// The reverse check must be wired into the real create path, not just the
	// helper, or the platform handler keeps accepting the shadowing role.
	if _, err := NewRoleService(db).CreateRole(&RoleCreateReq{
		RoleName: "platform shadow", RoleKey: "custom_ops", Status: 1,
	}); err == nil || common.ErrMessage(err) != "role.key.reserved" {
		t.Fatalf("platform CreateRole on tenant-owned key: err = %v, want role.key.reserved", err)
	}

	var count int64
	if err := db.Model(&SystemRole{}).Where(condRoleKeyEquals, "custom_ops").Count(&count).Error; err != nil {
		t.Fatalf("count roles: %v", err)
	}
	if count != 1 {
		t.Fatalf("rejected create still wrote a row: %d custom_ops roles", count)
	}
}
