package business

import (
	"testing"

	"pantheon-ops/backend/pkg/testmysql"

	"gorm.io/gorm"
)

func TestCleanupRetiredBusinessModules_RemovesManagedRetiredModuleMetadataWithoutDroppingBusinessTables(t *testing.T) {
	db := openRetiredModuleTestDB(t)
	mustCreateRetiredModuleGovernanceTables(t, db)
	mustCreateRetiredModuleBusinessTable(t, db, "biz_test_retired")
	mustCreateRetiredModuleBusinessTable(t, db, "biz_test_retired_sub")

	rootMenuID := mustInsertRetiredMenuRow(t, db, "/test/retired", "test.retired", "", "", "")
	subMenuID := mustInsertRetiredMenuRow(
		t,
		db,
		"/test/retired/sub",
		"test.retired.sub",
		"test/retired/sub/TestRetiredList",
		"test:retired:sub:list",
		"test:retired:sub:view",
	)
	mustInsertRetiredRoleMenu(t, db, rootMenuID)
	mustInsertRetiredRoleMenu(t, db, subMenuID)
	mustInsertRetiredPermission(t, db, "test:retired:sub:view")
	mustInsertRetiredPermission(t, db, "test:retired:other:view")
	mustInsertRetiredI18n(t, db, "test.retired")
	mustInsertRetiredI18n(t, db, "test.retired.sub")
	mustInsertRetiredScopedI18n(t, db, "system.config", "test.retired.title")
	mustInsertRetiredScopedI18n(t, db, "system.config", "test.retired.sub.detail")
	mustInsertManagedRegistration(t, db, "test.retired", "", 1)
	mustInsertManagedRegistration(t, db, "test.retired.sub", "biz_test_retired_sub", 1)

	// Temporarily set retired modules for this test
	original := retiredBusinessModules
	retiredBusinessModules = []retiredModuleSpec{{
		ModuleNames:        []string{"test.retired", "test.retired.sub"},
		PermissionPrefixes: []string{"test:retired:"},
		MenuPaths:          []string{"/test/retired", "/test/retired/sub"},
		ComponentKeys:      []string{"test/retired/sub/TestRetiredList"},
	}}
	defer func() { retiredBusinessModules = original }()

	if err := cleanupRetiredBusinessModules(db); err != nil {
		t.Fatalf("cleanup retired modules: %v", err)
	}

	assertTableExists(t, db, "biz_test_retired")
	assertTableExists(t, db, "biz_test_retired_sub")
	assertRecordCount(t, db, "system_menu", "1 = 1", 0)
	assertRecordCount(t, db, "system_role_menu", "1 = 1", 0)
	assertRecordCount(t, db, "system_role_permission", "permission_key LIKE 'test:retired:%'", 0)
	assertRecordCount(t, db, "system_i18n", "module IN ('test.retired', 'test.retired.sub')", 0)
	assertRecordCount(t, db, "system_i18n", "module = 'system.config' AND (`key` = 'test.retired.title' OR `key` = 'test.retired.sub.detail')", 0)
	assertRecordCount(t, db, "system_module_registration", "name IN ('test.retired', 'test.retired.sub')", 0)
}

func TestCleanupRetiredBusinessModules_RemovesLegacyMetadataWithoutDroppingBusinessTables(t *testing.T) {
	db := openRetiredModuleTestDB(t)
	mustCreateRetiredModuleGovernanceTables(t, db)
	mustCreateRetiredModuleBusinessTable(t, db, "biz_test_retired_legacy")

	mustInsertRetiredMenuRow(t, db, "/test/retired/legacy", "test.retired.legacy", "test/retired/legacy/TestLegacyList", "test:retired:legacy:list", "test:retired:legacy:view")
	mustInsertRetiredPermission(t, db, "test:retired:legacy:view")
	mustInsertRetiredI18n(t, db, "test.retired.legacy")
	mustInsertRetiredScopedI18n(t, db, "system.config", "test.retired.legacy.title")

	original := retiredBusinessModules
	retiredBusinessModules = []retiredModuleSpec{{
		ModuleNames:        []string{"test.retired.legacy"},
		PermissionPrefixes: []string{"test:retired:legacy:"},
		MenuPaths:          []string{"/test/retired/legacy"},
		ComponentKeys:      []string{"test/retired/legacy/TestLegacyList"},
	}}
	defer func() { retiredBusinessModules = original }()

	if err := cleanupRetiredBusinessModules(db); err != nil {
		t.Fatalf("cleanup retired modules: %v", err)
	}

	assertTableExists(t, db, "biz_test_retired_legacy")
	assertRecordCount(t, db, "system_menu", "1 = 1", 0)
	assertRecordCount(t, db, "system_role_permission", "1 = 1", 0)
	assertRecordCount(t, db, "system_i18n", "1 = 1", 0)
}

func openRetiredModuleTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	return testmysql.Open(t)
}

