package common

import (
	"fmt"
	"os"
	"strings"
)

const (
	DefaultAccessTokenSecret    = "pantheon-indigo-access-secret"
	DefaultRefreshTokenSecret   = "pantheon-indigo-refresh-secret"
	DefaultOperationTokenSecret = "pantheon-indigo-op-secret"
	DefaultSettingSecret        = "pantheon-setting-dev-secret-key!"
	DefaultMFASecret            = "pantheon-mfa-dev-secret-key!"
)

func IsProductionEnv() bool {
	for _, key := range []string{"PANTHEON_ENV", "APP_ENV", "GO_ENV"} {
		value := strings.ToLower(strings.TrimSpace(os.Getenv(key)))
		if value == "prod" || value == "production" {
			return true
		}
	}
	return strings.EqualFold(strings.TrimSpace(os.Getenv("GIN_MODE")), "release")
}

func ResolveSecret(name string, fallback string) string {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback
	}
	return value
}

func ValidateRequiredProductionSecret(name string, fallback string) error {
	if !IsProductionEnv() {
		return nil
	}
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" || value == fallback {
		return fmt.Errorf("%s must be explicitly configured in production", name)
	}
	return nil
}

func InitSecurityConfig() error {
	for _, item := range []struct {
		name     string
		fallback string
	}{
		{name: "PANTHEON_ACCESS_TOKEN_SECRET", fallback: DefaultAccessTokenSecret},
		{name: "PANTHEON_REFRESH_TOKEN_SECRET", fallback: DefaultRefreshTokenSecret},
		{name: "PANTHEON_OP_TOKEN_SECRET", fallback: DefaultOperationTokenSecret},
		{name: "PANTHEON_SETTING_SECRET", fallback: DefaultSettingSecret},
		{name: "PANTHEON_MFA_SECRET", fallback: DefaultMFASecret},
	} {
		if err := ValidateRequiredProductionSecret(item.name, item.fallback); err != nil {
			return err
		}
	}

	AccessTokenSecret = []byte(ResolveSecret("PANTHEON_ACCESS_TOKEN_SECRET", DefaultAccessTokenSecret))
	RefreshTokenSecret = []byte(ResolveSecret("PANTHEON_REFRESH_TOKEN_SECRET", DefaultRefreshTokenSecret))
	OperationTokenSecret = []byte(ResolveSecret("PANTHEON_OP_TOKEN_SECRET", DefaultOperationTokenSecret))
	return nil
}
