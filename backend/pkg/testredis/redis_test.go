package testredis

import (
	"fmt"
	"slices"
	"testing"
)

// TestOpenAddressSourcesPinnedAsContract guards the trap that let Redis-backed
// tests report green while skipping: quality.yml's backend-tests job sets the
// runtime variable PANTHEON_REDIS_ADDR, so Open must accept it (not only the
// dedicated PANTHEON_TEST_REDIS_ADDR). Dropping either name from this list
// silently reduces coverage in CI.
func TestOpenAddressSourcesPinnedAsContract(t *testing.T) {
	for _, required := range []string{"PANTHEON_TEST_REDIS_ADDR", "PANTHEON_REDIS_ADDR", "REDIS_ADDR"} {
		if !slices.Contains(addressEnvVars, required) {
			t.Fatalf("addressEnvVars must keep accepting %q, got %v", required, addressEnvVars)
		}
	}
}

func TestOpenPasswordSourcesPinnedAsContract(t *testing.T) {
	for _, required := range []string{"PANTHEON_TEST_REDIS_PASSWORD", "PANTHEON_REDIS_PASSWORD", "REDIS_PASSWORD"} {
		if !slices.Contains(passwordEnvVars, required) {
			t.Fatalf("passwordEnvVars must keep accepting %q, got %v", required, passwordEnvVars)
		}
	}
}

func TestFirstEnvPrefersEarlierNameAndIgnoresBlank(t *testing.T) {
	t.Setenv("PANTHEON_TEST_REDIS_ADDR", "   ")
	t.Setenv("PANTHEON_REDIS_ADDR", "127.0.0.1:6379")
	t.Setenv("REDIS_ADDR", "127.0.0.1:6380")

	value, name := firstEnv(addressEnvVars)
	if value != "127.0.0.1:6379" || name != "PANTHEON_REDIS_ADDR" {
		t.Fatalf("expected the blank dedicated variable to be skipped, got %q from %q", value, name)
	}
}

func TestFirstEnvReturnsEmptyWhenNothingIsSet(t *testing.T) {
	clearRedisEnv(t)

	if value, name := firstEnv(addressEnvVars); value != "" || name != "" {
		t.Fatalf("expected no source, got %q from %q", value, name)
	}
}

func TestIsTruthy(t *testing.T) {
	for _, truthy := range []string{"1", "true", "TRUE", " yes ", "on"} {
		if !isTruthy(truthy) {
			t.Fatalf("expected %q to be truthy", truthy)
		}
	}
	for _, falsy := range []string{"", "0", "false", "off", "no", "maybe"} {
		if isTruthy(falsy) {
			t.Fatalf("expected %q to be falsy", falsy)
		}
	}
}

// fakeReporter records which of Fatalf/Skipf open chose. A real *testing.T
// cannot be used here because both methods terminate the calling goroutine.
type fakeReporter struct {
	skipped []string
	failed  []string
}

func (f *fakeReporter) Helper() {}

func (f *fakeReporter) Cleanup(func()) {}

func (f *fakeReporter) Fatalf(format string, args ...any) {
	f.failed = append(f.failed, fmt.Sprintf(format, args...))
}

func (f *fakeReporter) Skipf(format string, args ...any) {
	f.skipped = append(f.skipped, fmt.Sprintf(format, args...))
}

// TestOpenFailsLoudlyWhenRequired pins the second half of the guard: jobs that
// provision Redis set PANTHEON_TEST_REDIS_REQUIRED, so a renamed or missing
// address variable fails the build instead of skipping.
func TestOpenFailsLoudlyWhenRequired(t *testing.T) {
	clearRedisEnv(t)
	t.Setenv(requiredEnvVar, "true")

	reporter := &fakeReporter{}
	open(reporter)

	if len(reporter.failed) != 1 {
		t.Fatalf("open must fail (not skip) when %s is set and no address is configured; failed=%v skipped=%v", requiredEnvVar, reporter.failed, reporter.skipped)
	}
	if len(reporter.skipped) != 0 {
		t.Fatalf("open must not skip when %s is set: %v", requiredEnvVar, reporter.skipped)
	}
}

// TestOpenSkipsWhenUnconfiguredAndNotRequired records the intended local
// behaviour: without Redis configured and without the required flag the helper
// skips, so `go test ./...` still works on a bare machine.
func TestOpenSkipsWhenUnconfiguredAndNotRequired(t *testing.T) {
	clearRedisEnv(t)

	reporter := &fakeReporter{}
	open(reporter)

	if len(reporter.skipped) != 1 {
		t.Fatalf("open must skip when Redis is simply not configured; skipped=%v failed=%v", reporter.skipped, reporter.failed)
	}
	if len(reporter.failed) != 0 {
		t.Fatalf("open must not fail when Redis is simply not configured: %v", reporter.failed)
	}
}

func clearRedisEnv(t *testing.T) {
	t.Helper()
	for _, name := range append(slices.Clone(addressEnvVars), passwordEnvVars...) {
		t.Setenv(name, "")
	}
	t.Setenv(requiredEnvVar, "")
}
