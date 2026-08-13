package system

import (
	"testing"

	"gorm.io/gorm"
	"pantheon-base/pkg/testmysql"
)

func TestEnsureMenuSeedsReparentsLegacyFlatMenus(t *testing.T) {
	db := testmysql.Open(t)
	if err := createSeedTestTables(db); err != nil {
		t.Fatalf("create tables: %v", err)
	}

	if err := db.Exec("INSERT INTO system_role (id, role_key, status) VALUES (1, 'admin', 1)").Error; err != nil {
		t.Fatalf("seed admin role: %v", err)
	}
	if err := db.Exec(`
INSERT INTO system_menu (id, parent_id, title_key, path, component, page_perm, perms, type, icon, route_name, module, sort, is_visible, is_cache, is_external, active_menu, hide_in_nav)
VALUES
(100, 0, 'system.menu.user', '/system/user', 'system/user/UserList', 'system:user:list', '', 'C', 'user', 'system-user', 'system', 2, 1, 0, 0, '', 0),
(101, 0, 'system.menu.dict', '/system/dict', 'system/dict/DictPage', 'system:dict:list', '', 'C', 'list', 'system-dict', 'system', 11, 1, 1, 0, '', 0),
(102, 0, 'system.menu.modules', '/system/modules', 'system/dynamicmodule/ModuleManager', 'system:module:list', '', 'C', 'apps', 'system-modules', 'system', 12, 1, 0, 0, '', 0),
(103, 0, 'system.menu.generator', '/system/generator', 'system/generator/ModuleWizard', 'system:generator:use', '', 'C', 'code', 'system-generator', 'system', 13, 1, 0, 0, '', 0)
`).Error; err != nil {
		t.Fatalf("seed legacy menus: %v", err)
	}

	seeds := append([]menuSeed{}, baseMenuGroupSeeds()...)
	seeds = append(seeds, coreMenuSeeds()...)
	seeds = append(seeds, dictMenuSeeds()...)
	seeds = append(seeds, platformToolMenuSeeds()...)
	if err := ensureMenuSeeds(db, seeds); err != nil {
		t.Fatalf("ensure seeds: %v", err)
	}

	assertMenuParent(t, db, "/system/user", "/system/access", "system.iam")
	assertMenuParent(t, db, "/system/dict", "/system/config", "system.config")
	assertMenuParent(t, db, "/system/modules", "/system/lowcode", "system.lowcode")
	assertMenuParent(t, db, "/system/generator", "/system/lowcode", "system.lowcode")
	assertAdminMenuBound(t, db, "/system/access")
	assertAdminMenuBound(t, db, "/system/user")
	assertAdminMenuBound(t, db, "/system/lowcode")
}

func TestNormalizeSeedMenuTypePreservesDirectoryMarker(t *testing.T) {
	if got := normalizeSeedMenuType("D"); got != "D" {
		t.Fatalf("expected D to be preserved, got %s", got)
	}
}

func TestCoreMenuSeedsExcludeObsoletePlatformContainers(t *testing.T) {
	for _, seed := range coreMenuSeeds() {
		switch seed.Path {
		case "/workspace", "/operations":
			t.Fatalf("expected obsolete platform container %s to be excluded from core menu seeds", seed.Path)
		}
		if seed.TitleKey == "app.workspace" || seed.TitleKey == "operations.menu" {
			t.Fatalf("expected obsolete platform container title %s to be excluded from core menu seeds", seed.TitleKey)
		}
	}

	var dashboard menuSeed
	for _, seed := range coreMenuSeeds() {
		if seed.Path == "/dashboard" {
			dashboard = seed
			break
		}
	}
	if dashboard.Path == "" {
		t.Fatalf("expected dashboard seed to remain")
	}
	if dashboard.ParentKey != "" {
		t.Fatalf("expected dashboard seed to be root, got parent key %s", dashboard.ParentKey)
	}
}

