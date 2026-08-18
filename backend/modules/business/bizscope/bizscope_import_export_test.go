package bizscope

import (
	"testing"

	"pantheon-base/pkg/impexp"
)

const activeStatus = "active"

func TestBizScopeServiceExportHonorsRowCap(t *testing.T) {
	db := setupBizScopeTestDB(t)
	svc := NewService(db)

	oldCap := maxBizScopeExportRows
	maxBizScopeExportRows = 2
	t.Cleanup(func() {
		maxBizScopeExportRows = oldCap
	})

	for _, item := range []BizScope{
		{Code: "scope-a", Name: "域 A", Environment: "dev", Status: "active"},
		{Code: "scope-b", Name: "域 B", Environment: "test", Status: "active"},
		{Code: "scope-c", Name: "域 C", Environment: "prod", Status: "active"},
	} {
		if err := db.Create(&item).Error; err != nil {
			t.Fatalf("seed scope: %v", err)
		}
	}

	exported, err := svc.Export(BizScopeListQuery{}, nil)
	if err != nil {
		t.Fatalf("export scopes: %v", err)
	}
	if len(exported.Rows) != 2 {
		t.Fatalf("expected export capped at 2 rows, got %d", len(exported.Rows))
	}
	if exported.Filename != "business-scope-export.csv" {
		t.Fatalf("unexpected export filename %q", exported.Filename)
	}
}

func TestBizScopeServiceImportEmptyFile(t *testing.T) {
	svc := NewService(setupBizScopeTestDB(t))

	result, err := svc.Import([][]string{}, "1")
	if err != nil {
		t.Fatalf("import empty file: %v", err)
	}
	if result.Applied {
		t.Fatal("expected applied=false for empty import")
	}
	if !hasBizScopeImportError(result.Errors, "import.file.empty") {
		t.Fatalf("expected import.file.empty, got %+v", result.Errors)
	}
}

func TestBizScopeServiceImportCreatesAndUpdatesByCode(t *testing.T) {
	db := setupBizScopeTestDB(t)
	svc := NewService(db)

	existing := BizScope{Code: "his-prod", Name: "旧生产", Environment: "prod", Status: "inactive"}
	if err := db.Create(&existing).Error; err != nil {
		t.Fatalf("seed existing scope: %v", err)
	}

	records := [][]string{
		bizscopeImportExportHeaders(),
		{"his-prod", "HIS 生产", "ops", "prod", activeStatus, "更新后的备注"},
		{"his-dev", "HIS 开发", "ops", "dev", "", "开发环境"},
	}

	result, err := svc.Import(records, "99")
	if err != nil {
		t.Fatalf("import scopes: %v", err)
	}
	if !result.Applied || result.Created != 1 || result.Updated != 1 || result.Failed != 0 {
		t.Fatalf("unexpected import result: %+v", result)
	}

	var updated BizScope
	if err := db.Where("code = ?", "his-prod").First(&updated).Error; err != nil {
		t.Fatalf("load updated scope: %v", err)
	}
	if updated.Name != "HIS 生产" || updated.Status != activeStatus || updated.Remark != "更新后的备注" {
		t.Fatalf("unexpected updated scope: %+v", updated)
	}

	var created BizScope
	if err := db.Where("code = ?", "his-dev").First(&created).Error; err != nil {
		t.Fatalf("load created scope: %v", err)
	}
	if created.Status != activeStatus || created.Environment != "dev" {
		t.Fatalf("unexpected created scope (default status): %+v", created)
	}
}

func TestBizScopeServiceImportRejectsInvalidValues(t *testing.T) {
	svc := NewService(setupBizScopeTestDB(t))
	records := [][]string{
		bizscopeImportExportHeaders(),
		{"scope-x", "坏环境", "ops", "uat", "active", ""},
		{"scope-y", "坏状态", "ops", "dev", "paused", ""},
	}

	result, err := svc.Import(records, "1")
	if err != nil {
		t.Fatalf("import invalid values: %v", err)
	}
	if result.Applied {
		t.Fatal("expected applied=false for invalid values")
	}
	if !hasBizScopeImportError(result.Errors, "business.bizscope.environment_invalid") {
		t.Fatalf("expected environment_invalid, got %+v", result.Errors)
	}
	if !hasBizScopeImportError(result.Errors, "business.bizscope.status_invalid") {
		t.Fatalf("expected status_invalid, got %+v", result.Errors)
	}
}

func TestBizScopeServiceImportRejectsDuplicateCode(t *testing.T) {
	db := setupBizScopeTestDB(t)
	svc := NewService(db)
	records := [][]string{
		bizscopeImportExportHeaders(),
		{"dup-scope", "重复一", "", "dev", "", ""},
		{"dup-scope", "重复二", "", "dev", "", ""},
	}

	result, err := svc.Import(records, "1")
	if err != nil {
		t.Fatalf("import duplicate code: %v", err)
	}
	if result.Applied || !hasBizScopeImportError(result.Errors, "import.duplicate.row.2") {
		t.Fatalf("expected duplicate row error, got %+v", result)
	}
	var count int64
	if err := db.Model(&BizScope{}).Count(&count).Error; err != nil {
		t.Fatalf("count scopes: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected no scopes written, got %d", count)
	}
}

func hasBizScopeImportError(errors []impexp.ImportError, message string) bool {
	for _, item := range errors {
		if item.Message == message {
			return true
		}
	}
	return false
}
