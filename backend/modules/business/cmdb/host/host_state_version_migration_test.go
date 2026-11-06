package host_test

import (
	"database/sql"
	"embed"
	"os"
	"strings"
	"testing"

	mysqlDriver "github.com/go-sql-driver/mysql"
	"github.com/golang-migrate/migrate/v4"
	migrateMySQL "github.com/golang-migrate/migrate/v4/database/mysql"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"gorm.io/gorm"

	cmdbgroup "github.com/duanxldragon/pantheon-base/backend/modules/business/cmdb/group"
	"github.com/duanxldragon/pantheon-base/backend/modules/business/cmdb/host"
	"github.com/duanxldragon/pantheon-base/backend/pkg/database"
	"github.com/duanxldragon/pantheon-base/backend/pkg/testmysql"
)

//go:embed migrations/*.sql
var hostBackfillMigrations embed.FS

// TestHostStateVersionBackfillEnablesCAS proves migration 000032 does what the
// host fix relies on: a legacy row created at version 0 is moved to 1, and from
// then on compare-and-swap rejects a stale expectation instead of falling
// through to lifecycle validation.
//
// The row is inserted with raw SQL on purpose. Seeding through the model would
// apply the column default (now 1) and hide the exact state this migration has
// to repair.
func TestHostStateVersionBackfillEnablesCAS(t *testing.T) {
	for _, name := range []string{"missing_table", "missing_column"} {
		t.Run(name, func(t *testing.T) {
			db := testmysql.Open(t)
			if name == "missing_column" {
				if err := db.Exec("CREATE TABLE biz_cmdb_host (id BIGINT PRIMARY KEY, hostname VARCHAR(64) NOT NULL)").Error; err != nil {
					t.Fatalf("create host table without state_version: %v", err)
				}
				if err := db.Exec("INSERT INTO biz_cmdb_host (id, hostname) VALUES (1, 'legacy-without-version')").Error; err != nil {
					t.Fatalf("seed host without state_version: %v", err)
				}
			}

			m := newHostBackfillRunner(t, hostBackfillTestDSN(t, db))
			if err := m.Up(); err != nil {
				t.Fatalf("run guarded backfill: %v", err)
			}
			assertHostBackfillApplied(t, m)
			if got := db.Migrator().HasTable("biz_cmdb_host"); got != (name == "missing_column") {
				t.Fatalf("backfill changed host table existence: %v", got)
			}
			if db.Migrator().HasColumn("biz_cmdb_host", "state_version") {
				t.Fatal("backfill unexpectedly created state_version")
			}
			if name == "missing_column" {
				var hostname string
				if err := db.Raw("SELECT hostname FROM biz_cmdb_host WHERE id = 1").Scan(&hostname).Error; err != nil {
					t.Fatalf("read host after guarded backfill: %v", err)
				}
				if hostname != "legacy-without-version" {
					t.Fatalf("backfill changed a host without state_version: %q", hostname)
				}
			}
		})
	}
	t.Run("down_reapply_and_cas", testHostBackfillDownReapplyAndCAS)
}

func testHostBackfillDownReapplyAndCAS(t *testing.T) {
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

	// Roll back the module-owned migration, then really execute its SQL again.
	m := newHostBackfillRunner(t, dsn)
	if err := m.Down(); err != nil {
		t.Fatalf("roll back host backfill: %v", err)
	}
	if _, _, err := m.Version(); err != migrate.ErrNilVersion {
		t.Fatalf("expected no module migration version after down, got %v", err)
	}
	if got := hostStateVersionOf(t, db, legacyIP); got != 1 {
		t.Fatalf("down must retain backfilled version 1, got %d", got)
	}
	if got := hostStateVersionOf(t, db, currentIP); got != 5 {
		t.Fatalf("down must retain current version 5, got %d", got)
	}

	// A new zero-version row proves reapply executes SQL, not ErrNoChange.
	replayIP := "10.9.9.33"
	if err := db.Exec(
		"INSERT INTO biz_cmdb_host (hostname, ip, os, state_version, status) VALUES (?, ?, 'linux', 0, 'pending')",
		"replay-host", replayIP,
	).Error; err != nil {
		t.Fatalf("seed host before reapply: %v", err)
	}
	if err := m.Up(); err != nil {
		t.Fatalf("reapply host backfill after down: %v", err)
	}
	assertHostBackfillApplied(t, m)
	for ip, want := range map[string]uint64{legacyIP: 1, currentIP: 5, replayIP: 1} {
		if got := hostStateVersionOf(t, db, ip); got != want {
			t.Fatalf("state version after reapply for %s: want %d, got %d", ip, want, got)
		}
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
// package (this file lives in host_test and validates the assembled migration chain).
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

func newHostBackfillRunner(t *testing.T, dsn string) *migrate.Migrate {
	t.Helper()

	source, err := iofs.New(hostBackfillMigrations, "migrations")
	if err != nil {
		t.Fatalf("create module migration source: %v", err)
	}
	cfg, err := mysqlDriver.ParseDSN(dsn)
	if err != nil {
		t.Fatalf("parse isolated migration dsn: %v", err)
	}
	cfg.MultiStatements = true
	sqlDB, err := sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		t.Fatalf("open isolated migration connection: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	driver, err := migrateMySQL.WithInstance(sqlDB, &migrateMySQL.Config{})
	if err != nil {
		t.Fatalf("create migration database driver: %v", err)
	}
	m, err := migrate.NewWithInstance("iofs", source, "mysql", driver)
	if err != nil {
		t.Fatalf("create module migration runner: %v", err)
	}
	t.Cleanup(func() { _, _ = m.Close() })
	return m
}

func assertHostBackfillApplied(t *testing.T, m *migrate.Migrate) {
	t.Helper()

	version, dirty, err := m.Version()
	if err != nil || dirty || version != 32 {
		t.Fatalf("expected clean host backfill version 32, got version=%d dirty=%v err=%v", version, dirty, err)
	}
}
