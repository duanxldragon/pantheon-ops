package authtoken

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	TypeAccess    = "access"
	TypeRefresh   = "refresh"
	TypeOperation = "operation"

	ScopeSecureAction = "secure_action"

	DefaultAccessTokenTTL  = 15 * time.Minute
	DefaultRefreshTokenTTL = 7 * 24 * time.Hour
)

const (
	sessionPrefix   = "pantheon:session:"
	refreshPrefix   = "pantheon:refresh:"
	operationPrefix = "pantheon:op:"
)

var (
	AccessTokenTTL  = DefaultAccessTokenTTL
	RefreshTokenTTL = DefaultRefreshTokenTTL

	ErrNotFound             = errors.New("not_found")
	ErrStoreNotInitialized  = errors.New("token.store.not_initialized")
	ErrInvalid              = errors.New("token.invalid")
	ErrExpired              = errors.New("token.expired")
	ErrType                 = errors.New("token.type.invalid")
	ErrOperationTokenAbsent = errors.New("operation token not found or expired")
)

func SetTokenTTL(accessTTL, refreshTTL time.Duration) {
	if accessTTL > 0 {
		AccessTokenTTL = accessTTL
	}
	if refreshTTL > 0 {
		RefreshTokenTTL = refreshTTL
	}
}

type Pair struct {
	AccessToken      string    `json:"accessToken"`
	RefreshToken     string    `json:"refreshToken"`
	TokenType        string    `json:"tokenType"`
	AccessExpiresAt  time.Time `json:"accessExpiresAt"`
	RefreshExpiresAt time.Time `json:"refreshExpiresAt"`
	SessionID        string    `json:"sessionId"`
}

type SessionData struct {
	UserID         uint64   `json:"uid"`
	Username       string   `json:"un"`
	RoleKeys       []string `json:"rk"`
	SessionID      string   `json:"sid"`
	LastIP         string   `json:"ip"`
	UserAgent      string   `json:"ua"`
	LastActivityAt int64    `json:"lat"`
}

type refreshEntry struct {
	UserID    uint64 `json:"uid"`
	SessionID string `json:"sid"`
}

type OperationData struct {
	UserID    uint64 `json:"uid"`
	SessionID string `json:"sid"`
	Scope     string `json:"scope"`
}

func SessionKey(tok string) string   { return sessionPrefix + tok }
func RefreshKey(tok string) string   { return refreshPrefix + tok }
func OperationKey(tok string) string { return operationPrefix + tok }

func NewAccessToken() string    { return randHex(32) }
func NewRefreshToken() string   { return randHex(32) }
func NewOperationToken() string { return randHex(32) }

func randHex(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		panic("crypto/rand unavailable: " + err.Error())
	}
	return hex.EncodeToString(b)
}

func StoreSession(ctx context.Context, rdb *redis.Client, tok string, d *SessionData, ttl time.Duration) error {
	if rdb == nil {
		return ErrStoreNotInitialized
	}
	b, _ := json.Marshal(d)
	return rdb.Set(ctx, SessionKey(tok), b, ttl).Err()
}

