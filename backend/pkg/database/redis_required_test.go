package database

import "testing"

// TestRequireRedis_Semantics locks the startup fail-fast contract (task
// 2026-09-22-production-redis-and-security-gates): Redis is mandatory in
// production and for any deployment that sets PANTHEON_REDIS_REQUIRED=true;
// non-production stays compatible (degrade to nil) without the explicit flag.
func TestRequireRedis_Semantics(t *testing.T) {
	cases := []struct {
		name    string
		env     string
		require string
		want    bool
	}{
		{"production always requires", "production", "", true},
		{"dev without flag is optional", "development", "", false},
		{"empty env without flag is optional", "", "", false},
		{"explicit required overrides dev", "development", "true", true},
		{"explicit required is case-insensitive", "development", "TRUE", true},
		{"explicit false in production still requires", "production", "false", true},
		{"arbitrary value is not required", "development", "yes", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("PANTHEON_ENV", tc.env)
			t.Setenv("PANTHEON_REDIS_REQUIRED", tc.require)
			if got := RequireRedis(); got != tc.want {
				t.Fatalf("RequireRedis() = %v, want %v", got, tc.want)
			}
		})
	}
}
