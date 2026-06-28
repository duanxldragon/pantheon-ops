package login

import (
	"testing"

	"gorm.io/gorm"
	"pantheon-ops/backend/pkg/testmysql"
)

func TestSeedAuthModuleMenusReparentsLegacyFlatMenus(t *testing.T) {
	db := testmysql.Open(t)
	if err := createAuthSeedTestTables(db); err != nil {
		t.Fatalf("create tables: %v", err)
	}

	if err := db.Exec("INSERT INTO system_role (id, role_key, status) VALUES (1, 'admin', 1)").Error; err != nil {
		t.Fatalf("seed admin role: %v", err)
	}
	if err := db.Exec(`
INSERT INTO system_menu (id, parent_id, title_key, path, component, page_perm, perms, type, icon, route_name, module, sort, is_visible, is_cache, is_external, active_menu)
VALUES
(5, 0, 'system.menu.security', '/system/security', '', '', '', 'M', 'safe', 'system-security', 'system.auth', 50, 1, 0, 0, ''),
(40, 0, 'system.menu.loginLog', '/system/login-log', 'auth/LoginLogList', 'system:login-log:list', '', 'C', 'safe', 'system-login-log', 'system.auth', 10, 1, 0, 0, ''),
(41, 0, 'system.menu.session', '/system/session', 'auth/SessionList', 'system:session:list', '', 'C', 'safe', 'system-session', 'system.auth', 20, 1, 0, 0, '')
`).Error; err != nil {
		t.Fatalf("seed legacy auth menus: %v", err)
	}

	if err := SeedAuthModuleMenus(db); err != nil {
		t.Fatalf("seed auth menus: %v", err)
	}

	assertAuthMenuParent(t, db, "/system/login-log", "/system/security")
	assertAuthMenuParent(t, db, "/system/session", "/system/security")
	assertAdminAuthMenuBound(t, db, "/system/login-log")
	assertAdminAuthMenuBound(t, db, "/system/session")
}

func createAuthSeedTestTables(db *gorm.DB) error {
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
	active_menu VARCHAR(255) DEFAULT ''
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
	return db.Exec(`
CREATE TABLE system_role_menu (
	role_id BIGINT NOT NULL,
	menu_id BIGINT NOT NULL,
	PRIMARY KEY (role_id, menu_id)
)`).Error
}

func assertAuthMenuParent(t *testing.T, db *gorm.DB, childPath string, parentPath string) {
	t.Helper()
	var childParentID uint64
	if err := db.Table("system_menu").Select("parent_id").Where("path = ?", childPath).Limit(1).Pluck("parent_id", &childParentID).Error; err != nil {
		t.Fatalf("load child %s: %v", childPath, err)
	}
	var parentID uint64
	if err := db.Table("system_menu").Select("id").Where("path = ?", parentPath).Limit(1).Pluck("id", &parentID).Error; err != nil {
		t.Fatalf("load parent %s: %v", parentPath, err)
	}
	if parentID == 0 {
		t.Fatalf("expected parent %s to exist", parentPath)
	}
	if childParentID != parentID {
		t.Fatalf("expected %s parent %d, got %d", childPath, parentID, childParentID)
	}
}

func assertAdminAuthMenuBound(t *testing.T, db *gorm.DB, menuPath string) {
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
