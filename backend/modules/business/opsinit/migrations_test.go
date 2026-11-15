package opsinit

import (
	"database/sql"
	"embed"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/go-sql-driver/mysql"
)

//go:embed migrations/000033_core_business_schema.*.sql
var baselineSQL embed.FS

// This opt-in check creates and removes only its own disposable database.
// Supply a MySQL DSN without a database, e.g. root@tcp(127.0.0.1:33316)/.
func TestBusinessBaselineFreshReplayAndRetainedRollback(t *testing.T) {
	dsn := os.Getenv("PANTHEON_OPS_MIGRATION_TEST_DSN")
	if dsn == "" {
		t.Skip("PANTHEON_OPS_MIGRATION_TEST_DSN required for isolated MySQL acceptance")
	}
	config, err := mysql.ParseDSN(dsn)
	if err != nil {
		t.Fatal(err)
	}
	config.DBName = ""
	config.MultiStatements = true
	admin, err := sql.Open("mysql", config.FormatDSN())
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()
	name := fmt.Sprintf("ops_baseline_test_%d", time.Now().UnixNano())
	if _, err := admin.Exec("CREATE DATABASE `" + name + "` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci"); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if _, err := admin.Exec("DROP DATABASE `" + name + "`"); err != nil {
			t.Error(err)
		}
	}()
	config.DBName = name
	db, err := sql.Open("mysql", config.FormatDSN())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	up, err := baselineSQL.ReadFile("migrations/000033_core_business_schema.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	down, err := baselineSQL.ReadFile("migrations/000033_core_business_schema.down.sql")
	if err != nil {
		t.Fatal(err)
	}
	apply := func(script []byte) {
		t.Helper()
		if _, err := db.Exec(string(script)); err != nil {
			t.Fatal(err)
		}
	}
	apply(up)
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = ?", name).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 29 {
		t.Fatalf("fresh baseline created %d tables, want 29", count)
	}
	if _, err := db.Exec("INSERT INTO biz_cmdb_host (hostname, ip, os) VALUES ('retained-fixture','198.51.100.20','linux')"); err != nil {
		t.Fatal(err)
	}
	var stateVersion uint64
	if err := db.QueryRow("SELECT state_version FROM biz_cmdb_host WHERE hostname='retained-fixture'").Scan(&stateVersion); err != nil || stateVersion != 1 {
		t.Fatalf("new host version: %d, error: %v", stateVersion, err)
	}
	if _, err := db.Exec("INSERT INTO biz_cmdb_host (hostname, ip, os) VALUES ('duplicate-fixture','198.51.100.20','linux')"); err == nil {
		t.Fatal("active host uniqueness must be enforced by MySQL, not only an API precheck")
	}
	if _, err := db.Exec("UPDATE biz_cmdb_host SET deleted_at=NOW() WHERE hostname='retained-fixture'"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("INSERT INTO biz_cmdb_host (hostname, ip, os) VALUES ('replacement-fixture','198.51.100.20','linux')"); err != nil {
		t.Fatalf("soft-deleted IP reuse failed: %v", err)
	}
	// Simulate a partial install. Replay must recreate missing tables without losing data.
	if _, err := db.Exec("DROP TABLE biz_k8s_storage_class"); err != nil {
		t.Fatal(err)
	}
	apply(up)
	apply(down)
	if err := db.QueryRow("SELECT COUNT(*) FROM biz_cmdb_host WHERE hostname='retained-fixture'").Scan(&count); err != nil || count != 1 {
		t.Fatalf("replay/down lost existing row: count=%d, error=%v", count, err)
	}
	if err := db.QueryRow("SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = ?", name).Scan(&count); err != nil || count != 29 {
		t.Fatalf("replay/down lost schema: count=%d, error=%v", count, err)
	}
}
