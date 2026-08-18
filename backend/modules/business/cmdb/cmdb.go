package cmdb

import (
	bizcap "pantheon-base/modules/business/capability"
	"pantheon-base/modules/business/cmdb/group"
	"pantheon-base/modules/business/cmdb/host"
	"pantheon-base/modules/business/cmdb/label"
	"pantheon-base/pkg/contracts"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const (
	cmdbRoutePath = "/business/cmdb"
	idWhereClause = "id = ?"
	idAscOrder    = "id ASC"
)

func InitCmdbModule(r *gin.RouterGroup, db *gorm.DB, readers ...bizcap.BizScopeReader) *host.HostService {
	var bizScopeReader bizcap.BizScopeReader
	if len(readers) > 0 {
		bizScopeReader = readers[0]
	}
	hostSvc := host.NewHostService(db, bizScopeReader)
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
				cmdb := contracts.DataScopedGroup(r, cmdbRoutePath, db)
				hostHandler.RegisterRoutes(cmdb)
			},
		},
		contracts.FuncModule{
			ModuleName:    "business.cmdb.group",
			MigrateFunc:   func(db *gorm.DB) error { return groupSvc.Migrate() },
			SeedMenusFunc: seedGroupMenus,
			SeedI18nFunc:  seedGroupI18n,
			Register: func(r *gin.RouterGroup) {
				cmdb := contracts.DataScopedGroup(r, cmdbRoutePath, db)
				groupHandler.RegisterRoutes(cmdb)
			},
		},
		contracts.FuncModule{
			ModuleName:    "business.cmdb.label",
			MigrateFunc:   func(db *gorm.DB) error { return labelSvc.Migrate() },
			SeedMenusFunc: seedLabelMenus,
			SeedI18nFunc:  seedLabelI18n,
			Register: func(r *gin.RouterGroup) {
				cmdb := contracts.ProtectedGroup(r, cmdbRoutePath)
				labelHandler.RegisterRoutes(cmdb)
			},
		},
	}

	contracts.RegisterBackendModules(r, db, modules...)
	return hostSvc
}
