package middleware

import (
	"context"
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
			val, _ := database.RDB.Get(context.Background(), key).Result()
			if val != "" {
				common.Fail(c, common.CodeUnauthorized, "token.expired.force")
				c.Abort()
				return
			}
		}

		if database.DB != nil && claims.SessionID != "" {
			var session struct {
				LastActivityAt   *time.Time `gorm:"column:last_activity_at"`
				LastRefreshAt    *time.Time `gorm:"column:last_refresh_at"`
				CreatedAt        time.Time  `gorm:"column:created_at"`
				RefreshExpiresAt time.Time  `gorm:"column:refresh_expires_at"`
			}
			now := time.Now()
			err := database.DB.Table("system_user_session").
				Select("last_activity_at, last_refresh_at, created_at, refresh_expires_at").
				Where("session_id = ? AND user_id = ? AND revoked_at IS NULL AND refresh_expires_at > ?", claims.SessionID, claims.UserID, now).
				Take(&session).Error
			if err != nil {
				common.Fail(c, common.CodeUnauthorized, "session.invalid")
				c.Abort()
				return
			}
			lastActivityAt := session.LastActivityAt
			if lastActivityAt == nil {
				lastActivityAt = session.LastRefreshAt
			}
			if lastActivityAt == nil {
				lastActivityAt = &session.CreatedAt
			}
			if lastActivityAt != nil {
				idleMinutes := loadSessionIdleMinutes()
				if idleMinutes > 0 && lastActivityAt.Add(time.Duration(idleMinutes)*time.Minute).Before(now) {
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
