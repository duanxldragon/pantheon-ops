package system

import (
	"reflect"
	"testing"
)

func TestNewI18nServiceWithoutDBInitializesEmptyCache(t *testing.T) {
	service := NewI18nService(nil)

	if service == nil {
		t.Fatal("expected service instance")
	}
	if service.db != nil {
		t.Fatal("expected nil db to remain nil")
	}
	if service.cache == nil {
		t.Fatal("expected cache to be initialized")
	}
	if len(service.cache) != 0 {
		t.Fatalf("expected empty cache, got %d locales", len(service.cache))
	}
}

func TestBuiltinLocaleHelpersReturnSafeCopiesAndFilterPlaceholders(t *testing.T) {
	zhPack := getBuiltinLocalePack(" zh-CN ")
	if len(zhPack) == 0 {
		t.Fatal("expected builtin zh-CN locale pack")
	}

	original := zhPack["common.username"]
	zhPack["common.username"] = "mutated"

	refetched := getBuiltinLocalePack("zh-CN")
	if refetched["common.username"] != original {
		t.Fatal("expected builtin locale pack to return a defensive copy")
	}

	value, ok := getBuiltinLocaleValue("en-US", "common.username")
	if !ok || value == "" {
		t.Fatal("expected builtin locale value for common.username")
	}

	if _, ok := getBuiltinLocaleValue("zh-CN", "missing.key"); ok {
		t.Fatal("expected missing builtin key lookup to fail")
	}
}

func TestGetLangPackUsesCacheAndBuiltinFallbackWithoutDB(t *testing.T) {
	service := NewI18nService(nil)
	service.cache["zh-CN"] = map[string]string{
		"common.username": "账号",
		"custom.only":     "仅中文",
		"custom.fallback": "中文回退",
	}
	service.cache["en-US"] = map[string]string{
		"common.username": "[common.username]",
		"custom.only":     "English only",
		"custom.fallback": "[custom.fallback]",
	}

	pack, err := service.GetLangPack("en-US")
	if err != nil {
		t.Fatalf("get lang pack: %v", err)
	}
	if pack["common.username"] != "Username" {
		t.Fatalf("expected builtin target-locale value to win, got %q", pack["common.username"])
	}
	if pack["custom.only"] != "English only" {
		t.Fatalf("expected locale-specific override, got %q", pack["custom.only"])
	}
	if pack["custom.fallback"] != "中文回退" {
		t.Fatalf("expected zh fallback for missing target-locale value, got %q", pack["custom.fallback"])
	}

	pack["custom.fallback"] = "mutated"
	again, err := service.GetLangPack("en-US")
	if err != nil {
		t.Fatalf("get lang pack again: %v", err)
	}
	if again["custom.fallback"] != "中文回退" {
		t.Fatal("expected returned lang pack to be a defensive copy")
	}
}

func TestGetRawLangPackReturnsCacheClone(t *testing.T) {
	service := NewI18nService(nil)
	service.cache["fr-FR"] = map[string]string{"common.username": "Utilisateur"}

	pack, err := service.getRawLangPack("fr-FR")
	if err != nil {
		t.Fatalf("get raw lang pack: %v", err)
	}
	pack["common.username"] = "mutated"

	if service.cache["fr-FR"]["common.username"] != "Utilisateur" {
		t.Fatal("expected cached locale pack to remain unchanged")
	}
}

