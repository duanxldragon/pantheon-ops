---
title: SSO/OIDC Design Document
doc_type: Design
layer: system/auth
status: Draft
linked_contracts:
  - docs/contracts/SYSTEM_AUTH_CONTRACT.md
updated_at: 2026-09-08
---

# SSO/OIDC Design Document

**Version**: 1.0  
**Date**: 2026-09-08  
**Status**: Design Phase  
**Target**: Pantheon Base Enterprise Authentication

---

## Executive Summary

This document defines the architecture and implementation strategy for integrating OpenID Connect (OIDC) Single Sign-On (SSO) into Pantheon Base. The design enables enterprise customers to authenticate users via their existing identity providers (Auth0, Okta, Azure AD, Google Workspace, etc.) while maintaining Pantheon's existing session and permission model.

**Key Goals**:
- Enable enterprise identity source integration
- Support Just-In-Time (JIT) user provisioning
- Maintain existing security model
- Minimize breaking changes to current authentication

---

## Architecture Overview

### High-Level Architecture

```
┌──────────────────────────────────────────────────────────────┐
│                        User Browser                           │
└───┬──────────────────────────────────────────────────────┬───┘
    │                                                        │
    │ (1) Initiate SSO Login                                │ (6) Session Cookie
    │                                                        │
┌───▼────────────────────────────────────────────────────────▼───┐
│                     Pantheon Backend                           │
│  ┌──────────────────────────────────────────────────────────┐ │
│  │  /api/v1/auth/oidc/login                                 │ │
│  │  - Generate state & nonce                                │ │
│  │  - Store in Redis (5 min TTL)                            │ │
│  │  - Redirect to IdP authorize endpoint                    │ │
│  └──────────────────────────────────────────────────────────┘ │
│  ┌──────────────────────────────────────────────────────────┐ │
│  │  /api/v1/auth/oidc/callback                              │ │
│  │  - Validate state & nonce                                │ │
│  │  - Exchange auth code for tokens                         │ │
│  │  - Verify ID token signature                             │ │
│  │  - Extract user claims                                   │ │
│  │  - Provision or update user                              │ │
│  │  - Create Pantheon session                               │ │
│  └──────────────────────────────────────────────────────────┘ │
└────────────────┬────────────────────────────────┬──────────────┘
                 │                                │
                 │ (2) Authorization Request      │ (4) Token Exchange
                 │ (3) Auth Code                  │ (5) UserInfo
                 │                                │
         ┌───────▼────────────────────────────────▼──────────┐
         │         OIDC Identity Provider                     │
         │  (Auth0, Okta, Azure AD, Keycloak, etc.)          │
         └────────────────────────────────────────────────────┘
```

### Authentication Flow (Authorization Code Flow)

```
User          Browser         Pantheon           Redis          OIDC Provider
 │               │                │                │                │
 │ Click SSO     │                │                │                │
 ├──────────────>│                │                │                │
 │               │ GET /oidc/login│                │                │
 │               ├───────────────>│                │                │
 │               │                │ Generate state │                │
 │               │                │   & nonce      │                │
 │               │                ├───────────────>│                │
 │               │                │ Store 5min TTL │                │
 │               │                │<───────────────│                │
 │               │ 302 Redirect   │                │                │
 │               │<───────────────│                │                │
 │               │                │                │                │
 │               │ GET /authorize?client_id=...&state=...&nonce=...│
 │               ├───────────────────────────────────────────────>│
 │               │                │                │                │
 │ [User logs in via IdP]         │                │                │
 │               │                │                │                │
 │               │ 302 /callback?code=XXX&state=YYY                │
 │               │<───────────────────────────────────────────────│
 │               │                │                │                │
 │               │ GET /oidc/callback?code=XXX&state=YYY           │
 │               ├───────────────>│                │                │
 │               │                │ Validate state │                │
 │               │                ├───────────────>│                │
 │               │                │<───────────────│                │
 │               │                │ POST /token    │                │
 │               │                ├───────────────────────────────>│
 │               │                │ access_token   │                │
 │               │                │ id_token       │                │
 │               │                │ refresh_token  │                │
 │               │                │<───────────────────────────────│
 │               │                │                │                │
 │               │                │ Verify ID token│                │
 │               │                │ (signature,    │                │
 │               │                │  issuer, aud,  │                │
 │               │                │  exp, nonce)   │                │
 │               │                │                │                │
 │               │                │ GET /userinfo  │                │
 │               │                ├───────────────────────────────>│
 │               │                │ {sub, email,   │                │
 │               │                │  name, ...}    │                │
 │               │                │<───────────────────────────────│
 │               │                │                │                │
 │               │                │ Provision User │                │
 │               │                │ Create Session │                │
 │               │                │                │                │
 │               │ 302 /dashboard │                │                │
 │               │ Set-Cookie     │                │                │
 │               │<───────────────│                │                │
 │               │                │                │                │
 │ Dashboard     │                │                │                │
 │<──────────────│                │                │                │
```

