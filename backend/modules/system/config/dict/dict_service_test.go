package config

import (
	"strings"
	"testing"
	"time"

	"gorm.io/gorm"
	"pantheon-ops/backend/pkg/testmysql"
)

func setupDictTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db := testmysql.Open(t)
	if err := db.AutoMigrate(&SystemDictType{}, &SystemDictItem{}); err != nil {
		t.Fatalf("migrate dict: %v", err)
	}
	return db
}

func TestDictService_DeleteTypeReleasesDictCode(t *testing.T) {
	db := setupDictTestDB(t)
	service := NewDictService(db)

	created, err := service.CreateDictType(&DictTypeCreateReq{
		DictCode: "biz_status",
		DictName: "Business Status",
		Status:   1,
	})
	if err != nil {
		t.Fatalf("create dict type: %v", err)
	}
	if err := service.DeleteDictType(created.ID); err != nil {
		t.Fatalf("delete dict type: %v", err)
	}

	var deleted SystemDictType
	if err := db.Unscoped().First(&deleted, created.ID).Error; err != nil {
		t.Fatalf("load deleted dict type: %v", err)
	}
	if !strings.HasPrefix(deleted.DictCode, deletedDictTypeCodePrefix) {
		t.Fatalf("expected archived dict code, got %s", deleted.DictCode)
	}

	recreated, err := service.CreateDictType(&DictTypeCreateReq{
		DictCode: "biz_status",
		DictName: "Business Status",
		Status:   1,
	})
	if err != nil {
		t.Fatalf("recreate dict type: %v", err)
	}
	if recreated.DictCode != "biz_status" {
		t.Fatalf("expected biz_status, got %s", recreated.DictCode)
	}
}

func TestDictService_DeleteItemReleasesItemValue(t *testing.T) {
	db := setupDictTestDB(t)
	service := NewDictService(db)

	if _, err := service.CreateDictType(&DictTypeCreateReq{DictCode: "ticket_status", DictName: "Ticket Status", Status: 1}); err != nil {
		t.Fatalf("create dict type: %v", err)
	}
	created, err := service.CreateDictItem(&DictItemCreateReq{
		DictCode:     "ticket_status",
		ItemLabelKey: "dict.ticket.open",
		ItemValue:    "open",
		Status:       1,
	})
	if err != nil {
		t.Fatalf("create dict item: %v", err)
	}
	if err := service.DeleteDictItem(created.ID); err != nil {
		t.Fatalf("delete dict item: %v", err)
	}

	var deleted SystemDictItem
	if err := db.Unscoped().First(&deleted, created.ID).Error; err != nil {
		t.Fatalf("load deleted dict item: %v", err)
	}
	if !strings.HasPrefix(deleted.ItemValue, deletedDictItemValuePrefix) {
		t.Fatalf("expected archived item value, got %s", deleted.ItemValue)
	}

	recreated, err := service.CreateDictItem(&DictItemCreateReq{
		DictCode:     "ticket_status",
		ItemLabelKey: "dict.ticket.open",
		ItemValue:    "open",
		Status:       1,
	})
	if err != nil {
		t.Fatalf("recreate dict item: %v", err)
	}
	if recreated.ItemValue != "open" {
		t.Fatalf("expected open, got %s", recreated.ItemValue)
	}
}

func TestDictService_MigrateReleasesLegacyDeletedDictKeys(t *testing.T) {
	db := setupDictTestDB(t)
	service := NewDictService(db)

	legacyType := SystemDictType{DictCode: "legacy_dict", DictName: "Legacy Dict", Status: 1}
	if err := db.Create(&legacyType).Error; err != nil {
		t.Fatalf("seed legacy dict type: %v", err)
	}
	if err := db.Model(&legacyType).Update("deleted_at", time.Now()).Error; err != nil {
		t.Fatalf("soft delete legacy dict type: %v", err)
	}

	activeType := SystemDictType{DictCode: "legacy_item_dict", DictName: "Legacy Item Dict", Status: 1}
	if err := db.Create(&activeType).Error; err != nil {
		t.Fatalf("seed active dict type: %v", err)
	}
	legacyItem := SystemDictItem{DictCode: activeType.DictCode, ItemLabelKey: "dict.legacy.item", ItemValue: "same", Status: 1}
	if err := db.Create(&legacyItem).Error; err != nil {
		t.Fatalf("seed legacy dict item: %v", err)
	}
	if err := db.Model(&legacyItem).Update("deleted_at", time.Now()).Error; err != nil {
		t.Fatalf("soft delete legacy dict item: %v", err)
	}

	if err := service.Migrate(); err != nil {
		t.Fatalf("migrate dict: %v", err)
	}

	var repairedType SystemDictType
	if err := db.Unscoped().First(&repairedType, legacyType.ID).Error; err != nil {
		t.Fatalf("load repaired dict type: %v", err)
	}
	if !strings.HasPrefix(repairedType.DictCode, deletedDictTypeCodePrefix) {
		t.Fatalf("expected archived legacy dict code, got %s", repairedType.DictCode)
	}
	var repairedItem SystemDictItem
	if err := db.Unscoped().First(&repairedItem, legacyItem.ID).Error; err != nil {
		t.Fatalf("load repaired dict item: %v", err)
	}
	if !strings.HasPrefix(repairedItem.ItemValue, deletedDictItemValuePrefix) {
		t.Fatalf("expected archived legacy item value, got %s", repairedItem.ItemValue)
	}

	if _, err := service.CreateDictType(&DictTypeCreateReq{DictCode: "legacy_dict", DictName: "Legacy Dict", Status: 1}); err != nil {
		t.Fatalf("expected legacy dict code to be reusable: %v", err)
	}
	if _, err := service.CreateDictItem(&DictItemCreateReq{DictCode: activeType.DictCode, ItemLabelKey: "dict.legacy.item", ItemValue: "same", Status: 1}); err != nil {
		t.Fatalf("expected legacy item value to be reusable: %v", err)
	}
}

