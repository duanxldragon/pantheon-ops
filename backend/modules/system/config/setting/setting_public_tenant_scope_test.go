package config

import (
	"testing"

	"github.com/duanxldragon/pantheon-base/backend/pkg/tenant"
)

// Queue-5 settings slice + 2026-09-22 public-settings scope task: public
// settings must resolve per-tenant in multi mode and the process-local cache
// must be tenant-namespaced so tenant A/B never read each other's rows.

const (
	publicSettingKeySiteName = "site.name"
	publicSettingGlobalValue = "global-name"
	publicSettingTenant101   = "tenant-101-name"
	publicSettingTenant202   = "tenant-202-name"
)

func seedPublicSetting(t *testing.T, svc *SettingService, tenantID uint64, key, value string) {
	t.Helper()
	if err := svc.db.Create(&SystemSetting{
		TenantID:     tenantID,
		SettingKey:   key,
		SettingValue: value,
		ValueType:    "string",
		GroupKey:     "site",
		IsPublic:     1,
	}).Error; err != nil {
		t.Fatalf("seed public setting t%d %s: %v", tenantID, key, err)
	}
}

func TestPublicSettings_TenantOverridesResolvePerTenant(t *testing.T) {
	svc := setupSettingTenantFixture(t)
	seedPublicSetting(t, svc, 0, publicSettingKeySiteName, publicSettingGlobalValue)
	seedPublicSetting(t, svc, 101, publicSettingKeySiteName, publicSettingTenant101)
	seedPublicSetting(t, svc, 202, publicSettingKeySiteName, publicSettingTenant202)

	resp101, err := svc.WithTenantContext(&tenant.Context{TenantID: 101, Mode: tenant.ModeMulti}).GetPublicSettings()
	if err != nil {
		t.Fatalf("public settings 101: %v", err)
	}
	if got := resp101.Settings[publicSettingKeySiteName]; got != publicSettingTenant101 {
		t.Fatalf("tenant 101 override must win, got %q", got)
	}

	resp202, err := svc.WithTenantContext(&tenant.Context{TenantID: 202, Mode: tenant.ModeMulti}).GetPublicSettings()
	if err != nil {
		t.Fatalf("public settings 202: %v", err)
	}
	if got := resp202.Settings[publicSettingKeySiteName]; got != publicSettingTenant202 {
		t.Fatalf("tenant 202 override must win, got %q", got)
	}
}

func TestPublicSettings_TenantWithoutOverrideInheritsGlobal(t *testing.T) {
	svc := setupSettingTenantFixture(t)
	seedPublicSetting(t, svc, 0, publicSettingKeySiteName, publicSettingGlobalValue)
	seedPublicSetting(t, svc, 202, publicSettingKeySiteName, publicSettingTenant202)

	resp, err := svc.WithTenantContext(&tenant.Context{TenantID: 101, Mode: tenant.ModeMulti}).GetPublicSettings()
	if err != nil {
		t.Fatalf("public settings 101: %v", err)
	}
	if got := resp.Settings[publicSettingKeySiteName]; got != publicSettingGlobalValue {
		t.Fatalf("tenant 101 must inherit global default, got %q", got)
	}
}

func TestPublicSettings_TenantCacheIsolation(t *testing.T) {
	svc := setupSettingTenantFixture(t)
	seedPublicSetting(t, svc, 0, publicSettingKeySiteName, publicSettingGlobalValue)

	// Warm the cache for tenant 101.
	bound101 := svc.WithTenantContext(&tenant.Context{TenantID: 101, Mode: tenant.ModeMulti})
	if _, err := bound101.GetPublicSettings(); err != nil {
		t.Fatalf("warm 101: %v", err)
	}

	// Mutate the DB directly (simulating another instance/tenant write) and
	// add tenant 202's override. Tenant 101's cached view must not leak it.
	seedPublicSetting(t, svc, 202, publicSettingKeySiteName, publicSettingTenant202)

	// Tenant 202 gets its own cache entry reflecting its override.
	resp202, err := svc.WithTenantContext(&tenant.Context{TenantID: 202, Mode: tenant.ModeMulti}).GetPublicSettings()
	if err != nil {
		t.Fatalf("public settings 202: %v", err)
	}
	if got := resp202.Settings[publicSettingKeySiteName]; got != publicSettingTenant202 {
		t.Fatalf("tenant 202 cache entry must be its own view, got %q", got)
	}

	// Tenant 101's cached entry must still resolve the global default — the
	// shared (pre-task) cache would have returned tenant 202's value here.
	resp101, err := bound101.GetPublicSettings()
	if err != nil {
		t.Fatalf("cached 101: %v", err)
	}
	if resp101.Settings[publicSettingKeySiteName] == publicSettingTenant202 {
		t.Fatal("tenant 101 cache leaked tenant 202's override through the shared publicCache")
	}
	if got := resp101.Settings[publicSettingKeySiteName]; got != publicSettingGlobalValue {
		t.Fatalf("tenant 101 cached view drifted, got %q", got)
	}
}

func TestPublicSettings_CompatSeesGlobalOnly(t *testing.T) {
	svc := setupSettingTenantFixture(t)
	seedPublicSetting(t, svc, 0, publicSettingKeySiteName, publicSettingGlobalValue)
	seedPublicSetting(t, svc, 101, publicSettingKeySiteName, publicSettingTenant101)

	resp, err := svc.GetPublicSettings()
	if err != nil {
		t.Fatalf("compat public settings: %v", err)
	}
	if got := resp.Settings[publicSettingKeySiteName]; got != publicSettingGlobalValue {
		t.Fatalf("compat must see global rows only, got %q", got)
	}
}

func TestPublicSettings_InvalidateClearsAllTenantEntries(t *testing.T) {
	svc := setupSettingTenantFixture(t)
	seedPublicSetting(t, svc, 0, publicSettingKeySiteName, publicSettingGlobalValue)

	bound101 := svc.WithTenantContext(&tenant.Context{TenantID: 101, Mode: tenant.ModeMulti})
	bound202 := svc.WithTenantContext(&tenant.Context{TenantID: 202, Mode: tenant.ModeMulti})
	if _, err := bound101.GetPublicSettings(); err != nil {
		t.Fatalf("warm 101: %v", err)
	}
	if _, err := bound202.GetPublicSettings(); err != nil {
		t.Fatalf("warm 202: %v", err)
	}

	// Refresh with no groups = full invalidation; afterwards a stale tenant
	// override added directly to the DB must become visible.
	seedPublicSetting(t, svc, 101, publicSettingKeySiteName, publicSettingTenant101)
	if _, err := svc.RefreshSettingCache(nil); err != nil {
		t.Fatalf("refresh cache: %v", err)
	}
	resp, err := bound101.GetPublicSettings()
	if err != nil {
		t.Fatalf("post-invalidate 101: %v", err)
	}
	if got := resp.Settings[publicSettingKeySiteName]; got != publicSettingTenant101 {
		t.Fatalf("invalidation must clear tenant-namespaced entries, got %q", got)
	}
}
