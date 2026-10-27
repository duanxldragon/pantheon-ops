package session

import (
	"fmt"
	"testing"
	"time"

	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
	"github.com/duanxldragon/pantheon-base/backend/pkg/testmysql"
)

// Task 2026-09-22-export-and-session-pagination: ListAllSessions must run
// COUNT and LIMIT/OFFSET in the database; counts must match page results;
// UA-derived filters must be SQL-backed.

type stubPolicyForPagination struct{}

func (stubPolicyForPagination) GetSessionPolicy() AuthRuntimePolicy {
	return AuthRuntimePolicy{SessionIdleMinutes: 30, SessionRetentionDays: 90}
}

func newPaginationHarness(t *testing.T, sessions int) *Service {
	t.Helper()
	db := testmysql.Open(t)
	if err := db.AutoMigrate(&SystemUserSession{}); err != nil {
		t.Fatalf("migrate sessions: %v", err)
	}
	// ListAllSessions LEFT JOINs system_user for the username column; create
	// a minimal stand-in table (the session package does not own the user
	// model, so AutoMigrate would be a cross-module reach).
	if err := db.Exec("CREATE TABLE IF NOT EXISTS system_user (id BIGINT UNSIGNED PRIMARY KEY, username VARCHAR(64), nickname VARCHAR(64))").Error; err != nil {
		t.Fatalf("create system_user fixture: %v", err)
	}
	if err := db.Exec("INSERT INTO system_user (id, username, nickname) VALUES (42, 'alice', 'Alice')").Error; err != nil {
		t.Fatalf("seed system_user: %v", err)
	}
	now := time.Now()
	rows := make([]SystemUserSession, 0, sessions)
	for i := 0; i < sessions; i++ {
		rows = append(rows, SystemUserSession{
			SessionID:        fmt.Sprintf("sess-%04d", i),
			UserID:           42,
			RefreshJTI:       fmt.Sprintf("jti-%04d", i),
			RefreshExpiresAt: now.Add(time.Hour),
			LastActivityAt:   &now,
			LastIP:           fmt.Sprintf("10.0.0.%d", i%250),
			UserAgent:        "Mozilla/5.0 (Windows NT 10.0) Chrome/126.0 Safari/537.36",
			CreatedAt:        now.Add(-time.Duration(sessions-i) * time.Minute),
		})
	}
	if err := db.CreateInBatches(&rows, 100).Error; err != nil {
		t.Fatalf("seed sessions: %v", err)
	}
	return NewService(db, stubPolicyForPagination{}, nil, nil)
}

