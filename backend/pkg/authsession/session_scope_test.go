package authsession

import (
	"testing"
	"time"
)

func TestColumnPrefix_Empty(t *testing.T) {
	if got := columnPrefix(""); got != "" {
		t.Fatalf("expected empty string, got %q", got)
	}
}

func TestColumnPrefix_Whitespace(t *testing.T) {
	if got := columnPrefix("  "); got != "" {
		t.Fatalf("expected empty string, got %q", got)
	}
}

func TestColumnPrefix_WithAlias(t *testing.T) {
	if got := columnPrefix("s"); got != "s." {
		t.Fatalf("expected \"s.\", got %q", got)
	}
}

func TestColumnPrefix_AlreadyDot(t *testing.T) {
	if got := columnPrefix("s."); got != "s." {
		t.Fatalf("expected \"s.\", got %q", got)
	}
}

func TestColumnPrefix_MultiCharAlias(t *testing.T) {
	if got := columnPrefix("session"); got != "session." {
		t.Fatalf("expected \"session.\", got %q", got)
	}
}

func TestLastSeenExpr_EmptyPrefix(t *testing.T) {
	expr := lastSeenExpr("")
	expected := "COALESCE(last_activity_at, last_refresh_at, created_at)"
	if expr != expected {
		t.Fatalf("expected %q, got %q", expected, expr)
	}
}

func TestLastSeenExpr_WithPrefix(t *testing.T) {
	expr := lastSeenExpr("s.")
	expected := "COALESCE(s.last_activity_at, s.last_refresh_at, s.created_at)"
	if expr != expected {
		t.Fatalf("expected %q, got %q", expected, expr)
	}
}

func TestLastSeenExpr_DotlessPrefix(t *testing.T) {
	// lastSeenExpr expects a caller that already applies columnPrefix;
	// verify it correctly builds with "t." which is what columnPrefix("t") produces
	expr := lastSeenExpr("t.")
	expected := "COALESCE(t.last_activity_at, t.last_refresh_at, t.created_at)"
	if expr != expected {
		t.Fatalf("expected %q, got %q", expected, expr)
	}
}

func TestCleanupInactiveSessions_NilDB(t *testing.T) {
	err := CleanupInactiveSessions(nil, time.Now(), 30)
	if err != nil {
		t.Fatalf("expected nil error for nil DB, got %v", err)
	}
}

func TestCleanupInactiveSessions_ZeroIdle(t *testing.T) {
	err := CleanupInactiveSessions(nil, time.Now(), 0)
	if err != nil {
		t.Fatalf("expected nil error for nil DB, got %v", err)
	}
}

func TestCleanupUserOverflowSessions_NilDB(t *testing.T) {
	err := CleanupUserOverflowSessions(nil, 1, time.Now(), 30, 5)
	if err != nil {
		t.Fatalf("expected nil error for nil DB, got %v", err)
	}
}

func TestCleanupUserOverflowSessions_ZeroUserID(t *testing.T) {
	err := CleanupUserOverflowSessions(nil, 0, time.Now(), 30, 5)
	if err != nil {
		t.Fatalf("expected nil error for zero userID, got %v", err)
	}
}

func TestCleanupUserOverflowSessions_ZeroMaxActive(t *testing.T) {
	err := CleanupUserOverflowSessions(nil, 1, time.Now(), 30, 0)
	if err != nil {
		t.Fatalf("expected nil error for zero maxActive, got %v", err)
	}
}

func TestCleanupUserOverflowSessions_NegativeMaxActive(t *testing.T) {
	err := CleanupUserOverflowSessions(nil, 1, time.Now(), 30, -1)
	if err != nil {
		t.Fatalf("expected nil error for negative maxActive, got %v", err)
	}
}

func TestPurgeHistoricSessions_NilDB(t *testing.T) {
	err := PurgeHistoricSessions(nil, time.Now(), 90)
	if err != nil {
		t.Fatalf("expected nil error for nil DB, got %v", err)
	}
}

func TestPurgeHistoricSessions_ZeroRetention(t *testing.T) {
	err := PurgeHistoricSessions(nil, time.Now(), 0)
	if err != nil {
		t.Fatalf("expected nil error for zero retention, got %v", err)
	}
}

func TestPurgeHistoricSessions_NegativeRetention(t *testing.T) {
	err := PurgeHistoricSessions(nil, time.Now(), -1)
	if err != nil {
		t.Fatalf("expected nil error for negative retention, got %v", err)
	}
}

func TestLoadSessionIdleMinutes_NilDB(t *testing.T) {
	got := LoadSessionIdleMinutes(nil, DefaultSessionIdleMinutes)
	if got != DefaultSessionIdleMinutes {
		t.Fatalf("expected %d for nil DB, got %d", DefaultSessionIdleMinutes, got)
	}
}

func TestDefaultConstants(t *testing.T) {
	if DefaultSessionIdleMinutes != 30 {
		t.Fatalf("expected DefaultSessionIdleMinutes=30, got %d", DefaultSessionIdleMinutes)
	}
	if DefaultMaxActiveSessionsPerUser != 1 {
		t.Fatalf("expected DefaultMaxActiveSessionsPerUser=1, got %d", DefaultMaxActiveSessionsPerUser)
	}
	if DefaultSessionRetentionDays != 90 {
		t.Fatalf("expected DefaultSessionRetentionDays=90, got %d", DefaultSessionRetentionDays)
	}
}
