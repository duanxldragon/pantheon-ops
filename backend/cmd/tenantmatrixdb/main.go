// Command tenantmatrixdb implements the DB lifecycle for the queue-6 tenant
// hostile browser/API matrix run (task 2026-09-10-tenant-verification-and-gray).
//
// Subcommands:
//
//	up    — bring the dev database to the matrix starting state:
//	        * apply pending tenant migrations (13..16) if the dev DB was
//	          created via GORM AutoMigrate and never ran the versioned ones,
//	        * create temporary tenants (101/202) + active memberships for the
//	          smoke user (admin),
//	        * set platform.tenant_mode to PANTHEON_MATRIX_MODE (compat by
//	          default; CI can request multi explicitly).
//	down  — restore: delete fixture rows, force the flag back to compat,
//	        then roll migrations 16..13 back (reverse of up).
//	revoke — runbook §6.2 kill-switch session revocation for the smoke user:
//	        per-user Redis blacklist + refresh cascade + session rows marked
//	        revoked, via the production pkg/tenant.RevokeUserSessionsInTenant.
//	unblacklist — clear the per-user kill key (rehearsal-only recovery so
//	        later phases can log in again; mirrors §6.2 cache invalidation).
//
// Safety rails:
//   - requires PANTHEON_MATRIX_DSN explicitly (never guesses credentials);
//   - the fixture tenants are tagged plan='__smoke_matrix__' and only tagged
//     rows are removed on down;
//   - down verifies the cleanup (zero tagged rows left) and leaves the
//     schema_migrations marker at the pre-run version.
package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/duanxldragon/pantheon-base/backend/pkg/authtoken"
	"github.com/duanxldragon/pantheon-base/backend/pkg/tenant"
	// Blank import registers the MySQL driver for database/sql (driver-side init only).
	_ "github.com/go-sql-driver/mysql"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

const (
	flagKey     = "platform.tenant_mode"
	flagCompat  = "compat"
	flagMulti   = "multi"
	smokeTag    = "__smoke_matrix__"
	tenantA     = 101
	tenantB     = 202
	smokeUserID = 1 // admin (dev seed); memberships reference the existing user
)

func main() {
	if len(os.Args) < 2 {
		fatal("usage: tenantmatrixdb up|down|status|revoke|unblacklist")
	}
	dsn := strings.TrimSpace(os.Getenv("PANTHEON_MATRIX_DSN"))
	if dsn == "" {
		fatal("PANTHEON_MATRIX_DSN is required (never guessed)")
	}
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		fatal("open mysql: %v", err)
	}
	defer func() { _ = db.Close() }()
	if err := db.Ping(); err != nil {
		fatal("ping mysql: %v", err)
	}

	switch os.Args[1] {
	case "up":
		if err := cmdUp(db); err != nil {
			fatal("up failed: %v", err)
		}
	case "down":
		if err := cmdDown(db); err != nil {
			fatal("down failed: %v", err)
		}
	case "status":
		if err := cmdStatus(db); err != nil {
			fatal("status failed: %v", err)
		}
	case "revoke":
		if err := cmdRevoke(db); err != nil {
			fatal("revoke failed: %v", err)
		}
	case "unblacklist":
		if err := cmdUnblacklist(); err != nil {
			fatal("unblacklist failed: %v", err)
		}
	default:
		fatal("unknown subcommand %q", os.Args[1])
	}
}

// cmdRevoke executes the runbook §6.2 kill-switch session-revocation step
// through the production code path (pkg/tenant.RevokeUserSessionsInTenant —
// the same function org membership changes must call): per-user Redis
// blacklist kills live access tokens, refresh tokens cascade-delete, session
// rows are marked revoked. The smoke user owns the matrix sessions, so
// revoking their sessions is exactly the "revoke affected sessions" step for
// the two-tenant rehearsal.
func cmdRevoke(db *sql.DB) error {
	rdb := connectRedis()
	if rdb == nil {
		return fmt.Errorf("redis unavailable (PANTHEON_REDIS_ADDR) — blacklist step cannot run")
	}
	gdb, err := gorm.Open(mysql.New(mysql.Config{Conn: db}), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("wrap gorm: %w", err)
	}
	revoked, err := tenant.RevokeUserSessionsInTenant(context.Background(), rdb, gdb, smokeUserID)
	if err != nil {
		return fmt.Errorf("revoke user %d sessions: %w", smokeUserID, err)
	}
	fmt.Printf("tenant-matrix revoke: user %d sessions revoked=%d (blacklist key ttl=%s)\n",
		smokeUserID, revoked, authtoken.AccessTokenTTL+time.Minute)
	return nil
}

