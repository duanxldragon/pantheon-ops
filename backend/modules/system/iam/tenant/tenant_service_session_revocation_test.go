package tenant

import (
	"testing"

	"github.com/duanxldragon/pantheon-base/backend/pkg/tenant"
	"github.com/duanxldragon/pantheon-base/backend/pkg/testmysql"
)

// TestStatusChangeRequiresSessionRevocation pins the pure decision half of the
// FIX-3 wiring (no database): only a transition into a non-active lifecycle
// state (suspended / archived / deleted marker) must kill member sessions;
// reactivation and no-op writes must not.
func TestStatusChangeRequiresSessionRevocation(t *testing.T) {
	cases := []struct {
		name     string
		previous string
		next     string
		want     bool
	}{
		{"active to suspended revokes", tenant.TenantStatusActive, tenant.TenantStatusSuspended, true},
		{"active to archived revokes", tenant.TenantStatusActive, tenant.TenantStatusArchived, true},
		{"active to deleted marker revokes", tenant.TenantStatusActive, tenant.TenantStatusDeleted, true},
		{"suspended to active does not", tenant.TenantStatusSuspended, tenant.TenantStatusActive, false},
		{"no-op status write does not", tenant.TenantStatusSuspended, tenant.TenantStatusSuspended, false},
		{"empty next does not", tenant.TenantStatusActive, "", false},
		{"active to active does not", tenant.TenantStatusActive, tenant.TenantStatusActive, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := statusChangeRequiresSessionRevocation(tc.previous, tc.next); got != tc.want {
				t.Fatalf("statusChangeRequiresSessionRevocation(%q, %q) = %v, want %v",
					tc.previous, tc.next, got, tc.want)
			}
		})
	}
}

// TestRevokeTenantMemberSessions_KillsActiveMemberSessions drives the real
// wiring path against MySQL (skipped without one): only the tenant's ACTIVE
// members are revoked, their live session rows gain revoked_at, and members of
// other tenants keep their sessions.
func TestRevokeTenantMemberSessions_KillsActiveMemberSessions(t *testing.T) {
	db := testmysql.Open(t)
	if err := db.AutoMigrate(&Membership{}); err != nil {
		t.Fatalf("migrate memberships: %v", err)
	}
	if err := db.Exec("DELETE FROM tenant_memberships").Error; err != nil {
		t.Fatalf("clear memberships: %v", err)
	}
	if err := db.Exec("CREATE TABLE IF NOT EXISTS system_user_session (" +
		"id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY," +
		"session_id VARCHAR(64) NOT NULL," +
		"user_id BIGINT UNSIGNED NOT NULL," +
		"revoked_at DATETIME NULL)").Error; err != nil {
		t.Fatalf("create session table: %v", err)
	}
	if err := db.Exec("DELETE FROM system_user_session").Error; err != nil {
		t.Fatalf("clear sessions: %v", err)
	}

	seedMembership := func(tenantID, userID uint64, status string) {
		t.Helper()
		if err := db.Create(&Membership{TenantID: tenantID, UserID: userID, Role: "member", Status: status}).Error; err != nil {
			t.Fatalf("seed membership tenant=%d user=%d: %v", tenantID, userID, err)
		}
	}
	seedMembership(101, 42, "active")
	seedMembership(101, 43, "inactive")
	seedMembership(202, 44, "active")

	for _, row := range []struct {
		sid string
		uid uint64
	}{
		{"sess-42", 42}, {"sess-43", 43}, {"sess-44", 44},
	} {
		if err := db.Exec("INSERT INTO system_user_session (session_id, user_id) VALUES (?, ?)", row.sid, row.uid).Error; err != nil {
			t.Fatalf("seed session %s: %v", row.sid, err)
		}
	}

	NewService(db).revokeTenantMemberSessions(101)

	var live int64
	if err := db.Table("system_user_session").Where("user_id = ? AND revoked_at IS NULL", 42).Count(&live).Error; err != nil {
		t.Fatalf("count user 42 live sessions: %v", err)
	}
	if live != 0 {
		t.Fatalf("active member's session must be revoked, %d still live", live)
	}
	for _, uid := range []uint64{43, 44} {
		if err := db.Table("system_user_session").Where("user_id = ? AND revoked_at IS NULL", uid).Count(&live).Error; err != nil {
			t.Fatalf("count user %d live sessions: %v", uid, err)
		}
		if live != 1 {
			t.Fatalf("user %d must keep its session (inactive member / other tenant), got %d live", uid, live)
		}
	}
}
