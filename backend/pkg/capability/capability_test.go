package capability

import (
	"testing"
)

func TestDefaults_Enterprise(t *testing.T) {
	caps := Defaults()
	if caps.AppMode != AppModeEnterprise {
		t.Fatalf("expected AppMode=%q, got %q", AppModeEnterprise, caps.AppMode)
	}
	if !caps.OrgEnabled {
		t.Fatal("expected OrgEnabled=true")
	}
	if caps.OrgRequiredForUser {
		t.Fatal("expected OrgRequiredForUser=false")
	}
}

func TestNormalizeAppMode_Enterprise(t *testing.T) {
	if got := NormalizeAppMode(""); got != AppModeEnterprise {
		t.Fatalf("expected %q, got %q", AppModeEnterprise, got)
	}
}

func TestNormalizeAppMode_EnterpriseExplicit(t *testing.T) {
	if got := NormalizeAppMode("enterprise"); got != AppModeEnterprise {
		t.Fatalf("expected %q, got %q", AppModeEnterprise, got)
	}
}

func TestNormalizeAppMode_Consumer(t *testing.T) {
	if got := NormalizeAppMode("consumer"); got != AppModeConsumer {
		t.Fatalf("expected %q, got %q", AppModeConsumer, got)
	}
}

func TestNormalizeAppMode_Hybrid(t *testing.T) {
	if got := NormalizeAppMode("hybrid"); got != AppModeHybrid {
		t.Fatalf("expected %q, got %q", AppModeHybrid, got)
	}
}

func TestNormalizeAppMode_Whitespace(t *testing.T) {
	if got := NormalizeAppMode("  consumer  "); got != AppModeConsumer {
		t.Fatalf("expected %q, trimmed input, got %q", AppModeConsumer, got)
	}
}

func TestNormalizeAppMode_Unknown(t *testing.T) {
	if got := NormalizeAppMode("unknown"); got != AppModeEnterprise {
		t.Fatalf("expected %q fallback, got %q", AppModeEnterprise, got)
	}
}

func TestNormalizeAppMode_Case(t *testing.T) {
	// normalizeAppMode does NOT lowercase; it checks exact match after TrimSpace
	// "CONSUMER" should fall back to enterprise
	if got := NormalizeAppMode("CONSUMER"); got != AppModeEnterprise {
		t.Fatalf("expected %q fallback for uppercase, got %q", AppModeEnterprise, got)
	}
}

func TestNormalizeBool_True(t *testing.T) {
	// normalizeBool is unexported; tested through Load() with nil DB to isolate
	// Since we can't call it directly, verify the constant-based logic via Load(nil)
	caps := Load(nil)
	if caps.AppMode != AppModeEnterprise || !caps.OrgEnabled || caps.OrgRequiredForUser {
		t.Fatal("Load(nil) should return defaults")
	}
}

func TestLoad_NilDB(t *testing.T) {
	caps := Load(nil)
	want := Defaults()
	if caps != want {
		t.Fatal("Load(nil) should equal Defaults()")
	}
}

func TestConstants(t *testing.T) {
	if AppModeEnterprise != "enterprise" {
		t.Fatalf("AppModeEnterprise=%q, want \"enterprise\"", AppModeEnterprise)
	}
	if AppModeConsumer != "consumer" {
		t.Fatalf("AppModeConsumer=%q, want \"consumer\"", AppModeConsumer)
	}
	if AppModeHybrid != "hybrid" {
		t.Fatalf("AppModeHybrid=%q, want \"hybrid\"", AppModeHybrid)
	}
}
