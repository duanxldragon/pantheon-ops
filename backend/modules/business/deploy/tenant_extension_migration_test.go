package deploy

import (
	"context"
	"database/sql"
	_ "embed"
	"testing"

	"github.com/duanxldragon/pantheon-base/backend/pkg/testmysql"
)

//go:embed migrations/000043_extension_business_tenant_ownership.up.sql
var extensionTenantUp string

//go:embed migrations/000043_extension_business_tenant_ownership.down.sql
var extensionTenantDown string

func openExtensionConn(t *testing.T) (*sql.Conn, context.Context) {
	t.Helper()
	sqlDB, err := testmysql.Open(t).DB()
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	conn, err := sqlDB.Conn(ctx)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	return conn, ctx
}

func execAll(t *testing.T, conn *sql.Conn, statements ...string) {
	t.Helper()
	for _, statement := range statements {
		if _, err := conn.ExecContext(context.Background(), statement); err != nil {
			t.Fatalf("%v\n%s", err, statement)
		}
	}
}

func requireCount(t *testing.T, conn *sql.Conn, want int, query string, args ...any) {
	t.Helper()
	var got int
	if err := conn.QueryRowContext(context.Background(), query, args...).Scan(&got); err != nil || got != want {
		t.Fatalf("%s = %d (err=%v), want %d", query, got, err, want)
	}
}

// uniqueNameKeys counts UNIQUE indexes whose only column is col.
func uniqueNameKeys(t *testing.T, conn *sql.Conn, table, col string) int {
	t.Helper()
	var got int
	err := conn.QueryRowContext(context.Background(), `
		SELECT COUNT(*) FROM (SELECT INDEX_NAME FROM information_schema.STATISTICS
		WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME=? AND NON_UNIQUE=0 AND INDEX_NAME<>'PRIMARY'
		GROUP BY INDEX_NAME HAVING COUNT(*)=1 AND MAX(COLUMN_NAME)=?) k`, table, col).Scan(&got)
	if err != nil {
		t.Fatal(err)
	}
	return got
}

// requireTwoTenantsOneName proves the tenant-first key: the same natural value is
// accepted in two tenants and rejected as a duplicate inside one.
func requireTwoTenantsOneName(t *testing.T, conn *sql.Conn, table, column string) {
	t.Helper()
	insert := func(tenant int) error {
		_, err := conn.ExecContext(context.Background(),
			"INSERT INTO "+table+" (tenant_id, "+column+") VALUES (?, 'shared-name')", tenant)
		return err
	}
	if err := insert(0); err != nil {
		t.Fatalf("%s tenant 0: %v", table, err)
	}
	if err := insert(7); err != nil {
		t.Fatalf("%s tenant 7 same value: %v", table, err)
	}
	if err := insert(7); err == nil {
		t.Fatalf("%s accepted a duplicate inside tenant 7", table)
	}
}

