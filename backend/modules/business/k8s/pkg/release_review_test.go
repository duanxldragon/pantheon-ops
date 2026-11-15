package pkg

import (
	"strings"
	"testing"
)

func TestReleaseReviewMACBoundsTamperAndRotation(t *testing.T) {
	t.Setenv("PANTHEON_K8S_KUBECONFIG_KEY", strings.Repeat("12", 32))
	token, err := SignReleaseReview([]byte("review"))
	if err != nil {
		t.Fatal(err)
	}
	payload, err := OpenReleaseReview(token)
	if err != nil || string(payload) != "review" {
		t.Fatalf("roundtrip %v", err)
	}
	for _, invalid := range []string{"x" + token, token + "x", strings.Repeat("a", 4097), "a.b.c"} {
		if _, err := OpenReleaseReview(invalid); err == nil {
			t.Fatal("invalid MAC accepted")
		}
	}
	if _, err := SignReleaseReview([]byte(strings.Repeat("x", 2049))); err == nil {
		t.Fatal("oversized payload signed")
	}
	t.Setenv("PANTHEON_K8S_KUBECONFIG_KEY", strings.Repeat("34", 32))
	if _, err := OpenReleaseReview(token); err == nil {
		t.Fatal("rotated key accepted prior review")
	}
	t.Setenv("PANTHEON_K8S_KUBECONFIG_KEY", "")
	if _, err := SignReleaseReview([]byte("review")); err == nil {
		t.Fatal("missing key accepted")
	}
}
