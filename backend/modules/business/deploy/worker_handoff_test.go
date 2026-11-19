package deploy

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	serviceowner "github.com/duanxldragon/pantheon-base/backend/modules/business/service"
	"gorm.io/gorm"
)

// Block only the transaction that actually pauses the selected attempt. Claim,
// renewal and checkpoint transactions must retain their normal commit replies.
type ownedPauseCommitPool struct {
	gorm.ConnPool
	sqlDB     *sql.DB
	attemptID *atomic.Uint64
	committed chan struct{}
	reply     <-chan struct{}
	blocked   *atomic.Bool
}

func (p ownedPauseCommitPool) BeginTx(ctx context.Context, opts *sql.TxOptions) (gorm.ConnPool, error) {
	tx, err := p.sqlDB.BeginTx(ctx, opts) // NOSONAR godre:S8168
	if err != nil {
		return nil, err
	}
	return &ownedPauseCommitTx{ownershipCommitTx: ownershipCommitTx{Tx: tx}, pool: p}, nil
}

type ownedPauseCommitTx struct {
	ownershipCommitTx
	pool   ownedPauseCommitPool
	pauses bool
}

type finalSettlementCommitPool struct {
	gorm.ConnPool
	sqlDB     *sql.DB
	taskID    uint64
	committed chan struct{}
	reply     <-chan struct{}
}

func (p finalSettlementCommitPool) BeginTx(ctx context.Context, opts *sql.TxOptions) (gorm.ConnPool, error) {
	tx, err := p.sqlDB.BeginTx(ctx, opts) // NOSONAR godre:S8168
	if err != nil {
		return nil, err
	}
	return &finalSettlementCommitTx{ownershipCommitTx: ownershipCommitTx{Tx: tx}, pool: p}, nil
}

type finalSettlementCommitTx struct {
	ownershipCommitTx
	pool    finalSettlementCommitPool
	settles bool
}

func (tx *finalSettlementCommitTx) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	result, err := tx.Tx.ExecContext(ctx, query, args...)
	if err == nil && strings.HasPrefix(query, "UPDATE `biz_deploy_task` SET ") {
		statusAt, idAt := strings.Index(query, "`status`=?"), strings.Index(query, "WHERE id = ?")
		if statusAt >= 0 && idAt > statusAt {
			statusArg, idArg := strings.Count(query[:statusAt], "?"), strings.Count(query[:idAt], "?")
			if idArg < len(args) && args[statusArg] == TaskStatusCanceled && args[idArg] == tx.pool.taskID {
				tx.settles = true
			}
		}
	}
	return result, err
}

func (tx *finalSettlementCommitTx) Commit() error {
	if tx.settles {
		tx.committed, tx.commitReply = tx.pool.committed, tx.pool.reply
	}
	return tx.ownershipCommitTx.Commit()
}

