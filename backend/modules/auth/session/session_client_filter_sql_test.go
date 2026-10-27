package session

import (
	"fmt"
	"strings"
	"testing"

	"github.com/duanxldragon/pantheon-base/backend/pkg/testmysql"

	"gorm.io/gorm"
)

// TestAdminSessionDeviceFilterRendersDetectDevicePredicates pins the SQL the
// admin session list pushes into the database for each device filter.
//
// The clauses are the SQL-side twin of DetectDevice in session_user_agent.go:
// "Android Phone" requires android+mobile, "Android Tablet" requires android
// without mobile, "Desktop" requires no mobile token. A refactor of the clause
// constants must not change what these filters select, and the rendered SQL is
// the only place that contract is observable.
func TestAdminSessionDeviceFilterRendersDetectDevicePredicates(t *testing.T) {
	db := testmysql.Open(t)
	dry := db.Session(&gorm.Session{DryRun: true})

	cases := []struct {
		device        string
		wantPredicate string
		wantArgs      []string
	}{
		{
			device:        "mobile",
			wantPredicate: "LOWER(system_user_session.user_agent) LIKE ?",
			wantArgs:      []string{"%mobile%"},
		},
		{
			device:        "desktop",
			wantPredicate: "LOWER(system_user_session.user_agent) NOT LIKE ?",
			wantArgs:      []string{"%mobile%"},
		},
		{
			device:        "android phone",
			wantPredicate: "LOWER(system_user_session.user_agent) LIKE ? AND LOWER(system_user_session.user_agent) LIKE ?",
			wantArgs:      []string{"%android%", "%mobile%"},
		},
		{
			device:        "android tablet",
			wantPredicate: "LOWER(system_user_session.user_agent) LIKE ? AND LOWER(system_user_session.user_agent) NOT LIKE ?",
			wantArgs:      []string{"%android%", "%mobile%"},
		},
		{
			device:        "ipad",
			wantPredicate: "LOWER(system_user_session.user_agent) LIKE ?",
			wantArgs:      []string{"%ipad%"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.device, func(t *testing.T) {
			var rows []SystemUserSession
			stmt := applyAdminSessionDeviceFilter(dry, tc.device).Find(&rows).Statement

			sql := stmt.SQL.String()
			if !strings.Contains(sql, tc.wantPredicate) {
				t.Fatalf("device %q rendered %q, want predicate %q", tc.device, sql, tc.wantPredicate)
			}
			if len(stmt.Vars) != len(tc.wantArgs) {
				t.Fatalf("device %q bound %v, want %v", tc.device, stmt.Vars, tc.wantArgs)
			}
			for i, want := range tc.wantArgs {
				if got := fmt.Sprint(stmt.Vars[i]); got != want {
					t.Fatalf("device %q arg %d = %q, want %q", tc.device, i, got, want)
				}
			}
		})
	}
}
