// Package secrets provides the shared AES-256-GCM helper used to encrypt
// sensitive business credentials at rest. It is the single canonical
// implementation for business modules (filedist, observability, backup);
// the deploy/kubeconfig/argocd encryptors predate this package and are
// scheduled to converge on it in a later cleanup.
package secrets

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"strings"
)

// BusinessSecretKeyEnv is the 32-byte hex key environment variable used to
// encrypt business credentials at rest.
const BusinessSecretKeyEnv = "PANTHEON_BUSINESS_SECRET_KEY"

const (
	errKeyMissing = "business.secrets.key_missing"
	errKeyInvalid = "business.secrets.key_invalid"
	errMalformed  = "business.secrets.malformed"
	errTampered   = "business.secrets.tampered"
)

// Encrypt encrypts plaintext with AES-256-GCM and returns a "v1:"-prefixed
// base64 value. Empty plaintext is returned unchanged (no ciphertext).
func Encrypt(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}
	key, err := key()
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	return "v1:" + base64.StdEncoding.EncodeToString(gcm.Seal(nonce, nonce, []byte(plaintext), nil)), nil
}

// Decrypt reverses Encrypt. Empty ciphertext is returned unchanged.
func Decrypt(ciphertext string) (string, error) {
	if ciphertext == "" {
		return "", nil
	}
	version, encoded, ok := strings.Cut(ciphertext, ":")
	if !ok || version != "v1" {
		return "", errors.New(errMalformed)
	}
	key, err := key()
	if err != nil {
		return "", err
	}
	raw, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", errors.New(errMalformed)
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	if len(raw) < gcm.NonceSize() {
		return "", errors.New(errMalformed)
	}
	plain, err := gcm.Open(nil, raw[:gcm.NonceSize()], raw[gcm.NonceSize():], nil)
	if err != nil {
		return "", errors.New(errTampered)
	}
	return string(plain), nil
}

func key() ([]byte, error) {
	raw := strings.TrimSpace(os.Getenv(BusinessSecretKeyEnv))
	if raw == "" {
		return nil, errors.New(errKeyMissing)
	}
	k, err := hex.DecodeString(raw)
	if err != nil || len(k) != 32 {
		return nil, errors.New(errKeyInvalid)
	}
	return k, nil
}