func TestWorkerHandoffFixtureWaitsForFinalSettlement(t *testing.T) {
	db, svc, _, task, _ := ownershipFixture(t)
	configureUnclaimedSSHWorker(t, svc, task.ID)
	cancelUnclaimedTask(t, svc, task)
	committed, reply, drained, cleanupReturned := make(chan struct{}), make(chan struct{}), make(chan struct{}), make(chan struct{})
	releaseReply := sync.OnceFunc(func() { close(reply) })
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	isolated := db.Session(&gorm.Session{NewDB: true}).WithContext(context.Background())
	isolated.Statement.ConnPool = finalSettlementCommitPool{ConnPool: sqlDB, sqlDB: sqlDB, taskID: task.ID, committed: committed, reply: reply}
	svc.db = isolated
	captured := make(chan *taskWorkerSlot, 1)
	var capture sync.Once
	if err := db.Callback().Query().Before("gorm:query").Register("final_settlement_worker_capture", func(tx *gorm.DB) {
		if tx.Statement.Table == "biz_deploy_task" {
			if current, present := svc.taskWorkers.Load(task.ID); present {
				capture.Do(func() { captured <- current.(*taskWorkerSlot) })
			}
		}
	}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Callback().Query().Remove("final_settlement_worker_capture") })
	var original *taskWorkerSlot
	var cleanupStarted bool
	go func() { defer close(drained); svc.runAsyncTask(task.ID, "fixture-settlement") }()
	t.Cleanup(func() {
		releaseReply()
		waitOwnership(t, drained, "final settlement worker cleanup join")
		if original != nil {
			waitOwnership(t, original.done, "original whole worker cleanup join")
		}
		if cleanupStarted {
			waitOwnership(t, cleanupReturned, "final settlement fixture cleanup join")
		}
	})
	waitOwnership(t, committed, "actual final settlement SQL commit")
	select {
	case original = <-captured:
	default:
		t.Fatal("final settlement fixture did not capture its real worker")
	}
	if _, present := svc.taskWorkers.Load(task.ID); present {
		t.Fatal("final settlement must occur after worker reservation release")
	}
	select {
	case <-original.done:
		t.Fatal("whole worker exit signaled before its final settlement reply")
	default:
	}
	cleanupStarted = true
	go func() { defer close(cleanupReturned); waitHandoffWorkerReleased(t, svc, task.ID, original) }()
	select {
	case <-cleanupReturned:
		t.Fatal("fixture cleanup returned while final settlement commit reply was pending")
	case <-time.After(50 * time.Millisecond):
	}
	releaseReply()
	waitOwnership(t, drained, "final settlement complete worker lifecycle")
	waitOwnership(t, original.done, "whole worker exit after settlement reply")
	waitOwnership(t, cleanupReturned, "fixture cleanup after final settlement")
	assertCanceledUnclaimed(t, db, task.ID)
}

func (tx *ownedPauseCommitTx) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	result, err := tx.Tx.ExecContext(ctx, query, args...)
	if err == nil && strings.HasPrefix(query, "UPDATE `biz_deploy_task_attempt` SET ") {
		statusAt, idAt := strings.Index(query, "`status`=?"), strings.Index(query, "WHERE id = ?")
		if statusAt >= 0 && idAt > statusAt {
			statusArg, idArg := strings.Count(query[:statusAt], "?"), strings.Count(query[:idAt], "?")
			if idArg < len(args) && args[statusArg] == TaskStatusPaused && args[idArg] == tx.pool.attemptID.Load() {
				tx.pauses = true
			}
		}
	}
	return result, err
}

func (tx *ownedPauseCommitTx) Commit() error {
	if tx.pauses && tx.pool.blocked.CompareAndSwap(false, true) {
		tx.committed, tx.commitReply = tx.pool.committed, tx.pool.reply
	}
	return tx.ownershipCommitTx.Commit()
}

type handoffSSHRunner struct {
	entered chan struct{}
	release <-chan struct{}
	once    sync.Once
}

func (r *handoffSSHRunner) RunScript(ctx context.Context, _ string) (string, string, error) {
	r.once.Do(func() { close(r.entered) })
	select {
	case <-r.release:
		return "", "", nil
	case <-ctx.Done():
		return "", "", ctx.Err()
	}
}

func (r *handoffSSHRunner) Close() error { return nil }

func TestWorkerHandoffOwnedPauseResumeDispatch(t *testing.T) {
	testOwnedPauseResumeHandoff(t, "sync")
}

func TestWorkerHandoffOwnedPauseResumeAsyncDispatch(t *testing.T) {
	testOwnedPauseResumeHandoff(t, "async")
}

func TestWorkerHandoffOwnedPauseResumeCancel(t *testing.T) {
	testOwnedPauseResumeHandoff(t, "cancel")
}

func TestWorkerHandoffOwnedPauseResumeSuperseded(t *testing.T) {
	testOwnedPauseResumeHandoff(t, "superseded")
}

func TestWorkerHandoffOwnedPauseResumeForeignLease(t *testing.T) {
	testOwnedPauseResumeHandoff(t, "foreign-lease")
}

func TestWorkerHandoffOwnedPauseResumeForeignAttempt(t *testing.T) {
	testOwnedPauseResumeHandoff(t, "foreign-attempt")
}

