package service

import (
	"pantheon-base/pkg/contracts"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const serviceModuleKey = "business.service"

// InitServiceModule registers the service business module and returns its reader capability.
func InitServiceModule(r *gin.RouterGroup, db *gorm.DB, deps Dependencies) Reader {
	manager := NewManager(db, deps)
	handler := NewHandler(manager)
	contracts.RegisterBackendModules(r, db, contracts.FuncModule{
		ModuleName:    serviceModuleKey,
		MigrateFunc:   func(_ *gorm.DB) error { return manager.Migrate() },
		SeedMenusFunc: seedMenus,
		SeedI18nFunc:  seedI18n,
		Register: func(r *gin.RouterGroup) {
			group := contracts.DataScopedGroup(r, "/business/service", db)
			handler.RegisterRoutes(group)
		},
	})
	return manager
}

var _ Reader = (*Manager)(nil)
var _ Command = (*Manager)(nil)