func ValidateSession(ctx context.Context, rdb *redis.Client, tok string) (*SessionData, error) {
	if rdb == nil {
		return nil, ErrStoreNotInitialized
	}
	b, err := rdb.Get(ctx, SessionKey(tok)).Bytes()
	if err == redis.Nil {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	var d SessionData
	if err := json.Unmarshal(b, &d); err != nil {
		return nil, ErrInvalid
	}
	return &d, nil
}

func DeleteSession(ctx context.Context, rdb *redis.Client, tok string) error {
	if rdb == nil {
		return nil
	}
	return rdb.Del(ctx, SessionKey(tok)).Err()
}

func RefreshSessionActivity(ctx context.Context, rdb *redis.Client, tok string, d *SessionData) error {
	if rdb == nil {
		return ErrStoreNotInitialized
	}
	b, _ := json.Marshal(d)
	ttl, err := rdb.TTL(ctx, SessionKey(tok)).Result()
	if err != nil || ttl <= 0 {
		ttl = AccessTokenTTL
	}
	return rdb.Set(ctx, SessionKey(tok), b, ttl).Err()
}

func StoreRefresh(ctx context.Context, rdb *redis.Client, tok string, uid uint64, sid string, ttl time.Duration) error {
	if rdb == nil {
		return ErrStoreNotInitialized
	}
	b, _ := json.Marshal(refreshEntry{UserID: uid, SessionID: sid})
	return rdb.Set(ctx, RefreshKey(tok), b, ttl).Err()
}

func ValidateRefresh(ctx context.Context, rdb *redis.Client, tok string) (uint64, string, error) {
	if rdb == nil {
		return 0, "", ErrStoreNotInitialized
	}
	b, err := rdb.Get(ctx, RefreshKey(tok)).Bytes()
	if err == redis.Nil {
		return 0, "", ErrNotFound
	}
	if err != nil {
		return 0, "", err
	}
	var e refreshEntry
	if err := json.Unmarshal(b, &e); err != nil {
		return 0, "", ErrInvalid
	}
	return e.UserID, e.SessionID, nil
}

func DeleteRefresh(ctx context.Context, rdb *redis.Client, tok string) error {
	if rdb == nil {
		return nil
	}
	return rdb.Del(ctx, RefreshKey(tok)).Err()
}

func DeleteSessionPair(ctx context.Context, rdb *redis.Client, accessToken, refreshToken string) error {
	if rdb == nil {
		return nil
	}
	pipe := rdb.Pipeline()
	pipe.Del(ctx, SessionKey(accessToken))
	pipe.Del(ctx, RefreshKey(refreshToken))
	_, err := pipe.Exec(ctx)
	return err
}

func StoreOperation(ctx context.Context, rdb *redis.Client, tok string, data *OperationData, ttl time.Duration) error {
	if rdb == nil {
		return ErrStoreNotInitialized
	}
	b, _ := json.Marshal(data)
	return rdb.Set(ctx, OperationKey(tok), b, ttl).Err()
}

func ValidateOperation(ctx context.Context, rdb *redis.Client, tok string) (*OperationData, error) {
	if rdb == nil {
		return nil, ErrStoreNotInitialized
	}
	b, err := rdb.Get(ctx, OperationKey(tok)).Bytes()
	if err == redis.Nil {
		return nil, ErrOperationTokenAbsent
	}
	if err != nil {
		return nil, err
	}
	var d OperationData
	if err := json.Unmarshal(b, &d); err != nil {
		return nil, ErrInvalid
	}
	return &d, nil
}

func DeleteOperation(ctx context.Context, rdb *redis.Client, tok string) error {
	if rdb == nil {
		return nil
	}
	return rdb.Del(ctx, OperationKey(tok)).Err()
}

func GenerateOperationTokenWithContext(ctx context.Context, userID uint64, sessionID string, operationScope string, ttl time.Duration, rdb *redis.Client) (string, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	opToken := NewOperationToken()
	data := &OperationData{
		UserID:    userID,
		SessionID: sessionID,
		Scope:     operationScope,
	}
	if err := StoreOperation(ctx, rdb, opToken, data, ttl); err != nil {
		return "", err
	}
	return opToken, nil
}

func GenerateOperationToken(userID uint64, sessionID string, operationScope string, ttl time.Duration, rdb *redis.Client) (string, error) {
	return GenerateOperationTokenWithContext(context.Background(), userID, sessionID, operationScope, ttl, rdb)
}

func ParseOperationTokenWithContext(ctx context.Context, tokenString string, rdb *redis.Client) (*OperationData, error) {
	if rdb == nil {
		return nil, ErrStoreNotInitialized
	}
	if ctx == nil {
		ctx = context.Background()
	}
	return ValidateOperation(ctx, rdb, tokenString)
}

func ParseOperationToken(tokenString string, rdb *redis.Client) (*OperationData, error) {
	return ParseOperationTokenWithContext(context.Background(), tokenString, rdb)
}
