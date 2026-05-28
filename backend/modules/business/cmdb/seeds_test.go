package cmdb

import (
	"testing"

	"pantheon-ops/backend/pkg/testmysql"

	"gorm.io/gorm"
)

func TestSeedHostMenusCreatesCmdbPagesAndActionPermissions(t *testing.T) {
	db := testmysql.Open(t)
	mustCreateCmdbSeedTables(t, db)
	if err := db.Exec("INSERT INTO system_role (id, role_key) VALUES (1, 'admin')").Error; err != nil {
		t.Fatalf("seed admin role: %v", err)
	}

	if err := seedHostMenus(db); err != nil {
		t.Fatalf("seed cmdb menus: %v", err)
	}

	assertCmdbRecordCount(t, db, "system_menu", "path = '/operations/cmdb/host' AND page_perm = 'business:cmdb:host:list'", 1)
	assertCmdbRecordCount(t, db, "system_menu", "path = '/operations/cmdb/group' AND page_perm = 'business:cmdb:group:list'", 1)
	assertCmdbRecordCount(t, db, "system_menu", "path = '/operations/cmdb/label' AND page_perm = 'business:cmdb:label:list'", 1)
	assertCmdbRecordCount(t, db, "system_menu", "perms = 'business:cmdb:host:create' AND type = 'F'", 1)
	assertCmdbRecordCount(t, db, "system_menu", "perms = 'business:cmdb:host:collect' AND type = 'F'", 1)
	assertCmdbRecordCount(t, db, "system_menu", "perms = 'business:cmdb:group:delete' AND type = 'F'", 1)
	assertCmdbRecordCount(t, db, "system_menu", "perms = 'business:cmdb:label:delete' AND type = 'F'", 1)
	assertCmdbRecordCount(t, db, "system_role_permission", "role_id = 1 AND permission_key = 'business:cmdb:host:list'", 1)
	assertCmdbRecordCount(t, db, "system_role_permission", "role_id = 1 AND permission_key = 'business:cmdb:host:create'", 1)
	assertCmdbRecordCount(t, db, "system_role_permission", "role_id = 1 AND permission_key = 'business:cmdb:group:delete'", 1)
	assertCmdbRecordCount(t, db, "system_role_permission", "role_id = 1 AND permission_key = 'business:cmdb:label:delete'", 1)
	assertCmdbRecordCount(t, db, "system_role_menu", "role_id = 1", 5)
	assertCmdbRecordCount(t, db, "biz_cmdb_label_schema", "`key` = 'env' AND value_mode = 'dict'", 1)
	assertCmdbRecordCount(t, db, "biz_cmdb_label_schema", "`key` = 'region' AND JSON_CONTAINS(options, '\"cn-east-1\"')", 1)
	assertCmdbRecordCount(t, db, "biz_cmdb_label_schema", "`key` = 'env' AND category = 'environment'", 1)
}

func TestSeedHostMenusNormalizesLegacyLabelSchemaCategory(t *testing.T) {
	db := testmysql.Open(t)
	mustCreateCmdbSeedTables(t, db)
	if err := db.Exec("INSERT INTO system_role (id, role_key) VALUES (1, 'admin')").Error; err != nil {
		t.Fatalf("seed admin role: %v", err)
	}
	if err := db.Exec(`
INSERT INTO biz_cmdb_label_schema (` + "`key`" + `, name, category, value_mode, dict_code, options, required, status, description, created_at, updated_at)
VALUES
('legacy_blank', '历史空分类标签', '', 'free', '', JSON_ARRAY(), 0, 'enabled', 'legacy', NOW(), NOW()),
('env', '环境', '', 'enum', '', JSON_ARRAY('legacy-env'), 0, 'enabled', 'legacy env', NOW(), NOW())
`).Error; err != nil {
		t.Fatalf("seed legacy label schema: %v", err)
	}

	if err := seedHostMenus(db); err != nil {
		t.Fatalf("seed cmdb menus: %v", err)
	}

	assertCmdbRecordCount(t, db, "biz_cmdb_label_schema", "`key` = 'legacy_blank' AND category = 'base'", 1)
	assertCmdbRecordCount(t, db, "biz_cmdb_label_schema", "`key` = 'env' AND category = 'environment'", 1)
	assertCmdbRecordCount(t, db, "biz_cmdb_label_schema", "`key` = 'env' AND value_mode = 'dict'", 1)
	assertCmdbRecordCount(t, db, "biz_cmdb_label_schema", "`key` = 'env' AND JSON_CONTAINS(options, '\"dev\"')", 1)
}

