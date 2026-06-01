package sonarbaseline

import "testing"

func TestNormalizeLabel(t *testing.T) {
	t.Parallel()

	if got := NormalizeLabel(""); got != "baseline" {
		t.Fatalf("expected baseline fallback, got %q", got)
	}
	if got := NormalizeLabel("ops"); got != "ops" {
		t.Fatalf("expected original label, got %q", got)
	}
}