func testOwnedPauseResumeHandoff(t *testing.T, mode string) {
	t.Helper()
	db, svc, task, owner, instance := startedServiceLink(t)
	configureUnclaimedSSHWorker(t, svc, task.ID)
	svc.SetAsyncExecution(mode == "async")
	firstEntered, replacementEntered := make(chan struct{}), make(chan struct{})
	replacementReply := make(chan struct{})
	releaseReplacement := sync.OnceFunc(func() { close(replacementReply) })
	var runnerCount atomic.Uint64
	successorSlots := make(chan *taskWorkerSlot, 1)
	var originalSlot, successorSlot *taskWorkerSlot
	var replacementRequested bool
	svc.sshRunnerFactory = func(context.Context, cmdbHostSnapshot, StartTaskRequest) (deploySSHRunner, error) {
		count := runnerCount.Add(1)
		if count == 1 {
			return &blockingDeploySSHRunner{started: firstEntered}, nil
		}
		if count == 2 {
			current, present := svc.taskWorkers.Load(task.ID)
			if !present {
				return nil, errors.New("replacement SSH runner has no worker reservation")
			}
			successorSlots <- current.(*taskWorkerSlot)
			return &handoffSSHRunner{entered: replacementEntered, release: replacementReply}, nil
		}
		return nil, errors.New("unexpected third handoff SSH runner")
	}
	committed, reply, drained := make(chan struct{}), make(chan struct{}), make(chan struct{})
	releaseReply := sync.OnceFunc(func() { close(reply) })
	var targetAttempt atomic.Uint64
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	isolated := db.Session(&gorm.Session{NewDB: true}).WithContext(context.Background())
	isolated.Statement.ConnPool = ownedPauseCommitPool{
		ConnPool: sqlDB, sqlDB: sqlDB, attemptID: &targetAttempt,
		committed: committed, reply: reply, blocked: &atomic.Bool{},
	}
	svc.db = isolated
	go func() { defer close(drained); svc.runAsyncTask(task.ID, "original") }()
	t.Cleanup(func() {
		releaseReply()
		releaseReplacement()
		if cancel, present := svc.taskCancels.Load(task.ID); present {
			cancel.(context.CancelFunc)()
		}
		waitOwnership(t, drained, "owned pause worker cleanup join")
		if successorSlot == nil {
			if replacementRequested {
				select {
				case successorSlot = <-successorSlots:
				case <-time.After(5 * time.Second):
					// A failed replacement may exit before constructing SSH.
					// Join any captured original/current worker before reporting it.
					waitHandoffWorkerReleased(t, svc, task.ID, originalSlot)
					t.Fatal("replacement worker never reached its fixture runner")
				}
			} else {
				select {
				case successorSlot = <-successorSlots:
				default:
				}
			}
		}
		waitHandoffWorkerReleased(t, svc, task.ID, originalSlot, successorSlot)
	})
	waitOwnership(t, firstEntered, "owned SSH command entered")
	if current, present := svc.taskWorkers.Load(task.ID); present {
		originalSlot = current.(*taskWorkerSlot)
	} else {
		t.Fatal("original SSH runner has no worker reservation")
	}
	var first DeployTaskAttempt
	if err := db.Where("task_id = ? AND status = ?", task.ID, TaskHostStatusRunning).First(&first).Error; err != nil {
		t.Fatal(err)
	}
	targetAttempt.Store(first.ID)
	if _, err := svc.PauseTask(task.ID, "test", nil); err != nil {
		t.Fatal(err)
	}
	waitOwnership(t, committed, "selected attempt pause SQL commit")
	var paused DeployTask
	if err := db.First(&paused, task.ID).Error; err != nil || paused.Status != TaskStatusPaused {
		t.Fatalf("owned pause not committed: status=%s error=%v", paused.Status, err)
	}
	// Resume's transaction is real. Invoke its dispatcher synchronously to make
	// the duplicate dispatch happen before the original commit reply is released.
	if _, err := svc.ResumeTask(task.ID, "test", nil); err != nil {
		t.Fatal(err)
	}
	replacementRequested = mode == "sync" || mode == "async"
	if mode == "async" {
		waitHandoffDispatchPending(t, svc, task.ID)
	} else {
		svc.runAsyncTask(task.ID, "replacement")
	}
	if runnerCount.Load() != 1 {
		t.Fatal("replacement executed before the original worker drained")
	}
	var protectedFacts string
	switch mode {
	case "cancel":
		if _, err := svc.CancelTask(task.ID, "test", nil); err != nil {
			t.Fatal(err)
		}
	case "superseded":
		desired := "superseding-handoff-owner"
		if _, err := owner.UpdateInstance(instance.ID, serviceowner.UpdateInstanceRequest{DesiredVersion: &desired}, "next-owner", nil); err != nil {
			t.Fatal(err)
		}
		protectedFacts = serviceOwnerFacts(t, db)
	case "foreign-lease":
		if err := svc.fencedTransaction(task.ID, func(tx *gorm.DB, state lockedDeployment) error {
			now, err := databaseNow(tx)
			if err != nil {
				return err
			}
			return tx.Create(&DeployHostLease{HostID: state.hosts[0].HostID, TaskID: task.ID + 100, Owner: "live-foreign", ExpiresAt: now.Add(time.Minute)}).Error
		}); err != nil {
			t.Fatal(err)
		}
		protectedFacts = ownershipFacts(t, db)
	case "foreign-attempt":
		var running DeployTask
		var host DeployTaskHost
		if err := db.First(&running, task.ID).Error; err != nil {
			t.Fatal(err)
		}
		if err := db.Where("task_id = ?", task.ID).First(&host).Error; err != nil {
			t.Fatal(err)
		}
		_, _ = ownedClaim(t, svc, running, host, "live-foreign-attempt")
		protectedFacts = ownershipFacts(t, db)
	}
	releaseReply()
	waitOwnership(t, drained, "original owned worker drained")
	waitOwnership(t, originalSlot.done, "original whole worker exit")
	if mode != "sync" && mode != "async" {
		if runnerCount.Load() != 1 {
			t.Fatal("covered execution intent started a replacement SSH runner")
		}
		if _, present := svc.taskWorkers.Load(task.ID); present {
			t.Fatal("covered execution intent retained a replacement worker")
		}
		switch mode {
		case "cancel":
			assertCanceledUnclaimed(t, db, task.ID)
		case "superseded":
			if serviceOwnerFacts(t, db) != protectedFacts {
				t.Fatal("pending dispatch overwrote the newer Service owner")
			}
		default:
			if ownershipFacts(t, db) != protectedFacts {
				t.Fatal("pending dispatch changed foreign live ownership")
			}
		}
		return
	}
	waitOwnership(t, replacementEntered, "pending Resume replacement entered without recovery")
	successorSlot = <-successorSlots
	var attempts []DeployTaskAttempt
	if err := db.Where("task_id = ?", task.ID).Order("attempt_no ASC").Find(&attempts).Error; err != nil {
		t.Fatal(err)
	}
	if len(attempts) != 2 || attempts[0].Status != TaskStatusPaused || attempts[1].AttemptNo != first.AttemptNo+1 || attempts[1].Status != TaskHostStatusRunning {
		t.Fatalf("pause checkpoint/replacement attempt lost: %+v", attempts)
	}
	if current, present := svc.taskWorkers.Load(task.ID); !present || current != successorSlot {
		t.Fatal("original deferred cleanup erased the live replacement worker")
	}
	if _, present := svc.taskCancels.Load(task.ID); !present {
		t.Fatal("original deferred cleanup erased the live replacement cancellation")
	}
	releaseReplacement()
	waitHandoffWorkerReleased(t, svc, task.ID, successorSlot)
	var completed DeployTask
	if err := db.First(&completed, task.ID).Error; err != nil || completed.Status != TaskStatusSuccess {
		t.Fatalf("replacement did not complete: status=%s error=%v", completed.Status, err)
	}
}

