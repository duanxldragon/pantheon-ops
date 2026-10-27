package setting

import "testing"

// The seed list the application actually uses comes from seed_data.yaml; the Go
// slice is only a fallback for a missing or malformed YAML file. Two
// declarations of the same key that disagree are therefore invisible at runtime
// — the YAML wins, and the Go copy silently documents something else.
//
// That is exactly how platform.tenant_mode became non-public: the YAML entry was
// added later with isPublic: 0 while the Go fallback said 1, so every deployment
// seeded from that commit created the row non-public. The frontend reads
// GET /system/setting/public before sign-in and gates the whole multi-mode
// surface on that key, so the tenant picker, the tenant switcher and the tenant
// viewpoint switcher stayed hidden even after the mode was switched to multi.
func TestSettingSeedYAMLMatchesGoFallback(t *testing.T) {
	yamlSeeds := make(map[string]defaultSettingSeed, len(loadedSettingSeeds))
	for _, item := range loadedSettingSeeds {
		yamlSeeds[item.SettingKey] = item
	}
	goSeeds := make(map[string]defaultSettingSeed, len(defaultSettingSeeds))
	for _, item := range defaultSettingSeeds {
		goSeeds[item.SettingKey] = item
	}

	if len(yamlSeeds) != len(goSeeds) {
		t.Fatalf("seed lists disagree on size: yaml=%d go-fallback=%d", len(yamlSeeds), len(goSeeds))
	}
	for key, want := range goSeeds {
		got, ok := yamlSeeds[key]
		if !ok {
			t.Fatalf("seed key %q is declared in the Go fallback but missing from seed_data.yaml", key)
		}
		if got.IsPublic != want.IsPublic {
			t.Fatalf("seed key %q: yaml isPublic=%d but the Go fallback says %d", key, got.IsPublic, want.IsPublic)
		}
		if got.SettingValue != want.SettingValue {
			t.Fatalf("seed key %q: yaml value %q but the Go fallback says %q", key, got.SettingValue, want.SettingValue)
		}
		if got.GroupKey != want.GroupKey || got.Module != want.Module {
			t.Fatalf("seed key %q: yaml group/module %q/%q but the Go fallback says %q/%q",
				key, got.GroupKey, got.Module, want.GroupKey, want.Module)
		}
	}
}

// The deployment mode must reach unauthenticated callers: the sign-in page has
// to know whether to render the tenant picker, and the shell has to know whether
// to render the tenant controls at all.
func TestSettingSeedDeclaresTenantModePublic(t *testing.T) {
	for _, item := range settingSeeds() {
		if item.SettingKey != settingKeyPlatformTenantMode {
			continue
		}
		if item.IsPublic != 1 {
			t.Fatalf("%s must be public, got isPublic=%d", settingKeyPlatformTenantMode, item.IsPublic)
		}
		return
	}
	t.Fatalf("%s is not declared in the seed list", settingKeyPlatformTenantMode)
}

// bootstrapSettingSeed only inserts missing keys, so a row created by an older
// seed keeps its original is_public forever. Bootstrap therefore has to repair
// it, otherwise an upgraded deployment stays in the broken state the YAML fix
// was meant to leave behind.
func TestSettingBootstrapRepairsNonPublicTenantMode(t *testing.T) {
	db := setupSettingTestDB(t)
	service := NewSettingService(db)
	if err := service.Migrate(); err != nil {
		t.Fatalf("migrate setting: %v", err)
	}

	// Simulate the row an older seed produced: present, correct value, hidden.
	if err := db.Model(&SystemSetting{}).
		Where("tenant_id = ? AND setting_key = ?", 0, settingKeyPlatformTenantMode).
		Update("is_public", 0).Error; err != nil {
		t.Fatalf("hide tenant mode setting: %v", err)
	}

	if err := service.Bootstrap(); err != nil {
		t.Fatalf("re-bootstrap settings: %v", err)
	}

	publicSettings, err := service.GetPublicSettings()
	if err != nil {
		t.Fatalf("load public settings: %v", err)
	}
	if got := publicSettings.Settings[settingKeyPlatformTenantMode]; got != "compat" {
		t.Fatalf("expected %s=compat in the public payload, got %q (payload: %+v)",
			settingKeyPlatformTenantMode, got, publicSettings.Settings)
	}
}
