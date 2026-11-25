package deploy

import (
	"context"
	"testing"
	"time"
)

// TestAsyncRecoveryKeepsOriginatingTenant pins T04's async gap: a durable
// running task must keep the tenant that created it when a fresh worker
// process resumes the queue. The resumed worker reloads the task from the
// database, so the tenant must come from the row rather than from a request
// context that no longer exists.
func TestAsyncRecoveryKeepsOriginatingTenant(t *testing.T) {
	db, _, restarted, task, _ := ownershipFixture(t)
	const tenantID uint64 = 4242
	if err := db.Model(&DeployTask{}).Where("id = ?", task.ID).Update("tenant_id", tenantID).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.First(&task, task.ID).Error; err != nil {
		t.Fatal(err)
	}
	if task.TenantID != tenantID {
		t.Fatalf("fixture tenant setup: row tenant_id = %d, want %d", task.TenantID, tenantID)
	}

	// A fresh service instance over the same database models the process
	// restart: no in-memory task state, only the durable queue.
	configureUnclaimedSSHWorker(t, restarted, task.ID)
	executed := make(chan DeployExecutionHostRequest, 1)
	restarted.SetExecutor(ownershipExecutorFunc(func(_ context.Context, req DeployExecutionHostRequest) error {
		select {
		case executed <- req:
		default:
		}
		return nil
	}))

	if _, err := restarted.ReconcileDeployQueue("tenant-recovery"); err != nil {
		t.Fatalf("queue recovery: %v", err)
	}

	select {
	case req := <-executed:
		if req.Task.TenantID != tenantID {
			t.Fatalf("executor received tenant %d, want the originating tenant %d", req.Task.TenantID, tenantID)
		}
		if req.Task.ID != task.ID {
			t.Fatalf("executor received task %d, want %d", req.Task.ID, task.ID)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("resumed worker never executed the retained task")
	}

	// Let the background settlement finish before the per-test database is
	// closed, otherwise the cleanup races an in-flight worker transaction.
	waitForDeployWorkerStop(t, restarted, task.ID)

	var reloaded DeployTask
	if err := db.First(&reloaded, task.ID).Error; err != nil {
		t.Fatal(err)
	}
	if reloaded.TenantID != tenantID {
		t.Fatalf("after recovery row tenant_id = %d, want %d", reloaded.TenantID, tenantID)
	}

	// Recovery must not invent a tenant-less copy of the task row.
	var tasks int64
	if err := db.Model(&DeployTask{}).Where("tenant_id = 0").Count(&tasks).Error; err != nil {
		t.Fatal(err)
	}
	if tasks != 0 {
		t.Fatalf("recovery left %d tenant-less task row(s)", tasks)
	}
}
