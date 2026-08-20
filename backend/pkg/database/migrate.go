package database

import (
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"strconv"
	"strings"

	mysqlDriver "github.com/go-sql-driver/mysql"
	"github.com/golang-migrate/migrate/v4"
	// Blank import registers the golang-migrate MySQL database driver used by RunMigrations.
	_ "github.com/golang-migrate/migrate/v4/database/mysql"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"gorm.io/gorm"
)

//go:embed migrations/*.sql
var migrationFS embed.FS

const migrationsTableName = "schema_migrations"
const menuHideInNavCompatMigrationVersion = 6
const moduleRegistrationCompatMigrationVersion = 8
const opsSchemaRepairMigrationVersion = 16

type schemaColumnMarker struct {
	table  string
	column string
}

var preMenuHideInNavRuntimeSchemaMarkers = []schemaColumnMarker{
	{table: "system_menu", column: "active_menu"},
	{table: "system_user_session", column: "refresh_expires_at"},
	{table: "system_setting", column: "is_encrypted"},
	{table: "system_i18n", column: "key"},
	{table: "system_dict_type", column: "module"},
	{table: "system_log_oper", column: "oper_time"},
	{table: "system_log_oper", column: "source_domain"},
	{table: "system_refresh_version", column: "created_at"},
	{table: "system_generator_datasource", column: "password_encrypted"},
}

var preModuleRegistrationRuntimeSchemaMarkers = append(
	[]schemaColumnMarker{
		{table: "system_menu", column: "hide_in_nav"},
		{table: "permission_workbench_remediation_event", column: "issue_key"},
		{table: "permission_workbench_remediation_event", column: "before_state"},
		{table: "permission_workbench_remediation_event", column: "after_state"},
		{table: "permission_workbench_remediation_event", column: "action"},
		{table: "permission_workbench_remediation_event", column: "created_count"},
		{table: "permission_workbench_remediation_event", column: "skipped_count"},
	},
	preMenuHideInNavRuntimeSchemaMarkers...,
)

var currentRuntimeSchemaMarkers = append(
	[]schemaColumnMarker{
		{table: "system_module_registration", column: "table_name"},
	},
	preModuleRegistrationRuntimeSchemaMarkers...,
)

// RunMigrations executes all pending database migrations.
// It uses the golang-migrate library with embedded SQL files.
// Returns nil if all migrations applied successfully, or an error on failure.
func RunMigrations(dsn string) error {
	if err := bootstrapExistingCurrentSchema(dsn); err != nil {
		return fmt.Errorf("failed to bootstrap existing current schema: %w", err)
	}
	if err := repairLegacyBusinessSchema(dsn); err != nil {
		return fmt.Errorf("failed to repair legacy business schema: %w", err)
	}
	if err := repairSkippedOpsMigrations(dsn); err != nil {
		return fmt.Errorf("failed to repair skipped ops migrations: %w", err)
	}

	d, err := iofs.New(migrationFS, "migrations")
	if err != nil {
		return fmt.Errorf("failed to create migration source: %w", err)
	}

	migrateDSN, err := buildMigrateDSN(dsn)
	if err != nil {
		return fmt.Errorf("failed to build migration DSN: %w", err)
	}

	m, err := migrate.NewWithSourceInstance("iofs", d, migrateDSN)
	if err != nil {
		return fmt.Errorf("failed to create migrate instance: %w", err)
	}
	defer func() { _, _ = m.Close() }()

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("migration failed: %w", err)
	}

	if err == migrate.ErrNoChange {
		slog.Info("database migrations: no new migrations to apply")
	} else {
		slog.Info("database migrations: all migrations applied successfully")
	}

	return nil
}

