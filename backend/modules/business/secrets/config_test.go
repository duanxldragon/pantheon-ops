package secrets

import (
	"encoding/hex"
	"os"
	"testing"
)

func TestConfigSecretLifecycle(t *testing.T) {
	previous := os.Getenv(BusinessSecretKeyEnv)
	t.Cleanup(func() { _ = os.Setenv(BusinessSecretKeyEnv, previous) })
	_ = os.Setenv(BusinessSecretKeyEnv, hex.EncodeToString(make([]byte, 32)))

	encrypted, err := EncryptConfigFields(map[string]interface{}{
		"endpoint":        "https://storage.example",
		"secretAccessKey": "top-secret",
	}, "secretAccessKey")
	if err != nil {
		t.Fatalf("encrypt config: %v", err)
	}
	if encrypted["secretAccessKey"] == "top-secret" {
		t.Fatal("secret remained plaintext")
	}

	decrypted, err := DecryptConfigFields(encrypted, "secretAccessKey")
	if err != nil {
		t.Fatalf("decrypt config: %v", err)
	}
	if decrypted["secretAccessKey"] != "top-secret" {
		t.Fatalf("unexpected decrypted secret: %v", decrypted["secretAccessKey"])
	}

	redacted := RedactConfigFields(encrypted, "secretAccessKey")
	if _, exists := redacted["secretAccessKey"]; exists {
		t.Fatal("redacted config exposed secret field")
	}
	if redacted["hasSecret"] != true {
		t.Fatal("redacted config did not preserve secret presence metadata")
	}
}
