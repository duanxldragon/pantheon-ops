package middleware

import (
	"context"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"pantheon-base/pkg/authtoken"
	"pantheon-base/pkg/common"
	commonhttp "pantheon-base/pkg/common/http"
	"pantheon-base/pkg/database"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

const defaultSessionIdleMinutes = 30

var (
	tokenSessionCacheMu  sync.RWMutex
	tokenSessionCache    = make(map[string]*tokenCachedSession)
	tokenSessionCacheTTL = loadTokenSessionCacheTTL()
)

type tokenCachedSession struct {
	data      *authtoken.SessionData
	cachedAt  time.Time
	expiresAt time.Time
}

func tokenSessionCacheKey(token string) string {
	return token
}

func lookupTokenSessionCache(token string) (*tokenCachedSession, bool) {
	if tokenSessionCacheTTL <= 0 {
		return nil, false
	}
	tokenSessionCacheMu.RLock()
	defer tokenSessionCacheMu.RUnlock()
	entry, ok := tokenSessionCache[tokenSessionCacheKey(token)]
	if !ok {
		return nil, false
	}
	if time.Since(entry.cachedAt) > tokenSessionCacheTTL {
		return nil, false
	}
	return entry, true
}

func storeTokenSessionCache(token string, data *authtoken.SessionData, expiresAt time.Time) {
	if tokenSessionCacheTTL <= 0 {
		return
	}
	tokenSessionCacheMu.Lock()
	defer tokenSessionCacheMu.Unlock()
	tokenSessionCache[tokenSessionCacheKey(token)] = &tokenCachedSession{
		data:      data,
		cachedAt:  time.Now(),
		expiresAt: expiresAt,
	}
	if len(tokenSessionCache) > 10000 {
		now := time.Now()
		for k, v := range tokenSessionCache {
			if now.Sub(v.cachedAt) > tokenSessionCacheTTL {
				delete(tokenSessionCache, k)
			}
		}
	}
}

func loadTokenSessionCacheTTL() time.Duration {
	raw := strings.TrimSpace(os.Getenv("PANTHEON_TOKEN_CACHE_TTL_SECONDS"))
	if raw == "" {
		return 60 * time.Second
	}
	seconds, err := strconv.Atoi(raw)
	if err != nil || seconds < 0 {
		return 60 * time.Second
	}
	return time.Duration(seconds) * time.Second
}

func invalidateTokenSessionCache(token string) {
	tokenSessionCacheMu.Lock()
	defer tokenSessionCacheMu.Unlock()
	delete(tokenSessionCache, tokenSessionCacheKey(token))
}

var (
	sessionIdleMinutesMu       sync.RWMutex
	cachedSessionIdleMinutes   = defaultSessionIdleMinutes
	cachedSessionIdleMinutesAt time.Time
)

// extractToken extracts the access token from Authorization header or cookie.
func extractToken(c *gin.Context) string {
	if token, err := c.Cookie(commonhttp.CookieAccessToken); err == nil && token != "" {
		return token
	}
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		return ""
	}
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || parts[0] != "Bearer" {
		return ""
	}
	return parts[1]
}

// resolveTokenSession 解析 token 对应的会话：优先命中本地缓存（未过期），
// 否则回源 Redis 校验并写入缓存。校验失败（无效/过期）时直接以 401 中断请求，
// 返回 (nil, true)，调用方须立即 return。
func resolveTokenSession(ctx context.Context, c *gin.Context, rdb *redis.Client, token string) (*authtoken.SessionData, bool) {
	if cached, cacheHit := lookupTokenSessionCache(token); cacheHit && cached.expiresAt.After(time.Now()) {
		return cached.data, false
	}

	sessionData, err := authtoken.ValidateSession(ctx, rdb, token)
	if err != nil {
		invalidateTokenSessionCache(token)
		common.Fail(c, common.CodeUnauthorized, "token.invalid")
		c.Abort()
		return nil, true
	}

	ttl, ttlErr := rdb.TTL(ctx, authtoken.SessionKey(token)).Result()
	if ttlErr == nil && ttl > 0 {
		storeTokenSessionCache(token, sessionData, time.Now().Add(ttl))
	}
	return sessionData, false
}

// checkTokenBlacklist 检查用户是否被强制拉黑；命中则直接 401 中断并返回 true。
func checkTokenBlacklist(ctx context.Context, c *gin.Context, rdb *redis.Client, token string, sessionData *authtoken.SessionData) bool {
	if rdb == nil {
		return false
	}
	blacklistKey := authtoken.BlacklistUserKey(sessionData.UserID)
	val, err := rdb.Get(ctx, blacklistKey).Result()
	if err == nil && val != "" {
		invalidateTokenSessionCache(token)
		common.Fail(c, common.CodeUnauthorized, "token.expired.force")
		c.Abort()
		return true
	}
	return false
}

