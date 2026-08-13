package deploy

import (
	"testing"

	"pantheon-base/pkg/testmysql"

	"gorm.io/gorm"
)

func TestSeedDeployMenusCreatesTemplateActionPermissions(t *testing.T) {
	db := testmysql.Open(t)
	mustCreateDeploySeedTables(t, db)
	if err := db.Exec("INSERT INTO system_role (id, role_key) VALUES (1, 'admin')").Error; err != nil {
		t.Fatalf("seed admin role: %v", err)
	}

	if err := seedDeployMenus(db); err != nil {
		t.Fatalf("seed deploy menus: %v", err)
	}

	assertDeployRecordCount(t, db, "system_menu", "perms = 'business:deploy:template:create' AND title_key = 'business.deploy.template.permission.create' AND type = 'F'", 1)
	assertDeployRecordCount(t, db, "system_menu", "perms = 'business:deploy:template:update' AND title_key = 'business.deploy.template.permission.update' AND type = 'F'", 1)
	assertDeployRecordCount(t, db, "system_menu", "perms = 'business:deploy:template:delete' AND title_key = 'business.deploy.template.permission.delete' AND type = 'F'", 1)
	assertDeployRecordCount(t, db, "system_menu", "parent_id = (SELECT id FROM (SELECT id FROM system_menu WHERE route_name = 'deploy-template-list' LIMIT 1) AS template_menu) AND perms IN ('business:deploy:template:create', 'business:deploy:template:update', 'business:deploy:template:delete')", 3)
	assertDeployRecordCount(t, db, "system_role_permission", "role_id = 1 AND permission_key IN ('business:deploy:template:create', 'business:deploy:template:update', 'business:deploy:template:delete')", 3)
}

func TestSeedDeployI18nCreatesTemplatePermissionTranslations(t *testing.T) {
	db := testmysql.Open(t)
	mustCreateDeployI18nTable(t, db)

	if err := seedDeployI18n(db); err != nil {
		t.Fatalf("seed deploy i18n: %v", err)
	}

	assertDeployRecordCount(t, db, "system_i18n", "`key` IN ('business.deploy.template.permission.create', 'business.deploy.template.permission.update', 'business.deploy.template.permission.delete') AND locale = 'zh-CN' AND group_name = 'permission' AND lifecycle_status = 'active'", 3)
	assertDeployRecordCount(t, db, "system_i18n", "`key` IN ('business.deploy.template.permission.create', 'business.deploy.template.permission.update', 'business.deploy.template.permission.delete') AND locale = 'en-US' AND group_name = 'permission' AND lifecycle_status = 'active'", 3)
}

func mustCreateDeploySeedTables(t *testing.T, db *gorm.DB) {
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
	}
	for _, statement := range statements {
		if err := db.Exec(statement).Error; err != nil {
			t.Fatalf("create deploy seed table: %v", err)
		}
	}
}

func mustCreateDeployI18nTable(t *testing.T, db *gorm.DB) {
	t.Helper()
	if err := db.Exec(`CREATE TABLE system_i18n (
		id BIGINT PRIMARY KEY AUTO_INCREMENT,
		module VARCHAR(64),
		locale VARCHAR(16),
		group_name VARCHAR(64),
		` + "`key`" + ` VARCHAR(255),
		value TEXT,
		lifecycle_status VARCHAR(16),
		created_at DATETIME,
		updated_at DATETIME
	)`).Error; err != nil {
		t.Fatalf("create deploy i18n table: %v", err)
	}
}

func assertDeployRecordCount(t *testing.T, db *gorm.DB, table string, where string, expected int64) {
	t.Helper()
	var count int64
	if err := db.Table(table).Where(where).Count(&count).Error; err != nil {
		t.Fatalf("count %s: %v", table, err)
	}
	if count != expected {
		t.Fatalf("unexpected %s count for %q: got %d want %d", table, where, count, expected)
	}
}
