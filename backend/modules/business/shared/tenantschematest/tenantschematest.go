// Package tenantschematest asserts the tenant ownership contract of GORM
// models without a database: fresh installs build tables from model tags while
// upgrades run the SQL migration, so the two must describe the same schema.
package tenantschematest

import (
	"reflect"
	"sync"
	"testing"

	"gorm.io/gorm/schema"
)

func parse(t *testing.T, model any) *schema.Schema {
	t.Helper()
	parsed, err := schema.Parse(model, &sync.Map{}, schema.NamingStrategy{})
	if err != nil {
		t.Fatalf("parse %T: %v", model, err)
	}
	return parsed
}

// RequireTenantOwned fails unless every model has a NOT NULL tenant_id column
// defaulting to 0 (compat / platform-owned rows keep working unchanged).
func RequireTenantOwned(t *testing.T, models ...any) {
	t.Helper()
	for _, model := range models {
		field := parse(t, model).LookUpField("tenant_id")
		if field == nil {
			t.Errorf("%T has no tenant_id column", model)
			continue
		}
		if !field.NotNull || field.DefaultValue != "0" {
			t.Errorf("%T.tenant_id must be NOT NULL DEFAULT 0, got notNull=%v default=%q", model, field.NotNull, field.DefaultValue)
		}
	}
}

// RequireTenantFirstUnique fails unless the named index is UNIQUE and covers
// exactly the given columns in order, starting with tenant_id.
func RequireTenantFirstUnique(t *testing.T, model any, name string, columns ...string) {
	t.Helper()
	var index *schema.Index
	for _, candidate := range parse(t, model).ParseIndexes() {
		if candidate.Name == name {
			index = candidate
		}
	}
	if index == nil {
		t.Fatalf("%T has no index %s", model, name)
	}
	got := make([]string, 0, len(index.Fields))
	for _, option := range index.Fields {
		got = append(got, option.DBName)
	}
	if index.Class != "UNIQUE" || !reflect.DeepEqual(got, columns) || len(columns) == 0 || columns[0] != "tenant_id" {
		t.Fatalf("%T.%s = %s%v, want UNIQUE tenant-first %v", model, name, index.Class, got, columns)
	}
}
