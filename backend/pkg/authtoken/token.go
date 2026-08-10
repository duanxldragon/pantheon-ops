package authtoken

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

// Token types, scopes, and default TTL values used by auth session helpers.
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
	// sessionRefreshIndexPrefix 存 sessionID → 当前 refresh token 的反向索引，
	// 使会话吊销（logout/强制下线/改密）能级联删除 Redis 中的 refresh entry，
	// 不依赖 DB session 状态校验作为唯一防线。
	sessionRefreshIndexPrefix = "pantheon:sessref:"
)

// Shared token TTL overrides and sentinel errors.
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

// SetTokenTTL updates the access and refresh token TTL overrides.
func SetTokenTTL(accessTTL, refreshTTL time.Duration) {
	if accessTTL > 0 {
		AccessTokenTTL = accessTTL
	}
	if refreshTTL > 0 {
		RefreshTokenTTL = refreshTTL
	}
}

// Pair holds the issued access and refresh tokens for a session.
type Pair struct {
	AccessToken      string    `json:"accessToken"`
	RefreshToken     string    `json:"refreshToken"`
	TokenType        string    `json:"tokenType"`
	AccessExpiresAt  time.Time `json:"accessExpiresAt"`
	RefreshExpiresAt time.Time `json:"refreshExpiresAt"`
	SessionID        string    `json:"sessionId"`
}

// SessionData is the Redis payload stored for an authenticated session.
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

// OperationData captures the payload stored for a short-lived operation token.
type OperationData struct {
	UserID    uint64 `json:"uid"`
	SessionID string `json:"sid"`
	Scope     string `json:"scope"`
}

// SessionKey builds the Redis key for an access token.
func SessionKey(tok string) string { return sessionPrefix + tok }

// RefreshKey builds the Redis key for a refresh token.
func RefreshKey(tok string) string { return refreshPrefix + tok }

// OperationKey builds the Redis key for an operation token.
func OperationKey(tok string) string { return operationPrefix + tok }

// NewAccessToken generates a new access token value.
func NewAccessToken() string { return randHex(32) }

// NewRefreshToken generates a new refresh token value.
func NewRefreshToken() string { return randHex(32) }

// NewOperationToken generates a new operation token value.
func NewOperationToken() string { return randHex(32) }

func randHex(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		panic("crypto/rand unavailable: " + err.Error())
	}
	return hex.EncodeToString(b)
}

// StoreSession writes an access token session payload to Redis.
func StoreSession(ctx context.Context, rdb *redis.Client, tok string, d *SessionData, ttl time.Duration) error {
	if rdb == nil {
		return ErrStoreNotInitialized
	}
	b, err := json.Marshal(d)
	if err != nil {
		return err
	}
	return rdb.Set(ctx, SessionKey(tok), b, ttl).Err()
}

// ValidateSession loads and decodes an access token session payload from Redis.
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

// DeleteSession removes an access token session payload from Redis.
func DeleteSession(ctx context.Context, rdb *redis.Client, tok string) error {
	if rdb == nil {
		return nil
	}
	return rdb.Del(ctx, SessionKey(tok)).Err()
}

// RefreshSessionActivity rewrites the session payload while preserving its TTL.
func RefreshSessionActivity(ctx context.Context, rdb *redis.Client, tok string, d *SessionData) error {
	if rdb == nil {
		return ErrStoreNotInitialized
	}
	b, err := json.Marshal(d)
	if err != nil {
		return err
	}
	ttl, err := rdb.TTL(ctx, SessionKey(tok)).Result()
	if err != nil || ttl <= 0 {
		ttl = AccessTokenTTL
	}
	return rdb.Set(ctx, SessionKey(tok), b, ttl).Err()
}

// StoreRefresh writes a refresh token mapping to Redis, plus a sessionID→token
// reverse index (same TTL) so session revocation can cascade-delete the token.
func StoreRefresh(ctx context.Context, rdb *redis.Client, tok string, uid uint64, sid string, ttl time.Duration) error {
	if rdb == nil {
		return ErrStoreNotInitialized
	}
	b, err := json.Marshal(refreshEntry{UserID: uid, SessionID: sid})
	if err != nil {
		return err
	}
	pipe := rdb.Pipeline()
	pipe.Set(ctx, RefreshKey(tok), b, ttl)
	if sid != "" {
		// 同一会话轮换 refresh token 时，旧 token 由调用方显式删除；
		// 这里只维护"当前有效 token"的索引，Set 覆盖即可。
		pipe.Set(ctx, sessionRefreshIndexKey(sid), tok, ttl)
	}
	_, err = pipe.Exec(ctx)
	return err
}

