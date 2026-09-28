package config

import (
	"testing"

	"github.com/duanxldragon/pantheon-base/backend/pkg/tenant"
)

// Queue-5 settings slice: tenant override + inheritance for system_setting.
// Model: global row (tenant_id=0) is the default; a tenant row (tenant_id>0)
// overrides it; uniqueness is (tenant_id, setting_key); a tenant subject can
// never read another tenant's override and never mutates the global row.
func setupSettingTenantFixture(t *testing.T) *SettingService {
	t.Helper()
	db := setupSettingTestDB(t)
	if err := db.AutoMigrate(&SystemSetting{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if err := db.Exec("DELETE FROM system_setting").Error; err != nil {
		t.Fatalf("clear: %v", err)
	}
	return NewSettingService(db)
}

func TestSettingTenantOverride_GlobalDefaultWithoutOverride(t *testing.T) {
	svc := setupSettingTenantFixture(t)
	if err := svc.db.Create(&SystemSetting{TenantID: 0, SettingKey: "ui.default_theme", SettingValue: "indigo", ValueType: "string", GroupKey: "ui"}).Error; err != nil {
		t.Fatalf("seed global: %v", err)
	}

	bound := svc.WithTenantContext(&tenant.Context{TenantID: 101, Mode: tenant.ModeMulti})
	got, err := bound.GetByKey("ui.default_theme")
	if err != nil {
		t.Fatalf("inherit global default: %v", err)
	}
	if got != "indigo" {
		t.Fatalf("expected global default indigo, got %q", got)
	}
}

func TestSettingTenantOverride_OverrideWins(t *testing.T) {
	svc := setupSettingTenantFixture(t)
	if err := svc.db.Create(&SystemSetting{TenantID: 0, SettingKey: "ui.default_theme", SettingValue: "indigo", ValueType: "string", GroupKey: "ui"}).Error; err != nil {
		t.Fatalf("seed global: %v", err)
	}
	if err := svc.db.Create(&SystemSetting{TenantID: 101, SettingKey: "ui.default_theme", SettingValue: "dark", ValueType: "string", GroupKey: "ui"}).Error; err != nil {
		t.Fatalf("seed override: %v", err)
	}

	got, err := svc.WithTenantContext(&tenant.Context{TenantID: 101, Mode: tenant.ModeMulti}).GetByKey("ui.default_theme")
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if got != "dark" {
		t.Fatalf("tenant override must win, got %q", got)
	}

	// Tenant 202 (no override) still inherits the global default.
	got202, err := svc.WithTenantContext(&tenant.Context{TenantID: 202, Mode: tenant.ModeMulti}).GetByKey("ui.default_theme")
	if err != nil {
		t.Fatalf("resolve 202: %v", err)
	}
	if got202 != "indigo" {
		t.Fatalf("tenant 202 must inherit global, got %q", got202)
	}
}

func TestSettingTenantOverride_CannotReadForeignOverride(t *testing.T) {
	svc := setupSettingTenantFixture(t)
	if err := svc.db.Create(&SystemSetting{TenantID: 0, SettingKey: "site.name", SettingValue: "global", ValueType: "string", GroupKey: "site"}).Error; err != nil {
		t.Fatalf("seed global: %v", err)
	}
	if err := svc.db.Create(&SystemSetting{TenantID: 202, SettingKey: "site.name", SettingValue: "tenant-202-secret", ValueType: "string", GroupKey: "site"}).Error; err != nil {
		t.Fatalf("seed 202 override: %v", err)
	}

	got, err := svc.WithTenantContext(&tenant.Context{TenantID: 101, Mode: tenant.ModeMulti}).GetByKey("site.name")
	if err != nil {
		t.Fatalf("resolve as 101: %v", err)
	}
	if got == "tenant-202-secret" {
		t.Fatalf("tenant 101 must never read tenant 202's override")
	}
	if got != "global" {
		t.Fatalf("tenant 101 should fall back to global, got %q", got)
	}
}

func TestSettingTenantOverride_CompatSeesGlobalOnly(t *testing.T) {
	svc := setupSettingTenantFixture(t)
	if err := svc.db.Create(&SystemSetting{TenantID: 0, SettingKey: "site.name", SettingValue: "global", ValueType: "string", GroupKey: "site"}).Error; err != nil {
		t.Fatalf("seed global: %v", err)
	}
	if err := svc.db.Create(&SystemSetting{TenantID: 101, SettingKey: "site.name", SettingValue: "tenant-101", ValueType: "string", GroupKey: "site"}).Error; err != nil {
		t.Fatalf("seed 101 override: %v", err)
	}

	// Compat/global (unbound service): resolves global row, never an override.
	got, err := svc.GetByKey("site.name")
	if err != nil {
		t.Fatalf("compat resolve: %v", err)
	}
	if got != "global" {
		t.Fatalf("compat must resolve global row, got %q", got)
	}

	// Compat list also sees global rows only.
	rows, err := svc.List(nil)
	if err != nil {
		t.Fatalf("compat list: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("compat list must exclude tenant override rows, got %d", len(rows))
	}
}

func TestSettingTenantOverride_WriteCreatesOverrideNotMutateGlobal(t *testing.T) {
	svc := setupSettingTenantFixture(t)
	if err := svc.db.Create(&SystemSetting{TenantID: 0, SettingKey: "ui.default_theme", SettingValue: "indigo", ValueType: "string", GroupKey: "ui"}).Error; err != nil {
		t.Fatalf("seed global: %v", err)
	}

	// Tenant write: first write creates a tenant-owned override row.
	bound := svc.WithTenantContext(&tenant.Context{TenantID: 101, Mode: tenant.ModeMulti})
	if _, err := bound.UpdateGroup("ui", &SettingGroupUpdateReq{Items: []SettingUpdateItemReq{{SettingKey: "ui.default_theme", SettingValue: "slate"}}}); err != nil {
		t.Fatalf("tenant update: %v", err)
	}

	var globalRow SystemSetting
	if err := svc.db.Where("setting_key = ? AND tenant_id = 0", "ui.default_theme").First(&globalRow).Error; err != nil {
		t.Fatalf("global row must survive: %v", err)
	}
	if globalRow.SettingValue != "indigo" {
		t.Fatalf("global row must NOT be mutated by a tenant write, got %q", globalRow.SettingValue)
	}

	var overrideRow SystemSetting
	if err := svc.db.Where("setting_key = ? AND tenant_id = ?", "ui.default_theme", 101).First(&overrideRow).Error; err != nil {
		t.Fatalf("tenant override row must be created: %v", err)
	}
	if overrideRow.SettingValue != "slate" {
		t.Fatalf("override row must carry the new value, got %q", overrideRow.SettingValue)
	}

	// Second tenant write updates the override row in place (no duplicate).
	if _, err := bound.UpdateGroup("ui", &SettingGroupUpdateReq{Items: []SettingUpdateItemReq{{SettingKey: "ui.default_theme", SettingValue: "emerald"}}}); err != nil {
		t.Fatalf("second tenant update: %v", err)
	}
	var count int64
	svc.db.Model(&SystemSetting{}).Where("setting_key = ? AND tenant_id = ?", "ui.default_theme", 101).Count(&count)
	if count != 1 {
		t.Fatalf("override must update in place, found %d rows", count)
	}

	// Other tenants unaffected.
	got202, err := svc.WithTenantContext(&tenant.Context{TenantID: 202, Mode: tenant.ModeMulti}).GetByKey("ui.default_theme")
	if err != nil {
		t.Fatalf("resolve 202: %v", err)
	}
	if got202 != "indigo" {
		t.Fatalf("tenant 202 must still inherit global, got %q", got202)
	}
}

func TestSettingTenantOverride_UniquenessPerTenant(t *testing.T) {
	svc := setupSettingTenantFixture(t)
	// Same key may exist once globally and once per tenant...
	if err := svc.db.Create(&SystemSetting{TenantID: 0, SettingKey: "dup.key", SettingValue: "g", ValueType: "string", GroupKey: "x"}).Error; err != nil {
		t.Fatalf("global: %v", err)
	}
	if err := svc.db.Create(&SystemSetting{TenantID: 101, SettingKey: "dup.key", SettingValue: "t101", ValueType: "string", GroupKey: "x"}).Error; err != nil {
		t.Fatalf("tenant 101 copy: %v", err)
	}
	if err := svc.db.Create(&SystemSetting{TenantID: 202, SettingKey: "dup.key", SettingValue: "t202", ValueType: "string", GroupKey: "x"}).Error; err != nil {
		t.Fatalf("tenant 202 copy: %v", err)
	}
	// ...but a duplicate within the same tenant is rejected by the composite key.
	if err := svc.db.Create(&SystemSetting{TenantID: 101, SettingKey: "dup.key", SettingValue: "dup", ValueType: "string", GroupKey: "x"}).Error; err == nil {
		t.Fatalf("duplicate (tenant_id, setting_key) must violate the composite unique key")
	}
}

func TestSettingTenantOverride_PlatformWriteMutatesGlobalRow(t *testing.T) {
	svc := setupSettingTenantFixture(t)
	if err := svc.db.Create(&SystemSetting{TenantID: 0, SettingKey: "site.name", SettingValue: "old", ValueType: "string", GroupKey: "site"}).Error; err != nil {
		t.Fatalf("seed global: %v", err)
	}
	// Unbound service = platform write path: mutates the global row.
	if _, err := svc.UpdateGroup("site", &SettingGroupUpdateReq{Items: []SettingUpdateItemReq{{SettingKey: "site.name", SettingValue: "new-global"}}}); err != nil {
		t.Fatalf("platform update: %v", err)
	}
	var globalRow SystemSetting
	if err := svc.db.Where("setting_key = ? AND tenant_id = 0", "site.name").First(&globalRow).Error; err != nil {
		t.Fatalf("global row: %v", err)
	}
	if globalRow.SettingValue != "new-global" {
		t.Fatalf("platform write must mutate the global row, got %q", globalRow.SettingValue)
	}
}

func TestSettingTenantOverride_ListAndOverviewScoped(t *testing.T) {
	svc := setupSettingTenantFixture(t)
	seed := []SystemSetting{
		{TenantID: 0, SettingKey: "a.key", SettingValue: "g-a", ValueType: "string", GroupKey: "g1"},
		{TenantID: 0, SettingKey: "b.key", SettingValue: "g-b", ValueType: "string", GroupKey: "g1"},
		{TenantID: 101, SettingKey: "a.key", SettingValue: "t101-a", ValueType: "string", GroupKey: "g1"},
		{TenantID: 202, SettingKey: "b.key", SettingValue: "t202-b", ValueType: "string", GroupKey: "g1"},
	}
	if err := svc.db.Create(&seed).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}

	rows, err := svc.WithTenantContext(&tenant.Context{TenantID: 101, Mode: tenant.ModeMulti}).List(nil)
	if err != nil {
		t.Fatalf("tenant list: %v", err)
	}
	if len(rows) != 3 { // a.key(global) + a.key(101) + b.key(global); 202 rows excluded
		t.Fatalf("tenant 101 list must contain global + own rows only, got %d", len(rows))
	}
	for _, row := range rows {
		if row.SettingValue == "t202-b" {
			t.Fatalf("tenant 101 list leaked tenant 202's override")
		}
	}

	overview, err := svc.WithTenantContext(&tenant.Context{TenantID: 101, Mode: tenant.ModeMulti}).GetOverview()
	if err != nil {
		t.Fatalf("tenant overview: %v", err)
	}
	if overview.TotalSettingCount != 3 {
		t.Fatalf("tenant overview must be scoped, got %d", overview.TotalSettingCount)
	}
}
