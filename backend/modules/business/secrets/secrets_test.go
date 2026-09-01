package secrets

import (
	"os"
	"strings"
	"testing"
)

const testKey = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef" // 64 hex chars = 32 bytes

func TestEncryptDecryptRoundtrip(t *testing.T) {
	os.Setenv(BusinessSecretKeyEnv, testKey)
	defer os.Unsetenv(BusinessSecretKeyEnv)

	plain := "s3cr3t-p@ssw0rd"
	enc, err := Encrypt(plain)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	if enc == plain {
		t.Fatalf("ciphertext must differ from plaintext, got %q", enc)
	}
	if !strings.HasPrefix(enc, "v1:") {
		t.Fatalf("expected v1: prefix, got %q", enc)
	}

	dec, err := Decrypt(enc)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	if dec != plain {
		t.Fatalf("roundtrip mismatch: got %q want %q", dec, plain)
	}
}

func TestEncryptEmpty(t *testing.T) {
	os.Unsetenv(BusinessSecretKeyEnv)
	enc, err := Encrypt("")
	if err != nil {
		t.Fatalf("encrypt empty: %v", err)
	}
	if enc != "" {
		t.Fatalf("expected empty ciphertext, got %q", enc)
	}
}

func TestEncryptWithoutKey(t *testing.T) {
	os.Unsetenv(BusinessSecretKeyEnv)
	if _, err := Encrypt("secret"); err == nil || err.Error() != errKeyMissing {
		t.Fatalf("expected key_missing error, got %v", err)
	}
}

func TestDecryptTampered(t *testing.T) {
	os.Setenv(BusinessSecretKeyEnv, testKey)
	defer os.Unsetenv(BusinessSecretKeyEnv)

	enc, err := Encrypt("secret")
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	if _, err := Decrypt(enc + "x"); err == nil {
		t.Fatalf("expected tampered/decrypt error for corrupted ciphertext")
	}
}
