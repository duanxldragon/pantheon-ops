package common

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const (
	TokenTypeAccess    = "access"
	TokenTypeRefresh   = "refresh"
	TokenTypeOperation = "operation"

	// DefaultAccessTokenTTL is the default lifetime for access tokens.
	DefaultAccessTokenTTL = 15 * time.Minute
	// DefaultRefreshTokenTTL is the default lifetime for refresh tokens.
	DefaultRefreshTokenTTL = 7 * 24 * time.Hour
)

var (
	// AccessTokenTTL is the effective TTL for access tokens, overridable via SetTokenTTL.
	AccessTokenTTL = DefaultAccessTokenTTL
	// RefreshTokenTTL is the effective TTL for refresh tokens, overridable via SetTokenTTL.
	RefreshTokenTTL = DefaultRefreshTokenTTL

	AccessTokenSecret    = []byte(DefaultDevSecrets.AccessToken)
	RefreshTokenSecret   = []byte(DefaultDevSecrets.RefreshToken)
	OperationTokenSecret = []byte(DefaultDevSecrets.OperationToken)
	ErrTokenInvalid      = errors.New("token.invalid")
	ErrTokenExpired      = errors.New("token.expired")
	ErrTokenType         = errors.New("token.type.invalid")
)

// SetTokenTTL overrides the effective token TTLs. Pass zero to use defaults.
// This allows the Setting system to configure TTLs at startup without circular imports.
func SetTokenTTL(accessTTL, refreshTTL time.Duration) {
	if accessTTL > 0 {
		AccessTokenTTL = accessTTL
	}
	if refreshTTL > 0 {
		RefreshTokenTTL = refreshTTL
	}
}

type TokenPair struct {
	AccessToken      string    `json:"accessToken"`
	RefreshToken     string    `json:"refreshToken"`
	TokenType        string    `json:"tokenType"`
	AccessExpiresAt  time.Time `json:"accessExpiresAt"`
	RefreshExpiresAt time.Time `json:"refreshExpiresAt"`
	SessionID        string    `json:"sessionId"`
}

type CustomClaims struct {
	UserID         uint64   `json:"userId"`
	Username       string   `json:"username"`
	RoleKeys       []string `json:"roleKeys"`
	TokenType      string   `json:"tokenType"`
	SessionID      string   `json:"sessionId"`
	OperationScope string   `json:"operationScope,omitempty"`
	jwt.RegisteredClaims
}

func accessTokenTTL() time.Duration {
	return AccessTokenTTL
}

func refreshTokenTTL() time.Duration {
	return RefreshTokenTTL
}

func buildClaims(userID uint64, username string, roleKeys []string, tokenType string, sessionID string, tokenID string, expiresAt time.Time) CustomClaims {
	return CustomClaims{
		UserID:    userID,
		Username:  username,
		RoleKeys:  roleKeys,
		TokenType: tokenType,
		SessionID: sessionID,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        tokenID,
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}
}

func signToken(claims CustomClaims, secret []byte) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(secret)
}

func GenerateAccessToken(userID uint64, username string, roleKeys []string, sessionID string, tokenID string) (string, time.Time, error) {
	expiresAt := time.Now().Add(accessTokenTTL())
	claims := buildClaims(userID, username, roleKeys, TokenTypeAccess, sessionID, tokenID, expiresAt)
	token, err := signToken(claims, AccessTokenSecret)
	return token, expiresAt, err
}

func GenerateRefreshToken(userID uint64, username string, roleKeys []string, sessionID string, tokenID string) (string, time.Time, error) {
	expiresAt := time.Now().Add(refreshTokenTTL())
	claims := buildClaims(userID, username, roleKeys, TokenTypeRefresh, sessionID, tokenID, expiresAt)
	token, err := signToken(claims, RefreshTokenSecret)
	return token, expiresAt, err
}

func GenerateTokenPair(userID uint64, username string, roleKeys []string, sessionID string, accessTokenID string, refreshTokenID string) (*TokenPair, error) {
	accessToken, accessExpiresAt, err := GenerateAccessToken(userID, username, roleKeys, sessionID, accessTokenID)
	if err != nil {
		return nil, err
	}
	refreshToken, refreshExpiresAt, err := GenerateRefreshToken(userID, username, roleKeys, sessionID, refreshTokenID)
	if err != nil {
		return nil, err
	}
	return &TokenPair{
		AccessToken:      accessToken,
		RefreshToken:     refreshToken,
		TokenType:        "Bearer",
		AccessExpiresAt:  accessExpiresAt,
		RefreshExpiresAt: refreshExpiresAt,
		SessionID:        sessionID,
	}, nil
}

func ParseToken(tokenString, expectedType string) (*CustomClaims, error) {
	secret := AccessTokenSecret
	if expectedType == TokenTypeRefresh {
		secret = RefreshTokenSecret
	}
	token, err := jwt.ParseWithClaims(tokenString, &CustomClaims{}, func(token *jwt.Token) (interface{}, error) {
		return secret, nil
	})
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		return nil, err
	}
	if claims, ok := token.Claims.(*CustomClaims); ok && token.Valid {
		if claims.TokenType != expectedType {
			return nil, ErrTokenType
		}
		return claims, nil
	}
	return nil, ErrTokenInvalid
}

func GenerateOperationToken(userID uint64, sessionID string, operationScope string, ttl time.Duration) (string, error) {
	expiresAt := time.Now().Add(ttl)
	tokenID := "op-" + uuid.NewString()
	claims := buildClaims(userID, "op-verify", []string{}, TokenTypeOperation, sessionID, tokenID, expiresAt)
	claims.OperationScope = operationScope
	return signToken(claims, OperationTokenSecret)
}

func ParseOperationToken(tokenString string) (*CustomClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &CustomClaims{}, func(token *jwt.Token) (interface{}, error) {
		return OperationTokenSecret, nil
	})
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		return nil, err
	}
	if claims, ok := token.Claims.(*CustomClaims); ok && token.Valid {
		if claims.TokenType != TokenTypeOperation {
			return nil, ErrTokenType
		}
		return claims, nil
	}
	return nil, ErrTokenInvalid
}
