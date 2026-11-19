package service

import (
	"errors"
	"strings"
	"testing"
	"time"

	mposmenu "github.com/duanxldragon/pantheon-base/backend/modules/system/iam/menu"
	"github.com/duanxldragon/pantheon-base/backend/modules/system/iam/role"
	"github.com/duanxldragon/pantheon-base/backend/pkg/testmysql"
	"gorm.io/gorm"
)

// insertLegacyMenu writes one system_menu row and its fk-less bindings so the
// fixture below can reproduce the exact upgraded-old state (id159 /business/service).
func insertLegacyMenu(t *testing.T, db *gorm.DB, id, parent uint64, path, module, mtype, titleKey string) {
	t.Helper()
	now := time.Now()
	if err := db.Exec("INSERT INTO system_menu (id, parent_id, title_key, path, component, page_perm, perms, type, icon, route_name, module, sort, is_visible, is_cache, created_at, updated_at) VALUES (?, ?, ?, ?, '', '', '', ?, '', '', ?, 0, 1, 0, ?, ?)",
		id, parent, titleKey, path, mtype, module, now, now).Error; err != nil {
		t.Fatalf("insert menu %d: %v", id, err)
	}
}

func insertRoleMenu(t *testing.T, db *gorm.DB, tenant, roleID, menuID uint64) {
	t.Helper()
	if err := db.Exec("INSERT INTO system_role_menu (tenant_id, role_id, menu_id) VALUES (?, ?, ?)", tenant, roleID, menuID).Error; err != nil {
		t.Fatalf("insert role_menu (%d,%d,%d): %v", tenant, roleID, menuID, err)
	}
}

func countRows(t *testing.T, db *gorm.DB, table, where string, args ...any) int64 {
	t.Helper()
	var n int64
	if err := db.Table(table).Where(where, args...).Count(&n).Error; err != nil {
		t.Fatalf("count %s %s: %v", table, where, err)
	}
	return n
}

func fixtureUpgradedOldMenuDB(t *testing.T) *gorm.DB {
	t.Helper()
	db := testmysql.Open(t)
	if err := db.AutoMigrate(&mposmenu.SystemMenu{}, &role.SystemRoleMenu{}); err != nil {
		t.Fatalf("create menu tables: %v", err)
	}
	// Upgraded legacy + current duplicate parents.
	insertLegacyMenu(t, db, 159, 0, "/business/service", serviceModuleKey, "C", menuTitleKeyService)
	insertLegacyMenu(t, db, 1818, 0, "/business/service-catalog", serviceModuleKey, "C", menuTitleKeyService)
	insertLegacyMenu(t, db, 1819, 1818, "/business/service-catalog/applications", serviceModuleKey, "C", menuTitleKeyServiceApplication)
	insertLegacyMenu(t, db, 1820, 1818, "/business/service-catalog/services", serviceModuleKey, "C", menuTitleKeyServiceService)
	insertLegacyMenu(t, db, 1821, 1818, "/business/service-catalog/instances", serviceModuleKey, "C", menuTitleKeyServiceInstance)
	// A user-custom child under the canonical parent (module not owned by
	// business.service) -> merge must NOT leave it dangling under a deleted parent.
	insertLegacyMenu(t, db, 4301, 1818, "/business/custom/child", "business.custom", "C", "custom.child.menu")
	// Same legacy path but owned elsewhere (a user custom menu) -> must survive.
	insertLegacyMenu(t, db, 291, 0, "/business/service", "business.custom", "C", "custom.service.menu")
	// Bindings: legacy owns two, canonical owns one unique + one duplicate tenant.
	insertRoleMenu(t, db, 0, 10, 159)
	insertRoleMenu(t, db, 7, 20, 159)
	insertRoleMenu(t, db, 0, 30, 1818)
	insertRoleMenu(t, db, 7, 20, 1818)
	return db
}

