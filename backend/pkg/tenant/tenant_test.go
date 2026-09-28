package tenant

import (
	"errors"
	"testing"
)

func TestResolveForCanary_CompatAlwaysGlobal(t *testing.T) {
	cases := []struct {
		name       string
		mode       string
		subject    string
		header     string
		headerOK   bool
		wantBy     string
		wantTenant uint64
	}{
		{name: "default empty mode", mode: "", subject: "7", header: "", headerOK: false, wantBy: "compat-fallback", wantTenant: 0},
		{name: "explicit compat ignores subject", mode: "compat", subject: "7", header: "", headerOK: false, wantBy: "compat-fallback", wantTenant: 0},
		{name: "compat ignores header even if allowed", mode: "compat", subject: "7", header: "7", headerOK: true, wantBy: "compat-fallback", wantTenant: 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, err := ResolveForCanary(tc.mode, tc.subject, tc.header, tc.headerOK)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if ctx.TenantID != tc.wantTenant || ctx.ResolvedBy != tc.wantBy || ctx.IsMulti() {
				t.Fatalf("got %+v", ctx)
			}
		})
	}
}

func TestResolveForCanary_MultiMode(t *testing.T) {
	ctx, err := ResolveForCanary("multi", "42", "", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ctx.IsMulti() || ctx.TenantID != 42 || ctx.ResolvedBy != "subject" {
		t.Fatalf("got %+v", ctx)
	}

	// header path allowed: wins over subject
	ctx, err = ResolveForCanary("multi", "42", "43", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ctx.IsMulti() || ctx.TenantID != 43 || ctx.ResolvedBy != "header" {
		t.Fatalf("got %+v", ctx)
	}

	// header without permission: forbidden
	if _, err := ResolveForCanary("multi", "42", "43", false); !errors.Is(err, ErrTenantForbidden) {
		t.Fatalf("expected ErrTenantForbidden, got %v", err)
	}

	// malformed header value: forbidden
	if _, err := ResolveForCanary("multi", "42", "abc", true); !errors.Is(err, ErrTenantForbidden) {
		t.Fatalf("expected ErrTenantForbidden for malformed id, got %v", err)
	}

	// explicit 0 in header: invalid, never widens to global
	if _, err := ResolveForCanary("multi", "42", "0", true); !errors.Is(err, ErrTenantForbidden) {
		t.Fatalf("expected ErrTenantForbidden for tenant 0 header, got %v", err)
	}

	// no trusted context in multi mode: deny-by-default
	if _, err := ResolveForCanary("multi", "", "", false); !errors.Is(err, ErrTenantContextMissing) {
		t.Fatalf("expected ErrTenantContextMissing, got %v", err)
	}
}

func TestNormalizeMode(t *testing.T) {
	if NormalizeMode("multi") != ModeMulti {
		t.Fatal("multi must pass through")
	}
	for _, raw := range []string{"", "compat", "unknown", " MULTI "} {
		if NormalizeMode(raw) != ModeCompat {
			t.Fatalf("expected compat fallback for %q", raw)
		}
	}
}

func TestModeLoader_CachesAndFailsSafe(t *testing.T) {
	now := int64(1_000_000_000)
	timeNowNano = func() int64 { return now }
	t.Cleanup(func() { timeNowNano = realTimeNowNano })

	calls := 0
	loader := NewModeLoader(func() string {
		calls++
		return "multi"
	}, 1_000)

	for i := 0; i < 3; i++ {
		if got := loader.Load(); got != ModeMulti {
			t.Fatalf("expected multi, got %s", got)
		}
	}
	if calls != 1 {
		t.Fatalf("expected 1 load call, got %d", calls)
	}

	// TTL expiry re-reads
	now += 2_000
	if got := loader.Load(); got != ModeMulti {
		t.Fatalf("expected multi after ttl, got %s", got)
	}
	if calls != 2 {
		t.Fatalf("expected reload after ttl, got %d calls", calls)
	}

	// loader error / unknown value fails safe to compat
	failing := NewModeLoader(func() string { return "bogus" }, 0)
	if got := failing.Load(); got != ModeCompat {
		t.Fatalf("expected compat fail-safe, got %s", got)
	}
	nilLoader := NewModeLoader(nil, 0)
	if got := nilLoader.Load(); got != ModeCompat {
		t.Fatalf("expected compat for nil loader, got %s", got)
	}
}