// repairSkippedOpsMigrations repairs databases that were incorrectly advanced
// to the latest version by the legacy runtime-schema bootstrap. It only repairs
// objects whose recorded version says they should already exist.
func repairSkippedOpsMigrations(dsn string) error {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return fmt.Errorf("open mysql connection: %w", err)
	}
	defer func() { _ = db.Close() }()

	version, _, recorded, err := recordedMigrationVersion(db)
	if err != nil || !recorded || version < opsSchemaRepairMigrationVersion {
		return err
	}

	statements := []struct {
		version int
		sql     string
	}{
		{16, "CREATE TABLE IF NOT EXISTS `biz_deploy_task_attempt` (`id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT, `task_id` BIGINT UNSIGNED NOT NULL, `task_host_id` BIGINT UNSIGNED NOT NULL, `attempt_no` INT NOT NULL, `status` VARCHAR(32) NOT NULL DEFAULT 'running', `worker_id` VARCHAR(128) DEFAULT '', `lease_expires_at` DATETIME(3) DEFAULT NULL, `started_at` DATETIME(3) DEFAULT NULL, `finished_at` DATETIME(3) DEFAULT NULL, `error_message` VARCHAR(512) DEFAULT '', `created_at` DATETIME(3) DEFAULT NULL, `updated_at` DATETIME(3) DEFAULT NULL, PRIMARY KEY (`id`), KEY `idx_deploy_attempt_task_host` (`task_host_id`), KEY `idx_deploy_attempt_task_status` (`task_id`, `status`), KEY `idx_deploy_attempt_lease` (`lease_expires_at`)) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4"},
		{17, "CREATE TABLE IF NOT EXISTS `biz_k8s_namespace_binding` (`id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT, `cluster_id` BIGINT UNSIGNED NOT NULL, `namespace` VARCHAR(255) NOT NULL, `business_scope_id` BIGINT UNSIGNED NOT NULL, `environment` VARCHAR(32) NOT NULL, `allowed_actions` TEXT, `created_by` VARCHAR(64) DEFAULT '', `updated_by` VARCHAR(64) DEFAULT '', `created_at` DATETIME(3) DEFAULT NULL, `updated_at` DATETIME(3) DEFAULT NULL, PRIMARY KEY (`id`), UNIQUE KEY `uk_k8s_namespace_binding` (`cluster_id`, `namespace`), KEY `idx_k8s_namespace_binding_scope` (`business_scope_id`)) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4"},
		{18, "CREATE TABLE IF NOT EXISTS `biz_deploy_credential_ref` (`id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT, `name` VARCHAR(128) NOT NULL, `username` VARCHAR(128) NOT NULL, `auth_mode` VARCHAR(32) NOT NULL, `secret_encrypted` TEXT NOT NULL, `version` BIGINT UNSIGNED NOT NULL DEFAULT 1, `status` VARCHAR(32) NOT NULL DEFAULT 'active', `created_at` DATETIME(3) DEFAULT NULL, `updated_at` DATETIME(3) DEFAULT NULL, `deleted_at` DATETIME(3) DEFAULT NULL, PRIMARY KEY (`id`), UNIQUE KEY `uk_deploy_credential_name_deleted` (`name`, `deleted_at`)) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4"},
		{19, "CREATE TABLE IF NOT EXISTS `biz_k8s_cluster_credential_ref` (`id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT, `cluster_id` BIGINT UNSIGNED NOT NULL, `encrypted` TEXT NOT NULL, `version` BIGINT UNSIGNED NOT NULL DEFAULT 1, `status` VARCHAR(32) NOT NULL DEFAULT 'active', `created_at` DATETIME(3) DEFAULT NULL, `updated_at` DATETIME(3) DEFAULT NULL, PRIMARY KEY (`id`), KEY `idx_k8s_cluster_credential_cluster` (`cluster_id`)) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4"},
	}
	for _, statement := range statements {
		if version >= statement.version {
			if _, err := db.Exec(statement.sql); err != nil {
				return fmt.Errorf("repair ops migration %d: %w", statement.version, err)
			}
		}
	}
	if version >= 19 {
		if err := ensureColumnAndIndex(db, "biz_k8s_cluster", "kubeconfig_credential_ref_id", "BIGINT UNSIGNED NOT NULL DEFAULT 0", "idx_k8s_cluster_credential_ref", "(`kubeconfig_credential_ref_id`)"); err != nil {
			return err
		}
	}
	if version >= 20 {
		columns := []struct{ name, definition string }{
			{"credential_ref_id", "BIGINT UNSIGNED NOT NULL DEFAULT 0"},
			{"credential_ref_version", "BIGINT UNSIGNED NOT NULL DEFAULT 0"},
			{"ssh_host_fingerprint", "VARCHAR(255) NOT NULL DEFAULT ''"},
			{"execution_timeout_seconds", "INT NOT NULL DEFAULT 1800"},
		}
		for _, column := range columns {
			if err := ensureColumnAndIndex(db, "biz_deploy_task", column.name, column.definition, "", ""); err != nil {
				return err
			}
		}
		if err := ensureColumnAndIndex(db, "biz_deploy_task", "credential_ref_id", "BIGINT UNSIGNED NOT NULL DEFAULT 0", "idx_deploy_task_credential_ref", "(`credential_ref_id`)"); err != nil {
			return err
		}
	}
	return nil
}