func waitHandoffDispatchPending(t *testing.T, svc *DeployService, taskID uint64) {
	t.Helper()
	deadline := time.NewTimer(5 * time.Second)
	defer deadline.Stop()
	poll := time.NewTicker(5 * time.Millisecond)
	defer poll.Stop()
	for {
		if current, present := svc.taskWorkers.Load(taskID); present {
			slot := current.(*taskWorkerSlot)
			slot.mu.Lock()
			pending := slot.pending
			slot.mu.Unlock()
			if pending {
				return
			}
		}
		select {
		case <-poll.C:
		case <-deadline.C:
			t.Fatal("Resume API did not reserve its asynchronous dispatcher")
		}
	}
}

func TestWorkerHandoffReservationRetirementOrders(t *testing.T) {
	for _, first := range []string{"dispatch", "retirement"} {
		t.Run(first, func(t *testing.T) {
			svc := &DeployService{}
			const taskID uint64 = 71
			original := svc.reserveTaskWorker(taskID)
			ctx, cancel := context.WithCancel(context.Background())
			svc.taskCancels.Store(taskID, cancel)
			if first == "dispatch" {
				if svc.reserveTaskWorker(taskID) != nil || !svc.releaseTaskWorker(taskID, original, cancel) {
					t.Fatal("dispatch received before retirement was not captured")
				}
			} else {
				stale := original // dispatcher already loaded the old pointer
				if svc.releaseTaskWorker(taskID, original, cancel) || stale.reserveDispatch() {
					t.Fatal("retired pointer accepted a dispatch that must retry the map")
				}
			}
			if ctx.Err() == nil {
				t.Fatal("retirement exposed the slot before canceling its context")
			}
			if _, present := svc.taskCancels.Load(taskID); present {
				t.Fatal("retirement retained the old cancellation")
			}
			replacement := svc.reserveTaskWorker(taskID)
			if replacement == nil || replacement == original {
				t.Fatal("dispatch could not reserve a fresh successor")
			}
			if svc.reserveTaskWorker(taskID) != nil {
				t.Fatal("successor did not retain the next dispatch reservation")
			}
			// Identity deletion must not remove a newer map slot.
			svc.releaseTaskWorker(taskID, original, nil)
			if current, present := svc.taskWorkers.Load(taskID); !present || current != replacement {
				t.Fatal("old retirement deleted the successor's identity")
			}
			if !svc.releaseTaskWorker(taskID, replacement, nil) {
				t.Fatal("old retirement erased the successor's pending reservation")
			}
		})
	}
}

