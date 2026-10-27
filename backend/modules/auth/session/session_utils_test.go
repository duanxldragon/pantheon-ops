package session

import (
	"testing"
	"time"
)

func TestTruncateString(t *testing.T) {
	if got := TruncateString("hello", 10); got != "hello" {
		t.Errorf("TruncateString should keep short strings, got %q", got)
	}
	if got := TruncateString("hello", 3); got != "hel" {
		t.Errorf("TruncateString should truncate long strings, got %q", got)
	}
	if got := TruncateString("", 5); got != "" {
		t.Errorf("TruncateString empty input, got %q", got)
	}
}

func TestNormalizeSessionClientIP(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  string
	}{
		{"empty", "", ""},
		{"whitespace only", "   ", ""},
		{"invalid text", "not-an-ip", ""},
		{"valid ipv4", " 192.168.1.1 ", "192.168.1.1"},
		{"ipv6 loopback", "::1", "::1"},
		{"ipv4 with port-like suffix", "192.168.1.1:8080", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := NormalizeSessionClientIP(tc.input); got != tc.want {
				t.Errorf("NormalizeSessionClientIP(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestNormalizeSessionUserAgent(t *testing.T) {
	if got := NormalizeSessionUserAgent("  "); got != "" {
		t.Errorf("whitespace-only UA should normalize to empty, got %q", got)
	}
	if got := NormalizeSessionUserAgent(" Mozilla/5.0 "); got != "Mozilla/5.0" {
		t.Errorf("UA should be trimmed, got %q", got)
	}
	// Non-printable control characters must be stripped.
	dirty := "Mozilla/5.0\x00<script>"
	cleaned := NormalizeSessionUserAgent(dirty)
	for _, r := range cleaned {
		if r == '\x00' {
			t.Errorf("control character survived sanitization: %q", cleaned)
		}
	}
	// Over-long UA must be truncated to 255 characters.
	long := make([]byte, 400)
	for i := range long {
		long[i] = 'a'
	}
	if got := NormalizeSessionUserAgent(string(long)); len(got) != 255 {
		t.Errorf("UA should be truncated to 255 bytes, got %d", len(got))
	}
}

func TestFormatNullableTime(t *testing.T) {
	if got := FormatNullableTime(nil); got != nil {
		t.Errorf("nil time should format to nil, got %v", got)
	}
	ts := time.Date(2026, 9, 8, 10, 0, 0, 0, time.UTC)
	got := FormatNullableTime(&ts)
	if got == nil || *got != "2026-09-08T10:00:00Z" {
		t.Errorf("FormatNullableTime = %v, want 2026-09-08T10:00:00Z", got)
	}
}

func TestBuildSessionRespMarksCurrentSession(t *testing.T) {
	now := time.Date(2026, 9, 8, 10, 0, 0, 0, time.UTC)
	item := SystemUserSession{
		SessionID:        "sess-1",
		UserID:           42,
		RefreshExpiresAt: now,
		LastIP:           "10.0.0.1",
		UserAgent:        "Mozilla/5.0",
		CreatedAt:        now,
	}
	resp := BuildSessionResp(item, "sess-1")
	if !resp.IsCurrent {
		t.Error("BuildSessionResp should mark the matching session as current")
	}
	if resp.SessionID != "sess-1" || resp.LastIP != "10.0.0.1" {
		t.Errorf("unexpected resp fields: %+v", resp)
	}

	other := BuildSessionResp(item, "sess-other")
	if other.IsCurrent {
		t.Error("BuildSessionResp should not mark a different session as current")
	}
}

func TestSortSessionsCurrentFirstThenNewest(t *testing.T) {
	items := []SessionResp{
		{SessionID: "old", CreatedAt: "2026-09-01T00:00:00Z"},
		{SessionID: "current", IsCurrent: true, CreatedAt: "2026-09-02T00:00:00Z"},
		{SessionID: "new", CreatedAt: "2026-09-08T00:00:00Z"},
	}
	SortSessions(items, "current")
	if items[0].SessionID != "current" {
		t.Errorf("current session should sort first, got %q first", items[0].SessionID)
	}
	if items[1].SessionID != "new" || items[2].SessionID != "old" {
		t.Errorf("non-current sessions should sort newest first, got %q, %q", items[1].SessionID, items[2].SessionID)
	}
}

func TestParseCleanupWindow(t *testing.T) {
	const invalidErr = "invalid cleanup window"

	t.Run("both empty returns nil window", func(t *testing.T) {
		window, err := parseCleanupWindow("", "", invalidErr)
		if window != nil || err != nil {
			t.Errorf("expected nil window and nil error, got %v, %v", window, err)
		}
	})

	t.Run("only one bound is invalid", func(t *testing.T) {
		if _, err := parseCleanupWindow("2026-09-08T00:00:00Z", "", invalidErr); err == nil {
			t.Error("startedAt without endedAt should be an error")
		}
		if _, err := parseCleanupWindow("", "2026-09-08T00:00:00Z", invalidErr); err == nil {
			t.Error("endedAt without startedAt should be an error")
		}
	})

	t.Run("malformed timestamps are invalid", func(t *testing.T) {
		if _, err := parseCleanupWindow("yesterday", "tomorrow", invalidErr); err == nil {
			t.Error("malformed timestamps should be an error")
		}
	})

	t.Run("end before start is invalid", func(t *testing.T) {
		if _, err := parseCleanupWindow("2026-09-08T10:00:00Z", "2026-09-08T09:00:00Z", invalidErr); err == nil {
			t.Error("end before start should be an error")
		}
	})

	t.Run("valid window parses", func(t *testing.T) {
		window, err := parseCleanupWindow("2026-09-08T09:00:00Z", "2026-09-08T10:00:00Z", invalidErr)
		if err != nil || window == nil {
			t.Fatalf("expected valid window, got %v, %v", window, err)
		}
		if !window.StartedAt.Equal(time.Date(2026, 9, 8, 9, 0, 0, 0, time.UTC)) {
			t.Errorf("unexpected StartedAt: %v", window.StartedAt)
		}
	})
}

func TestIsAllowedSessionCleanupRetentionDays(t *testing.T) {
	if !isAllowedSessionCleanupRetentionDays(7, nil) {
		t.Error("7 days should be allowed by default policy")
	}
	if isAllowedSessionCleanupRetentionDays(5, nil) {
		t.Error("5 days should not be allowed by default policy")
	}
	if !isAllowedSessionCleanupRetentionDays(14, []int{14}) {
		t.Error("custom policy should allow its own values")
	}
}

func TestNormalizePageQuery(t *testing.T) {
	page, size := normalizePageQuery(0, 0)
	if page != 1 || size != 10 {
		t.Errorf("zero page/size should normalize to 1/10, got %d/%d", page, size)
	}
	page, size = normalizePageQuery(3, 500)
	if page != 3 || size != 100 {
		t.Errorf("oversized page size should clamp to 100, got %d/%d", page, size)
	}
}

func TestNormalizeSessionIDs(t *testing.T) {
	if got := normalizeSessionIDs(nil); got != nil {
		t.Errorf("nil input should return nil, got %v", got)
	}
	got := normalizeSessionIDs([]string{" a ", "", "a", "b", " "})
	if len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Errorf("expected deduplicated trimmed ids [a b], got %v", got)
	}
}
