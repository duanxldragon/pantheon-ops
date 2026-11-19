package deploy

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func TestWorkerClaimParentCanceledBeforeClaim(t *testing.T) {
	db, svc, _, task, host := ownershipFixture(t)
	before := ownershipFacts(t, db)
	var executions atomic.Int32
	svc.SetExecutor(ownershipExecutorFunc(func(context.Context, DeployExecutionHostRequest) error {
		executions.Add(1)
		return nil
	}))
	parent, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := svc.runOwnedHost(parent, task, host, StartTaskRequest{}, "test", "already-canceled", false)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("pre-canceled parent claim: %v", err)
	}
	if executions.Load() != 0 || ownershipFacts(t, db) != before {
		t.Fatal("pre-canceled worker consumed an attempt or changed execution facts")
	}
}

func TestWorkerClaimParentCancellationWhileTaskLocked(t *testing.T) {
	db, svc, _, task, host := ownershipFixture(t)
	before := ownershipFacts(t, db)
	blocker := db.Begin()
	if blocker.Error != nil {
		t.Fatal(blocker.Error)
	}
	releaseBlocker := sync.OnceFunc(func() {
		if err := blocker.Rollback().Error; err != nil {
			t.Errorf("release task-row blocker: %v", err)
		}
	})
	t.Cleanup(releaseBlocker)
	var locked DeployTask
	if err := blocker.Clauses(clause.Locking{Strength: "UPDATE"}).First(&locked, task.ID).Error; err != nil {
		t.Fatal(err)
	}

	claimEntered := make(chan struct{})
	var signal sync.Once
	if err := db.Callback().Query().Before("gorm:query").Register("worker_parent_claim_lock", func(tx *gorm.DB) {
		if tx.Statement.Table == "biz_deploy_task" {
			if _, locking := tx.Statement.Clauses["FOR"]; locking {
				signal.Do(func() { close(claimEntered) })
			}
		}
	}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Callback().Query().Remove("worker_parent_claim_lock") })
	var executions atomic.Int32
	svc.SetExecutor(ownershipExecutorFunc(func(context.Context, DeployExecutionHostRequest) error {
		executions.Add(1)
		return nil
	}))
	parent, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	var runErr error
	go func() {
		defer close(done)
		_, runErr = svc.runOwnedHost(parent, task, host, StartTaskRequest{}, "test", "blocked-canceled", false)
	}()
	t.Cleanup(func() {
		cancel()
		releaseBlocker()
		waitOwnership(t, done, "blocked claim cleanup join")
	})
	waitOwnership(t, claimEntered, "claim task locking query")
	select {
	case <-done:
		t.Fatalf("claim returned before cancellation while task-row lock was held: %v", runErr)
	case <-time.After(100 * time.Millisecond):
	}
	cancel()
	// The blocker deliberately stays locked until after the canceled claim exits.
	// Releasing it first would hide an old worker consuming a resumed task's claim.
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("parent cancellation did not interrupt the blocked claim before task-row lock release")
	}
	if !errors.Is(runErr, context.Canceled) {
		t.Fatalf("blocked claim cancellation: %v", runErr)
	}
	releaseBlocker()
	if executions.Load() != 0 || ownershipFacts(t, db) != before {
		t.Fatal("canceled blocked worker consumed an attempt or changed execution facts")
	}
}
