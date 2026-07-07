package mfa

import (
	"strings"
	"testing"
	"time"
)

func TestGenerateTOTPSecret_Length(t *testing.T) {
	secret, err := GenerateTOTPSecret()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(secret) < 20 {
		t.Fatalf("expected secret length >= 20, got %d", len(secret))
	}
}

func TestGenerateTOTPSecret_Base32(t *testing.T) {
	secret, err := GenerateTOTPSecret()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	decoded, err := decodeBase32NoPadding(secret)
	if err != nil {
		t.Fatalf("generated secret should be valid base32: %v", err)
	}
	if len(decoded) != 20 {
		t.Fatalf("expected 20 bytes, got %d", len(decoded))
	}
}

func TestGenerateTOTPSecret_Unique(t *testing.T) {
	s1, _ := GenerateTOTPSecret()
	s2, _ := GenerateTOTPSecret()
	if s1 == s2 {
		t.Fatal("secrets should be unique")
	}
}

func TestBuildTOTPURL_WithUsername(t *testing.T) {
	url := BuildTOTPURL("alice", "SECRETKEY")
	if !strings.HasPrefix(url, "otpauth://totp/") {
		t.Fatalf("expected otpauth://totp/ prefix, got %s", url)
	}
	if !strings.Contains(url, "alice") {
		t.Fatalf("expected URL to contain username, got %s", url)
	}
	if !strings.Contains(url, "secret=SECRETKEY") {
		t.Fatalf("expected URL to contain secret, got %s", url)
	}
}

func TestBuildTOTPURL_EmptyUsername(t *testing.T) {
	url := BuildTOTPURL("", "SECRET")
	if !strings.Contains(url, "pantheon-user") {
		t.Fatalf("expected fallback username, got %s", url)
	}
}

func TestBuildTOTPURL_TrimsSpace(t *testing.T) {
	url := BuildTOTPURL("  bob  ", "  SECRET  ")
	if !strings.Contains(url, "secret=SECRET") {
		t.Fatalf("expected secret to be trimmed, got %s", url)
	}
}

func TestBuildTOTPURL_IncludesIssuer(t *testing.T) {
	url := BuildTOTPURL("alice", "SECRET")
	if !strings.Contains(url, "issuer=Pantheon+Base") {
		t.Fatalf("expected issuer, got %s", url)
	}
}

func TestBuildTOTPURL_IncludesParams(t *testing.T) {
	url := BuildTOTPURL("alice", "SECRET")
	if !strings.Contains(url, "algorithm=SHA1") {
		t.Fatalf("expected algorithm, got %s", url)
	}
	if !strings.Contains(url, "digits=6") {
		t.Fatalf("expected digits, got %s", url)
	}
	if !strings.Contains(url, "period=30") {
		t.Fatalf("expected period, got %s", url)
	}
}

func TestDecodeBase32NoPadding_Valid(t *testing.T) {
	secret := "JBSWY3DPEHPK3PXP"
	decoded, err := decodeBase32NoPadding(secret)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(decoded) == 0 {
		t.Fatal("expected decoded bytes")
	}
}

func TestDecodeBase32NoPadding_Empty(t *testing.T) {
	_, err := decodeBase32NoPadding("")
	if err == nil {
		t.Fatal("expected error for empty secret")
	}
}

func TestDecodeBase32NoPadding_Whitespace(t *testing.T) {
	_, err := decodeBase32NoPadding("  ")
	if err == nil {
		t.Fatal("expected error for whitespace-only secret")
	}
}

func TestDecodeBase32NoPadding_Uppercases(t *testing.T) {
	secret, _ := GenerateTOTPSecret()
	lower := strings.ToLower(secret)
	decoded, err := decodeBase32NoPadding(lower)
	if err != nil {
		t.Fatalf("should uppercase before decode: %v", err)
	}
	if len(decoded) != 20 {
		t.Fatalf("expected 20 bytes, got %d", len(decoded))
	}
}

func TestGenerateTOTPCode_Deterministic(t *testing.T) {
	secret := "JBSWY3DPEHPK3PXP"
	code1 := generateTOTPCode(secret, 1000)
	code2 := generateTOTPCode(secret, 1000)
	if code1 != code2 {
		t.Fatalf("same counter should produce same code, got %s vs %s", code1, code2)
	}
}

func TestGenerateTOTPCode_DifferentCounters(t *testing.T) {
	secret := "JBSWY3DPEHPK3PXP"
	code1 := generateTOTPCode(secret, 1000)
	code2 := generateTOTPCode(secret, 1001)
	if code1 == code2 {
		t.Fatal("different counters should produce different codes")
	}
}

func TestGenerateTOTPCode_SixDigits(t *testing.T) {
	secret := "JBSWY3DPEHPK3PXP"
	code := generateTOTPCode(secret, 1000)
	if len(code) != 6 {
		t.Fatalf("expected 6 digits, got %d: %s", len(code), code)
	}
}

func TestGenerateTOTPCode_InvalidSecretReturnsEmpty(t *testing.T) {
	code := generateTOTPCode("!", 1000)
	if code != "" {
		t.Fatalf("expected empty for invalid secret, got %s", code)
	}
}

func TestValidateTOTPCode_InvalidLength(t *testing.T) {
	secret := "JBSWY3DPEHPK3PXP"
	if ValidateTOTPCode(secret, "12345", time.Now()) {
		t.Fatal("should reject 5-digit code")
	}
	if ValidateTOTPCode(secret, "1234567", time.Now()) {
		t.Fatal("should reject 7-digit code")
	}
}

func TestValidateTOTPCode_NonNumeric(t *testing.T) {
	secret := "JBSWY3DPEHPK3PXP"
	if ValidateTOTPCode(secret, "abcdef", time.Now()) {
		t.Fatal("should reject non-numeric code")
	}
}

func TestValidateTOTPCode_Empty(t *testing.T) {
	if ValidateTOTPCode("JBSWY3DPEHPK3PXP", "", time.Now()) {
		t.Fatal("should reject empty code")
	}
}

func TestValidateTOTPCode_RoundTrip(t *testing.T) {
	secret := "JBSWY3DPEHPK3PXP"
	now := time.Now()
	counter := now.Unix() / totpPeriod
	code := generateTOTPCode(secret, counter)
	if code == "" {
		t.Fatal("expected valid code")
	}
	if !ValidateTOTPCode(secret, code, now) {
		t.Fatal("should validate own code")
	}
}

func TestValidateTOTPCode_AllowsSkew(t *testing.T) {
	secret := "JBSWY3DPEHPK3PXP"
	now := time.Now()
	prevCounter := (now.Unix() / totpPeriod) - 1
	prevCode := generateTOTPCode(secret, prevCounter)
	if !ValidateTOTPCode(secret, prevCode, now) {
		t.Fatal("should allow 1-period skew")
	}
}

func TestValidateTOTPCode_RejectsFarSkew(t *testing.T) {
	secret := "JBSWY3DPEHPK3PXP"
	now := time.Now()
	farCounter := (now.Unix() / totpPeriod) - 3
	farCode := generateTOTPCode(secret, farCounter)
	if ValidateTOTPCode(secret, farCode, now) {
		t.Fatal("should reject code from 3 periods ago")
	}
}

func TestTOTPConstants(t *testing.T) {
	if totpPeriod != 30 {
		t.Fatalf("expected period 30, got %d", totpPeriod)
	}
	if totpDigits != 6 {
		t.Fatalf("expected digits 6, got %d", totpDigits)
	}
	if totpAllowedSkew != 1 {
		t.Fatalf("expected skew 1, got %d", totpAllowedSkew)
	}
}
