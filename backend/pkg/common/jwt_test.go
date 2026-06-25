package common

import (
	"testing"
	"time"
)

func TestGenerateAccessTokenProducesValidToken(t *testing.T) {
	token, expiresAt, err := GenerateAccessToken(1, "admin", []string{"admin"}, "sess-1", "tid-1")
	if err != nil {
		t.Fatalf("GenerateAccessToken failed: %v", err)
	}
	if token == "" {
		t.Fatal("expected non-empty token")
	}
	if expiresAt.Before(time.Now()) {
		t.Fatal("expected future expiration")
	}
}

func TestGenerateRefreshTokenProducesValidToken(t *testing.T) {
	token, expiresAt, err := GenerateRefreshToken(1, "admin", []string{"admin"}, "sess-1", "tid-2")
	if err != nil {
		t.Fatalf("GenerateRefreshToken failed: %v", err)
	}
	if token == "" {
		t.Fatal("expected non-empty token")
	}
	if expiresAt.Before(time.Now()) {
		t.Fatal("expected future expiration")
	}
}

func TestGenerateTokenPairReturnsBothTokens(t *testing.T) {
	pair, err := GenerateTokenPair(1, "admin", []string{"admin"}, "sess-1", "atid", "rtid")
	if err != nil {
		t.Fatalf("GenerateTokenPair failed: %v", err)
	}
	if pair.AccessToken == "" {
		t.Fatal("expected non-empty access token")
	}
	if pair.RefreshToken == "" {
		t.Fatal("expected non-empty refresh token")
	}
	if pair.TokenType != "Bearer" {
		t.Fatalf("expected Bearer token type, got %s", pair.TokenType)
	}
	if pair.SessionID != "sess-1" {
		t.Fatalf("expected session ID sess-1, got %s", pair.SessionID)
	}
}

func TestParseAccessTokenReturnsClaims(t *testing.T) {
	token, _, err := GenerateAccessToken(42, "testuser", []string{"role1", "role2"}, "sess-abc", "tid-100")
	if err != nil {
		t.Fatalf("GenerateAccessToken failed: %v", err)
	}

	claims, err := ParseToken(token, TokenTypeAccess)
	if err != nil {
		t.Fatalf("ParseToken failed: %v", err)
	}
	if claims.UserID != 42 {
		t.Fatalf("expected UserID 42, got %d", claims.UserID)
	}
	if claims.Username != "testuser" {
		t.Fatalf("expected Username testuser, got %s", claims.Username)
	}
	if len(claims.RoleKeys) != 2 || claims.RoleKeys[0] != "role1" {
		t.Fatalf("unexpected roles: %v", claims.RoleKeys)
	}
	if claims.SessionID != "sess-abc" {
		t.Fatalf("expected session sess-abc, got %s", claims.SessionID)
	}
}

func TestParseRefreshTokenRejectsAccessTokenType(t *testing.T) {
	token, _, err := GenerateAccessToken(1, "admin", nil, "sess-1", "tid-1")
	if err != nil {
		t.Fatalf("GenerateAccessToken failed: %v", err)
	}

	// Access tokens are signed with AccessTokenSecret. ParseToken with
	// TokenTypeRefresh uses RefreshTokenSecret, so signature validation
	// fails before type check. Any error here is acceptable.
	_, err = ParseToken(token, TokenTypeRefresh)
	if err == nil {
		t.Fatal("expected error when parsing access token as refresh token")
	}
}

func TestParseRefreshTokenWithRefreshTokenSucceeds(t *testing.T) {
	token, _, err := GenerateRefreshToken(1, "admin", []string{"admin"}, "sess-1", "tid-2")
	if err != nil {
		t.Fatalf("GenerateRefreshToken failed: %v", err)
	}

	claims, err := ParseToken(token, TokenTypeRefresh)
	if err != nil {
		t.Fatalf("ParseToken with refresh token failed: %v", err)
	}
	if claims.UserID != 1 {
		t.Fatalf("expected UserID 1, got %d", claims.UserID)
	}
}

