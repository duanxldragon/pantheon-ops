package common

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
	"pantheon-ops/backend/pkg/authtoken"
	commonhttp "pantheon-ops/backend/pkg/common/http"
)

const (
	TokenTypeAccess    = authtoken.TypeAccess
	TokenTypeRefresh   = authtoken.TypeRefresh
	TokenTypeOperation = authtoken.TypeOperation

	DefaultAccessTokenTTL  = authtoken.DefaultAccessTokenTTL
	DefaultRefreshTokenTTL = authtoken.DefaultRefreshTokenTTL
)

var (
	AccessTokenTTL  = authtoken.AccessTokenTTL
	RefreshTokenTTL = authtoken.RefreshTokenTTL

	ErrTokenInvalid = NewUnauthorized("token.invalid")
	ErrTokenExpired = NewUnauthorized("token.expired")
	ErrTokenType    = NewUnauthorized("token.type.invalid")
)

type TokenPair = authtoken.Pair
type TokenSessionData = authtoken.SessionData
type OperationTokenData = authtoken.OperationData

func SetTokenTTL(accessTTL, refreshTTL time.Duration) {
	authtoken.SetTokenTTL(accessTTL, refreshTTL)
	commonhttp.SetTokenTTL(accessTTL, refreshTTL)
	if accessTTL > 0 {
		AccessTokenTTL = accessTTL
	}
	if refreshTTL > 0 {
		RefreshTokenTTL = refreshTTL
	}
}

func TokenSessionKey(tok string) string   { return authtoken.SessionKey(tok) }
func TokenRefreshKey(tok string) string   { return authtoken.RefreshKey(tok) }
func TokenOperationKey(tok string) string { return authtoken.OperationKey(tok) }

func NewAccessToken() string    { return authtoken.NewAccessToken() }
func NewRefreshToken() string   { return authtoken.NewRefreshToken() }
func NewOperationToken() string { return authtoken.NewOperationToken() }

func TokenStoreSession(ctx context.Context, rdb *redis.Client, tok string, d *TokenSessionData, ttl time.Duration) error {
	return authtoken.StoreSession(ctx, rdb, tok, d, ttl)
}

func TokenValidateSession(ctx context.Context, rdb *redis.Client, tok string) (*TokenSessionData, error) {
	data, err := authtoken.ValidateSession(ctx, rdb, tok)
	return data, mapAuthTokenError(err)
}

func TokenDeleteSession(ctx context.Context, rdb *redis.Client, tok string) error {
	return authtoken.DeleteSession(ctx, rdb, tok)
}

func TokenRefreshSessionActivity(ctx context.Context, rdb *redis.Client, tok string, d *TokenSessionData) error {
	return authtoken.RefreshSessionActivity(ctx, rdb, tok, d)
}

func TokenStoreRefresh(ctx context.Context, rdb *redis.Client, tok string, uid uint64, sid string, ttl time.Duration) error {
	return authtoken.StoreRefresh(ctx, rdb, tok, uid, sid, ttl)
}

func TokenValidateRefresh(ctx context.Context, rdb *redis.Client, tok string) (uint64, string, error) {
	userID, sessionID, err := authtoken.ValidateRefresh(ctx, rdb, tok)
	return userID, sessionID, mapAuthTokenError(err)
}

func TokenDeleteRefresh(ctx context.Context, rdb *redis.Client, tok string) error {
	return authtoken.DeleteRefresh(ctx, rdb, tok)
}

func TokenDeleteSessionPair(ctx context.Context, rdb *redis.Client, accessToken, refreshToken string) error {
	return authtoken.DeleteSessionPair(ctx, rdb, accessToken, refreshToken)
}

func TokenStoreOperation(ctx context.Context, rdb *redis.Client, tok string, data *OperationTokenData, ttl time.Duration) error {
	return authtoken.StoreOperation(ctx, rdb, tok, data, ttl)
}

func TokenValidateOperation(ctx context.Context, rdb *redis.Client, tok string) (*OperationTokenData, error) {
	data, err := authtoken.ValidateOperation(ctx, rdb, tok)
	return data, mapAuthTokenError(err)
}

func TokenDeleteOperation(ctx context.Context, rdb *redis.Client, tok string) error {
	return authtoken.DeleteOperation(ctx, rdb, tok)
}

// Deprecated: use authtoken.GenerateOperationToken or auth/security operation verification.
func GenerateOperationToken(userID uint64, sessionID string, operationScope string, ttl time.Duration, rdb *redis.Client) (string, error) {
	return authtoken.GenerateOperationToken(userID, sessionID, operationScope, ttl, rdb)
}

// Deprecated: use authtoken.ParseOperationToken.
func ParseOperationToken(tokenString string, rdb *redis.Client) (*OperationTokenData, error) {
	data, err := authtoken.ParseOperationToken(tokenString, rdb)
	return data, mapAuthTokenError(err)
}

func mapAuthTokenError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, authtoken.ErrNotFound):
		return ErrNotFound
	case errors.Is(err, authtoken.ErrInvalid):
		return ErrTokenInvalid
	case errors.Is(err, authtoken.ErrExpired):
		return ErrTokenExpired
	case errors.Is(err, authtoken.ErrType):
		return ErrTokenType
	case errors.Is(err, authtoken.ErrStoreNotInitialized):
		return ErrDatabaseNotInitialized
	default:
		return err
	}
}
