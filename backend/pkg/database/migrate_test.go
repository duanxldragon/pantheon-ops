package database

import (
	"database/sql"
	"os"
	"strings"
	"testing"

	mysqlDriver "github.com/go-sql-driver/mysql"
	"gorm.io/gorm"

	"pantheon-ops/backend/pkg/testmysql"
)

func TestRunMigrationsAlignsRuntimeSchemaWithCurrentContracts(t *testing.T) {
	db := testmysql.Open(t)
	dsn := migrationTestDSN(t, db)

	if err := RunMigrations(dsn); err != nil {
		t.Fatalf("run migrations: %v", err)
	}

	expectedColumns := map[string][]string{
		"system_menu": {
			"page_perm", "perms", "type", "route_name", "is_visible", "is_cache", "is_external", "active_menu", "hide_in_nav",
		},
		"system_user": {
			"preference_json", "failed_login_attempts", "login_locked_until",
		},
		"system_dept": {
			"is_root", "leader", "phone", "email",
		},
		"system_user_session": {
			"refresh_expires_at", "last_refresh_at", "last_activity_at", "last_ip", "revoked_at",
		},
		"system_log_login": {
			"ipaddr", "login_location", "login_time",
		},
		"system_login_throttle": {
			"window_started_at", "last_attempt_at", "blocked_until", "created_at", "updated_at",
		},
		"system_auth_factor": {
			"secret_encrypted", "enabled", "confirmed_at",
		},
		"system_auth_mfa_challenge": {
			"purpose", "secret_encrypted", "setup_required", "consumed_at", "updated_at",
		},
		"system_auth_security_event": {
			"severity", "source_key", "message_key", "metadata", "acknowledged_at", "acknowledged_by", "acknowledged_by_user", "acknowledgement_note",
		},
		"system_setting": {
			"module", "is_encrypted", "remark",
		},
		"system_i18n": {
			"key", "remark", "lifecycle_status", "lifecycle_marked_at",
		},
		"system_dict_type": {
			"module",
		},
		"system_dict_item": {
			"item_color",
		},
		"system_log_oper": {
			"business_type", "oper_name", "oper_url", "oper_ip", "source_domain", "source_page", "json_result", "error_msg", "oper_time", "cost_time", "failure_category",
		},
		"system_refresh_version": {
			"created_at",
		},
		"system_generator_datasource": {
			"host", "port", "database_name", "username", "password_encrypted", "readonly_scope", "remark", "last_checked_at", "last_check_status", "last_check_error",
		},
		"system_role_data_scope": {
			"dept_ids",
		},
		"permission_workbench_remediation_event": {
			"issue_key", "before_state", "after_state", "action", "created_count", "skipped_count",
		},
	}

	for table, columns := range expectedColumns {
		for _, column := range columns {
			assertMigrationColumnExists(t, db, table, column)
		}
	}

	assertMigrationIndexExists(t, db, "system_log_oper", "idx_system_log_oper_source_domain_page")
	assertMigrationIndexExists(t, db, "system_log_oper", "idx_system_log_oper_source_page")
	assertMigrationIndexExists(t, db, "system_log_oper", "idx_system_log_oper_failure_category")
	assertMigrationIndexDoesNotExist(t, db, "system_dept", "idx_system_dept_dept_code")
	assertLegacyColumnDoesNotBlockRuntimeWrites(t, db, "system_user_session", "expires_at")
	assertLegacyColumnDoesNotBlockRuntimeWrites(t, db, "system_i18n", "locale_key")

	assertCurrentRuntimeWritesSucceed(t, db)
}

func TestRunMigrationsBootstrapsExistingCurrentSchema(t *testing.T) {
	db := testmysql.Open(t)
	dsn := migrationTestDSN(t, db)

	seedCurrentSchemaBootstrapMarkers(t, db)

	if err := RunMigrations(dsn); err != nil {
		t.Fatalf("run migrations on current schema without version table: %v", err)
	}

	assertLatestMigrationVersion(t, db)
}

