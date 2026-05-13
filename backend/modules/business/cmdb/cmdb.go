package cmdb

import (
	"pantheon-ops/backend/modules/business/cmdb/group"
	"pantheon-ops/backend/modules/business/cmdb/host"
	"pantheon-ops/backend/modules/business/cmdb/label"
	"pantheon-ops/backend/pkg/contracts"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func InitCmdbModule(r *gin.RouterGroup, db *gorm.DB) {
	hostSvc := host.NewHostService(db)
	hostHandler := host.NewHostHandler(hostSvc)

	groupSvc := group.NewGroupService(db)
	groupHandler := group.NewGroupHandler(groupSvc)

	labelSvc := label.NewLabelService(db)
	labelHandler := label.NewLabelHandler(labelSvc)

	modules := []contracts.BackendModule{
		contracts.FuncModule{
			ModuleName:    "business.cmdb.host",
			MigrateFunc:   func(db *gorm.DB) error { return hostSvc.Migrate() },
			SeedMenusFunc: seedHostMenus,
			SeedI18nFunc:  seedHostI18n,
			Register: func(r *gin.RouterGroup) {
				cmdb := contracts.DataScopedGroup(r, "/business/cmdb", db)
				hostHandler.RegisterRoutes(cmdb)
			},
		},
		contracts.FuncModule{
			ModuleName:    "business.cmdb.group",
			MigrateFunc:   func(db *gorm.DB) error { return groupSvc.Migrate() },
			SeedMenusFunc: seedGroupMenus,
			SeedI18nFunc:  seedGroupI18n,
			Register: func(r *gin.RouterGroup) {
				cmdb := contracts.DataScopedGroup(r, "/business/cmdb", db)
				groupHandler.RegisterRoutes(cmdb)
			},
		},
		contracts.FuncModule{
			ModuleName:    "business.cmdb.label",
			MigrateFunc:   func(db *gorm.DB) error { return labelSvc.Migrate() },
			SeedMenusFunc: seedLabelMenus,
			SeedI18nFunc:  seedLabelI18n,
			Register: func(r *gin.RouterGroup) {
				cmdb := contracts.ProtectedGroup(r, "/business/cmdb")
				labelHandler.RegisterRoutes(cmdb)
			},
		},
	}

	contracts.RegisterBackendModules(r, db, modules...)
}
