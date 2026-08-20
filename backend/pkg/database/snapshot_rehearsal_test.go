package database

import (
	"database/sql"
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
	if err := RunMigrations(dsn); err != nil {
		t.Fatalf("upgrade restored snapshot: %v", err)
	}
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatalf("open restored snapshot: %v", err)
	}
	defer func() { _ = db.Close() }()
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
	for _, table := range []string{"biz_deploy_task_attempt", "biz_k8s_namespace_binding", "biz_deploy_credential_ref", "biz_k8s_cluster_credential_ref"} {
		assertSQLTableExists(t, db, table)
	}
}

func snapshotBusinessRowCounts(t *testing.T, db *sql.DB) map[string]int64 {
	t.Helper()
	tables := []string{"biz_cmdb_group", "biz_cmdb_host", "biz_cmdb_label_schema", "biz_deploy_package", "biz_deploy_template"}
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
