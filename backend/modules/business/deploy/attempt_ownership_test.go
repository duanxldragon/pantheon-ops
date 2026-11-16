package deploy

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	bizcap "github.com/duanxldragon/pantheon-base/backend/modules/business/capability"
	"github.com/duanxldragon/pantheon-base/backend/modules/business/cmdb"
	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
	"gorm.io/gorm"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type ownershipExecutorFunc func(context.Context, DeployExecutionHostRequest) error

func TestOwnershipCommandRechecksDeadlineAfterScopeRead(t *testing.T) {
	_, a, _, task, host := ownershipFixture(t)
	inner, ctx := ownedClaim(t, a, task, host, "scope-deadline")
	reader := inner.bizScopeReader
	inner.bizScopeReader = admissionScopeReader{BizScopeReader: reader, read: func(ctx context.Context, id uint64, scope *common.DataScopeReq) (bizcap.BizScopeRef, error) {
		owner, err := reader.GetActive(ctx, id, scope)
		// Model a valid read whose reply consumed the last confirmed horizon.
		// Deliberately omit the watchdog to prove the command's own deadline guard.
		inner.ownership.mu.Lock()
		inner.ownership.deadline = time.Now().Add(-time.Second)
		inner.ownership.mu.Unlock()
		return owner, err
	}}
	transport := &fakeDeploySSHRunner{}
	runner := &executionScopeSSHRunner{deploySSHRunner: transport, service: inner, task: task, hostID: host.HostID}
	_, _, err := runner.RunScript(ctx, "late command")
	if err == nil || transport.script != "" {
		t.Fatalf("expired command admitted: %v script=%s", err, transport.script)
	}
}

func TestOwnershipHistoricalComponentHelperCannotBypassLiveOwner(t *testing.T) {
	db, a, b, task, host := ownershipFixture(t)
	_, _ = ownedClaim(t, a, task, host, "live-projection")
	before := ownershipFacts(t, db)
	err := b.upsertHostInstalledComponent(host.HostID, task.ID+100, "absent historical task", ExecutorTypeSSH, deployPackageRef{name: "stale", version: "1"}, "old", time.Now())
	if err == nil || ownershipFacts(t, db) != before {
		t.Fatal("historical component projection bypassed live physical ownership")
	}
}

func TestOwnershipStartDiscoveryDoesNotLockCMDBBeforeAdmission(t *testing.T) {
	for _, selected := range []bool{false, true} {
		t.Run(map[bool]string{false: "legacy", true: "selected"}[selected], func(t *testing.T) {
			db, s, task, _ := previewFixture(t, selected)
			var earlyLock, physicalLock, projectionLock atomic.Bool
			if err := db.Callback().Query().Before("gorm:query").Register("ownership_discovery_lock_order", func(tx *gorm.DB) {
				_, transaction := tx.Statement.ConnPool.(gorm.TxCommitter)
				_, locking := tx.Statement.Clauses["FOR"]
				if locking && tx.Statement.Table == "biz_deploy_host_lease" {
					physicalLock.Store(true)
				}
				if locking && (tx.Statement.Table == "biz_cmdb_host" || tx.Statement.Table == "biz_business_scope") {
					projectionLock.Store(true)
					if !transaction || !physicalLock.Load() {
						earlyLock.Store(true)
					}
				}
			}); err != nil {
				t.Fatal(err)
			}
			preview, err := s.PreviewTask(task.ID, StartTaskRequest{}, "1", nil)
			if err != nil {
				t.Fatal(err)
			}
			req := StartTaskRequest{}
			if selected {
				req.RequirePreview, req.IntentVersion = true, preview.IntentVersion
			}
			if _, err := s.StartTask(task.ID, req, "1", nil); err != nil {
				t.Fatal(err)
			}
			if earlyLock.Load() || !physicalLock.Load() || !projectionLock.Load() {
				t.Fatal("discovery/admission did not keep physical lease before locked CMDB/source validation")
			}
		})
	}
}

func (f ownershipExecutorFunc) ExecuteHost(ctx context.Context, req DeployExecutionHostRequest) error {
	return f(ctx, req)
}

func ownershipFixture(t *testing.T) (*gorm.DB, *DeployService, *DeployService, DeployTask, DeployTaskHost) {
	t.Helper()
	db := setupDeployTestDB(t)
	scopeID, physical := seedDeployReliabilityFixture(t, db, "ownership-host", "10.40.0.233")
	task := DeployTask{Name: "ownership", Status: TaskStatusRunning, ExecutorType: ExecutorTypeSSH, Action: TaskActionInstall, BusinessScopeID: scopeID, PackageName: "owned", PackageVersion: "1"}
	if err := db.Create(&task).Error; err != nil {
		t.Fatal(err)
	}
	host := DeployTaskHost{TaskID: task.ID, HostID: physical, Status: TaskHostStatusRunning, BusinessScopeID: scopeID}
	if err := db.Create(&host).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&DeployTaskStep{TaskID: task.ID, TaskHostID: host.ID, StepKey: "owned", Status: StepStatusPending, StateVersion: 1}).Error; err != nil {
		t.Fatal(err)
	}
	a := NewDeployService(db, cmdb.NewDeployCMDBCapability(db))
	b := NewDeployService(db, cmdb.NewDeployCMDBCapability(db))
	return db, a, b, task, host
}

func ownedClaim(t *testing.T, s *DeployService, task DeployTask, host DeployTaskHost, worker string) (*DeployService, context.Context) {
	t.Helper()
	a, err := s.claimOwnedAttempt(task.ID, host.ID, worker)
	if err != nil {
		t.Fatal(err)
	}
	o := tokenForAttempt(*a, host)
	ctx, cancel := context.WithCancelCause(context.Background())
	o.cancel = cancel
	t.Cleanup(func() { cancel(context.Canceled) })
	inner := s.ownedService(o)
	if err := inner.renewOwnership(o); err != nil {
		t.Fatal(err)
	}
	return inner, ctx
}

