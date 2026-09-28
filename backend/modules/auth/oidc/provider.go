package oidc

import (
	"context"
	"fmt"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

// Provider wraps the OIDC provider and OAuth2 config
type Provider struct {
	provider     *oidc.Provider
	verifier     *oidc.IDTokenVerifier
	oauth2Config oauth2.Config
}

// NewOIDCProvider creates a new OIDC provider instance
func NewOIDCProvider(ctx context.Context, cfg *Config) (*Provider, error) {
	if !cfg.Enabled {
		return nil, fmt.Errorf("OIDC is not enabled")
	}

	// Discover OIDC provider
	provider, err := oidc.NewProvider(ctx, cfg.ProviderURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create OIDC provider: %w", err)
	}

	// Create ID token verifier
	verifier := provider.Verifier(&oidc.Config{
		ClientID: cfg.ClientID,
	})

	// Create OAuth2 config
	oauth2Config := oauth2.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		Endpoint:     provider.Endpoint(),
		RedirectURL:  cfg.RedirectURL,
		Scopes:       cfg.Scopes,
	}

	return &Provider{
		provider:     provider,
		verifier:     verifier,
		oauth2Config: oauth2Config,
	}, nil
}

// AuthCodeURL generates the authorization URL with state and nonce
func (p *Provider) AuthCodeURL(state, nonce string) string {
	return p.oauth2Config.AuthCodeURL(state,
		oauth2.SetAuthURLParam("nonce", nonce))
}

// Exchange exchanges the authorization code for tokens
func (p *Provider) Exchange(ctx context.Context, code string) (*oauth2.Token, error) {
	return p.oauth2Config.Exchange(ctx, code)
}

// VerifyIDToken verifies the ID token signature and claims
func (p *Provider) VerifyIDToken(ctx context.Context, rawIDToken string) (*oidc.IDToken, error) {
	return p.verifier.Verify(ctx, rawIDToken)
}

// UserInfo fetches user information from the UserInfo endpoint
func (p *Provider) UserInfo(ctx context.Context, token *oauth2.Token) (*UserInfo, error) {
	userInfo, err := p.provider.UserInfo(ctx, oauth2.StaticTokenSource(token))
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}

	var claims UserInfo
	if err := userInfo.Claims(&claims); err != nil {
		return nil, fmt.Errorf("failed to parse user info claims: %w", err)
	}

	return &claims, nil
}

// UserInfo represents user information from OIDC provider
type UserInfo struct {
	Subject       string `json:"sub"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	Name          string `json:"name"`
	GivenName     string `json:"given_name"`
	FamilyName    string `json:"family_name"`
	Picture       string `json:"picture"`
	Locale        string `json:"locale"`
}
