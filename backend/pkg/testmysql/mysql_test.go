package testmysql

import (
	"strings"
	"testing"
)

func TestBuildTestDBNameSanitizesSegments(t *testing.T) {
	name := buildTestDBName("Pantheon`;DROP", "Test/Case #1")
	if name == "" {
		t.Fatal("expected database name")
	}
	if strings.ContainsAny(name, "`;#- ") {
		t.Fatalf("expected sanitized database name, got %q", name)
	}
	if len(name) > 60 {
		t.Fatalf("expected database name length <= 60, got %d", len(name))
	}
	if !testDBIdentifierPattern.MatchString(name) {
		t.Fatalf("expected database name to match identifier pattern, got %q", name)
	}
}

func TestQuoteDatabaseIdentifierRejectsUnsafeNames(t *testing.T) {
	if _, err := quoteDatabaseIdentifier("safe_name"); err != nil {
		t.Fatalf("expected safe identifier accepted, got %v", err)
	}
	if _, err := quoteDatabaseIdentifier("unsafe-name"); err == nil {
		t.Fatal("expected unsafe identifier rejected")
	}
}
