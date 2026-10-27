package platform

import (
	"github.com/duanxldragon/pantheon-base/backend/internal/middleware"
	"github.com/duanxldragon/pantheon-base/backend/pkg/database"
	"github.com/duanxldragon/pantheon-base/backend/pkg/tenant"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// RegisterPlatformRoutes wires platform routes. The org governance task loader is
// injected by the composition root (cmd/server) so this module never imports the
// system/org/dept implementation directly (REPOSITORY_LAYOUT.md §8.2).
func RegisterPlatformRoutes(r *gin.RouterGroup, db *gorm.DB, orgGovernanceTaskLoader OrgGovernanceTaskLoader) {
	tokenMiddleware := middleware.TokenAuthMiddleware(database.RDB)
	tenantModeLoader := tenant.NewModeLoader(func() string {
		var value string
		if db == nil || db.Table("system_setting").Where("setting_key = ?", "platform.tenant_mode").Pluck("setting_value", &value).Error != nil {
			return tenant.ModeCompat
		}
		return value
	}, int64(5*time.Second))

	dashboardSvc := NewDashboardService(db, WithOrgGovernanceTaskLoader(orgGovernanceTaskLoader))
	dashboardHandler := NewDashboardHandler(dashboardSvc)

	// The dashboard summary aggregates console-wide counts, so it is a
	// management surface: it requires an explicit permission grant on top of
	// authentication (contract §3.3). `admin` holds `/api/v1/*` GET and tenant
	// administrators hold it through role:<key>@tenant:<id>; a subject with no
	// grant is denied instead of reading platform-wide statistics.
	dashboardGroup := r.Group("/dashboard").
		Use(tokenMiddleware).
		Use(middleware.PlatformScopeTenantContextMiddleware(tenantModeLoader, db)).
		Use(middleware.CasbinMiddleware())
	{
		dashboardGroup.GET("/summary", dashboardHandler.GetSummary)
	}

	RegisterHealthRoutes(r, db)
}