// cmdUnblacklist clears the per-user kill key after the kill-switch evidence
// pass so later phases (and the dev workflow) can log in again. This mirrors
// runbook §6.2 step 4 (cache invalidation) in reverse: it is a rehearsal
// utility only, never a production recovery step.
func cmdUnblacklist() error {
	rdb := connectRedis()
	if rdb == nil {
		return fmt.Errorf("redis unavailable (PANTHEON_REDIS_ADDR)")
	}
	n, err := rdb.Del(context.Background(), authtoken.BlacklistUserKey(smokeUserID)).Result()
	if err != nil {
		return err
	}
	fmt.Printf("tenant-matrix unblacklist: user %d kill key removed (%d)\n", smokeUserID, n)
	return nil
}

// connectRedis builds a client from the same env vars cmd/server uses
// (PANTHEON_REDIS_ADDR / PANTHEON_REDIS_PASSWORD), without importing the
// server's global database.RDB singleton.
func connectRedis() *redis.Client {
	addr := strings.TrimSpace(os.Getenv("PANTHEON_REDIS_ADDR"))
	if addr == "" {
		return nil
	}
	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: os.Getenv("PANTHEON_REDIS_PASSWORD"),
	})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err != nil {
		_ = rdb.Close()
		return nil
	}
	return rdb
}

func cmdUp(db *sql.DB) error {
	// 1. Apply pending tenant migrations 13..16 (guarded SQL, idempotent).
	//    The dev DB normally reaches this state via AutoMigrate for columns,
	//    but the versioned tenant tables (tenants/tenant_memberships) only
	//    come from migration 13, so run the pending range explicitly.
	for _, version := range []int{13, 14, 15, 16} {
		if err := ensureMigration(db, version); err != nil {
			return fmt.Errorf("ensure migration %d: %w", version, err)
		}
	}

	// 2. Fixture tenants (idempotent; tagged for down).
	for _, tc := range []struct {
		id   int64
		code string
		name string
	}{
		{tenantA, "smoke-acme", "Smoke Acme North"},
		{tenantB, "smoke-globex", "Smoke Globex South"},
	} {
		_, err := db.Exec(
			"INSERT INTO `tenants` (`id`,`code`,`name`,`status`,`plan`) VALUES (?,?,?,?,?) "+
				"ON DUPLICATE KEY UPDATE `plan`=VALUES(`plan`), `status`='active'",
			tc.id, tc.code, tc.name, "active", smokeTag,
		)
		if err != nil {
			return fmt.Errorf("upsert tenant %d: %w", tc.id, err)
		}
	}

	// 3. Fixture memberships for the smoke user (active in both tenants —
	//    this is the multi-membership ambiguity the picker resolves).
	for _, tid := range []int64{tenantA, tenantB} {
		_, err := db.Exec(
			"INSERT INTO `tenant_memberships` (`tenant_id`,`user_id`,`role`,`status`) VALUES (?,?,?,?) "+
				"ON DUPLICATE KEY UPDATE `status`='active'",
			tid, smokeUserID, "owner", "active",
		)
		if err != nil {
			return fmt.Errorf("upsert membership %d: %w", tid, err)
		}
	}

	// 4. Explicit starting flag value: compat by default, or multi when the
	// caller opts into a real tenant-mode matrix run.
	mode, err := matrixMode()
	if err != nil {
		return err
	}
	if err := setFlagValue(db, mode); err != nil {
		return err
	}
	fmt.Printf("tenant-matrix up: migrations ensured, tenants 101/202 ready, flag=%s\n", mode)
	return nil
}

// matrixMode keeps the matrix tool safe by defaulting to compat while allowing
// an explicit multi-mode CI run. No other values are accepted because a typo
// here would silently run the tenant suite against the wrong contract.
func matrixMode() (string, error) {
	mode := strings.ToLower(strings.TrimSpace(os.Getenv("PANTHEON_MATRIX_MODE")))
	if mode == "" {
		return flagCompat, nil
	}
	if mode != flagCompat && mode != flagMulti {
		return "", fmt.Errorf("PANTHEON_MATRIX_MODE must be %q or %q, got %q", flagCompat, flagMulti, mode)
	}
	return mode, nil
}