func TestListAllSessions_SQLPaginationConsistency(t *testing.T) {
	svc := newPaginationHarness(t, 57)

	page1, err := svc.ListAllSessions(&AdminSessionQuery{Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("page 1: %v", err)
	}
	if len(page1.Items) != 20 || page1.Total != 57 {
		t.Fatalf("expected 20 items / total 57, got %d / %d", len(page1.Items), page1.Total)
	}
	if page1.ActiveCount != 57 || page1.RevokedCount != 0 {
		t.Fatalf("expected active=57 revoked=0, got %d/%d", page1.ActiveCount, page1.RevokedCount)
	}
	if page1.Items[0].SessionID != "sess-0056" {
		t.Fatalf("expected newest session first, got %s", page1.Items[0].SessionID)
	}

	page3, err := svc.ListAllSessions(&AdminSessionQuery{Page: 3, PageSize: 20})
	if err != nil {
		t.Fatalf("page 3: %v", err)
	}
	if len(page3.Items) != 17 {
		t.Fatalf("expected 17 items on last page, got %d", len(page3.Items))
	}

	// Counts must match page results: filter to one IP and verify.
	pageIP, err := svc.ListAllSessions(&AdminSessionQuery{LastIP: "10.0.0.7", Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("ip filter: %v", err)
	}
	if pageIP.Total < 1 {
		t.Fatalf("expected at least one session for the filtered IP")
	}
	for _, item := range pageIP.Items {
		if item.LastIP != "10.0.0.7" {
			t.Fatalf("page leaked non-matching row: %s", item.LastIP)
		}
	}
}

func TestListAllSessions_RevokedCountsMatchPages(t *testing.T) {
	svc := newPaginationHarness(t, 30)
	now := time.Now()
	if err := svc.db.Model(&SystemUserSession{}).Where("session_id IN ?", []string{"sess-0000", "sess-0001", "sess-0002", "sess-0003", "sess-0004"}).
		Update("revoked_at", now).Error; err != nil {
		t.Fatalf("revoke seed rows: %v", err)
	}

	resp, err := svc.ListAllSessions(&AdminSessionQuery{Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if resp.Total != 30 || resp.ActiveCount != 25 || resp.RevokedCount != 5 {
		t.Fatalf("expected total=30 active=25 revoked=5, got %d/%d/%d", resp.Total, resp.ActiveCount, resp.RevokedCount)
	}

	revokedOnly, err := svc.ListAllSessions(&AdminSessionQuery{Status: sessionStatusPtr(common.SessionStatusRevoked), Page: 1, PageSize: 100})
	if err != nil {
		t.Fatalf("revoked filter: %v", err)
	}
	if revokedOnly.Total != 5 {
		t.Fatalf("expected 5 revoked rows via SQL filter, got %d", revokedOnly.Total)
	}
	for _, item := range revokedOnly.Items {
		if item.RevokedAt == nil {
			t.Fatalf("revoked filter leaked an active row: %s", item.SessionID)
		}
	}
}

func TestListAllSessions_ClientFiltersPushedToSQL(t *testing.T) {
	svc := newPaginationHarness(t, 10)

	// All rows share the Chrome UA; the SQL-backed browser filter must match
	// the same set the old in-memory matcher produced.
	resp, err := svc.ListAllSessions(&AdminSessionQuery{Browser: "Chrome", Page: 1, PageSize: 100})
	if err != nil {
		t.Fatalf("browser filter: %v", err)
	}
	if resp.Total != 10 {
		t.Fatalf("expected chrome filter to match all 10, got %d", resp.Total)
	}

	resp, err = svc.ListAllSessions(&AdminSessionQuery{Browser: "Firefox", Page: 1, PageSize: 100})
	if err != nil {
		t.Fatalf("firefox filter: %v", err)
	}
	if resp.Total != 0 {
		t.Fatalf("expected firefox filter to match none, got %d", resp.Total)
	}

	resp, err = svc.ListAllSessions(&AdminSessionQuery{OS: "Windows", Page: 1, PageSize: 100})
	if err != nil {
		t.Fatalf("os filter: %v", err)
	}
	if resp.Total != 10 {
		t.Fatalf("expected windows filter to match all 10, got %d", resp.Total)
	}
}

// BenchmarkListAllSessions_PageQuery exercises the SQL-paginated admin list
// against a 10k-row table; before the pagination task this scanned the whole
// table per request and paginated in memory.
func BenchmarkListAllSessions_PageQuery(b *testing.B) {
	db := testmysql.OpenTB(b)
	if err := db.AutoMigrate(&SystemUserSession{}); err != nil {
		b.Fatalf("migrate sessions: %v", err)
	}
	if err := db.Exec("CREATE TABLE IF NOT EXISTS system_user (id BIGINT UNSIGNED PRIMARY KEY, username VARCHAR(64), nickname VARCHAR(64))").Error; err != nil {
		b.Fatalf("create system_user fixture: %v", err)
	}
	now := time.Now()
	// Row i is (10000-i) seconds old: walk a timestamp forward instead of
	// converting the loop counter, which keeps the fixture free of uint64->int
	// conversions (gosec G115) without changing the seeded ordering.
	createdAt := now.Add(-10000 * time.Second)
	rows := make([]SystemUserSession, 0, 10000)
	for i := uint64(0); i < 10000; i++ {
		rows = append(rows, SystemUserSession{
			SessionID:        fmt.Sprintf("bench-%06d", i),
			UserID:           i%500 + 1,
			RefreshJTI:       fmt.Sprintf("jti-%06d", i),
			RefreshExpiresAt: now.Add(time.Hour),
			LastActivityAt:   &now,
			LastIP:           fmt.Sprintf("10.1.%d.%d", i/250%250, i%250),
			UserAgent:        "Mozilla/5.0 (Windows NT 10.0) Chrome/126.0 Safari/537.36",
			CreatedAt:        createdAt,
		})
		createdAt = createdAt.Add(time.Second)
	}
	if err := db.CreateInBatches(&rows, 500).Error; err != nil {
		b.Fatalf("seed sessions: %v", err)
	}
	svc := NewService(db, stubPolicyForPagination{}, nil, nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := svc.ListAllSessions(&AdminSessionQuery{Page: 2, PageSize: 20, Status: sessionStatusPtr(common.SessionStatusActive)}); err != nil {
			b.Fatalf("list: %v", err)
		}
	}
}

func sessionStatusPtr(v int) *int { return &v }
