package dept

import (
	"reflect"
	"testing"
)

func TestDeptService_BootstrapPreservesTenantTrees(t *testing.T) {
	for _, withPlatformRoot := range []bool{false, true} {
		name := "only-tenant-roots"
		if withPlatformRoot {
			name = "existing-platform-root"
		}
		t.Run(name, func(t *testing.T) {
			runBootstrapPreservesTenantTrees(t, withPlatformRoot)
		})
	}
}

func runBootstrapPreservesTenantTrees(t *testing.T, withPlatformRoot bool) {
	t.Helper()
	db := setupDeptTestDB(t)
	if err := db.AutoMigrate(&SystemDept{}); err != nil {
		t.Fatal(err)
	}
	rows := []SystemDept{
		{ID: 10, TenantID: 42, DeptName: "tenant-root", IsRoot: 1, Status: 1},
		{ID: 11, TenantID: 42, ParentID: 10, Ancestors: "10", DeptName: "tenant-child", Status: 1},
		{ID: 12, TenantID: 99, DeptName: "another-tenant-root", IsRoot: 1, Status: 1},
	}
	if withPlatformRoot {
		rows = append(rows, SystemDept{ID: 20, DeptName: "platform-root", IsRoot: 1, Status: 1})
	}
	if err := db.Create(&rows).Error; err != nil {
		t.Fatal(err)
	}
	var before []SystemDept
	if err := db.Where("tenant_id <> ?", 0).Order("id asc").Find(&before).Error; err != nil {
		t.Fatal(err)
	}
	service := NewDeptService(db)
	for run := 0; run < 2; run++ {
		if err := service.Bootstrap(); err != nil {
			t.Fatal(err)
		}
		assertBootstrapTenantTrees(t, service, before)
	}
}

func TestDeptService_BootstrapDoesNotRewriteForeignDescendant(t *testing.T) {
	db := setupDeptTestDB(t)
	if err := db.AutoMigrate(&SystemDept{}); err != nil {
		t.Fatal(err)
	}
	rows := []SystemDept{
		{ID: 1, DeptName: "platform-root", IsRoot: 1, Status: 1},
		{ID: 2, DeptName: "platform-top-level", Status: 1},
		// Legacy/malformed data does not authorize a cross-tenant ancestor update.
		{ID: 3, TenantID: 42, ParentID: 2, Ancestors: "legacy", DeptName: "foreign-descendant", Status: 1},
	}
	if err := db.Create(&rows).Error; err != nil {
		t.Fatal(err)
	}
	var before SystemDept
	if err := db.First(&before, 3).Error; err != nil {
		t.Fatal(err)
	}
	if err := NewDeptService(db).Bootstrap(); err != nil {
		t.Fatal(err)
	}
	var after, platformDept SystemDept
	if err := db.First(&after, 3).Error; err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(before, after) {
		t.Fatalf("Bootstrap rewrote foreign descendant: before=%+v after=%+v", before, after)
	}
	if err := db.First(&platformDept, 2).Error; err != nil {
		t.Fatal(err)
	}
	if platformDept.ParentID != 1 || platformDept.Ancestors != "1" {
		t.Fatalf("platform reparenting must still work: %+v", platformDept)
	}
}

func assertBootstrapTenantTrees(t *testing.T, service *DeptService, before []SystemDept) {
	t.Helper()
	var after []SystemDept
	if err := service.db.Where("tenant_id <> ?", 0).Order("id asc").Find(&after).Error; err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(before, after) {
		t.Fatalf("Bootstrap changed tenant trees: before=%+v after=%+v", before, after)
	}
	var roots int64
	if err := service.db.Model(&SystemDept{}).Where("tenant_id = ? AND is_root = ?", 0, 1).Count(&roots).Error; err != nil {
		t.Fatal(err)
	}
	if roots != 1 {
		t.Fatalf("expected one platform root after Bootstrap, got %d", roots)
	}
}
