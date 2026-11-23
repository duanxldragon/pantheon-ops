package deploy

import (
	"context"
	_ "embed"
	"database/sql"
	"strings"
	"testing"

	"github.com/duanxldragon/pantheon-base/backend/pkg/testmysql"
)

//go:embed migrations/000041_core_business_tenant_ownership.up.sql
var tenantOwnershipUp string

// applyTenantOwnershipMigration replays the T04 migration the same way
// golang-migrate does (whole file, statement by statement).
func applyTenantOwnershipMigration(t *testing.T, conn *sql.Conn) {
	t.Helper()
	ctx := context.Background()
	var statements []string
	for line := range strings.SplitSeq(tenantOwnershipUp, "\n") {
		if !strings.HasPrefix(strings.TrimSpace(line), "--") {
			statements = append(statements, line)
		}
	}
	for statement := range strings.SplitSeq(strings.Join(statements, "\n"), ";") {
		if strings.TrimSpace(statement) == "" {
			continue
		}
		if _, err := conn.ExecContext(ctx, statement); err != nil {
			t.Fatalf("apply T04 migration: %v\n%s", err, statement)
		}
	}
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
		if _, err := conn.ExecContext(ctx, statement); err != nil {
			t.Fatal(err)
		}
	}

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
