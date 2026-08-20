package deploy

import (
	bizscope "pantheon-base/modules/business/bizscope"
	bizcap "pantheon-base/modules/business/capability"
	"pantheon-base/modules/business/cmdb"
	"pantheon-base/pkg/contracts"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// InitDeployModule registers the deploy business module.
func InitDeployModule(r *gin.RouterGroup, db *gorm.DB, cmdbCapability cmdb.DeployCMDBCapability, readers ...bizcap.BizScopeReader) {
	initDeployModule(r, db, cmdbCapability, nil, readers...)
}

// InitDeployModuleWithServiceState registers deploy with service-instance state callbacks.
func InitDeployModuleWithServiceState(r *gin.RouterGroup, db *gorm.DB, cmdbCapability cmdb.DeployCMDBCapability, stateCommand bizcap.ServiceInstanceStateCommand, readers ...bizcap.BizScopeReader) {
	initDeployModule(r, db, cmdbCapability, stateCommand, readers...)
}

func initDeployModule(r *gin.RouterGroup, db *gorm.DB, cmdbCapability cmdb.DeployCMDBCapability, stateCommand bizcap.ServiceInstanceStateCommand, readers ...bizcap.BizScopeReader) {
	var bizScopeReader bizcap.BizScopeReader
	if len(readers) > 0 {
		bizScopeReader = readers[0]
	}
	if bizScopeReader == nil {
		bizScopeReader = bizscope.NewService(db)
	}
	svc := NewDeployService(db, cmdbCapability, bizScopeReader)
	svc.SetAsyncExecution(true)
	svc.SetServiceInstanceStateCommand(stateCommand)
	// Recover leases and queued work from a prior process before accepting new
	// deploy starts. Each host is still claimed atomically by the worker.
	go func() { _, _ = svc.ReconcileDeployQueue("system") }()
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