func TestOrgAccessControlSeedContract(t *testing.T) {
	db := testmysql.Open(t)
	if err := createSeedTestTables(db); err != nil {
		t.Fatalf("create tables: %v", err)
	}
	if err := db.Exec("INSERT INTO system_role (id, role_key, status) VALUES (1, 'admin', 1)").Error; err != nil {
		t.Fatalf("seed admin role: %v", err)
	}

	seeds := append([]menuSeed{}, baseMenuGroupSeeds()...)
	seeds = append(seeds, coreMenuSeeds()...)
	seeds = append(seeds, deptMenuSeeds()...)
	seeds = append(seeds, postMenuSeeds()...)
	seeds = append(seeds, permissionMenuSeeds()...)
	if err := ensureMenuSeeds(db, seeds); err != nil {
		t.Fatalf("ensure seeds: %v", err)
	}

	assertPageMenuContract(t, db, "/system/dept", "/system/org", "system.org", "system:dept:list", "system/dept/DeptList")
	assertPageMenuContract(t, db, "/system/post", "/system/org", "system.org", "system:post:list", "system/post/PostList")
	assertPageMenuContract(t, db, "/system/user", "/system/access", "system.iam", "system:user:list", "system/user/UserList")

	assertActionPermissionContract(t, db, "system:post:create", "/system/post")
	assertActionPermissionContract(t, db, "system:post:update", "/system/post")
	assertActionPermissionContract(t, db, "system:post:delete", "/system/post")
	assertActionPermissionContract(t, db, "system:dept:batch-update", "/system/dept")
	assertActionPermissionContract(t, db, "system:user:view", "/system/user")
}

func TestEnsureMenuSeedsCleansObsoleteMenuMatrixEntries(t *testing.T) {
	db := testmysql.Open(t)
	if err := createSeedTestTables(db); err != nil {
		t.Fatalf("create tables: %v", err)
	}
	if err := db.Exec("INSERT INTO system_role (id, role_key, status) VALUES (1, 'admin', 1)").Error; err != nil {
		t.Fatalf("seed admin role: %v", err)
	}
	if err := db.Exec(`
INSERT INTO system_menu (id, parent_id, title_key, path, component, page_perm, perms, type, icon, route_name, module, sort, is_visible, is_cache, is_external, active_menu, hide_in_nav)
VALUES
(200, 0, 'system.menu-matrix', '/system/menu-matrix', 'system/menu/MenuMatrix', 'system:menu:matrix', '', 'C', 'menu', 'system-menu-matrix', 'system.iam', 99, 1, 0, 0, '', 0),
(201, 200, 'system.permission.menu.matrix.export', '', '', '', 'system:menu:matrix', 'F', '', '', 'system.iam', 1, 1, 0, 0, '', 0)
`).Error; err != nil {
		t.Fatalf("seed obsolete menu matrix: %v", err)
	}
	if err := db.Exec("INSERT INTO system_role_menu (role_id, menu_id) VALUES (1, 200), (1, 201)").Error; err != nil {
		t.Fatalf("seed obsolete role menu bindings: %v", err)
	}
	if err := db.Exec("INSERT INTO system_role_permission (role_id, permission_key) VALUES (1, 'system:menu:matrix')").Error; err != nil {
		t.Fatalf("seed obsolete role permission binding: %v", err)
	}

	seeds := append([]menuSeed{}, baseMenuGroupSeeds()...)
	seeds = append(seeds, coreMenuSeeds()...)
	if err := ensureMenuSeeds(db, seeds); err != nil {
		t.Fatalf("ensure seeds: %v", err)
	}

	var menuCount int64
	if err := db.Table("system_menu").
		Where("path = ? OR title_key = ? OR route_name = ?", "/system/menu-matrix", "system.menu-matrix", "system-menu-matrix").
		Count(&menuCount).Error; err != nil {
		t.Fatalf("count obsolete menu matrix records: %v", err)
	}
	if menuCount != 0 {
		t.Fatalf("expected obsolete menu matrix records to be removed, got %d", menuCount)
	}

	var roleMenuCount int64
	if err := db.Table("system_role_menu").Where("menu_id IN ?", []int{200, 201}).Count(&roleMenuCount).Error; err != nil {
		t.Fatalf("count obsolete role menu bindings: %v", err)
	}
	if roleMenuCount != 0 {
		t.Fatalf("expected obsolete role menu bindings to be removed, got %d", roleMenuCount)
	}

	var rolePermCount int64
	if err := db.Table("system_role_permission").Where("permission_key = ?", "system:menu:matrix").Count(&rolePermCount).Error; err != nil {
		t.Fatalf("count obsolete role permission binding: %v", err)
	}
	if rolePermCount != 0 {
		t.Fatalf("expected obsolete role permission binding to be removed, got %d", rolePermCount)
	}
}