func ownershipFacts(t *testing.T, db *gorm.DB) string {
	t.Helper()
	tables := []string{"biz_deploy_task", "biz_deploy_task_host", "biz_deploy_task_attempt", "biz_deploy_host_lease", "biz_deploy_task_step", "biz_deploy_step_history", "biz_deploy_task_event", "biz_deploy_task_log_chunk", "biz_cmdb_host"}
	result := make([]any, 0, len(tables))
	for _, table := range tables {
		var rows []map[string]any
		if err := db.Table(table).Order("id").Find(&rows).Error; err != nil {
			t.Fatal(err)
		}
		result = append(result, rows)
	}
	raw, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

func waitOwnership(t *testing.T, ch <-chan struct{}, what string) {
	t.Helper()
	select {
	case <-ch:
	case <-time.After(5 * time.Second):
		t.Fatalf("timed out: %s", what)
	}
}

func TestOwnershipDualRenewalAndPhysicalContention(t *testing.T) {
	db, a, b, task, host := ownershipFixture(t)
	a.leaseTiming = &ownershipTiming{2 * time.Second, 3 * time.Second, 100 * time.Millisecond, time.Second, 100 * time.Millisecond}
	entered, release, done := make(chan struct{}), make(chan struct{}), make(chan struct{})
	var executions atomic.Int32
	a.SetExecutor(ownershipExecutorFunc(func(ctx context.Context, _ DeployExecutionHostRequest) error {
		executions.Add(1)
		close(entered)
		select {
		case <-release:
			return nil
		case <-ctx.Done():
			return context.Cause(ctx)
		}
	}))
	var runErr error
	go func() {
		defer close(done)
		_, runErr = a.runOwnedHost(context.Background(), task, host, StartTaskRequest{}, "test", "A", false)
	}()
	releaseRunner := sync.OnceFunc(func() { close(release) })
	t.Cleanup(func() { releaseRunner(); waitOwnership(t, done, "A cleanup join") })
	waitOwnership(t, entered, "A executor")
	var initialA DeployTaskAttempt
	var initialH DeployHostLease
	if err := db.Where("task_host_id = ?", host.ID).First(&initialA).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Where("host_id = ?", host.HostID).First(&initialH).Error; err != nil {
		t.Fatal(err)
	}
	// A remains blocked across both original injected lease horizons.
	timer := time.NewTimer(3500 * time.Millisecond)
	<-timer.C
	if _, err := b.ReconcileDeployAttempts(time.Now().Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	if attempt := b.claimAttempt(task.ID, host.ID, "B"); attempt != nil {
		t.Fatal("same-task duplicate execution admitted")
	}
	other := DeployTask{Name: "competitor", Status: TaskStatusRunning, ExecutorType: ExecutorTypeSSH}
	if err := db.Create(&other).Error; err != nil {
		t.Fatal(err)
	}
	rival := DeployTaskHost{TaskID: other.ID, HostID: host.HostID, Status: TaskHostStatusRunning}
	if err := db.Create(&rival).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := b.claimOwnedAttempt(other.ID, rival.ID, "C"); err == nil || err.Error() != errDeployTaskLeaseConflict {
		t.Fatalf("cross-task contention: %v", err)
	}
	var currentA DeployTaskAttempt
	var currentH DeployHostLease
	db.First(&currentA, initialA.ID)
	db.Where("host_id = ?", host.HostID).First(&currentH)
	if currentA.LeaseExpiresAt == nil || !currentA.LeaseExpiresAt.After(*initialA.LeaseExpiresAt) || !currentH.ExpiresAt.After(initialH.ExpiresAt) || currentH.TaskID != task.ID {
		t.Fatal("both leases did not renew under A")
	}
	releaseRunner()
	waitOwnership(t, done, "A joined")
	if runErr != nil || executions.Load() != 1 {
		t.Fatalf("execution: %v count=%d", runErr, executions.Load())
	}
	db.First(&task, task.ID)
	db.First(&host, host.ID)
	if task.Status != TaskStatusSuccess || host.Status != TaskHostStatusSuccess {
		t.Fatal("owned finish did not settle")
	}
}

func TestOwnershipStaleCallbacksCannotChangeAnyTable(t *testing.T) {
	db, a, b, task, host := ownershipFixture(t)
	inner, ctx := ownedClaim(t, a, task, host, "old")
	stop := inner.maintainOwnership(inner.ownership)
	defer stop()
	// Revoke A and expire its claim under the persisted serialization keys.
	if err := b.fencedTransaction(task.ID, func(tx *gorm.DB, _ lockedDeployment) error {
		if err := tx.Model(&DeployTaskAttempt{}).Where("id = ?", inner.ownership.attemptID).Update("lease_expires_at", gorm.Expr("CURRENT_TIMESTAMP(6) - INTERVAL 1 SECOND")).Error; err != nil {
			return err
		}
		return tx.Where("host_id = ?", host.HostID).Delete(&DeployHostLease{}).Error
	}); err != nil {
		t.Fatal(err)
	}
	if err := inner.renewOwnership(inner.ownership); err == nil {
		t.Fatal("revoked renewal accepted")
	}
	waitOwnership(t, ctx.Done(), "revocation cancellation")
	if _, err := b.ReconcileDeployAttempts(time.Now()); err != nil {
		t.Fatal(err)
	}
	newer, _ := ownedClaim(t, b, task, host, "new")
	defer newer.ownership.fail(context.Canceled)
	var step DeployTaskStep
	db.Where("task_host_id = ?", host.ID).First(&step)
	var oldAttempt DeployTaskAttempt
	db.First(&oldAttempt, inner.ownership.attemptID)
	callbacks := []struct {
		name string
		run  func()
	}{
		{"log", func() { inner.WriteLog(task.ID, host.ID, "owned", "stdout", "stale") }},
		{"event", func() {
			_ = inner.appendTaskEvent(taskEventInput{taskID: task.ID, taskHostID: host.ID, eventType: "stale"})
		}},
		{"checkpoint/history", func() { _ = inner.updateTaskStepStatus(&step, StepStatusSuccess, "", "") }},
		{"trace", func() { _ = inner.appendTaskHostTrace(task.ID, host.ID, []map[string]any{{"phase": "stale"}}) }},
		{"result/component", func() {
			_, _ = inner.markHostResultWithSummary(host.ID, MarkHostResultRequest{Status: TaskHostStatusSuccess}, "old", nil, deployExecutionSummary{})
		}},
		{"component-helper", func() {
			_ = inner.upsertHostInstalledComponent(host.HostID, task.ID, "old", ExecutorTypeSSH, deployPackageRef{name: "stale", version: "9"}, "old", time.Now())
		}},
		{"finish/release/requeue", func() { _ = inner.completeOwnedAttempt("success", nil, "old", true) }},
		{"final-task", func() { _ = inner.recomputeTaskStatus(task.ID, "old") }},
		{"legacy-finish", func() { a.finishAttempt(inner.ownership, nil) }},
	}
	for _, cb := range callbacks {
		t.Run(cb.name, func(t *testing.T) {
			before := ownershipFacts(t, db)
			inner = a.ownedService(tokenForAttempt(oldAttempt, host))
			cb.run()
			if after := ownershipFacts(t, db); after != before {
				t.Fatal("stale callback mutated fenced state")
			}
		})
	}
}

func TestOwnershipWatchdogDoesNotWaitForDBOrResurrect(t *testing.T) {
	db, a, _, task, host := ownershipFixture(t)
	a.leaseTiming = &ownershipTiming{700 * time.Millisecond, time.Second, 100 * time.Millisecond, 5 * time.Second, 100 * time.Millisecond}
	inner, ctx := ownedClaim(t, a, task, host, "watchdog")
	blocked, reply := make(chan struct{}), make(chan struct{})
	var once sync.Once
	var block atomic.Bool
	block.Store(true)
	if err := db.Callback().Query().After("gorm:query").Register("ownership_delayed_reply", func(tx *gorm.DB) {
		if tx.Statement.Table == "biz_deploy_host_lease" && block.CompareAndSwap(true, false) {
			once.Do(func() { close(blocked) })
			<-reply
		}
	}); err != nil {
		t.Fatal(err)
	}
	stop := sync.OnceFunc(inner.maintainOwnership(inner.ownership))
	releaseReply := sync.OnceFunc(func() { close(reply) })
	t.Cleanup(func() { releaseReply(); stop() })
	waitOwnership(t, blocked, "delayed DB reply")
	waitOwnership(t, ctx.Done(), "independent monotonic watchdog")
	inner.ownership.mu.Lock()
	before := inner.ownership.deadline
	inner.ownership.mu.Unlock()
	releaseReply()
	stop()
	inner.ownership.mu.Lock()
	after := inner.ownership.deadline
	failure := inner.ownership.failure
	inner.ownership.mu.Unlock()
	if failure == nil || !after.Equal(before) {
		t.Fatal("late success resurrected ownership")
	}
	if err := inner.renewOwnership(inner.ownership); err == nil {
		t.Fatal("renewal resurrected failed ownership")
	}
}

func TestOwnershipFirstReadErrorCancelsVoidWriter(t *testing.T) {
	db, a, _, task, host := ownershipFixture(t)
	inner, ctx := ownedClaim(t, a, task, host, "read-error")
	before := ownershipFacts(t, db)
	injected := errors.New("owned database read failed")
	if err := db.Callback().Query().Before("gorm:query").Register("ownership_read_error", func(tx *gorm.DB) {
		if tx.Statement.Table == "biz_deploy_task_step" {
			tx.AddError(injected)
		}
	}); err != nil {
		t.Fatal(err)
	}
	inner.WriteLog(task.ID, host.ID, "owned", "stdout", "do not persist")
	waitOwnership(t, ctx.Done(), "void writer cancellation")
	if !errors.Is(context.Cause(ctx), injected) {
		t.Fatalf("lost cancel cause: %v", context.Cause(ctx))
	}
	// Remove injection solely to inspect committed facts.
	db.Callback().Query().Remove("ownership_read_error")
	if ownershipFacts(t, db) != before {
		t.Fatal("failed log altered DB")
	}
}

func TestOwnershipUnknownOuterTransactionsRejected(t *testing.T) {
	db, a, _, task, host := ownershipFixture(t)
	for _, isolation := range []sql.IsolationLevel{sql.LevelDefault, sql.LevelRepeatableRead, sql.LevelReadCommitted} {
		if err := db.Transaction(func(tx *gorm.DB) error {
			inner := a.targetTransactionService(tx, cmdb.NewDeployCMDBCapability(db).WithTargetTransaction(tx).(cmdb.DeployCMDBCapability))
			for _, write := range []func() error{
				func() error {
					_, err := inner.MarkHostResult(host.ID, MarkHostResultRequest{Status: TaskHostStatusSuccess}, "manual", nil)
					return err
				},
				func() error {
					return inner.upsertHostInstalledComponent(host.HostID, task.ID, task.Name, ExecutorTypeSSH, deployPackageRef{name: "unsafe", version: "1"}, "manual", time.Now())
				},
			} {
				if err := write(); err == nil || !strings.Contains(err.Error(), "transactionIsolationUnknown") {
					t.Fatalf("unmarked %s entry accepted: %v", isolation, err)
				}
			}
			return nil
		}, &sql.TxOptions{Isolation: isolation}); err != nil {
			t.Fatal(err)
		}
	}
}

type ownershipCommitPool struct {
	gorm.ConnPool
	sqlDB           *sql.DB
	fail            error
	late            <-chan struct{}
	rollbackEntered chan struct{}
	rollbackRelease <-chan struct{}
	committed       chan struct{}
	commitReply     <-chan struct{}
}

type ownershipObservedPool struct {
	gorm.ConnPool
	sqlDB  *sql.DB
	begins *atomic.Int32
	bad    *atomic.Bool
}

func (p ownershipObservedPool) BeginTx(ctx context.Context, opts *sql.TxOptions) (gorm.ConnPool, error) {
	p.begins.Add(1)
	if opts == nil || opts.Isolation != sql.LevelReadCommitted {
		p.bad.Store(true)
	}
	return p.sqlDB.BeginTx(ctx, opts)
}

func TestOwnershipEveryProjectionEntryBeginsRCBeforeRead(t *testing.T) {
	type projectionBarrierKey struct{}
	for _, entry := range []string{"manual", "component-helper", "owned-finish"} {
		t.Run(entry, func(t *testing.T) {
			db, a, _, task, host := ownershipFixture(t)
			writer := a
			if entry == "owned-finish" {
				writer, _ = ownedClaim(t, a, task, host, "projection")
			}
			var begins atomic.Int32
			var bad atomic.Bool
			sqlDB, err := db.DB()
			if err != nil {
				t.Fatal(err)
			}
			isolated := db.Session(&gorm.Session{NewDB: true}).WithContext(context.WithValue(context.Background(), projectionBarrierKey{}, true))
			isolated.Statement.ConnPool = ownershipObservedPool{ConnPool: sqlDB, sqlDB: sqlDB, begins: &begins, bad: &bad}
			writer.db = isolated
			var barrier atomic.Bool
			var barrierErr error
			if err := db.Callback().Query().After("gorm:query").Register("projection_first_read_barrier", func(tx *gorm.DB) {
				if tx.Statement.Context.Value(projectionBarrierKey{}) != true || !barrier.CompareAndSwap(false, true) {
					return
				}
				if begins.Load() != 1 || tx.Statement.Context.Value(deploymentTxKey{}) != true {
					bad.Store(true)
				}
				// A has read inside its OUTERMOST transaction. B now commits an unrelated
				// component before A locks the CMDB host and merges its result.
				barrierErr = db.Transaction(func(b *gorm.DB) error {
					bound := cmdb.NewDeployCMDBCapability(db).WithTargetTransaction(b).(cmdb.DeployCMDBCapability)
					if _, err := bound.ResolveDeployTargets(cmdb.DeployHostResolveRequest{TargetType: TargetTypeHost, TargetIDs: []uint64{host.HostID}}); err != nil {
						return err
					}
					return bound.WriteDeployHostResult(cmdb.DeployHostWritebackRequest{HostID: host.HostID, InstalledComponents: []cmdb.InstalledComponentUpsert{{Name: "B", Version: "1", DeployedAt: time.Now()}}})
				}, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
			}); err != nil {
				t.Fatal(err)
			}
			switch entry {
			case "manual":
				_, err = writer.MarkHostResult(host.ID, MarkHostResultRequest{Status: TaskHostStatusSuccess}, "manual", nil)
			case "component-helper":
				err = writer.upsertHostInstalledComponent(host.HostID, task.ID, task.Name, ExecutorTypeSSH, deployPackageRef{name: "owned", version: "1"}, "legacy", time.Now())
			case "owned-finish":
				err = writer.completeOwnedAttempt("success", nil, "worker", false)
			}
			if err != nil {
				t.Fatal(err)
			}
			if barrierErr != nil {
				t.Fatal(barrierErr)
			}
			if !barrier.Load() || begins.Load() != 1 || bad.Load() {
				t.Fatal("entry did not select outermost RC before its first read")
			}
			var h deployTestHost
			if err := db.First(&h, host.HostID).Error; err != nil {
				t.Fatal(err)
			}
			if !containsComponents(h.InstalledComponents, "B", "owned") {
				t.Fatalf("entry lost B: %s", h.InstalledComponents)
			}
		})
	}
}

func (p ownershipCommitPool) BeginTx(ctx context.Context, opts *sql.TxOptions) (gorm.ConnPool, error) {
	tx, err := p.sqlDB.BeginTx(ctx, opts)
	if err != nil {
		return nil, err
	}
	return &ownershipCommitTx{Tx: tx, fail: p.fail, late: p.late, rollbackEntered: p.rollbackEntered, rollbackRelease: p.rollbackRelease, committed: p.committed, commitReply: p.commitReply}, nil
}

type ownershipCommitTx struct {
	*sql.Tx
	fail            error
	late            <-chan struct{}
	rollbackEntered chan struct{}
	rollbackRelease <-chan struct{}
	rollbackOnce    sync.Once
	committed       chan struct{}
	commitReply     <-chan struct{}
}

func (tx *ownershipCommitTx) Commit() error {
	if tx.late != nil {
		<-tx.late
	}
	if tx.fail != nil {
		if tx.rollbackEntered == nil {
			_ = tx.Tx.Rollback()
		}
		return tx.fail
	}
	err := tx.Tx.Commit()
	if err == nil && tx.committed != nil {
		close(tx.committed)
		<-tx.commitReply
	}
	return err
}

func TestOwnershipFinishCommitAndQueuedRenewalHaveOneLifecycle(t *testing.T) {
	db, a, b, task, host := ownershipFixture(t)
	inner, ctx := ownedClaim(t, a, task, host, "finish-renew")
	committed, reply, finished := make(chan struct{}), make(chan struct{}), make(chan struct{})
	releaseReply := sync.OnceFunc(func() { close(reply) })
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	isolated := db.Session(&gorm.Session{NewDB: true}).WithContext(context.Background())
	isolated.Statement.ConnPool = ownershipCommitPool{ConnPool: sqlDB, sqlDB: sqlDB, committed: committed, commitReply: reply}
	inner.db = isolated
	var finishErr, renewErr error
	go func() { defer close(finished); finishErr = inner.completeOwnedAttempt("success", nil, "worker", false) }()
	t.Cleanup(func() { releaseReply(); waitOwnership(t, finished, "finish cleanup join") })
	// The real SQL commit has released locks/deleted the lease, while its reply
	// is still in flight. A queued renewal must wait for local retirement.
	waitOwnership(t, committed, "actual finish SQL commit")
	renewStarted, renewed := make(chan struct{}), make(chan struct{})
	go func() { defer close(renewed); close(renewStarted); renewErr = inner.renewOwnership(inner.ownership) }()
	t.Cleanup(func() { releaseReply(); waitOwnership(t, renewed, "queued renewal cleanup join") })
	waitOwnership(t, renewStarted, "queued renewal")
	select {
	case <-renewed:
		t.Fatal("renewal read retired DB ownership before finish reply")
	case <-time.After(50 * time.Millisecond):
	}
	releaseReply()
	waitOwnership(t, finished, "finish reply/retirement")
	waitOwnership(t, renewed, "queued renewal retirement")
	if finishErr != nil || !errors.Is(renewErr, errExecutionOwnership) || ctx.Err() != nil {
		t.Fatalf("finish=%v renew=%v cancellation=%v", finishErr, renewErr, context.Cause(ctx))
	}
	before := ownershipFacts(t, db)
	if _, err := b.ReconcileDeployAttempts(time.Now()); err != nil || ownershipFacts(t, db) != before {
		t.Fatal("reconcile changed the committed winner")
	}
}

func (tx *ownershipCommitTx) Rollback() error {
	if tx.rollbackEntered != nil {
		tx.rollbackOnce.Do(func() { close(tx.rollbackEntered) })
		<-tx.rollbackRelease
	}
	return tx.Tx.Rollback()
}

func TestOwnershipCommitErrorCancelsBeforeBlockedRollback(t *testing.T) {
	db, a, _, task, host := ownershipFixture(t)
	inner, ctx := ownedClaim(t, a, task, host, "rollback-block")
	before := ownershipFacts(t, db)
	injected := errors.New("commit reply failed")
	entered, release, done := make(chan struct{}), make(chan struct{}), make(chan struct{})
	var releaseOnce sync.Once
	t.Cleanup(func() { releaseOnce.Do(func() { close(release) }); waitOwnership(t, done, "rollback joined") })
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	isolated := db.Session(&gorm.Session{NewDB: true}).WithContext(context.Background())
	isolated.Statement.ConnPool = ownershipCommitPool{ConnPool: sqlDB, sqlDB: sqlDB, fail: injected, rollbackEntered: entered, rollbackRelease: release}
	inner.db = isolated
	var renewalErr error
	go func() { defer close(done); renewalErr = inner.renewOwnership(inner.ownership) }()
	waitOwnership(t, entered, "GORM blocked rollback")
	select {
	case <-ctx.Done():
	case <-time.After(200 * time.Millisecond):
		t.Fatal("commit error cancellation waited for rollback")
	}
	if !errors.Is(context.Cause(ctx), injected) {
		t.Fatalf("commit cause: %v", context.Cause(ctx))
	}
	releaseOnce.Do(func() { close(release) })
	waitOwnership(t, done, "renewal/rollback join")
	if !errors.Is(renewalErr, injected) {
		t.Fatal(renewalErr)
	}
	if ownershipFacts(t, db) != before {
		t.Fatal("failed commit partly persisted leases")
	}
}

func TestOwnershipCommitFailureCancelsAndRollsBackBothRenewals(t *testing.T) {
	db, a, _, task, host := ownershipFixture(t)
	inner, ctx := ownedClaim(t, a, task, host, "commit-failure")
	before := ownershipFacts(t, db)
	injected := errors.New("owned commit failed")
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	isolated := db.Session(&gorm.Session{NewDB: true}).WithContext(context.Background())
	isolated.Statement.ConnPool = ownershipCommitPool{ConnPool: sqlDB, sqlDB: sqlDB, fail: injected}
	inner.db = isolated
	if err := inner.renewOwnership(inner.ownership); !errors.Is(err, injected) {
		t.Fatalf("commit failure: %v", err)
	}
	waitOwnership(t, ctx.Done(), "commit cancels")
	if !errors.Is(context.Cause(ctx), injected) {
		t.Fatal("commit cancel cause lost")
	}
	if ownershipFacts(t, db) != before {
		t.Fatal("failed renewal partly committed")
	}
}

func TestOwnershipBlockedMySQLLockCancelsBeforeRunnerReturns(t *testing.T) {
	db, a, _, task, host := ownershipFixture(t)
	a.leaseTiming = &ownershipTiming{time.Second, 2 * time.Second, 100 * time.Millisecond, 200 * time.Millisecond, 100 * time.Millisecond}
	inner, ctx := ownedClaim(t, a, task, host, "blocked-db")
	locked, release, done := make(chan struct{}), make(chan struct{}), make(chan struct{})
	var blockerErr error
	go func() {
		defer close(done)
		blockerErr = db.Transaction(func(tx *gorm.DB) error {
			var row DeployTaskHost
			if err := tx.Raw("SELECT * FROM biz_deploy_task_host WHERE id = ? FOR UPDATE", host.ID).Scan(&row).Error; err != nil {
				return err
			}
			close(locked)
			<-release
			return nil
		})
	}()
	releaseLock := sync.OnceFunc(func() { close(release) })
	t.Cleanup(func() { releaseLock(); waitOwnership(t, done, "blocker cleanup join") })
	waitOwnership(t, locked, "real MySQL row lock")
	renewalDone := make(chan struct{})
	var renewalErr error
	go func() { defer close(renewalDone); renewalErr = inner.renewOwnership(inner.ownership) }()
	t.Cleanup(func() { releaseLock(); waitOwnership(t, renewalDone, "renewal cleanup join") })
	waitOwnership(t, ctx.Done(), "DB deadline cancellation")
	releaseLock()
	waitOwnership(t, done, "lock owner joins")
	waitOwnership(t, renewalDone, "renew joins")
	if blockerErr != nil {
		t.Fatal(blockerErr)
	}
	if renewalErr == nil {
		t.Fatal("blocked renewal accepted")
	}
	if err := inner.ownership.check(); err == nil {
		t.Fatal("canceled lease resurrected")
	}
}

func TestOwnershipRevokedRunnerAndLateExecutorReturn(t *testing.T) {
	db, a, b, task, host := ownershipFixture(t)
	a.leaseTiming = &ownershipTiming{time.Second, 2 * time.Second, 100 * time.Millisecond, time.Second, 100 * time.Millisecond}
	entered, canceled, late, done := make(chan struct{}), make(chan struct{}), make(chan struct{}), make(chan struct{})
	a.SetExecutor(ownershipExecutorFunc(func(ctx context.Context, _ DeployExecutionHostRequest) error {
		close(entered)
		<-ctx.Done()
		close(canceled)
		<-late
		// A stale runner deliberately ignores cancellation and reports success.
		return nil
	}))
	var runErr error
	go func() {
		defer close(done)
		_, runErr = a.runOwnedHost(context.Background(), task, host, StartTaskRequest{}, "old", "runner-A", false)
	}()
	releaseLate := sync.OnceFunc(func() { close(late) })
	t.Cleanup(func() { releaseLate(); waitOwnership(t, done, "late executor cleanup join") })
	waitOwnership(t, entered, "runner start")
	if err := b.fencedTransaction(task.ID, func(tx *gorm.DB, _ lockedDeployment) error {
		return tx.Model(&DeployHostLease{}).Where("host_id = ?", host.HostID).Updates(map[string]any{"task_id": task.ID + 99, "owner": "replacement", "expires_at": gorm.Expr("CURRENT_TIMESTAMP(6) + INTERVAL 30 MINUTE")}).Error
	}); err != nil {
		t.Fatal(err)
	}
	waitOwnership(t, canceled, "replaced physical lease cancels runner")
	before := ownershipFacts(t, db)
	releaseLate()
	waitOwnership(t, done, "late executor joins")
	if runErr == nil {
		t.Fatal("late success accepted")
	}
	if ownershipFacts(t, db) != before {
		t.Fatal("late executor mutated DB")
	}
}

func TestOwnershipProjectionRollbackIsAtomic(t *testing.T) {
	db, a, _, task, host := ownershipFixture(t)
	inner, _ := ownedClaim(t, a, task, host, "projection-failure")
	var checkpoint DeployTaskStep
	db.Where("task_host_id = ?", host.ID).First(&checkpoint)
	if err := inner.updateTaskStepStatus(&checkpoint, StepStatusRunning, "", ""); err != nil {
		t.Fatal(err)
	}
	if err := inner.updateTaskStepStatus(&checkpoint, StepStatusSuccess, "", ""); err != nil {
		t.Fatal(err)
	}
	before := ownershipFacts(t, db)
	injected := errors.New("CMDB projection refused")
	if err := db.Callback().Update().Before("gorm:update").Register("ownership_projection_failure", func(tx *gorm.DB) {
		if tx.Statement.Table == "biz_cmdb_host" {
			tx.AddError(injected)
		}
	}); err != nil {
		t.Fatal(err)
	}
	if err := inner.completeOwnedAttempt("success", nil, "test", false); !errors.Is(err, injected) {
		t.Fatalf("projection failure: %v", err)
	}
	db.Callback().Update().Remove("ownership_projection_failure")
	if ownershipFacts(t, db) != before {
		t.Fatal("projection failure did not rollback host/attempt/lease/trace/task")
	}
}

func TestOwnershipCrashBudgetPauseCancelAndHistory(t *testing.T) {
	db, a, b, task, host := ownershipFixture(t)
	inner, _ := ownedClaim(t, a, task, host, "crashed")
	var step DeployTaskStep
	db.Where("task_host_id = ?", host.ID).First(&step)
	if err := inner.updateTaskStepStatus(&step, StepStatusRunning, "", ""); err != nil {
		t.Fatal(err)
	}
	if err := inner.updateTaskStepStatus(&step, StepStatusSuccess, "", ""); err != nil {
		t.Fatal(err)
	}
	// Crash: no renewal goroutine survives, expire both retained claims.
	if err := db.Model(&DeployTaskAttempt{}).Where("id = ?", inner.ownership.attemptID).Update("lease_expires_at", gorm.Expr("CURRENT_TIMESTAMP(6) - INTERVAL 1 SECOND")).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&DeployHostLease{}).Where("host_id = ?", host.HostID).Update("expires_at", gorm.Expr("CURRENT_TIMESTAMP(6) - INTERVAL 1 SECOND")).Error; err != nil {
		t.Fatal(err)
	}
	if changed, err := b.ReconcileDeployAttempts(time.Now()); err != nil || changed != 1 {
		t.Fatalf("crash reconcile: %d %v", changed, err)
	}
	recovered, _ := ownedClaim(t, b, task, host, "recovered")
	if err := db.First(&step, step.ID).Error; err != nil {
		t.Fatal(err)
	}
	if step.Status != StepStatusSuccess || step.AttemptNo != 1 {
		t.Fatal("recovery lost successful checkpoint")
	}
	if err := db.Model(&DeployTask{}).Where("id = ?", task.ID).Update("status", TaskStatusPauseRequested).Error; err != nil {
		t.Fatal(err)
	}
	if err := recovered.completeOwnedAttempt(TaskStatusPaused, nil, "test", false); err != nil {
		t.Fatal(err)
	}
	db.First(&task, task.ID)
	if task.Status != TaskStatusPaused {
		t.Fatal("pause not settled")
	}
	if err := db.Model(&DeployTask{}).Where("id = ?", task.ID).Update("status", TaskStatusRunning).Error; err != nil {
		t.Fatal(err)
	}
	// The paused attempt is free, but its monotonic number is retained.
	third, _ := ownedClaim(t, a, task, host, "third-number")
	var attempt DeployTaskAttempt
	db.First(&attempt, third.ownership.attemptID)
	if attempt.AttemptNo != 3 {
		t.Fatal("paused history reset")
	}
	if err := third.completeOwnedAttempt("failed", errors.New("failed chargeable two"), "test", true); err != nil {
		t.Fatal(err)
	}
	last, _ := ownedClaim(t, b, task, host, "third-charge")
	if err := last.completeOwnedAttempt("failed", errors.New("failed chargeable three"), "test", true); err != nil {
		t.Fatal(err)
	}
	db.First(&task, task.ID)
	db.First(&host, host.ID)
	if task.Status != TaskStatusFailed || host.Status != TaskHostStatusFailed {
		t.Fatal("third chargeable attempt not exhausted")
	}
	if attempt := a.claimAttempt(task.ID, host.ID, "fourth-charge"); attempt != nil {
		t.Fatal("budget exceeded")
	}
	var history []DeployStepHistory
	if err := db.Where("task_id = ?", task.ID).Order("id").Find(&history).Error; err != nil {
		t.Fatal(err)
	}
	if len(history) != 2 {
		t.Fatalf("history lost: %d", len(history))
	}
	// A separate task proves cancellation keeps successful checkpoint history.
	cancelTask := DeployTask{Name: "cancel", Status: TaskStatusRunning, ExecutorType: ExecutorTypeSSH, Action: TaskActionInstall, BusinessScopeID: task.BusinessScopeID}
	if err := db.Create(&cancelTask).Error; err != nil {
		t.Fatal(err)
	}
	cancelHost := DeployTaskHost{TaskID: cancelTask.ID, HostID: host.HostID, Status: TaskHostStatusRunning}
	if err := db.Create(&cancelHost).Error; err != nil {
		t.Fatal(err)
	}
	canceled, _ := ownedClaim(t, a, cancelTask, cancelHost, "cancel-worker")
	if err := db.Model(&DeployTask{}).Where("id = ?", cancelTask.ID).Update("status", TaskStatusCancelRequested).Error; err != nil {
		t.Fatal(err)
	}
	if err := canceled.completeOwnedAttempt(TaskStatusCanceled, nil, "test", false); err != nil {
		t.Fatal(err)
	}
	db.First(&cancelTask, cancelTask.ID)
	db.First(&cancelHost, cancelHost.ID)
	if cancelTask.Status != TaskStatusCanceled || cancelHost.Status != TaskHostStatusSkipped {
		t.Fatal("cancel did not settle")
	}
}

func TestOwnershipReconcileSerializesRenewAndFinish(t *testing.T) {
	type barrierKey struct{}
	for _, operation := range []string{"renew", "finish"} {
		for _, first := range []string{"owner", "reconcile"} {
			t.Run(operation+"/"+first, func(t *testing.T) {
				db, a, b, task, host := ownershipFixture(t)
				inner, _ := ownedClaim(t, a, task, host, "A")
				if first == "reconcile" {
					if err := db.Model(&DeployTaskAttempt{}).Where("id = ?", inner.ownership.attemptID).Update("lease_expires_at", gorm.Expr("CURRENT_TIMESTAMP(6) - INTERVAL 1 SECOND")).Error; err != nil {
						t.Fatal(err)
					}
				}
				locked, release, done := make(chan struct{}), make(chan struct{}), make(chan struct{})
				var gated atomic.Bool
				if err := db.Callback().Query().After("gorm:query").Register("ownership_winner_barrier", func(tx *gorm.DB) {
					if tx.Statement.Table == "biz_deploy_task_host" && tx.Statement.Context.Value(barrierKey{}) == true && gated.CompareAndSwap(false, true) {
						close(locked)
						<-release
					}
				}); err != nil {
					t.Fatal(err)
				}
				winner := inner
				if first == "reconcile" {
					winner = b
				}
				winner.db = winner.db.WithContext(context.WithValue(context.Background(), barrierKey{}, true))
				var winnerErr error
				go func() {
					defer close(done)
					if first == "reconcile" {
						_, winnerErr = winner.ReconcileDeployAttempts(time.Now())
					} else if operation == "renew" {
						winnerErr = winner.renewOwnership(inner.ownership)
					} else {
						winnerErr = winner.completeOwnedAttempt("success", nil, "test", false)
					}
				}()
				waitOwnership(t, locked, "actual winner lock")
				loserStarted, loserDone := make(chan struct{}), make(chan struct{})
				var loserErr error
				go func() {
					defer close(loserDone)
					close(loserStarted)
					if first == "owner" {
						_, loserErr = b.ReconcileDeployAttempts(time.Now())
					} else if operation == "renew" {
						loserErr = inner.renewOwnership(inner.ownership)
					} else {
						loserErr = inner.completeOwnedAttempt("success", nil, "late", false)
					}
				}()
				waitOwnership(t, loserStarted, "loser invocation")
				close(release)
				waitOwnership(t, done, "winner commit")
				waitOwnership(t, loserDone, "loser joins")
				if winnerErr != nil {
					t.Fatal(winnerErr)
				}
				if first == "reconcile" && loserErr == nil {
					t.Fatal("revoked loser committed")
				}
				if first == "owner" && loserErr != nil {
					t.Fatal(loserErr)
				}
				var latest DeployTaskAttempt
				db.First(&latest, inner.ownership.attemptID)
				want := "retryable"
				if first == "owner" {
					want = "running"
					if operation == "finish" {
						want = "success"
					}
				}
				if latest.Status != want {
					t.Fatalf("serialized winner: want %s got %s", want, latest.Status)
				}
			})
		}
	}
}

func TestOwnershipExactDBExpiryAndNullLegacy(t *testing.T) {
	db, a, _, task, host := ownershipFixture(t)
	inner, _ := ownedClaim(t, a, task, host, "expiry")
	for _, field := range []string{"attempt", "physical", "null"} {
		t.Run(field, func(t *testing.T) {
			// Pin one connection and freeze server time to prove equality, rather
			// than allowing a sleep to turn equality into an already-expired case.
			err := db.Connection(func(conn *gorm.DB) error {
				if err := conn.Exec("SET timestamp = 1794300000").Error; err != nil {
					return err
				}
				defer conn.Exec("SET timestamp = DEFAULT")
				local := a.targetTransactionService(conn, cmdb.NewDeployCMDBCapability(conn))
				return local.deploymentTransaction(func(tx *gorm.DB) error {
					var state lockedDeployment
					var err error
					if _, err = lockDeployment(tx, task.ID); err != nil {
						return err
					}
					if err := tx.Model(&DeployTaskAttempt{}).Where("id = ?", inner.ownership.attemptID).Update("lease_expires_at", gorm.Expr("CURRENT_TIMESTAMP(6) + INTERVAL 5 MINUTE")).Error; err != nil {
						return err
					}
					if err := tx.Model(&DeployHostLease{}).Where("host_id = ?", host.HostID).Update("expires_at", gorm.Expr("CURRENT_TIMESTAMP(6) + INTERVAL 30 MINUTE")).Error; err != nil {
						return err
					}
					switch field {
					case "attempt":
						err = tx.Model(&DeployTaskAttempt{}).Where("id = ?", inner.ownership.attemptID).Update("lease_expires_at", gorm.Expr("CURRENT_TIMESTAMP(6)")).Error
					case "physical":
						err = tx.Model(&DeployHostLease{}).Where("host_id = ?", host.HostID).Update("expires_at", gorm.Expr("CURRENT_TIMESTAMP(6)")).Error
					case "null":
						err = tx.Model(&DeployTaskAttempt{}).Where("id = ?", inner.ownership.attemptID).Update("lease_expires_at", nil).Error
					}
					if err != nil {
						return err
					}
					state, err = lockDeployment(tx, task.ID)
					if err != nil {
						return err
					}
					if err := inner.ownership.fence(tx, state); err == nil {
						return errors.New("exact expired/null ownership accepted")
					}
					bound := local.targetTransactionService(tx, cmdb.NewDeployCMDBCapability(tx))
					if active, err := bound.hasActiveTaskAttempt(task.ID, host.ID, time.Now()); err != nil || active {
						return errors.New("exact expired/null ownership reported active")
					}
					if changed, err := bound.ReconcileDeployAttempts(time.Now()); err != nil || changed != 1 {
						return errors.New("exact expired/null ownership was not reconciled")
					}
					if next, err := bound.claimOwnedAttempt(task.ID, host.ID, "replacement"); err != nil || next.AttemptNo != 2 {
						return errors.New("exact expired/null ownership prevented recovery claim")
					}
					return errors.New("expected rollback")
				})
			})
			if err == nil || err.Error() != "expected rollback" {
				t.Fatal(err)
			}
		})
	}
}

func TestOwnershipNullLegacyReconcilesWithoutResettingHistory(t *testing.T) {
	db, a, b, task, host := ownershipFixture(t)
	legacy := DeployTaskAttempt{TaskID: task.ID, TaskHostID: host.ID, AttemptNo: 7, Status: "running", WorkerID: "legacy"}
	if err := db.Create(&legacy).Error; err != nil {
		t.Fatal(err)
	}
	log := DeployTaskLogChunk{TaskID: task.ID, TaskHostID: host.ID, Sequence: 1, AttemptNo: 0, Stream: "stdout", Content: "retained legacy output"}
	if err := db.Create(&log).Error; err != nil {
		t.Fatal(err)
	}
	if active, err := a.hasActiveTaskAttempt(task.ID, host.ID, time.Now()); err != nil || active {
		t.Fatalf("null legacy ownership: %t %v", active, err)
	}
	if n, err := b.ReconcileDeployAttempts(time.Now()); err != nil || n != 1 {
		t.Fatalf("null legacy reconcile: %d %v", n, err)
	}
	var retained DeployTaskLogChunk
	if err := db.First(&retained, log.ID).Error; err != nil {
		t.Fatal(err)
	}
	if retained.AttemptNo != 0 || retained.Content != log.Content {
		t.Fatal("legacy history reconstructed or overwritten")
	}
	next, err := a.claimOwnedAttempt(task.ID, host.ID, "new")
	if err != nil {
		t.Fatal(err)
	}
	if next.AttemptNo != 8 {
		t.Fatal("legacy attempt numbers reset")
	}
	if _, err := a.MarkHostResult(host.ID, MarkHostResultRequest{Status: TaskHostStatusSuccess}, "tokenless", nil); err == nil {
		t.Fatal("legacy writer bypassed new ownership")
	}
}

func containsComponents(raw []byte, names ...string) bool {
	var rows []struct {
		Name string `json:"name"`
	}
	if json.Unmarshal(raw, &rows) != nil {
		return false
	}
	for _, name := range names {
		found := false
		for _, row := range rows {
			found = found || row.Name == name
		}
		if !found {
			return false
		}
	}
	return true
}

// Replays the exact RR snapshot interleaving identified in the approved v3
// review. Both writes use the existing transaction-bound CMDB capability.
func TestOwnershipComponentSnapshot(t *testing.T) {
	db := setupDeployTestDB(t)
	_, hostID := seedDeployReliabilityFixture(t, db, "ownership-snapshot", "10.40.0.231")
	owner := cmdb.NewDeployCMDBCapability(db)
	svc := NewDeployService(db, owner)
	err := svc.deploymentTransaction(func(a *gorm.DB) error {
		var h deployTestHost
		if err := a.First(&h, hostID).Error; err != nil {
			return err
		}
		// B commits after A's plain read, before A takes the ordered CMDB lock.
		if err := db.Transaction(func(b *gorm.DB) error {
			bound := owner.WithTargetTransaction(b).(cmdb.DeployCMDBCapability)
			if _, err := bound.ResolveDeployTargets(cmdb.DeployHostResolveRequest{TargetType: "host", TargetIDs: []uint64{hostID}}); err != nil {
				return err
			}
			return bound.WriteDeployHostResult(cmdb.DeployHostWritebackRequest{HostID: hostID, InstalledComponents: []cmdb.InstalledComponentUpsert{{Name: "B", Version: "1", DeployedAt: time.Now()}}})
		}, &sql.TxOptions{Isolation: sql.LevelReadCommitted}); err != nil {
			return err
		}
		bound := owner.WithTargetTransaction(a).(cmdb.DeployCMDBCapability)
		if _, err := bound.ResolveDeployTargets(cmdb.DeployHostResolveRequest{TargetType: "host", TargetIDs: []uint64{hostID}}); err != nil {
			return err
		}
		return bound.WriteDeployHostResult(cmdb.DeployHostWritebackRequest{HostID: hostID, InstalledComponents: []cmdb.InstalledComponentUpsert{{Name: "A", Version: "1", DeployedAt: time.Now()}}})
	})
	if err != nil {
		t.Fatal(err)
	}
	var h deployTestHost
	if err := db.First(&h, hostID).Error; err != nil {
		t.Fatal(err)
	}
	if !containsComponents(h.InstalledComponents, "A", "B") {
		t.Fatalf("projection lost committed B component: %s", h.InstalledComponents)
	}
}
