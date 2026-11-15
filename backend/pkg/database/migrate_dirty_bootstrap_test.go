package database

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/golang-migrate/migrate/v4"
	"gorm.io/gorm"

	"github.com/duanxldragon/pantheon-base/backend/pkg/testmysql"
)

// A recorded failure outranks all compatibility markers, even when the failed
// version is newer than the embedded source. Neither startup entry point nor
// the direct bootstrap writer may infer a successful version from schema shape.
func TestDirtyMigrationBootstrapPreservesRecordedState(t *testing.T) {
	for _, shape := range []string{"current", "pre-module", "pre-menu", "partial", "empty"} {
		for _, version := range []int{37, currentRuntimeSchemaVersion, 2, 0, -1} {
			for _, entry := range []string{"run", "bootstrap", "writer"} {
				t.Run(fmt.Sprintf("%s/v%d/%s", shape, version, entry), func(t *testing.T) {
					assertDirtyBootstrapPreservesRecordedState(t, shape, version, entry)
				})
			}
		}
	}
}

func assertDirtyBootstrapPreservesRecordedState(t *testing.T, shape string, version int, entry string) {
	t.Helper()
	db := testmysql.Open(t)
	dsn := migrationTestDSN(t, db)
	seedDirtyBootstrapShape(t, db, shape)
	seedBootstrapVersion(t, db, version, true)
	installBootstrapWriteWitness(t, db)
	beforeState := bootstrapStateSnapshot(t, db)
	beforeSchema := bootstrapSchemaSnapshot(t, db)
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	switch entry {
	case "run":
		err = RunMigrations(dsn)
	case "bootstrap":
		err = bootstrapExistingCurrentSchema(dsn)
	case "writer":
		err = bootstrapMigrationVersion(sqlDB, currentRuntimeSchemaVersion)
	}
	var dirtyErr migrate.ErrDirty
	if !errors.As(err, &dirtyErr) || dirtyErr.Version != version {
		t.Errorf("%s returned %v, want typed dirty version %d", entry, err, version)
	}
	if after := bootstrapStateSnapshot(t, db); after != beforeState {
		t.Errorf("record rewritten: before=%s after=%s", beforeState, after)
	}
	if after := bootstrapSchemaSnapshot(t, db); after != beforeSchema {
		t.Errorf("schema changed after startup refusal: before=%s after=%s", beforeSchema, after)
	}
	if MigrationsHealthy(db) {
		t.Error("dirty migration state reported healthy")
	}
	assertBootstrapWriteCount(t, db, 0)
	t.Logf("entry=%s shape=%s before=%s after=%s error=%v", entry, shape, beforeState, bootstrapStateSnapshot(t, db), err)
}

func TestMigrationBootstrapRejectsUnknownState(t *testing.T) {
	cases := []struct {
		name, ddl, seed string
	}{
		{"multiple-clean", "version BIGINT, dirty BOOLEAN", "(24, false), (37, false)"},
		{"multiple-dirty", "version BIGINT, dirty BOOLEAN", "(24, false), (37, true)"},
		{"invalid-dirty", "version BIGINT, dirty INT", "(24, 2)"},
		{"null-dirty", "version BIGINT, dirty BOOLEAN", "(24, NULL)"},
		{"null-version", "version BIGINT, dirty BOOLEAN", "(NULL, false)"},
		{"invalid-version", "version VARCHAR(64), dirty BOOLEAN", "('unknown', false)"},
		{"negative-clean", "version BIGINT, dirty BOOLEAN", "(-1, false)"},
		{"missing-dirty-column", "version BIGINT", "(24)"},
	}
	for _, tc := range cases {
		for _, shape := range []string{"current", "empty"} {
			for _, entry := range []string{"run", "bootstrap", "writer"} {
				t.Run(tc.name+"/"+shape+"/"+entry, func(t *testing.T) {
					assertBootstrapRejectsUnknownState(t, tc.name, tc.ddl, tc.seed, shape, entry)
				})
			}
		}
	}
}

func assertBootstrapRejectsUnknownState(t *testing.T, name, ddl, seed, shape, entry string) {
	t.Helper()
	db := testmysql.Open(t)
	dsn := migrationTestDSN(t, db)
	seedDirtyBootstrapShape(t, db, shape)
	if err := db.Exec("CREATE TABLE schema_migrations (" + ddl + ")").Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("INSERT INTO schema_migrations VALUES " + seed).Error; err != nil {
		t.Fatal(err)
	}
	installBootstrapWriteWitness(t, db)
	beforeRows := unknownBootstrapRowsSnapshot(t, db)
	beforeSchema := bootstrapSchemaSnapshot(t, db)
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	switch entry {
	case "run":
		err = RunMigrations(dsn)
	case "bootstrap":
		err = bootstrapExistingCurrentSchema(dsn)
	case "writer":
		err = bootstrapMigrationVersion(sqlDB, currentRuntimeSchemaVersion)
	}
	if err == nil {
		t.Error("unknown migration state accepted")
	}
	assertMixedDirtyRecord(t, name, err)
	if after := unknownBootstrapRowsSnapshot(t, db); after != beforeRows {
		t.Errorf("unknown state changed: before=%s after=%s", beforeRows, after)
	}
	if bootstrapSchemaSnapshot(t, db) != beforeSchema {
		t.Error("schema changed after unknown state refusal")
	}
	if MigrationsHealthy(db) {
		t.Error("unknown migration state reported healthy")
	}
	assertBootstrapWriteCount(t, db, 0)
}

