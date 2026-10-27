package role

import (
	"testing"

	"github.com/duanxldragon/pantheon-base/backend/pkg/database"
	tenantctx "github.com/duanxldragon/pantheon-base/backend/pkg/tenant"

	"gorm.io/gorm"
)

// A tenant may not shadow the platform role namespace. Role keys are unique
// per tenant, so `admin` was creatable inside a tenant; the login session then
// carried the bare key and Casbin consulted the global `admin` policies.

func tenantScopedRoleService(db *gorm.DB, tenantID uint64) *RoleService {
	return NewRoleService(db).WithTenantContext(&tenantctx.Context{TenantID: tenantID, Mode: tenantctx.ModeMulti})
}

func TestRoleTenantScope_RejectsReservedPlatformKeys(t *testing.T) {
	db := setupRoleTestDB(t)
	if err := db.Create(&SystemRole{ID: 1, TenantID: 0, RoleName: "admin", RoleKey: "admin", Status: 1}).Error; err != nil {
		t.Fatalf("seed platform admin role: %v", err)
	}
	svc := tenantScopedRoleService(db, 101)

	for _, key := range []string{"admin", "platform_ops"} {
		if _, err := svc.CreateRole(&RoleCreateReq{RoleName: "shadow", RoleKey: key, Status: 1}); err == nil {
			t.Fatalf("tenant role key %q was accepted; reserved keys must be rejected", key)
		}
	}
}

func TestRoleTenantScope_DeleteRemovesOnlyOwnDomainPolicies(t *testing.T) {
	db := setupRoleTestDB(t)
	svc := tenantScopedRoleService(db, 101)
	// The first auto-increment id is 1, which is the protected builtin admin
	// id; create a placeholder so the role under test is a normal row.
	if _, err := svc.CreateRole(&RoleCreateReq{RoleName: "placeholder", RoleKey: "placeholder", Status: 1}); err != nil {
		t.Fatalf("create placeholder role: %v", err)
	}

	created, err := svc.CreateRole(&RoleCreateReq{RoleName: "editor", RoleKey: "editor", Status: 1})
	if err != nil {
		t.Fatalf("create role: %v", err)
	}
	if err := db.Create(&[]database.CasbinRule{
		{PType: "p", V0: "role:editor@tenant:101", V1: "/api/v1/system/user/list", V2: "GET"},
		{PType: "p", V0: "admin", V1: "/api/v1/*", V2: "GET"},
	}).Error; err != nil {
		t.Fatalf("seed policies: %v", err)
	}

	if err := svc.DeleteRole(created.ID); err != nil {
		t.Fatalf("delete role: %v", err)
	}

	var domainCount int64
	if err := db.Model(&database.CasbinRule{}).Where("v0 = ?", "role:editor@tenant:101").Count(&domainCount).Error; err != nil {
		t.Fatalf("count domain policies: %v", err)
	}
	if domainCount != 0 {
		t.Fatalf("deleted role left %d domain policies behind", domainCount)
	}
	var platformCount int64
	if err := db.Model(&database.CasbinRule{}).Where("v0 = ?", "admin").Count(&platformCount).Error; err != nil {
		t.Fatalf("count platform policies: %v", err)
	}
	if platformCount != 1 {
		t.Fatalf("deleting a tenant role removed the platform policy (remaining=%d)", platformCount)
	}
}

func TestRoleTenantScope_RenameMigratesPolicySubject(t *testing.T) {
	db := setupRoleTestDB(t)
	svc := tenantScopedRoleService(db, 101)

	created, err := svc.CreateRole(&RoleCreateReq{RoleName: "editor", RoleKey: "editor", Status: 1})
	if err != nil {
		t.Fatalf("create role: %v", err)
	}
	if err := db.Create(&database.CasbinRule{
		PType: "p", V0: "role:editor@tenant:101", V1: "/api/v1/custom/editor", V2: "GET",
	}).Error; err != nil {
		t.Fatalf("seed policy: %v", err)
	}

	if _, err := svc.UpdateRole(created.ID, &RoleUpdateReq{
		RoleName: "editor v2",
		RoleKey:  "editor_v2",
		Status:   1,
	}); err != nil {
		t.Fatalf("rename role: %v", err)
	}

	var renamed int64
	if err := db.Model(&database.CasbinRule{}).
		Where("v0 = ?", "role:editor_v2@tenant:101").Count(&renamed).Error; err != nil {
		t.Fatalf("count renamed policies: %v", err)
	}
	if renamed != 1 {
		t.Fatalf("renamed role policies = %d, want 1", renamed)
	}
	var stale int64
	if err := db.Model(&database.CasbinRule{}).
		Where("v0 = ?", "role:editor@tenant:101").Count(&stale).Error; err != nil {
		t.Fatalf("count stale policies: %v", err)
	}
	if stale != 0 {
		t.Fatalf("rename left %d stale policies on the old subject", stale)
	}
}

func TestRoleTenantScope_ImportCreatesTenantOwnedRole(t *testing.T) {
	db := setupRoleTestDB(t)
	svc := tenantScopedRoleService(db, 101)

	result, err := svc.ImportRoles([][]string{
		{"roleName", "roleKey", "sort", "status"},
		{"Imported", "imported_role", "1", "1"},
	})
	if err != nil {
		t.Fatalf("import roles: %v", err)
	}
	if result.Created != 1 {
		t.Fatalf("created = %d, want 1 (%+v)", result.Created, result)
	}
	var role SystemRole
	if err := db.Where("role_key = ?", "imported_role").First(&role).Error; err != nil {
		t.Fatalf("read imported role: %v", err)
	}
	if role.TenantID != 101 {
		t.Fatalf("imported role tenant_id = %d, want 101 (tenant-global leak)", role.TenantID)
	}
}