func TestParseInvalidTokenReturnsError(t *testing.T) {
	_, err := ParseToken("invalid-token-string", TokenTypeAccess)
	if err == nil {
		t.Fatal("expected error for invalid token")
	}
}

func TestParseTamperedTokenReturnsError(t *testing.T) {
	token, _, err := GenerateAccessToken(1, "admin", nil, "sess-1", "tid-1")
	if err != nil {
		t.Fatalf("GenerateAccessToken failed: %v", err)
	}

	tampered := token + "x"
	_, err = ParseToken(tampered, TokenTypeAccess)
	if err == nil {
		t.Fatal("expected error for tampered token")
	}
}

func TestGenerateAndParseOperationToken(t *testing.T) {
	token, err := GenerateOperationToken(1, "sess-1", "view:secret", 5*time.Minute)
	if err != nil {
		t.Fatalf("GenerateOperationToken failed: %v", err)
	}
	if token == "" {
		t.Fatal("expected non-empty operation token")
	}

	claims, err := ParseOperationToken(token)
	if err != nil {
		t.Fatalf("ParseOperationToken failed: %v", err)
	}
	if claims.UserID != 1 {
		t.Fatalf("expected UserID 1, got %d", claims.UserID)
	}
	if claims.OperationScope != "view:secret" {
		t.Fatalf("expected scope view:secret, got %s", claims.OperationScope)
	}
	if claims.TokenType != TokenTypeOperation {
		t.Fatalf("expected TokenTypeOperation, got %s", claims.TokenType)
	}
}

func TestParseOperationTokenWithAccessTokenFails(t *testing.T) {
	accessToken, _, err := GenerateAccessToken(1, "admin", nil, "sess-1", "tid-1")
	if err != nil {
		t.Fatalf("GenerateAccessToken failed: %v", err)
	}

	// Access tokens use AccessTokenSecret, operation tokens use
	// OperationTokenSecret, so signature validation fails before type check.
	_, err = ParseOperationToken(accessToken)
	if err == nil {
		t.Fatal("expected error when parsing access token as operation token")
	}
}

func TestParseOperationTokenInvalidStringFails(t *testing.T) {
	_, err := ParseOperationToken("not-a-real-token")
	if err == nil {
		t.Fatal("expected error for invalid operation token")
	}
}

func TestGenerateTokenPairWithEmptyRoleKeys(t *testing.T) {
	pair, err := GenerateTokenPair(0, "anonymous", []string{}, "sess-empty", "atid", "rtid")
	if err != nil {
		t.Fatalf("GenerateTokenPair with empty roles failed: %v", err)
	}
	if pair.AccessToken == "" {
		t.Fatal("expected non-empty access token")
	}
}

func TestAccessTokenTTL(t *testing.T) {
	ttl := accessTokenTTL()
	if ttl != 15*time.Minute {
		t.Fatalf("expected 15m TTL, got %v", ttl)
	}
}

func TestRefreshTokenTTL(t *testing.T) {
	ttl := refreshTokenTTL()
	if ttl != 7*24*time.Hour {
		t.Fatalf("expected 7d TTL, got %v", ttl)
	}
}

func TestBuildClaimsSetsRegisteredClaims(t *testing.T) {
	now := time.Now()
	expiresAt := now.Add(15 * time.Minute)
	claims := buildClaims(1, "user", []string{"role"}, TokenTypeAccess, "sess-1", "tid-1", expiresAt)

	if claims.ID != "tid-1" {
		t.Fatalf("expected ID tid-1, got %s", claims.ID)
	}
	if claims.ExpiresAt == nil {
		t.Fatal("expected ExpiresAt to be set")
	}
	if claims.IssuedAt == nil {
		t.Fatal("expected IssuedAt to be set")
	}
	if claims.NotBefore == nil {
		t.Fatal("expected NotBefore to be set")
	}
}
