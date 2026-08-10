package generator

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"io"
	"strings"

	"pantheon-ops/backend/pkg/common"
	commonsecurity "pantheon-ops/backend/pkg/common/security"
)

const generatorEncryptedPrefix = "enc:v1:"

func encryptDatasourcePassword(plain string) (string, error) {
	if strings.TrimSpace(plain) == "" {
		return "", nil
	}

	block, err := aes.NewCipher(getDatasourceCipherKey())
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
	return generatorEncryptedPrefix + base64.StdEncoding.EncodeToString(cipherText), nil
}

func decryptDatasourcePassword(value string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", nil
	}
	if !strings.HasPrefix(trimmed, generatorEncryptedPrefix) {
		return trimmed, nil
	}

	raw, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(trimmed, generatorEncryptedPrefix))
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(getDatasourceCipherKey())
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	if len(raw) < gcm.NonceSize() {
		return "", common.NewBadRequest("generator.datasource.decrypt.invalid")
	}

	nonce, cipherText := raw[:gcm.NonceSize()], raw[gcm.NonceSize():]
	plain, err := gcm.Open(nil, nonce, cipherText, nil)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}

func getDatasourceCipherKey() []byte {
	value := commonsecurity.ResolveSecret("PANTHEON_GENERATOR_DATASOURCE_SECRET", "")
	if strings.TrimSpace(value) == "" {
		value = commonsecurity.ResolveSecret("PANTHEON_SETTING_SECRET", commonsecurity.DefaultDevSecrets.Setting)
	}

	key := []byte(value)
	if len(key) >= 32 {
		return key[:32]
	}
	padded := make([]byte, 32)
	copy(padded, key)
	return padded
}
