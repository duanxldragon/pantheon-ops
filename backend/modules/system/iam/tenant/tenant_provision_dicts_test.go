package tenant

import (
	"os"
	"strings"
	"testing"

	"github.com/duanxldragon/pantheon-base/backend/modules/system/config/dict"
	"github.com/duanxldragon/pantheon-base/backend/pkg/testmysql"
)

// TestProvisionTenantWiresDictCopy is the no-database static assertion for
// FIX-4 (B4): ProvisionTenant must invoke copyPlatformDictsToTenant after the
// settings copy, so the provisioning and the repair/re-provision entries both
// gain dictionaries without a second wiring point.
func TestProvisionTenantWiresDictCopy(t *testing.T) {
	src, err := os.ReadFile("tenant_provision.go")
	if err != nil {
		t.Fatalf("read tenant_provision.go: %v", err)
	}
	text := string(src)
	provisionStart := strings.Index(text, "func (s *Service) ProvisionTenant(")
	if provisionStart < 0 {
		t.Fatal("ProvisionTenant not found in tenant_provision.go")
	}
	provisionBody := text[provisionStart:]
	if end := strings.Index(provisionBody, "\n// provisionTenantRoles"); end < 0 {
		t.Fatal("provisionTenantRoles marker not found; update this pin")
	} else {
		provisionBody = provisionBody[:end]
	}

	settingsAt := strings.Index(provisionBody, "s.copyPlatformSettingsToTenant(tx, in.TenantID)")
	dictsAt := strings.Index(provisionBody, "s.copyPlatformDictsToTenant(tx, in.TenantID)")
	if settingsAt < 0 {
		t.Fatal("provisioning baseline changed: settings copy call not found")
	}
	if dictsAt < 0 {
		t.Fatal("ProvisionTenant must call copyPlatformDictsToTenant (FIX-4 wiring)")
	}
	if dictsAt < settingsAt {
		t.Fatal("dict copy must run after the settings copy (same copy transaction)")
	}
}

// TestCopyPlatformDictsToTenantIdempotent pins the copy semantics against a
// real MySQL (skipped without one): platform defaults are copied, an existing
// tenant-private row wins and keeps its value, deleted platform rows are not
// copied, and a second run adds nothing (retry-safe against the unique keys).
func TestCopyPlatformDictsToTenantIdempotent(t *testing.T) {
	db := testmysql.Open(t)
	if err := db.AutoMigrate(&dict.SystemDictType{}, &dict.SystemDictItem{}); err != nil {
		t.Fatalf("migrate dict tables: %v", err)
	}
	for _, table := range []string{"system_dict_type", "system_dict_item"} {
		if err := db.Exec("DELETE FROM " + table).Error; err != nil {
			t.Fatalf("clear %s: %v", table, err)
		}
	}

	// Platform defaults (tenant_id = 0).
	if err := db.Create(&dict.SystemDictType{TenantID: 0, DictCode: "system_yes_no", DictName: "Yes/No", Module: "system", Status: 1}).Error; err != nil {
		t.Fatalf("seed platform dict type: %v", err)
	}
	if err := db.Create(&dict.SystemDictType{TenantID: 0, DictCode: "gone", DictName: "Deleted", Module: "system", Status: 1}).Error; err != nil {
		t.Fatalf("seed second platform dict type: %v", err)
	}
	if err := db.Create(&[]dict.SystemDictItem{
		{TenantID: 0, DictCode: "system_yes_no", ItemLabelKey: "dict.system_yes_no.yes", ItemValue: "1", Sort: 1, Status: 1},
		{TenantID: 0, DictCode: "system_yes_no", ItemLabelKey: "dict.system_yes_no.no", ItemValue: "0", Sort: 2, Status: 1},
		{TenantID: 0, DictCode: "gone", ItemLabelKey: "dict.gone.x", ItemValue: "x", Sort: 1, Status: 1},
	}).Error; err != nil {
		t.Fatalf("seed platform dict items: %v", err)
	}
	// A deleted platform dict must not propagate (type and its items).
	if err := db.Where("dict_code = ?", "gone").Delete(&dict.SystemDictType{}).Error; err != nil {
		t.Fatalf("soft-delete platform dict: %v", err)
	}
	if err := db.Where("dict_code = ?", "gone").Delete(&dict.SystemDictItem{}).Error; err != nil {
		t.Fatalf("soft-delete platform dict items: %v", err)
	}
	// Tenant-private customization that collides with a platform key.
	if err := db.Create(&dict.SystemDictType{TenantID: 630, DictCode: "system_yes_no", DictName: "Custom Yes/No", Module: "system", Status: 1}).Error; err != nil {
		t.Fatalf("seed tenant private dict: %v", err)
	}

	svc := NewService(db)
	types, items, err := svc.copyPlatformDictsToTenant(db, 630)
	if err != nil {
		t.Fatalf("copy platform dicts: %v", err)
	}
	// The colliding tenant row already owns system_yes_no; only nothing (or
	// deleted-excluded rows) may be inserted. `gone` is soft-deleted upstream.
	if types != 0 {
		t.Fatalf("dict type copy must skip tenant-owned keys, inserted %d", types)
	}
	if items != 2 {
		t.Fatalf("expected 2 platform items copied for system_yes_no, got %d", items)
	}

	// Tenant customization survived the copy.
	var name string
	if err := db.Model(&dict.SystemDictType{}).
		Where("tenant_id = ? AND dict_code = ?", 630, "system_yes_no").
		Select("dict_name").Take(&name).Error; err != nil {
		t.Fatalf("read tenant dict: %v", err)
	}
	if name != "Custom Yes/No" {
		t.Fatalf("tenant-private dict must win, got %q", name)
	}

	// Second run: nothing new, no unique-key clash against the rows just
	// inserted (and none against soft-deleted dst rows either).
	if _, _, err := svc.copyPlatformDictsToTenant(db, 630); err != nil {
		t.Fatalf("re-run must stay idempotent: %v", err)
	}
	var typeCount, itemCount int64
	if err := db.Model(&dict.SystemDictType{}).Where("tenant_id = ?", 630).Count(&typeCount).Error; err != nil {
		t.Fatalf("count tenant dict types: %v", err)
	}
	if err := db.Model(&dict.SystemDictItem{}).Where("tenant_id = ?", 630).Count(&itemCount).Error; err != nil {
		t.Fatalf("count tenant dict items: %v", err)
	}
	if typeCount != 1 || itemCount != 2 {
		t.Fatalf("idempotency broken: types=%d items=%d, want 1/2", typeCount, itemCount)
	}

	// tenantID == 0 must never copy (platform population is the source).
	if types, items, err := svc.copyPlatformDictsToTenant(db, 0); err != nil || types != 0 || items != 0 {
		t.Fatalf("copy to tenant 0 must be a no-op, got (%d, %d, %v)", types, items, err)
	}
}
