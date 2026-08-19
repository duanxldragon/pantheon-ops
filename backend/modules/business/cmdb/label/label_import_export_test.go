package label

import (
	"testing"

	"pantheon-base/pkg/impexp"
)

func TestLabelServiceExportHonorsRowCap(t *testing.T) {
	db := setupTestDB(t)
	svc := NewLabelService(db)

	oldCap := maxLabelExportRows
	maxLabelExportRows = 2
	t.Cleanup(func() {
		maxLabelExportRows = oldCap
	})

	for i, item := range []LabelSchema{
		{Key: "env", Name: "环境", Category: "base", ValueMode: "enum"},
		{Key: "region", Name: "区域", Category: "network", ValueMode: "free"},
		{Key: "team", Name: "团队", Category: "business", ValueMode: "dict"},
	} {
		if err := db.Create(&item).Error; err != nil {
			t.Fatalf("seed label %d: %v", i, err)
		}
	}

	exported, err := svc.Export(LabelSchemaQuery{}, nil)
	if err != nil {
		t.Fatalf("export labels: %v", err)
	}
	if len(exported.Rows) != 2 {
		t.Fatalf("expected export capped at 2 rows, got %d", len(exported.Rows))
	}
	if exported.Filename != "cmdb-label-export.csv" {
		t.Fatalf("unexpected export filename %q", exported.Filename)
	}
}

func TestLabelServiceImportEmptyFile(t *testing.T) {
	svc := NewLabelService(setupTestDB(t))

	result, err := svc.Import([][]string{}, "1")
	if err != nil {
		t.Fatalf("import empty file: %v", err)
	}
	if result.Applied {
		t.Fatal("expected applied=false for empty import")
	}
	if !hasLabelImportError(result.Errors, "import.file.empty") {
		t.Fatalf("expected import.file.empty, got %+v", result.Errors)
	}
}

func TestLabelServiceImportCreatesAndUpdatesByKey(t *testing.T) {
	db := setupTestDB(t)
	svc := NewLabelService(db)

	existing := LabelSchema{
		Key:       "env",
		Name:      "旧环境",
		Category:  "base",
		ValueMode: "free",
		Status:    "enabled",
	}
	if err := db.Create(&existing).Error; err != nil {
		t.Fatalf("seed existing label: %v", err)
	}

	records := [][]string{
		labelImportExportHeaders(),
		{"ENV", "环境", "base", "enum", "", "dev; test; prod", "true", "enabled", "运行环境标签"},
		{"team", "团队", "business", "dict", "dept_teams", "", "false", "enabled", "所属团队"},
	}

	result, err := svc.Import(records, "99")
	if err != nil {
		t.Fatalf("import labels: %v", err)
	}
	if !result.Applied || result.Created != 1 || result.Updated != 1 || result.Failed != 0 {
		t.Fatalf("unexpected import result: %+v", result)
	}

	var updated LabelSchema
	if err := db.Where("`key` = ?", "env").First(&updated).Error; err != nil {
		t.Fatalf("load updated label: %v", err)
	}
	if updated.Name != "环境" || updated.ValueMode != "enum" || updated.Category != "base" || updated.Required != true {
		t.Fatalf("unexpected updated label: %+v", updated)
	}

	var created LabelSchema
	if err := db.Where("`key` = ?", "team").First(&created).Error; err != nil {
		t.Fatalf("load created label: %v", err)
	}
	if created.DictCode != "dept_teams" || created.Status != "enabled" {
		t.Fatalf("unexpected created label: %+v", created)
	}
}

func TestLabelServiceImportRejectsInvalidEnums(t *testing.T) {
	svc := NewLabelService(setupTestDB(t))
	records := [][]string{
		labelImportExportHeaders(),
		{"bad-category", "分类错误", "unknown", "enum", "", "a; b", "false", "enabled", ""},
		{"bad-mode", "模式错误", "base", "weird", "", "", "false", "enabled", ""},
		{"bad-status", "状态错误", "base", "free", "", "", "false", "paused", ""},
	}

	result, err := svc.Import(records, "1")
	if err != nil {
		t.Fatalf("import invalid enums: %v", err)
	}
	if result.Applied {
		t.Fatal("expected applied=false for invalid enums")
	}
	for _, key := range []string{"cmdblabel.category.invalid", "cmdblabel.valueMode.invalid", "cmdblabel.status.invalid"} {
		if !hasLabelImportError(result.Errors, key) {
			t.Fatalf("expected error %q, got %+v", key, result.Errors)
		}
	}
}

func TestLabelServiceImportRejectsMissingBusinessFields(t *testing.T) {
	svc := NewLabelService(setupTestDB(t))
	records := [][]string{
		labelImportExportHeaders(),
		{"no-options", "缺选项", "base", "enum", "", "", "false", "enabled", ""},
		{"no-dict", "缺字典", "base", "dict", "", "", "false", "enabled", ""},
	}

	result, err := svc.Import(records, "1")
	if err != nil {
		t.Fatalf("import missing business fields: %v", err)
	}
	if result.Applied {
		t.Fatal("expected applied=false for missing business fields")
	}
	if !hasLabelImportError(result.Errors, "cmdblabel.options_required_for_enum") {
		t.Fatalf("expected options_required_for_enum, got %+v", result.Errors)
	}
	if !hasLabelImportError(result.Errors, "cmdblabel.dictCode_required_for_dict") {
		t.Fatalf("expected dictCode_required_for_dict, got %+v", result.Errors)
	}
}

func TestLabelServiceImportRejectsDuplicateKey(t *testing.T) {
	db := setupTestDB(t)
	svc := NewLabelService(db)
	records := [][]string{
		labelImportExportHeaders(),
		{"dup", "重复一", "base", "free", "", "", "false", "enabled", ""},
		{"DUP", "重复二", "base", "free", "", "", "false", "enabled", ""},
	}

	result, err := svc.Import(records, "1")
	if err != nil {
		t.Fatalf("import duplicate key: %v", err)
	}
	if result.Applied || !hasLabelImportError(result.Errors, "import.duplicate.row.2") {
		t.Fatalf("expected duplicate row error, got %+v", result)
	}
	var count int64
	if err := db.Model(&LabelSchema{}).Count(&count).Error; err != nil {
		t.Fatalf("count labels: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected no labels written, got %d", count)
	}
}

func hasLabelImportError(errors []impexp.ImportError, message string) bool {
	for _, item := range errors {
		if item.Message == message {
			return true
		}
	}
	return false
}