func TestDictService_ImportTemplateAndExport(t *testing.T) {
	db := setupDictTestDB(t)
	service := NewDictService(db)

	typeTemplate := service.BuildDictTypeImportTemplate()
	if len(typeTemplate.Rows) == 0 || !strings.HasPrefix(typeTemplate.Rows[0][0], "#") {
		t.Fatalf("expected dict type template instructions, got %+v", typeTemplate.Rows)
	}
	typeTemplateResult, err := service.ImportDictTypes(append([][]string{typeTemplate.Headers}, typeTemplate.Rows...))
	if err != nil {
		t.Fatalf("import type template comments: %v", err)
	}
	if !typeTemplateResult.Applied || typeTemplateResult.Created != 0 || typeTemplateResult.Failed != 0 {
		t.Fatalf("expected type template comments to be ignored, got %+v", typeTemplateResult)
	}

	typeResult, err := service.ImportDictTypes([][]string{
		typeTemplate.Headers,
		{"biz_status", "业务状态", "business", "1", "业务通用状态字典"},
	})
	if err != nil {
		t.Fatalf("import dict type: %v", err)
	}
	if !typeResult.Applied || typeResult.Created != 1 || typeResult.Failed != 0 {
		t.Fatalf("unexpected type import result: %+v", typeResult)
	}

	itemTemplate := service.BuildDictItemImportTemplate()
	if len(itemTemplate.Rows) == 0 || !strings.HasPrefix(itemTemplate.Rows[0][0], "#") {
		t.Fatalf("expected dict item template instructions, got %+v", itemTemplate.Rows)
	}
	itemTemplateResult, err := service.ImportDictItems(append([][]string{itemTemplate.Headers}, itemTemplate.Rows...))
	if err != nil {
		t.Fatalf("import item template comments: %v", err)
	}
	if !itemTemplateResult.Applied || itemTemplateResult.Created != 0 || itemTemplateResult.Failed != 0 {
		t.Fatalf("expected item template comments to be ignored, got %+v", itemTemplateResult)
	}

	itemResult, err := service.ImportDictItems([][]string{
		itemTemplate.Headers,
		{"biz_status", "dict.biz_status.enabled", "enabled", "green", "10", "1", "启用"},
	})
	if err != nil {
		t.Fatalf("import dict item: %v", err)
	}
	if !itemResult.Applied || itemResult.Created != 1 || itemResult.Failed != 0 {
		t.Fatalf("unexpected item import result: %+v", itemResult)
	}

	exportedTypes, err := service.ExportDictTypes(&DictTypeListQuery{DictCode: "biz_status"})
	if err != nil {
		t.Fatalf("export dict type: %v", err)
	}
	if len(exportedTypes.Rows) != 1 || exportedTypes.Rows[0][0] != "biz_status" {
		t.Fatalf("unexpected type export rows: %+v", exportedTypes.Rows)
	}

	exportedItems, err := service.ExportDictItems(&DictItemListQuery{DictCode: "biz_status"})
	if err != nil {
		t.Fatalf("export dict item: %v", err)
	}
	if len(exportedItems.Rows) != 1 || exportedItems.Rows[0][0] != "biz_status" || exportedItems.Rows[0][2] != "enabled" {
		t.Fatalf("unexpected item export rows: %+v", exportedItems.Rows)
	}
}