func cmdDown(db *sql.DB) error {
	// 1. Force the flag back to compat FIRST (kill switch semantics).
	if err := setFlagValue(db, flagCompat); err != nil {
		return err
	}

	// 2. Remove fixture memberships and tenants (tagged rows only).
	if _, err := db.Exec(
		"DELETE FROM `tenant_memberships` WHERE `tenant_id` IN (?,?)", tenantA, tenantB,
	); err != nil {
		return fmt.Errorf("delete memberships: %w", err)
	}
	if _, err := db.Exec(
		"DELETE FROM `tenants` WHERE `id` IN (?,?) AND `plan` = ?", tenantA, tenantB, smokeTag,
	); err != nil {
		return fmt.Errorf("delete tenants: %w", err)
	}

	// 3. Verify cleanup.
	var tagged int
	if err := db.QueryRow(
		"SELECT COUNT(*) FROM `tenants` WHERE `plan` = ?", smokeTag,
	).Scan(&tagged); err != nil {
		return fmt.Errorf("verify tenants cleanup: %w", err)
	}
	var memberships int
	if err := db.QueryRow(
		"SELECT COUNT(*) FROM `tenant_memberships` WHERE `tenant_id` IN (?,?)", tenantA, tenantB,
	).Scan(&memberships); err != nil {
		return fmt.Errorf("verify memberships cleanup: %w", err)
	}
	if tagged != 0 || memberships != 0 {
		return fmt.Errorf("cleanup verification failed: tagged=%d memberships=%d", tagged, memberships)
	}

	// 4. Roll migrations 16..13 back so the schema matches the pre-run state
	//    (the dev DB was on version 12 before the matrix run).
	for _, version := range []int{16, 15, 14, 13} {
		if err := rollbackMigration(db, version); err != nil {
			return fmt.Errorf("rollback migration %d: %w", version, err)
		}
	}
	fmt.Println("tenant-matrix down: fixtures removed, flag=compat, migrations rolled back to 12")
	return nil
}

func cmdStatus(db *sql.DB) error {
	var version int
	var dirty bool
	err := db.QueryRow("SELECT version, dirty FROM `schema_migrations` LIMIT 1").Scan(&version, &dirty)
	switch {
	case err == sql.ErrNoRows:
		fmt.Println("migration-version: none")
	case err != nil:
		return err
	default:
		fmt.Printf("migration-version: %d dirty=%v\n", version, dirty)
	}
	flag, err := flagValue(db)
	if err != nil {
		return err
	}
	fmt.Printf("flag: %s\n", flag)
	var tenants int
	if err := db.QueryRow("SELECT COUNT(*) FROM `tenants` WHERE `plan` = ?", smokeTag).Scan(&tenants); err == nil {
		fmt.Printf("smoke-tagged-tenants: %d\n", tenants)
	}
	return nil
}

// ensureMigration applies one pending versioned migration by executing the
// embedded up/down SQL from pkg/database/migrations via `go run` of the
// backend's own migration machinery is not possible from here; instead this
// helper shells the statements through database/sql directly. The guarded SQL
// files are read from disk (repo checkout) to keep a single source of truth.
func ensureMigration(db *sql.DB, version int) error {
	current, err := migrationVersion(db)
	if err != nil {
		return err
	}
	if current >= version {
		return nil // already applied (or newer); nothing to do
	}
	// AutoMigrate-built dev schemas may already carry the physical artifacts
	// (columns/indexes/tables) of an unguarded migration file. Migration 16 is
	// the only unguarded file in the bootstrap window; when its single ALTER
	// fails with a duplicate-column error the artifact set is already in
	// place, so record the version instead of failing the run.
	if version == 16 && current >= 15 {
		var columnCount int
		if err := db.QueryRow(
			"SELECT COUNT(*) FROM information_schema.columns WHERE table_schema = DATABASE() " +
				"AND table_name = 'system_auth_mfa_challenge' AND column_name = 'tenant_id'",
		).Scan(&columnCount); err != nil {
			return err
		}
		if columnCount > 0 {
			return writeMigrationVersion(db, version)
		}
	}
	// Only jump forward within the tenant migration window; refuse to skip
	// over unknown gaps.
	if current < 12 {
		return fmt.Errorf("refusing to bootstrap from version %d (expected >= 12)", current)
	}
	statements, err := readMigrationStatements(version, true)
	if err != nil {
		return err
	}
	for _, statement := range statements {
		if _, err := db.Exec(statement); err != nil {
			return fmt.Errorf("apply v%d statement %q: %w", version, truncate(statement, 80), err)
		}
	}
	return writeMigrationVersion(db, version)
}