// TestExtensionTenantMigrationUpgradesLegacyKeys covers the upgrade path on a real
// database: legacy rows survive with tenant_id=0, an AutoMigrate-named key and a
// key with an arbitrary name are both found by their column set and replaced, a
// rerun is a no-op, and the retained down script leaves schema and rows intact.
func TestExtensionTenantMigrationUpgradesLegacyKeys(t *testing.T) {
	conn, _ := openExtensionConn(t)
	execAll(t, conn,
		`CREATE TABLE biz_argocd_server (id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT, name VARCHAR(128) NOT NULL,
			PRIMARY KEY (id), UNIQUE INDEX idx_biz_argocd_server_name (name)) ENGINE=InnoDB`,
		`CREATE TABLE filedist_storage (id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT, name VARCHAR(128) NOT NULL,
			PRIMARY KEY (id), UNIQUE INDEX an_unrelated_key_name (name)) ENGINE=InnoDB`,
		`CREATE TABLE biz_k8s_release (id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT, idempotency_key VARCHAR(128) NOT NULL,
			PRIMARY KEY (id), UNIQUE INDEX uk_k8s_release_idempotency (idempotency_key)) ENGINE=InnoDB`,
		`CREATE TABLE biz_application (id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT, name VARCHAR(128) NOT NULL,
			PRIMARY KEY (id)) ENGINE=InnoDB`,
		`INSERT INTO biz_argocd_server (name) VALUES ('legacy-server')`,
		`INSERT INTO biz_application (name) VALUES ('legacy-app')`,
	)

	// Tables missing from this fixture are skipped by the guards.
	applySQLScript(t, conn, extensionTenantUp, "T14 migration")
	applySQLScript(t, conn, extensionTenantUp, "T14 migration rerun")

	tables := []string{"biz_argocd_server", "filedist_storage", "biz_k8s_release", "biz_application"}
	for _, table := range tables {
		requireCount(t, conn, 1, `SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME=? AND COLUMN_NAME='tenant_id'`, table)
	}
	requireCount(t, conn, 1, `SELECT COUNT(*) FROM biz_argocd_server WHERE name='legacy-server' AND tenant_id=0`)
	requireCount(t, conn, 1, `SELECT COUNT(*) FROM biz_application WHERE name='legacy-app' AND tenant_id=0`)

	requireTenantFirstIndex(t, conn, "biz_argocd_server", "uk_argocd_server_name", []string{"tenant_id", "name"})
	requireTenantFirstIndex(t, conn, "filedist_storage", "uk_filedist_storage_name", []string{"tenant_id", "name"})
	requireTenantFirstIndex(t, conn, "biz_k8s_release", "uk_k8s_release_idempotency", []string{"tenant_id", "idempotency_key"})
	for _, table := range []string{"biz_argocd_server", "filedist_storage"} {
		if got := uniqueNameKeys(t, conn, table, "name"); got != 0 {
			t.Fatalf("%s kept %d global single-column name key(s)", table, got)
		}
	}
	if got := uniqueNameKeys(t, conn, "biz_k8s_release", "idempotency_key"); got != 0 {
		t.Fatalf("biz_k8s_release kept %d global idempotency key(s)", got)
	}

	requireTwoTenantsOneName(t, conn, "biz_argocd_server", "name")
	requireTwoTenantsOneName(t, conn, "filedist_storage", "name")
	requireTwoTenantsOneName(t, conn, "biz_k8s_release", "idempotency_key")

	applySQLScript(t, conn, extensionTenantDown, "T14 down")
	for _, table := range tables {
		requireCount(t, conn, 1, `SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME=? AND COLUMN_NAME='tenant_id'`, table)
	}
	requireTenantFirstIndex(t, conn, "biz_argocd_server", "uk_argocd_server_name", []string{"tenant_id", "name"})
}

// TestExtensionTenantMigrationRecoversAutoMigratedState covers a startup where
// AutoMigrate already added tenant_id and the new key from the model tags while the
// old global key is still present: the migration must drop only the old key.
func TestExtensionTenantMigrationRecoversAutoMigratedState(t *testing.T) {
	conn, _ := openExtensionConn(t)
	execAll(t, conn,
		`CREATE TABLE obs_alert_rule (id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT, tenant_id BIGINT UNSIGNED NOT NULL DEFAULT 0,
			name VARCHAR(128) NOT NULL, PRIMARY KEY (id),
			UNIQUE INDEX idx_obs_alert_rule_name (name),
			UNIQUE INDEX uk_obs_alert_rule_name (tenant_id, name)) ENGINE=InnoDB`,
	)
	applySQLScript(t, conn, extensionTenantUp, "T14 migration")
	requireTenantFirstIndex(t, conn, "obs_alert_rule", "uk_obs_alert_rule_name", []string{"tenant_id", "name"})
	if got := uniqueNameKeys(t, conn, "obs_alert_rule", "name"); got != 0 {
		t.Fatalf("obs_alert_rule kept %d global name key(s)", got)
	}
	requireTwoTenantsOneName(t, conn, "obs_alert_rule", "name")
}
