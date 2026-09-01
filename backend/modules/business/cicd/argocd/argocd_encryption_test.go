package argocd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEncryptDecryptSecret(t *testing.T) {
	// Set up test encryption key (32 bytes hex-encoded = 64 characters)
	t.Setenv("PANTHEON_ENCRYPTION_KEY", "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef")

	tests := []struct {
		name      string
		plaintext string
	}{
		{
			name:      "normal token",
			plaintext: "test-auth-token-12345",
		},
		{
			name:      "empty string",
			plaintext: "",
		},
		{
			name:      "long token",
			plaintext: "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJhcmdvY2QiLCJzdWIiOiJhZG1pbiIsImlhdCI6MTYxMjM0NTY3OH0.abcdef",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Encrypt
			encrypted, err := EncryptSecret(tt.plaintext)
			require.NoError(t, err)

			if tt.plaintext == "" {
				assert.Empty(t, encrypted)
				return
			}

			// Should be base64 encoded with version prefix
			assert.Contains(t, encrypted, "v1:")
			assert.NotEqual(t, tt.plaintext, encrypted)

			// Decrypt
			decrypted, err := DecryptSecret(encrypted)
			require.NoError(t, err)
			assert.Equal(t, tt.plaintext, decrypted)
		})
	}
}

func TestEncryptSecretWithoutKey(t *testing.T) {
	// Unset encryption key
	t.Setenv("PANTHEON_ENCRYPTION_KEY", "")
	t.Setenv("PANTHEON_ARGOCD_ENCRYPTION_KEY", "")

	_, err := EncryptSecret("test")
	assert.Error(t, err)
	assert.Equal(t, errEncryptionKeyMissing, err)
}

func TestDecryptInvalidCiphertext(t *testing.T) {
	t.Setenv("PANTHEON_ENCRYPTION_KEY", "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef")

	tests := []struct {
		name       string
		ciphertext string
		wantErr    error
	}{
		{
			name:       "no version prefix",
			ciphertext: "invalid-ciphertext",
			wantErr:    errCiphertextMalformed,
		},
		{
			name:       "wrong version",
			ciphertext: "v2:abcdef",
			wantErr:    errCiphertextMalformed,
		},
		{
			name:       "invalid base64",
			ciphertext: "v1:!!!invalid!!!",
			wantErr:    errCiphertextMalformed,
		},
		{
			name:       "tampered data",
			ciphertext: "v1:YWJjZGVm", // valid base64 but wrong data
			wantErr:    errCiphertextMalformed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := DecryptSecret(tt.ciphertext)
			assert.Error(t, err)
		})
	}
}

func TestEncryptSecretDeterminism(t *testing.T) {
	t.Setenv("PANTHEON_ENCRYPTION_KEY", "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef")

	plaintext := "test-token"

	// Encrypt twice
	encrypted1, err1 := EncryptSecret(plaintext)
	require.NoError(t, err1)

	encrypted2, err2 := EncryptSecret(plaintext)
	require.NoError(t, err2)

	// Should be different (different nonces)
	assert.NotEqual(t, encrypted1, encrypted2)

	// But both should decrypt to the same plaintext
	decrypted1, err := DecryptSecret(encrypted1)
	require.NoError(t, err)
	assert.Equal(t, plaintext, decrypted1)

	decrypted2, err := DecryptSecret(encrypted2)
	require.NoError(t, err)
	assert.Equal(t, plaintext, decrypted2)
}