// ValidateRefresh loads a refresh token mapping from Redis.
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

// DeleteRefresh removes a refresh token mapping from Redis.
func DeleteRefresh(ctx context.Context, rdb *redis.Client, tok string) error {
	if rdb == nil {
		return nil
	}
	return rdb.Del(ctx, RefreshKey(tok)).Err()
}

// sessionRefreshIndexKey builds the Redis key for the session→refresh reverse index.
func sessionRefreshIndexKey(sessionID string) string {
	return sessionRefreshIndexPrefix + sessionID
}

// blacklistPrefix must stay in sync with the per-request check in
// internal/middleware/token_middleware.go (TokenAuthMiddleware).
const blacklistPrefix = "blacklist:"

// BlacklistUserKey builds the Redis key TokenAuthMiddleware checks on every
// authenticated request.
func BlacklistUserKey(userID uint64) string {
	return blacklistPrefix + strconv.FormatUint(userID, 10)
}

// BlacklistUser force-expires every live access token of a user by writing the
// per-user blacklist key. TTL covers the longest possible remaining access
// token life (RefreshSessionActivity preserves, never extends, session TTLs)
// plus a margin for the middleware's in-memory session cache and clock skew.
func BlacklistUser(ctx context.Context, rdb *redis.Client, userID uint64) error {
	if rdb == nil || userID == 0 {
		return nil
	}
	return rdb.Set(ctx, BlacklistUserKey(userID), "1", AccessTokenTTL+time.Minute).Err()
}

// RevokeSessionRefresh cascade-deletes the refresh token bound to a session,
// along with the reverse-index entry. Missing keys are treated as success —
// the session may never have issued a token or it may have already expired.
func RevokeSessionRefresh(ctx context.Context, rdb *redis.Client, sessionID string) error {
	if rdb == nil || sessionID == "" {
		return nil
	}
	indexKey := sessionRefreshIndexKey(sessionID)
	tok, err := rdb.Get(ctx, indexKey).Result()
	if err == redis.Nil {
		return nil
	}
	if err != nil {
		return err
	}
	pipe := rdb.Pipeline()
	pipe.Del(ctx, RefreshKey(tok))
	pipe.Del(ctx, indexKey)
	_, err = pipe.Exec(ctx)
	return err
}

// DeleteSessionPair removes both access and refresh token keys from Redis.
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

// StoreOperation writes a privileged operation token payload to Redis.
func StoreOperation(ctx context.Context, rdb *redis.Client, tok string, data *OperationData, ttl time.Duration) error {
	if rdb == nil {
		return ErrStoreNotInitialized
	}
	b, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return rdb.Set(ctx, OperationKey(tok), b, ttl).Err()
}

// ValidateOperation loads and decodes an operation token payload from Redis.
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

// DeleteOperation removes an operation token payload from Redis.
func DeleteOperation(ctx context.Context, rdb *redis.Client, tok string) error {
	if rdb == nil {
		return nil
	}
	return rdb.Del(ctx, OperationKey(tok)).Err()
}

// GenerateOperationTokenWithContext creates and stores a new operation token.
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

// GenerateOperationToken creates and stores a new operation token.
func GenerateOperationToken(userID uint64, sessionID string, operationScope string, ttl time.Duration, rdb *redis.Client) (string, error) {
	return GenerateOperationTokenWithContext(context.Background(), userID, sessionID, operationScope, ttl, rdb)
}

// ParseOperationTokenWithContext validates an operation token using the provided context.
func ParseOperationTokenWithContext(ctx context.Context, tokenString string, rdb *redis.Client) (*OperationData, error) {
	if rdb == nil {
		return nil, ErrStoreNotInitialized
	}
	if ctx == nil {
		ctx = context.Background()
	}
	return ValidateOperation(ctx, rdb, tokenString)
}

// ParseOperationToken validates an operation token using a background context.
func ParseOperationToken(tokenString string, rdb *redis.Client) (*OperationData, error) {
	return ParseOperationTokenWithContext(context.Background(), tokenString, rdb)
}
