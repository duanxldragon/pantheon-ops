package deploy

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"gorm.io/gorm"
)

func TestWorkerQueueRecoveryAfterPendingWorkerError(t *testing.T) {
	for _, phase := range []string{"startup-reconcile", "startup-task-read", "startup-host-read", "final-recompute"} {
		t.Run(phase, func(t *testing.T) {
			db, svc, _, task, _ := ownershipFixture(t)
			configureUnclaimedSSHWorker(t, svc, task.ID)
			var executions atomic.Int32
			svc.SetExecutor(ownershipExecutorFunc(func(context.Context, DeployExecutionHostRequest) error {
				executions.Add(1)
				return nil
			}))
			before := ownershipFacts(t, db)
			injected := errors.New("pending worker " + phase + " query refused")
			var failed, canceled, pendingMiss atomic.Bool
			var seen sync.Map
			captured := make(chan *taskWorkerSlot, 2)
			requestPending := func() {
				if extra := svc.reserveTaskWorker(task.ID); extra != nil {
					pendingMiss.Store(true)
					svc.releaseTaskWorker(task.ID, extra, nil)
					close(extra.done)
				}
			}
			if err := db.Callback().Query().Before("gorm:query").Register("queue_worker_error", func(tx *gorm.DB) {
				if current, present := svc.taskWorkers.Load(task.ID); present {
					if _, loaded := seen.LoadOrStore(current, true); !loaded {
						captured <- current.(*taskWorkerSlot)
					}
				}
				if failed.Load() {
					return
				}
				_, locking := tx.Statement.Clauses["FOR"]
				match := false
				switch phase {
				case "startup-reconcile":
					match = tx.Statement.Table == "biz_deploy_task_attempt"
				case "startup-task-read":
					match = tx.Statement.Table == "biz_deploy_task" && !locking
				case "startup-host-read":
					match = tx.Statement.Table == "biz_deploy_task_host" && !locking
				case "final-recompute":
					// Cancel the old process-local worker after its context is registered,
					// leaving durable task intent Running and all hosts unclaimed.
					if tx.Statement.Table == "biz_deploy_task" && !locking {
						if cancel, present := svc.taskCancels.Load(task.ID); present {
							if canceled.CompareAndSwap(false, true) {
								requestPending()
								cancel.(context.CancelFunc)()
							}
						}
					}
					match = tx.Statement.Table == "biz_deploy_task" && locking && canceled.Load()
				}
				if match && failed.CompareAndSwap(false, true) {
					// A second dispatcher reaches the actual active slot before its
					// failing query returns. It queues pending work without a new worker.
					if phase != "final-recompute" {
						requestPending()
					}
					tx.AddError(injected)
				}
			}); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() {
				if cancel, present := svc.taskCancels.Load(task.ID); present {
					cancel.(context.CancelFunc)()
				}
				seen.Range(func(slot, _ any) bool {
					waitOwnership(t, slot.(*taskWorkerSlot).done, "queue recovery complete worker cleanup")
					return true
				})
				db.Callback().Query().Remove("queue_worker_error")
			})

			svc.runAsyncTask(task.ID, "initial")
			if !failed.Load() {
				t.Fatal("worker did not reach the selected injected failure")
			}
			var original *taskWorkerSlot
			select {
			case original = <-captured:
			default:
				t.Fatal("failure did not capture its actual worker reservation")
			}
			waitOwnership(t, original.done, "failed worker complete exit")
			original.mu.Lock()
			retired, pending := original.retired, original.pending
			original.mu.Unlock()
			if !retired || !pending || pendingMiss.Load() {
				t.Fatal("query failure did not retire a worker with pending dispatch")
			}
			if _, present := svc.taskWorkers.Load(task.ID); present {
				t.Fatal("query failure stranded a process-local worker reservation")
			}
			if executions.Load() != 0 || ownershipFacts(t, db) != before {
				t.Fatal("failed unclaimed worker changed durable task/host/attempt/lease facts")
			}

			count, err := svc.ReconcileDeployQueue("recovered")
			if err != nil || count != 1 {
				t.Fatalf("durable queue recovery: queued=%d err=%v", count, err)
			}
			var recovered *taskWorkerSlot
			select {
			case recovered = <-captured:
			case <-time.After(5 * time.Second):
				t.Fatal("durable recovery did not reserve its replacement worker")
			}
			waitOwnership(t, recovered.done, "recovered complete worker exit")
			var completed DeployTask
			var attempts []DeployTaskAttempt
			var leases int64
			if err := db.First(&completed, task.ID).Error; err != nil {
				t.Fatal(err)
			}
			if err := db.Where("task_id = ?", task.ID).Find(&attempts).Error; err != nil {
				t.Fatal(err)
			}
			if err := db.Model(&DeployHostLease{}).Count(&leases).Error; err != nil {
				t.Fatal(err)
			}
			if completed.Status != TaskStatusSuccess || executions.Load() != 1 || len(attempts) != 1 || attempts[0].Status != TaskHostStatusSuccess || leases != 0 {
				t.Fatalf("queue recovery did not complete exactly once: status=%s executions=%d attempts=%+v leases=%d", completed.Status, executions.Load(), attempts, leases)
			}
			settled := ownershipFacts(t, db)
			count, err = svc.ReconcileDeployQueue("replay")
			if err != nil || count != 0 || ownershipFacts(t, db) != settled {
				t.Fatalf("settled queue replay changed facts: queued=%d err=%v", count, err)
			}
		})
	}
}
