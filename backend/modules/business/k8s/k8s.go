package k8s

import (
	bizcap "pantheon-base/modules/business/capability"
	"pantheon-base/modules/business/k8s/cluster"
	"pantheon-base/modules/business/k8s/configmap"
	"pantheon-base/modules/business/k8s/namespace"
	"pantheon-base/modules/business/k8s/release"
	"pantheon-base/modules/business/k8s/secret"
	"pantheon-base/modules/business/k8s/workload"
	"pantheon-base/pkg/contracts"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const (
	k8sModuleKey               = "business.k8s"
	businessK8sMenu            = "business.k8s.menu"
	operationsK8sClusterMenu   = "operations.k8s.cluster.menu"
	operationsK8sWorkloadMenu  = "operations.k8s.workload.menu"
	operationsK8sReleaseMenu   = "operations.k8s.release.menu"
	k8sClusterPermViewKey      = "business.k8s.cluster.permission.view"
	k8sClusterPermCreateKey    = "business.k8s.cluster.permission.create"
	k8sClusterPermUpdateKey    = "business.k8s.cluster.permission.update"
	k8sClusterPermDeleteKey    = "business.k8s.cluster.permission.delete"
	k8sClusterPermSyncKey      = "business.k8s.cluster.permission.sync"
	k8sClusterPermNodesKey     = "business.k8s.cluster.permission.nodes"
	k8sWorkloadPermViewKey     = "business.k8s.workload.permission.view"
	k8sWorkloadPermUpdateKey   = "business.k8s.workload.permission.update"
	k8sWorkloadPermScaleKey    = "business.k8s.workload.permission.scale"
	k8sWorkloadPermRestartKey  = "business.k8s.workload.permission.restart"
	k8sWorkloadPermLogsKey     = "business.k8s.workload.permission.logs"
	k8sNamespacePermViewKey    = "business.k8s.namespace.permission.view"
	k8sNamespacePermCreateKey  = "business.k8s.namespace.permission.create"
	k8sNamespacePermDeleteKey  = "business.k8s.namespace.permission.delete"
	k8sConfigMapPermViewKey    = "business.k8s.configmap.permission.view"
	k8sConfigMapPermCreateKey  = "business.k8s.configmap.permission.create"
	k8sConfigMapPermUpdateKey  = "business.k8s.configmap.permission.update"
	k8sConfigMapPermDeleteKey  = "business.k8s.configmap.permission.delete"
	k8sSecretPermViewKey       = "business.k8s.secret.permission.view"
	k8sSecretPermCreateKey     = "business.k8s.secret.permission.create"
	k8sSecretPermUpdateKey     = "business.k8s.secret.permission.update"
	k8sSecretPermDeleteKey     = "business.k8s.secret.permission.delete"
	k8sReleasePermViewKey      = "business.k8s.release.permission.view"
	k8sReleasePermCreateKey    = "business.k8s.release.permission.create"
	k8sReleasePermRollbackKey  = "business.k8s.release.permission.rollback"
	k8sReleasePermReconcileKey = "business.k8s.release.permission.reconcile"
	menuPathWhereClause        = "path = ?"
)

const k8sRoutePath = "/business/k8s"

// InitK8sModule registers Kubernetes business modules and returns target capabilities.
func InitK8sModule(r *gin.RouterGroup, db *gorm.DB, readers ...bizcap.BizScopeReader) bizcap.K8sTargetReader {
	clusterSvc := cluster.NewClusterService(db, readers...)
	clusterHandler := cluster.NewClusterHandler(clusterSvc)
	namespaceSvc := namespace.NewNamespaceService(clusterSvc, db)
	namespaceHandler := namespace.NewNamespaceHandler(namespaceSvc)
	workloadSvc := workload.NewWorkloadService(clusterSvc, namespaceSvc)
	workloadHandler := workload.NewWorkloadHandler(workloadSvc)
	releaseSvc := release.NewReleaseService(db, clusterSvc, namespaceSvc)
	clusterSvc.SetReferenceChecker(releaseSvc)
	releaseHandler := release.NewReleaseHandler(releaseSvc)
	configMapSvc := configmap.NewConfigMapService(clusterSvc, namespaceSvc)
	configMapHandler := configmap.NewConfigMapHandler(configMapSvc)
	secretSvc := secret.NewSecretService(clusterSvc, namespaceSvc)
	secretHandler := secret.NewSecretHandler(secretSvc)

	contracts.RegisterBackendModules(r, db, contracts.FuncModule{
		ModuleName:    k8sModuleKey,
		MigrateFunc:   func(_ *gorm.DB) error { return migrateAll(db, clusterSvc, namespaceSvc, releaseSvc) },
		SeedMenusFunc: seedMenus,
		SeedI18nFunc:  seedI18n,
		Register: func(r *gin.RouterGroup) {
			k8s := contracts.DataScopedGroup(r, k8sRoutePath, db)
			clusterHandler.RegisterRoutes(k8s)
			namespaceHandler.RegisterRoutes(k8s)
			workloadHandler.RegisterRoutes(k8s)
			releaseHandler.RegisterRoutes(k8s)
			configMapHandler.RegisterRoutes(k8s)
			secretHandler.RegisterRoutes(k8s)
		},
	})
	return NewTargetReader(clusterSvc)
}

func migrateAll(db *gorm.DB, services ...interface{ Migrate() error }) error {
	for _, svc := range services {
		if err := svc.Migrate(); err != nil {
			return err
		}
	}
	return nil
}
