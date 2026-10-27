package argocd

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
	"os"
	"strings"
)

const (
	argoCDEncryptionVersion = "v1"
	encryptionKeyEnv        = "PANTHEON_ARGOCD_ENCRYPTION_KEY"
)

var (
	errEncryptionKeyMissing = errors.New("argocd.encryption_key_missing")
	errEncryptionKeyInvalid = errors.New("argocd.encryption_key_invalid")
	errCiphertextMalformed  = errors.New("argocd.ciphertext_malformed")
	errCiphertextTampered   = errors.New("argocd.ciphertext_tampered")
)

// EncryptSecret encrypts sensitive data using AES-256-GCM.
func EncryptSecret(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}

	key, err := loadEncryptionKey()
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", errEncryptionKeyInvalid
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", errEncryptionKeyInvalid
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	sealed := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return argoCDEncryptionVersion + ":" + base64.StdEncoding.EncodeToString(sealed), nil
}

// DecryptSecret decrypts sensitive data.
func DecryptSecret(ciphertext string) (string, error) {
	if ciphertext == "" {
		return "", nil
	}

	version, payload, ok := strings.Cut(ciphertext, ":")
	if !ok || version != argoCDEncryptionVersion {
		return "", errCiphertextMalformed
	}

	raw, err := base64.StdEncoding.DecodeString(payload)
	if err != nil {
		return "", errCiphertextMalformed
	}

	key, err := loadEncryptionKey()
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", errEncryptionKeyInvalid
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", errEncryptionKeyInvalid
	}

	if len(raw) < gcm.NonceSize() {
		return "", errCiphertextMalformed
	}

	nonce, cipherData := raw[:gcm.NonceSize()], raw[gcm.NonceSize():]
	plain, err := gcm.Open(nil, nonce, cipherData, nil)
	if err != nil {
		return "", errCiphertextTampered
	}

	return string(plain), nil
}

func loadEncryptionKey() ([]byte, error) {
	// Try ArgoCD specific key first
	raw := strings.TrimSpace(os.Getenv(encryptionKeyEnv))
	if raw == "" {
		// Fall back to general encryption key
		raw = strings.TrimSpace(os.Getenv("PANTHEON_ENCRYPTION_KEY"))
	}
	if raw == "" {
		return nil, errEncryptionKeyMissing
	}

	// If exactly 32 bytes, use as-is
	if len(raw) == 32 {
		return []byte(raw), nil
	}

	// Try base64 decode
	key, err := base64.StdEncoding.DecodeString(raw)
	if err == nil && len(key) == 32 {
		return key, nil
	}

	// If 64 characters, might be hex
	if len(raw) == 64 {
		// Use first 32 bytes as key
		return []byte(raw[:32]), nil
	}

	return nil, errEncryptionKeyInvalid
}