// enforceTokenIdleTimeout 检查会话空闲超时；超时则清理 Redis 会话并 401 中断，返回 true。
func enforceTokenIdleTimeout(ctx context.Context, c *gin.Context, rdb *redis.Client, token string, sessionData *authtoken.SessionData) bool {
	idleMinutes := loadSessionIdleMinutes()
	if idleMinutes > 0 && sessionData.LastActivityAt > 0 {
		lastActivity := time.Unix(sessionData.LastActivityAt, 0)
		if lastActivity.Add(time.Duration(idleMinutes) * time.Minute).Before(time.Now()) {
			invalidateTokenSessionCache(token)
			_ = authtoken.DeleteSession(ctx, rdb, token)
			common.Fail(c, common.CodeUnauthorized, "session.idle_timeout")
			c.Abort()
			return true
		}
	}
	return false
}

// refreshTokenActivity 尽力刷新 Redis 中的活跃时间戳并同步本地缓存；失败不影响请求。
func refreshTokenActivity(ctx context.Context, rdb *redis.Client, token string, sessionData *authtoken.SessionData) {
	if sessionData.LastActivityAt == 0 || time.Since(time.Unix(sessionData.LastActivityAt, 0)) > time.Minute {
		sessionData.LastActivityAt = time.Now().Unix()
		if err := authtoken.RefreshSessionActivity(ctx, rdb, token, sessionData); err == nil {
			if cached, ok := lookupTokenSessionCache(token); ok {
				cached.data.LastActivityAt = sessionData.LastActivityAt
			}
		}
	}
}

// applyTokenContext 将鉴权主体写入 Gin 上下文，键名与顺序与原实现完全一致，供下游读取。
func applyTokenContext(c *gin.Context, sessionData *authtoken.SessionData) {
	c.Set("userId", sessionData.UserID)
	c.Set("username", sessionData.Username)
	c.Set("roleKeys", sessionData.RoleKeys)
	c.Set("sessionId", sessionData.SessionID)
	if len(sessionData.RoleKeys) > 0 {
		c.Set("roleKey", sessionData.RoleKeys[0])
	}
}

// TokenAuthMiddleware is the unified middleware for Redis opaque token authentication.
// It combines the simplicity of Redis token validation with session management features:
// - 60-second in-memory cache for performance
// - Session idle timeout checking
// - Redis blacklist checking
// - Automatic activity tracking
func TokenAuthMiddleware(rdb *redis.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := extractToken(c)
		if token == "" {
			common.Fail(c, common.CodeUnauthorized, "token.missing")
			c.Abort()
			return
		}

		ctx := c.Request.Context()
		sessionData, aborted := resolveTokenSession(ctx, c, rdb, token)
		if aborted {
			return
		}
		if checkTokenBlacklist(ctx, c, rdb, token, sessionData) {
			return
		}
		if enforceTokenIdleTimeout(ctx, c, rdb, token, sessionData) {
			return
		}

		refreshTokenActivity(ctx, rdb, token, sessionData)
		applyTokenContext(c, sessionData)
		c.Next()
	}
}

func loadSessionIdleMinutes() int {
	sessionIdleMinutesMu.RLock()
	if !cachedSessionIdleMinutesAt.IsZero() && time.Since(cachedSessionIdleMinutesAt) < time.Minute {
		value := cachedSessionIdleMinutes
		sessionIdleMinutesMu.RUnlock()
		return value
	}
	sessionIdleMinutesMu.RUnlock()

	value := defaultSessionIdleMinutes
	if database.DB != nil {
		var raw string
		err := database.DB.Table("system_setting").
			Select("setting_value").
			Where("setting_key = ?", "login.session_idle_minutes").
			Limit(1).
			Pluck("setting_value", &raw).Error
		if err == nil {
			if parsed, parseErr := strconv.Atoi(strings.TrimSpace(raw)); parseErr == nil && parsed > 0 {
				value = parsed
			}
		}
	}

	sessionIdleMinutesMu.Lock()
	cachedSessionIdleMinutes = value
	cachedSessionIdleMinutesAt = time.Now()
	sessionIdleMinutesMu.Unlock()
	return value
}

// InvalidateTokenAuthCache invalidates the token session cache for the given token.
// Call this when logging out or revoking a session.
func InvalidateTokenAuthCache(token string) {
	invalidateTokenSessionCache(token)
}
