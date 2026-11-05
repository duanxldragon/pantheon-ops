package database_test

import (
	"os"
	"strings"
	"testing"

	mysqlDriver "github.com/go-sql-driver/mysql"
	"gorm.io/gorm"

	cmdbgroup "github.com/duanxldragon/pantheon-base/backend/modules/business/cmdb/group"
	"github.com/duanxldragon/pantheon-base/backend/modules/business/cmdb/host"
	"github.com/duanxldragon/pantheon-base/backend/pkg/database"
	"github.com/duanxldragon/pantheon-base/backend/pkg/testmysql"
)

// TestHostStateVersionBackfillEnablesCAS proves migration 000031 does what the
// host fix relies on: a legacy row created at version 0 is moved to 1, and from
// then on compare-and-swap rejects a stale expectation instead of falling
// through to lifecycle validation.
//
// The row is inserted with raw SQL on purpose. Seeding through the model would
// apply the column default (now 1) and hide the exact state this migration has
// to repair.
func TestHostStateVersionBackfillEnablesCAS(t *testing.T) {
	db := testmysql.Open(t)
	if err := db.AutoMigrate(&host.Host{}, &cmdbgroup.Group{}); err != nil {
		t.Fatalf("migrate host tables: %v", err)
	}

	legacyIP := "10.9.9.31"
	if err := db.Exec(
		"INSERT INTO biz_cmdb_host (hostname, ip, os, state_version, status) VALUES (?, ?, 'linux', 0, 'pending')",
		"legacy-host", legacyIP,
	).Error; err != nil {
		t.Fatalf("seed legacy host: %v", err)
	}
	currentIP := "10.9.9.32"
	if err := db.Exec(
		"INSERT INTO biz_cmdb_host (hostname, ip, os, state_version, status) VALUES (?, ?, 'linux', 5, 'pending')",
		"current-host", currentIP,
	).Error; err != nil {
		t.Fatalf("seed current host: %v", err)
	}

	dsn := hostBackfillTestDSN(t, db)
	if err := database.RunMigrations(dsn); err != nil {
		t.Fatalf("run migrations: %v", err)
	}

	if got := hostStateVersionOf(t, db, legacyIP); got != 1 {
		t.Fatalf("expected legacy host backfilled to version 1, got %d", got)
	}
	if got := hostStateVersionOf(t, db, currentIP); got != 5 {
		t.Fatalf("expected current host version 5 untouched, got %d", got)
	}

	// Re-running the chain must not move the backfilled row again.
	if err := database.RunMigrations(dsn); err != nil {
		t.Fatalf("re-run migrations: %v", err)
	}
	if got := hostStateVersionOf(t, db, legacyIP); got != 1 {
		t.Fatalf("expected backfill to be idempotent, version now %d", got)
	}

	svc := host.NewHostService(db)
	legacy := findHostByIP(t, db, legacyIP)

	assigned, err := svc.TransitionState(legacy.ID, host.HostStateTransitionRequest{
		Action:               host.HostTransitionAssign,
		ExpectedStateVersion: 1,
	}, "1", nil)
	if err != nil {
		t.Fatalf("assigned transition with backfilled version: %v", err)
	}
	if assigned.StateVersion != 2 {
		t.Fatalf("expected version 2 after one transition, got %d", assigned.StateVersion)
	}

	_, err = svc.TransitionState(legacy.ID, host.HostStateTransitionRequest{
		Action:               host.HostTransitionRetire,
		ExpectedStateVersion: 1,
	}, "1", nil)
	if err == nil || err.Error() != "cmdbhost.state.stale" {
		t.Fatalf("expected stale rejection after backfill, got %v", err)
	}
}

func hostStateVersionOf(t *testing.T, db *gorm.DB, ip string) uint64 {
	t.Helper()

	var version uint64
	if err := db.Raw("SELECT state_version FROM biz_cmdb_host WHERE ip = ?", ip).Scan(&version).Error; err != nil {
		t.Fatalf("read state version for %s: %v", ip, err)
	}
	return version
}

func findHostByIP(t *testing.T, db *gorm.DB, ip string) host.Host {
	t.Helper()

	var row host.Host
	if err := db.Where("ip = ?", ip).Take(&row).Error; err != nil {
		t.Fatalf("load host %s: %v", ip, err)
	}
	return row
}

// hostBackfillTestDSN points the migration runner at the throwaway database
// testmysql just created, mirroring migrationTestDSN in the internal test
// package (this file lives in database_test so it can import the host service).
func hostBackfillTestDSN(t *testing.T, db *gorm.DB) string {
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
