package setting

import (
	"testing"

	"github.com/duanxldragon/pantheon-base/backend/pkg/tenant"
	"github.com/duanxldragon/pantheon-base/backend/pkg/testmysql"
)

// TestIsTenantModeDowngradeToCompat pins the pure decision half of the FIX-2
// downgrade guard (no database): only a multi -> compat transition on the
// platform.tenant_mode key is a downgrade.
func TestIsTenantModeDowngradeToCompat(t *testing.T) {
	cases := []struct {
		name    string
		key     string
		next    string
		current string
		want    bool
	}{
		{"multi to compat is a downgrade", settingKeyPlatformTenantMode, "compat", "multi", true},
		{"empty next normalizes to compat", settingKeyPlatformTenantMode, "", "multi", true},
		{"unknown next normalizes to compat", settingKeyPlatformTenantMode, "weird", "multi", true},
		{"compat to compat is not", settingKeyPlatformTenantMode, "compat", "compat", false},
		{"compat to multi is an upgrade", settingKeyPlatformTenantMode, "multi", "compat", false},
		{"multi to multi is not", settingKeyPlatformTenantMode, "multi", "multi", false},
		{"other key never triggers", "platform.app_mode", "compat", "multi", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := isTenantModeDowngradeToCompat(tc.key, tc.next, tc.current); got != tc.want {
				t.Fatalf("isTenantModeDowngradeToCompat(%q, %q, %q) = %v, want %v", tc.key, tc.next, tc.current, got, tc.want)
			}
		})
	}
}

// TestGuardTenantModeDowngrade counts live tenants against the `tenants`
// master table (skip without MySQL, same testmysql policy as the rest of the
// package): any non-deleted tenant row blocks the multi -> compat switch; a
// soft-deleted tenant does not.
func TestGuardTenantModeDowngrade(t *testing.T) {
	db := testmysql.Open(t)
	if err := db.AutoMigrate(&tenant.Tenant{}); err != nil {
		t.Fatalf("migrate tenants: %v", err)
	}
	if err := db.Exec("DELETE FROM tenants").Error; err != nil {
		t.Fatalf("clear tenants: %v", err)
	}

	seedTenant := func(id uint64, status string) {
		t.Helper()
		if err := db.Create(&tenant.Tenant{
			ID: id, Code: "tenant-" + uint64ToString(id), Name: "Tenant fixture", Status: status,
		}).Error; err != nil {
			t.Fatalf("seed tenant %d: %v", id, err)
		}
	}

	// No tenants at all: the downgrade is free.
	if err := guardTenantModeDowngrade(db, settingKeyPlatformTenantMode, "compat", "multi"); err != nil {
		t.Fatalf("empty tenants table must allow the downgrade, got %v", err)
	}

	seedTenant(101, tenant.TenantStatusActive)
	if err := guardTenantModeDowngrade(db, settingKeyPlatformTenantMode, "compat", "multi"); err == nil {
		t.Fatal("active tenant must block the multi -> compat downgrade")
	}

	// Non-downgrade transitions stay unguarded even with tenants present.
	if err := guardTenantModeDowngrade(db, settingKeyPlatformTenantMode, "multi", "compat"); err != nil {
		t.Fatalf("upgrade direction must not be guarded, got %v", err)
	}
	if err := guardTenantModeDowngrade(db, "platform.app_mode", "compat", "multi"); err != nil {
		t.Fatalf("other keys must not be guarded, got %v", err)
	}

	// Soft-deleted tenants no longer host sessions: they must not block.
	if err := db.Model(&tenant.Tenant{}).Where("id = ?", 101).Update("status", tenant.TenantStatusDeleted).Error; err != nil {
		t.Fatalf("soft-delete tenant: %v", err)
	}
	if err := guardTenantModeDowngrade(db, settingKeyPlatformTenantMode, "compat", "multi"); err != nil {
		t.Fatalf("only soft-deleted tenants must not block the downgrade, got %v", err)
	}

	// Suspended/archived tenants still exist (data not purged): they block.
	for _, status := range []string{tenant.TenantStatusSuspended, tenant.TenantStatusArchived} {
		if err := db.Model(&tenant.Tenant{}).Where("id = ?", 101).Update("status", status).Error; err != nil {
			t.Fatalf("set tenant status %s: %v", status, err)
		}
		if err := guardTenantModeDowngrade(db, settingKeyPlatformTenantMode, "compat", "multi"); err == nil {
			t.Fatalf("%s tenant must block the downgrade", status)
		}
	}
}

func uint64ToString(v uint64) string {
	if v == 0 {
		return "0"
	}
	digits := make([]byte, 0, 20)
	for v > 0 {
		digits = append(digits, byte('0'+v%10))
		v /= 10
	}
	for i, j := 0, len(digits)-1; i < j; i, j = i+1, j-1 {
		digits[i], digits[j] = digits[j], digits[i]
	}
	return string(digits)
}