func TestWorkerHandoffManualResumeDoesNotCreateWorker(t *testing.T) {
	_, svc, task, _, _ := startedServiceLink(t)
	svc.SetAsyncExecution(true)
	if _, err := svc.PauseTask(task.ID, "test", nil); err != nil {
		t.Fatal(err)
	}
	resumed, err := svc.ResumeTask(task.ID, "test", nil)
	if err != nil {
		t.Fatal(err)
	}
	if resumed.ExecutorType != ExecutorTypeManual || resumed.Status != TaskStatusRunning {
		t.Fatal("Manual Resume changed its synchronous execution contract")
	}
	if _, present := svc.taskWorkers.Load(task.ID); present {
		t.Fatal("Manual Resume created a worker reservation")
	}
	if _, present := svc.taskCancels.Load(task.ID); present {
		t.Fatal("Manual Resume created a worker cancellation")
	}
}

func TestWorkerHandoffPendingReadPreservesDatabaseError(t *testing.T) {
	db, svc, task, _, _ := startedServiceLink(t)
	configureUnclaimedSSHWorker(t, svc, task.ID)
	injected := errors.New("pending dispatch database read refused")
	if err := db.Callback().Query().Before("gorm:query").Register("pending_read_error", func(tx *gorm.DB) {
		if tx.Statement.Table == "biz_deploy_task" {
			tx.AddError(injected)
		}
	}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Callback().Query().Remove("pending_read_error") })
	if err := svc.dispatchPendingTask(task.ID, "test"); !errors.Is(err, injected) {
		t.Fatalf("pending dispatch swallowed database error: %v", err)
	}
	if _, present := svc.taskWorkers.Load(task.ID); present {
		t.Fatal("failed pending admission created a worker reservation")
	}
}

func waitHandoffWorkerReleased(t *testing.T, svc *DeployService, taskID uint64, captured ...*taskWorkerSlot) {
	t.Helper()
	for _, slot := range captured {
		if slot != nil {
			waitOwnership(t, slot.done, "captured whole worker exit before database cleanup")
		}
	}
	for {
		current, present := svc.taskWorkers.Load(taskID)
		if !present {
			return
		}
		waitOwnership(t, current.(*taskWorkerSlot).done, "active whole worker exit before database cleanup")
	}
}