func TestRunMigrationsAppliesLatestCompatWhenBootstrappedSchemaMissesMenuHideInNav(t *testing.T) {
	db := testmysql.Open(t)
	dsn := migrationTestDSN(t, db)

	seedCurrentSchemaBootstrapMarkers(t, db)
	dropMigrationColumnIfExists(t, db, "system_menu", "hide_in_nav")

	if err := RunMigrations(dsn); err != nil {
		t.Fatalf("run migrations on pre-hide-in-nav current schema: %v", err)
	}

	assertMigrationColumnExists(t, db, "system_menu", "hide_in_nav")
	assertLatestMigrationVersion(t, db)
}

func TestRunMigrationsAppliesLatestCompatWhenBootstrappedSchemaMissesPermissionWorkbenchRemediationColumns(t *testing.T) {
	db := testmysql.Open(t)
	dsn := migrationTestDSN(t, db)

	seedCurrentSchemaBootstrapMarkers(t, db)
	reseedLegacyPermissionWorkbenchRemediationEventSchema(t, db)

	if err := RunMigrations(dsn); err != nil {
		t.Fatalf("run migrations on pre-remediation current schema: %v", err)
	}

	assertMigrationColumnExists(t, db, "permission_workbench_remediation_event", "issue_key")
	assertMigrationColumnExists(t, db, "permission_workbench_remediation_event", "before_state")
	assertMigrationColumnExists(t, db, "permission_workbench_remediation_event", "after_state")
	assertMigrationColumnExists(t, db, "permission_workbench_remediation_event", "action")
	assertMigrationColumnExists(t, db, "permission_workbench_remediation_event", "created_count")
	assertMigrationColumnExists(t, db, "permission_workbench_remediation_event", "skipped_count")
	assertPermissionWorkbenchRemediationRuntimeWriteSucceeds(t, db, "legacy-role")
	assertLatestMigrationVersion(t, db)
}

func TestRunMigrationsRepairsDirtyCurrentSchemaVersion(t *testing.T) {
	db := testmysql.Open(t)
	dsn := migrationTestDSN(t, db)

	seedCurrentSchemaBootstrapMarkers(t, db)
	if err := db.Exec("CREATE TABLE schema_migrations (version BIGINT NOT NULL PRIMARY KEY, dirty BOOLEAN NOT NULL)").Error; err != nil {
		t.Fatalf("create schema_migrations: %v", err)
	}
	if err := db.Exec("INSERT INTO schema_migrations (version, dirty) VALUES (?, ?)", 2, true).Error; err != nil {
		t.Fatalf("seed dirty schema_migrations: %v", err)
	}

	if err := RunMigrations(dsn); err != nil {
		t.Fatalf("run migrations on dirty current schema: %v", err)
	}

	assertLatestMigrationVersion(t, db)
}

func assertLatestMigrationVersion(t *testing.T, db *gorm.DB) {
	t.Helper()

	latestVersion, err := latestMigrationVersion()
	if err != nil {
		t.Fatalf("resolve latest migration version: %v", err)
	}
	assertMigrationVersion(t, db, latestVersion)
}

func dropMigrationColumnIfExists(t *testing.T, db *gorm.DB, table string, column string) {
	t.Helper()

	var count int64
	err := db.Raw(`
SELECT COUNT(*)
FROM information_schema.columns
WHERE table_schema = DATABASE()
  AND table_name = ?
  AND column_name = ?
`, table, column).Scan(&count).Error
	if err != nil {
		t.Fatalf("check column before drop %s.%s: %v", table, column, err)
	}
	if count == 0 {
		return
	}
	if err := db.Exec("ALTER TABLE `" + table + "` DROP COLUMN `" + column + "`").Error; err != nil {
		t.Fatalf("drop column %s.%s: %v", table, column, err)
	}
}

func migrationTestDSN(t *testing.T, db *gorm.DB) string {
	t.Helper()

	baseDSN := strings.TrimSpace(os.Getenv("PANTHEON_TEST_DSN"))
	if baseDSN == "" {
		baseDSN = strings.TrimSpace(os.Getenv("PANTHEON_DSN"))
	}
	if baseDSN == "" {
		t.Fatal("mysql test dsn is not configured")
	}

	cfg, err := mysqlDriver.ParseDSN(baseDSN)
	if err != nil {
		t.Fatalf("parse mysql dsn: %v", err)
	}

	var dbName string
	if err := db.Raw("SELECT DATABASE()").Scan(&dbName).Error; err != nil {
		t.Fatalf("query current database: %v", err)
	}
	if strings.TrimSpace(dbName) == "" {
		t.Fatal("current database name is empty")
	}

	cfg.DBName = dbName
	cfg.ParseTime = true
	return cfg.FormatDSN()
}

