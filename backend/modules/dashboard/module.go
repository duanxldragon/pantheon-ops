package dashboard

import (
	"pantheon-ops/backend/internal/middleware"
	"pantheon-ops/backend/pkg/contracts"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func InitDashboardModule(r *gin.RouterGroup, db *gorm.DB) {
	dashboardSvc := NewDashboardService(db)
	dashboardHandler := NewDashboardHandler(dashboardSvc)

	modules := []contracts.BackendModule{
		contracts.FuncModule{
			ModuleName: "dashboard",
			Register: func(r *gin.RouterGroup) {
				r.Group("/platform").Use(middleware.JWTAuthMiddleware()).GET("/dashboard/summary", dashboardHandler.GetSummary)
			},
		},
	}

	contracts.RegisterBackendModules(r, db, modules...)
}
