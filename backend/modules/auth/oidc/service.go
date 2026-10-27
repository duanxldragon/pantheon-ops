package oidc

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// oidcProviderName is the provider key stamped into every provisioned user.
// The (tenant_id, oidc_provider, oidc_subject) unique key (migration 000029)
// is only sound when lookups and writes agree on one provider identity.
const oidcProviderName = "default"

// Service handles OIDC authentication flow
type Service struct {
	db       *gorm.DB
	redis    *redis.Client
	provider *Provider
	config   *Config
}

// NewOIDCService creates a new OIDC service
func NewOIDCService(db *gorm.DB, redisClient *redis.Client, provider *Provider, config *Config) *Service {
	return &Service{
		db:       db,
		redis:    redisClient,
		provider: provider,
		config:   config,
	}
}

// InitiateLogin starts the OIDC login flow
func (s *Service) InitiateLogin(ctx context.Context, clientIP string) (authURL string, err error) {
	// Generate state and nonce
	state, err := generateRandomString(32)
	if err != nil {
		return "", fmt.Errorf("failed to generate state: %w", err)
	}

	nonce, err := generateRandomString(32)
	if err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Store state and nonce in Redis with 5-minute TTL
	key := "oidc:state:" + state
	err = s.redis.HSet(ctx, key, map[string]interface{}{
		"nonce": nonce,
		"ip":    clientIP,
	}).Err()
	if err != nil {
		return "", fmt.Errorf("failed to store state: %w", err)
	}

	err = s.redis.Expire(ctx, key, 5*time.Minute).Err()
	if err != nil {
		return "", fmt.Errorf("failed to set expiry: %w", err)
	}

	// Generate authorization URL
	authURL = s.provider.AuthCodeURL(state, nonce)
	return authURL, nil
}

// HandleCallback processes the OIDC callback. tenantID is the tenancy the
// login flow was started under, resolved by the handler from the request's
// tenant context (platform-global when no context is present); user lookup
// and provisioning are scoped to it so a subject from one tenant can never
// authenticate as the same-subject user of another.
func (s *Service) HandleCallback(ctx context.Context, code, state, clientIP string, tenantID uint64) (*CallbackResult, error) {
	// Validate state
	key := "oidc:state:" + state
	data, err := s.redis.HGetAll(ctx, key).Result()
	if err != nil || len(data) == 0 {
		return nil, errors.New("invalid or expired state")
	}

	nonce := data["nonce"]
	storedIP := data["ip"]

	// Optional: Verify IP hasn't changed. Intentionally a no-op for now:
	// users behind load balancers may rotate IPs mid-flow, so mismatches
	// must not fail the callback. A warning will be logged once a structured
	// logger is wired into this service.
	if storedIP != "" && storedIP != clientIP { //nolint:staticcheck // empty branch is deliberate scaffolding, see comment above
	}

	// Delete state (one-time use)
	s.redis.Del(ctx, key)

	// Exchange code for tokens
	token, err := s.provider.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code: %w", err)
	}

	// Get raw ID token
	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok {
		return nil, errors.New("no id_token in response")
	}

	// Verify ID token
	idToken, err := s.provider.VerifyIDToken(ctx, rawIDToken)
	if err != nil {
		return nil, fmt.Errorf("failed to verify ID token: %w", err)
	}

	// Verify nonce
	var claims struct {
		Nonce string `json:"nonce"`
	}
	if err := idToken.Claims(&claims); err != nil {
		return nil, fmt.Errorf("failed to parse claims: %w", err)
	}

	if claims.Nonce != nonce {
		return nil, errors.New("nonce mismatch")
	}

	// Get user info
	userInfo, err := s.provider.UserInfo(ctx, token)
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}

	// Provision or update user
	user, isNew, err := s.provisionUser(ctx, userInfo, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to provision user: %w", err)
	}

	return &CallbackResult{
		User:  user,
		IsNew: isNew,
	}, nil
}

// provisionUser creates or updates a user from OIDC claims, scoped to the
// tenant the login flow resolved. The subject predicate matches the
// (tenant_id, oidc_provider, oidc_subject) unique key from migration 000029;
// an unscoped lookup would let one provider subject authenticate as the same
// subject in every tenant.
func (s *Service) provisionUser(ctx context.Context, claims *UserInfo, tenantID uint64) (*User, bool, error) {
	var user User
	result := s.db.
		Where("tenant_id = ? AND oidc_provider = ? AND oidc_subject = ?", tenantID, oidcProviderName, claims.Subject).
		First(&user)

	if result.Error == gorm.ErrRecordNotFound {
		return s.provisionNewUser(ctx, claims, tenantID)
	}
	if result.Error != nil {
		return nil, false, result.Error
	}
	return s.syncExistingUser(user, claims)
}