func TestEnsureMenuSeedsRemovesObsoletePlatformContainers(t *testing.T) {
	db := testmysql.Open(t)
	if err := createSeedTestTables(db); err != nil {
		t.Fatalf("create tables: %v", err)
	}
	if err := db.Exec("INSERT INTO system_role (id, role_key, status) VALUES (1, 'admin', 1)").Error; err != nil {
		t.Fatalf("seed admin role: %v", err)
	}
	if err := db.Exec(`
INSERT INTO system_menu (id, parent_id, title_key, path, component, page_perm, perms, type, icon, route_name, module, sort, is_visible, is_cache, is_external, active_menu, hide_in_nav)
VALUES
(300, 0, 'app.workspace', '/workspace', '', '', '', 'D', 'dashboard', 'workspace', 'platform', 10, 1, 0, 0, '', 1),
(301, 300, 'system.menu.dashboard', '/dashboard', 'dashboard', 'platform:dashboard:view', '', 'C', 'dashboard', 'dashboard', 'platform', 1, 1, 0, 0, '', 0),
(302, 300, 'operations.menu', '/operations', '', '', '', 'M', 'desktop', 'operations', 'platform', 20, 1, 0, 0, '', 1),
(303, 302, 'business.ticket.menu', '/operations/ticket', 'business/ticket/TicketList', 'business:ticket:list', '', 'C', 'file', 'business-ticket', 'business.ticket', 1, 1, 0, 0, '', 0)
`).Error; err != nil {
		t.Fatalf("seed obsolete platform containers: %v", err)
	}
	if err := db.Exec("INSERT INTO system_role_menu (role_id, menu_id) VALUES (1, 300), (1, 301), (1, 302), (1, 303)").Error; err != nil {
		t.Fatalf("seed obsolete role menu bindings: %v", err)
	}

	seeds := append([]menuSeed{}, baseMenuGroupSeeds()...)
	seeds = append(seeds, coreMenuSeeds()...)
	if err := ensureMenuSeeds(db, seeds); err != nil {
		t.Fatalf("ensure seeds: %v", err)
	}

	var containerCount int64
	if err := db.Table("system_menu").
		Where(
			"path IN ? OR title_key IN ? OR route_name IN ?",
			[]string{"/workspace", "/operations"},
			[]string{"app.workspace", "operations.menu"},
			[]string{"workspace", "operations"},
		).
		Count(&containerCount).Error; err != nil {
		t.Fatalf("count obsolete platform containers: %v", err)
	}
	if containerCount != 0 {
		t.Fatalf("expected obsolete platform containers to be removed, got %d", containerCount)
	}

	var obsoleteBindingCount int64
	if err := db.Table("system_role_menu").Where("menu_id IN ?", []int{300, 302}).Count(&obsoleteBindingCount).Error; err != nil {
		t.Fatalf("count obsolete platform container bindings: %v", err)
	}
	if obsoleteBindingCount != 0 {
		t.Fatalf("expected obsolete platform container bindings to be removed, got %d", obsoleteBindingCount)
	}

	assertMenuRoot(t, db, "/dashboard", "platform")
	assertMenuRoot(t, db, "/operations/ticket", "business.ticket")
	assertAdminMenuBound(t, db, "/dashboard")
	assertAdminMenuBound(t, db, "/operations/ticket")
	assertAdminPermissionBound(t, db, "platform:dashboard:view")
}

func createSeedTestTables(db *gorm.DB) error {
	if err := db.Exec(`
CREATE TABLE system_menu (
	id BIGINT PRIMARY KEY AUTO_INCREMENT,
	parent_id BIGINT DEFAULT 0,
	title_key VARCHAR(128) NOT NULL,
	path VARCHAR(255) DEFAULT '',
	component VARCHAR(255) DEFAULT '',
	page_perm VARCHAR(128) DEFAULT '',
	perms VARCHAR(128) DEFAULT '',
	type VARCHAR(8) DEFAULT 'M',
	icon VARCHAR(64) DEFAULT '',
	route_name VARCHAR(128) DEFAULT '',
	module VARCHAR(64) DEFAULT 'system',
	sort INT DEFAULT 0,
	is_visible TINYINT DEFAULT 1,
	is_cache TINYINT DEFAULT 0,
	is_external TINYINT DEFAULT 0,
	active_menu VARCHAR(255) DEFAULT '',
	hide_in_nav TINYINT DEFAULT 0
)`).Error; err != nil {
		return err
	}
	if err := db.Exec(`
CREATE TABLE system_role (
	id BIGINT PRIMARY KEY,
	role_key VARCHAR(64),
	status INT
)`).Error; err != nil {
		return err
	}
	if err := db.Exec(`
CREATE TABLE system_role_menu (
	role_id BIGINT NOT NULL,
	menu_id BIGINT NOT NULL,
	PRIMARY KEY (role_id, menu_id)
)`).Error; err != nil {
		return err
	}
	return db.Exec(`
CREATE TABLE system_role_permission (
	role_id BIGINT NOT NULL,
	permission_key VARCHAR(128) NOT NULL,
	PRIMARY KEY (role_id, permission_key)
)`).Error
}

