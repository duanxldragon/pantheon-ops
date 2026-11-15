package deploy

import (
	"context"
	"errors"
	"testing"

	"gorm.io/gorm"
)

func cancelUnclaimedTask(t *testing.T, s *DeployService, task DeployTask) {
	t.Helper()
	current, err := s.GetTask(task.ID, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.ControlTask(context.Background(), task.ID, "test", nil, "cancel", TaskCommandRequest{StateVersion: current.StateVersion}); err != nil {
		t.Fatal(err)
	}
}

func TestCancellationUnclaimedRecoveryRollsBackAndReplays(t *testing.T) {
	db, a, b, task, host := ownershipFixture(t)
	if err := a.deploymentTransaction(func(tx *gorm.DB) error {
		return acquireHostLease(tx, host.HostID, task.ID, deployHostLeaseOwner(task.ID), task.CreatedAt)
	}); err != nil {
		t.Fatal(err)
	}
	cancelUnclaimedTask(t, a, task)
	before := ownershipFacts(t, db)
	injected := errors.New("cancel event refused after lease release")
	if err := db.Callback().Create().Before("gorm:create").Register("cancel_event_failure", func(tx *gorm.DB) {
		if tx.Statement.Table == "biz_deploy_task_event" {
			tx.AddError(injected)
		}
	}); err != nil {
		t.Fatal(err)
	}
	_, err := b.ReconcileDeployQueue("test")
	db.Callback().Create().Remove("cancel_event_failure")
	if !errors.Is(err, injected) {
		t.Fatalf("cancel event failure: %v", err)
	}
	if ownershipFacts(t, db) != before {
		t.Fatal("cancel failure did not roll back host/task/event/lease together")
	}
	if _, err := b.ReconcileDeployQueue("test"); err != nil {
		t.Fatal(err)
	}
	assertCanceledUnclaimed(t, db, task.ID)
	settled := ownershipFacts(t, db)
	if _, err := a.ReconcileDeployQueue("test"); err != nil {
		t.Fatal(err)
	}
	if ownershipFacts(t, db) != settled {
		t.Fatal("cancel recovery replay wrote duplicate facts")
	}
}

func TestCancellationUnclaimedRecoveryWaitsForLiveOwner(t *testing.T) {
	db, a, b, task, host := ownershipFixture(t)
	inner, _ := ownedClaim(t, a, task, host, "live-cancel")
	physical := deployTestHost{Hostname: "never-claimed", IP: "10.40.0.234", BusinessScopeID: task.BusinessScopeID}
	if err := db.Create(&physical).Error; err != nil {
		t.Fatal(err)
	}
	unclaimed := DeployTaskHost{TaskID: task.ID, HostID: physical.ID, Status: TaskHostStatusRunning, BusinessScopeID: task.BusinessScopeID}
	if err := db.Create(&unclaimed).Error; err != nil {
		t.Fatal(err)
	}
	if err := a.deploymentTransaction(func(tx *gorm.DB) error {
		return acquireHostLease(tx, physical.ID, task.ID, deployHostLeaseOwner(task.ID), task.CreatedAt)
	}); err != nil {
		t.Fatal(err)
	}
	cancelUnclaimedTask(t, a, task)
	before := ownershipFacts(t, db)
	if _, err := b.ReconcileDeployQueue("test"); err != nil {
		t.Fatal(err)
	}
	if ownershipFacts(t, db) != before {
		t.Fatal("recovery settled or released leases while a live owner remained")
	}
	if err := inner.completeOwnedAttempt("canceled", nil, "test", false); err != nil {
		t.Fatal(err)
	}
	assertCanceledUnclaimed(t, db, task.ID)
}

func assertCanceledUnclaimed(t *testing.T, db *gorm.DB, taskID uint64) {
	t.Helper()
	var task DeployTask
	if err := db.First(&task, taskID).Error; err != nil {
		t.Fatal(err)
	}
	var hosts []DeployTaskHost
	if err := db.Where("task_id = ?", taskID).Find(&hosts).Error; err != nil {
		t.Fatal(err)
	}
	if task.Status != TaskStatusCanceled || task.FinishedAt == nil {
		t.Fatal("cancel recovery did not settle task")
	}
	for _, h := range hosts {
		if h.Status != TaskHostStatusSkipped || h.FinishedAt == nil {
			t.Fatalf("unclaimed host did not settle: %+v", h)
		}
	}
	var leases, events int64
	if err := db.Model(&DeployHostLease{}).Where("task_id = ?", taskID).Count(&leases).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&DeployTaskEvent{}).Where("task_id = ? AND event_type = ?", taskID, "task.canceled").Count(&events).Error; err != nil {
		t.Fatal(err)
	}
	if leases != 0 || events != 1 {
		t.Fatalf("cancel leases=%d events=%d", leases, events)
	}
}
