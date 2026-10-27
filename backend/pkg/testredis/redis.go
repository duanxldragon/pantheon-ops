package testredis

import (
	"context"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

// Address sources, in priority order. PANTHEON_TEST_REDIS_ADDR is the dedicated
// test variable; PANTHEON_REDIS_ADDR is the runtime variable the server itself
// reads and the one most CI jobs and local .env files set. Accepting both means a
// Redis-backed test can no longer silently skip just because the runtime name was
// used (see docs/operations/RUNBOOK.md §Redis-backed tests).
var addressEnvVars = []string{
	"PANTHEON_TEST_REDIS_ADDR",
	"PANTHEON_REDIS_ADDR",
	"REDIS_ADDR",
}

var passwordEnvVars = []string{
	"PANTHEON_TEST_REDIS_PASSWORD",
	"PANTHEON_REDIS_PASSWORD",
	"REDIS_PASSWORD",
}

// requiredEnvVar switches Open from "skip when unconfigured" to "fail". Set it in
// any job that provisions Redis so a typo'd or renamed variable fails the build
// instead of reducing coverage silently.
const requiredEnvVar = "PANTHEON_TEST_REDIS_REQUIRED"

// dbEnvVar selects the Redis logical database for a test process. The default
// (1) is shared by every package, which is fine for tests that only write and
// read within one call, but not for tests asserting that a key survives a
// concurrent `go test ./...` run: another package's cleanup flushes the DB.
const dbEnvVar = "PANTHEON_TEST_REDIS_DB"

// Logical database assignment for packages that assert a key survives across
// calls (for example a revocation test that seeds a token, calls the service
// and then checks the token is gone).
//
// `go test ./...` runs packages in parallel and every Open flushes its database
// on cleanup, so two packages on the same index wipe each other mid-assertion.
// The default index cannot be fixed for them by an environment variable either:
// the variable is per test process, so it cannot differ between packages inside
// one run. Each such package therefore claims a reserved index here, and
// OpenDB is how it does so. The registry is the whole point - adding a new
// reserved index means adding a constant, so two packages cannot silently pick
// the same number.
//
// DB 1 stays the shared default. Indexes 2-5 are reserved and must not be used
// by an Open caller.
const (
	DBDefault    = 1
	DBAuthToken  = 2
	DBLogin      = 3
	DBMiddleware = 4
	DBSession    = 5
)

func isTruthy(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func firstEnv(names []string) (string, string) {
	for _, name := range names {
		if value := strings.TrimSpace(os.Getenv(name)); value != "" {
			return value, name
		}
	}
	return "", ""
}

// reporter is the subset of *testing.T that open needs. Taking the narrow
// interface (instead of *testing.T directly) keeps the fail-vs-skip decision
// testable: t.Fatalf/t.Skipf end the calling goroutine, so a test cannot assert
// on a real *testing.T.
type reporter interface {
	Helper()
	Fatalf(format string, args ...any)
	Skipf(format string, args ...any)
	Cleanup(func())
}

// Open creates a Redis client for integration tests.
//
// It uses the shared default database (or PANTHEON_TEST_REDIS_DB when set), so
// it is only safe for tests that finish reading before another package could
// flush. A test that asserts a key survives a call must use OpenDB instead.
func Open(t *testing.T) *redis.Client {
	return open(t)
}

// OpenDB creates a Redis client in an explicit logical database, for a package
// that asserts keys survive across calls. Pass one of the reserved DB*
// constants so two packages cannot collide.
//
// PANTHEON_TEST_REDIS_DB is deliberately ignored: an explicit index is a
// stronger statement than an environment default, and honouring the variable
// here would let a single run put every reserved package back on one database.
func OpenDB(t *testing.T, dbIndex int) *redis.Client {
	t.Helper()
	return openAt(t, dbIndex)
}

func open(t reporter) *redis.Client {
	t.Helper()
	return openAt(t, resolveDBIndex(t))
}

// resolveDBIndex reads the per-process override, falling back to the shared
// default. The range check is a failure, not a skip: a typo'd index would
// otherwise land tests on whatever database the number happened to name.
func resolveDBIndex(t reporter) int {
	t.Helper()

	raw := strings.TrimSpace(os.Getenv(dbEnvVar))
	if raw == "" {
		return DBDefault
	}
	parsed, err := strconv.Atoi(raw)
	if err != nil || parsed < 0 || parsed > 15 {
		t.Fatalf("%s must be an integer 0-15, got %q", dbEnvVar, raw)
	}
	return parsed
}

func openAt(t reporter, dbIndex int) *redis.Client {
	t.Helper()

	if dbIndex < 0 || dbIndex > 15 {
		t.Fatalf("redis database index must be 0-15, got %d", dbIndex)
	}

	addr, addrVar := firstEnv(addressEnvVars)
	if addr == "" {
		if isTruthy(os.Getenv(requiredEnvVar)) {
			t.Fatalf(
				"redis is required (%s is set) but none of %s is configured; a misconfigured variable would otherwise skip this test silently",
				requiredEnvVar,
				strings.Join(addressEnvVars, ", "),
			)
		} else {
			t.Skipf("redis addr is not configured (%s unset)", strings.Join(addressEnvVars, ", "))
		}
		// Both reporters terminate the test, but return explicitly so the decision
		// stays correct for a reporter that records instead of stopping.
		return nil
	}

	password, _ := firstEnv(passwordEnvVars)

	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       dbIndex,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		// A reachable address that refuses the connection is always a failure: the
		// caller configured Redis, so the test must not be skipped.
		t.Fatalf("ping redis at %s (from %s): %v", addr, addrVar, err)
	}

	t.Cleanup(func() {
		// Flush test DB
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = rdb.FlushDB(ctx).Err()
		_ = rdb.Close()
	})

	return rdb
}
