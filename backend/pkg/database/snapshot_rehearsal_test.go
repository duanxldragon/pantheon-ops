package database

import (
	"database/sql"
	"fmt"
	"os"
	"testing"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

func TestProductionSnapshotRollbackAndReapply(t *testing.T) {
	dsn := os.Getenv("PANTHEON_SNAPSHOT_REHEARSAL_DSN")
	if dsn == "" {
		t.Skip("PANTHEON_SNAPSHOT_REHEARSAL_DSN is not configured")
	}
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatalf("open restored snapshot: %v", err)
	}
	defer func() { _ = db.Close() }()
	assertNoActiveBusinessKeyDuplicates(t, db)
	if err := RunMigrations(dsn); err != nil {
		t.Fatalf("upgrade restored snapshot: %v", err)
	}
	assertBusinessGeneratedKeysAndIndexes(t, db)
	before := snapshotBusinessRowCounts(t, db)

	source, err := iofs.New(migrationFS, "migrations")
	if err != nil {
		t.Fatalf("create migration source: %v", err)
	}
	migrateDSN, err := buildMigrateDSN(dsn)
	if err != nil {
		t.Fatalf("build migration dsn: %v", err)
	}
	m, err := migrate.NewWithSourceInstance("iofs", source, migrateDSN)
	if err != nil {
		t.Fatalf("create migration instance: %v", err)
	}
	defer func() { _, _ = m.Close() }()

	if err := m.Steps(-6); err != nil {
		t.Fatalf("rollback snapshot to version 16: %v", err)
	}
	assertSQLMigrationVersion(t, db, 16)
	if err := m.Up(); err != nil {
		t.Fatalf("reapply snapshot migrations: %v", err)
	}
	assertSQLMigrationVersion(t, db, 22)
	after := snapshotBusinessRowCounts(t, db)
	for table, count := range before {
		if after[table] != count {
			t.Fatalf("row count changed for %s: before=%d after=%d", table, count, after[table])
		}
	}
	assertNoActiveBusinessKeyDuplicates(t, db)
	assertBusinessGeneratedKeysAndIndexes(t, db)
	for _, table := range []string{"biz_deploy_task_attempt", "biz_k8s_namespace_binding", "biz_deploy_credential_ref", "biz_k8s_cluster_credential_ref"} {
		assertSQLTableExists(t, db, table)
	}
}

type businessUniqueKeySpec struct {
	table               string
	activeExpression    string
	generatedColumn     string
	generatedExpression string
	index               string
}

var businessUniqueKeySpecs = []businessUniqueKeySpec{
	{table: "biz_business_scope", activeExpression: "`code`", generatedColumn: "active_code", generatedExpression: "IF(`deleted_at` IS NULL, `code`, NULL)", index: "uk_business_scope_code_active"},
	{table: "biz_cmdb_host", activeExpression: "`ip`", generatedColumn: "active_ip", generatedExpression: "IF(`deleted_at` IS NULL, `ip`, NULL)", index: "uk_cmdb_host_ip_active"},
	{table: "biz_cmdb_label_schema", activeExpression: "`key`", generatedColumn: "active_key", generatedExpression: "IF(`deleted_at` IS NULL, `key`, NULL)", index: "uk_cmdb_label_schema_key_active"},
	{table: "biz_deploy_package", activeExpression: "CONCAT(`name`, '#', `version`)", generatedColumn: "active_name_version", generatedExpression: "IF(`deleted_at` IS NULL, CONCAT(`name`, '#', `version`), NULL)", index: "uk_deploy_package_name_version_active"},
	{table: "biz_deploy_template", activeExpression: "CONCAT(`name`, '#', `version`)", generatedColumn: "active_name_version", generatedExpression: "IF(`deleted_at` IS NULL, CONCAT(`name`, '#', `version`), NULL)", index: "uk_deploy_template_name_version_active"},
	{table: "biz_k8s_cluster", activeExpression: "`code`", generatedColumn: "active_code", generatedExpression: "IF(`deleted_at` IS NULL, `code`, NULL)", index: "uk_k8s_cluster_code_active"},
}

