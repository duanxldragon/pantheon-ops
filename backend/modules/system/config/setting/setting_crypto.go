package config

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
	"pantheon-ops/backend/pkg/common"
	"strings"
)

const encryptedSettingPrefix = "enc:v1:"

func encryptSettingValue(plain string) (string, error) {
	if strings.TrimSpace(plain) == "" {
		return "", nil
	}

	block, err := aes.NewCipher(getSettingCipherKey())
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

	cipherText := gcm.Seal(nonce, nonce, []byte(plain), nil)
	return encryptedSettingPrefix + base64.StdEncoding.EncodeToString(cipherText), nil
}

func decryptSettingValue(value string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", nil
	}
	if !strings.HasPrefix(trimmed, encryptedSettingPrefix) {
		return trimmed, nil
	}

	encoded := strings.TrimPrefix(trimmed, encryptedSettingPrefix)
	raw, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(getSettingCipherKey())
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	if len(raw) < gcm.NonceSize() {
		return "", errors.New("setting.decrypt.invalid")
	}

	nonce, cipherText := raw[:gcm.NonceSize()], raw[gcm.NonceSize():]
	plain, err := gcm.Open(nil, nonce, cipherText, nil)
	if err != nil {
		return "", err
	}

	return string(plain), nil
}

func getSettingCipherKey() []byte {
	value := common.ResolveSecret("PANTHEON_SETTING_SECRET", common.DefaultSettingSecret)

	key := []byte(value)
	switch {
	case len(key) >= 32:
		return key[:32]
	default:
		padded := make([]byte, 32)
		copy(padded, key)
		return padded
	}
}