// NOSONAR - parameterized nil-db guard test covering 7+ API paths; each block is a simple call+assert
func TestI18nServiceNilDBGuardPathsReturnStableShapes(t *testing.T) {
	service := NewI18nService(nil)

	locales, err := service.ListSupportedLocales()
	if err != nil {
		t.Fatalf("list supported locales: %v", err)
	}
	expectedLocales := []string{"zh-CN", "en-US", "ja-JP", "ko-KR", "fr-FR"}
	if !reflect.DeepEqual(locales, expectedLocales) {
		t.Fatalf("expected default locales %v, got %v", expectedLocales, locales)
	}

	overview, err := service.GetOverview()
	if err != nil {
		t.Fatalf("get overview: %v", err)
	}
	if !reflect.DeepEqual(overview.Locales, expectedLocales) {
		t.Fatalf("expected overview locales %v, got %v", expectedLocales, overview.Locales)
	}
	if overview.TotalEntries != 0 || overview.UniqueKeyCount != 0 || len(overview.Coverage) != 0 {
		t.Fatalf("expected empty nil-db overview, got %+v", overview)
	}

	missing, err := service.ListMissingLocales(" system.config ")
	if err != nil {
		t.Fatalf("list missing locales: %v", err)
	}
	if missing.Total != 0 || len(missing.Items) != 0 {
		t.Fatalf("expected empty missing-locales response, got %+v", missing)
	}

	filled, err := service.FillMissingLocales("system.config")
	if err != nil {
		t.Fatalf("fill missing locales: %v", err)
	}
	if filled.Created != 0 || len(filled.Locales) != 0 || len(filled.Keys) != 0 {
		t.Fatalf("expected empty fill response, got %+v", filled)
	}

	hydrated, err := service.HydrateBuiltinLocales("system.config")
	if err != nil {
		t.Fatalf("hydrate builtin locales: %v", err)
	}
	if hydrated.Created != 0 || hydrated.Updated != 0 || len(hydrated.Locales) != 0 || len(hydrated.Keys) != 0 {
		t.Fatalf("expected empty hydrate response, got %+v", hydrated)
	}

	audit, err := service.GetAudit()
	if err != nil {
		t.Fatalf("get audit: %v", err)
	}
	if len(audit.DuplicateKeys) != 0 || len(audit.UnusedKeys) != 0 || len(audit.Modules) != 0 {
		t.Fatalf("expected empty audit response, got %+v", audit)
	}
	if audit.StalePlaceholderThresholdDays != I18nStalePlaceholderThresholdDays {
		t.Fatalf("expected stale placeholder threshold %d, got %d", I18nStalePlaceholderThresholdDays, audit.StalePlaceholderThresholdDays)
	}
	if audit.UnusedObservationThresholdDays != I18nUnusedObservationThresholdDays {
		t.Fatalf("expected unused observation threshold %d, got %d", I18nUnusedObservationThresholdDays, audit.UnusedObservationThresholdDays)
	}
	if audit.ArchivedRetentionThresholdDays != I18nArchivedRetentionThresholdDays {
		t.Fatalf("expected archived retention threshold %d, got %d", I18nArchivedRetentionThresholdDays, audit.ArchivedRetentionThresholdDays)
	}
}

func TestI18nHelpersNormalizeQueriesAndCanonicalEntries(t *testing.T) {
	query := normalizeI18nQuery(&I18nQuery{
		Module:    " system.config ",
		Group:     " menu ",
		Locale:    " en-US ",
		Key:       " common.username ",
		SortBy:    " locale ",
		SortOrder: " desc ",
		Page:      0,
		PageSize:  999,
	})

	if query.Page != 1 {
		t.Fatalf("expected normalized page 1, got %d", query.Page)
	}
	if query.PageSize != 200 {
		t.Fatalf("expected capped page size 200, got %d", query.PageSize)
	}
	if query.Module != "system.config" || query.Group != "menu" || query.Locale != "en-US" || query.Key != "common.username" {
		t.Fatalf("expected trimmed query values, got %+v", query)
	}
	if query.SortBy != "locale" || query.SortOrder != "desc" {
		t.Fatalf("expected trimmed sort fields, got %+v", query)
	}

	entry, ok := canonicalEntryFor(" en-US ", " system.menu.security ")
	if !ok {
		t.Fatal("expected canonical menu entry")
	}
	if entry.Module != "system.auth" || entry.Group != "menu" || entry.Value != "Security & Audit" {
		t.Fatalf("unexpected canonical entry: %+v", entry)
	}

	if err := (&I18nService{}).ensureLocaleKeyUniqueIndex(); err != nil {
		t.Fatalf("nil-db unique index guard should no-op: %v", err)
	}
	if err := (&I18nService{}).ensureCanonicalMenuEntries(); err != nil {
		t.Fatalf("nil-db canonical entry guard should no-op: %v", err)
	}
}

func TestI18nHelpersTreatPlaceholdersAsMissingButBuiltinAsEffective(t *testing.T) {
	if !hasStoredLocaleValue("value") {
		t.Fatal("expected non-empty non-placeholder value to be stored")
	}
	if hasStoredLocaleValue(" [missing.key] ") {
		t.Fatal("expected placeholder to be treated as missing")
	}
	if hasStoredLocaleValue("   ") {
		t.Fatal("expected blank value to be treated as missing")
	}

	if !hasEffectiveLocaleValue("en-US", "common.username", "[common.username]") {
		t.Fatal("expected builtin locale fallback to count as effective")
	}
	if hasEffectiveLocaleValue("en-US", "missing.key", "[missing.key]") {
		t.Fatal("expected unknown placeholder to remain ineffective")
	}

	source := map[string]string{"a": "1"}
	cloned := cloneLangPack(source)
	cloned["a"] = "2"
	if source["a"] != "1" {
		t.Fatal("expected cloneLangPack to return a defensive copy")
	}
}
