package database

import (
	"embed"
	"regexp"
	"strings"
	"testing"

	"github.com/golang-migrate/migrate/v4"

	"github.com/duanxldragon/pantheon-base/backend/pkg/testmysql"
)

// The preflight script is embedded here, not in migrate.go: production must not
// ship it, and `//go:embed migrations/*.sql` deliberately does not recurse, so
// the subdirectory stays out of the migration source. Embedding it in the test
// keeps the file operators actually run identical to the one exercised below —
// a copy-pasted script in a runbook would drift the first time someone edited
// the table shape.
//
//go:embed migrations/preflight/*.sql
var preflightFS embed.FS

// TestPreflight000031ReportsWhatTheDropWouldDestroy pins the operator-facing
// preflight script to reality.
//
// 000031 is `DROP TABLE` and its down migration restores structure only, so the
// only thing standing between a production database and silent data loss is a
// human reading the preflight output before deploying. That makes the preflight
// itself safety-critical: if it under-reports the row count, or claims the
// replacement table exists when it does not, the human approves the drop on
// false information.
//
// This test builds a schema-30 database, puts rows in the doomed table, runs the
// exact statements from the script, and checks each number it is supposed to
// surface — then applies the migration and confirms what the script predicted
// actually happened.
func TestPreflight000031ReportsWhatTheDropWouldDestroy(t *testing.T) {
	db := testmysql.Open(t)
	dsn := migrationTestDSN(t, db)
	m := newMigrationRunner(t, dsn)

	if err := m.Migrate(30); err != nil && err != migrate.ErrNoChange {
		t.Fatalf("migrate to 30 from empty: %v", err)
	}

	// Two rows across two tenants: the shape 000024 made legal (same role_key,
	// different tenant), which is exactly why the table was superseded.
	seed := []struct {
		tenantID uint64
		roleKey  string
	}{
		{0, "platform_admin"},
		{7, "platform_admin"},
	}
	for _, row := range seed {
		if err := db.Exec(
			"INSERT INTO `permission_role_data_scope_policy` (tenant_id, role_key, mode, created_at, updated_at) VALUES (?, ?, 'all', NOW(3), NOW(3))",
			row.tenantID, row.roleKey,
		).Error; err != nil {
			t.Fatalf("seed permission_role_data_scope_policy: %v", err)
		}
	}

	statements := preflightStatements(t)

	// The script must not reach production. `//go:embed migrations/*.sql` does
	// not recurse, so the subdirectory stays out of migrationFS — but that is a
	// property of the pattern, not of the directory, and a future widening of
	// the pattern to `migrations` would pull the preflight in as a malformed
	// migration. Pin it instead of trusting the comment above.
	if _, err := migrationFS.ReadFile("migrations/preflight/000031_preflight.sql"); err == nil {
		t.Fatal("preflight script is embedded in migrationFS — production would ship it as a migration file")
	}

	// Statement 1: the version the operator must read before trusting the rest.
	var version int64
	var dirty bool
	if err := db.Raw(statements[0]).Row().Scan(&version, &dirty); err != nil {
		t.Fatalf("preflight statement 1 (schema version): %v", err)
	}
	if version != 30 || dirty {
		t.Fatalf("preflight reports version=%d dirty=%v, want 30/false", version, dirty)
	}

	// Statement 2: the doomed table is present.
	var deadTableExists int64
	if err := db.Raw(statements[1]).Row().Scan(&deadTableExists); err != nil {
		t.Fatalf("preflight statement 2 (dead table exists): %v", err)
	}
	if deadTableExists != 1 {
		t.Fatalf("preflight reports dead_table_exists=%d, want 1", deadTableExists)
	}

	// Statement 3: the number the operator makes the discard decision on.
	var rowsToBeDropped int64
	if err := db.Raw(statements[2]).Row().Scan(&rowsToBeDropped); err != nil {
		t.Fatalf("preflight statement 3 (rows to be dropped): %v", err)
	}
	if rowsToBeDropped != int64(len(seed)) {
		t.Fatalf("preflight reports rows_to_be_dropped=%d, want %d — an under-report here approves a destructive deploy on false information", rowsToBeDropped, len(seed))
	}

	// Statements 4 and 5: the replacement table exists and is readable.
	var replacementExists int64
	if err := db.Raw(statements[3]).Row().Scan(&replacementExists); err != nil {
		t.Fatalf("preflight statement 4 (replacement exists): %v", err)
	}
	if replacementExists != 1 {
		t.Fatalf("preflight reports replacement_table_exists=%d, want 1", replacementExists)
	}

	var replacementRows int64
	if err := db.Raw(statements[4]).Row().Scan(&replacementRows); err != nil {
		t.Fatalf("preflight statement 5 (replacement rows): %v", err)
	}

	// Statement 6 lists the rows themselves; it must run without error so the
	// operator can eyeball what is about to disappear.
	var sampleCount int64
	if err := db.Raw("SELECT COUNT(*) FROM (" + statements[5] + ") AS sample").Row().Scan(&sampleCount); err != nil {
		t.Fatalf("preflight statement 6 (sample rows): %v", err)
	}
	if sampleCount != int64(len(seed)) {
		t.Fatalf("preflight sample lists %d rows, want %d", sampleCount, len(seed))
	}

	// Now do what the operator would do next, and confirm the prediction held.
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		t.Fatalf("apply 000031: %v", err)
	}

	if db.Migrator().HasTable("permission_role_data_scope_policy") {
		t.Fatal("000031 applied but permission_role_data_scope_policy still exists")
	}
	if !db.Migrator().HasTable("system_role_data_scope") {
		t.Fatal("000031 removed the replacement table system_role_data_scope as well — the drop is not narrow")
	}
}

// preflightStatements returns the script's executable statements in order.
// The file separates statements with a lone `;` so it works both as a
// `mysql` script and as a sequence the test can assert on individually.
//
// The separator is matched as a regexp anchored to the whole line rather than
// as the literal "\n;\n": the file is checked out with CRLF on Windows, and a
// literal split would then see one giant statement and report a confusing
// "parsed 1" instead of the assertion that actually matters.
func preflightStatements(t *testing.T) []string {
	t.Helper()

	raw, err := preflightFS.ReadFile("migrations/preflight/000031_preflight.sql")
	if err != nil {
		t.Fatalf("read preflight script: %v", err)
	}

	// The script is asserted to hold exactly six statements below.
	statements := make([]string, 0, 6)
	for _, chunk := range preflightSeparator.Split(string(raw), -1) {
		var body []string
		for _, line := range strings.Split(chunk, "\n") {
			if strings.HasPrefix(strings.TrimSpace(line), "--") {
				continue
			}
			body = append(body, line)
		}
		statement := strings.TrimSpace(strings.Join(body, "\n"))
		if statement == "" {
			continue
		}
		statements = append(statements, statement)
	}

	// 1 version / 2 dead table / 3 rows dropped / 4 replacement exists /
	// 5 replacement rows / 6 sample rows.
	if len(statements) != 6 {
		t.Fatalf("preflight script must hold 6 statements, parsed %d: %q", len(statements), statements)
	}
	return statements
}

// preflightSeparator matches the lone `;` line that ends each statement,
// tolerating CRLF and trailing spaces.
var preflightSeparator = regexp.MustCompile(`(?m)^;[ \t\r]*$`)
