package middleware

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"pantheon-ops/backend/pkg/common"
	"pantheon-ops/backend/pkg/database"

	"github.com/gin-gonic/gin"
)

const defaultSessionIdleMinutes = 30

// Session cache to avoid DB hit on every request.
// Key: "sessionID:userID", Value: cached session info.
// TTL: 60 seconds — balances freshness vs DB load.
var (
	sessionCacheMu sync.RWMutex
	sessionCache    = make(map[string]*cachedSession)
	sessionCacheTTL = 60 * time.Second
)

type cachedSession struct {
	lastActivityAt   *time.Time
	lastRefreshAt    *time.Time
	createdAt        time.Time
	refreshExpiresAt time.Time
	cachedAt         time.Time
}

func sessionCacheKey(sessionID string, userID uint64) string {
	return sessionID + ":" + strconv.FormatUint(userID, 10)
}

func lookupSessionCache(sessionID string, userID uint64) (*cachedSession, bool) {
	sessionCacheMu.RLock()
	defer sessionCacheMu.RUnlock()
	entry, ok := sessionCache[sessionCacheKey(sessionID, userID)]
	if !ok {
		return nil, false
	}
	if time.Since(entry.cachedAt) > sessionCacheTTL {
		return nil, false
	}
	return entry, true
}

func storeSessionCache(sessionID string, userID uint64, cs *cachedSession) {
	cs.cachedAt = time.Now()
	sessionCacheMu.Lock()
	defer sessionCacheMu.Unlock()
	sessionCache[sessionCacheKey(sessionID, userID)] = cs
	// Evict stale entries periodically (lazy eviction)
	if len(sessionCache) > 10000 {
		now := time.Now()
		for k, v := range sessionCache {
			if now.Sub(v.cachedAt) > sessionCacheTTL {
				delete(sessionCache, k)
			}
		}
	}
}

func invalidateSessionCache(sessionID string, userID uint64) {
	sessionCacheMu.Lock()
	defer sessionCacheMu.Unlock()
	delete(sessionCache, sessionCacheKey(sessionID, userID))
}

var (
	sessionIdleMinutesMu       sync.RWMutex
	cachedSessionIdleMinutes   = defaultSessionIdleMinutes
	cachedSessionIdleMinutesAt time.Time
)

func extractToken(c *gin.Context) string {
	if token, err := c.Cookie(common.CookieAccessToken); err == nil && token != "" {
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

// JWTAuthMiddleware 权限校验中间件
func JWTAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := extractToken(c)
		if token == "" {
			common.Fail(c, common.CodeUnauthorized, "token.missing")
			c.Abort()
			return
		}

		claims, err := common.ParseToken(token, common.TokenTypeAccess)
		if err != nil {
			switch {
			case errors.Is(err, common.ErrTokenExpired):
				common.Fail(c, common.CodeUnauthorized, "token.expired")
			case errors.Is(err, common.ErrTokenType):
				common.Fail(c, common.CodeUnauthorized, "token.type.invalid")
			default:
				common.Fail(c, common.CodeUnauthorized, "token.invalid")
			}
			c.Abort()
			return
		}

		if database.RDB != nil {
			key := fmt.Sprintf("blacklist:%d", claims.UserID)
			val, _ := database.RDB.Get(c.Request.Context(), key).Result()
			if val != "" {
				common.Fail(c, common.CodeUnauthorized, "token.expired.force")
				c.Abort()
				return
			}
		}

		if database.DB != nil && claims.SessionID != "" {
			now := time.Now()

			// Try cache first to avoid DB hit on every request.
			var lastActivityAt *time.Time
			var sessionRefreshExpiresAt time.Time
			cached, cacheHit := lookupSessionCache(claims.SessionID, claims.UserID)
			if cacheHit {
				lastActivityAt = cached.lastActivityAt
				if lastActivityAt == nil {
					lastActivityAt = cached.lastRefreshAt
				}
				if lastActivityAt == nil {
					lastActivityAt = &cached.createdAt
				}
				sessionRefreshExpiresAt = cached.refreshExpiresAt
				// Check expiry from cached data
				if sessionRefreshExpiresAt.Before(now) {
					invalidateSessionCache(claims.SessionID, claims.UserID)
					cacheHit = false
				}
			}

			if !cacheHit {
				// Cache miss — query DB
				var session struct {
					LastActivityAt   *time.Time `gorm:"column:last_activity_at"`
					LastRefreshAt    *time.Time `gorm:"column:last_refresh_at"`
					CreatedAt        time.Time  `gorm:"column:created_at"`
					RefreshExpiresAt time.Time  `gorm:"column:refresh_expires_at"`
				}
				err := database.DB.Table("system_user_session").
					Select("last_activity_at, last_refresh_at, created_at, refresh_expires_at").
					Where("session_id = ? AND user_id = ? AND revoked_at IS NULL AND refresh_expires_at > ?", claims.SessionID, claims.UserID, now).
					Take(&session).Error
				if err != nil {
					common.Fail(c, common.CodeUnauthorized, "session.invalid")
					c.Abort()
					return
				}
				lastActivityAt = session.LastActivityAt
				if lastActivityAt == nil {
					lastActivityAt = session.LastRefreshAt
				}
				if lastActivityAt == nil {
					lastActivityAt = &session.CreatedAt
				}
				// Store in cache for subsequent requests
				storeSessionCache(claims.SessionID, claims.UserID, &cachedSession{
					lastActivityAt:   session.LastActivityAt,
					lastRefreshAt:    session.LastRefreshAt,
					createdAt:        session.CreatedAt,
					refreshExpiresAt: session.RefreshExpiresAt,
				})
			}

			if lastActivityAt != nil {
				idleMinutes := loadSessionIdleMinutes()
				if idleMinutes > 0 && lastActivityAt.Add(time.Duration(idleMinutes)*time.Minute).Before(now) {
					invalidateSessionCache(claims.SessionID, claims.UserID)
					_ = database.DB.Table("system_user_session").
						Where("session_id = ? AND user_id = ? AND revoked_at IS NULL", claims.SessionID, claims.UserID).
						Update("revoked_at", now).Error
					common.Fail(c, common.CodeUnauthorized, "session.idle_timeout")
					c.Abort()
					return
				}
			}
		}

		c.Set("userId", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("roleKeys", claims.RoleKeys)
		c.Set("sessionId", claims.SessionID)
		if len(claims.RoleKeys) > 0 {
			c.Set("roleKey", claims.RoleKeys[0])
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
