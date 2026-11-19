// Package opsinit is the Ops business overlay composition root. It replaces
// the historical hand-edited business_overlay_registry.go: the rebuild
// generator emits a single opsinit.Init(r, db) call and this file owns all
// cross-module wiring, including the business-scoped idempotency middleware
// that used to be registered globally in the (base-owned) server main.go.
package opsinit

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	bizscope "github.com/duanxldragon/pantheon-base/backend/modules/business/bizscope"
	bizcap "github.com/duanxldragon/pantheon-base/backend/modules/business/capability"
	cicd "github.com/duanxldragon/pantheon-base/backend/modules/business/cicd"
	"github.com/duanxldragon/pantheon-base/backend/modules/business/cicd/argocd"
	cmdb "github.com/duanxldragon/pantheon-base/backend/modules/business/cmdb"
	deploy "github.com/duanxldragon/pantheon-base/backend/modules/business/deploy"
	filedist "github.com/duanxldragon/pantheon-base/backend/modules/business/filedist"
	k8s "github.com/duanxldragon/pantheon-base/backend/modules/business/k8s"
	observability "github.com/duanxldragon/pantheon-base/backend/modules/business/observability"
	opsservice "github.com/duanxldragon/pantheon-base/backend/modules/business/service"

	"github.com/duanxldragon/pantheon-base/backend/pkg/database"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Preflight is the read-only startup adapter registered by the canonical
// overlay generator before Base business retirement cleanup can have effects.
func Preflight(db *gorm.DB) error {
	_, err := observability.QueryWorkspaceMaintenanceReady(db)
	return err
}

// Init wires all Ops business modules into the business router group.
func Init(r *gin.RouterGroup, db *gorm.DB) {
	if err := Preflight(db); err != nil {
		panic(err)
	}
	if err := observability.RegisterQueryPolicies(); err != nil {
		panic(err)
	}
	if database.ShouldAutoMigrate() {
		if err := observability.BootstrapQueryWorkspaceDevelopment(db); err != nil {
			panic(err)
		}
	}
	if err := observability.QueryWorkspaceReady(db); err != nil {
		panic(err)
	}
	// Idempotency protection for business write endpoints. Scoped to the
	// business router group (migrated from the former global main.go
	// registration); requests without X-Idempotency-Key pass through.
	if database.RDB != nil {
		r.Use(NewIdempotencyMiddleware(database.RDB).Handle())
	}

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
	if manager, ok := serviceReader.(*opsservice.Manager); ok && manager != nil {
		deploy.InitDeployModuleWithServiceOwner(r, db, cmdbCapability, serviceReader, manager, manager.WithServiceInstanceStateTransaction, bizScopeService)
	} else if stateApplier, ok := serviceReader.(bizcap.ServiceInstanceStateApplier); ok {
		deploy.InitDeployModuleWithServiceOwner(r, db, cmdbCapability, serviceReader, stateApplier, nil, bizScopeService)
	} else {
		deploy.InitDeployModuleWithServiceOwner(r, db, cmdbCapability, serviceReader, nil, nil, bizScopeService)
	}
	if referenceReader, ok := serviceReader.(bizcap.ServiceInstanceReferenceReader); ok {
		hostService.SetServiceInstanceReferenceReader(referenceReader)
	}

	// Observability module: migrate schema, then register routes.
	observabilityModule := observability.InitObservabilityModule(r, db, bizScopeService)
	if err := observabilityModule.Migrate(db); err != nil {
		panic(err)
	}
	observabilityModule.RegisterRoutes(r)

	// ArgoCD (CICD) module: v1.2 real integration is active — the client wires
	// create/sync/rollback/delete to the ArgoCD API, credentials are encrypted,
	// and routes run under the DataScopedGroup authz chain.
	_ = cicd.InitCICDModule(r, db)

	// File distribution module: migrate schema, then register routes.
	fileDistModule := filedist.InitFileDistModule(r, db, "./uploads")
	fileDistModule.SetTargetSelectionCapability(cmdbCapability)
	if err := fileDistModule.Migrate(db); err != nil {
		panic(err)
	}
	// The locked Base Init adapter has no lifecycle context/cleanup callback.
	// Own cancellation here without changing Base or claiming that its HTTP
	// shutdown waits for business cleanup. Job authorization stays fail-closed
	// until the public current-job/source authority contract is available.
	extensionCtx, cancelExtensions := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	stopExtensions, err := startExtensionRuntime(extensionCtx, fileDistModule, argocd.NewReconciler(db, 30*time.Second))
	if err != nil {
		cancelExtensions()
		panic(err)
	}
	routesRegistered := false
	defer func() {
		if !routesRegistered {
			cancelExtensions()
			stopExtensions()
		}
	}()
	fileDistModule.RegisterRoutes(r)
	routesRegistered = true
	go func() {
		<-extensionCtx.Done()
		stopExtensions()
		cancelExtensions()
	}()
}
