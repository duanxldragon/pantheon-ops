package deploy

import (
	"pantheon-ops/backend/internal/middleware"
	"pantheon-ops/backend/pkg/contracts"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func InitDeployModule(r *gin.RouterGroup, db *gorm.DB) {
	svc := NewDeployService(db)
	handler := NewDeployHandler(svc)

	modules := []contracts.BackendModule{
		contracts.FuncModule{
			ModuleName:    "business.deploy",
			MigrateFunc:   func(db *gorm.DB) error { return svc.Migrate() },
			SeedMenusFunc: seedDeployMenus,
			SeedI18nFunc:  seedDeployI18n,
			Register: func(r *gin.RouterGroup) {
				deploy := r.Group("/business/deploy").
					Use(middleware.JWTAuthMiddleware()).
					Use(middleware.CasbinMiddleware()).
					Use(middleware.DataScopeMiddleware(db))
				handler.RegisterRoutes(deploy)
			},
		},
	}

	contracts.RegisterBackendModules(r, db, modules...)
}