func TestSeedHostMenusPrioritizesBusinessOperationsEntry(t *testing.T) {
	topSeeds := topLevelMenuSeeds()
	if len(topSeeds) == 0 || topSeeds[0].Path != "/operations/cmdb" {
		t.Fatalf("expected first top-level CMDB seed to be /operations/cmdb, got %+v", topSeeds)
	}
	if topSeeds[0].ParentKey != "operations" {
		t.Fatalf("expected /operations/cmdb to attach to operations, got parent %q", topSeeds[0].ParentKey)
	}
	if topSeeds[0].Sort != 1 {
		t.Fatalf("expected /operations/cmdb sort to be 1 under operations, got %d", topSeeds[0].Sort)
	}

	db := testmysql.Open(t)
	mustCreateCmdbSeedTables(t, db)
	if err := db.Exec("INSERT INTO system_role (id, role_key) VALUES (1, 'admin')").Error; err != nil {
		t.Fatalf("seed admin role: %v", err)
	}
	if err := db.Exec(`
INSERT INTO system_menu (parent_id, title_key, path, component, page_perm, type, icon, route_name, module, sort, is_visible)
VALUES
(0, 'app.workspace', '/workspace', '', '', 'D', 'dashboard', 'workspace', 'platform', 10, 1),
(1, 'system.menu.dashboard', '/dashboard', 'dashboard', 'platform:dashboard:view', 'C', 'dashboard', 'dashboard', 'platform', 1, 1),
(1, 'operations.menu', '/operations', '', '', 'M', 'desktop', 'operations', 'platform', 20, 1),
(0, 'system.menu.access', '/system/access', '', '', 'M', 'idcard', 'system-access', 'system.iam', 20, 1)
`).Error; err != nil {
		t.Fatalf("seed platform/system menus: %v", err)
	}

	if err := seedHostMenus(db); err != nil {
		t.Fatalf("seed cmdb menus: %v", err)
	}

	var rows []struct {
		Path string
		Sort int
	}
	if err := db.Table("system_menu").
		Select("path, sort").
		Where("parent_id = 1 AND path IN ?", []string{"/operations", "/dashboard"}).
		Order("sort asc, id asc").
		Scan(&rows).Error; err != nil {
		t.Fatalf("query top-level menu order: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("expected workspace children to include dashboard and operations, got %+v", rows)
	}
	if rows[0].Path != "/dashboard" || rows[1].Path != "/operations" {
		t.Fatalf("expected workspace child order dashboard then operations, got %+v", rows)
	}

	var cmdbRow struct {
		Path     string
		ParentID uint64
		Sort     int
	}
	if err := db.Table("system_menu").
		Select("path, parent_id, sort").
		Where("path = ?", "/operations/cmdb").
		Scan(&cmdbRow).Error; err != nil {
		t.Fatalf("query cmdb menu: %v", err)
	}
	if cmdbRow.Path != "/operations/cmdb" || cmdbRow.Sort != 1 {
		t.Fatalf("expected /operations/cmdb seeded under operations with sort 1, got %+v", cmdbRow)
	}
}

func mustCreateCmdbSeedTables(t *testing.T, db *gorm.DB) {
	t.Helper()
	statements := []string{
		`CREATE TABLE system_role (
			id BIGINT PRIMARY KEY AUTO_INCREMENT,
			role_key VARCHAR(64)
		)`,
		`CREATE TABLE system_menu (
			id BIGINT PRIMARY KEY AUTO_INCREMENT,
			parent_id BIGINT,
			title_key VARCHAR(128),
			path VARCHAR(255),
			component VARCHAR(255),
			page_perm VARCHAR(255),
			perms VARCHAR(255),
			type VARCHAR(8),
			icon VARCHAR(64),
			route_name VARCHAR(128),
			module VARCHAR(64),
			sort INT,
			is_visible INT,
			is_cache INT,
			created_at DATETIME,
			updated_at DATETIME
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
		`CREATE TABLE system_dict_type (
			id BIGINT PRIMARY KEY AUTO_INCREMENT,
			dict_code VARCHAR(64),
			dict_name VARCHAR(64),
			module VARCHAR(64),
			status INT,
			created_at DATETIME,
			updated_at DATETIME
		)`,
		`CREATE TABLE system_dict_item (
			id BIGINT PRIMARY KEY AUTO_INCREMENT,
			dict_code VARCHAR(64),
			item_label_key VARCHAR(128),
			item_value VARCHAR(64),
			sort INT,
			status INT,
			created_at DATETIME,
			updated_at DATETIME
		)`,
		`CREATE TABLE biz_cmdb_label_schema (
			id BIGINT PRIMARY KEY AUTO_INCREMENT,
			` + "`key`" + ` VARCHAR(64),
			name VARCHAR(128),
			category VARCHAR(32),
			value_mode VARCHAR(16),
			dict_code VARCHAR(64),
			options JSON,
			required BOOLEAN,
			status VARCHAR(16),
			description VARCHAR(512),
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME
		)`,
	}
	for _, statement := range statements {
		if err := db.Exec(statement).Error; err != nil {
			t.Fatalf("create seed table: %v", err)
		}
	}
}

func assertCmdbRecordCount(t *testing.T, db *gorm.DB, table string, where string, expected int64) {
	t.Helper()
	var count int64
	if err := db.Table(table).Where(where).Count(&count).Error; err != nil {
		t.Fatalf("count %s: %v", table, err)
	}
	if count != expected {
		t.Fatalf("unexpected %s count for %q: got %d want %d", table, where, count, expected)
	}
}