func assertNoActiveBusinessKeyDuplicates(t *testing.T, db *sql.DB) {
	t.Helper()
	for _, spec := range businessUniqueKeySpecs {
		query := fmt.Sprintf("SELECT COUNT(*) FROM (SELECT %s FROM `%s` WHERE `deleted_at` IS NULL GROUP BY %s HAVING COUNT(*) > 1) duplicate_keys", spec.activeExpression, spec.table, spec.activeExpression)
		var count int64
		if err := db.QueryRow(query).Scan(&count); err != nil {
			t.Fatalf("scan active duplicate keys for %s: %v", spec.table, err)
		}
		if count != 0 {
			t.Fatalf("found %d duplicate active keys in %s", count, spec.table)
		}
	}
}

func assertBusinessGeneratedKeysAndIndexes(t *testing.T, db *sql.DB) {
	t.Helper()
	for _, spec := range businessUniqueKeySpecs {
		query := fmt.Sprintf("SELECT COUNT(*) FROM `%s` WHERE NOT (`%s` <=> %s)", spec.table, spec.generatedColumn, spec.generatedExpression)
		var mismatches int64
		if err := db.QueryRow(query).Scan(&mismatches); err != nil {
			t.Fatalf("validate generated key for %s.%s: %v", spec.table, spec.generatedColumn, err)
		}
		if mismatches != 0 {
			t.Fatalf("found %d generated-key mismatches in %s.%s", mismatches, spec.table, spec.generatedColumn)
		}

		var nonUnique int
		var columns string
		if err := db.QueryRow(`
SELECT MIN(non_unique), GROUP_CONCAT(column_name ORDER BY seq_in_index)
FROM information_schema.statistics
WHERE table_schema = DATABASE() AND table_name = ? AND index_name = ?
GROUP BY index_name
`, spec.table, spec.index).Scan(&nonUnique, &columns); err != nil {
			t.Fatalf("validate unique index %s.%s: %v", spec.table, spec.index, err)
		}
		if nonUnique != 0 || columns != spec.generatedColumn {
			t.Fatalf("unexpected unique index %s.%s: nonUnique=%d columns=%s", spec.table, spec.index, nonUnique, columns)
		}
	}
}

func snapshotBusinessRowCounts(t *testing.T, db *sql.DB) map[string]int64 {
	t.Helper()
	tables := []string{"biz_business_scope", "biz_cmdb_group", "biz_cmdb_host", "biz_cmdb_label_schema", "biz_deploy_package", "biz_deploy_template", "biz_k8s_cluster"}
	counts := make(map[string]int64, len(tables))
	for _, table := range tables {
		var count int64
		if err := db.QueryRow("SELECT COUNT(*) FROM `" + table + "`").Scan(&count); err != nil {
			t.Fatalf("count %s: %v", table, err)
		}
		counts[table] = count
	}
	return counts
}

func assertSQLMigrationVersion(t *testing.T, db *sql.DB, expected uint) {
	t.Helper()
	var version uint
	var dirty bool
	if err := db.QueryRow("SELECT version, dirty FROM schema_migrations LIMIT 1").Scan(&version, &dirty); err != nil {
		t.Fatalf("read migration version: %v", err)
	}
	if version != expected || dirty {
		t.Fatalf("expected migration version %d clean, got version=%d dirty=%v", expected, version, dirty)
	}
}

func assertSQLTableExists(t *testing.T, db *sql.DB, table string) {
	t.Helper()
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name = ?", table).Scan(&count); err != nil {
		t.Fatalf("check table %s: %v", table, err)
	}
	if count != 1 {
		t.Fatalf("expected table %s after reapply", table)
	}
}
