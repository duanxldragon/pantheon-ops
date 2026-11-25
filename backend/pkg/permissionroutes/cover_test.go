package permissionroutes

import "testing"

func TestCoversKeepsStaticAndParamSiblingsApart(t *testing.T) {
	cases := []struct {
		name   string
		route  string
		policy string
		want   bool
	}{
		{"detail policy covers detail", "/api/v1/business/cmdb/hosts/:id", "/api/v1/business/cmdb/hosts/:id", true},
		{"detail policy must not cover export", "/api/v1/business/cmdb/hosts/export", "/api/v1/business/cmdb/hosts/:id", false},
		{"export policy must not cover detail", "/api/v1/business/cmdb/hosts/:id", "/api/v1/business/cmdb/hosts/export", false},
		{"static policy covers its own route", "/api/v1/business/cmdb/hosts/export", "/api/v1/business/cmdb/hosts/export", true},
		{"param names are interchangeable", "/api/v1/business/cmdb/hosts/:id", "/api/v1/business/cmdb/hosts/:hostID", true},
		{"explicit wildcard keeps granting", "/api/v1/business/cmdb/hosts/export", "/api/v1/*", true},
		{"wildcard covers param route too", "/api/v1/business/cmdb/hosts/:id", "/api/v1/*", true},
		{"wildcard also matches the bare prefix", "/api/v1", "/api/v1/*", true},
		{"longer policy cannot cover shorter route", "/api/v1/business", "/api/v1/business/cmdb/hosts/:id", false},
		{"empty sides never match", "", "/api/v1/*", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := Covers(tc.route, tc.policy); got != tc.want {
				t.Fatalf("Covers(%q, %q) = %v, want %v", tc.route, tc.policy, got, tc.want)
			}
		})
	}
}