func ensureColumnAndIndex(db *sql.DB, table, column, definition, index, indexDefinition string) error {
	exists, err := tableExists(db, table)
	if err != nil || !exists {
		return err
	}
	columnPresent, err := columnExists(db, table, column)
	if err != nil {
		return err
	}
	if !columnPresent {
		if _, err := db.Exec("ALTER TABLE `" + table + "` ADD COLUMN `" + column + "` " + definition); err != nil {
			return fmt.Errorf("add %s.%s: %w", table, column, err)
		}
	}
	if index == "" {
		return nil
	}
	indexPresent, err := indexExists(db, table, index)
	if err != nil {
		return err
	}
	if !indexPresent {
		if _, err := db.Exec("CREATE INDEX `" + index + "` ON `" + table + "` " + indexDefinition); err != nil {
			return fmt.Errorf("add index %s.%s: %w", table, index, err)
		}
	}
	return nil
}

// repairLegacyBusinessSchema handles tables created by the historical
// AutoMigrate path. The generated keys are already part of migration 000012
// for a clean install, so this repair is conditional and only touches missing
// columns/indexes on older tables. Keeping the check in Go avoids MySQL's lack
// of ALTER TABLE ... ADD COLUMN IF NOT EXISTS support and keeps migration SQL
// portable across the supported 8.0 releases.
func repairLegacyBusinessSchema(dsn string) error {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return fmt.Errorf("open mysql connection: %w", err)
	}
	defer func() { _ = db.Close() }()
	if err := db.Ping(); err != nil {
		return fmt.Errorf("ping mysql connection: %w", err)
	}
	type repair struct {
		table, column, definition, index, indexDefinition string
	}
	repairs := []repair{
		{"biz_business_scope", "active_code", "VARCHAR(255) GENERATED ALWAYS AS (IF(`deleted_at` IS NULL, `code`, NULL)) STORED", "uk_business_scope_code_active", "(`active_code`)"},
		{"biz_cmdb_host", "active_ip", "VARCHAR(45) GENERATED ALWAYS AS (IF(`deleted_at` IS NULL, `ip`, NULL)) STORED", "uk_cmdb_host_ip_active", "(`active_ip`)"},
		{"biz_cmdb_label_schema", "active_key", "VARCHAR(64) GENERATED ALWAYS AS (IF(`deleted_at` IS NULL, `key`, NULL)) STORED", "uk_cmdb_label_schema_key_active", "(`active_key`)"},
		{"biz_deploy_package", "active_name_version", "VARCHAR(255) GENERATED ALWAYS AS (IF(`deleted_at` IS NULL, CONCAT(`name`, '#', `version`), NULL)) STORED", "uk_deploy_package_name_version_active", "(`active_name_version`)"},
		{"biz_deploy_template", "active_name_version", "VARCHAR(255) GENERATED ALWAYS AS (IF(`deleted_at` IS NULL, CONCAT(`name`, '#', `version`), NULL)) STORED", "uk_deploy_template_name_version_active", "(`active_name_version`)"},
		{"biz_k8s_cluster", "active_code", "VARCHAR(128) GENERATED ALWAYS AS (IF(`deleted_at` IS NULL, `code`, NULL)) STORED", "uk_k8s_cluster_code_active", "(`active_code`)"},
	}
	for _, item := range repairs {
		exists, err := tableExists(db, item.table)
		if err != nil || !exists {
			if err != nil {
				return err
			}
			continue
		}
		column, err := columnExists(db, item.table, item.column)
		if err != nil {
			return err
		}
		if !column {
			if _, err := db.Exec("ALTER TABLE `" + item.table + "` ADD COLUMN `" + item.column + "` " + item.definition); err != nil {
				return fmt.Errorf("add %s.%s: %w", item.table, item.column, err)
			}
		}
		idx, err := indexExists(db, item.table, item.index)
		if err != nil {
			return err
		}
		if !idx {
			if _, err := db.Exec("CREATE UNIQUE INDEX `" + item.index + "` ON `" + item.table + "` " + item.indexDefinition); err != nil {
				return fmt.Errorf("add index %s.%s: %w", item.table, item.index, err)
			}
		}
	}
	return nil
}