func assertMenuParent(t *testing.T, db *gorm.DB, childPath string, parentPath string, expectedModule string) {
	t.Helper()
	var child struct {
		ParentID uint64
		Module   string
	}
	if err := db.Table("system_menu").
		Select("parent_id, module").
		Where("path = ?", childPath).
		Limit(1).
		Scan(&child).Error; err != nil {
		t.Fatalf("load child %s: %v", childPath, err)
	}
	var parentID uint64
	if err := db.Table("system_menu").
		Select("id").
		Where("path = ?", parentPath).
		Limit(1).
		Pluck("id", &parentID).Error; err != nil {
		t.Fatalf("load parent %s: %v", parentPath, err)
	}
	if parentID == 0 {
		t.Fatalf("expected parent %s to exist", parentPath)
	}
	if child.ParentID != parentID {
		t.Fatalf("expected %s parent %d, got %d", childPath, parentID, child.ParentID)
	}
	if child.Module != expectedModule {
		t.Fatalf("expected %s module %s, got %s", childPath, expectedModule, child.Module)
	}
}

func assertMenuRoot(t *testing.T, db *gorm.DB, path string, expectedModule string) {
	t.Helper()
	var row struct {
		ID       uint64
		ParentID uint64
		Module   string
	}
	if err := db.Table("system_menu").
		Select("id, parent_id, module").
		Where("path = ?", path).
		Limit(1).
		Scan(&row).Error; err != nil {
		t.Fatalf("load menu %s: %v", path, err)
	}
	if row.ID == 0 {
		t.Fatalf("expected menu %s to exist", path)
	}
	if row.ParentID != 0 {
		t.Fatalf("expected %s to be a root menu, got parent %d", path, row.ParentID)
	}
	if row.Module != expectedModule {
		t.Fatalf("expected %s module %s, got %s", path, expectedModule, row.Module)
	}
}

func assertPageMenuContract(t *testing.T, db *gorm.DB, path string, parentPath string, expectedModule string, expectedPagePerm string, expectedComponent string) {
	t.Helper()
	var row struct {
		ParentID  uint64
		Module    string
		PagePerm  string
		Component string
		Type      string
	}
	if err := db.Table("system_menu").
		Select("parent_id, module, page_perm, component, type").
		Where("path = ?", path).
		Limit(1).
		Scan(&row).Error; err != nil {
		t.Fatalf("load page menu %s: %v", path, err)
	}
	var parentID uint64
	if err := db.Table("system_menu").
		Select("id").
		Where("path = ?", parentPath).
		Limit(1).
		Pluck("id", &parentID).Error; err != nil {
		t.Fatalf("load parent %s: %v", parentPath, err)
	}
	if parentID == 0 {
		t.Fatalf("expected parent %s to exist", parentPath)
	}
	if row.ParentID != parentID {
		t.Fatalf("expected %s parent %d, got %d", path, parentID, row.ParentID)
	}
	if row.Module != expectedModule {
		t.Fatalf("expected %s module %s, got %s", path, expectedModule, row.Module)
	}
	if row.PagePerm != expectedPagePerm {
		t.Fatalf("expected %s page perm %s, got %s", path, expectedPagePerm, row.PagePerm)
	}
	if row.Component != expectedComponent {
		t.Fatalf("expected %s component %s, got %s", path, expectedComponent, row.Component)
	}
	if row.Type != "C" {
		t.Fatalf("expected %s to be page menu C, got %s", path, row.Type)
	}
	assertAdminMenuBound(t, db, path)
}