func assertMixedDirtyRecord(t *testing.T, name string, err error) {
	t.Helper()
	if name != "multiple-dirty" {
		return
	}
	var dirtyErr migrate.ErrDirty
	if !errors.As(err, &dirtyErr) || dirtyErr.Version != 37 {
		t.Errorf("mixed records returned %v, want dirty version 37", err)
	}
}

func TestMigrationBootstrapRetainsCleanAndAbsentState(t *testing.T) {
	cases := []struct {
		name, shape string
		version     int
		recorded    bool
		emptyTable  bool
		wantVersion int
	}{
		{"fresh", "empty", 0, false, false, -1},
		{"legacy-current", "current", 0, false, false, currentRuntimeSchemaVersion},
		{"legacy-pre-module", "pre-module", 0, false, false, moduleRegistrationCompatMigrationVersion - 1},
		{"legacy-pre-menu", "pre-menu", 0, false, false, menuHideInNavCompatMigrationVersion - 1},
		{"legacy-empty-record", "current", 0, false, true, currentRuntimeSchemaVersion},
		{"clean-lower", "current", 2, true, false, currentRuntimeSchemaVersion},
		{"clean-current", "current", currentRuntimeSchemaVersion, true, false, currentRuntimeSchemaVersion},
		{"clean-future", "current", 37, true, false, 37},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assertBootstrapRetainsCleanAndAbsentState(t, tc.name, tc.shape, tc.version, tc.recorded, tc.emptyTable, tc.wantVersion)
		})
	}
}

func assertBootstrapRetainsCleanAndAbsentState(t *testing.T, name, shape string, seedVersion int, isRecorded, isEmptyTable bool, wantVersion int) {
	t.Helper()
	db := testmysql.Open(t)
	seedDirtyBootstrapShape(t, db, shape)
	if isRecorded {
		seedBootstrapVersion(t, db, seedVersion, false)
		installBootstrapWriteWitness(t, db)
	} else if isEmptyTable {
		if err := db.Exec("CREATE TABLE schema_migrations (version BIGINT NOT NULL PRIMARY KEY, dirty BOOLEAN NOT NULL)").Error; err != nil {
			t.Fatal(err)
		}
	}
	if MigrationsHealthy(db) != isRecorded {
		t.Error("initial readiness does not match clean recorded state")
	}
	if err := bootstrapExistingCurrentSchema(migrationTestDSN(t, db)); err != nil {
		t.Fatal(err)
	}
	assertRecordedMigrationClean(t, db, wantVersion)
	if isRecorded && seedVersion >= wantVersion {
		assertBootstrapWriteCount(t, db, 0)
	}
	// The real startup path must still finish fresh and legacy databases.
	// Future versions are only a bootstrap/readiness contract: the current
	// embedded source cannot apply an unknown future migration.
	if name != "clean-future" {
		runBootstrapRealStartupPath(t, db)
	}
}

func assertRecordedMigrationClean(t *testing.T, db *gorm.DB, wantVersion int) {
	t.Helper()
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	version, dirty, recorded, err := recordedMigrationVersion(sqlDB)
	if err != nil || dirty || recorded != (wantVersion >= 0) || (recorded && version != wantVersion) {
		t.Fatalf("version=%d dirty=%v recorded=%v err=%v, want clean %d", version, dirty, recorded, err, wantVersion)
	}
	if MigrationsHealthy(db) != recorded {
		t.Error("readiness does not match clean recorded state after bootstrap")
	}
}

func runBootstrapRealStartupPath(t *testing.T, db *gorm.DB) {
	t.Helper()
	if err := RunMigrations(migrationTestDSN(t, db)); err != nil {
		t.Fatal(err)
	}
	assertLatestMigrationVersion(t, db)
	if !MigrationsHealthy(db) {
		t.Error("successful startup reported unhealthy")
	}
}

func TestMigrationStateReadFailureClosesSafely(t *testing.T) {
	if MigrationsHealthy(nil) {
		t.Fatal("nil database reported healthy")
	}
	db := testmysql.Open(t)
	seedBootstrapVersion(t, db, 24, false)
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	if err := sqlDB.Close(); err != nil {
		t.Fatal(err)
	}
	if err := bootstrapMigrationVersion(sqlDB, 24); err == nil {
		t.Error("closed connection accepted for bootstrap")
	}
	if MigrationsHealthy(db) {
		t.Error("unreadable database reported healthy")
	}
}

