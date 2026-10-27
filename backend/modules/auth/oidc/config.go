package oidc

import (
	"fmt"
	"os"
	"strings"
)

// Config holds OIDC provider configuration
type Config struct {
	Enabled        bool
	ProviderURL    string
	ClientID       string
	ClientSecret   string
	RedirectURL    string
	Scopes         []string
	AutoProvision  bool
	AllowedDomains []string
	DefaultRole    string
}

// LoadOIDCConfig loads OIDC configuration from environment variables
func LoadOIDCConfig() (*Config, error) {
	enabled := strings.ToLower(os.Getenv("PANTHEON_OIDC_ENABLED")) == "true"

	if !enabled {
		return &Config{Enabled: false}, nil
	}

	// Required fields
	providerURL := os.Getenv("PANTHEON_OIDC_PROVIDER_URL")
	clientID := os.Getenv("PANTHEON_OIDC_CLIENT_ID")
	clientSecret := os.Getenv("PANTHEON_OIDC_CLIENT_SECRET")
	redirectURL := os.Getenv("PANTHEON_OIDC_REDIRECT_URL")

	if providerURL == "" || clientID == "" || clientSecret == "" || redirectURL == "" {
		return nil, fmt.Errorf("OIDC enabled but missing required config: PROVIDER_URL, CLIENT_ID, CLIENT_SECRET, or REDIRECT_URL")
	}

	// Optional fields
	scopesStr := os.Getenv("PANTHEON_OIDC_SCOPES")
	if scopesStr == "" {
		scopesStr = "openid,profile,email"
	}
	scopes := strings.Split(scopesStr, ",")
	for i := range scopes {
		scopes[i] = strings.TrimSpace(scopes[i])
	}

	autoProvision := strings.ToLower(os.Getenv("PANTHEON_OIDC_AUTO_PROVISION")) != "false" // default true

	allowedDomainsStr := os.Getenv("PANTHEON_OIDC_ALLOWED_DOMAINS")
	var allowedDomains []string
	if allowedDomainsStr != "" {
		allowedDomains = strings.Split(allowedDomainsStr, ",")
		for i := range allowedDomains {
			allowedDomains[i] = strings.TrimSpace(allowedDomains[i])
		}
	}

	defaultRole := os.Getenv("PANTHEON_OIDC_DEFAULT_ROLE")
	if defaultRole == "" {
		defaultRole = "user"
	}

	return &Config{
		Enabled:        true,
		ProviderURL:    providerURL,
		ClientID:       clientID,
		ClientSecret:   clientSecret,
		RedirectURL:    redirectURL,
		Scopes:         scopes,
		AutoProvision:  autoProvision,
		AllowedDomains: allowedDomains,
		DefaultRole:    defaultRole,
	}, nil
}