func assertActionPermissionContract(t *testing.T, db *gorm.DB, perms string, parentPath string) {
	t.Helper()
	var row struct {
		ID       uint64
		ParentID uint64
		Type     string
	}
	if err := db.Table("system_menu").
		Select("id, parent_id, type").
		Where("perms = ?", perms).
		Limit(1).
		Scan(&row).Error; err != nil {
		t.Fatalf("load permission %s: %v", perms, err)
	}
	if row.ID == 0 {
		t.Fatalf("expected permission %s to exist", perms)
	}
	var parentID uint64
	if err := db.Table("system_menu").
		Select("id").
		Where("path = ?", parentPath).
		Limit(1).
		Pluck("id", &parentID).Error; err != nil {
		t.Fatalf("load parent %s: %v", parentPath, err)
	}
	if parentID == 0 {
		t.Fatalf("expected parent %s to exist", parentPath)
	}
	if row.ParentID != parentID {
		t.Fatalf("expected permission %s parent %d, got %d", perms, parentID, row.ParentID)
	}
	if row.Type != "F" {
		t.Fatalf("expected permission %s type F, got %s", perms, row.Type)
	}
	assertAdminPermissionBound(t, db, perms)
}

func assertAdminMenuBound(t *testing.T, db *gorm.DB, menuPath string) {
	t.Helper()
	var count int64
	if err := db.Table("system_role_menu").
		Joins("JOIN system_menu ON system_menu.id = system_role_menu.menu_id").
		Where("system_role_menu.role_id = ? AND system_menu.path = ?", 1, menuPath).
		Count(&count).Error; err != nil {
		t.Fatalf("count admin binding for %s: %v", menuPath, err)
	}
	if count != 1 {
		t.Fatalf("expected admin binding for %s, got %d", menuPath, count)
	}
}

func assertAdminPermissionBound(t *testing.T, db *gorm.DB, perms string) {
	t.Helper()
	var count int64
	if err := db.Table("system_role_permission").
		Where("role_id = ? AND permission_key = ?", 1, perms).
		Count(&count).Error; err != nil {
		t.Fatalf("count admin binding for %s: %v", perms, err)
	}
	if count != 1 {
		t.Fatalf("expected admin binding for %s, got %d", perms, count)
	}
	if err := db.Table("system_role_menu").
		Joins("JOIN system_menu ON system_menu.id = system_role_menu.menu_id").
		Where("system_role_menu.role_id = ? AND system_menu.perms = ?", 1, perms).
		Count(&count).Error; err != nil {
		t.Fatalf("count admin menu binding for %s: %v", perms, err)
	}
	if count != 0 {
		t.Fatalf("expected admin action permission %s to be excluded from navigation menu bindings, got %d", perms, count)
	}
}

func TestCleanupDuplicateMenusRemovesDuplicatesByPath(t *testing.T) {
	db := testmysql.Open(t)
	if err := createSeedTestTables(db); err != nil {
		t.Fatalf("create tables: %v", err)
	}
	if err := db.Exec("INSERT INTO system_role (id, role_key, status) VALUES (1, 'admin', 1)").Error; err != nil {
		t.Fatalf("seed admin role: %v", err)
	}

	// Insert duplicate menus with same path
	if err := db.Exec(`
INSERT INTO system_menu (id, parent_id, title_key, path, component, page_perm, perms, type, icon, route_name, module, sort, is_visible, is_cache, is_external, active_menu, hide_in_nav)
VALUES
(500, 0, 'system.menu.lowcode', '/system/lowcode', '', '', '', 'M', 'code', 'system-lowcode', 'system.lowcode', 45, 1, 0, 0, '', 0),
(501, 0, 'system.menu.lowcode', '/system/lowcode', '', '', '', 'M', 'code', 'system-lowcode', 'system.lowcode', 45, 1, 0, 0, '', 0),
(502, 0, 'system.menu.lowcode', '/system/lowcode', '', '', '', 'M', 'code', 'system-lowcode', 'system.lowcode', 45, 1, 0, 0, '', 0)
`).Error; err != nil {
		t.Fatalf("seed duplicate menus: %v", err)
	}

	// Run cleanup
	if err := cleanupDuplicateMenus(db); err != nil {
		t.Fatalf("cleanup duplicate menus: %v", err)
	}

	// Should only have 1 menu with path /system/lowcode
	var count int64
	if err := db.Table("system_menu").Where("path = ?", "/system/lowcode").Count(&count).Error; err != nil {
		t.Fatalf("count menus: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 menu after cleanup, got %d", count)
	}

	// Should keep the smallest id (500)
	var keepID int64
	if err := db.Table("system_menu").Where("path = ?", "/system/lowcode").Pluck("id", &keepID).Error; err != nil {
		t.Fatalf("get menu id: %v", err)
	}
	if keepID != 500 {
		t.Fatalf("expected smallest id 500 to be kept, got %d", keepID)
	}
}

