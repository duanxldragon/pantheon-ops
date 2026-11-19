package deploy

import (
	"context"
	"errors"
	"testing"

	serviceowner "github.com/duanxldragon/pantheon-base/backend/modules/business/service"
	"gorm.io/gorm"
)

// Independent review regressions. Loaded by Go overlay only; deploy sources stay unchanged.
func TestIndependentO2CancelBeforeFirstClaimSettles(t *testing.T) {
	db, a, b, task, host := ownershipFixture(t)
	if err := a.deploymentTransaction(func(tx *gorm.DB) error {
		return acquireHostLease(tx, host.HostID, task.ID, deployHostLeaseOwner(task.ID), task.CreatedAt)
	}); err != nil {
		t.Fatal(err)
	}
	current, err := a.GetTask(task.ID, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := a.ControlTask(context.Background(), task.ID, "independent-review", nil, "cancel", TaskCommandRequest{StateVersion: current.StateVersion}); err != nil {
		t.Fatal(err)
	}
	if _, err := b.ReconcileDeployQueue("independent-review"); err != nil {
		t.Fatal(err)
	}
	if err := db.First(&task, task.ID).Error; err != nil {
		t.Fatal(err)
	}
	var leases int64
	if err := db.Model(&DeployHostLease{}).Where("task_id = ?", task.ID).Count(&leases).Error; err != nil {
		t.Fatal(err)
	}
	t.Logf("after cancel + recovery: task=%s retained physical leases=%d", task.Status, leases)
	if task.Status != TaskStatusCanceled || leases != 0 {
		t.Fatalf("cancellation with no worker claim never settles/releases: task=%s leases=%d", task.Status, leases)
	}
}

func TestIndependentO2ServiceProjectionRollsBackWithFinish(t *testing.T) {
	db, a, _, task, host := ownershipFixture(t)
	manager := serviceowner.NewManager(db, serviceowner.Dependencies{})
	if err := manager.Migrate(); err != nil {
		t.Fatal(err)
	}
	app := serviceowner.Application{Code: "independent-app", Name: "Independent app", BusinessScopeID: task.BusinessScopeID, Status: "active"}
	if err := db.Create(&app).Error; err != nil {
		t.Fatal(err)
	}
	service := serviceowner.Service{ApplicationID: app.ID, Code: "independent-service", Name: "Independent service", RuntimeType: "vm", Status: "active"}
	if err := db.Create(&service).Error; err != nil {
		t.Fatal(err)
	}
	instance := serviceowner.ServiceInstance{ServiceID: service.ID, Environment: "review", TargetType: "vm", HostID: host.HostID, DesiredState: "stopped", ObservedState: "installing", HealthState: "unknown", Status: "active"}
	if err := db.Create(&instance).Error; err != nil {
		t.Fatal(err)
	}
	task.ServiceID, task.ServiceInstanceID = service.ID, instance.ID
	setLinkedFixtureSnapshot(t, &task, host, instance)
	if err := db.Save(&task).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.First(&task, task.ID).Error; err != nil {
		t.Fatal(err)
	}
	a.SetServiceInstanceStateApplierWithTransaction(manager, manager.WithServiceInstanceStateTransaction)
	a.SetServiceReader(manager)
	inner, ctx := ownedClaim(t, a, task, host, "independent-finish")
	before := ownershipFacts(t, db)
	injected := errors.New("independent injected outer commit failure")
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	isolated := db.Session(&gorm.Session{NewDB: true}).WithContext(context.Background())
	isolated.Statement.ConnPool = ownershipCommitPool{ConnPool: sqlDB, sqlDB: sqlDB, fail: injected}
	inner.db = isolated
	err = inner.completeOwnedAttempt("success", nil, "independent-review", false)
	if !errors.Is(err, injected) {
		t.Fatalf("did not reach injected outer commit failure: %v", err)
	}
	if !errors.Is(context.Cause(ctx), injected) {
		t.Fatalf("failure cause lost: %v", context.Cause(ctx))
	}
	if ownershipFacts(t, db) != before {
		t.Fatal("deploy/CMDB facts did not rollback")
	}
	if err := db.First(&instance, instance.ID).Error; err != nil {
		t.Fatal(err)
	}
	t.Logf("outer finish failed and deploy/CMDB rolled back; actual service instance=%s version=%s lifecycle=%d", instance.ObservedState, instance.CurrentVersion, instance.LifecycleVersion)
	if instance.ObservedState != "installing" || instance.CurrentVersion != "" || instance.LifecycleVersion != 0 {
		t.Fatalf("service instance survived failed finish: observed=%s version=%s lifecycle=%d", instance.ObservedState, instance.CurrentVersion, instance.LifecycleVersion)
	}
}
