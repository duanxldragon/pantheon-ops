package auth

import (
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/duanxldragon/pantheon-base/backend/internal/middleware"
	"github.com/duanxldragon/pantheon-base/backend/modules/auth/login"
	"github.com/duanxldragon/pantheon-base/backend/pkg/contracts"
	"github.com/duanxldragon/pantheon-base/backend/pkg/contracts/authuser"
	"github.com/duanxldragon/pantheon-base/backend/pkg/database"
	"github.com/duanxldragon/pantheon-base/backend/pkg/maintenance"
	"github.com/duanxldragon/pantheon-base/backend/pkg/tenant"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// InitAuthModule wires the auth domain. userRepo is the credential port over
// system user state, built by the composition root
// (backend/cmd/server/main.go) and injected here so auth never imports
// modules/system/iam/user (docs/designs/REPOSITORY_LAYOUT.md §8.2).
func InitAuthModule(r *gin.RouterGroup, db *gorm.DB, userRepo authuser.Repository) {
	tenantModeLoader := tenant.NewModeLoader(func() string {
		// Pin the read to the platform-global row (tenant_id = 0): the flag is
		// platform-scoped, and an unfiltered lookup would resolve from an
		// arbitrary tenant's settings copy once tenants exist.
		return tenant.FeatureFlagSettingKeyReader(db)
	}, int64(5*time.Second))
	authSvc := login.NewRuntime(db, userRepo)
	authHandler := login.NewAuthHandler(authSvc, middleware.InvalidateTokenAuthCache)
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

	// Auth-domain maintenance (session inventory governance, login-log and
	// security-event retention) runs on the background maintenance runner, not
	// on list/export request paths (task 2026-09-22-request-path-maintenance).
	authSvc.RegisterMaintenanceTasks(maintenance.Default())

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
					// Login tenant picker (slice 2, contract §3.1): lists active
					// memberships for an authenticated subject. Compat mode and
					// errors return an empty list (fail closed, UI stays hidden).
					apiAuth.GET("/login-tenants", middleware.TokenAuthMiddleware(database.RDB), authHandler.GetLoginTenantCandidates)
				}

				systemProtected := r.Group("/system").Use(middleware.TokenAuthMiddleware(database.RDB)).Use(middleware.PlatformScopeTenantContextMiddleware(tenantModeLoader, db)).Use(middleware.CasbinMiddleware())
				{
					systemProtected.POST("/logout", authHandler.LogoutHandler)
					systemProtected.GET("/user/info", authHandler.GetCurrentUserInfo)
					systemProtected.PUT("/profile/password", authHandler.UpdatePassword)
					systemProtected.GET("/login-log/list", authHandler.GetLoginLogList)
					systemProtected.POST("/login-log/export", authHandler.ExportLoginLogs)
					systemProtected.POST("/login-log/cleanup", middleware.SecureActionMiddleware(), authHandler.CleanupLoginLogs)
					systemProtected.POST("/login-log/batch-delete", middleware.SecureActionMiddleware(), authHandler.BatchDeleteLoginLogs)
					systemProtected.GET("/security-event/list", authHandler.GetSecurityEventList)
					systemProtected.POST("/security-event/export", authHandler.ExportSecurityEvents)
					systemProtected.POST("/security-event/:id/acknowledge", middleware.SecureActionMiddleware(), authHandler.AcknowledgeSecurityEvent)
					systemProtected.POST("/security-event/batch-acknowledge", middleware.SecureActionMiddleware(), authHandler.BatchAcknowledgeSecurityEvents)
					systemProtected.POST("/security-event/cleanup", middleware.SecureActionMiddleware(), authHandler.CleanupSecurityEvents)
					systemProtected.GET("/session/list", authHandler.GetSessionList)
					systemProtected.POST("/session/export", authHandler.ExportAdminSessions)
					systemProtected.POST("/session/cleanup", middleware.SecureActionMiddleware(), authHandler.CleanupHistoricSessions)
					systemProtected.POST("/session/batch-revoke", middleware.SecureActionMiddleware(), authHandler.BatchRevokeSessions)
					systemProtected.DELETE("/session/:id", middleware.SecureActionMiddleware(), authHandler.RevokeAnySession)
				}

				authV2 := r.Group("/auth").Use(middleware.TokenAuthMiddleware(database.RDB)).Use(middleware.PlatformScopeTenantContextMiddleware(tenantModeLoader, db)).Use(middleware.CasbinMiddleware())
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

				// Tenant switcher (self-service, contract §2.2). Deliberately NOT
				// Casbin-gated: the authority for moving between tenants is the
				// membership row, which the service checks directly — a Casbin
				// policy would let a permission grant widen tenancy. The tenant
				// context middleware is still required so "current tenant" is the
				// resolved request context rather than a client-supplied value.
				authTenants := r.Group("/auth").Use(middleware.TokenAuthMiddleware(database.RDB)).Use(middleware.PlatformScopeTenantContextMiddleware(tenantModeLoader, db))
				{
					authTenants.GET("/tenants/switchable", authHandler.ListSwitchableTenants)
					authTenants.GET("/tenants/current", authHandler.GetCurrentTenant)
					authTenants.POST("/tenants/switch", authHandler.SwitchTenant)
				}
			},
		},
	}

	contracts.RegisterBackendModules(r, db, modules...)
}

func publicAuthRateLimitKey(c *gin.Context) string {
	return c.FullPath() + ":" + c.ClientIP()
}

func publicAuthRateLimitMax(productionDefault, nonProductionDefault int) int {
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
