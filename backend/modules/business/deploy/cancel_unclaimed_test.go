package deploy

import (
	"context"
	"database/sql"
	"errors"
	"sync"
	"sync/atomic"
	"testing"

	serviceowner "github.com/duanxldragon/pantheon-base/backend/modules/business/service"
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

// Cancellation can win before the resumed worker claims any host. The worker
// must acknowledge the request when its dispatch loop drains, without relying
// on a later queue-recovery pass.
func TestCancellationUnclaimedWorkerDrainSettlesAndReplays(t *testing.T) {
	db, svc, _, task, host := ownershipFixture(t)
	configureUnclaimedSSHWorker(t, svc, task.ID)
	if err := svc.deploymentTransaction(func(tx *gorm.DB) error {
		return acquireHostLease(tx, host.HostID, task.ID, deployHostLeaseOwner(task.ID), task.CreatedAt)
	}); err != nil {
		t.Fatal(err)
	}
	cancelUnclaimedTask(t, svc, task)
	svc.runAsyncTask(task.ID, "test")
	assertCanceledUnclaimed(t, db, task.ID)
	assertWorkerNeverClaimed(t, db, task.ID)
	settled := ownershipFacts(t, db)
	svc.runAsyncTask(task.ID, "test")
	if ownershipFacts(t, db) != settled {
		t.Fatal("worker drain replay changed settled task/host/event/lease facts")
	}
}

func TestCancellationUnclaimedWorkerDrainPreservesSupersededOwner(t *testing.T) {
	db, svc, task, owner, instance := startedServiceLink(t)
	configureUnclaimedSSHWorker(t, svc, task.ID)
	desiredVersion := "new-owner-intent"
	if _, err := owner.UpdateInstance(instance.ID, serviceowner.UpdateInstanceRequest{DesiredVersion: &desiredVersion}, "new-owner", nil); err != nil {
		t.Fatal(err)
	}
	ownerBefore := serviceOwnerFacts(t, db)
	if _, err := svc.CancelTask(task.ID, "test", nil); err != nil {
		t.Fatal(err)
	}
	svc.runAsyncTask(task.ID, "test")
	assertCanceledUnclaimed(t, db, task.ID)
	assertWorkerNeverClaimed(t, db, task.ID)
	if serviceOwnerFacts(t, db) != ownerBefore {
		t.Fatal("worker drain overwrote superseding Service owner facts")
	}
	settled := linkedServiceFacts(t, db)
	svc.runAsyncTask(task.ID, "test")
	if linkedServiceFacts(t, db) != settled {
		t.Fatal("linked worker drain replay changed persisted facts")
	}
}

func configureUnclaimedSSHWorker(t *testing.T, svc *DeployService, taskID uint64) {
	t.Helper()
	credentialID := setupDeployTestCredential(t, svc)
	if err := svc.db.Model(&DeployTask{}).Where("id = ?", taskID).Updates(map[string]any{
		"executor_type": ExecutorTypeSSH, "credential_ref_id": credentialID,
		"credential_ref_version": 1, "ssh_host_fingerprint": "SHA256:unclaimed-worker",
	}).Error; err != nil {
		t.Fatal(err)
	}
	svc.sshRunnerFactory = func(context.Context, cmdbHostSnapshot, StartTaskRequest) (deploySSHRunner, error) {
		return nil, errors.New("canceled unclaimed task must not execute SSH")
	}
}

func assertWorkerNeverClaimed(t *testing.T, db *gorm.DB, taskID uint64) {
	t.Helper()
	var attempts int64
	if err := db.Model(&DeployTaskAttempt{}).Where("task_id = ?", taskID).Count(&attempts).Error; err != nil {
		t.Fatal(err)
	}
	if attempts != 0 {
		t.Fatalf("canceled worker claimed %d attempts", attempts)
	}
}

type workerDrainCommitPool struct {
	ownershipCommitPool
	first *atomic.Bool
}

func (p workerDrainCommitPool) BeginTx(ctx context.Context, opts *sql.TxOptions) (gorm.ConnPool, error) {
	if !p.first.CompareAndSwap(false, true) {
		p.committed = nil
		p.commitReply = nil
	}
	return p.ownershipCommitPool.BeginTx(ctx, opts)
}

func TestCancellationUnclaimedWorkerDrainPauseResumeHandoff(t *testing.T) {
	db, svc, task, _, _ := startedServiceLink(t)
	configureUnclaimedSSHWorker(t, svc, task.ID)
	svc.sshRunnerFactory = func(context.Context, cmdbHostSnapshot, StartTaskRequest) (deploySSHRunner, error) {
		return &fakeDeploySSHRunner{}, nil
	}
	if _, err := svc.PauseTask(task.ID, "test", nil); err != nil {
		t.Fatal(err)
	}
	committed, reply, drained := make(chan struct{}), make(chan struct{}), make(chan struct{})
	releaseReply := sync.OnceFunc(func() { close(reply) })
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	isolated := db.Session(&gorm.Session{NewDB: true}).WithContext(context.Background())
	isolated.Statement.ConnPool = workerDrainCommitPool{
		ownershipCommitPool: ownershipCommitPool{ConnPool: sqlDB, sqlDB: sqlDB, committed: committed, commitReply: reply},
		first:               &atomic.Bool{},
	}
	svc.db = isolated
	go func() { defer close(drained); svc.runAsyncTask(task.ID, "test") }()
	t.Cleanup(func() {
		releaseReply()
		waitOwnership(t, drained, "paused worker cleanup join")
		svc.taskWorkers.Delete(task.ID)
	})
	// The pause is visible to Resume, but the original worker has not received
	// its commit reply. Dispatch the replacement directly to fix the ordering.
	waitOwnership(t, committed, "worker drain pause SQL commit")
	var paused DeployTask
	if err := db.First(&paused, task.ID).Error; err != nil {
		t.Fatal(err)
	}
	if paused.Status != TaskStatusPaused {
		t.Fatalf("drain committed %s instead of paused", paused.Status)
	}
	if _, err := svc.ResumeTask(task.ID, "test", nil); err != nil {
		t.Fatal(err)
	}
	svc.runAsyncTask(task.ID, "replacement")
	var attempts int64
	if err := db.Model(&DeployTaskAttempt{}).Where("task_id = ?", task.ID).Count(&attempts).Error; err != nil {
		t.Fatal(err)
	}
	if attempts != 1 {
		t.Fatalf("resumed replacement dispatched %d attempts while old pause reply was pending", attempts)
	}
	var completed DeployTask
	if err := db.First(&completed, task.ID).Error; err != nil {
		t.Fatal(err)
	}
	if completed.Status != TaskStatusSuccess {
		t.Fatalf("replacement did not complete: %s", completed.Status)
	}
	// Represent a later dispatcher reservation after the replacement completed.
	// The original worker's deferred cleanup must not erase a successor slot.
	successor := &taskWorkerSlot{}
	svc.taskWorkers.Store(task.ID, successor)
	releaseReply()
	waitOwnership(t, drained, "old pause reply drained")
	if current, present := svc.taskWorkers.Load(task.ID); !present || current != successor {
		t.Fatal("old worker cleanup deleted a successor dispatcher reservation")
	}
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
