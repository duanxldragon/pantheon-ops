package permission

import (
	"errors"
	"testing"

	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
	"github.com/duanxldragon/pantheon-base/backend/pkg/database"
	tenantctx "github.com/duanxldragon/pantheon-base/backend/pkg/tenant"

	"gorm.io/gorm"
)

// These cases pin the tenant boundary of the policy console. Before the fix the
// handler used the unbound service, the request body chose the namespace, and
// the workbench wrote bare (global) subjects — so a tenant administrator could
// create a global policy for its own role key, write into another tenant's
// domain, and read every tenant's policies.

func seedPolicyIsolationFixture(t *testing.T) (*PermissionService, *gorm.DB) {
	t.Helper()
	db := setupPermissionTestDB(t)
	if err := db.AutoMigrate(&tenantctx.Tenant{}, &PermissionWorkbenchRemediationEvent{}); err != nil {
		t.Fatalf("migrate tenant/remediation tables: %v", err)
	}
	for _, id := range []uint64{101, 202} {
		if err := db.Create(&tenantctx.Tenant{ID: id, Code: "t", Name: "t", Status: tenantctx.TenantStatusActive}).Error; err != nil {
			t.Fatalf("seed tenant %d: %v", id, err)
		}
	}
	roles := []permissionTestRole{
		{ID: 2, TenantID: 101, RoleName: "admin", RoleKey: "tenant_admin", Status: 1},
		{ID: 3, TenantID: 202, RoleName: "editor", RoleKey: "editor", Status: 1},
	}
	if err := db.Create(&roles).Error; err != nil {
		t.Fatalf("seed roles: %v", err)
	}
	if err := db.Create(&database.CasbinRule{
		PType: "p", V0: "role:editor@tenant:202", V1: "/api/v1/system/secret", V2: "GET",
	}).Error; err != nil {
		t.Fatalf("seed other-tenant policy: %v", err)
	}
	svc := NewPermissionService(db).WithTenantContext(&tenantctx.Context{TenantID: 101, Mode: tenantctx.ModeMulti})
	return svc, db
}

func TestPermissionPolicyTenantScope_CannotCreateGlobalSubject(t *testing.T) {
	svc, _ := seedPolicyIsolationFixture(t)

	policy, err := svc.CreatePolicy([]string{"tenant_admin"}, &PermissionPolicyCreateReq{
		RoleKey: "tenant_admin",
		Path:    "/api/v1/system/tenant/*",
		Method:  "GET",
	})
	if err != nil {
		t.Fatalf("tenant create policy: %v", err)
	}
	if policy.RoleKey != "role:tenant_admin@tenant:101" {
		t.Fatalf("stored subject = %q, want the caller's domain", policy.RoleKey)
	}
}

func TestPermissionPolicyTenantScope_RejectsForeignTenantTarget(t *testing.T) {
	svc, _ := seedPolicyIsolationFixture(t)

	_, err := svc.CreatePolicy([]string{"tenant_admin"}, &PermissionPolicyCreateReq{
		RoleKey:  "tenant_admin",
		Path:     "/api/v1/system/user/*",
		Method:   "GET",
		TenantId: 202,
	})
	if !errorsIsForbidden(err) {
		t.Fatalf("cross-tenant target error = %v, want forbidden", err)
	}
}

func TestPermissionPolicyTenantScope_ListAndExportStayInTenant(t *testing.T) {
	svc, _ := seedPolicyIsolationFixture(t)

	page, err := svc.ListPolicies(&PermissionPolicyQuery{Page: 1, PageSize: 100})
	if err != nil {
		t.Fatalf("list policies: %v", err)
	}
	for _, item := range page.Items {
		if item.RoleKey == "role:editor@tenant:202" {
			t.Fatalf("tenant 101 can read tenant 202's policy: %+v", item)
		}
	}

	exported, err := svc.ExportPolicies(&PermissionPolicyQuery{})
	if err != nil {
		t.Fatalf("export policies: %v", err)
	}
	for _, row := range exported.Rows {
		if row[0] == "role:editor@tenant:202" {
			t.Fatalf("tenant 101 can export tenant 202's policy: %+v", row)
		}
	}
}