---

## Component Design

### 1. OIDC Configuration

**Environment Variables**:
```bash
# Required
PANTHEON_OIDC_ENABLED=true
PANTHEON_OIDC_PROVIDER_URL=https://accounts.example.com
PANTHEON_OIDC_CLIENT_ID=pantheon-prod-client
PANTHEON_OIDC_CLIENT_SECRET=super-secret-value
PANTHEON_OIDC_REDIRECT_URL=https://pantheon.example.com/api/v1/auth/oidc/callback

# Optional
PANTHEON_OIDC_SCOPES=openid,profile,email
PANTHEON_OIDC_AUTO_PROVISION=true
PANTHEON_OIDC_ALLOWED_DOMAINS=example.com,partner.com
PANTHEON_OIDC_DEFAULT_ROLE=user
```

**Configuration Structure** (`backend/modules/auth/oidc/config.go`):
```go
type OIDCConfig struct {
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
```

### 2. Database Schema Changes

**Migration**: `YYYYMMDDHHMMSS_add_oidc_fields.sql`

```sql
-- Add OIDC fields to users table
ALTER TABLE users 
ADD COLUMN oidc_subject VARCHAR(255) UNIQUE COMMENT 'OIDC sub claim (unique user ID from provider)',
ADD COLUMN oidc_provider VARCHAR(100) COMMENT 'OIDC provider identifier',
ADD COLUMN oidc_last_sync DATETIME COMMENT 'Last time profile was synced from provider',
ADD COLUMN auth_type VARCHAR(20) DEFAULT 'local' COMMENT 'Authentication type: local or oidc';

-- Index for OIDC subject lookup
CREATE INDEX idx_users_oidc_subject ON users(oidc_subject);
CREATE INDEX idx_users_auth_type ON users(auth_type);

-- Optional: Add provider config table for multi-provider support (future)
-- CREATE TABLE oidc_providers (...);
```

**User Model Update** (`backend/modules/system/iam/user/models.go`):
```go
type User struct {
    // ... existing fields
    OIDCSubject    *string    `gorm:"uniqueIndex;column:oidc_subject" json:"oidc_subject,omitempty"`
    OIDCProvider   *string    `gorm:"column:oidc_provider" json:"oidc_provider,omitempty"`
    OIDCLastSync   *time.Time `gorm:"column:oidc_last_sync" json:"oidc_last_sync,omitempty"`
    AuthType       string     `gorm:"default:'local'" json:"auth_type"` // 'local' or 'oidc'
}
```

### 3. OIDC Provider Wrapper

**File**: `backend/modules/auth/oidc/provider.go`

