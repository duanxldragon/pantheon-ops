package middleware

import (
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
)

// dataScopeModelColumns is the model-to-column map for SystemRoleDataScope.
//
// It is written out by hand instead of being derived from the struct tags so
// that adding a field to the model forces a decision here: the test asserts the
// map covers exactly the exported fields, so a new field cannot be added
// without also declaring which column it maps to (and therefore without the
// migration chain gaining that column).
var dataScopeModelColumns = map[string]string{
	"ID":       "id",
	"TenantID": "tenant_id",
	"RoleKey":  "role_key",
	"Mode":     "mode",
	"DeptIDs":  "dept_ids",
}

// migrationChainFiles are the versioned migrations that own the
// `system_role_data_scope` shape, in application order:
//
//	000001 creates the table (id, role_key, mode, scope_defs, timestamps)
//	000002 adds dept_ids and backfills it from scope_defs
//	000024 adds tenant_id and the (tenant_id, role_key) unique key
var migrationChainFiles = []string{
	"000001_init_schema.up.sql",
	"000002_align_runtime_schema.up.sql",
	"000024_tenant_core_tables_repair.up.sql",
}

// normalizeSQL collapses whitespace and drops identifier quotes so a formatting
// change in a migration or in dataScopePolicyDDL does not fail the shape
// assertions. The migration chain quotes identifiers (ADD UNIQUE INDEX `uk_...`)
// while the bootstrap DDL does not.
func normalizeSQL(sql string) string {
	return strings.Join(strings.Fields(strings.ReplaceAll(sql, "`", "")), " ")
}

// readMigrationChain returns the concatenated up-migrations that own the table.
func readMigrationChain(t *testing.T) string {
	t.Helper()

	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot resolve the path of data_scope_schema_test.go")
	}
	migrationsDir := filepath.Join(filepath.Dir(thisFile), "..", "..", "pkg", "database", "migrations")

	var chain strings.Builder
	for _, name := range migrationChainFiles {
		raw, err := os.ReadFile(filepath.Join(migrationsDir, name))
		if err != nil {
			t.Fatalf("read migration %s: %v", name, err)
		}
		chain.Write(raw)
		chain.WriteString("\n")
	}
	return chain.String()
}

// TestDataScopeModelMatchesMigrationChain pins DB-01: the versioned migration
// chain is the single schema authority, so every column the Go model reads or
// writes must exist in that chain.
//
// If this fails because a model field has no column, the fix is a new migration
// (and 000001/000002/000024 must not be edited) — not re-adding a runtime
// AutoMigrate, which is what made the two shapes drift in the first place.
func TestDataScopeModelMatchesMigrationChain(t *testing.T) {
	chain := normalizeSQL(readMigrationChain(t))
	ddl := normalizeSQL(dataScopePolicyDDL)

	modelType := reflect.TypeOf(SystemRoleDataScope{})
	fields := make(map[string]bool, modelType.NumField())
	for i := 0; i < modelType.NumField(); i++ {
		field := modelType.Field(i)
		if field.PkgPath != "" { // unexported
			continue
		}
		fields[field.Name] = true

		column, ok := dataScopeModelColumns[field.Name]
		if !ok {
			t.Fatalf("model field %s has no column mapping; add it to dataScopeModelColumns and to a migration", field.Name)
		}
		if !strings.Contains(chain, column) {
			t.Errorf("column %q (model field %s) does not appear in the migration chain %v", column, field.Name, migrationChainFiles)
		}
		if !strings.Contains(ddl, column) {
			t.Errorf("column %q (model field %s) does not appear in dataScopePolicyDDL", column, field.Name)
		}
	}

	for name := range dataScopeModelColumns {
		if !fields[name] {
			t.Errorf("dataScopeModelColumns declares %s but SystemRoleDataScope has no such field", name)
		}
	}
}

// TestDataScopeUniqueKeyIsTheIdentity pins the tenant-scoped identity of the
// policy table. The key is not cosmetic: with role_key alone as the key, two
// tenants could not own a role with the same key, and a tenant-wide policy read
// would merge every tenant's rows (acceptance F7/F10).
func TestDataScopeUniqueKeyIsTheIdentity(t *testing.T) {
	const uniqueKey = "uk_role_data_scope_tenant_role (tenant_id, role_key)"

	if !strings.Contains(normalizeSQL(dataScopePolicyDDL), uniqueKey) {
		t.Fatalf("dataScopePolicyDDL does not declare UNIQUE KEY %s", uniqueKey)
	}
	if !strings.Contains(normalizeSQL(readMigrationChain(t)), uniqueKey) {
		t.Fatalf("migration chain %v does not declare UNIQUE KEY %s", migrationChainFiles, uniqueKey)
	}
}

// TestMigrateDataScopePolicyDoesNotAutoMigrate is the ratchet for DB-01: the
// bootstrap guard may create the table, but it must not call AutoMigrate on it,
// or the Go model becomes a second schema authority again in dev mode.
func TestMigrateDataScopePolicyDoesNotAutoMigrate(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot resolve the path of data_scope_schema_test.go")
	}
	source, err := os.ReadFile(filepath.Join(filepath.Dir(thisFile), "data_scope_middleware.go"))
	if err != nil {
		t.Fatalf("read data_scope_middleware.go: %v", err)
	}

	text := string(source)
	start := strings.Index(text, "func MigrateDataScopePolicy(")
	if start < 0 {
		t.Fatal("MigrateDataScopePolicy not found in data_scope_middleware.go")
	}
	body := text[start:]
	if end := strings.Index(body[1:], "\nfunc "); end >= 0 {
		body = body[:end+1]
	}

	if strings.Contains(body, "AutoMigrate") {
		t.Fatalf("MigrateDataScopePolicy calls AutoMigrate again; the migration chain is the only schema authority (DB-01)")
	}
	if !strings.Contains(body, "dataScopePolicyDDL") {
		t.Fatalf("MigrateDataScopePolicy no longer uses dataScopePolicyDDL; the bootstrap shape is unverified")
	}
}