func TestPermissionPolicyTenantScope_ImportStoresDomainSubject(t *testing.T) {
	svc, db := seedPolicyIsolationFixture(t)

	records := [][]string{
		{"roleKey", "path", "method"},
		{"tenant_admin", "/api/v1/system/tenant/*", "GET"},
	}
	result, err := svc.ImportPolicies([]string{"tenant_admin"}, records)
	if err != nil {
		t.Fatalf("import policies: %v", err)
	}
	if result.Created != 1 {
		t.Fatalf("created = %d, want 1 (%+v)", result.Created, result)
	}
	var row database.CasbinRule
	if err := db.Where("v0 = ?", "role:tenant_admin@tenant:101").First(&row).Error; err != nil {
		t.Fatalf("domain policy missing: %v", err)
	}
	var bareCount int64
	if err := db.Model(&database.CasbinRule{}).Where("v0 = ?", "tenant_admin").Count(&bareCount).Error; err != nil {
		t.Fatalf("count bare policies: %v", err)
	}
	if bareCount != 0 {
		t.Fatalf("import created %d global policies; tenant import must stay in the domain", bareCount)
	}
}

func TestPermissionPolicyTenantScope_WorkbenchRemediationStaysInDomain(t *testing.T) {
	svc, db := seedPolicyIsolationFixture(t)
	if err := db.Create(&permissionTestMenu{
		ID:       10,
		TitleKey: "system.menu.tenant",
		Path:     "/system/tenant",
		PagePerm: "system:tenant:list",
		Type:     "C",
		Module:   "system",
	}).Error; err != nil {
		t.Fatalf("seed menu catalog: %v", err)
	}
	if err := db.Create(&permissionTestRolePermission{
		RoleID:        2,
		PermissionKey: "system:tenant:list",
	}).Error; err != nil {
		t.Fatalf("seed role permission: %v", err)
	}

	resp, err := svc.RemediateWorkbenchPolicies(&PermissionWorkbenchRemediateReq{RoleKey: "tenant_admin"})
	if err != nil {
		t.Fatalf("remediate: %v", err)
	}
	if resp.CreatedCount == 0 {
		t.Fatalf("expected missing policies to be created: %+v", resp)
	}
	var bareCount int64
	if err := db.Model(&database.CasbinRule{}).Where("v0 = ?", "tenant_admin").Count(&bareCount).Error; err != nil {
		t.Fatalf("count bare policies: %v", err)
	}
	if bareCount != 0 {
		t.Fatalf("remediation wrote %d global policies; it must stay in the tenant domain", bareCount)
	}
	var domainCount int64
	if err := db.Model(&database.CasbinRule{}).
		Where("v0 = ? AND v1 = ? AND v2 = ?", "role:tenant_admin@tenant:101", "/api/v1/system/tenant/list", "GET").
		Count(&domainCount).Error; err != nil {
		t.Fatalf("count domain policies: %v", err)
	}
	if domainCount != 1 {
		t.Fatalf("expected the tenant-domain policy for /system/tenant/list, got %d", domainCount)
	}
}

func TestPermissionBootstrapPreservesTenantDomainPolicies(t *testing.T) {
	_, db := seedPolicyIsolationFixture(t)
	if err := db.Create(&[]database.CasbinRule{
		{PType: "p", V0: "role:tenant_admin@tenant:101", V1: "/api/v1/system/user/list", V2: "GET"},
		{PType: "p", V0: "role:gone@tenant:101", V1: "/api/v1/system/user/list", V2: "GET"},
	}).Error; err != nil {
		t.Fatalf("seed policies: %v", err)
	}

	if err := NewPermissionService(db).Bootstrap(); err != nil {
		t.Fatalf("bootstrap: %v", err)
	}

	var live int64
	if err := db.Model(&database.CasbinRule{}).Where("v0 = ?", "role:tenant_admin@tenant:101").Count(&live).Error; err != nil {
		t.Fatalf("count live policy: %v", err)
	}
	if live != 1 {
		t.Fatalf("bootstrap deleted a live tenant role policy (remaining=%d)", live)
	}
	var orphan int64
	if err := db.Model(&database.CasbinRule{}).Where("v0 = ?", "role:gone@tenant:101").Count(&orphan).Error; err != nil {
		t.Fatalf("count orphan policy: %v", err)
	}
	if orphan != 0 {
		t.Fatalf("bootstrap kept %d orphan policies", orphan)
	}
}

func errorsIsForbidden(err error) bool {
	return err != nil && errors.Is(err, common.ErrForbidden)
}
