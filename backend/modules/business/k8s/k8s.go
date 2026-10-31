package k8s

import (
	bizcap "github.com/duanxldragon/pantheon-base/backend/modules/business/capability"
	"github.com/duanxldragon/pantheon-base/backend/modules/business/cmdb/host"
	"github.com/duanxldragon/pantheon-base/backend/modules/business/k8s/backup"
	"github.com/duanxldragon/pantheon-base/backend/modules/business/k8s/cluster"
	"github.com/duanxldragon/pantheon-base/backend/modules/business/k8s/configmap"
	"github.com/duanxldragon/pantheon-base/backend/modules/business/k8s/helm"
	"github.com/duanxldragon/pantheon-base/backend/modules/business/k8s/ingress"
	"github.com/duanxldragon/pantheon-base/backend/modules/business/k8s/install"
	"github.com/duanxldragon/pantheon-base/backend/modules/business/k8s/namespace"
	"github.com/duanxldragon/pantheon-base/backend/modules/business/k8s/rbac"
	"github.com/duanxldragon/pantheon-base/backend/modules/business/k8s/release"
	"github.com/duanxldragon/pantheon-base/backend/modules/business/k8s/secret"
	"github.com/duanxldragon/pantheon-base/backend/modules/business/k8s/service"
	"github.com/duanxldragon/pantheon-base/backend/modules/business/k8s/storage"
	"github.com/duanxldragon/pantheon-base/backend/modules/business/k8s/workload"
	"github.com/duanxldragon/pantheon-base/backend/pkg/contracts"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const (
	k8sModuleKey               = "business.k8s"
	businessK8sMenu            = "business.k8s.menu"
	operationsK8sClusterMenu   = "operations.k8s.cluster.menu"
	operationsK8sWorkloadMenu  = "operations.k8s.workload.menu"
	operationsK8sReleaseMenu   = "operations.k8s.release.menu"
	operationsK8sServiceMenu   = "operations.k8s.service.menu"
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
	k8sSecretPermViewKey       = "business.k8s.secret.permission.view"   //nolint:gosec // permission key, not secret material
	k8sSecretPermCreateKey     = "business.k8s.secret.permission.create" //nolint:gosec // permission key, not secret material
	k8sSecretPermUpdateKey     = "business.k8s.secret.permission.update" //nolint:gosec // permission key, not secret material
	k8sSecretPermDeleteKey     = "business.k8s.secret.permission.delete" //nolint:gosec // permission key, not secret material
	k8sReleasePermViewKey      = "business.k8s.release.permission.view"
	k8sReleasePermCreateKey    = "business.k8s.release.permission.create"
	k8sReleasePermRollbackKey  = "business.k8s.release.permission.rollback"
	k8sReleasePermReconcileKey = "business.k8s.release.permission.reconcile"
	operationsK8sIngressMenu   = "operations.k8s.ingress.menu"
	k8sIngressPermViewKey      = "business.k8s.ingress.permission.view"
	k8sIngressPermCreateKey    = "business.k8s.ingress.permission.create"
	k8sIngressPermUpdateKey    = "business.k8s.ingress.permission.update"
	k8sIngressPermDeleteKey    = "business.k8s.ingress.permission.delete"
	operationsK8sStorageMenu   = "operations.k8s.storage.menu"
	k8sStoragePermViewKey      = "business.k8s.storage.permission.view"
	k8sStoragePermCreateKey    = "business.k8s.storage.permission.create"
	k8sStoragePermUpdateKey    = "business.k8s.storage.permission.update"
	k8sStoragePermDeleteKey    = "business.k8s.storage.permission.delete"
	k8sServicePermViewKey      = "business.k8s.service.permission.view"
	k8sServicePermCreateKey    = "business.k8s.service.permission.create"
	k8sServicePermUpdateKey    = "business.k8s.service.permission.update"
	k8sServicePermDeleteKey    = "business.k8s.service.permission.delete"
	// Velero backup module is disabled in v1.0; the menu/permission keys are
	// retained as constants for the v1.2 re-enablement (unused in v1.0).
	operationsK8sBackupMenu = "operations.k8s.backup.menu"
	k8sBackupPermViewKey    = "business.k8s.backup.permission.view"
	k8sBackupPermCreateKey  = "business.k8s.backup.permission.create"
	k8sBackupPermDeleteKey  = "business.k8s.backup.permission.delete"
	menuPathWhereClause     = "path = ?"
)

const k8sRoutePath = "/business/k8s"

// InitK8sModule registers Kubernetes business modules and returns target capabilities.
func InitK8sModule(r *gin.RouterGroup, db *gorm.DB, hostService *host.HostService, readers ...bizcap.BizScopeReader) bizcap.K8sTargetReader {
	clusterSvc := cluster.NewClusterService(db, readers...)
	clusterHandler := cluster.NewClusterHandler(clusterSvc)
	namespaceSvc := namespace.NewNamespaceService(clusterSvc, db)
	namespaceHandler := namespace.NewNamespaceHandler(namespaceSvc)
	workloadSvc := workload.NewWorkloadService(clusterSvc, namespaceSvc)
	releaseSvc := release.NewReleaseService(db, clusterSvc, namespaceSvc)
	clusterSvc.SetReferenceChecker(releaseSvc)
	releaseHandler := release.NewReleaseHandler(releaseSvc)
	configMapSvc := configmap.NewConfigMapService(clusterSvc, namespaceSvc)
	secretSvc := secret.NewSecretService(clusterSvc, namespaceSvc)
	serviceSvc := service.NewServiceService(clusterSvc, namespaceSvc)

	helmSvc := helm.NewHelmService(db)
	helmHandler := helm.NewHelmHandler(helmSvc)

	ingressSvc := ingress.NewIngressService(db, clusterSvc, namespaceSvc)
	storageSvc := storage.NewStorageService(db, clusterSvc, namespaceSvc)

	// Initialize K8s Install module
	installSvc := install.InitInstallModule(r, db, hostService)

	// Velero Backup module (v1.2 real integration: CRD create/delete, status
	// reconciler, DataScope on reads). Routes are registered under the
	// DataScopedGroup so token + Casbin + data-scope middleware apply.
	backup.InitBackupModule(contracts.DataScopedGroup(r, k8sRoutePath, db), db, clusterSvc, namespaceSvc)

	// Initialize RBAC module
	rbacSvc := rbac.InitK8sRBACModule(r, db, clusterSvc)

	workloadHandler := workload.NewWorkloadHandler(workloadSvc, rbacSvc)
	configMapHandler := configmap.NewConfigMapHandler(configMapSvc, rbacSvc)
	secretHandler := secret.NewSecretHandler(secretSvc, rbacSvc)
	serviceHandler := service.NewServiceHandler(serviceSvc, rbacSvc)
	ingressHandler := ingress.NewIngressHandler(ingressSvc, rbacSvc)
	storageHandler := storage.NewStorageHandler(storageSvc, rbacSvc)

	contracts.RegisterBackendModules(r, db, contracts.FuncModule{
		ModuleName: k8sModuleKey,
		MigrateFunc: func(_ *gorm.DB) error {
			return migrateAll(db, clusterSvc, namespaceSvc, releaseSvc, helmSvc, ingressSvc, storageSvc, rbacSvc, installSvc)
		},
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
			serviceHandler.RegisterRoutes(k8s)
			helmHandler.RegisterRoutes(k8s)
			ingressHandler.RegisterRoutes(k8s)
			storageHandler.RegisterRoutes(k8s)
			// Install and RBAC routes are already registered in their Init functions
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
