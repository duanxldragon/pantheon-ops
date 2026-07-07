package auth

import (
	"os"
	"strconv"
	"strings"
	"time"

	"pantheon-ops/backend/internal/middleware"
	"pantheon-ops/backend/modules/auth/login"
	"pantheon-ops/backend/pkg/contracts"
	"pantheon-ops/backend/pkg/database"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func InitAuthModule(r *gin.RouterGroup, db *gorm.DB) {
	authSvc := login.NewRuntime(db)
	authHandler := login.NewAuthHandler(authSvc)
	loginRateLimiter := middleware.RateLimiter(middleware.RateLimiterConfig{
		MaxRequests: publicAuthRateLimitMax(5, 120),
		Window:      time.Minute,
		KeyFunc:     publicAuthRateLimitKey,
		Store:       middleware.NewRedisRateLimitStore(),
	})
	mfaRateLimiter := middleware.RateLimiter(middleware.RateLimiterConfig{
		MaxRequests: publicAuthRateLimitMax(10, 120),
		Window:      time.Minute,
		KeyFunc:     publicAuthRateLimitKey,
		Store:       middleware.NewRedisRateLimitStore(),
	})
	refreshRateLimiter := middleware.RateLimiter(middleware.RateLimiterConfig{
		MaxRequests: publicAuthRateLimitMax(10, 120),
		Window:      time.Minute,
		KeyFunc:     publicAuthRateLimitKey,
		Store:       middleware.NewRedisRateLimitStore(),
	})

	contracts.RegisterRuntimeSettingReloader("system/auth", authSvc.ReloadSettings)

	// 监听核心设置变更
	authSvc.WatchSettings()

	modules := []contracts.BackendModule{
		contracts.FuncModule{
			ModuleName:    "auth",
			MigrateFunc:   func(_ *gorm.DB) error { return authSvc.Migrate() },
			SeedMenusFunc: login.SeedAuthModuleMenus,
			Register: func(r *gin.RouterGroup) {
				sys := r.Group("/system")
				{
					sys.POST("/login", loginRateLimiter, authHandler.LoginHandler)
					sys.POST("/refresh", refreshRateLimiter, authHandler.RefreshTokenHandler)
				}

				apiAuth := r.Group("/auth")
				{
					apiAuth.POST("/login", loginRateLimiter, authHandler.LoginHandler)
					apiAuth.POST("/mfa/verify", mfaRateLimiter, authHandler.VerifyMFAHandler)
					apiAuth.POST("/refresh", refreshRateLimiter, authHandler.RefreshTokenHandler)
				}

				systemProtected := r.Group("/system").Use(middleware.TokenAuthMiddleware(database.RDB)).Use(middleware.CasbinMiddleware())
				{
					systemProtected.POST("/logout", authHandler.LogoutHandler)
					systemProtected.GET("/user/info", authHandler.GetCurrentUserInfo)
					systemProtected.PUT("/profile/password", authHandler.UpdatePassword)
					systemProtected.GET("/login-log/list", authHandler.GetLoginLogList)
					systemProtected.POST("/login-log/export", authHandler.ExportLoginLogs)
					systemProtected.POST("/login-log/cleanup", middleware.SecureActionMiddleware(), authHandler.CleanupLoginLogs)
					systemProtected.POST("/login-log/batch-delete", middleware.SecureActionMiddleware(), authHandler.BatchDeleteLoginLogs)
					systemProtected.GET("/security-event/list", authHandler.GetSecurityEventList)
					systemProtected.POST("/security-event/:id/acknowledge", middleware.SecureActionMiddleware(), authHandler.AcknowledgeSecurityEvent)
					systemProtected.GET("/session/list", authHandler.GetSessionList)
					systemProtected.POST("/session/cleanup", middleware.SecureActionMiddleware(), authHandler.CleanupHistoricSessions)
					systemProtected.POST("/session/batch-revoke", middleware.SecureActionMiddleware(), authHandler.BatchRevokeSessions)
					systemProtected.DELETE("/session/:id", authHandler.RevokeAnySession)
				}

				authV2 := r.Group("/auth").Use(middleware.TokenAuthMiddleware(database.RDB)).Use(middleware.CasbinMiddleware())
				{
					authV2.POST("/logout", authHandler.LogoutHandler)
					authV2.POST("/activity", authHandler.TouchActivity)
					authV2.POST("/operation-verify", authHandler.VerifyOperationPassword)
					authV2.GET("/me", authHandler.GetCurrentUserInfo)
					authV2.PUT("/me/preferences", authHandler.UpdateCurrentUserPreferences)
					authV2.GET("/security", authHandler.GetSecurityOverview)
					authV2.PUT("/password", authHandler.UpdatePassword)
					authV2.GET("/sessions", authHandler.GetSessions)
					authV2.DELETE("/sessions/:id", authHandler.RevokeSession)
					authV2.GET("/login-logs", authHandler.GetOwnLoginLogs)
				}
			},
		},
	}

	contracts.RegisterBackendModules(r, db, modules...)
}

func publicAuthRateLimitKey(c *gin.Context) string {
	return c.FullPath() + ":" + c.ClientIP()
}

func publicAuthRateLimitMax(productionDefault int, nonProductionDefault int) int {
	if override := strings.TrimSpace(os.Getenv("PANTHEON_PUBLIC_AUTH_RATE_LIMIT_MAX")); override != "" {
		value, err := strconv.Atoi(override)
		if err == nil && value > 0 {
			return value
		}
	}

	env := strings.ToLower(strings.TrimSpace(os.Getenv("PANTHEON_ENV")))
	if env == "production" {
		return productionDefault
	}
	return nonProductionDefault
}