func TestDictService_ListDictItemsSupportsPagingAndKeyword(t *testing.T) {
	db := setupDictTestDB(t)
	service := NewDictService(db)

	if _, err := service.CreateDictType(&DictTypeCreateReq{
		DictCode: "service_status",
		DictName: "Service Status",
		Module:   "business.cmdb",
		Status:   1,
	}); err != nil {
		t.Fatalf("create dict type: %v", err)
	}

	seeds := []DictItemCreateReq{
		{DictCode: "service_status", ItemLabelKey: "dict.service_status.pending", ItemValue: "pending", Sort: 10, Status: 1, Remark: "pending item"},
		{DictCode: "service_status", ItemLabelKey: "dict.service_status.running", ItemValue: "running", Sort: 20, Status: 1, Remark: "running item"},
		{DictCode: "service_status", ItemLabelKey: "dict.service_status.stopped", ItemValue: "stopped", Sort: 30, Status: 2, Remark: "stopped item"},
	}
	for _, item := range seeds {
		if _, err := service.CreateDictItem(&item); err != nil {
			t.Fatalf("create dict item %s: %v", item.ItemValue, err)
		}
	}

	pageResp, err := service.ListDictItems(&DictItemListQuery{
		DictCode: "service_status",
		Page:     1,
		PageSize: 2,
	})
	if err != nil {
		t.Fatalf("list dict items: %v", err)
	}
	if pageResp.Total != 3 || len(pageResp.Items) != 2 || pageResp.PageSize != 2 {
		t.Fatalf("unexpected page resp: %+v", pageResp)
	}

	filteredResp, err := service.ListDictItems(&DictItemListQuery{
		DictCode: "service_status",
		Keyword:  "run",
		Status:   func() *int { value := 1; return &value }(),
		Page:     1,
		PageSize: 10,
	})
	if err != nil {
		t.Fatalf("filter dict items: %v", err)
	}
	if filteredResp.Total != 1 || len(filteredResp.Items) != 1 || filteredResp.Items[0].ItemValue != "running" {
		t.Fatalf("unexpected filtered resp: %+v", filteredResp)
	}

	typeRows, err := service.ListDictTypes(&DictTypeListQuery{DictCode: "service_status"})
	if err != nil {
		t.Fatalf("list dict types: %v", err)
	}
	if len(typeRows) != 1 {
		t.Fatalf("expected one dict type, got %d", len(typeRows))
	}
	if typeRows[0].ItemCount != 3 || typeRows[0].ActiveItemCount != 2 || typeRows[0].DisabledItemCount != 1 {
		t.Fatalf("unexpected dict type stats: %+v", typeRows[0])
	}
}

func TestDictService_BatchUpdateAndReorderDictItems(t *testing.T) {
	db := setupDictTestDB(t)
	service := NewDictService(db)

	typeResp, err := service.CreateDictType(&DictTypeCreateReq{
		DictCode: "deploy_status",
		DictName: "Deploy Status",
		Status:   1,
	})
	if err != nil {
		t.Fatalf("create dict type: %v", err)
	}
	first, err := service.CreateDictItem(&DictItemCreateReq{
		DictCode:     typeResp.DictCode,
		ItemLabelKey: "dict.deploy_status.pending",
		ItemValue:    "pending",
		Sort:         10,
		Status:       1,
	})
	if err != nil {
		t.Fatalf("create first item: %v", err)
	}
	second, err := service.CreateDictItem(&DictItemCreateReq{
		DictCode:     typeResp.DictCode,
		ItemLabelKey: "dict.deploy_status.running",
		ItemValue:    "running",
		Sort:         20,
		Status:       1,
	})
	if err != nil {
		t.Fatalf("create second item: %v", err)
	}

	updatedCount, err := service.BatchUpdateDictTypeStatus([]uint64{typeResp.ID, typeResp.ID}, 2)
	if err != nil || updatedCount != 1 {
		t.Fatalf("batch update dict type status failed: count=%d err=%v", updatedCount, err)
	}
	updatedCount, err = service.BatchUpdateDictItemStatus([]uint64{first.ID, second.ID, second.ID}, 2)
	if err != nil || updatedCount != 2 {
		t.Fatalf("batch update dict item status failed: count=%d err=%v", updatedCount, err)
	}

	moved, err := service.ReorderDictItem(second.ID, "up")
	if err != nil {
		t.Fatalf("reorder dict item: %v", err)
	}
	if moved.Sort != 10 {
		t.Fatalf("expected moved item sort=10, got %+v", moved)
	}

	pageResp, err := service.ListDictItems(&DictItemListQuery{DictCode: typeResp.DictCode, Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("list dict items: %v", err)
	}
	if len(pageResp.Items) != 2 || pageResp.Items[0].ID != second.ID {
		t.Fatalf("unexpected reorder result: %+v", pageResp.Items)
	}
}