func installBootstrapWriteWitness(t *testing.T, db *gorm.DB) {
	t.Helper()
	// MyISAM keeps the witness even if an attempted state transaction rolls
	// back, so zero entries proves no INSERT/UPDATE/DELETE was attempted.
	if err := db.Exec("CREATE TABLE migration_write_witness (action VARCHAR(16)) ENGINE=MyISAM").Error; err != nil {
		t.Fatal(err)
	}
	for _, action := range []string{"INSERT", "UPDATE", "DELETE"} {
		statement := "CREATE TRIGGER witness_" + strings.ToLower(action) + " BEFORE " + action +
			" ON schema_migrations FOR EACH ROW INSERT INTO migration_write_witness VALUES ('" + action + "')"
		if err := db.Exec(statement).Error; err != nil {
			t.Fatal(err)
		}
	}
}

func assertBootstrapWriteCount(t *testing.T, db *gorm.DB, want int64) {
	t.Helper()
	var count int64
	if err := db.Table("migration_write_witness").Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != want {
		t.Errorf("migration record write attempts=%d, want %d", count, want)
	}
}

func unknownBootstrapRowsSnapshot(t *testing.T, db *gorm.DB) string {
	t.Helper()
	rows, err := db.Raw("SELECT * FROM schema_migrations ORDER BY version").Rows()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = rows.Close() }()
	columns, err := rows.Columns()
	if err != nil {
		t.Fatal(err)
	}
	var result []string
	for rows.Next() {
		values := make([]sql.RawBytes, len(columns))
		targets := make([]any, len(columns))
		for i := range values {
			targets[i] = &values[i]
		}
		if err := rows.Scan(targets...); err != nil {
			t.Fatal(err)
		}
		result = append(result, fmt.Sprintf("%x", values))
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return strings.Join(result, ";")
}

func seedDirtyBootstrapShape(t *testing.T, db *gorm.DB, shape string) {
	t.Helper()
	if shape == "empty" {
		return
	}
	if shape == "partial" {
		if err := db.Exec("CREATE TABLE system_menu (id BIGINT PRIMARY KEY, active_menu VARCHAR(255))").Error; err != nil {
			t.Fatal(err)
		}
		return
	}
	seedCurrentSchemaBootstrapMarkers(t, db)
	// The existing legacy fixture lacks the tenant markers needed to prove v24.
	// Pre-compatibility schemas retain the original legacy shape so replay can
	// build their user tables in full rather than encounter partial v24 tables.
	if shape == "current" {
		for _, statement := range []string{
			"CREATE TABLE system_user (id BIGINT PRIMARY KEY, tenant_id BIGINT NOT NULL DEFAULT 0)",
			"ALTER TABLE system_role ADD COLUMN tenant_id BIGINT NOT NULL DEFAULT 0",
			"CREATE TABLE system_user_role (user_id BIGINT, role_id BIGINT, tenant_id BIGINT NOT NULL DEFAULT 0)",
		} {
			if err := db.Exec(statement).Error; err != nil {
				t.Fatal(err)
			}
		}
	}
	switch shape {
	case "pre-module":
		if err := db.Exec("DROP TABLE system_module_registration").Error; err != nil {
			t.Fatal(err)
		}
	case "pre-menu":
		dropMigrationColumnIfExists(t, db, "system_menu", "hide_in_nav")
	}
}

func seedBootstrapVersion(t *testing.T, db *gorm.DB, version int, dirty bool) {
	t.Helper()
	if err := db.Exec("CREATE TABLE schema_migrations (version BIGINT NOT NULL PRIMARY KEY, dirty BOOLEAN NOT NULL)").Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("INSERT INTO schema_migrations (version, dirty) VALUES (?, ?)", version, dirty).Error; err != nil {
		t.Fatal(err)
	}
}

// Compare both recorded values and their SQL byte serialization, including row
// count. A schema snapshot additionally catches migration DDL on refusal.
func bootstrapStateSnapshot(t *testing.T, db *gorm.DB) string {
	t.Helper()
	var rows []string
	if err := db.Raw("SELECT CONCAT(version, ':', dirty, ':', HEX(CAST(version AS CHAR)), ':', HEX(CAST(dirty AS CHAR))) FROM schema_migrations ORDER BY version").Scan(&rows).Error; err != nil {
		t.Fatal(err)
	}
	return strings.Join(rows, ",")
}

func bootstrapSchemaSnapshot(t *testing.T, db *gorm.DB) string {
	t.Helper()
	var rows []string
	if err := db.Raw(`SELECT CONCAT(table_name, ':', column_name, ':', column_type, ':', is_nullable, ':', COALESCE(column_default, '<NULL>'))
FROM information_schema.columns WHERE table_schema = DATABASE() ORDER BY table_name, ordinal_position`).Scan(&rows).Error; err != nil {
		t.Fatal(err)
	}
	return strings.Join(rows, ",")
}