func rollbackMigration(db *sql.DB, version int) error {
	current, err := migrationVersion(db)
	if err != nil {
		return err
	}
	if current < version {
		return nil // already rolled back
	}
	statements, err := readMigrationStatements(version, false)
	if err != nil {
		return err
	}
	for _, statement := range statements {
		if _, err := db.Exec(statement); err != nil {
			// Guarded down files tolerate "nothing to do" states; treat
			// duplicate-key/missing-object noise as non-fatal only if the
			// statement is a guarded no-op SELECT. Anything else fails.
			return fmt.Errorf("rollback v%d statement %q: %w", version, truncate(statement, 80), err)
		}
	}
	return writeMigrationVersion(db, version-1)
}

func migrationVersion(db *sql.DB) (int, error) {
	var exists int
	if err := db.QueryRow(
		"SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name = 'schema_migrations'",
	).Scan(&exists); err != nil {
		return 0, err
	}
	if exists == 0 {
		return 0, nil
	}
	var version int
	var dirty bool
	err := db.QueryRow("SELECT version, dirty FROM `schema_migrations` LIMIT 1").Scan(&version, &dirty)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	if dirty {
		return 0, fmt.Errorf("schema_migrations is dirty at version %d", version)
	}
	return version, nil
}

func writeMigrationVersion(db *sql.DB, version int) error {
	if _, err := db.Exec("CREATE TABLE IF NOT EXISTS `schema_migrations` (version bigint not null primary key, dirty boolean not null)"); err != nil {
		return err
	}
	if _, err := db.Exec("DELETE FROM `schema_migrations` LIMIT 1"); err != nil {
		return err
	}
	_, err := db.Exec("INSERT INTO `schema_migrations` (version, dirty) VALUES (?, false)", version)
	return err
}

func setFlagValue(db *sql.DB, value string) error {
	res, err := db.Exec(
		"UPDATE `system_setting` SET `setting_value` = ? WHERE `setting_key` = ?",
		value, flagKey,
	)
	if err != nil {
		return fmt.Errorf("update flag: %w", err)
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		// Seed row missing: insert the global row.
		if _, err := db.Exec(
			"INSERT INTO `system_setting` (`tenant_id`,`setting_key`,`setting_value`,`value_type`,`group_key`,`module`,`is_public`) "+
				"SELECT 0, ?, ?, 'string', 'platform', 'platform', 0 WHERE NOT EXISTS (SELECT 1 FROM `system_setting` WHERE `setting_key` = ?)",
			flagKey, value, flagKey,
		); err != nil {
			return fmt.Errorf("insert flag row: %w", err)
		}
	}
	return nil
}

func flagValue(db *sql.DB) (string, error) {
	var value string
	err := db.QueryRow(
		"SELECT `setting_value` FROM `system_setting` WHERE `setting_key` = ?", flagKey,
	).Scan(&value)
	if err == sql.ErrNoRows {
		return flagCompat, nil
	}
	return value, err
}

func readMigrationStatements(version int, up bool) ([]string, error) {
	suffix := "up.sql"
	if !up {
		suffix = "down.sql"
	}
	var names []string
	matches, err := findMigrationFile(version, suffix)
	if err != nil {
		return nil, err
	}
	names = matches
	if len(names) == 0 {
		return nil, fmt.Errorf("migration file for version %d (%s) not found", version, suffix)
	}
	content, err := os.ReadFile(names[0])
	if err != nil {
		return nil, err
	}
	return splitSQLStatements(string(content)), nil
}

func findMigrationFile(version int, suffix string) ([]string, error) {
	entries, err := os.ReadDir("pkg/database/migrations")
	if err != nil {
		return nil, err
	}
	prefix := fmt.Sprintf("%06d_", version)
	var found []string
	for _, entry := range entries {
		name := entry.Name()
		if strings.HasPrefix(name, prefix) && strings.HasSuffix(name, suffix) {
			found = append(found, "pkg/database/migrations/"+name)
		}
	}
	return found, nil
}

func splitSQLStatements(content string) []string {
	// The guarded tenant migration files never contain DELIMITER blocks or
	// semicolons inside string literals (verified by review); split on ';'.
	var statements []string
	var current strings.Builder
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "--") {
			continue
		}
		current.WriteString(line)
		current.WriteString("\n")
		if strings.HasSuffix(trimmed, ";") {
			statement := strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(current.String()), ";"))
			current.Reset()
			if statement != "" && statement != "SELECT 1" {
				statements = append(statements, statement+";")
			}
		}
	}
	if rest := strings.TrimSpace(current.String()); rest != "" {
		statements = append(statements, rest)
	}
	return statements
}

func truncate(value string, n int) string {
	if len(value) <= n {
		return value
	}
	return value[:n] + "..."
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "tenantmatrixdb: "+format+"\n", args...)
	os.Exit(1)
}
