package business

import (
	bizscope "pantheon-base/modules/business/bizscope"
	bizcap "pantheon-base/modules/business/capability"
	cmdb "pantheon-base/modules/business/cmdb"
	deploy "pantheon-base/modules/business/deploy"
	k8s "pantheon-base/modules/business/k8s"
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
	k8sReader := k8s.InitK8sModule(r, db, bizScopeService)
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
}