func TestMergeLegacyServiceMenuConvergesDuplicate(t *testing.T) {
	db := fixtureUpgradedOldMenuDB(t)

	if err := seedMenus(db); err != nil {
		t.Fatalf("first seed: %v", err)
	}

	// Old id survives as the single canonical parent; the duplicate is gone.
	if n := countRows(t, db, "system_menu", "id = ?", 159); n != 1 {
		t.Fatalf("expected legacy id159 to survive, count=%d", n)
	}
	var legacyPath string
	if err := db.Table("system_menu").Where("id = ?", 159).Pluck("path", &legacyPath).Error; err != nil {
		t.Fatalf("read legacy path: %v", err)
	}
	if legacyPath != serviceMenuPathCurrent {
		t.Fatalf("expected legacy path promoted to %s, got %q", serviceMenuPathCurrent, legacyPath)
	}
	if n := countRows(t, db, "system_menu", "id = ? AND module = ?", 1818, serviceModuleKey); n != 0 {
		t.Fatalf("expected duplicate canonical id1818 removed, count=%d", n)
	}
	// Canonical children reparented onto the surviving legacy id.
	for _, child := range []uint64{1819, 1820, 1821} {
		var parent uint64
		if err := db.Table("system_menu").Where("id = ?", child).Pluck("parent_id", &parent).Error; err != nil {
			t.Fatalf("read child %d parent: %v", child, err)
		}
		if parent != 159 {
			t.Fatalf("expected child %d parent_id=159, got %d", child, parent)
		}
	}
	// Ownership boundary: a non-business.service same-path custom menu is untouched.
	var customPath string
	if err := db.Table("system_menu").Where("id = ?", 291).Pluck("path", &customPath).Error; err != nil {
		t.Fatalf("read custom menu: %v", err)
	}
	if customPath != "/business/service" {
		t.Fatalf("expected custom menu path preserved, got %q", customPath)
	}
	// A user-custom child of the deleted canonical parent is reparented onto
	// legacy (not deleted, not left dangling).
	if n := countRows(t, db, "system_menu", "id = ?", 4301); n != 1 {
		t.Fatalf("expected custom child id4301 preserved, count=%d", n)
	}
	var customChildParent uint64
	if err := db.Table("system_menu").Where("id = ?", 4301).Pluck("parent_id", &customChildParent).Error; err != nil {
		t.Fatalf("read custom child parent: %v", err)
	}
	if customChildParent != 159 {
		t.Fatalf("expected custom child reparented to 159, got parent_id=%d", customChildParent)
	}

	// Binding absorption preserved tenant_id; duplicate (tenant,role) not dup'd.
	if n := countRows(t, db, "system_role_menu", "menu_id = ?", 159); n != 3 {
		t.Fatalf("expected 3 bindings absorbed onto legacy, got %d", n)
	}
	if n := countRows(t, db, "system_role_menu", "tenant_id = ? AND role_id = ? AND menu_id = ?", 7, 20, 159); n != 1 {
		t.Fatalf("expected tenant-7 duplicate binding deduplicated, got %d", n)
	}
	if n := countRows(t, db, "system_role_menu", "tenant_id = ? AND role_id = ? AND menu_id = ?", 0, 30, 159); n != 1 {
		t.Fatalf("expected network-tenant binding preserved, got %d", n)
	}
	if n := countRows(t, db, "system_role_menu", "menu_id = ?", 1818); n != 0 {
		t.Fatalf("expected canonical bindings removed, got %d", n)
	}
}

func TestMergeLegacyServiceMenuIdempotentOverTwoSeeds(t *testing.T) {
	db := fixtureUpgradedOldMenuDB(t)

	if err := seedMenus(db); err != nil {
		t.Fatalf("first seed: %v", err)
	}
	// Seed set for module=business.service: 1 parent + 3 children + 4 perms.
	const ownedRows = 8
	if n := countRows(t, db, "system_menu", "module = ?", serviceModuleKey); n != ownedRows {
		t.Fatalf("expected %d owned rows after first seed, got %d", ownedRows, n)
	}

	if err := seedMenus(db); err != nil {
		t.Fatalf("second seed: %v", err)
	}
	// Idempotent: same row count, old id persists, duplicate never recurs.
	if n := countRows(t, db, "system_menu", "module = ?", serviceModuleKey); n != ownedRows {
		t.Fatalf("expected %d owned rows after second seed, got %d", ownedRows, n)
	}
	if n := countRows(t, db, "system_menu", "path = ? AND module = ?", serviceMenuPathCurrent, serviceModuleKey); n != 1 {
		t.Fatalf("expected exactly one canonical parent, got %d", n)
	}
	if n := countRows(t, db, "system_menu", "id = ?", 159); n != 1 {
		t.Fatalf("expected legacy id159 to persist across seeds, got %d", n)
	}
	if n := countRows(t, db, "system_menu", "path = ? AND module = ?", serviceMenuPathLegacy, serviceModuleKey); n != 0 {
		t.Fatalf("expected legacy path gone after convergence, got %d", n)
	}
	if n := countRows(t, db, "system_role_menu", "menu_id = ?", 159); n != 3 {
		t.Fatalf("expected bindings stable across seeds, got %d", n)
	}
	var legacySort int
	if err := db.Table("system_menu").Where("id = ?", 159).Pluck("sort", &legacySort).Error; err != nil {
		t.Fatalf("read legacy sort: %v", err)
	}
	if legacySort != 3 {
		t.Fatalf("expected legacy parent sort to become 3, got %d", legacySort)
	}
}

