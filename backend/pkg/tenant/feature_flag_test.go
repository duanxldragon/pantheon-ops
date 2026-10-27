package tenant

import (
	"testing"

	"github.com/duanxldragon/pantheon-base/backend/pkg/testmysql"
)

// TestNormalizeFlagValue pins the flag-value normalisation: the settings API
// stores the bare token while bootstrap historically stored the JSON-quoted
// form, and both must resolve to the same mode.
func TestNormalizeFlagValue(t *testing.T) {
	cases := map[string]string{
		"multi":    "multi",
		" multi ":  "multi",
		`"multi"`:  "multi",
		`"compat"`: "compat",
		"compat":   "compat",
		"":         "",
		`"`:        `"`,
	}
	for raw, want := range cases {
		if got := NormalizeFlagValue(raw); got != want {
			t.Errorf("NormalizeFlagValue(%q) = %q, want %q", raw, got, want)
		}
	}
}

// TestFeatureFlagSettingKeyReaderIgnoresTenantCopies is the regression test for
// the deployment-mode defect: provisioning copies platform settings into every
// tenant, so an unfiltered read of `platform.tenant_mode` could resolve from a
// tenant's copy and silently degrade the whole deployment to compat. The reader
// must always resolve the platform-global row.
func TestFeatureFlagSettingKeyReaderIgnoresTenantCopies(t *testing.T) {
	db := testmysql.Open(t)
	if err := db.Exec("CREATE TABLE system_setting (" +
		"id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY, " +
		"tenant_id BIGINT UNSIGNED NOT NULL DEFAULT 0, " +
		"setting_key VARCHAR(100) NOT NULL, " +
		"setting_value TEXT NULL)").Error; err != nil {
		t.Fatalf("create system_setting failed: %v", err)
	}

	// Platform row says multi; two tenants carry stale copies that say compat.
	if err := db.Exec("INSERT INTO system_setting (tenant_id, setting_key, setting_value) VALUES (?, ?, ?)",
		PlatformGlobalTenantID, FeatureFlagSettingKey, "multi").Error; err != nil {
		t.Fatalf("insert platform row failed: %v", err)
	}
	for _, tid := range []uint64{101, 202} {
		if err := db.Exec("INSERT INTO system_setting (tenant_id, setting_key, setting_value) VALUES (?, ?, ?)",
			tid, FeatureFlagSettingKey, "compat").Error; err != nil {
			t.Fatalf("insert tenant row failed: %v", err)
		}
	}

	if got := FeatureFlagSettingKeyReader(db); got != ModeMulti {
		t.Fatalf("FeatureFlagSettingKeyReader = %q, want %q (tenant copies must not shadow the platform flag)", got, ModeMulti)
	}

	// A JSON-quoted platform value must resolve identically.
	if err := db.Exec("UPDATE system_setting SET setting_value = ? WHERE tenant_id = ? AND setting_key = ?",
		`"multi"`, PlatformGlobalTenantID, FeatureFlagSettingKey).Error; err != nil {
		t.Fatalf("quote platform row failed: %v", err)
	}
	if got := FeatureFlagSettingKeyReader(db); got != ModeMulti {
		t.Fatalf("FeatureFlagSettingKeyReader(quoted) = %q, want %q", got, ModeMulti)
	}

	// Missing row falls back to compat (flag-off = today's behaviour).
	if err := db.Exec("DELETE FROM system_setting WHERE setting_key = ?", FeatureFlagSettingKey).Error; err != nil {
		t.Fatalf("delete rows failed: %v", err)
	}
	if got := FeatureFlagSettingKeyReader(db); got != ModeCompat {
		t.Fatalf("FeatureFlagSettingKeyReader(missing) = %q, want %q", got, ModeCompat)
	}
}