func bootstrapExistingCurrentSchema(dsn string) error {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return fmt.Errorf("open mysql connection: %w", err)
	}
	defer func() { _ = db.Close() }()

	if err := db.Ping(); err != nil {
		return fmt.Errorf("ping mysql connection: %w", err)
	}
	_, dirty, versionRecorded, err := recordedMigrationVersion(db)
	if err != nil {
		return err
	}
	if versionRecorded && !dirty {
		return nil
	}

	latestVersion, err := latestMigrationVersion()
	if err != nil {
		return err
	}

	looksCurrent, err := looksLikeCurrentRuntimeSchema(db)
	if err != nil {
		return err
	}
	if looksCurrent {
		return bootstrapMigrationVersion(db, latestVersion)
	}

	looksPreModuleRegistration, err := looksLikePreModuleRegistrationRuntimeSchema(db)
	if err != nil {
		return err
	}
	if looksPreModuleRegistration && latestVersion >= moduleRegistrationCompatMigrationVersion {
		return bootstrapMigrationVersion(db, moduleRegistrationCompatMigrationVersion-1)
	}

	looksPreMenuHideInNav, err := looksLikePreMenuHideInNavRuntimeSchema(db)
	if err != nil {
		return err
	}
	if looksPreMenuHideInNav && latestVersion >= menuHideInNavCompatMigrationVersion {
		return bootstrapMigrationVersion(db, menuHideInNavCompatMigrationVersion-1)
	}

	return nil
}

func bootstrapMigrationVersion(db *sql.DB, targetVersion int) error {
	version, dirty, versionRecorded, err := recordedMigrationVersion(db)
	if err != nil {
		return err
	}
	if versionRecorded && !dirty && version == targetVersion {
		return nil
	}

	if err := writeMigrationVersion(db, targetVersion, false); err != nil {
		return err
	}

	if versionRecorded && dirty {
		slog.Info("database migrations: repaired dirty current schema migration state", "version", targetVersion)
		return nil
	}
	if versionRecorded && version != targetVersion {
		slog.Info("database migrations: aligned current schema migration state", "from", version, "to", targetVersion)
		return nil
	}

	slog.Info("database migrations: bootstrapped existing current schema", "version", targetVersion)
	return nil
}

func writeMigrationVersion(db *sql.DB, version int, dirty bool) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("begin migration state transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.Exec("CREATE TABLE IF NOT EXISTS `" + migrationsTableName + "` (version bigint not null primary key, dirty boolean not null)"); err != nil {
		return fmt.Errorf("create schema migrations table: %w", err)
	}
	if _, err := tx.Exec("DELETE FROM `" + migrationsTableName + "` LIMIT 1"); err != nil {
		return fmt.Errorf("clear schema migrations table: %w", err)
	}
	if _, err := tx.Exec("INSERT INTO `"+migrationsTableName+"` (version, dirty) VALUES (?, ?)", version, dirty); err != nil {
		return fmt.Errorf("seed schema migrations table: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit migration state transaction: %w", err)
	}
	return nil
}

func recordedMigrationVersion(db *sql.DB) (int, bool, bool, error) {
	exists, err := tableExists(db, migrationsTableName)
	if err != nil {
		return 0, false, false, err
	}
	if !exists {
		return 0, false, false, nil
	}

	var version int
	var dirty bool
	err = db.QueryRow("SELECT version, dirty FROM `"+migrationsTableName+"` LIMIT 1").Scan(&version, &dirty)
	switch {
	case err == sql.ErrNoRows:
		return 0, false, false, nil
	case err != nil:
		return 0, false, false, fmt.Errorf("query schema migrations row: %w", err)
	default:
		return version, dirty, true, nil
	}
}

func looksLikeCurrentRuntimeSchema(db *sql.DB) (bool, error) {
	return looksLikeRuntimeSchema(db, currentRuntimeSchemaMarkers)
}

func looksLikePreModuleRegistrationRuntimeSchema(db *sql.DB) (bool, error) {
	return looksLikeRuntimeSchema(db, preModuleRegistrationRuntimeSchemaMarkers)
}

func looksLikePreMenuHideInNavRuntimeSchema(db *sql.DB) (bool, error) {
	return looksLikeRuntimeSchema(db, preMenuHideInNavRuntimeSchemaMarkers)
}

func looksLikeRuntimeSchema(db *sql.DB, markers []schemaColumnMarker) (bool, error) {
	for _, marker := range markers {
		exists, err := columnExists(db, marker.table, marker.column)
		if err != nil {
			return false, err
		}
		if !exists {
			return false, nil
		}
	}

	compatibilityColumns := []struct {
		table  string
		column string
	}{
		{table: "system_user_session", column: "expires_at"},
		{table: "system_i18n", column: "locale_key"},
	}

	for _, column := range compatibilityColumns {
		compatible, err := legacyColumnAllowsRuntimeOmission(db, column.table, column.column)
		if err != nil {
			return false, err
		}
		if !compatible {
			return false, nil
		}
	}

	absentIndexes := []struct {
		table string
		index string
	}{
		{table: "system_dept", index: "idx_system_dept_dept_code"},
	}

	for _, item := range absentIndexes {
		exists, err := indexExists(db, item.table, item.index)
		if err != nil {
			return false, err
		}
		if exists {
			return false, nil
		}
	}

	return true, nil
}