func assertMigrationColumnExists(t *testing.T, db *gorm.DB, table string, column string) {
	t.Helper()

	var count int64
	err := db.Raw(`
SELECT COUNT(*)
FROM information_schema.columns
WHERE table_schema = DATABASE()
  AND table_name = ?
  AND column_name = ?
`, table, column).Scan(&count).Error
	if err != nil {
		t.Fatalf("check column %s.%s: %v", table, column, err)
	}
	if count != 1 {
		t.Fatalf("expected column %s.%s to exist after migrations, got count=%d", table, column, count)
	}
}

func assertMigrationIndexExists(t *testing.T, db *gorm.DB, table string, index string) {
	t.Helper()

	var count int64
	err := db.Raw(`
SELECT COUNT(*)
FROM information_schema.statistics
WHERE table_schema = DATABASE()
  AND table_name = ?
  AND index_name = ?
`, table, index).Scan(&count).Error
	if err != nil {
		t.Fatalf("check index %s.%s: %v", table, index, err)
	}
	if count < 1 {
		t.Fatalf("expected index %s.%s to exist after migrations", table, index)
	}
}

func assertMigrationIndexDoesNotExist(t *testing.T, db *gorm.DB, table string, index string) {
	t.Helper()

	var count int64
	err := db.Raw(`
SELECT COUNT(*)
FROM information_schema.statistics
WHERE table_schema = DATABASE()
  AND table_name = ?
  AND index_name = ?
`, table, index).Scan(&count).Error
	if err != nil {
		t.Fatalf("check absent index %s.%s: %v", table, index, err)
	}
	if count != 0 {
		t.Fatalf("expected index %s.%s to be absent after migrations", table, index)
	}
}

func assertLegacyColumnDoesNotBlockRuntimeWrites(t *testing.T, db *gorm.DB, table string, column string) {
	t.Helper()

	var count int64
	var isNullable string
	var columnDefault sql.NullString
	err := db.Raw(`
SELECT
	COUNT(*) AS count,
	COALESCE(MAX(is_nullable), 'YES') AS is_nullable,
	MAX(column_default) AS column_default
FROM information_schema.columns
WHERE table_schema = DATABASE()
  AND table_name = ?
  AND column_name = ?
`, table, column).Row().Scan(&count, &isNullable, &columnDefault)
	if err != nil {
		t.Fatalf("check legacy column compatibility %s.%s: %v", table, column, err)
	}
	if count == 0 {
		return
	}
	if strings.EqualFold(strings.TrimSpace(isNullable), "YES") {
		return
	}
	if columnDefault.Valid {
		return
	}
	t.Fatalf("expected legacy column %s.%s to be nullable or have a default value", table, column)
}