func TestMergeLegacyServiceMenuPromotesOnlyLegacy(t *testing.T) {
	db := testmysql.Open(t)
	if err := db.AutoMigrate(&mposmenu.SystemMenu{}, &role.SystemRoleMenu{}); err != nil {
		t.Fatalf("create menu tables: %v", err)
	}
	// Only the stale legacy row exists -> promoted in place, keeps current id/path source.
	insertLegacyMenu(t, db, 159, 0, "/business/service", serviceModuleKey, "C", menuTitleKeyService)
	insertRoleMenu(t, db, 7, 42, 159)

	if err := seedMenus(db); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if n := countRows(t, db, "system_menu", "id = ?", 159); n != 1 {
		t.Fatalf("expected legacy id159 promoted, got %d", n)
	}
	if n := countRows(t, db, "system_menu", "path = ? AND module = ?", serviceMenuPathCurrent, serviceModuleKey); n != 1 {
		t.Fatalf("expected promoted row on canonical path, got %d", n)
	}
	if n := countRows(t, db, "system_menu", "path = ? AND module = ?", serviceMenuPathLegacy, serviceModuleKey); n != 0 {
		t.Fatalf("expected legacy path gone, got %d", n)
	}
	if n := countRows(t, db, "system_role_menu", "menu_id = ?", 159); n != 1 {
		t.Fatalf("expected original binding preserved, got %d", n)
	}
}

// injectNextReadError installs a Row callback that fails the next Scan only. GORM
// routes Table.Scan through the Row callback (gorm:row), not the Query callback, so
// the Query processor would never observe mergeLegacyServiceMenu's initial reads.
// When the one-shot error fires the merge must surface it and the enclosing
// transaction must roll back without persisting partial merge/seed.
func injectNextReadError(t *testing.T, db *gorm.DB) {
	t.Helper()
	processors := db.Callback().Row()
	name := "ops_nav_repeat_read_error"
	armed := true
	err := processors.Before("gorm:row").
		Register(name, func(tx *gorm.DB) {
			if !armed {
				return
			}
			armed = false
			tx.AddError(errors.New("injected read error for merge scan"))
		})
	if err != nil {
		t.Fatalf("register read-error callback: %v", err)
	}
	t.Cleanup(func() { processors.Before("gorm:row").Remove(name) })
}

// injectNextWriteError installs a one-shot error on the Raw (gorm:raw) processor, so
// the FIRST Exec inside the merge's transaction — the role-menu absorption INSERT
// that immediately follows the child reparent — dies. The enclosing transaction must
// roll back every prior write (reparent UPDATE included).
func injectNextWriteError(t *testing.T, db *gorm.DB) {
	t.Helper()
	processors := db.Callback().Raw()
	name := "ops_nav_repeat_write_error"
	armed := true
	err := processors.Before("gorm:raw").
		Register(name, func(tx *gorm.DB) {
			if !armed {
				return
			}
			armed = false
			tx.AddError(errors.New("injected write error for merge"))
		})
	if err != nil {
		t.Fatalf("register write-error callback: %v", err)
	}
	t.Cleanup(func() { processors.Before("gorm:raw").Remove(name) })
}

