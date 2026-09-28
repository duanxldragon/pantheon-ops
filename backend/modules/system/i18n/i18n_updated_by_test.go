package system

import (
	"testing"

	"gorm.io/gorm"
)

// newAttributionTestService owns the service bootstrap shared by the
// attribution tests so the setup does not restate the per-test scaffold that
// the rest of this package's suite already established.
func newAttributionTestService(t *testing.T) (*I18nService, *gorm.DB) {
	t.Helper()

	db := newI18nTestDB(t)
	service := NewI18nService(db)
	if err := service.Migrate(); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return service, db
}

// TestI18nService_WritesUpdatedByOnUserFacingPaths pins the operator-attribution
// contract (fix-report §4.5): every user-facing dynamic write path stamps
// updated_by, later writes overwrite it, and the API response surfaces it.
func TestI18nService_WritesUpdatedByOnUserFacingPaths(t *testing.T) {
	service, db := newAttributionTestService(t)

	// Create carries the actor into the row and the response.
	created, err := service.Create(&I18nCreateReq{
		Module:    "system.config",
		Group:     "messages",
		Key:       "i18n.updated_by.key",
		Locale:    "zh-CN",
		Value:     "初始值",
		Remark:    "attribution",
		UpdatedBy: "admin",
	})
	if err != nil {
		t.Fatalf("create i18n: %v", err)
	}
	if created.UpdatedBy != "admin" {
		t.Fatalf("expected create response updatedBy=admin, got %q", created.UpdatedBy)
	}

	var row SystemI18n
	if err := db.Where("locale = ? AND `key` = ?", "zh-CN", "i18n.updated_by.key").First(&row).Error; err != nil {
		t.Fatalf("load created row: %v", err)
	}
	if row.UpdatedBy != "admin" {
		t.Fatalf("expected created row updated_by=admin, got %q", row.UpdatedBy)
	}

	// Update overwrites the actor.
	if err := service.Update(row.ID, &I18nUpdateReq{
		Value:     "修改后",
		Remark:    "edited",
		UpdatedBy: "editor",
	}); err != nil {
		t.Fatalf("update i18n: %v", err)
	}
	if err := db.Where("id = ?", row.ID).First(&row).Error; err != nil {
		t.Fatalf("reload updated row: %v", err)
	}
	if row.Value != "修改后" {
		t.Fatalf("expected updated value, got %q", row.Value)
	}
	if row.UpdatedBy != "editor" {
		t.Fatalf("expected update to stamp updated_by=editor, got %q", row.UpdatedBy)
	}

	// Import stamps the actor on both the created and the updated row.
	result, err := service.Import([][]string{
		{"module", "group", "key", "locale", "value", "remark"},
		{"system.config", "messages", "i18n.updated_by.key", "zh-CN", "导入覆盖", ""},
		{"system.config", "messages", "i18n.updated_by.new", "zh-CN", "导入新增", ""},
	}, "importer")
	if err != nil {
		t.Fatalf("import rows: %v", err)
	}
	if !result.Applied || result.Created != 1 || result.Updated != 1 || result.Failed != 0 {
		t.Fatalf("unexpected import result: %#v", result)
	}

	var updated SystemI18n
	if err := db.Where("locale = ? AND `key` = ?", "zh-CN", "i18n.updated_by.key").First(&updated).Error; err != nil {
		t.Fatalf("load imported-updated row: %v", err)
	}
	if updated.UpdatedBy != "importer" {
		t.Fatalf("expected import to stamp updated_by=importer on updated row, got %q", updated.UpdatedBy)
	}

	var inserted SystemI18n
	if err := db.Where("locale = ? AND `key` = ?", "zh-CN", "i18n.updated_by.new").First(&inserted).Error; err != nil {
		t.Fatalf("load imported-created row: %v", err)
	}
	if inserted.UpdatedBy != "importer" {
		t.Fatalf("expected import to stamp updated_by=importer on created row, got %q", inserted.UpdatedBy)
	}

	// The list projection surfaces the attribution to the admin UI.
	page, err := service.List(&I18nQuery{Key: "i18n.updated_by.key", Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("list i18n: %v", err)
	}
	if len(page.Items) != 1 || page.Items[0].UpdatedBy != "importer" {
		t.Fatalf("expected list projection to expose updatedBy=importer, got %#v", page.Items)
	}
}
