package host

import (
	"strings"
	"testing"

	"golang.org/x/crypto/ssh"
)

func TestNormalizeHostFingerprint(t *testing.T) {
	got := strings.TrimSpace(" SHA256:AbC  ")
	if got != "SHA256:AbC" {
		t.Fatalf("unexpected normalized fingerprint: %q", got)
	}
}

func TestHostKeyCallbackRejectsMismatch(t *testing.T) {
	callback := hostKeyCallback("sha256:expected")
	key, _, _, _, err := ssh.ParseAuthorizedKey([]byte("ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIHdF1m6M9tQGJm8a7F8x6J4xZpK6k3fYzQ6qzQe6Jf2p test"))
	if err != nil {
		t.Fatalf("parse key: %v", err)
	}
	if err := callback("host", nil, key); err == nil {
		t.Fatal("expected mismatch error")
	}
}

func TestHostKeyCallbackAcceptsMatchingFingerprint(t *testing.T) {
	key, _, _, _, err := ssh.ParseAuthorizedKey([]byte("ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIHdF1m6M9tQGJm8a7F8x6J4xZpK6k3fYzQ6qzQe6Jf2p test"))
	if err != nil {
		t.Fatalf("parse key: %v", err)
	}
	expected := strings.TrimSpace(ssh.FingerprintSHA256(key))
	callback := hostKeyCallback(expected)
	if err := callback("host", nil, key); err != nil {
		t.Fatalf("expected match to pass, got %v", err)
	}
}
