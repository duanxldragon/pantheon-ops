package database

import (
	"testing"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"gorm.io/gorm"

	"github.com/duanxldragon/pantheon-base/backend/pkg/testmysql"
)

// TestMigrationChainRollsBackAndReapplies proves the down path is re-runnable.
//
// 000020.down used to drop indexes and columns unconditionally, so the rollback
// aborted as soon as a later migration (000024.down) had already removed them.
// The file now guards every destructive step on information_schema. This test is
// the evidence: it migrates a throwaway database all the way up, all the way
// down, and all the way up again.
func TestMigrationChainRollsBackAndReapplies(t *testing.T) {
	db := testmysql.Open(t)
	dsn := migrationTestDSN(t, db)

	if err := RunMigrations(dsn); err != nil {
		t.Fatalf("run migrations up: %v", err)
	}

	m := newMigrationRunner(t, dsn)

	version, dirty, err := m.Version()
	if err != nil {
		t.Fatalf("read schema version before rollback: %v", err)
	}
	if dirty {
		t.Fatalf("schema version %d is dirty before rollback", version)
	}

	if err := m.Down(); err != nil && err != migrate.ErrNoChange {
		t.Fatalf("roll migrations down from version %d: %v", version, err)
	}
	if _, _, err := m.Version(); err != migrate.ErrNilVersion {
		t.Fatalf("expected ErrNilVersion after full rollback, got %v", err)
	}

	// The tenant dimension must be gone, not half-dropped: 000020.down is the
	// migration that owns system_user.tenant_id on the way down.
	assertMigrationColumnAbsent(t, db, "system_user", "tenant_id")
	assertMigrationColumnAbsent(t, db, "system_role", "tenant_id")

	// Re-applying must succeed on the schema the rollback left behind. If any
	// down file left a stray index or column, this is where it surfaces.
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		t.Fatalf("re-apply migrations after rollback: %v", err)
	}

	reapplied, dirty, err := m.Version()
	if err != nil {
		t.Fatalf("read schema version after re-apply: %v", err)
	}
	if dirty {
		t.Fatalf("schema version %d is dirty after re-apply", reapplied)
	}
	if reapplied != version {
		t.Fatalf("expected to return to version %d after rollback and re-apply, got %d", version, reapplied)
	}

	assertMigrationColumnExists(t, db, "system_user", "tenant_id")
}

func assertMigrationColumnAbsent(t *testing.T, db *gorm.DB, table string, column string) {
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
	if count != 0 {
		t.Fatalf("expected column %s.%s to be dropped by the rollback, got count=%d", table, column, count)
	}
}

func newMigrationRunner(t *testing.T, dsn string) *migrate.Migrate {
	t.Helper()

	source, err := iofs.New(migrationFS, "migrations")
	if err != nil {
		t.Fatalf("create migration source: %v", err)
	}
	migrateDSN, err := buildMigrateDSN(dsn)
	if err != nil {
		t.Fatalf("build migrate dsn: %v", err)
	}
	m, err := migrate.NewWithSourceInstance("iofs", source, migrateDSN)
	if err != nil {
		t.Fatalf("create migrate instance: %v", err)
	}
	t.Cleanup(func() { _, _ = m.Close() })

	return m
}
