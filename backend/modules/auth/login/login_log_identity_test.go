package login

import (
	"testing"

	"github.com/duanxldragon/pantheon-base/backend/pkg/tenant"
)

// F04: the own-login-log query must match the authenticated identity exactly.
// A substring match previously exposed login IPs, devices and timestamps of
// similar usernames (alice → alice_admin).
func TestListOwnLoginLogs_ExactIdentityMatch(t *testing.T) {
	db := setupTestDB(t)
	r := NewRuntime(db, testCredentialRepo(db))

	seeded := []SystemLogLogin{
		{Username: "alice", Ipaddr: "1.1.1.1", Browser: "Chrome", Os: "Windows", Status: 1, TenantID: 0},
		{Username: "alice_admin", Ipaddr: "9.9.9.9", Browser: "Firefox", Os: "Linux", Status: 1, TenantID: 0},
		{Username: "xalice", Ipaddr: "8.8.8.8", Browser: "Safari", Os: "macOS", Status: 1, TenantID: 0},
	}
	for i := range seeded {
		if err := db.Create(&seeded[i]).Error; err != nil {
			t.Fatalf("seed login log: %v", err)
		}
	}

	page, err := r.ListOwnLoginLogs("alice", &LoginLogQuery{})
	if err != nil {
		t.Fatalf("ListOwnLoginLogs: %v", err)
	}
	if page.Total != 1 {
		t.Fatalf("expected exactly 1 record for 'alice', got %d", page.Total)
	}
	for _, item := range page.Items {
		if item.Username != "alice" {
			t.Fatalf("identity leak: own-log returned record for %q", item.Username)
		}
		if item.Ipaddr != "1.1.1.1" {
			t.Fatalf("wrong record returned: ip %q", item.Ipaddr)
		}
	}
}

// F04: similar usernames must not be able to infer each other's identity —
// the empty and filtered admin variants of the admin list may keep LIKE on
// purpose, but the self-service entry point cannot.
func TestListOwnLoginLogs_DoesNotInheritAdminLikeFilter(t *testing.T) {
	db := setupTestDB(t)
	r := NewRuntime(db, testCredentialRepo(db))

	seeded := []SystemLogLogin{
		{Username: "bob", Ipaddr: "2.2.2.2", Status: 1, TenantID: 0},
		{Username: "bobby", Ipaddr: "3.3.3.3", Status: 1, TenantID: 0},
	}
	for i := range seeded {
		if err := db.Create(&seeded[i]).Error; err != nil {
			t.Fatalf("seed login log: %v", err)
		}
	}

	// Even if a client smuggles a username filter into the own-log query, the
	// exact subject filter wins: 'bobby' sees only their own rows.
	page, err := r.ListOwnLoginLogs("bobby", &LoginLogQuery{Username: "bob"})
	if err != nil {
		t.Fatalf("ListOwnLoginLogs: %v", err)
	}
	if page.Total != 1 || page.Items[0].Username != "bobby" || page.Items[0].Ipaddr != "3.3.3.3" {
		t.Fatalf("expected only bobby's record, got total=%d items=%+v", page.Total, page.Items)
	}
}

// F04: different tenants must not read each other's login IP/device/timestamps
// through the own-log endpoint.
func TestListOwnLoginLogs_TenantScoped(t *testing.T) {
	db := setupTestDB(t)

	seeded := []SystemLogLogin{
		{Username: "carol", Ipaddr: "4.4.4.4", Status: 1, TenantID: 101},
		{Username: "carol", Ipaddr: "5.5.5.5", Status: 1, TenantID: 202},
		{Username: "carol", Ipaddr: "6.6.6.6", Status: 1, TenantID: 0},
	}
	for i := range seeded {
		if err := db.Create(&seeded[i]).Error; err != nil {
			t.Fatalf("seed login log: %v", err)
		}
	}

	r101 := NewRuntime(db, testCredentialRepo(db)).WithTenantContext(&tenant.Context{TenantID: 101, Mode: tenant.ModeMulti, ResolvedBy: "subject"})
	page, err := r101.ListOwnLoginLogs("carol", &LoginLogQuery{})
	if err != nil {
		t.Fatalf("ListOwnLoginLogs: %v", err)
	}
	if page.Total != 1 || page.Items[0].Ipaddr != "4.4.4.4" {
		t.Fatalf("tenant scope leaked: total=%d items=%+v", page.Total, page.Items)
	}
}

// F04 acceptance 3: the admin login-log list keeps its intentional LIKE
// filtering (documented behavior, still tested).
func TestListLoginLogs_AdminKeywordFilterIntact(t *testing.T) {
	db := setupTestDB(t)
	r := NewRuntime(db, testCredentialRepo(db))

	seeded := []SystemLogLogin{
		{Username: "dave", Ipaddr: "7.7.7.7", Status: 1, TenantID: 0},
		{Username: "david", Ipaddr: "7.7.7.8", Status: 1, TenantID: 0},
		{Username: "erin", Ipaddr: "7.7.7.9", Status: 1, TenantID: 0},
	}
	for i := range seeded {
		if err := db.Create(&seeded[i]).Error; err != nil {
			t.Fatalf("seed login log: %v", err)
		}
	}

	filter := "dav"
	page, err := r.ListLoginLogs(&LoginLogQuery{Username: filter})
	if err != nil {
		t.Fatalf("ListLoginLogs: %v", err)
	}
	if page.Total != 2 {
		t.Fatalf("expected admin substring filter to match dave+david, got total=%d", page.Total)
	}
}
