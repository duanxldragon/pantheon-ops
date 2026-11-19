package deploy

import (
	"log/slog"
	"runtime/debug"

	bizscope "github.com/duanxldragon/pantheon-base/backend/modules/business/bizscope"
	bizcap "github.com/duanxldragon/pantheon-base/backend/modules/business/capability"
	"github.com/duanxldragon/pantheon-base/backend/modules/business/cmdb"
	"github.com/duanxldragon/pantheon-base/backend/modules/business/service"
	"github.com/duanxldragon/pantheon-base/backend/pkg/contracts"
	"github.com/duanxldragon/pantheon-base/backend/pkg/database"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// InitDeployModule registers the deploy business module.
func InitDeployModule(r *gin.RouterGroup, db *gorm.DB, cmdbCapability cmdb.DeployCMDBCapability, readers ...bizcap.BizScopeReader) {
	initDeployModule(r, db, cmdbCapability, nil, nil, nil, readers...)
}

// InitDeployModuleWithServiceState registers deploy with state callbacks and
// their same-owner transaction factory, configured before reconciliation starts.
func InitDeployModuleWithServiceState(r *gin.RouterGroup, db *gorm.DB, cmdbCapability cmdb.DeployCMDBCapability, stateApplier bizcap.ServiceInstanceStateApplier, stateTransaction func(*gorm.DB) bizcap.ServiceInstanceStateApplier, readers ...bizcap.BizScopeReader) {
	initDeployModule(r, db, cmdbCapability, nil, stateApplier, stateTransaction, readers...)
}

// InitDeployModuleWithServiceOwner connects admission, state commands and the
// reverse deletion guard before any queue recovery can access the owners.
func InitDeployModuleWithServiceOwner(r *gin.RouterGroup, db *gorm.DB, cmdbCapability cmdb.DeployCMDBCapability, serviceReader service.Reader, stateApplier bizcap.ServiceInstanceStateApplier, stateTransaction func(*gorm.DB) bizcap.ServiceInstanceStateApplier, readers ...bizcap.BizScopeReader) {
	initDeployModule(r, db, cmdbCapability, serviceReader, stateApplier, stateTransaction, readers...)
}

func initDeployModule(r *gin.RouterGroup, db *gorm.DB, cmdbCapability cmdb.DeployCMDBCapability, serviceReader service.Reader, stateApplier bizcap.ServiceInstanceStateApplier, stateTransaction func(*gorm.DB) bizcap.ServiceInstanceStateApplier, readers ...bizcap.BizScopeReader) {
	if err := registerDeployLogPolicies(); err != nil {
		panic(err)
	}
	if err := registerDeployPermissionRoutes(); err != nil {
		panic(err)
	}
	var bizScopeReader bizcap.BizScopeReader
	if len(readers) > 0 {
		bizScopeReader = readers[0]
	}
	if bizScopeReader == nil {
		bizScopeReader = bizscope.NewService(db)
	}
	svc := NewDeployService(db, cmdbCapability, bizScopeReader)
	svc.SetAsyncExecution(true)
	svc.SetServiceInstanceStateApplierWithTransaction(stateApplier, stateTransaction)
	svc.SetServiceReader(serviceReader)
	if owner, ok := serviceReader.(interface {
		SetDeploymentReferenceReader(bizcap.DeploymentReferenceReader)
	}); ok {
		owner.SetDeploymentReferenceReader(svc)
	}
	// Recover leases and queued work from a prior process before accepting new
	// deploy starts. Each host is still claimed atomically by the worker.
	go func() {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("deploy queue reconciliation panic recovered",
					"panic", r,
					"stack", string(debug.Stack()),
				)
			}
		}()
		if _, err := svc.ReconcileDeployQueue("system"); err != nil {
			slog.Error("deploy queue reconciliation failed", "error", err)
		}
	}()
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
				logDownloads, err := contracts.TenantScopedBusinessGroup(r, "/business/deploy", contracts.BusinessTenantGroupOptions{DB: db, RDB: database.RDB, DataScope: true, PlatformScope: true})
				if err != nil {
					panic(err)
				}
				handler.RegisterLogDownloadRoutes(logDownloads)
			},
		},
	}

	contracts.RegisterBackendModules(r, db, modules...)
}
