package security

import (
	"testing"
	"time"

	"github.com/duanxldragon/pantheon-base/backend/pkg/testmysql"
)

func newSecurityEventFixture(t *testing.T) *Service {
	t.Helper()
	db := testmysql.Open(t)
	if err := db.AutoMigrate(&SystemAuthSecurityEvent{}); err != nil {
		t.Fatalf("migrate security event table: %v", err)
	}
	return &Service{db: db}
}

func seedSecurityEvent(t *testing.T, svc *Service, event SystemAuthSecurityEvent) uint64 {
	t.Helper()
	if event.Severity == "" {
		event.Severity = "medium"
	}
	if event.EventType == "" {
		event.EventType = "password_wrong"
	}
	if event.MessageKey == "" {
		event.MessageKey = "auth.security.event.password_wrong"
	}
	if err := svc.db.Create(&event).Error; err != nil {
		t.Fatalf("seed security event: %v", err)
	}
	return event.ID
}

func TestBatchAcknowledgeSecurityEvents(t *testing.T) {
	svc := newSecurityEventFixture(t)
	ackedAt := time.Now().Add(-time.Hour)
	pending1 := seedSecurityEvent(t, svc, SystemAuthSecurityEvent{Username: "u1"})
	pending2 := seedSecurityEvent(t, svc, SystemAuthSecurityEvent{Username: "u2"})
	alreadyAcked := seedSecurityEvent(t, svc, SystemAuthSecurityEvent{
		Username:            "u3",
		AcknowledgedAt:      &ackedAt,
		AcknowledgedByUser:  "earlier-admin",
		AcknowledgementNote: "original note",
	})

	count, err := svc.BatchAcknowledgeSecurityEvents(
		[]uint64{pending1, pending2, alreadyAcked, pending1, 0},
		42, "admin", "batch note",
	)
	if err != nil {
		t.Fatalf("batch acknowledge: %v", err)
	}
	if count != 2 {
		t.Fatalf("expected 2 acknowledged (already-acked and dup/zero skipped), got %d", count)
	}

	var kept SystemAuthSecurityEvent
	if err := svc.db.First(&kept, alreadyAcked).Error; err != nil {
		t.Fatalf("reload already-acked event: %v", err)
	}
	if kept.AcknowledgementNote != "original note" || kept.AcknowledgedByUser != "earlier-admin" {
		t.Fatalf("already-acknowledged event must keep its original note, got %+v", kept)
	}

	var pendingLeft int64
	if err := svc.db.Model(&SystemAuthSecurityEvent{}).
		Where("acknowledged_at IS NULL").Count(&pendingLeft).Error; err != nil {
		t.Fatalf("count pending: %v", err)
	}
	if pendingLeft != 0 {
		t.Fatalf("expected no pending events left, got %d", pendingLeft)
	}

	if _, err := svc.BatchAcknowledgeSecurityEvents([]uint64{pending1}, 42, "admin", "  "); err == nil {
		t.Fatal("expected error for empty note")
	}
	if _, err := svc.BatchAcknowledgeSecurityEvents(nil, 42, "admin", "note"); err == nil {
		t.Fatal("expected error for empty id list")
	}
}

func TestListSecurityEventsAggregates(t *testing.T) {
	svc := newSecurityEventFixture(t)
	ackedAt := time.Now().Add(-time.Hour)
	seedSecurityEvent(t, svc, SystemAuthSecurityEvent{Username: "p1", Severity: "high"})
	seedSecurityEvent(t, svc, SystemAuthSecurityEvent{Username: "p2"})
	seedSecurityEvent(t, svc, SystemAuthSecurityEvent{Username: "a1", AcknowledgedAt: &ackedAt})
	seedSecurityEvent(t, svc, SystemAuthSecurityEvent{
		Username: "a2", Severity: "high", AcknowledgedAt: &ackedAt,
	})

	resp, err := svc.ListSecurityEvents(&SecurityEventQuery{Page: 1, PageSize: 2})
	if err != nil {
		t.Fatalf("list security events: %v", err)
	}
	if resp.Total != 4 {
		t.Fatalf("expected total 4, got %d", resp.Total)
	}
	if len(resp.Items) != 2 {
		t.Fatalf("expected page of 2 items, got %d", len(resp.Items))
	}
	// Aggregates must cover the whole filtered set, not the current page.
	if resp.PendingCount != 2 || resp.AcknowledgedCount != 2 || resp.HighSeverityCount != 2 {
		t.Fatalf("expected pending=2 acknowledged=2 high=2, got %d/%d/%d",
			resp.PendingCount, resp.AcknowledgedCount, resp.HighSeverityCount)
	}

	severity := "high"
	filtered, err := svc.ListSecurityEvents(&SecurityEventQuery{Severity: severity, Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("list filtered security events: %v", err)
	}
	if filtered.Total != 2 || filtered.PendingCount != 1 || filtered.AcknowledgedCount != 1 {
		t.Fatalf("filtered aggregates should follow the filter scope, got total=%d pending=%d acked=%d",
			filtered.Total, filtered.PendingCount, filtered.AcknowledgedCount)
	}
}

// TestRunSecurityEventRetention verifies the maintenance entry point for the
// security-event retention sweep. Throttling/overlap is the maintenance
// registry's responsibility and is covered in pkg/maintenance, so this only
// pins the sweep semantics. It also replaces the old read-path assertion: the
// list handler no longer runs retention at all (task
// 2026-09-22-request-path-maintenance).
func TestRunSecurityEventRetention(t *testing.T) {
	svc := newSecurityEventFixture(t)
	old := time.Now().AddDate(0, 0, -200)
	recent := time.Now().Add(-time.Hour)

	oldAcked := seedSecurityEvent(t, svc, SystemAuthSecurityEvent{
		Username: "old-acked", AcknowledgedAt: &old, CreatedAt: old,
	})
	oldPending := seedSecurityEvent(t, svc, SystemAuthSecurityEvent{
		Username: "old-pending", CreatedAt: old,
	})
	recentAcked := seedSecurityEvent(t, svc, SystemAuthSecurityEvent{
		Username: "recent-acked", AcknowledgedAt: &recent, CreatedAt: recent,
	})

	// List is a read path: it must leave every row in place.
	if _, err := svc.ListSecurityEvents(&SecurityEventQuery{Page: 1, PageSize: 50}); err != nil {
		t.Fatalf("list security events: %v", err)
	}
	if count := countSecurityEvents(t, svc); count != 3 {
		t.Fatalf("list request must not purge: expected 3 events, got %d", count)
	}

	// No system_setting table in this fixture: retention falls back to 180 days.
	if err := svc.RunSecurityEventRetention(); err != nil {
		t.Fatalf("run security event retention: %v", err)
	}

	var remaining []uint64
	if err := svc.db.Model(&SystemAuthSecurityEvent{}).Pluck("id", &remaining).Error; err != nil {
		t.Fatalf("collect remaining ids: %v", err)
	}
	seen := map[uint64]bool{}
	for _, id := range remaining {
		seen[id] = true
	}
	if seen[oldAcked] {
		t.Fatal("acknowledged event past retention must be deleted")
	}
	if !seen[oldPending] {
		t.Fatal("pending events must never be auto-swept, regardless of age")
	}
	if !seen[recentAcked] {
		t.Fatal("acknowledged event within retention must be kept")
	}
}

func countSecurityEvents(t *testing.T, svc *Service) int64 {
	t.Helper()
	var count int64
	if err := svc.db.Model(&SystemAuthSecurityEvent{}).Count(&count).Error; err != nil {
		t.Fatalf("count security events: %v", err)
	}
	return count
}
