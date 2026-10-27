package login

import (
	"testing"
	"time"
)

const errInvalidTestSentinel = "err.invalid"

// ---- loginSourceIP ----

func TestLoginSourceIP_ExtractsIPFromSourceKey(t *testing.T) {
	if got := loginSourceIP("ip:10.0.0.1"); got != "10.0.0.1" {
		t.Fatalf("expected 10.0.0.1, got %q", got)
	}
}

func TestLoginSourceIP_TrimsWhitespaceAroundIP(t *testing.T) {
	if got := loginSourceIP("ip: 10.0.0.1 "); got != "10.0.0.1" {
		t.Fatalf("expected trimmed ip, got %q", got)
	}
}

func TestLoginSourceIP_NonIPSourceKeyReturnsEmpty(t *testing.T) {
	if got := loginSourceIP("sso:oidc-provider"); got != "" {
		t.Fatalf("expected empty ip for non-ip source key, got %q", got)
	}
	if got := loginSourceIP(""); got != "" {
		t.Fatalf("expected empty ip for empty source key, got %q", got)
	}
}

// ---- sourceThrottleBlocked ----

func TestSourceThrottleBlocked_NilBlockedUntilIsNeverBlocked(t *testing.T) {
	if sourceThrottleBlocked(nil, time.Now()) {
		t.Fatal("expected nil blocked_until to never be blocked")
	}
}

func TestSourceThrottleBlocked_FutureBlockedUntilBlocks(t *testing.T) {
	blockedUntil := time.Now().Add(10 * time.Minute)
	if !sourceThrottleBlocked(&blockedUntil, time.Now()) {
		t.Fatal("expected future blocked_until to block")
	}
}

func TestSourceThrottleBlocked_PastBlockedUntilDoesNotBlock(t *testing.T) {
	blockedUntil := time.Now().Add(-time.Minute)
	if sourceThrottleBlocked(&blockedUntil, time.Now()) {
		t.Fatal("expected expired blocked_until to not block")
	}
}

// ---- sourceThrottleBlockedUntil ----

func TestSourceThrottleBlockedUntil_DisabledWhenShouldBlockFalse(t *testing.T) {
	policy := RuntimePolicy{SourceLockMinutes: 10}
	if got := sourceThrottleBlockedUntil(policy, time.Now(), false); got != nil {
		t.Fatalf("expected nil blocked_until when shouldBlock=false, got %v", got)
	}
}

func TestSourceThrottleBlockedUntil_UsesConfiguredLockMinutes(t *testing.T) {
	policy := RuntimePolicy{SourceLockMinutes: 10}
	now := time.Now()
	blockedUntil := sourceThrottleBlockedUntil(policy, now, true)
	if blockedUntil == nil {
		t.Fatal("expected blocked_until to be set")
	}
	if diff := blockedUntil.Sub(now); diff < 9*time.Minute || diff > 11*time.Minute {
		t.Fatalf("expected ~10 minute lock, got %v", diff)
	}
}

func TestSourceThrottleBlockedUntil_FallsBackToOneMinute(t *testing.T) {
	policy := RuntimePolicy{SourceLockMinutes: 0}
	now := time.Now()
	blockedUntil := sourceThrottleBlockedUntil(policy, now, true)
	if blockedUntil == nil {
		t.Fatal("expected blocked_until to be set")
	}
	if diff := blockedUntil.Sub(now); diff < 30*time.Second || diff > 90*time.Second {
		t.Fatalf("expected ~1 minute fallback lock, got %v", diff)
	}
}

// ---- normalizePageQuery ----

func TestNormalizePageQuery_ZeroValuesGetDefaults(t *testing.T) {
	page, pageSize := normalizePageQuery(0, 0)
	if page != 1 || pageSize != 10 {
		t.Fatalf("expected (1, 10), got (%d, %d)", page, pageSize)
	}
}

func TestNormalizePageQuery_NegativeValuesGetDefaults(t *testing.T) {
	page, pageSize := normalizePageQuery(-5, -1)
	if page != 1 || pageSize != 10 {
		t.Fatalf("expected (1, 10), got (%d, %d)", page, pageSize)
	}
}

func TestNormalizePageQuery_ValidValuesPassThrough(t *testing.T) {
	page, pageSize := normalizePageQuery(3, 25)
	if page != 3 || pageSize != 25 {
		t.Fatalf("expected (3, 25), got (%d, %d)", page, pageSize)
	}
}

func TestNormalizePageQuery_CapsPageSizeAt100(t *testing.T) {
	_, pageSize := normalizePageQuery(1, 500)
	if pageSize != 100 {
		t.Fatalf("expected pageSize capped at 100, got %d", pageSize)
	}
}

// ---- queryPage / queryPageSize nil handling ----

func TestQueryPageAndPageSize_NilQueryUsesDefaults(t *testing.T) {
	if got := queryPage(nil); got != 1 {
		t.Fatalf("expected page 1 for nil query, got %d", got)
	}
	if got := queryPageSize(nil); got != 10 {
		t.Fatalf("expected pageSize 10 for nil query, got %d", got)
	}
}

func TestQueryPageAndPageSize_ReadsQueryValues(t *testing.T) {
	query := &LoginLogQuery{Page: 2, PageSize: 50}
	if got := queryPage(query); got != 2 {
		t.Fatalf("expected page 2, got %d", got)
	}
	if got := queryPageSize(query); got != 50 {
		t.Fatalf("expected pageSize 50, got %d", got)
	}
}

