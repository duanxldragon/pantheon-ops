package dept

import (
	"errors"
	"testing"

	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
	"github.com/duanxldragon/pantheon-base/backend/pkg/tenant"

	"gorm.io/gorm"
)

// TestDeptCompatScopePinsGlobalPopulation pins the compat contract: a resolved
// compat context must restrict the tree to tenant_id=0, so tenant rows copied
// while the deployment ran in multi mode neither show up in the single-tenant
// console nor block platform-level mutations.
func TestDeptCompatScopePinsGlobalPopulation(t *testing.T) {
	db := setupDeptTestDB(t)
	service := NewDeptService(db)
	if err := service.Migrate(); err != nil {
		t.Fatalf("migrate dept service: %v", err)
	}
	migrateDeptFixturePosts(t, db)
	migrateDeptFixtureUsers(t, db)

	foreign := SystemDept{TenantID: 282, IsRoot: 1, DeptName: "tenant-only-root", Status: 1}
	if err := db.Create(&foreign).Error; err != nil {
		t.Fatalf("seed tenant root: %v", err)
	}

	compat := service.WithTenantContext(&tenant.Context{
		TenantID:   tenant.PlatformGlobalTenantID,
		Mode:       tenant.ModeCompat,
		ResolvedBy: "compat-fallback",
	})

	tree, err := compat.GetDeptTree(&DeptListQuery{})
	if err != nil {
		t.Fatalf("compat tree: %v", err)
	}
	for _, node := range tree {
		if node.DeptName == foreign.DeptName {
			t.Fatal("compat tree must not contain tenant rows")
		}
	}

	if err := compat.DeleteDept(foreign.ID); err == nil || (!errors.Is(err, common.ErrNotFound) && !errors.Is(err, gorm.ErrRecordNotFound)) {
		t.Fatalf("compat delete of a tenant row must resolve to not-found, got %v", err)
	}

	var root SystemDept
	if err := db.Where("tenant_id = ? AND is_root = ?", tenant.PlatformGlobalTenantID, 1).First(&root).Error; err != nil {
		t.Fatalf("load platform root: %v", err)
	}
	created, err := compat.CreateDept(&DeptCreateReq{ParentID: root.ID, DeptName: "compat-temp", Status: 1})
	if err != nil {
		t.Fatalf("compat create: %v", err)
	}
	if err := compat.DeleteDept(created.ID); err != nil {
		t.Fatalf("compat delete own dept: %v", err)
	}
}

// TestDeptImportCompatStaysInGlobalTree pins the import path: when another
// tenant owns a root with the same name, a compat import must resolve the path
// against tenant_id=0 and stamp the new department with tenant 0. Before the
// fix the path map was global, so the import resolved to the highest-id tenant
// root and created an orphan row invisible to the single-tenant console.
func TestDeptImportCompatStaysInGlobalTree(t *testing.T) {
	db := setupDeptTestDB(t)
	service := NewDeptService(db)
	if err := service.Migrate(); err != nil {
		t.Fatalf("migrate dept service: %v", err)
	}

	var globalRoot SystemDept
	if err := db.Where("tenant_id = ? AND is_root = ?", tenant.PlatformGlobalTenantID, 1).First(&globalRoot).Error; err != nil {
		t.Fatalf("load platform root: %v", err)
	}
	foreignRoot := SystemDept{TenantID: 282, IsRoot: 1, DeptName: defaultRootDeptName, Status: 1}
	if err := db.Create(&foreignRoot).Error; err != nil {
		t.Fatalf("seed tenant root: %v", err)
	}

	compat := service.WithTenantContext(&tenant.Context{
		TenantID:   tenant.PlatformGlobalTenantID,
		Mode:       tenant.ModeCompat,
		ResolvedBy: "compat-fallback",
	})
	result, err := compat.ImportDepts([][]string{
		{"parentDeptPath", "deptName", "sort", "leader", "phone", "email", "status"},
		{defaultRootDeptName, "compat-import-dept", "10", "", "", "", "1"},
	})
	if err != nil {
		t.Fatalf("compat import: %v", err)
	}
	if !result.Applied || result.Created != 1 || result.Failed != 0 {
		t.Fatalf("unexpected import result: %+v", result)
	}

	var imported SystemDept
	if err := db.Where("dept_name = ?", "compat-import-dept").First(&imported).Error; err != nil {
		t.Fatalf("load imported dept: %v", err)
	}
	if imported.TenantID != tenant.PlatformGlobalTenantID || imported.ParentID != globalRoot.ID {
		t.Fatalf("imported dept must belong to the platform tree: %+v (root=%d)", imported, globalRoot.ID)
	}

	var foreignChildren int64
	if err := db.Model(&SystemDept{}).Where("parent_id = ?", foreignRoot.ID).Count(&foreignChildren).Error; err != nil {
		t.Fatalf("count tenant children: %v", err)
	}
	if foreignChildren != 0 {
		t.Fatalf("compat import must not attach rows to the tenant tree, got %d children", foreignChildren)
	}
}
