package tenant

import "testing"

// TestQualifyTenantColumn pins the tenant-scope column resolution. A bare
// `tenant_id` breaks any query that joins another tenant-scoped table
// (MySQL error 1052), so an explicit table — with or without an alias — must
// be qualified, while model-based / non-identifier tables keep the legacy
// single-table behavior.
func TestQualifyTenantColumn(t *testing.T) {
	cases := []struct {
		name  string
		table string
		want  string
	}{
		{"model based (no table resolved)", "", "tenant_id"},
		{"plain table", "system_user", "system_user.tenant_id"},
		{"table with AS alias", "system_user AS u", "u.tenant_id"},
		{"table with bare alias", "system_user u", "u.tenant_id"},
		{"surrounding whitespace", "  system_user  ", "system_user.tenant_id"},
		{"subquery expression", "(SELECT id FROM t)", "tenant_id"},
		{"comma list", "system_user, system_post", "tenant_id"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := qualifyTenantColumn(tc.table); got != tc.want {
				t.Fatalf("qualifyTenantColumn(%q) = %q, want %q", tc.table, got, tc.want)
			}
		})
	}
}