```go
package oidc

import (
    "context"
    "github.com/coreos/go-oidc/v3/oidc"
    "golang.org/x/oauth2"
)

type OIDCProvider struct {
    provider *oidc.Provider
    verifier *oidc.IDTokenVerifier
    oauth2Config oauth2.Config
}

func NewOIDCProvider(ctx context.Context, cfg *OIDCConfig) (*OIDCProvider, error) {
    provider, err := oidc.NewProvider(ctx, cfg.ProviderURL)
    if err != nil {
        return nil, err
    }

    verifier := provider.Verifier(&oidc.Config{
        ClientID: cfg.ClientID,
    })

    oauth2Config := oauth2.Config{
        ClientID:     cfg.ClientID,
        ClientSecret: cfg.ClientSecret,
        Endpoint:     provider.Endpoint(),
        RedirectURL:  cfg.RedirectURL,
        Scopes:       cfg.Scopes,
    }

    return &OIDCProvider{
        provider:     provider,
        verifier:     verifier,
        oauth2Config: oauth2Config,
    }, nil
}

// AuthCodeURL generates the authorization URL with state and nonce
func (p *OIDCProvider) AuthCodeURL(state, nonce string) string {
    return p.oauth2Config.AuthCodeURL(state,
        oauth2.SetAuthURLParam("nonce", nonce))
}

// Exchange exchanges authorization code for tokens
func (p *OIDCProvider) Exchange(ctx context.Context, code string) (*oauth2.Token, error) {
    return p.oauth2Config.Exchange(ctx, code)
}

// VerifyIDToken verifies the ID token signature and claims
func (p *OIDCProvider) VerifyIDToken(ctx context.Context, rawIDToken string) (*oidc.IDToken, error) {
    return p.verifier.Verify(ctx, rawIDToken)
}

// UserInfo fetches user information from the UserInfo endpoint
func (p *OIDCProvider) UserInfo(ctx context.Context, token *oauth2.Token) (*OIDCUserInfo, error) {
    userInfo, err := p.provider.UserInfo(ctx, oauth2.StaticTokenSource(token))
    if err != nil {
        return nil, err
    }

    var claims OIDCUserInfo
    if err := userInfo.Claims(&claims); err != nil {
        return nil, err
    }

    return &claims, nil
}

type OIDCUserInfo struct {
    Subject       string `json:"sub"`
    Email         string `json:"email"`
    EmailVerified bool   `json:"email_verified"`
    Name          string `json:"name"`
    GivenName     string `json:"given_name"`
    FamilyName    string `json:"family_name"`
    Picture       string `json:"picture"`
}
```

### 4. OIDC Service Layer

**File**: `backend/modules/auth/oidc/service.go`

