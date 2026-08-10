package testmysql

import (
	"regexp"
	"strings"
	"testing"
)

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
