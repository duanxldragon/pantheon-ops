package group

import (
	"testing"

	"pantheon-base/pkg/impexp"
)

func TestGroupServiceExportHonorsRowCap(t *testing.T) {
	db := setupTestDB(t)
	svc := NewGroupService(db)

	oldCap := maxGroupExportRows
	maxGroupExportRows = 2
	t.Cleanup(func() {
		maxGroupExportRows = oldCap
	})

	for _, item := range []Group{
		{Name: "根分组", Conditions: jsonCondition(t, ConditionExpression{Operator: "AND"})},
		{Name: "生产环境", Conditions: jsonCondition(t, ConditionExpression{Operator: "AND"})},
		{Name: "测试环境", Conditions: jsonCondition(t, ConditionExpression{Operator: "AND"})},
	} {
		if err := db.Create(&item).Error; err != nil {
			t.Fatalf("seed group: %v", err)
		}
	}

	exported, err := svc.Export(nil)
	if err != nil {
		t.Fatalf("export groups: %v", err)
	}
	if len(exported.Rows) != 2 {
		t.Fatalf("expected export capped at 2 rows, got %d", len(exported.Rows))
	}
	if exported.Filename != "cmdb-group-export.csv" {
		t.Fatalf("unexpected export filename %q", exported.Filename)
	}
}

func TestGroupServiceImportEmptyFile(t *testing.T) {
	svc := NewGroupService(setupTestDB(t))

	result, err := svc.Import([][]string{}, "1")
	if err != nil {
		t.Fatalf("import empty file: %v", err)
	}
	if result.Applied {
		t.Fatal("expected applied=false for empty import")
	}
	if !hasGroupImportError(result.Errors, "import.file.empty") {
		t.Fatalf("expected import.file.empty, got %+v", result.Errors)
	}
}

func TestGroupServiceImportCreatesWithParentAndUpdatesByName(t *testing.T) {
	db := setupTestDB(t)
	svc := NewGroupService(db)

	existing := Group{Name: "根分组", Description: "旧描述", Conditions: jsonCondition(t, ConditionExpression{Operator: "AND"})}
	if err := db.Create(&existing).Error; err != nil {
		t.Fatalf("seed existing group: %v", err)
	}

	records := [][]string{
		groupImportExportHeaders(),
		{"根分组", "", `[{"field":"status","op":"=","value":"operational"}]`, "所有运营中的主机"},
		{"Web服务器", "根分组", "", "角色为 web 的主机"},
	}

	result, err := svc.Import(records, "99")
	if err != nil {
		t.Fatalf("import groups: %v", err)
	}
	if !result.Applied || result.Created != 1 || result.Updated != 1 || result.Failed != 0 {
		t.Fatalf("unexpected import result: %+v", result)
	}

	var updated Group
	if err := db.Where("name = ?", "根分组").First(&updated).Error; err != nil {
		t.Fatalf("load updated group: %v", err)
	}
	if updated.Description != "所有运营中的主机" {
		t.Fatalf("unexpected updated group: %+v", updated)
	}

	var created Group
	if err := db.Where("name = ?", "Web服务器").First(&created).Error; err != nil {
		t.Fatalf("load created group: %v", err)
	}
	if created.ParentID != existing.ID {
		t.Fatalf("expected parent id %d, got %d", existing.ID, created.ParentID)
	}
}

func TestGroupServiceImportRejectsUnknownParent(t *testing.T) {
	db := setupTestDB(t)
	svc := NewGroupService(db)
	records := [][]string{
		groupImportExportHeaders(),
		{"孤儿分组", "不存在的父分组", "", ""},
	}

	result, err := svc.Import(records, "1")
	if err != nil {
		t.Fatalf("import unknown parent: %v", err)
	}
	if !hasGroupImportError(result.Errors, "cmdbgroup.parent_not_found") {
		t.Fatalf("expected parent_not_found, got %+v", result.Errors)
	}
	var count int64
	if err := db.Model(&Group{}).Count(&count).Error; err != nil {
		t.Fatalf("count groups: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected no orphan groups written, got %d", count)
	}
}

func TestGroupServiceImportRejectsDuplicateName(t *testing.T) {
	db := setupTestDB(t)
	svc := NewGroupService(db)
	records := [][]string{
		groupImportExportHeaders(),
		{"重复分组", "", "", ""},
		{"重复分组", "", "", ""},
	}

	result, err := svc.Import(records, "1")
	if err != nil {
		t.Fatalf("import duplicate name: %v", err)
	}
	if result.Applied || !hasGroupImportError(result.Errors, "import.duplicate.row.2") {
		t.Fatalf("expected duplicate row error, got %+v", result)
	}
	var count int64
	if err := db.Model(&Group{}).Count(&count).Error; err != nil {
		t.Fatalf("count groups: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected no groups written, got %d", count)
	}
}

func TestGroupServiceImportRejectsInvalidConditions(t *testing.T) {
	svc := NewGroupService(setupTestDB(t))
	records := [][]string{
		groupImportExportHeaders(),
		{"坏条件", "", "not-json-at-all", ""},
	}

	result, err := svc.Import(records, "1")
	if err != nil {
		t.Fatalf("import invalid conditions: %v", err)
	}
	if result.Applied || !hasGroupImportError(result.Errors, "business.cmdb.group.conditions_invalid_json") {
		t.Fatalf("expected conditions_invalid_json, got %+v", result.Errors)
	}
}

func hasGroupImportError(errors []impexp.ImportError, message string) bool {
	for _, item := range errors {
		if item.Message == message {
			return true
		}
	}
	return false
}
