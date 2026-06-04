package auth

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
	"strings"

	"pantheon-platform/backend/pkg/common"
)

const encryptedMFASecretPrefix = "mfa:v1:"

func encryptMFASecret(plain string) (string, error) {
	trimmed := strings.TrimSpace(plain)
	if trimmed == "" {
		return "", nil
	}
	block, err := aes.NewCipher(getMFACipherKey())
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
	cipherText := gcm.Seal(nonce, nonce, []byte(trimmed), nil)
	return encryptedMFASecretPrefix + base64.StdEncoding.EncodeToString(cipherText), nil
}

func decryptMFASecret(value string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", nil
	}
	if !strings.HasPrefix(trimmed, encryptedMFASecretPrefix) {
		return trimmed, nil
	}
	raw, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(trimmed, encryptedMFASecretPrefix))
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(getMFACipherKey())
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	if len(raw) < gcm.NonceSize() {
		return "", errors.New("auth.mfa.secret.invalid")
	}
	plain, err := gcm.Open(nil, raw[:gcm.NonceSize()], raw[gcm.NonceSize():], nil)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}

func getMFACipherKey() []byte {
	value := common.ResolveSecret("PANTHEON_MFA_SECRET", common.DefaultMFASecret)
	key := []byte(value)
	if len(key) >= 32 {
		return key[:32]
	}
	padded := make([]byte, 32)
	copy(padded, key)
	return padded
}