func mustCreateRetiredModuleGovernanceTables(t *testing.T, db *gorm.DB) {
	t.Helper()
	statements := []string{
		`CREATE TABLE system_menu (
			id BIGINT PRIMARY KEY AUTO_INCREMENT,
			path VARCHAR(255),
			type VARCHAR(8),
			module VARCHAR(64),
			component VARCHAR(255),
			page_perm VARCHAR(255),
			perms VARCHAR(255)
		)`,
		`CREATE TABLE system_role_menu (
			id BIGINT PRIMARY KEY AUTO_INCREMENT,
			role_id BIGINT,
			menu_id BIGINT
		)`,
		`CREATE TABLE system_role_permission (
			id BIGINT PRIMARY KEY AUTO_INCREMENT,
			role_id BIGINT,
			permission_key VARCHAR(255)
		)`,
		`CREATE TABLE system_i18n (
			id BIGINT PRIMARY KEY AUTO_INCREMENT,
			module VARCHAR(64),
			` + "`key`" + ` VARCHAR(255)
		)`,
		`CREATE TABLE system_module_registration (
			id BIGINT PRIMARY KEY AUTO_INCREMENT,
			name VARCHAR(64),
			table_name VARCHAR(128),
			status INT
		)`,
	}
	for _, statement := range statements {
		if err := db.Exec(statement).Error; err != nil {
			t.Fatalf("create governance table failed: %v", err)
		}
	}
}

func mustCreateRetiredModuleBusinessTable(t *testing.T, db *gorm.DB, tableName string) {
	t.Helper()
	if err := db.Exec("CREATE TABLE " + tableName + " (id BIGINT PRIMARY KEY AUTO_INCREMENT, name VARCHAR(64))").Error; err != nil {
		t.Fatalf("create business table %s: %v", tableName, err)
	}
}

func mustInsertRetiredMenuRow(t *testing.T, db *gorm.DB, path string, module string, component string, pagePerm string, perms string) uint64 {
	t.Helper()
	if err := db.Exec(
		`INSERT INTO system_menu (path, type, module, component, page_perm, perms) VALUES (?, 'C', ?, ?, ?, ?)`,
		path,
		module,
		component,
		pagePerm,
		perms,
	).Error; err != nil {
		t.Fatalf("insert retired menu: %v", err)
	}
	var menuID uint64
	if err := db.Table("system_menu").Select("id").Where("path = ?", path).Limit(1).Pluck("id", &menuID).Error; err != nil {
		t.Fatalf("lookup retired menu id: %v", err)
	}
	return menuID
}

func mustInsertRetiredRoleMenu(t *testing.T, db *gorm.DB, menuID uint64) {
	t.Helper()
	if err := db.Exec(`INSERT INTO system_role_menu (role_id, menu_id) VALUES (1, ?)`, menuID).Error; err != nil {
		t.Fatalf("insert retired role menu: %v", err)
	}
}

func mustInsertRetiredPermission(t *testing.T, db *gorm.DB, permissionKey string) {
	t.Helper()
	if err := db.Exec(`INSERT INTO system_role_permission (role_id, permission_key) VALUES (1, ?)`, permissionKey).Error; err != nil {
		t.Fatalf("insert retired permission: %v", err)
	}
}

func mustInsertRetiredI18n(t *testing.T, db *gorm.DB, module string) {
	t.Helper()
	if err := db.Exec(`INSERT INTO system_i18n (module) VALUES (?)`, module).Error; err != nil {
		t.Fatalf("insert retired i18n: %v", err)
	}
}

func mustInsertRetiredScopedI18n(t *testing.T, db *gorm.DB, module string, key string) {
	t.Helper()
	if err := db.Exec(`INSERT INTO system_i18n (module, `+"`key`"+`) VALUES (?, ?)`, module, key).Error; err != nil {
		t.Fatalf("insert retired scoped i18n: %v", err)
	}
}

func mustInsertManagedRegistration(t *testing.T, db *gorm.DB, module string, tableName string, status int) {
	t.Helper()
	if err := db.Exec(`INSERT INTO system_module_registration (name, table_name, status) VALUES (?, ?, ?)`, module, tableName, status).Error; err != nil {
		t.Fatalf("insert managed registration: %v", err)
	}
}

func assertTableExists(t *testing.T, db *gorm.DB, tableName string) {
	t.Helper()
	if !db.Migrator().HasTable(tableName) {
		t.Fatalf("expected table %s to exist", tableName)
	}
}

func assertRecordCount(t *testing.T, db *gorm.DB, tableName string, where string, expected int64) {
	t.Helper()
	var count int64
	if err := db.Table(tableName).Where(where).Count(&count).Error; err != nil {
		t.Fatalf("count %s: %v", tableName, err)
	}
	if count != expected {
		t.Fatalf("unexpected %s count: got %d want %d", tableName, count, expected)
	}
}
