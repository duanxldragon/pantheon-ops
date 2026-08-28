package business

import (
	bizscope "pantheon-base/modules/business/bizscope"
	bizcap "pantheon-base/modules/business/capability"
	// ArgoCD (CICD) module is disabled in v1.0 (shell implementation); the
	// import stays commented until the v1.2 real integration lands.
	// cicd "pantheon-base/modules/business/cicd"
	cmdb "pantheon-base/modules/business/cmdb"
	deploy "pantheon-base/modules/business/deploy"
	filedist "pantheon-base/modules/business/filedist"
	k8s "pantheon-base/modules/business/k8s"
	observability "pantheon-base/modules/business/observability"
	opsservice "pantheon-base/modules/business/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func initOverlayBusinessModules(r *gin.RouterGroup, db *gorm.DB) {
	cmdbCapability := cmdb.NewDeployCMDBCapability(db)
	bizScopeService := bizscope.InitBizScopeModule(r, db, bizscope.ServiceDependencies{
		HostReader:       cmdbCapability,
		OwnershipCommand: cmdbCapability,
	})
	cmdbCapability.SetBizScopeReader(bizScopeService)
	hostService := cmdb.InitCmdbModule(r, db, bizScopeService)
	k8sReader := k8s.InitK8sModule(r, db, hostService, bizScopeService)
	serviceReader := opsservice.InitServiceModule(r, db, opsservice.Dependencies{
		BizScopeReader: bizScopeService,
		CMDBReader:     cmdbCapability,
		K8sReader:      k8sReader,
	})
	if stateCommand, ok := serviceReader.(bizcap.ServiceInstanceStateCommand); ok {
		deploy.InitDeployModuleWithServiceState(r, db, cmdbCapability, stateCommand, bizScopeService)
	} else {
		deploy.InitDeployModule(r, db, cmdbCapability, bizScopeService)
	}
	if referenceReader, ok := serviceReader.(bizcap.ServiceInstanceReferenceReader); ok {
		hostService.SetServiceInstanceReferenceReader(referenceReader)
	}

	// Initialize observability module
	observabilityModule := observability.InitObservabilityModule(r, db, bizScopeService)
	if err := observabilityModule.Migrate(db); err != nil {
		panic(err)
	}
	observabilityModule.RegisterRoutes(r)

	// ArgoCD (CICD) module is disabled in v1.0: the current implementation is
	// a shell (sync fakes Succeeded, connection test always returns true). It
	// ships as code for the v1.2 real integration (BUSINESS_ARGOCD_MODULE_DESIGN.md)
	// but its menu, routes and permission seeds are removed until then.
	// _ = cicd.InitCICDModule(r, db)

	// Initialize file distribution module
	fileDistModule := filedist.InitFileDistModule(r, db, "./uploads")
	if err := fileDistModule.Migrate(db); err != nil {
		panic(err)
	}
	fileDistModule.RegisterRoutes(r)
}
