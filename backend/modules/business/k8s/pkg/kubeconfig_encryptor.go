// Package pkg provides shared utilities for the k8s business module.
package pkg

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

const (
	// kubeconfigCipherVersion prefixes the ciphertext so future format changes
	// can coexist without breaking existing stored credentials.
	kubeconfigCipherVersion = "v1"

	// kubeconfigKeyEnv is the environment variable holding the 32-byte
	// (64 hex characters) AES key used to encrypt stored kubeconfigs.
	kubeconfigKeyEnv = "PANTHEON_K8S_KUBECONFIG_KEY"
)

var (
	errKubeconfigKeyMissing = errors.New("k8s.kubeconfig_key_missing")
	errKubeconfigKeyInvalid = errors.New("k8s.kubeconfig_key_invalid")
	errCiphertextMalformed  = errors.New("k8s.kubeconfig_ciphertext_malformed")
	errCiphertextTampered   = errors.New("k8s.kubeconfig_ciphertext_tampered")
)

// EncryptKubeconfig encrypts a plaintext kubeconfig using AES-256-GCM with a
// random nonce. The result is a versioned base64 string that embeds the nonce
// and ciphertext, suitable for storage in a single database column.
func EncryptKubeconfig(plaintext string) (string, error) {
	key, err := loadKey()
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", errKubeconfigKeyInvalid
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", errKubeconfigKeyInvalid
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	sealed := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return kubeconfigCipherVersion + ":" + base64.StdEncoding.EncodeToString(sealed), nil
}

// DecryptKubeconfig reverses EncryptKubeconfig. It returns a descriptive error
// for malformed input, an invalid key, or a ciphertext that fails GCM
// authentication (tampered data).
func DecryptKubeconfig(ciphertext string) (string, error) {
	if ciphertext == "" {
		return "", errCiphertextMalformed
	}
	version, payload, ok := strings.Cut(ciphertext, ":")
	if !ok || version != kubeconfigCipherVersion {
		return "", errCiphertextMalformed
	}
	raw, err := base64.StdEncoding.DecodeString(payload)
	if err != nil {
		return "", errCiphertextMalformed
	}

	key, err := loadKey()
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", errKubeconfigKeyInvalid
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", errKubeconfigKeyInvalid
	}
	if len(raw) < gcm.NonceSize() {
		return "", errCiphertextMalformed
	}

	nonce, cipher := raw[:gcm.NonceSize()], raw[gcm.NonceSize():]
	plain, err := gcm.Open(nil, nonce, cipher, nil)
	if err != nil {
		return "", errCiphertextTampered
	}
	return string(plain), nil
}

func loadKey() ([]byte, error) {
	raw := strings.TrimSpace(os.Getenv(kubeconfigKeyEnv))
	if raw == "" {
		return nil, errKubeconfigKeyMissing
	}
	key, err := hex.DecodeString(raw)
	if err != nil || len(key) != 32 {
		return nil, errKubeconfigKeyInvalid
	}
	return key, nil
}