func assertCurrentRuntimeWritesSucceed(t *testing.T, db *gorm.DB) {
	t.Helper()

	if err := db.Exec(`
INSERT INTO system_menu (
	parent_id, title_key, path, component, page_perm, perms, type, icon, route_name, module, sort, is_visible, is_cache, is_external, active_menu, hide_in_nav, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, NOW(3), NOW(3))
`, 0, "runtime.schema.menu", "/runtime-schema-menu", "runtime/SchemaMenu", "system:runtime:view", "", "C", "dashboard", "runtime-schema-menu", "system", 999, 1, 0, 0, "", 0).Error; err != nil {
		t.Fatalf("insert current runtime menu row: %v", err)
	}

	if err := db.Exec(`
INSERT INTO system_user_session (
	session_id, user_id, refresh_jti, refresh_expires_at, last_refresh_at, last_activity_at, last_ip, user_agent, revoked_at, created_at, updated_at
) VALUES (?, ?, ?, NOW(3), NULL, NOW(3), ?, ?, NULL, NOW(3), NOW(3))
`, "runtime-session-1", 1, "refresh-jti-1", "127.0.0.1", "pantheon-smoke").Error; err != nil {
		t.Fatalf("insert current runtime session row: %v", err)
	}

	if err := db.Exec(`
INSERT INTO system_i18n (
	module, group_name, `+"`key`"+`, locale, value, remark, lifecycle_status, lifecycle_marked_at, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, NULL, NOW(3), NOW(3))
`, "system.config", "menu", "runtime.schema.write", "zh-CN", "运行时写入", "compat", "active").Error; err != nil {
		t.Fatalf("insert current runtime i18n row: %v", err)
	}

	if err := db.Exec(`
INSERT INTO system_log_oper (
	request_id, title, business_type, method, oper_name, oper_url, oper_ip, source_domain, source_page, oper_param, json_result, status, failure_category, error_msg, oper_time, cost_time
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, NOW(3), ?)
`, "runtime-request-1", "runtime.schema.write", 2, "POST", "admin", "/api/v1/auth/login", "127.0.0.1", "auth", "session", `{"username":"admin"}`, `{"code":200}`, 1, "", "", 12).Error; err != nil {
		t.Fatalf("insert current runtime operation log row: %v", err)
	}

	if err := db.Exec(`
INSERT INTO system_refresh_version (
	topic, version, updated_at, created_at
) VALUES (?, ?, NOW(3), NOW(3))
`, "system:user:changed", 1).Error; err != nil {
		t.Fatalf("insert current runtime refresh sync row: %v", err)
	}

	if err := db.Exec(`
INSERT INTO system_dept (
	parent_id, ancestors, is_root, dept_name, sort, leader_user_id, leader, phone, email, status, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, NOW(3), NOW(3))
`, 1, "1", 0, "runtime-dept-a", 1, 0, "", "", "", 1).Error; err != nil {
		t.Fatalf("insert first current runtime dept row: %v", err)
	}

	if err := db.Exec(`
INSERT INTO system_dept (
	parent_id, ancestors, is_root, dept_name, sort, leader_user_id, leader, phone, email, status, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, NOW(3), NOW(3))
`, 1, "1", 0, "runtime-dept-b", 2, 0, "", "", "", 1).Error; err != nil {
		t.Fatalf("insert second current runtime dept row: %v", err)
	}

	if err := db.Exec(`
INSERT INTO system_generator_datasource (
	name, driver, host, port, database_name, username, password_encrypted, status, readonly_scope, remark, last_check_status, last_check_error, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, NOW(3), NOW(3))
`, "runtime-metadata", "mysql", "db.example.com", 3306, "metadata_schema", "reader", "ciphertext", 1, "metadata_only", "compat", "", "").Error; err != nil {
		t.Fatalf("insert current runtime generator datasource row: %v", err)
	}

	if err := db.Exec(`
INSERT INTO permission_workbench_remediation_event (
	role_key, issue_type, issue_key, before_state, after_state, action, created_count, skipped_count, created_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, NOW(3))
`, "runtime-role", "api-gap", "POST /api/v1/lowcode/dynamic-modules/generate", "api-gap", "complete", "remediated", 1, 0).Error; err != nil {
		t.Fatalf("insert current runtime remediation event row: %v", err)
	}
}

func assertPermissionWorkbenchRemediationRuntimeWriteSucceeds(t *testing.T, db *gorm.DB, roleKey string) {
	t.Helper()

	if err := db.Exec(`
INSERT INTO permission_workbench_remediation_event (
	role_key, issue_type, issue_key, before_state, after_state, action, created_count, skipped_count, created_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, NOW(3))
`, roleKey, "api-gap", "POST /api/v1/lowcode/dynamic-modules/generate", "api-gap", "complete", "remediated", 1, 0).Error; err != nil {
		t.Fatalf("insert current runtime remediation event row for %s: %v", roleKey, err)
	}
}

func assertMigrationVersion(t *testing.T, db *gorm.DB, expectedVersion int) {
	t.Helper()

	type schemaMigrationRow struct {
		Version int
		Dirty   bool
	}

	var row schemaMigrationRow
	if err := db.Raw("SELECT version, dirty FROM schema_migrations LIMIT 1").Scan(&row).Error; err != nil {
		t.Fatalf("read schema_migrations: %v", err)
	}
	if row.Version != expectedVersion {
		t.Fatalf("expected schema_migrations version=%d, got %d", expectedVersion, row.Version)
	}
	if row.Dirty {
		t.Fatal("expected schema_migrations dirty=false")
	}
}