```go
package oidc

type OIDCService struct {
    db       *gorm.DB
    provider *OIDCProvider
    userRepo *user.Repository
    redis    *redis.Client
}

// InitiateLogin starts the OIDC login flow
func (s *OIDCService) InitiateLogin(c *gin.Context) (authURL string, err error) {
    state := generateRandomString(32)
    nonce := generateRandomString(32)

    // Store state and nonce in Redis with 5-minute TTL
    key := "oidc:state:" + state
    data := map[string]string{
        "nonce": nonce,
        "ip":    c.ClientIP(),
    }
    if err := s.redis.HSet(c, key, data).Err(); err != nil {
        return "", err
    }
    if err := s.redis.Expire(c, key, 5*time.Minute).Err(); err != nil {
        return "", err
    }

    authURL = s.provider.AuthCodeURL(state, nonce)
    return authURL, nil
}

// HandleCallback processes the OIDC callback
func (s *OIDCService) HandleCallback(c *gin.Context, code, state string) (*models.User, error) {
    // 1. Validate state
    key := "oidc:state:" + state
    data, err := s.redis.HGetAll(c, key).Result()
    if err != nil || len(data) == 0 {
        return nil, errors.New("invalid or expired state")
    }
    nonce := data["nonce"]
    s.redis.Del(c, key) // One-time use

    // 2. Exchange code for tokens
    token, err := s.provider.Exchange(c, code)
    if err != nil {
        return nil, err
    }

    // 3. Verify ID token
    rawIDToken, ok := token.Extra("id_token").(string)
    if !ok {
        return nil, errors.New("no id_token in response")
    }

    idToken, err := s.provider.VerifyIDToken(c, rawIDToken)
    if err != nil {
        return nil, err
    }

    // 4. Verify nonce
    var claims struct {
        Nonce string `json:"nonce"`
    }
    if err := idToken.Claims(&claims); err != nil {
        return nil, err
    }
    if claims.Nonce != nonce {
        return nil, errors.New("nonce mismatch")
    }

    // 5. Get user info
    userInfo, err := s.provider.UserInfo(c, token)
    if err != nil {
        return nil, err
    }

    // 6. Provision or update user
    user, err := s.ProvisionUser(c, userInfo)
    if err != nil {
        return nil, err
    }

    return user, nil
}

// ProvisionUser creates or updates a user from OIDC claims
func (s *OIDCService) ProvisionUser(ctx context.Context, claims *OIDCUserInfo) (*models.User, error) {
    // Check if user exists by OIDC subject
    var user models.User
    result := s.db.Where("oidc_subject = ?", claims.Subject).First(&user)

    if result.Error == gorm.ErrRecordNotFound {
        // New user - check auto-provision settings
        if !s.config.AutoProvision {
            return nil, errors.New("user not found and auto-provision disabled")
        }

        // Check domain whitelist
        if len(s.config.AllowedDomains) > 0 {
            emailDomain := extractDomain(claims.Email)
            if !contains(s.config.AllowedDomains, emailDomain) {
                return nil, fmt.Errorf("domain %s not allowed", emailDomain)
            }
        }

        // Create new user
        user = models.User{
            Username:     generateUsername(claims.Email),
            Email:        claims.Email,
            Nickname:     claims.Name,
            OIDCSubject:  &claims.Subject,
            OIDCProvider: stringPtr("default"),
            AuthType:     "oidc",
            Status:       "active", // or "pending_approval"
        }

        if err := s.db.Create(&user).Error; err != nil {
            return nil, err
        }

        // Assign default role if configured
        if s.config.DefaultRole != "" {
            // Assign role logic here
        }

    } else if result.Error != nil {
        return nil, result.Error
    } else {
        // Existing user - sync profile
        user.Email = claims.Email
        user.Nickname = claims.Name
        now := time.Now()
        user.OIDCLastSync = &now

        if err := s.db.Save(&user).Error; err != nil {
            return nil, err
        }
    }

    return &user, nil
}
```

### 5. API Handlers

**File**: `backend/modules/auth/oidc/handler.go`

```go
package oidc

// InitiateOIDCLogin starts the OIDC login flow
func InitiateOIDCLogin(c *gin.Context) {
    service := getOIDCService(c)

    authURL, err := service.InitiateLogin(c)
    if err != nil {
        c.JSON(500, gin.H{"error": "Failed to initiate login"})
        return
    }

    c.Redirect(302, authURL)
}

// HandleOIDCCallback processes the OIDC provider callback
func HandleOIDCCallback(c *gin.Context) {
    code := c.Query("code")
    state := c.Query("state")

    if code == "" || state == "" {
        c.JSON(400, gin.H{"error": "Missing code or state"})
        return
    }

    service := getOIDCService(c)

    user, err := service.HandleCallback(c, code, state)
    if err != nil {
        c.JSON(401, gin.H{"error": "Authentication failed", "details": err.Error()})
        return
    }

    // Create Pantheon session (reuse existing session logic)
    sessionService := session.GetSessionService(c)
    sessionToken, err := sessionService.CreateSession(c, user)
    if err != nil {
        c.JSON(500, gin.H{"error": "Failed to create session"})
        return
    }

    // Set session cookie
    c.SetCookie("pantheon_session", sessionToken, 3600, "/", "", true, true)

    // Redirect to dashboard
    c.Redirect(302, "/dashboard")
}

// GetOIDCConfig returns OIDC configuration info for frontend
func GetOIDCConfig(c *gin.Context) {
    config := getOIDCConfigFromContext(c)

    c.JSON(200, gin.H{
        "enabled":   config.Enabled,
        "loginURL":  "/api/v1/auth/oidc/login",
        "provider":  config.ProviderURL,
    })
}
```

---

## Security Model

### 1. CSRF Protection (State Parameter)

- Generate cryptographically random state (32 bytes)
- Store in Redis with 5-minute TTL
- Validate on callback
- Single-use (delete after validation)

### 2. Replay Protection (Nonce)