// ---- parseCleanupWindow ----

func TestParseCleanupWindow_BothEmptyReturnsNilWindow(t *testing.T) {
	window, err := parseCleanupWindow("", "", "err.invalid")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if window != nil {
		t.Fatalf("expected nil window, got %+v", window)
	}
}

func TestParseCleanupWindow_WhitespaceOnlyCountsAsEmpty(t *testing.T) {
	window, err := parseCleanupWindow("  ", "  ", "err.invalid")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if window != nil {
		t.Fatalf("expected nil window, got %+v", window)
	}
}

func TestParseCleanupWindow_OneBoundOnlyIsInvalid(t *testing.T) {
	if _, err := parseCleanupWindow("2026-01-01T00:00:00Z", "", errInvalidTestSentinel); err == nil || err.Error() != errInvalidTestSentinel {
		t.Fatalf("expected err.invalid for missing end, got %v", err)
	}
	if _, err := parseCleanupWindow("", "2026-01-01T00:00:00Z", "err.invalid"); err == nil || err.Error() != "err.invalid" {
		t.Fatalf("expected err.invalid for missing start, got %v", err)
	}
}

func TestParseCleanupWindow_MalformedTimestampsAreInvalid(t *testing.T) {
	if _, err := parseCleanupWindow("not-a-time", "2026-01-01T00:00:00Z", "err.invalid"); err == nil {
		t.Fatal("expected error for malformed start")
	}
	if _, err := parseCleanupWindow("2026-01-01T00:00:00Z", "not-a-time", "err.invalid"); err == nil {
		t.Fatal("expected error for malformed end")
	}
}

func TestParseCleanupWindow_EndBeforeStartIsInvalid(t *testing.T) {
	_, err := parseCleanupWindow(
		"2026-01-02T00:00:00Z",
		"2026-01-01T00:00:00Z",
		errInvalidTestSentinel,
	)
	if err == nil || err.Error() != "err.invalid" {
		t.Fatalf("expected err.invalid when end precedes start, got %v", err)
	}
}

func TestParseCleanupWindow_ValidWindowIsReturned(t *testing.T) {
	window, err := parseCleanupWindow(
		"2026-01-01T00:00:00Z",
		"2026-01-02T00:00:00Z",
		errInvalidTestSentinel,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if window == nil {
		t.Fatal("expected non-nil window")
	}
	if window.StartedAt.IsZero() || window.EndedAt.IsZero() {
		t.Fatalf("expected parsed bounds, got %+v", window)
	}
	if !window.EndedAt.After(window.StartedAt) {
		t.Fatalf("expected end after start, got %+v", window)
	}
}

// ---- parseLoginLogTime ----

func TestParseLoginLogTime_EmptyIsNotParseable(t *testing.T) {
	if _, ok := parseLoginLogTime(""); ok {
		t.Fatal("expected empty value to be unparseable")
	}
	if _, ok := parseLoginLogTime("   "); ok {
		t.Fatal("expected whitespace value to be unparseable")
	}
}

func TestParseLoginLogTime_SupportsRFC3339(t *testing.T) {
	parsed, ok := parseLoginLogTime("2026-01-02T03:04:05Z")
	if !ok {
		t.Fatal("expected RFC3339 to parse")
	}
	if parsed.Year() != 2026 || parsed.Minute() != 4 {
		t.Fatalf("unexpected parsed time: %v", parsed)
	}
}

func TestParseLoginLogTime_SupportsDateTimeMinuteLayout(t *testing.T) {
	if _, ok := parseLoginLogTime("2026-01-02 03:04"); !ok {
		t.Fatal("expected '2006-01-02 15:04' layout to parse")
	}
}

func TestParseLoginLogTime_SupportsDateTimeSecondLayout(t *testing.T) {
	if _, ok := parseLoginLogTime("2026-01-02 03:04:05"); !ok {
		t.Fatal("expected '2006-01-02 15:04:05' layout to parse")
	}
}

func TestParseLoginLogTime_RejectsGarbage(t *testing.T) {
	if _, ok := parseLoginLogTime("yesterday"); ok {
		t.Fatal("expected garbage value to be unparseable")
	}
}

// ---- normalizeUint64IDs ----

func TestNormalizeUint64IDs_NilReturnsNil(t *testing.T) {
	if got := normalizeUint64IDs(nil); got != nil {
		t.Fatalf("expected nil for nil input, got %v", got)
	}
}

func TestNormalizeUint64IDs_DropsZerosAndDuplicates(t *testing.T) {
	got := normalizeUint64IDs([]uint64{3, 0, 1, 3, 0, 2})
	if len(got) != 3 || got[0] != 3 || got[1] != 1 || got[2] != 2 {
		t.Fatalf("expected [3 1 2] preserving first-seen order, got %v", got)
	}
}

func TestNormalizeUint64IDs_AllZeroInputYieldsEmpty(t *testing.T) {
	got := normalizeUint64IDs([]uint64{0, 0})
	if len(got) != 0 {
		t.Fatalf("expected empty result, got %v", got)
	}
}

// ---- maxInt ----

func TestMaxInt_ReturnsLargerValue(t *testing.T) {
	if maxInt(3, 7) != 7 || maxInt(7, 3) != 7 {
		t.Fatal("expected larger value from maxInt")
	}
	if maxInt(-2, -5) != -2 {
		t.Fatal("expected -2 from maxInt(-2, -5)")
	}
	if maxInt(4, 4) != 4 {
		t.Fatal("expected equal values to return the value")
	}
}