// provisionNewUser creates a user from OIDC claims after validating the
// auto-provision policy (extracted to keep provisionUser complexity low).
func (s *Service) provisionNewUser(ctx context.Context, claims *UserInfo, tenantID uint64) (*User, bool, error) {
	if !s.config.AutoProvision {
		return nil, false, errors.New("user not found and auto-provision is disabled")
	}
	if !claims.EmailVerified {
		return nil, false, errors.New("email not verified by provider")
	}
	if err := s.validateAllowedDomain(claims.Email); err != nil {
		return nil, false, err
	}

	subject := claims.Subject
	provider := oidcProviderName
	user := User{
		TenantID:     tenantID,
		Username:     generateUsername(claims.Email),
		Email:        claims.Email,
		Nickname:     claims.Name,
		OIDCSubject:  &subject,
		OIDCProvider: &provider,
		AuthType:     "oidc",
		Status:       "active",
	}
	if err := s.db.Create(&user).Error; err != nil {
		return nil, false, fmt.Errorf("failed to create user: %w", err)
	}

	// Assign default role if configured. Intentionally deferred:
	// the assignment runs once the role system exposes an assignment API.
	if s.config.DefaultRole != "" { //nolint:staticcheck // empty branch is deliberate scaffolding, see comment above
	}

	return &user, true, nil
}

// validateAllowedDomain enforces the configured email-domain whitelist.
func (s *Service) validateAllowedDomain(email string) error {
	if len(s.config.AllowedDomains) == 0 {
		return nil
	}
	emailDomain := extractDomain(email)
	if !contains(s.config.AllowedDomains, emailDomain) {
		return fmt.Errorf("domain %s not allowed", emailDomain)
	}
	return nil
}

// syncExistingUser refreshes the stored profile from fresh OIDC claims.
func (s *Service) syncExistingUser(user User, claims *UserInfo) (*User, bool, error) {
	updated := false
	if user.Email != claims.Email {
		user.Email = claims.Email
		updated = true
	}
	if user.Nickname != claims.Name {
		user.Nickname = claims.Name
		updated = true
	}

	if updated {
		now := time.Now()
		user.OIDCLastSync = &now
		if err := s.db.Save(&user).Error; err != nil {
			return nil, false, fmt.Errorf("failed to update user: %w", err)
		}
	}

	return &user, false, nil
}

// CallbackResult represents the result of OIDC callback processing
type CallbackResult struct {
	User  *User
	IsNew bool
}

// User represents a simplified user model (should import from actual user
// package). It maps the real system_user table (migration 000029); the
// (tenant_id, oidc_provider, oidc_subject) composite unique mirrors
// uk_system_user_tenant_oidc_subject so a subject is only unique per
// provider within one tenant.
type User struct {
	ID           uint64     `gorm:"primaryKey"`
	TenantID     uint64     `gorm:"column:tenant_id;uniqueIndex:uk_system_user_tenant_oidc_subject,priority:1"`
	Username     string     `gorm:"uniqueIndex;not null"`
	Email        string     `gorm:"uniqueIndex;not null"`
	Nickname     string     `gorm:"not null"`
	Status       string     `gorm:"default:'active'"`
	OIDCSubject  *string    `gorm:"column:oidc_subject;uniqueIndex:uk_system_user_tenant_oidc_subject,priority:3"`
	OIDCProvider *string    `gorm:"column:oidc_provider;uniqueIndex:uk_system_user_tenant_oidc_subject,priority:2"`
	OIDCLastSync *time.Time `gorm:"column:oidc_last_sync"`
	AuthType     string     `gorm:"column:auth_type;default:'local'"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// TableName pins the model to the versioned-migration table; without it GORM
// would silently query a non-existent `users` table and every OIDC lookup
// would fail on any deployment created through the numbered migration chain.
func (User) TableName() string { return "system_user" }

// Helper functions

func generateRandomString(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes)[:length], nil
}

func extractDomain(email string) string {
	parts := strings.Split(email, "@")
	if len(parts) == 2 {
		return parts[1]
	}
	return ""
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func generateUsername(email string) string {
	// Use email prefix as username
	parts := strings.Split(email, "@")
	if len(parts) > 0 {
		return parts[0]
	}
	return email
}
