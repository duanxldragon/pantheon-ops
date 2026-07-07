package security

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
)

// DefaultDevSecrets holds fallback secrets for development only.
// These are NEVER used in production — the process refuses to start
// if the corresponding environment variable is not set.
var DefaultDevSecrets = struct {
	AccessToken    string
	RefreshToken   string
	OperationToken string
	Setting        string
	MFA            string
}{
	AccessToken:    generateDevFallback("access"),
	RefreshToken:   generateDevFallback("refresh"),
	OperationToken: generateDevFallback("op"),
	Setting:        generateDevFallback("setting"),
	MFA:            generateDevFallback("mfa"),
}

// generateDevFallback creates a deterministic-but-unique dev secret
// derived from the label so different secrets are never interchangeable.
// It uses a fixed seed to keep tests reproducible — NOT for production.
func generateDevFallback(label string) string {
	// Use a fixed 32-byte hex pattern derived from label for dev convenience.
	// In production this is never reached; the process exits if env var is missing.
	return "dev-only-" + label + "-" + strings.Repeat("x", 32-len(label)-10)
}

func IsProductionEnv() bool {
	for _, key := range []string{"PANTHEON_ENV", "APP_ENV", "GO_ENV"} {
		value := strings.ToLower(strings.TrimSpace(os.Getenv(key)))
		if value == "prod" || value == "production" {
			return true
		}
	}
	return strings.EqualFold(strings.TrimSpace(os.Getenv("GIN_MODE")), "release")
}

// ResolveSecret returns the env var value if set, otherwise the fallback.
func ResolveSecret(name, fallback string) string {
	value := strings.TrimSpace(os.Getenv(name))
	if value != "" {
		return value
	}
	return fallback
}

// ValidateRequiredProductionSecret returns an error if running in production
// and the secret is either missing or still at its default fallback value.
func ValidateRequiredProductionSecret(name, fallback string) error {
	if !IsProductionEnv() {
		return nil
	}
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" || value == fallback {
		return fmt.Errorf("%s must be explicitly configured in production", name)
	}
	if len(value) < 32 {
		return fmt.Errorf("%s must be at least 32 bytes in production (got %d)", name, len(value))
	}
	return nil
}

// GenerateRandomSecret creates a cryptographically random hex secret
// of the specified byte length (the hex string will be 2*byteLen characters).
func GenerateRandomSecret(byteLen int) (string, error) {
	b := make([]byte, byteLen)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate random secret: %w", err)
	}
	return hex.EncodeToString(b), nil
}

func InitSecurityConfig() error {
	for _, item := range []struct {
		name     string
		fallback string
	}{
		{name: "PANTHEON_ACCESS_TOKEN_SECRET", fallback: DefaultDevSecrets.AccessToken},
		{name: "PANTHEON_REFRESH_TOKEN_SECRET", fallback: DefaultDevSecrets.RefreshToken},
		{name: "PANTHEON_OP_TOKEN_SECRET", fallback: DefaultDevSecrets.OperationToken},
		{name: "PANTHEON_SETTING_SECRET", fallback: DefaultDevSecrets.Setting},
		{name: "PANTHEON_MFA_SECRET", fallback: DefaultDevSecrets.MFA},
	} {
		if err := ValidateRequiredProductionSecret(item.name, item.fallback); err != nil {
			return err
		}
	}

	return nil
}
