package system

import (
	"errors"
	"strings"
	"testing"
	"time"

	"gorm.io/gorm"
	"pantheon-ops/backend/pkg/testmysql"
)

func newI18nTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	return testmysql.Open(t)
}

func TestI18nService_SeedI18nModuleI18nIsIdempotent(t *testing.T) {
	db := newI18nTestDB(t)
	service := NewI18nService(db)
	if err := service.Migrate(); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if err := service.SeedI18nModuleI18n(db); err != nil {
		t.Fatalf("first seed: %v", err)
	}

	var firstCount int64
	if err := db.Model(&SystemI18n{}).Count(&firstCount).Error; err != nil {
		t.Fatalf("count i18n rows after first seed: %v", err)
	}

	if err := service.SeedI18nModuleI18n(db); err != nil {
		t.Fatalf("second seed: %v", err)
	}

	var secondCount int64
	if err := db.Model(&SystemI18n{}).Count(&secondCount).Error; err != nil {
		t.Fatalf("count i18n rows: %v", err)
	}
	if secondCount != firstCount {
		t.Fatalf("expected seed to be idempotent, first=%d second=%d", firstCount, secondCount)
	}

	checks := []struct {
		module string
		locale string
		key    string
		value  string
	}{
		{module: "system.auth", locale: "zh-CN", key: "auth.security.title", value: "安全中心"},
		{module: "system.profile", locale: "en-US", key: "system.profile.title", value: "Profile Center"},
		{module: "system.auth", locale: "en-US", key: "system.permission.session.delete", value: "Session Revoke"},
		{module: "system.auth", locale: "zh-CN", key: "system.permission.session.clear", value: "清理历史会话"},
		{module: "system.audit", locale: "zh-CN", key: "system.permission.operation_log.export", value: "操作日志导出"},
	}

	for _, item := range checks {
		var row SystemI18n
		if err := db.Where("module = ? AND locale = ? AND `key` = ?", item.module, item.locale, item.key).First(&row).Error; err != nil {
			t.Fatalf("load seeded row %s/%s/%s: %v", item.module, item.locale, item.key, err)
		}
		if row.Value != item.value {
			t.Fatalf("expected value %q for %s/%s/%s, got %q", item.value, item.module, item.locale, item.key, row.Value)
		}
	}
}

func TestI18nService_SeedI18nModuleI18nBackfillsEmptyValue(t *testing.T) {
	db := newI18nTestDB(t)
	service := NewI18nService(db)
	if err := service.Migrate(); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	if err := db.Create(&SystemI18n{
		Module: "system.config",
		Group:  "messages",
		Key:    "i18n.value",
		Locale: "zh-CN",
		Value:  "",
		Remark: "legacy empty value",
	}).Error; err != nil {
		t.Fatalf("seed legacy empty row: %v", err)
	}

	if err := service.SeedI18nModuleI18n(db); err != nil {
		t.Fatalf("seed module i18n: %v", err)
	}

	var row SystemI18n
	if err := db.Where("module = ? AND locale = ? AND `key` = ?", "system.config", "zh-CN", "i18n.value").First(&row).Error; err != nil {
		t.Fatalf("load backfilled row: %v", err)
	}
	if row.Value != "内容" {
		t.Fatalf("expected backfilled value %q, got %q", "内容", row.Value)
	}
}

func TestI18nService_UpdateRejectsBlankValue(t *testing.T) {
	db := newI18nTestDB(t)
	service := NewI18nService(db)
	if err := service.Migrate(); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	row := SystemI18n{
		Module: "system.config",
		Group:  "messages",
		Key:    "i18n.value",
		Locale: "zh-CN",
		Value:  "内容",
	}
	if err := db.Create(&row).Error; err != nil {
		t.Fatalf("create row: %v", err)
	}

	if err := service.Update(row.ID, &I18nUpdateReq{Value: "   ", Remark: "blank"}); err == nil {
		t.Fatalf("expected blank update to be rejected")
	}

	var current SystemI18n
	if err := db.First(&current, row.ID).Error; err != nil {
		t.Fatalf("reload row: %v", err)
	}
	if current.Value != "内容" {
		t.Fatalf("expected original value preserved, got %q", current.Value)
	}
}

func TestI18nService_GetAndDelete(t *testing.T) {
	db := newI18nTestDB(t)
	service := NewI18nService(db)
	if err := service.Migrate(); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	row := SystemI18n{
		Module: "system.config",
		Group:  "messages",
		Key:    "i18n.value",
		Locale: "zh-CN",
		Value:  "内容",
	}
	if err := db.Create(&row).Error; err != nil {
		t.Fatalf("create row: %v", err)
	}

	detail, err := service.Get(row.ID)
	if err != nil {
		t.Fatalf("get row: %v", err)
	}
	if detail.Key != "i18n.value" || detail.Value != "内容" {
		t.Fatalf("unexpected detail: %#v", detail)
	}

	if err := service.Delete(row.ID); err != nil {
		t.Fatalf("delete row: %v", err)
	}

	if _, err := service.Get(row.ID); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("expected deleted row to be missing, got %v", err)
	}
}

