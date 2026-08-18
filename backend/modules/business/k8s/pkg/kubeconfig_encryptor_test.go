package pkg

import (
	"strings"
	"testing"
)

func validKey() string {
	return strings.Repeat("ab", 32) // 64 hex chars -> 32 bytes
}

func TestEncryptDecryptRoundTrip(t *testing.T) {
	t.Setenv(kubeconfigKeyEnv, validKey())
	plaintext := "apiVersion: v1\nkind: Config\nserver: https://127.0.0.1:6443\n"

	ciphertext, err := EncryptKubeconfig(plaintext)
	if err != nil {
		t.Fatalf("EncryptKubeconfig: %v", err)
	}
	if !strings.HasPrefix(ciphertext, "v1:") {
		t.Fatalf("expected versioned ciphertext, got %q", ciphertext)
	}

	got, err := DecryptKubeconfig(ciphertext)
	if err != nil {
		t.Fatalf("DecryptKubeconfig: %v", err)
	}
	if got != plaintext {
		t.Fatalf("round trip mismatch: got %q want %q", got, plaintext)
	}
}

func TestEncryptUsesRandomNonce(t *testing.T) {
	t.Setenv(kubeconfigKeyEnv, validKey())
	plaintext := "some kubeconfig payload"

	first, err := EncryptKubeconfig(plaintext)
	if err != nil {
		t.Fatalf("first encrypt: %v", err)
	}
	second, err := EncryptKubeconfig(plaintext)
	if err != nil {
		t.Fatalf("second encrypt: %v", err)
	}
	if first == second {
		t.Fatal("expected distinct ciphertexts due to random nonce")
	}
}

func TestDecryptRejectsWrongKey(t *testing.T) {
	t.Setenv(kubeconfigKeyEnv, validKey())
	ciphertext, err := EncryptKubeconfig("secret")
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}

	t.Setenv(kubeconfigKeyEnv, strings.Repeat("cd", 32))
	if _, err := DecryptKubeconfig(ciphertext); err != errCiphertextTampered {
		t.Fatalf("expected tampered error for wrong key, got %v", err)
	}
}

func TestDecryptRejectsMalformed(t *testing.T) {
	t.Setenv(kubeconfigKeyEnv, validKey())

	for _, input := range []string{"", "not-versioned", "v2:abcd", "v1:not-base64!!"} {
		if _, err := DecryptKubeconfig(input); err == nil {
			t.Fatalf("expected error for malformed input %q", input)
		}
	}
}

func TestEncryptRejectsMissingKey(t *testing.T) {
	t.Setenv(kubeconfigKeyEnv, "")
	if _, err := EncryptKubeconfig("secret"); err != errKubeconfigKeyMissing {
		t.Fatalf("expected missing key error, got %v", err)
	}
}

func TestEncryptRejectsInvalidKeyLength(t *testing.T) {
	t.Setenv(kubeconfigKeyEnv, "abcd") // 2 bytes, not 32
	if _, err := EncryptKubeconfig("secret"); err != errKubeconfigKeyInvalid {
		t.Fatalf("expected invalid key error, got %v", err)
	}
}