func legacyColumnAllowsRuntimeOmission(db *sql.DB, table string, column string) (bool, error) {
	var count int
	var isNullable string
	var columnDefault sql.NullString
	err := db.QueryRow(`
SELECT
	COUNT(*) AS count,
	COALESCE(MAX(is_nullable), 'YES') AS is_nullable,
	MAX(column_default) AS column_default
FROM information_schema.columns
WHERE table_schema = DATABASE()
  AND table_name = ?
  AND column_name = ?
`, table, column).Scan(&count, &isNullable, &columnDefault)
	if err != nil {
		return false, fmt.Errorf("check legacy column %s.%s compatibility: %w", table, column, err)
	}
	if count == 0 {
		return true, nil
	}
	if strings.EqualFold(strings.TrimSpace(isNullable), "YES") {
		return true, nil
	}
	return columnDefault.Valid, nil
}

func tableExists(db *sql.DB, table string) (bool, error) {
	var count int
	err := db.QueryRow(`
SELECT COUNT(*)
FROM information_schema.tables
WHERE table_schema = DATABASE()
  AND table_name = ?
`, table).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("check table %s existence: %w", table, err)
	}
	return count == 1, nil
}

func columnExists(db *sql.DB, table string, column string) (bool, error) {
	var count int
	err := db.QueryRow(`
SELECT COUNT(*)
FROM information_schema.columns
WHERE table_schema = DATABASE()
  AND table_name = ?
  AND column_name = ?
`, table, column).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("check column %s.%s existence: %w", table, column, err)
	}
	return count == 1, nil
}

func indexExists(db *sql.DB, table string, index string) (bool, error) {
	var count int
	err := db.QueryRow(`
SELECT COUNT(*)
FROM information_schema.statistics
WHERE table_schema = DATABASE()
  AND table_name = ?
  AND index_name = ?
`, table, index).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("check index %s.%s existence: %w", table, index, err)
	}
	return count > 0, nil
}

func latestMigrationVersion() (int, error) {
	entries, err := fs.ReadDir(migrationFS, "migrations")
	if err != nil {
		return 0, fmt.Errorf("read embedded migrations: %w", err)
	}

	latest := -1
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".up.sql") {
			continue
		}

		versionToken, _, found := strings.Cut(name, "_")
		if !found {
			continue
		}

		version, err := strconv.Atoi(versionToken)
		if err != nil {
			return 0, fmt.Errorf("parse migration version from %s: %w", name, err)
		}
		if version > latest {
			latest = version
		}
	}

	if latest < 0 {
		return 0, fmt.Errorf("no up migrations found in embedded filesystem")
	}

	return latest, nil
}

// RunAutoMigrate runs GORM AutoMigrate for all registered models.
// This is the legacy migration path, used when PANTHEON_AUTO_MIGRATE=true.
func RunAutoMigrate(db *gorm.DB, models ...interface{}) error {
	if err := db.AutoMigrate(models...); err != nil {
		return fmt.Errorf("auto-migrate failed: %w", err)
	}
	slog.Info("auto-migrate completed successfully")
	return nil
}

// ShouldAutoMigrate returns true if the PANTHEON_AUTO_MIGRATE env var is set to "true".
// When true, the application uses GORM AutoMigrate instead of versioned migrations.
func ShouldAutoMigrate() bool {
	return strings.EqualFold(strings.TrimSpace(os.Getenv("PANTHEON_AUTO_MIGRATE")), "true")
}

// buildMigrateDSN converts a go-sql-driver/mysql DSN string into the URL format
// expected by golang-migrate's mysql driver.
// golang-migrate expects: mysql://user:password@tcp(host:port)/dbname?params
func buildMigrateDSN(dsn string) (string, error) {
	cfg, err := mysqlDriver.ParseDSN(dsn)
	if err != nil {
		return "", fmt.Errorf("invalid mysql dsn: %w", err)
	}

	passwordPart := ""
	if cfg.Passwd != "" {
		passwordPart = ":" + cfg.Passwd
	}

	params := ""
	if len(cfg.Params) > 0 {
		parts := make([]string, 0, len(cfg.Params))
		for k, v := range cfg.Params {
			parts = append(parts, fmt.Sprintf("%s=%s", k, v))
		}
		params = "?" + strings.Join(parts, "&")
	}

	migrateDSN := fmt.Sprintf("mysql://%s%s@tcp(%s)/%s%s",
		cfg.User,
		passwordPart,
		cfg.Addr,
		cfg.DBName,
		params,
	)

	return migrateDSN, nil
}