func TestI18nService_ListSupportsGroupAndBatchDelete(t *testing.T) {
	db := newI18nTestDB(t)
	service := NewI18nService(db)
	if err := service.Migrate(); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	rows := []SystemI18n{
		{Module: "system.config", Group: "messages", Key: "i18n.value", Locale: "zh-CN", Value: "内容"},
		{Module: "system.config", Group: "permission", Key: "system.permission.i18n.update", Locale: "zh-CN", Value: "修改翻译"},
	}
	for _, row := range rows {
		if err := db.Create(&row).Error; err != nil {
			t.Fatalf("create row: %v", err)
		}
	}

	resp, err := service.List(&I18nQuery{Group: "messages", Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("list by group: %v", err)
	}
	if len(resp.Items) != 1 || resp.Items[0].Group != "messages" {
		t.Fatalf("unexpected group filter result: %#v", resp.Items)
	}

	var ids []uint64
	if err := db.Model(&SystemI18n{}).Pluck("id", &ids).Error; err != nil {
		t.Fatalf("pluck ids: %v", err)
	}
	if err := service.DeleteBatch(ids); err != nil {
		t.Fatalf("delete batch: %v", err)
	}

	var count int64
	if err := db.Model(&SystemI18n{}).Count(&count).Error; err != nil {
		t.Fatalf("count rows: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected rows deleted, remaining=%d", count)
	}
}

func TestI18nService_SyncMissingKeysReturnsCreatedKeys(t *testing.T) {
	db := newI18nTestDB(t)
	service := NewI18nService(db)
	if err := service.Migrate(); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	resp, err := service.SyncMissingKeys()
	if err != nil {
		t.Fatalf("sync keys: %v", err)
	}
	if resp == nil {
		t.Fatalf("expected sync response")
	}
	if resp.Count < 0 || len(resp.Keys) < 0 {
		t.Fatalf("invalid sync response: %#v", resp)
	}
	for _, key := range resp.Keys {
		for _, locale := range []string{"zh-CN", "en-US", "ja-JP", "ko-KR", "fr-FR"} {
			var count int64
			if err := db.Model(&SystemI18n{}).Where("`key` = ? AND locale = ?", key, locale).Count(&count).Error; err != nil {
				t.Fatalf("count %s/%s: %v", key, locale, err)
			}
			if count == 0 {
				t.Fatalf("expected synchronized key %s to exist for locale %s", key, locale)
			}
		}
	}
}

func TestI18nService_GetLangPackMergesZhFallback(t *testing.T) {
	db := newI18nTestDB(t)
	service := NewI18nService(db)
	if err := service.Migrate(); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	items := []SystemI18n{
		{Module: "system.config", Group: "messages", Key: "i18n.only.zh", Locale: "zh-CN", Value: "仅中文"},
		{Module: "system.config", Group: "messages", Key: "i18n.override", Locale: "zh-CN", Value: "中文值"},
		{Module: "system.config", Group: "messages", Key: "i18n.override", Locale: "en-US", Value: "English Value"},
		{Module: "system.config", Group: "messages", Key: "i18n.batchDelete", Locale: "fr-FR", Value: "[i18n.batchDelete]"},
	}
	if err := service.BatchInsert(items); err != nil {
		t.Fatalf("seed items: %v", err)
	}

	pack, err := service.GetLangPack("en-US")
	if err != nil {
		t.Fatalf("get lang pack: %v", err)
	}
	if pack["i18n.only.zh"] != "仅中文" {
		t.Fatalf("expected zh fallback value, got %q", pack["i18n.only.zh"])
	}
	if pack["i18n.override"] != "English Value" {
		t.Fatalf("expected locale override value, got %q", pack["i18n.override"])
	}

	frPack, err := service.GetLangPack("fr-FR")
	if err != nil {
		t.Fatalf("get fr lang pack: %v", err)
	}
	if frPack["i18n.batchDelete"] != "Suppression par lot" {
		t.Fatalf("expected builtin fr value to survive placeholder override, got %q", frPack["i18n.batchDelete"])
	}
}

func TestI18nService_GetOverviewSummarizesCoverage(t *testing.T) {
	db := newI18nTestDB(t)
	service := NewI18nService(db)
	if err := service.Migrate(); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	items := []SystemI18n{
		{Module: "system.config", Group: "messages", Key: "i18n.cover.zh_only", Locale: "zh-CN", Value: "中文"},
		{Module: "system.config", Group: "messages", Key: "i18n.cover.full", Locale: "zh-CN", Value: "完整"},
		{Module: "system.config", Group: "messages", Key: "i18n.cover.full", Locale: "en-US", Value: "Full"},
		{Module: "system.config", Group: "messages", Key: "i18n.cover.placeholder", Locale: "en-US", Value: "[i18n.cover.placeholder]"},
	}
	if err := service.BatchInsert(items); err != nil {
		t.Fatalf("seed items: %v", err)
	}

	overview, err := service.GetOverview()
	if err != nil {
		t.Fatalf("get overview: %v", err)
	}
	if overview.TotalEntries != 4 {
		t.Fatalf("expected total entries 4, got %d", overview.TotalEntries)
	}
	if overview.MissingValueCount != 1 {
		t.Fatalf("expected missing value count 1, got %d", overview.MissingValueCount)
	}
	if overview.MissingLocaleCount == 0 {
		t.Fatalf("expected missing locale count > 0, got %d", overview.MissingLocaleCount)
	}
	if len(overview.Locales) < 5 {
		t.Fatalf("expected locale list to contain the configured locales, got %#v", overview.Locales)
	}
}

func TestI18nService_ListMissingLocales(t *testing.T) {
	db := newI18nTestDB(t)
	service := NewI18nService(db)
	if err := service.Migrate(); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	items := []SystemI18n{
		{Module: "system.config", Group: "messages", Key: "i18n.missing.zh_only", Locale: "zh-CN", Value: "仅中文"},
		{Module: "system.config", Group: "messages", Key: "i18n.missing.complete", Locale: "zh-CN", Value: "完整"},
		{Module: "system.config", Group: "messages", Key: "i18n.missing.complete", Locale: "en-US", Value: "Complete"},
	}
	if err := service.BatchInsert(items); err != nil {
		t.Fatalf("seed items: %v", err)
	}

	resp, err := service.ListMissingLocales("")
	if err != nil {
		t.Fatalf("list missing locales: %v", err)
	}
	if resp.Total == 0 {
		t.Fatalf("expected missing locale rows")
	}
	found := false
	for _, item := range resp.Items {
		if item.Key == "i18n.missing.zh_only" {
			found = true
			if len(item.MissingLocales) == 0 {
				t.Fatalf("expected missing locales for %s", item.Key)
			}
		}
	}
	if !found {
		t.Fatalf("expected missing locale item for zh-only key")
	}

	filtered, err := service.ListMissingLocales("system.config")
	if err != nil {
		t.Fatalf("list missing locales by module: %v", err)
	}
	if filtered.Total != resp.Total {
		t.Fatalf("expected filtered total %d, got %d", resp.Total, filtered.Total)
	}

	emptyFiltered, err := service.ListMissingLocales("system.iam")
	if err != nil {
		t.Fatalf("list missing locales by empty module: %v", err)
	}
	if emptyFiltered.Total != 0 {
		t.Fatalf("expected empty filtered result, got %d", emptyFiltered.Total)
	}
}

func TestI18nService_FillMissingLocales(t *testing.T) {
	db := newI18nTestDB(t)
	service := NewI18nService(db)
	if err := service.Migrate(); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	items := []SystemI18n{
		{Module: "system.config", Group: "messages", Key: "i18n.fill.zh_only", Locale: "zh-CN", Value: "仅中文"},
		{Module: "system.iam", Group: "messages", Key: "i18n.fill.user.zh_only", Locale: "zh-CN", Value: "仅中文用户"},
	}
	if err := service.BatchInsert(items); err != nil {
		t.Fatalf("seed items: %v", err)
	}

	resp, err := service.FillMissingLocales("system.config")
	if err != nil {
		t.Fatalf("fill missing locales: %v", err)
	}
	if resp.Created != 4 {
		t.Fatalf("expected 4 created placeholders, got %d", resp.Created)
	}

	var count int64
	for _, locale := range []string{"en-US", "ja-JP", "ko-KR", "fr-FR"} {
		if err := db.Model(&SystemI18n{}).Where("`key` = ? AND locale = ?", "i18n.fill.zh_only", locale).Count(&count).Error; err != nil {
			t.Fatalf("count filled row %s: %v", locale, err)
		}
		if count != 1 {
			t.Fatalf("expected placeholder row for %s, got %d", locale, count)
		}
	}

	for _, locale := range []string{"en-US", "ja-JP", "ko-KR", "fr-FR"} {
		if err := db.Model(&SystemI18n{}).Where("`key` = ? AND locale = ?", "i18n.fill.user.zh_only", locale).Count(&count).Error; err != nil {
			t.Fatalf("count untouched row %s: %v", locale, err)
		}
		if count != 0 {
			t.Fatalf("expected filtered fill to skip system.iam row for %s, got %d", locale, count)
		}
	}
}

func TestI18nService_FillMissingLocalesUsesBuiltinValue(t *testing.T) {
	db := newI18nTestDB(t)
	service := NewI18nService(db)
	if err := service.Migrate(); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	if err := service.BatchInsert([]SystemI18n{
		{Module: "system.auth", Group: "menu", Key: "system.menu.session", Locale: "zh-CN", Value: "会话管理"},
	}); err != nil {
		t.Fatalf("seed items: %v", err)
	}

	resp, err := service.FillMissingLocales("system.auth")
	if err != nil {
		t.Fatalf("fill missing locales: %v", err)
	}
	if resp.Created == 0 {
		t.Fatalf("expected created rows")
	}

	var row SystemI18n
	if err := db.Where("`key` = ? AND locale = ?", "system.menu.session", "en-US").First(&row).Error; err != nil {
		t.Fatalf("load hydrated locale: %v", err)
	}
	if row.Value != "Session Management" {
		t.Fatalf("expected builtin value, got %q", row.Value)
	}
}

func TestI18nService_GetOverviewUsesBuiltinCoverage(t *testing.T) {
	db := newI18nTestDB(t)
	service := NewI18nService(db)
	if err := service.Migrate(); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	if err := service.BatchInsert([]SystemI18n{
		{Module: "system.auth", Group: "menu", Key: "system.menu.session", Locale: "zh-CN", Value: "会话管理"},
		{Module: "system.auth", Group: "menu", Key: "system.menu.session", Locale: "en-US", Value: "[system.menu.session]"},
	}); err != nil {
		t.Fatalf("seed items: %v", err)
	}

	overview, err := service.GetOverview()
	if err != nil {
		t.Fatalf("get overview: %v", err)
	}
	if overview.MissingValueCount != 0 {
		t.Fatalf("expected builtin fallback to clear missing value count, got %d", overview.MissingValueCount)
	}
	if overview.MissingLocaleCount != 0 {
		t.Fatalf("expected builtin coverage to clear missing locale count, got %d", overview.MissingLocaleCount)
	}
}

func TestI18nService_HydrateBuiltinLocales(t *testing.T) {
	db := newI18nTestDB(t)
	service := NewI18nService(db)
	if err := service.Migrate(); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	if err := service.BatchInsert([]SystemI18n{
		{Module: "system.auth", Group: "menu", Key: "system.menu.session", Locale: "zh-CN", Value: "会话管理"},
		{Module: "system.auth", Group: "menu", Key: "system.menu.session", Locale: "en-US", Value: "[system.menu.session]"},
	}); err != nil {
		t.Fatalf("seed items: %v", err)
	}

	resp, err := service.HydrateBuiltinLocales("system.auth")
	if err != nil {
		t.Fatalf("hydrate builtin locales: %v", err)
	}
	if resp.Updated == 0 || resp.Created == 0 {
		t.Fatalf("expected updated and created rows, got %+v", resp)
	}

	var enRow SystemI18n
	if err := db.Where("`key` = ? AND locale = ?", "system.menu.session", "en-US").First(&enRow).Error; err != nil {
		t.Fatalf("load en row: %v", err)
	}
	if enRow.Value != "Session Management" {
		t.Fatalf("expected hydrated en value, got %q", enRow.Value)
	}

	var frRow SystemI18n
	if err := db.Where("`key` = ? AND locale = ?", "system.menu.session", "fr-FR").First(&frRow).Error; err != nil {
		t.Fatalf("load fr row: %v", err)
	}
	if frRow.Value != "Gestion des sessions" {
		t.Fatalf("expected created fr value, got %q", frRow.Value)
	}
}

func TestI18nService_SeedI18nModuleI18nHydratesBuiltinLocales(t *testing.T) {
	db := newI18nTestDB(t)
	service := NewI18nService(db)
	if err := service.Migrate(); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	if err := service.SeedI18nModuleI18n(db); err != nil {
		t.Fatalf("seed i18n module: %v", err)
	}

	for _, locale := range []string{"zh-CN", "en-US", "ja-JP", "ko-KR", "fr-FR"} {
		var row SystemI18n
		if err := db.Where("`key` = ? AND locale = ?", "system.menu.session", locale).First(&row).Error; err != nil {
			t.Fatalf("load seeded row %s: %v", locale, err)
		}
		if strings.TrimSpace(row.Value) == "" || row.Value == "[system.menu.session]" {
			t.Fatalf("expected hydrated locale value for %s, got %q", locale, row.Value)
		}
	}
}

func TestI18nService_CreateAndReloadLocales(t *testing.T) {
	db := newI18nTestDB(t)
	service := NewI18nService(db)
	if err := service.Migrate(); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	created, err := service.Create(&I18nCreateReq{
		Module: "system.config",
		Group:  "messages",
		Key:    "i18n.qa.key",
		Locale: "zh-CN",
		Value:  "测试",
		Remark: "qa",
	})
	if err != nil {
		t.Fatalf("create i18n: %v", err)
	}
	if created.Key != "i18n.qa.key" {
		t.Fatalf("unexpected created row: %#v", created)
	}

	if err := service.ReloadLocales([]string{"zh-CN"}); err != nil {
		t.Fatalf("reload locales: %v", err)
	}
	pack, err := service.GetLangPack("zh-CN")
	if err != nil {
		t.Fatalf("get lang pack: %v", err)
	}
	if pack["i18n.qa.key"] != "测试" {
		t.Fatalf("expected reloaded pack value, got %q", pack["i18n.qa.key"])
	}
}

func TestI18nService_MigrateNormalizesLocaleKeyDuplicates(t *testing.T) {
	db := newI18nTestDB(t)
	if err := db.AutoMigrate(&SystemI18n{}); err != nil {
		t.Fatalf("auto migrate raw table: %v", err)
	}
	rows := []SystemI18n{
		{Module: "system", Group: "menu", Key: "system.menu.dept", Locale: "en-US", Value: "Department"},
		{Module: "system.org", Group: "menu", Key: "system.menu.dept", Locale: "en-US", Value: "Dept"},
		{Module: "system", Group: "menu", Key: "system.menu.session", Locale: "en-US", Value: "Session"},
		{Module: "system.auth", Group: "menu", Key: "system.menu.session", Locale: "en-US", Value: "Session"},
		{Module: "system.org", Group: "menu", Key: "system.menu.dept", Locale: "ja-JP", Value: "[system.menu.dept]"},
	}
	for _, row := range rows {
		if err := db.Create(&row).Error; err != nil {
			t.Fatalf("seed duplicate row: %v", err)
		}
	}

	service := NewI18nService(db)
	if err := service.Migrate(); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	var deptRows []SystemI18n
	if err := db.Where("`key` = ? AND locale = ?", "system.menu.dept", "en-US").Find(&deptRows).Error; err != nil {
		t.Fatalf("load dept rows: %v", err)
	}
	if len(deptRows) != 1 {
		t.Fatalf("expected 1 dept row after normalization, got %d", len(deptRows))
	}
	if deptRows[0].Module != "system.org" || deptRows[0].Value != "Department Management" {
		t.Fatalf("unexpected normalized dept row: %#v", deptRows[0])
	}

	var deptJaRows []SystemI18n
	if err := db.Where("`key` = ? AND locale = ?", "system.menu.dept", "ja-JP").Find(&deptJaRows).Error; err != nil {
		t.Fatalf("load ja dept rows: %v", err)
	}
	if len(deptJaRows) != 1 {
		t.Fatalf("expected 1 ja-JP dept row after canonical ensure, got %d", len(deptJaRows))
	}
	if deptJaRows[0].Module != "system.org" || deptJaRows[0].Value != "部門管理" {
		t.Fatalf("unexpected ja-JP dept row: %#v", deptJaRows[0])
	}

	var sessionRows []SystemI18n
	if err := db.Where("`key` = ? AND locale = ?", "system.menu.session", "en-US").Find(&sessionRows).Error; err != nil {
		t.Fatalf("load session rows: %v", err)
	}
	if len(sessionRows) != 1 {
		t.Fatalf("expected 1 session row after normalization, got %d", len(sessionRows))
	}
	if sessionRows[0].Module != "system.auth" || sessionRows[0].Value != "Session Management" {
		t.Fatalf("unexpected normalized session row: %#v", sessionRows[0])
	}
}

func TestI18nService_CreateRejectsCrossModuleLocaleKeyDuplicate(t *testing.T) {
	db := newI18nTestDB(t)
	service := NewI18nService(db)
	if err := service.Migrate(); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	if err := service.BatchInsert([]SystemI18n{
		{Module: "system.iam", Group: "menu", Key: "system.menu.access", Locale: "en-US", Value: "Access Control"},
	}); err != nil {
		t.Fatalf("seed initial row: %v", err)
	}

	_, err := service.Create(&I18nCreateReq{
		Module: "system.config",
		Group:  "menu",
		Key:    "system.menu.access",
		Locale: "en-US",
		Value:  "Platform Access",
	})
	if err == nil {
		t.Fatalf("expected duplicate rejection")
	}
	if err.Error() != "i18n.key.duplicate" {
		t.Fatalf("expected i18n.key.duplicate, got %v", err)
	}
}

func TestI18nService_ImportTemplateAndImport(t *testing.T) {
	db := newI18nTestDB(t)
	service := NewI18nService(db)
	if err := service.Migrate(); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	template := service.BuildImportTemplate()
	if template == nil || len(template.Headers) != 6 {
		t.Fatalf("unexpected template: %#v", template)
	}

	templateResult, err := service.Import(append([][]string{template.Headers}, template.Rows...))
	if err != nil {
		t.Fatalf("import template rows: %v", err)
	}
	if !templateResult.Applied || templateResult.Created < 2 {
		t.Fatalf("unexpected template import result: %#v", templateResult)
	}

	result, err := service.Import([][]string{
		{"module", "group", "key", "locale", "value", "remark"},
		{"system.config", "messages", "i18n.bulk.created", "zh-CN", "批量新增", "created"},
		{"system.config", "messages", "i18n.sample.key", "zh-CN", "示例文案已更新", "updated"},
	})
	if err != nil {
		t.Fatalf("import rows: %v", err)
	}
	if !result.Applied || result.Created != 1 || result.Updated != 1 || result.Failed != 0 {
		t.Fatalf("unexpected import result: %#v", result)
	}

	var created SystemI18n
	if err := db.Where("module = ? AND locale = ? AND `key` = ?", "system.config", "zh-CN", "i18n.bulk.created").First(&created).Error; err != nil {
		t.Fatalf("load created row: %v", err)
	}
	if created.Value != "批量新增" {
		t.Fatalf("expected created value %q, got %q", "批量新增", created.Value)
	}

	var updated SystemI18n
	if err := db.Where("module = ? AND locale = ? AND `key` = ?", "system.config", "zh-CN", "i18n.sample.key").First(&updated).Error; err != nil {
		t.Fatalf("load updated row: %v", err)
	}
	if updated.Value != "示例文案已更新" {
		t.Fatalf("expected updated value %q, got %q", "示例文案已更新", updated.Value)
	}
}

func TestI18nService_ImportBlocksCrossModuleOwnershipConflict(t *testing.T) {
	db := newI18nTestDB(t)
	service := NewI18nService(db)
	if err := service.Migrate(); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	if err := service.BatchInsert([]SystemI18n{
		{Module: "system.iam", Group: "menu", Key: "system.menu.access", Locale: "en-US", Value: "Access Control"},
	}); err != nil {
		t.Fatalf("seed initial row: %v", err)
	}

	result, err := service.Import([][]string{
		{"module", "group", "key", "locale", "value", "remark"},
		{"system.config", "menu", "system.menu.access", "en-US", "Platform Access", "translator"},
	})
	if err != nil {
		t.Fatalf("import rows: %v", err)
	}
	if result.Applied {
		t.Fatalf("expected import to be blocked, got applied result: %#v", result)
	}
	if result.Failed != 1 || len(result.Errors) != 1 {
		t.Fatalf("expected 1 import error, got %#v", result)
	}
	if result.Errors[0].Message != "import.conflict.owner.system.iam" {
		t.Fatalf("unexpected import error message: %#v", result.Errors[0])
	}

	var row SystemI18n
	if err := db.Where("locale = ? AND `key` = ?", "en-US", "system.menu.access").First(&row).Error; err != nil {
		t.Fatalf("load existing row: %v", err)
	}
	if row.Module != "system.iam" || row.Value != "Access Control" {
		t.Fatalf("expected existing row unchanged, got %#v", row)
	}
}

func TestI18nService_ImportRejectsInvalidRows(t *testing.T) {
	db := newI18nTestDB(t)
	service := NewI18nService(db)
	if err := service.Migrate(); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	result, err := service.Import([][]string{
		{"module", "group", "key", "locale", "value", "remark"},
		{"system.config", "messages", "i18n.invalid", "zh-CN", "", "missing value"},
		{"system.config", "messages", "i18n.invalid", "zh-CN", "重复", "duplicate"},
	})
	if err != nil {
		t.Fatalf("import invalid rows: %v", err)
	}
	if result.Applied {
		t.Fatalf("expected invalid import not applied")
	}
	if result.Failed == 0 {
		t.Fatalf("expected validation failures")
	}
}

func TestI18nService_GetAudit(t *testing.T) {
	db := newI18nTestDB(t)
	service := NewI18nService(db)
	if err := service.Migrate(); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	items := []SystemI18n{
		{Module: "system.config", Group: "messages", Key: "shared.audit.conflict", Locale: "zh-CN", Value: "配置冲突"},
		{Module: "system.iam", Group: "labels", Key: "shared.audit.conflict", Locale: "zh-CN", Value: "用户冲突"},
		{Module: "system.config", Group: "messages", Key: "zz.audit.unused.key", Locale: "zh-CN", Value: "未使用"},
		{Module: "system.config", Group: "messages", Key: "zz.audit.unused.key", Locale: "en-US", Value: "[zz.audit.unused.key]"},
	}
	if err := service.BatchInsert(items); err != nil {
		t.Fatalf("seed items: %v", err)
	}

	resp, err := service.GetAudit()
	if err != nil {
		t.Fatalf("get audit: %v", err)
	}
	if len(resp.DuplicateKeys) == 0 {
		t.Fatalf("expected duplicate key conflicts")
	}
	if len(resp.UnusedKeys) == 0 {
		t.Fatalf("expected unused keys")
	}

	foundDuplicate := false
	for _, item := range resp.DuplicateKeys {
		if item.Key == "shared.audit.conflict" {
			foundDuplicate = true
			if len(item.Modules) != 2 {
				t.Fatalf("expected conflict modules, got %#v", item.Modules)
			}
			if len(item.Suggestions) != 2 {
				t.Fatalf("expected conflict suggestions, got %#v", item.Suggestions)
			}
		}
	}
	if !foundDuplicate {
		t.Fatalf("expected shared.audit.conflict in duplicate audit")
	}

	foundUnused := false
	for _, item := range resp.UnusedKeys {
		if item.Key == "zz.audit.unused.key" {
			foundUnused = true
			if !containsString(item.Modules, "system.config") {
				t.Fatalf("expected unused key module system.config, got %#v", item.Modules)
			}
		}
	}
	if !foundUnused {
		t.Fatalf("expected zz.audit.unused.key in unused audit")
	}
}

func TestI18nService_GetAuditIncludesStalePlaceholders(t *testing.T) {
	db := newI18nTestDB(t)
	service := NewI18nService(db)
	if err := service.Migrate(); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	row := SystemI18n{
		Module: "system.config",
		Group:  "messages",
		Key:    "i18n.audit.stale",
		Locale: "zh-CN",
		Value:  "[i18n.audit.stale]",
	}
	if err := db.Create(&row).Error; err != nil {
		t.Fatalf("create row: %v", err)
	}
	if err := db.Model(&SystemI18n{}).Where("id = ?", row.ID).Update("updated_at", time.Now().AddDate(0, 0, -45)).Error; err != nil {
		t.Fatalf("age row: %v", err)
	}

	resp, err := service.GetAudit()
	if err != nil {
		t.Fatalf("get audit: %v", err)
	}
	if resp.StalePlaceholderThresholdDays != I18nStalePlaceholderThresholdDays {
		t.Fatalf("unexpected threshold days: %d", resp.StalePlaceholderThresholdDays)
	}
	if len(resp.StalePlaceholders) == 0 {
		t.Fatalf("expected stale placeholders")
	}
	if resp.StalePlaceholders[0].Key != "i18n.audit.stale" {
		t.Fatalf("unexpected stale placeholder: %#v", resp.StalePlaceholders[0])
	}
}

func TestI18nService_CleanupUnusedKeysByModule(t *testing.T) {
	db := newI18nTestDB(t)
	service := NewI18nService(db)
	if err := service.Migrate(); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	items := []SystemI18n{
		{Module: "system.config", Group: "messages", Key: "zz.audit.cleanup.key", Locale: "zh-CN", Value: "清理我"},
		{Module: "system.iam", Group: "messages", Key: "zz.audit.cleanup.key", Locale: "zh-CN", Value: "别删我"},
	}
	if err := service.BatchInsert(items); err != nil {
		t.Fatalf("seed items: %v", err)
	}

	resp, err := service.CleanupUnusedKeys("system.config")
	if err != nil {
		t.Fatalf("cleanup unused keys: %v", err)
	}
	if resp.Deleted != 1 {
		t.Fatalf("expected one deleted row, got %d", resp.Deleted)
	}

	var count int64
	if err := db.Model(&SystemI18n{}).Where("module = ? AND `key` = ?", "system.config", "zz.audit.cleanup.key").Count(&count).Error; err != nil {
		t.Fatalf("count cleaned row: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected cleaned module row deleted, got %d", count)
	}
	if err := db.Model(&SystemI18n{}).Where("module = ? AND `key` = ?", "system.iam", "zz.audit.cleanup.key").Count(&count).Error; err != nil {
		t.Fatalf("count preserved row: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected other module row preserved, got %d", count)
	}
}

func TestI18nService_ExportByModule(t *testing.T) {
	db := newI18nTestDB(t)
	service := NewI18nService(db)
	if err := service.Migrate(); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	items := []SystemI18n{
		{Module: "system.config", Group: "messages", Key: "i18n.export.module", Locale: "zh-CN", Value: "导出模块"},
		{Module: "system.iam", Group: "messages", Key: "i18n.export.other", Locale: "zh-CN", Value: "其他模块"},
	}
	if err := service.BatchInsert(items); err != nil {
		t.Fatalf("seed items: %v", err)
	}

	file, err := service.Export(&I18nQuery{Module: "system.config"})
	if err != nil {
		t.Fatalf("export by module: %v", err)
	}
	if len(file.Rows) != 1 {
		t.Fatalf("expected one exported row, got %d", len(file.Rows))
	}
	if file.Rows[0][0] != "system.config" {
		t.Fatalf("expected exported module system.config, got %#v", file.Rows[0])
	}
}

func TestI18nService_PreviewRenameKey(t *testing.T) {
	db := newI18nTestDB(t)
	service := NewI18nService(db)
	if err := service.Migrate(); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	items := []SystemI18n{
		{Module: "system.config", Group: "messages", Key: "common.refresh", Locale: "zh-CN", Value: "刷新"},
		{Module: "system.config", Group: "messages", Key: "common.refresh", Locale: "en-US", Value: "Refresh"},
	}
	if err := service.BatchInsert(items); err != nil {
		t.Fatalf("seed items: %v", err)
	}

	resp, err := service.PreviewRenameKey(&I18nRenamePreviewReq{
		Module: "system.config",
		OldKey: "common.refresh",
		NewKey: "system.config.common.refresh",
	})
	if err != nil {
		t.Fatalf("preview rename key: %v", err)
	}
	if resp.AffectedRows != 2 {
		t.Fatalf("expected 2 affected rows, got %d", resp.AffectedRows)
	}
	if !resp.RequiresCodeMigration {
		t.Fatalf("expected code migration requirement")
	}
	if len(resp.ReferenceFiles) == 0 {
		t.Fatalf("expected code reference files")
	}
	if resp.ReferenceFiles[0].MatchCount == 0 || len(resp.ReferenceFiles[0].Matches) == 0 {
		t.Fatalf("expected line-level reference matches, got %#v", resp.ReferenceFiles[0])
	}
}

func TestI18nService_RenameKey(t *testing.T) {
	db := newI18nTestDB(t)
	service := NewI18nService(db)
	if err := service.Migrate(); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	items := []SystemI18n{
		{Module: "system.config", Group: "messages", Key: "zz.rename.source", Locale: "zh-CN", Value: "源"},
		{Module: "system.config", Group: "messages", Key: "zz.rename.source", Locale: "en-US", Value: "Source"},
	}
	if err := service.BatchInsert(items); err != nil {
		t.Fatalf("seed items: %v", err)
	}

	resp, err := service.RenameKey(&I18nRenameExecuteReq{
		Module: "system.config",
		OldKey: "zz.rename.source",
		NewKey: "system.config.zz.rename.source",
	})
	if err != nil {
		t.Fatalf("rename key: %v", err)
	}
	if resp.RenamedRows != 2 {
		t.Fatalf("expected 2 renamed rows, got %d", resp.RenamedRows)
	}

	var count int64
	if err := db.Model(&SystemI18n{}).Where("module = ? AND `key` = ?", "system.config", "system.config.zz.rename.source").Count(&count).Error; err != nil {
		t.Fatalf("count renamed rows: %v", err)
	}
	if count != 2 {
		t.Fatalf("expected renamed key rows, got %d", count)
	}
}

func TestI18nService_RenameKeyRequiresSourceConfirmation(t *testing.T) {
	db := newI18nTestDB(t)
	service := NewI18nService(db)
	if err := service.Migrate(); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	if err := service.BatchInsert([]SystemI18n{
		{Module: "system.config", Group: "messages", Key: "common.close", Locale: "zh-CN", Value: "关闭"},
	}); err != nil {
		t.Fatalf("seed items: %v", err)
	}

	_, err := service.RenameKey(&I18nRenameExecuteReq{
		Module: "system.config",
		OldKey: "common.close",
		NewKey: "system.config.common.close",
	})
	if err == nil || err.Error() != "i18n.rename.source_not_confirmed" {
		t.Fatalf("expected source confirmation error, got %v", err)
	}
}

func TestI18nService_UnusedLifecycleFlow(t *testing.T) {
	db := newI18nTestDB(t)
	service := NewI18nService(db)
	if err := service.Migrate(); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	if err := service.BatchInsert([]SystemI18n{
		{Module: "system.config", Group: "messages", Key: "zz.lifecycle.key", Locale: "zh-CN", Value: "观察"},
		{Module: "system.config", Group: "messages", Key: "zz.lifecycle.key", Locale: "en-US", Value: "Observe"},
	}); err != nil {
		t.Fatalf("seed items: %v", err)
	}

	observeResp, err := service.StartUnusedObservation("system.config")
	if err != nil {
		t.Fatalf("start observation: %v", err)
	}
	if len(observeResp.AffectedKeys) == 0 {
		t.Fatalf("expected observation keys")
	}

	var rows []SystemI18n
	if err := db.Where("module = ? AND `key` = ?", "system.config", "zz.lifecycle.key").Find(&rows).Error; err != nil {
		t.Fatalf("load lifecycle rows: %v", err)
	}
	for _, row := range rows {
		if row.LifecycleStatus != I18nLifecycleStatusObserving {
			t.Fatalf("expected observing status, got %s", row.LifecycleStatus)
		}
	}

	if err := db.Model(&SystemI18n{}).
		Where("module = ? AND `key` = ?", "system.config", "zz.lifecycle.key").
		Update("lifecycle_marked_at", time.Now().AddDate(0, 0, -(I18nUnusedObservationThresholdDays+1))).Error; err != nil {
		t.Fatalf("age lifecycle rows: %v", err)
	}

	archiveResp, err := service.ArchiveObservedUnusedKeys("system.config")
	if err != nil {
		t.Fatalf("archive observed keys: %v", err)
	}
	if len(archiveResp.AffectedKeys) == 0 {
		t.Fatalf("expected archived keys")
	}

	if err := db.Where("module = ? AND `key` = ?", "system.config", "zz.lifecycle.key").Find(&rows).Error; err != nil {
		t.Fatalf("reload archived rows: %v", err)
	}
	for _, row := range rows {
		if row.LifecycleStatus != I18nLifecycleStatusArchived {
			t.Fatalf("expected archived status, got %s", row.LifecycleStatus)
		}
	}

	if _, err := service.DeleteArchivedUnusedKeys("system.config", false); err == nil || err.Error() != "i18n.lifecycle.delete.confirm_required" {
		t.Fatalf("expected confirm required error, got %v", err)
	}

	deleteResp, err := service.DeleteArchivedUnusedKeys("system.config", true)
	if err != nil {
		t.Fatalf("delete archived keys: %v", err)
	}
	if len(deleteResp.AffectedKeys) == 0 {
		t.Fatalf("expected deleted archived keys")
	}

	var count int64
	if err := db.Model(&SystemI18n{}).Where("module = ? AND `key` = ?", "system.config", "zz.lifecycle.key").Count(&count).Error; err != nil {
		t.Fatalf("count deleted rows: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected archived lifecycle rows deleted, got %d", count)
	}
}
