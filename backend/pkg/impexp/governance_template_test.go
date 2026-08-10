package impexp

import (
	"testing"
)

func TestCountGovernanceProblems_None(t *testing.T) {
	tags := []string{"clean", "root"}
	problems := map[string]struct{}{"leaderless": {}, "no-post": {}}
	count := CountGovernanceProblems(tags, problems)
	if count != 0 {
		t.Fatalf("expected 0 problems, got %d", count)
	}
}

func TestCountGovernanceProblems_Some(t *testing.T) {
	tags := []string{"leaderless", "clean", "no-post", "root"}
	problems := map[string]struct{}{"leaderless": {}, "no-post": {}}
	count := CountGovernanceProblems(tags, problems)
	if count != 2 {
		t.Fatalf("expected 2 problems, got %d", count)
	}
}

func TestCountGovernanceProblems_EmptyTags(t *testing.T) {
	count := CountGovernanceProblems([]string{}, map[string]struct{}{"leaderless": {}})
	if count != 0 {
		t.Fatalf("expected 0, got %d", count)
	}
}

func TestCountGovernanceProblems_EmptyProblems(t *testing.T) {
	count := CountGovernanceProblems([]string{"leaderless"}, map[string]struct{}{})
	if count != 0 {
		t.Fatalf("expected 0, got %d", count)
	}
}

func TestGovernanceScopeLabel_Known(t *testing.T) {
	if got := GovernanceScopeLabel("dept"); got != "Department" {
		t.Fatalf("expected \"Department\", got %q", got)
	}
	if got := GovernanceScopeLabel("post"); got != "Post" {
		t.Fatalf("expected \"Post\", got %q", got)
	}
}

func TestGovernanceScopeLabel_Unknown(t *testing.T) {
	if got := GovernanceScopeLabel("unknown"); got != "" {
		t.Fatalf("expected empty string, got %q", got)
	}
}

func TestGovernanceScopeLabel_Empty(t *testing.T) {
	if got := GovernanceScopeLabel(""); got != "" {
		t.Fatalf("expected empty string, got %q", got)
	}
}

func TestGovernanceTagLabels_Empty(t *testing.T) {
	if got := GovernanceTagLabels(nil); got != "" {
		t.Fatalf("expected empty string, got %q", got)
	}
}

func TestGovernanceTagLabels_EmptySlice(t *testing.T) {
	if got := GovernanceTagLabels([]string{}); got != "" {
		t.Fatalf("expected empty string, got %q", got)
	}
}

func TestGovernanceTagLabels_Single(t *testing.T) {
	if got := GovernanceTagLabels([]string{"clean"}); got != "Healthy" {
		t.Fatalf("expected \"Healthy\", got %q", got)
	}
}

func TestGovernanceTagLabels_Multiple(t *testing.T) {
	got := GovernanceTagLabels([]string{"leaderless", "clean"})
	// joinGovernanceLabels preserves input order
	if got != "Leader Missing | Healthy" {
		t.Fatalf("expected \"Leader Missing | Healthy\", got %q", got)
	}
}

func TestGovernanceTagLabels_UnknownFallback(t *testing.T) {
	got := GovernanceTagLabels([]string{"unknown-tag", "clean"})
	if got != "unknown-tag | Healthy" {
		t.Fatalf("expected \"unknown-tag | Healthy\", got %q", got)
	}
}

func TestGovernanceBlockedByLabels_Empty(t *testing.T) {
	if got := GovernanceBlockedByLabels(nil); got != "" {
		t.Fatalf("expected empty, got %q", got)
	}
}

func TestGovernanceBlockedByLabels_Single(t *testing.T) {
	if got := GovernanceBlockedByLabels([]string{"none"}); got != "No Blocker" {
		t.Fatalf("expected \"No Blocker\", got %q", got)
	}
}

func TestGovernanceBlockedByLabels_Multiple(t *testing.T) {
	got := GovernanceBlockedByLabels([]string{"children", "users"})
	if got != "Child Departments | Users" {
		t.Fatalf("expected \"Child Departments | Users\", got %q", got)
	}
}

func TestGovernanceActionLabels_Empty(t *testing.T) {
	if got := GovernanceActionLabels(nil); got != "" {
		t.Fatalf("expected empty, got %q", got)
	}
}

func TestGovernanceActionLabels_Single(t *testing.T) {
	if got := GovernanceActionLabels([]string{"assign-leader"}); got != "Assign Leader" {
		t.Fatalf("expected \"Assign Leader\", got %q", got)
	}
}

func TestGovernanceActionLabels_Multiple(t *testing.T) {
	got := GovernanceActionLabels([]string{"keep-observing", "delete-or-keep-disabled"})
	if got != "Keep Observing | Delete or Keep Disabled" {
		t.Fatalf("expected \"Keep Observing | Delete or Keep Disabled\", got %q", got)
	}
}

func TestGovernanceActionLabels_SortedByInput(t *testing.T) {
	// joinGovernanceLabels preserves input order; does NOT sort
	got := GovernanceActionLabels([]string{"delete-or-keep-disabled", "keep-observing"})
	if got != "Delete or Keep Disabled | Keep Observing" {
		t.Fatalf("expected input order preserved, got %q", got)
	}
}

func TestJoinGovernanceLabels_Single(t *testing.T) {
	if got := joinGovernanceLabels([]string{"a"}); got != "a" {
		t.Fatalf("expected \"a\", got %q", got)
	}
}

func TestJoinGovernanceLabels_Multiple(t *testing.T) {
	if got := joinGovernanceLabels([]string{"a", "b", "c"}); got != "a | b | c" {
		t.Fatalf("expected \"a | b | c\", got %q", got)
	}
}

func TestJoinGovernanceLabels_Empty(t *testing.T) {
	if got := joinGovernanceLabels(nil); got != "" {
		t.Fatalf("expected empty, got %q", got)
	}
}

func TestGovernanceValueLabels_NilWraps(t *testing.T) {
	// Ensure GovernanceTagLabels/BlockedBy/ActionLabels all route through GovernanceValueLabels
	if got := GovernanceValueLabels(nil, nil); got != "" {
		t.Fatalf("expected empty for nil input, got %q", got)
	}
}
