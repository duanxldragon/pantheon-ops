package common

import "testing"

func TestValidateRequiredProductionSecret(t *testing.T) {
	t.Setenv("PANTHEON_ENV", "production")
	t.Setenv("TEST_SECRET", "")

	if err := ValidateRequiredProductionSecret("TEST_SECRET", "fallback"); err == nil {
		t.Fatal("expected missing secret to fail in production")
	}

	t.Setenv("TEST_SECRET", "fallback")
	if err := ValidateRequiredProductionSecret("TEST_SECRET", "fallback"); err == nil {
		t.Fatal("expected fallback secret to fail in production")
	}

	t.Setenv("TEST_SECRET", "real-secret-that-is-at-least-32-bytes-long!")
	if err := ValidateRequiredProductionSecret("TEST_SECRET", "fallback"); err != nil {
		t.Fatalf("expected configured secret to pass: %v", err)
	}
}

func TestValidateRequiredProductionSecretNonProductionAllowsFallback(t *testing.T) {
	t.Setenv("PANTHEON_ENV", "development")
	t.Setenv("TEST_SECRET", "")

	if err := ValidateRequiredProductionSecret("TEST_SECRET", "fallback"); err != nil {
		t.Fatalf("expected non-production to allow fallback: %v", err)
	}
}