func TestMergeLegacyServiceMenuReadFailurePropagatesAndRollsBack(t *testing.T) {
	db := fixtureUpgradedOldMenuDB(t)

	// Arm a one-shot read failure against the merge's first Scan. mergeLegacyServiceMenu
	// must surface the propagated error (its Scan must not be swallowed) and the
	// transaction must roll back, leaving the fixture byte-for-byte unchanged: no
	// reparent, no binding absorption, no canonical deletion. Calling the merge
	// directly (rather than seedMenus) keeps the injection deterministic — GORM's
	// HasTable introspect in seedMenus would otherwise consume the one-shot error.
	injectNextReadError(t, db)
	mergeErr := db.Transaction(func(tx *gorm.DB) error {
		return mergeLegacyServiceMenu(tx)
	})
	if mergeErr == nil {
		t.Fatalf("expected merge to fail when a scan read errors, got nil")
	}
	if !strings.Contains(mergeErr.Error(), "injected read error") {
		t.Fatalf("expected the injected read error to propagate, got: %v", mergeErr)
	}

	// Transaction rolled back: the fixture state is byte-for-byte unchanged.
	if n := countRows(t, db, "system_menu", "id IN (159, 1818)"); n != 2 {
		t.Fatalf("expected both parents untouched after rollback, got %d", n)
	}
	for _, child := range []uint64{1819, 1820, 1821} {
		var parent uint64
		if err := db.Table("system_menu").Where("id = ?", child).Pluck("parent_id", &parent).Error; err != nil {
			t.Fatalf("read child %d parent: %v", child, err)
		}
		if parent != 1818 {
			t.Fatalf("expected child %d still under canonical 1818 after rollback, got %d", child, parent)
		}
	}
	if n := countRows(t, db, "system_role_menu", "menu_id = ?", 1818); n != 2 {
		t.Fatalf("expected canonical bindings intact after rollback, got %d", n)
	}
	if n := countRows(t, db, "system_menu", "path = ? AND module = ?", serviceMenuPathLegacy, serviceModuleKey); n != 1 {
		t.Fatalf("expected legacy path intact after rollback, got %d", n)
	}
}

// TestMergeLegacyServiceMenuWriteFailureRollsBackWholeTx fails the SQL write that
// runs right AFTER the child reparent (the role-menu absorption INSERT). The whole
// enclosing transaction must roll back, so the reparent itself, the tenant binding,
// and both parents all stay byte-for-byte as seeded.
func TestMergeLegacyServiceMenuWriteFailureRollsBackWholeTx(t *testing.T) {
	db := fixtureUpgradedOldMenuDB(t)

	// Arm AFTER the fixture (its own Exec inserts must not trip it); the one-shot
	// fires on the merge's first Exec, i.e. the binding absorption after the reparent.
	injectNextWriteError(t, db)
	mergeErr := db.Transaction(func(tx *gorm.DB) error {
		return mergeLegacyServiceMenu(tx)
	})
	if mergeErr == nil {
		t.Fatalf("expected merge to fail when a post-reparent write errors, got nil")
	}
	if !strings.Contains(mergeErr.Error(), "injected write error") {
		t.Fatalf("expected the injected write error to propagate, got: %v", mergeErr)
	}

	// Whole transaction rolled back: custom child still under the canonical parent
	// (never reparented onto legacy), both parents retained, canonical bindings intact.
	var customChildParent uint64
	if err := db.Table("system_menu").Where("id = ?", 4301).Pluck("parent_id", &customChildParent).Error; err != nil {
		t.Fatalf("read custom child parent: %v", err)
	}
	if customChildParent != 1818 {
		t.Fatalf("expected custom child still under canonical 1818 after rollback, got %d", customChildParent)
	}
	if n := countRows(t, db, "system_menu", "id IN (159, 1818)"); n != 2 {
		t.Fatalf("expected both parents untouched after rollback, got %d", n)
	}
	if n := countRows(t, db, "system_role_menu", "menu_id = ?", 1818); n != 2 {
		t.Fatalf("expected canonical bindings intact after rollback, got %d", n)
	}
	if n := countRows(t, db, "system_menu", "path = ? AND module = ?", serviceMenuPathLegacy, serviceModuleKey); n != 1 {
		t.Fatalf("expected legacy path intact after rollback, got %d", n)
	}
}