func TestCleanupDuplicateMenusRemovesDuplicatesByPerms(t *testing.T) {
	db := testmysql.Open(t)
	if err := createSeedTestTables(db); err != nil {
		t.Fatalf("create tables: %v", err)
	}
	if err := db.Exec("INSERT INTO system_role (id, role_key, status) VALUES (1, 'admin', 1)").Error; err != nil {
		t.Fatalf("seed admin role: %v", err)
	}

	// Insert parent menu first
	if err := db.Exec(`
INSERT INTO system_menu (id, parent_id, title_key, path, component, page_perm, perms, type, icon, route_name, module, sort, is_visible, is_cache, is_external, active_menu, hide_in_nav)
VALUES
(600, 0, 'system.menu.user', '/system/user', 'system/user/UserList', 'system:user:list', '', 'C', 'user', 'system-user', 'system.iam', 10, 1, 0, 0, '', 0)
`).Error; err != nil {
		t.Fatalf("seed parent menu: %v", err)
	}

	// Insert duplicate menus with same perms (no path)
	if err := db.Exec(`
INSERT INTO system_menu (id, parent_id, title_key, path, component, page_perm, perms, type, icon, route_name, module, sort, is_visible, is_cache, is_external, active_menu, hide_in_nav)
VALUES
(601, 600, 'system.permission.user.view', '', '', '', 'system:user:view', 'F', '', '', 'system.iam', 1, 1, 0, 0, '', 0),
(602, 600, 'system.permission.user.view', '', '', '', 'system:user:view', 'F', '', '', 'system.iam', 1, 1, 0, 0, '', 0)
`).Error; err != nil {
		t.Fatalf("seed duplicate permission menus: %v", err)
	}

	// Run cleanup
	if err := cleanupDuplicateMenus(db); err != nil {
		t.Fatalf("cleanup duplicate menus: %v", err)
	}

	// Should only have 1 menu with perms system:user:view
	var count int64
	if err := db.Table("system_menu").Where("perms = ?", "system:user:view").Count(&count).Error; err != nil {
		t.Fatalf("count menus: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 menu after cleanup, got %d", count)
	}

	// Should keep the smallest id (601)
	var keepID int64
	if err := db.Table("system_menu").Where("perms = ?", "system:user:view").Pluck("id", &keepID).Error; err != nil {
		t.Fatalf("get menu id: %v", err)
	}
	if keepID != 601 {
		t.Fatalf("expected smallest id 601 to be kept, got %d", keepID)
	}
}

func TestCleanupDuplicateMenusIsIdempotent(t *testing.T) {
	db := testmysql.Open(t)
	if err := createSeedTestTables(db); err != nil {
		t.Fatalf("create tables: %v", err)
	}
	if err := db.Exec("INSERT INTO system_role (id, role_key, status) VALUES (1, 'admin', 1)").Error; err != nil {
		t.Fatalf("seed admin role: %v", err)
	}

	// Insert unique menus (no duplicates)
	if err := db.Exec(`
INSERT INTO system_menu (id, parent_id, title_key, path, component, page_perm, perms, type, icon, route_name, module, sort, is_visible, is_cache, is_external, active_menu, hide_in_nav)
VALUES
(700, 0, 'system.menu.config', '/system/config', '', '', '', 'M', 'settings', 'system-config', 'system.config', 50, 1, 0, 0, '', 0),
(701, 0, 'system.menu.dict', '/system/dict', 'system/dict/DictPage', 'system:dict:list', '', 'C', 'list', 'system-dict', 'system.config', 10, 1, 1, 0, '', 0)
`).Error; err != nil {
		t.Fatalf("seed menus: %v", err)
	}

	// Run cleanup twice
	if err := cleanupDuplicateMenus(db); err != nil {
		t.Fatalf("first cleanup: %v", err)
	}
	if err := cleanupDuplicateMenus(db); err != nil {
		t.Fatalf("second cleanup: %v", err)
	}

	// Should still have 2 menus
	var count int64
	if err := db.Table("system_menu").Count(&count).Error; err != nil {
		t.Fatalf("count menus: %v", err)
	}
	if count != 2 {
		t.Fatalf("expected 2 menus after idempotent cleanup, got %d", count)
	}
}
