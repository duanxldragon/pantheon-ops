package common

import (
	"errors"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	TokenTypeAccess    = "access"
	TokenTypeRefresh   = "refresh"
	TokenTypeOperation = "operation"
)

var (
	AccessTokenSecret    = []byte(DefaultAccessTokenSecret)
	RefreshTokenSecret   = []byte(DefaultRefreshTokenSecret)
	OperationTokenSecret = []byte(DefaultOperationTokenSecret)
	ErrTokenInvalid      = errors.New("token.invalid")
	ErrTokenExpired      = errors.New("token.expired")
	ErrTokenType         = errors.New("token.type.invalid")
)

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
	return 15 * time.Minute
}

func refreshTokenTTL() time.Duration {
	return 7 * 24 * time.Hour
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

func ParseToken(tokenString string, expectedType string) (*CustomClaims, error) {
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
	claims := buildClaims(userID, "op-verify", []string{}, TokenTypeOperation, sessionID, "op-"+os.Getenv("HOSTNAME"), expiresAt)
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
