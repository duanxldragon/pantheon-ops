package deploy

import (
	"context"
	"database/sql"
	_ "embed"
	"strings"
	"testing"

	"github.com/duanxldragon/pantheon-base/backend/pkg/testmysql"
)

//go:embed migrations/000041_core_business_tenant_ownership.up.sql
var tenantOwnershipUp string

//go:embed migrations/000041_core_business_tenant_ownership.down.sql
var tenantOwnershipDown string

func applySQLScript(t *testing.T, conn *sql.Conn, script, label string) {
	t.Helper()
	ctx := context.Background()
	var statements []string
	for line := range strings.SplitSeq(script, "\n") {
		if !strings.HasPrefix(strings.TrimSpace(line), "--") {
			statements = append(statements, line)
		}
	}
	for statement := range strings.SplitSeq(strings.Join(statements, "\n"), ";") {
		if strings.TrimSpace(statement) == "" {
			continue
		}
		if _, err := conn.ExecContext(ctx, statement); err != nil {
			t.Fatalf("apply %s: %v\n%s", label, err, statement)
		}
	}
}

// applyTenantOwnershipMigration replays the T04 migration the same way
// golang-migrate does (whole file, statement by statement).
func applyTenantOwnershipMigration(t *testing.T, conn *sql.Conn) {
	t.Helper()
	applySQLScript(t, conn, tenantOwnershipUp, "T04 migration")
}

func tenantIndexColumns(t *testing.T, conn *sql.Conn, table, index string) []string {
	t.Helper()
	rows, err := conn.QueryContext(context.Background(), `
		SELECT COLUMN_NAME FROM information_schema.STATISTICS
		WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME=? AND INDEX_NAME=?
		ORDER BY SEQ_IN_INDEX`, table, index)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var columns []string
	for rows.Next() {
		var column string
		if err := rows.Scan(&column); err != nil {
			t.Fatal(err)
		}
		columns = append(columns, column)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return columns
}

func requireTenantFirstIndex(t *testing.T, conn *sql.Conn, table, index string, want []string) {
	t.Helper()
	got := tenantIndexColumns(t, conn, table, index)
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("%s.%s columns = %v, want %v", table, index, got, want)
	}
}

func createLegacyDeployTables(t *testing.T, conn *sql.Conn) {
	t.Helper()
	legacy := []string{
		`CREATE TABLE biz_deploy_package (
			id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
			name VARCHAR(128) NOT NULL,
			version VARCHAR(64) NOT NULL,
			deleted_at DATETIME(3) NULL,
			PRIMARY KEY (id),
			UNIQUE INDEX uk_deploy_package_name_version_deleted (name, version, deleted_at)
		) ENGINE=InnoDB`,
		`CREATE TABLE biz_deploy_credential_ref (
			id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
			name VARCHAR(128) NOT NULL,
			deleted_at DATETIME(3) NULL,
			PRIMARY KEY (id),
			UNIQUE INDEX uk_deploy_credential_name_deleted (name, deleted_at)
		) ENGINE=InnoDB`,
	}
	for _, statement := range legacy {
		if _, err := conn.ExecContext(context.Background(), statement); err != nil {
			t.Fatal(err)
		}
	}
}