- Generate cryptographically random nonce (32 bytes)
- Include in authorization request
- Verify nonce in ID token matches
- Prevents token replay attacks

### 3. ID Token Validation

```go
// Required validations:
// 1. Signature verification (using provider's JWK)
// 2. Issuer (iss) matches provider URL
// 3. Audience (aud) matches client ID
// 4. Expiry (exp) is in the future
// 5. Nonce matches stored value
// 6. Issued at (iat) is reasonable

func validateIDToken(token *oidc.IDToken, expectedNonce string) error {
    // Signature, issuer, audience, expiry checked by go-oidc library
    
    var claims struct {
        Nonce string `json:"nonce"`
    }
    if err := token.Claims(&claims); err != nil {
        return err
    }
    
    if claims.Nonce != expectedNonce {
        return errors.New("nonce mismatch")
    }
    
    return nil
}
```

### 4. Auto-Provision Safeguards

**Options**:

**Option A: Admin Approval Required** (Most Secure)
```go
user = models.User{
    // ...
    Status: "pending_approval", // Admin must approve
}
```

**Option B: Domain Whitelist** (Balanced)
```go
PANTHEON_OIDC_ALLOWED_DOMAINS=example.com,partner.com

// In provisioning logic:
emailDomain := extractDomain(claims.Email)
if !contains(allowedDomains, emailDomain) {
    return errors.New("domain not allowed")
}
```

**Option C: Email Verification Required** (OIDC Verification)
```go
if !claims.EmailVerified {
    return errors.New("email not verified by provider")
}
```

### 5. Session Security

After OIDC authentication:
- Create standard Pantheon session
- Use existing session token mechanism
- Session expiry/refresh logic unchanged
- CSRF protection still applies

---

## User Provisioning Rules

### Decision Tree

```
User authenticates via OIDC
    │
    ├─> Check if oidc_subject exists in DB
    │   │
    │   ├─> YES: Existing user
    │   │   ├─> Sync profile (email, name)
    │   │   ├─> Update oidc_last_sync
    │   │   └─> Create session
    │   │
    │   └─> NO: New user
    │       ├─> Check if auto_provision enabled
    │       │   │
    │       │   ├─> NO: Return error "User not found"
    │       │   │
    │       │   └─> YES: Check domain whitelist
    │       │       │
    │       │       ├─> Domain not allowed: Return error
    │       │       │
    │       │       └─> Domain allowed:
    │       │           ├─> Create user account
    │       │           ├─> Set status (active or pending)
    │       │           ├─> Assign default role (if configured)
    │       │           └─> Create session
```

### Profile Sync Strategy

**On Every Login**:
- Email (update if changed)
- Display name (update if changed)
- Last sync timestamp

**Not Synced** (manual management):
- Roles (assigned in Pantheon)
- Permissions (assigned in Pantheon)
- Department (assigned in Pantheon)
- Status (active/inactive)

---

## Configuration Examples

### Auth0

```bash
PANTHEON_OIDC_ENABLED=true
PANTHEON_OIDC_PROVIDER_URL=https://example.auth0.com
PANTHEON_OIDC_CLIENT_ID=your-client-id
PANTHEON_OIDC_CLIENT_SECRET=your-client-secret
PANTHEON_OIDC_REDIRECT_URL=https://pantheon.example.com/api/v1/auth/oidc/callback
PANTHEON_OIDC_SCOPES=openid,profile,email
```

### Okta

```bash
PANTHEON_OIDC_PROVIDER_URL=https://example.okta.com
PANTHEON_OIDC_CLIENT_ID=0oa...
PANTHEON_OIDC_CLIENT_SECRET=...
PANTHEON_OIDC_REDIRECT_URL=https://pantheon.example.com/api/v1/auth/oidc/callback
PANTHEON_OIDC_SCOPES=openid,profile,email
```

### Azure AD

```bash
PANTHEON_OIDC_PROVIDER_URL=https://login.microsoftonline.com/{tenant-id}/v2.0
PANTHEON_OIDC_CLIENT_ID=your-application-id
PANTHEON_OIDC_CLIENT_SECRET=your-client-secret
PANTHEON_OIDC_REDIRECT_URL=https://pantheon.example.com/api/v1/auth/oidc/callback
PANTHEON_OIDC_SCOPES=openid,profile,email
```

