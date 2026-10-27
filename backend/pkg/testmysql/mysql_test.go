package testmysql

import (
	"fmt"
	"regexp"
	"strings"
	"testing"
)

type fakeReporter struct {
	name    string
	fatals  []string
	skips   []string
	cleanup []func()
}

func (f *fakeReporter) Helper() {}

func (f *fakeReporter) Fatalf(format string, args ...any) {
	f.fatals = append(f.fatals, fmt.Sprintf(format, args...))
}

func (f *fakeReporter) Skipf(format string, args ...any) {
	f.skips = append(f.skips, fmt.Sprintf(format, args...))
}

func (f *fakeReporter) Cleanup(fn func()) { f.cleanup = append(f.cleanup, fn) }

func (f *fakeReporter) Name() string { return f.name }

func TestResolveDSNFailsWhenRequiredAndUnconfigured(t *testing.T) {
	t.Setenv("PANTHEON_TEST_DSN", "")
	t.Setenv("PANTHEON_DSN", "")
	t.Setenv(requiredEnvVar, "true")

	reporter := &fakeReporter{name: "TestResolveDSN"}
	dsn := resolveDSN(reporter)

	if dsn != "" {
		t.Fatalf("resolveDSN() = %q, want empty", dsn)
	}
	if len(reporter.fatals) != 1 {
		t.Fatalf("resolveDSN() fatals = %d, want 1", len(reporter.fatals))
	}
	if len(reporter.skips) != 0 {
		t.Fatalf("resolveDSN() skips = %d, want 0", len(reporter.skips))
	}
	if !strings.Contains(reporter.fatals[0], requiredEnvVar) {
		t.Fatalf("resolveDSN() fatal message should name %s, got %q", requiredEnvVar, reporter.fatals[0])
	}
}

func TestResolveDSNSkipsWhenOptionalAndUnconfigured(t *testing.T) {
	t.Setenv("PANTHEON_TEST_DSN", "")
	t.Setenv("PANTHEON_DSN", "")
	t.Setenv(requiredEnvVar, "")

	reporter := &fakeReporter{name: "TestResolveDSN"}
	dsn := resolveDSN(reporter)

	if dsn != "" {
		t.Fatalf("resolveDSN() = %q, want empty", dsn)
	}
	if len(reporter.skips) != 1 {
		t.Fatalf("resolveDSN() skips = %d, want 1", len(reporter.skips))
	}
	if len(reporter.fatals) != 0 {
		t.Fatalf("resolveDSN() fatals = %d, want 0", len(reporter.fatals))
	}
}

func TestResolveDSNReturnsConfiguredDSN(t *testing.T) {
	t.Setenv("PANTHEON_TEST_DSN", " user:pass@tcp(localhost:3306)/pantheon ")
	t.Setenv("PANTHEON_DSN", "")

	reporter := &fakeReporter{name: "TestResolveDSN"}
	dsn := resolveDSN(reporter)

	if dsn != "user:pass@tcp(localhost:3306)/pantheon" {
		t.Fatalf("resolveDSN() = %q, want trimmed DSN", dsn)
	}
	if len(reporter.fatals) != 0 || len(reporter.skips) != 0 {
		t.Fatalf("resolveDSN() recorded fatal/skip for a configured DSN")
	}
}

func TestResolveDSNRequiredAcceptsTruthyValues(t *testing.T) {
	for _, value := range []string{"true", "TRUE", "1", "yes", "on"} {
		t.Run(value, func(t *testing.T) {
			t.Setenv("PANTHEON_TEST_DSN", "")
			t.Setenv("PANTHEON_DSN", "")
			t.Setenv(requiredEnvVar, value)

			reporter := &fakeReporter{name: "TestResolveDSN"}
			dsn := resolveDSN(reporter)

			if dsn != "" || len(reporter.fatals) != 1 {
				t.Fatalf("resolveDSN() with %s=%q should fail, dsn=%q fatals=%d", requiredEnvVar, value, dsn, len(reporter.fatals))
			}
		})
	}
}

func TestResolveDSNRequiredIgnoresFalsyValues(t *testing.T) {
	for _, value := range []string{"0", "false", "no", "off", "anything"} {
		t.Run(value, func(t *testing.T) {
			t.Setenv("PANTHEON_TEST_DSN", "")
			t.Setenv("PANTHEON_DSN", "")
			t.Setenv(requiredEnvVar, value)

			reporter := &fakeReporter{name: "TestResolveDSN"}
			dsn := resolveDSN(reporter)

			if dsn != "" || len(reporter.skips) != 1 {
				t.Fatalf("resolveDSN() with %s=%q should skip, dsn=%q skips=%d", requiredEnvVar, value, dsn, len(reporter.skips))
			}
		})
	}
}

func TestResolveDSNFallsBackToPantheonDSN(t *testing.T) {
	t.Setenv("PANTHEON_TEST_DSN", "")
	t.Setenv("PANTHEON_DSN", " env:pass@tcp(localhost:3306)/pantheon ")
	t.Setenv(requiredEnvVar, "true")

	reporter := &fakeReporter{name: "TestResolveDSN"}
	dsn := resolveDSN(reporter)

	if dsn != "env:pass@tcp(localhost:3306)/pantheon" {
		t.Fatalf("resolveDSN() = %q, want PANTHEON_DSN fallback", dsn)
	}
}

func TestBuildTestDBNameSanitizesSegments(t *testing.T) {
	name, err := buildTestDBName(" Main DB ", "suite/Test Case")
	if err != nil {
		t.Fatalf("buildTestDBName() error = %v", err)
	}
	if !strings.HasPrefix(name, "main_db_suite_test_case_") {
		t.Fatalf("buildTestDBName() prefix = %q", name)
	}
	if len(name) > 60 {
		t.Fatalf("buildTestDBName() length = %d, want <= 60", len(name))
	}
}

func TestBuildTestDBNamePreservesRandomSuffixWhenPrefixIsTruncated(t *testing.T) {
	name, err := buildTestDBName(
		"pantheon_base",
		"TestAuthPreferencesContract_GetMeIncludesNormalizedSecurityPolicyAndSessionInventory",
	)
	if err != nil {
		t.Fatalf("buildTestDBName() error = %v", err)
	}
	if len(name) > maxTestDBNameLength {
		t.Fatalf("buildTestDBName() length = %d, want <= %d", len(name), maxTestDBNameLength)
	}
	if !strings.HasPrefix(name, "pantheon_base_") {
		t.Fatalf("buildTestDBName() prefix = %q", name)
	}
	if !strings.Contains(name, "testauthpreferences") {
		t.Fatalf("buildTestDBName() should retain part of normalized test name, got %q", name)
	}
	if !regexp.MustCompile(`_\d+_\d{4}$`).MatchString(name) {
		t.Fatalf("buildTestDBName() must retain numeric suffix, got %q", name)
	}
}

func TestQuoteMySQLIdentifierRejectsUnsafeNames(t *testing.T) {
	defer func() {
		recovered := recover()
		if recovered == nil {
			t.Fatal("quoteMySQLIdentifier() should panic for unsafe identifiers")
		}
	}()

	quoteMySQLIdentifier("unsafe-name")
}

func TestQuoteMySQLIdentifierWrapsSafeNames(t *testing.T) {
	if got := quoteMySQLIdentifier("safe_name_01"); got != "`safe_name_01`" {
		t.Fatalf("quoteMySQLIdentifier() = %q", got)
	}
}