// TestTenantOwnershipMigrationUpgradesLegacyNameKeys verifies the T04 schema
// contract on the legacy-upgrade path: a pre-existing table without tenant_id
// and with a global name unique key gains tenant_id, and the key is rebuilt as
// tenant-first so two tenants can reuse the same natural name while a
// duplicate inside one tenant is still rejected.
func TestTenantOwnershipMigrationUpgradesLegacyNameKeys(t *testing.T) {
	db := testmysql.Open(t)
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	conn, err := sqlDB.Conn(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	createLegacyDeployTables(t, conn)

	// Missing tables are skipped by the guards; the two legacy tables are upgraded.
	applyTenantOwnershipMigration(t, conn)
	applyTenantOwnershipMigration(t, conn) // rerun must be a no-op

	requireTenantFirstIndex(t, conn, "biz_deploy_package", "uk_deploy_package_name_version_deleted", []string{"tenant_id", "name", "version", "deleted_at"})
	requireTenantFirstIndex(t, conn, "biz_deploy_credential_ref", "uk_deploy_credential_name_deleted", []string{"tenant_id", "name", "deleted_at"})
	for _, table := range []string{"biz_deploy_package", "biz_deploy_credential_ref"} {
		var count int
		if err := conn.QueryRowContext(ctx, `SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME=? AND COLUMN_NAME='tenant_id'`, table).Scan(&count); err != nil || count != 1 {
			t.Fatalf("%s.tenant_id column count = %d, err=%v", table, count, err)
		}
	}

	insert := func(sqlText string) error {
		_, err := conn.ExecContext(ctx, sqlText)
		return err
	}
	if err := insert(`INSERT INTO biz_deploy_package (name, version, tenant_id) VALUES ('pkg-a','1.0',0)`); err != nil {
		t.Fatalf("tenant 0 package: %v", err)
	}
	if err := insert(`INSERT INTO biz_deploy_package (name, version, tenant_id) VALUES ('pkg-a','1.0',7)`); err != nil {
		t.Fatalf("tenant 7 same-name package: %v", err)
	}
	// MySQL allows repeated NULLs in a unique key, so the duplicate probe uses a
	// non-NULL deleted_at: the tenant-first key must reject a soft-deleted
	// duplicate inside one tenant.
	if err := insert(`INSERT INTO biz_deploy_package (name, version, tenant_id, deleted_at) VALUES ('pkg-b','1.0',0,'2026-01-01 00:00:00')`); err != nil {
		t.Fatalf("tenant 0 soft-deleted package: %v", err)
	}
	if err := insert(`INSERT INTO biz_deploy_package (name, version, tenant_id, deleted_at) VALUES ('pkg-b','1.0',0,'2026-01-01 00:00:00')`); err == nil {
		t.Fatal("duplicate soft-deleted package inside tenant 0 was accepted")
	}
	if err := insert(`INSERT INTO biz_deploy_credential_ref (name, tenant_id) VALUES ('cred','0')`); err != nil {
		t.Fatalf("tenant 0 credential: %v", err)
	}
	if err := insert(`INSERT INTO biz_deploy_credential_ref (name, tenant_id) VALUES ('cred','7')`); err != nil {
		t.Fatalf("tenant 7 same-name credential: %v", err)
	}
	if err := insert(`INSERT INTO biz_deploy_credential_ref (name, tenant_id, deleted_at) VALUES ('cred-2','0','2026-01-01 00:00:00')`); err != nil {
		t.Fatalf("tenant 0 soft-deleted credential: %v", err)
	}
	if err := insert(`INSERT INTO biz_deploy_credential_ref (name, tenant_id, deleted_at) VALUES ('cred-2','0','2026-01-01 00:00:00')`); err == nil {
		t.Fatal("duplicate soft-deleted credential name inside tenant 0 was accepted")
	}
}

// TestTenantOwnershipMigrationPreservesLegacyRowsAndRecoversPartialState covers
// the compat/multi upgrade path on a real database: pre-existing rows survive
// with tenant_id=0, a partially applied migration (column already added) is
// resumed by the idempotent guards, and the retained down script keeps schema
// and rows intact because rolling back requires a pre-T04 backup.
func TestTenantOwnershipMigrationPreservesLegacyRowsAndRecoversPartialState(t *testing.T) {
	db := testmysql.Open(t)
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	conn, err := sqlDB.Conn(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	createLegacyDeployTables(t, conn)
	for _, statement := range []string{
		`INSERT INTO biz_deploy_package (id, name, version) VALUES (1,'pkg-a','1.0'),(2,'pkg-b','2.0')`,
		`INSERT INTO biz_deploy_credential_ref (id, name) VALUES (1,'cred-a')`,
		// Simulate a run interrupted after adding the column but before the key rebuild.
		`ALTER TABLE biz_deploy_package ADD COLUMN tenant_id BIGINT UNSIGNED NOT NULL DEFAULT 0`,
	} {
		if _, err := conn.ExecContext(ctx, statement); err != nil {
			t.Fatal(err)
		}
	}

	applyTenantOwnershipMigration(t, conn)
	applySQLScript(t, conn, tenantOwnershipDown, "T04 down migration")

	requireTenantFirstIndex(t, conn, "biz_deploy_package", "uk_deploy_package_name_version_deleted", []string{"tenant_id", "name", "version", "deleted_at"})
	requireTenantFirstIndex(t, conn, "biz_deploy_credential_ref", "uk_deploy_credential_name_deleted", []string{"tenant_id", "name", "deleted_at"})

	countRows := func(table string, want int) {
		t.Helper()
		var count int
		if err := conn.QueryRowContext(ctx, `SELECT COUNT(*) FROM `+table+` WHERE tenant_id=0`).Scan(&count); err != nil {
			t.Fatalf("%s row count: %v", table, err)
		}
		if count != want {
			t.Fatalf("%s tenant-0 rows = %d, want %d", table, count, want)
		}
	}
	countRows("biz_deploy_package", 2)
	countRows("biz_deploy_credential_ref", 1)

	var packageName string
	if err := conn.QueryRowContext(ctx, `SELECT name FROM biz_deploy_package WHERE id=2`).Scan(&packageName); err != nil {
		t.Fatal(err)
	}
	if packageName != "pkg-b" {
		t.Fatalf("legacy package name = %q, want pkg-b", packageName)
	}

	var tenantColumn int
	if err := conn.QueryRowContext(ctx, `SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='biz_deploy_package' AND COLUMN_NAME='tenant_id'`).Scan(&tenantColumn); err != nil || tenantColumn != 1 {
		t.Fatalf("tenant_id column after down = %d, err=%v", tenantColumn, err)
	}
}
