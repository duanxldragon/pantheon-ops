package deploy

import (
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
				deploy := contracts.DataScopedGroup(r, "/business/deploy", db)
				handler.RegisterRoutes(deploy)
			},
		},
	}

	contracts.RegisterBackendModules(r, db, modules...)
}
