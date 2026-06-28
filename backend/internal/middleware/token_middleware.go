package middleware

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"pantheon-ops/backend/pkg/authtoken"
	"pantheon-ops/backend/pkg/common"
	commonhttp "pantheon-ops/backend/pkg/common/http"
	"pantheon-ops/backend/pkg/database"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

const defaultSessionIdleMinutes = 30

var (
	tokenSessionCacheMu  sync.RWMutex
	tokenSessionCache    = make(map[string]*tokenCachedSession)
	tokenSessionCacheTTL = 60 * time.Second
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
	if !(len(parts) == 2 && parts[0] == "Bearer") {
		return ""
	}
	return parts[1]
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
		var sessionData *authtoken.SessionData
		var err error

		// Try cache first
		cached, cacheHit := lookupTokenSessionCache(token)
		if cacheHit && cached.expiresAt.After(time.Now()) {
			sessionData = cached.data
		} else {
			// Cache miss or expired - query Redis
			sessionData, err = authtoken.ValidateSession(ctx, rdb, token)
			if err != nil {
				invalidateTokenSessionCache(token)
				common.Fail(c, common.CodeUnauthorized, "token.invalid")
				c.Abort()
				return
			}

			// Get TTL from Redis for cache expiry
			ttl, ttlErr := rdb.TTL(ctx, authtoken.SessionKey(token)).Result()
			if ttlErr == nil && ttl > 0 {
				storeTokenSessionCache(token, sessionData, time.Now().Add(ttl))
			}
		}

		// Check blacklist
		if rdb != nil {
			blacklistKey := fmt.Sprintf("blacklist:%d", sessionData.UserID)
			val, _ := rdb.Get(ctx, blacklistKey).Result()
			if val != "" {
				invalidateTokenSessionCache(token)
				common.Fail(c, common.CodeUnauthorized, "token.expired.force")
				c.Abort()
				return
			}
		}

		// Check idle timeout
		idleMinutes := loadSessionIdleMinutes()
		if idleMinutes > 0 && sessionData.LastActivityAt > 0 {
			lastActivity := time.Unix(sessionData.LastActivityAt, 0)
			if lastActivity.Add(time.Duration(idleMinutes) * time.Minute).Before(time.Now()) {
				invalidateTokenSessionCache(token)
				// Clean up the expired session from Redis
				_ = authtoken.DeleteSession(ctx, rdb, token)
				common.Fail(c, common.CodeUnauthorized, "session.idle_timeout")
				c.Abort()
				return
			}
		}

		// Update activity timestamp in Redis (best-effort, don't fail the request)
		if sessionData.LastActivityAt == 0 || time.Since(time.Unix(sessionData.LastActivityAt, 0)) > time.Minute {
			sessionData.LastActivityAt = time.Now().Unix()
			if err := authtoken.RefreshSessionActivity(ctx, rdb, token, sessionData); err == nil {
				// Update cache with new activity time
				if cached, ok := lookupTokenSessionCache(token); ok {
					cached.data.LastActivityAt = sessionData.LastActivityAt
				}
			}
		}

		// Set Gin context values
		c.Set("userId", sessionData.UserID)
		c.Set("username", sessionData.Username)
		c.Set("roleKeys", sessionData.RoleKeys)
		c.Set("sessionId", sessionData.SessionID)
		if len(sessionData.RoleKeys) > 0 {
			c.Set("roleKey", sessionData.RoleKeys[0])
		}

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