### Keycloak

```bash
PANTHEON_OIDC_PROVIDER_URL=https://keycloak.example.com/realms/pantheon
PANTHEON_OIDC_CLIENT_ID=pantheon-client
PANTHEON_OIDC_CLIENT_SECRET=...
PANTHEON_OIDC_REDIRECT_URL=https://pantheon.example.com/api/v1/auth/oidc/callback
PANTHEON_OIDC_SCOPES=openid,profile,email
```

---

## Testing Strategy

### Unit Tests

1. **OIDC Provider Tests**:
   - Provider initialization
   - AuthCodeURL generation
   - Token exchange (mock HTTP)
   - ID token verification (mock JWK)

2. **Service Tests**:
   - State/nonce generation and validation
   - User provisioning logic
   - Profile sync logic
   - Domain whitelist enforcement

3. **Handler Tests**:
   - Login initiation
   - Callback handling
   - Error cases

### Integration Tests

1. **Mock OIDC Provider**:
   - Use `httptest` to mock IdP endpoints
   - Full flow: login → callback → session

2. **Database Integration**:
   - User creation
   - User update
   - OIDC field persistence

### Manual Testing

**Test Cases**:
- [ ] Login with new user (auto-provision enabled)
- [ ] Login with new user (auto-provision disabled) → Error
- [ ] Login with existing user → Profile synced
- [ ] Invalid state parameter → Error
- [ ] Expired state → Error
- [ ] Invalid nonce → Error
- [ ] Disallowed domain → Error
- [ ] Session created correctly
- [ ] Logout works

---

## Rollout Plan

### Phase 1: MVP (P1 - Current)
- Single OIDC provider
- JIT user provisioning
- Basic profile sync
- Manual role assignment
- Configuration via environment variables

### Phase 2: Enhanced (P2 - Future)
- Multiple provider support
- Group/role mapping from IdP claims
- Custom claim parsing
- Admin UI for provider configuration
- Provider-specific user listing

### Phase 3: Advanced (P3 - Future)
- Single Logout (SLO)
- SAML 2.0 support
- Advanced claim transformation
- Conditional access policies

---

## Risks & Mitigation

| Risk | Impact | Mitigation |
|------|--------|------------|
| Auto-provision abuse | High | Domain whitelist + email verification |
| State/nonce replay | Medium | Redis storage with TTL, single-use |
| ID token forgery | High | Signature verification required |
| Provider downtime | Medium | Fallback to local auth always available |
| Profile sync conflicts | Low | OIDC fields separate from local fields |
| Role explosion | Medium | Manual role assignment initially |

---

## Open Questions

1. **Default role for auto-provisioned users?**
   - Option A: No role (pending approval)
   - Option B: "viewer" role (read-only)
   - **Recommendation**: No role, require admin assignment

2. **Existing users with same email?**
   - Option A: Link to existing account
   - Option B: Create new account with email+oidc
   - **Recommendation**: Link if email matches AND user has no password set

3. **Logout behavior?**
   - Option A: Local logout only
   - Option B: Redirect to IdP logout (SLO)
   - **Recommendation**: Local logout initially, SLO in P2

4. **Username generation for auto-provisioned users?**
   - Option A: Email prefix (before @)
   - Option B: OIDC sub (unique ID)
   - Option C: Email+random suffix if conflict
   - **Recommendation**: Option C

---

## Next Steps

1. **Review & Approve Design** (1 hour)
2. **Database Migration** (1 hour)
3. **OIDC Module Implementation** (4 hours)
4. **Integration & Testing** (3 hours)
5. **Documentation** (2 hours)
6. **Manual Testing with Real IdP** (1 hour)

**Total**: 12 hours

---

**Document Status**: ✅ Design Complete  
**Next Phase**: Implementation (awaiting approval)  
**Owner**: Pantheon Base Backend Team  
**Reviewers**: Security Team, Product Team
