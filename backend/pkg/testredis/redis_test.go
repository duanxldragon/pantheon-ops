package testredis

import (
	"fmt"
	"slices"
	"strings"
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

// TestReservedDatabasesAreDistinct pins the registry OpenDB hands out. Two
// packages sharing a reserved index would reintroduce exactly the cross-package
// flush the registry exists to prevent, and the collision would be invisible:
// both packages pass alone and fail only in a full `go test ./...` run.
func TestReservedDatabasesAreDistinct(t *testing.T) {
	seen := map[int]string{
		DBDefault: "DBDefault (shared, must stay unclaimed)",
	}
	reserved := map[string]int{
		"DBAuthToken":  DBAuthToken,
		"DBLogin":      DBLogin,
		"DBMiddleware": DBMiddleware,
		"DBSession":    DBSession,
	}

	for name, index := range reserved {
		if index < 0 || index > 15 {
			t.Fatalf("%s must be a valid logical database (0-15), got %d", name, index)
		}
		if owner, taken := seen[index]; taken {
			t.Fatalf("%s reuses database %d, already claimed by %s", name, index, owner)
		}
		seen[index] = name
	}
}

// TestOpenDBIgnoresPerProcessOverride is the contract OpenDB exists for: an
// explicit index wins over PANTHEON_TEST_REDIS_DB, because the variable is
// per-process and cannot distinguish two packages inside one run. If OpenDB
// ever starts honouring it, this fails on the invalid value rather than
// silently putting every reserved package back on one database.
func TestOpenDBIgnoresPerProcessOverride(t *testing.T) {
	clearRedisEnv(t)
	// Open would reject this; OpenDB must not look at it at all.
	t.Setenv(dbEnvVar, "not-a-number")
	// Unreachable but well-formed, so openAt builds the client before pinging.
	t.Setenv(addressEnvVars[0], "127.0.0.1:1")

	reporter := &fakeReporter{}
	client := openAt(reporter, DBSession)
	if client == nil {
		t.Fatalf("openAt returned no client; failed=%v skipped=%v", reporter.failed, reporter.skipped)
	}
	t.Cleanup(func() { _ = client.Close() })

	if got := client.Options().DB; got != DBSession {
		t.Fatalf("expected database %d, got %d", DBSession, got)
	}
	for _, message := range reporter.failed {
		if strings.Contains(message, dbEnvVar) {
			t.Fatalf("an explicit index must not consult %s: %s", dbEnvVar, message)
		}
	}
}

// TestResolveDBIndexValidatesTheOverride keeps the env path honest: a bad index
// is a failure, because silently falling back to the default would put the test
// on a database another package is flushing.
func TestResolveDBIndexValidatesTheOverride(t *testing.T) {
	for _, value := range []string{"-1", "16", "abc", "1.5"} {
		t.Run(value, func(t *testing.T) {
			clearRedisEnv(t)
			t.Setenv(dbEnvVar, value)

			reporter := &fakeReporter{}
			if got := resolveDBIndex(reporter); len(reporter.failed) != 1 {
				t.Fatalf("expected %q to be rejected, got index %d (failed=%v)", value, got, reporter.failed)
			}
		})
	}

	clearRedisEnv(t)
	t.Setenv(dbEnvVar, "")
	if got := resolveDBIndex(&fakeReporter{}); got != DBDefault {
		t.Fatalf("expected the shared default %d, got %d", DBDefault, got)
	}
}
