package role

import (
	"testing"

	"github.com/duanxldragon/pantheon-base/backend/pkg/database"

	"gorm.io/gorm"
)

func seedManagedPermissionMenu(t *testing.T, db *gorm.DB, id uint64, permissionKey string) {
	t.Helper()
	if err := db.Exec(
		"INSERT INTO system_menu (id, page_perm, perms, type) VALUES (?, ?, '', 'C')",
		id, permissionKey,
	).Error; err != nil {
		t.Fatalf("seed managed permission menu %q: %v", permissionKey, err)
	}
}

func countSubjectPolicies(t *testing.T, db *gorm.DB, subject string) int64 {
	t.Helper()
	var count int64
	if err := db.Model(&database.CasbinRule{}).
		Where("ptype = ? AND v0 = ?", "p", subject).Count(&count).Error; err != nil {
		t.Fatalf("count policies for %q: %v", subject, err)
	}
	return count
}

func hasPolicy(t *testing.T, db *gorm.DB, subject, path, method string) bool {
	t.Helper()
	var count int64
	if err := db.Model(&database.CasbinRule{}).
		Where("ptype = ? AND v0 = ? AND v1 = ? AND v2 = ?", "p", subject, path, method).
		Count(&count).Error; err != nil {
		t.Fatalf("lookup policy %s %s: %v", method, path, err)
	}
	return count > 0
}

func TestRoleService_CreateRoleSyncsKnownPermissionAPIPolicies(t *testing.T) {
	db := setupRoleTestDB(t)
	seedManagedPermissionMenu(t, db, 10, "system:user:list")
	seedManagedPermissionMenu(t, db, 11, "system:user:create")
	svc := NewRoleService(db)

	created, err := svc.CreateRole(&RoleCreateReq{
		RoleName:       "Synced",
		RoleKey:        "synced_role",
		Status:         1,
		PermissionKeys: []string{"system:user:list"},
	})
	if err != nil {
		t.Fatalf("create role: %v", err)
	}
	if !hasPolicy(t, db, "synced_role", "/api/v1/system/user/list", "GET") {
		t.Fatalf("role save did not create the API policy for its permission key")
	}
	if got := countSubjectPolicies(t, db, "synced_role"); got != 1 {
		t.Fatalf("subject policy count = %d, want 1", got)
	}

	if _, err := svc.UpdateRole(created.ID, &RoleUpdateReq{
		RoleName:       "Synced",
		RoleKey:        "synced_role",
		Status:         1,
		PermissionKeys: []string{"system:user:create"},
	}); err != nil {
		t.Fatalf("update role: %v", err)
	}
	if hasPolicy(t, db, "synced_role", "/api/v1/system/user/list", "GET") {
		t.Fatalf("removed permission key left its API policy behind")
	}
	if !hasPolicy(t, db, "synced_role", "/api/v1/system/user", "POST") {
		t.Fatalf("added permission key did not create its API policy")
	}

	if _, err := svc.UpdateRole(created.ID, &RoleUpdateReq{
		RoleName: "Synced",
		RoleKey:  "synced_role",
		Status:   1,
	}); err != nil {
		t.Fatalf("clear role permissions: %v", err)
	}
	if got := countSubjectPolicies(t, db, "synced_role"); got != 0 {
		t.Fatalf("clearing permission keys left %d managed policies behind", got)
	}
}

func TestRoleService_RolePolicySyncPreservesManualPolicies(t *testing.T) {
	db := setupRoleTestDB(t)
	seedManagedPermissionMenu(t, db, 10, "system:user:list")
	svc := NewRoleService(db)

	created, err := svc.CreateRole(&RoleCreateReq{
		RoleName:       "Mixed",
		RoleKey:        "mixed_role",
		Status:         1,
		PermissionKeys: []string{"system:user:list"},
	})
	if err != nil {
		t.Fatalf("create role: %v", err)
	}
	if err := db.Create(&database.CasbinRule{
		PType: "p",
		V0:    "mixed_role",
		V1:    "/api/v1/custom/manual",
		V2:    "GET",
	}).Error; err != nil {
		t.Fatalf("seed manual policy: %v", err)
	}

	if _, err := svc.UpdateRole(created.ID, &RoleUpdateReq{
		RoleName: "Mixed",
		RoleKey:  "mixed_role",
		Status:   1,
	}); err != nil {
		t.Fatalf("update role: %v", err)
	}
	if hasPolicy(t, db, "mixed_role", "/api/v1/system/user/list", "GET") {
		t.Fatalf("managed policy survived the permission removal")
	}
	if !hasPolicy(t, db, "mixed_role", "/api/v1/custom/manual", "GET") {
		t.Fatalf("role save removed a manual policy outside the managed route table")
	}
}

func TestRoleTenantScope_RolePolicySyncUsesTenantSubject(t *testing.T) {
	db := setupRoleTestDB(t)
	seedManagedPermissionMenu(t, db, 10, "system:user:list")
	svc := tenantScopedRoleService(db, 101)

	if _, err := svc.CreateRole(&RoleCreateReq{
		RoleName:       "Tenant Synced",
		RoleKey:        "tenant_synced",
		Status:         1,
		PermissionKeys: []string{"system:user:list"},
	}); err != nil {
		t.Fatalf("create tenant role: %v", err)
	}

	if !hasPolicy(t, db, "role:tenant_synced@tenant:101", "/api/v1/system/user/list", "GET") {
		t.Fatalf("tenant role save wrote the API policy outside its domain subject")
	}
	if hasPolicy(t, db, "tenant_synced", "/api/v1/system/user/list", "GET") {
		t.Fatalf("tenant role save leaked a platform-global policy")
	}
}