func seedCurrentSchemaBootstrapMarkers(t *testing.T, db *gorm.DB) {
	t.Helper()

	statements := []string{
		`CREATE TABLE system_menu (
			id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
			parent_id BIGINT DEFAULT 0,
			title_key VARCHAR(64) NOT NULL,
			path VARCHAR(255) DEFAULT '',
			component VARCHAR(255) DEFAULT '',
			page_perm VARCHAR(128) DEFAULT '',
			perms VARCHAR(128) DEFAULT '',
			type CHAR(1) DEFAULT 'M',
			icon VARCHAR(128) DEFAULT '',
			route_name VARCHAR(128) DEFAULT '',
			module VARCHAR(64) DEFAULT 'system',
			sort INT DEFAULT 0,
			is_visible TINYINT DEFAULT 1,
			is_cache TINYINT DEFAULT 0,
			is_external TINYINT DEFAULT 0,
			active_menu VARCHAR(255) DEFAULT '',
			hide_in_nav INT DEFAULT 0,
			created_at DATETIME(3) DEFAULT NULL,
			updated_at DATETIME(3) DEFAULT NULL,
			PRIMARY KEY (id)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,
		`CREATE TABLE system_user_session (
			session_id VARCHAR(64) NOT NULL,
			user_id BIGINT UNSIGNED NOT NULL,
			refresh_jti VARCHAR(64) NOT NULL,
			refresh_expires_at DATETIME(3) NOT NULL,
			last_refresh_at DATETIME(3) DEFAULT NULL,
			last_activity_at DATETIME(3) DEFAULT NULL,
			last_ip VARCHAR(128) DEFAULT '',
			user_agent VARCHAR(255) DEFAULT '',
			revoked_at DATETIME(3) DEFAULT NULL,
			created_at DATETIME(3) DEFAULT NULL,
			updated_at DATETIME(3) DEFAULT NULL,
			PRIMARY KEY (session_id)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,
		`CREATE TABLE system_setting (
			id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
			setting_key VARCHAR(128) NOT NULL,
			setting_value TEXT,
			value_type VARCHAR(16) NOT NULL DEFAULT 'string',
			group_key VARCHAR(32) NOT NULL,
			module VARCHAR(64) NOT NULL DEFAULT 'system',
			is_public TINYINT DEFAULT 0,
			is_encrypted TINYINT DEFAULT 0,
			remark VARCHAR(255) DEFAULT NULL,
			created_at DATETIME(3) DEFAULT NULL,
			updated_at DATETIME(3) DEFAULT NULL,
			PRIMARY KEY (id)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,
		"CREATE TABLE system_i18n (\n" +
			"\tid BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,\n" +
			"\tmodule VARCHAR(64) NOT NULL DEFAULT 'system',\n" +
			"\tgroup_name VARCHAR(64) NOT NULL DEFAULT 'messages',\n" +
			"\t`key` VARCHAR(128) NOT NULL,\n" +
			"\tlocale VARCHAR(10) NOT NULL,\n" +
			"\tvalue TEXT NOT NULL,\n" +
			"\tremark VARCHAR(255) DEFAULT NULL,\n" +
			"\tlifecycle_status VARCHAR(16) NOT NULL DEFAULT 'active',\n" +
			"\tlifecycle_marked_at DATETIME(3) DEFAULT NULL,\n" +
			"\tcreated_at DATETIME(3) DEFAULT NULL,\n" +
			"\tupdated_at DATETIME(3) DEFAULT NULL,\n" +
			"\tPRIMARY KEY (id)\n" +
			") ENGINE=InnoDB DEFAULT CHARSET=utf8mb4",
		`CREATE TABLE system_dict_type (
			id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
			dict_code VARCHAR(64) NOT NULL,
			dict_name VARCHAR(64) NOT NULL,
			module VARCHAR(64) NOT NULL DEFAULT 'system',
			status TINYINT DEFAULT 1,
			remark VARCHAR(255) DEFAULT NULL,
			created_at DATETIME(3) DEFAULT NULL,
			updated_at DATETIME(3) DEFAULT NULL,
			deleted_at DATETIME(3) DEFAULT NULL,
			PRIMARY KEY (id)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,
		`CREATE TABLE system_log_oper (
			id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
			title VARCHAR(64) DEFAULT '',
			business_type INT DEFAULT 0,
			method VARCHAR(128) DEFAULT '',
			oper_name VARCHAR(64) DEFAULT '',
			oper_url VARCHAR(255) DEFAULT '',
			oper_ip VARCHAR(128) DEFAULT '',
			source_domain VARCHAR(32) DEFAULT '',
			source_page VARCHAR(32) DEFAULT '',
			oper_param TEXT,
			json_result TEXT,
			status INT DEFAULT 1,
			failure_category VARCHAR(32) DEFAULT '',
			error_msg TEXT,
			oper_time DATETIME(3) DEFAULT NULL,
			cost_time BIGINT DEFAULT 0,
			PRIMARY KEY (id)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,
		`CREATE TABLE system_refresh_version (
			topic VARCHAR(64) NOT NULL,
			version BIGINT NOT NULL DEFAULT 0,
			created_at DATETIME(3) DEFAULT NULL,
			updated_at DATETIME(3) DEFAULT NULL,
			PRIMARY KEY (topic)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,
		`CREATE TABLE system_generator_datasource (
			id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
			name VARCHAR(128) NOT NULL,
			driver VARCHAR(32) NOT NULL DEFAULT 'mysql',
			host VARCHAR(255) NOT NULL DEFAULT '',
			port INT NOT NULL DEFAULT 3306,
			database_name VARCHAR(128) NOT NULL DEFAULT '',
			username VARCHAR(128) NOT NULL DEFAULT '',
			password_encrypted VARCHAR(1024) DEFAULT '',
			status TINYINT DEFAULT 1,
			readonly_scope VARCHAR(32) NOT NULL DEFAULT 'metadata_only',
			remark VARCHAR(255) DEFAULT '',
			last_checked_at DATETIME(3) DEFAULT NULL,
			last_check_status VARCHAR(32) DEFAULT '',
			last_check_error VARCHAR(255) DEFAULT '',
			created_at DATETIME(3) DEFAULT NULL,
			updated_at DATETIME(3) DEFAULT NULL,
			PRIMARY KEY (id)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,
		`CREATE TABLE permission_workbench_remediation_event (
			id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
			role_key VARCHAR(64) NOT NULL,
			issue_type VARCHAR(32) NOT NULL,
			issue_key VARCHAR(255) NOT NULL,
			before_state VARCHAR(32) NOT NULL,
			after_state VARCHAR(32) NOT NULL,
			action VARCHAR(32) NOT NULL,
			created_count INT DEFAULT 0,
			skipped_count INT DEFAULT 0,
			created_at DATETIME(3) DEFAULT NULL,
			PRIMARY KEY (id),
			INDEX idx_permission_remediation_role_created (role_key, created_at),
			INDEX idx_permission_remediation_issue_type (issue_type),
			INDEX idx_permission_remediation_action (action)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,
	}

	for _, statement := range statements {
		if err := db.Exec(statement).Error; err != nil {
			t.Fatalf("seed current schema bootstrap markers: %v", err)
		}
	}
}

func reseedLegacyPermissionWorkbenchRemediationEventSchema(t *testing.T, db *gorm.DB) {
	t.Helper()

	if err := db.Exec("DROP TABLE IF EXISTS permission_workbench_remediation_event").Error; err != nil {
		t.Fatalf("drop current remediation event table: %v", err)
	}
	if err := db.Exec(`
CREATE TABLE permission_workbench_remediation_event (
	id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
	role_key VARCHAR(64) NOT NULL,
	issue_type VARCHAR(32) NOT NULL,
	severity VARCHAR(16) NOT NULL DEFAULT 'medium',
	detail TEXT,
	remediated TINYINT DEFAULT 0,
	created_at DATETIME(3) DEFAULT NULL,
	updated_at DATETIME(3) DEFAULT NULL,
	PRIMARY KEY (id),
	INDEX idx_permission_remediation_role_created (role_key, created_at),
	INDEX idx_permission_remediation_issue_type (issue_type)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4
`).Error; err != nil {
		t.Fatalf("create legacy remediation event table: %v", err)
	}
}
