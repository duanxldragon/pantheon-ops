package database

import (
	"testing"

	"github.com/golang-migrate/migrate/v4"
	"gorm.io/gorm"

	"github.com/duanxldragon/pantheon-base/backend/pkg/testmysql"
)

// TestDropPolicyTableDownReconstructsVersion30Shape pins 000031.down to the
// shape that actually existed at version 30.
//
// The first version of 000031.down recreated the table from 000001 and then
// bolted on the tenant-scoped unique index from 000024 — but 000024 also DROPS
// the role_key-only unique index, so the rollback produced a table that no
// version ever had. The extra index is not cosmetic: it makes role_key globally
// unique again, so a rolled-back deployment would reject the same role_key in
// two different tenants, which multi-tenant mode allows.
//
// The assertion is a byte comparison of SHOW CREATE TABLE between
// (a) a schema migrated straight up to 30 and (b) a schema migrated to 31 and
// then rolled back to 30. Any future edit to 000031.down that drifts from the
// real v30 shape fails here instead of in a restore drill.
func TestDropPolicyTableDownReconstructsVersion30Shape(t *testing.T) {
	db := testmysql.Open(t)
	dsn := migrationTestDSN(t, db)

	m := newMigrationRunner(t, dsn)

	// (a) Authoritative v30: fresh schema migrated straight up to 30.
	if err := m.Migrate(30); err != nil && err != migrate.ErrNoChange {
		t.Fatalf("migrate to 30 from empty: %v", err)
	}
	atThirty := showCreateTable(t, db)

	// (b) Same schema at 31, then rolled back through 000031.down.
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		t.Fatalf("migrate up to 31: %v", err)
	}
	if err := m.Migrate(30); err != nil && err != migrate.ErrNoChange {
		t.Fatalf("migrate back down to 30: %v", err)
	}
	afterDown := showCreateTable(t, db)

	if afterDown != atThirty {
		t.Fatalf("000031.down does not reconstruct the version 30 shape\n--- version 30 ---\n%s\n--- after 000031.down ---\n%s", atThirty, afterDown)
	}
}

// showCreateTable returns the canonical DDL MySQL reports for the dropped table.
func showCreateTable(t *testing.T, db *gorm.DB) string {
	t.Helper()

	var name, ddl string
	if err := db.Raw("SHOW CREATE TABLE `permission_role_data_scope_policy`").Row().Scan(&name, &ddl); err != nil {
		t.Fatalf("show create table permission_role_data_scope_policy: %v", err)
	}
	return ddl
}
