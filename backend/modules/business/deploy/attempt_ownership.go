package deploy

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"runtime/debug"
	"sort"
	"sync"
	"time"

	"github.com/duanxldragon/pantheon-base/backend/modules/business/cmdb"
	biztx "github.com/duanxldragon/pantheon-base/backend/modules/business/shared/transaction"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var errExecutionOwnership = errors.New("business.deploy.taskHost.staleReport")

const msgDeployAttemptExhausted = "business.deploy.attempt.exhausted"

// Legacy reports have no worker claim. After durable attempts exist a host
// writer must carry its original token, including after reconciliation.
func (s *DeployService) requireHostWriter(tx *gorm.DB, taskID, hostID uint64) error {
	if s.ownership != nil {
		return nil
	}
	var count int64
	query := tx.Model(&DeployTaskAttempt{}).Where("task_id = ?", taskID)
	if hostID > 0 {
		query = query.Where("task_host_id = ?", hostID)
	}
	if err := query.Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return errExecutionOwnership
	}
	return nil
}

func (s *DeployService) executionTaskStatus(taskID uint64) (string, error) {
	ctx, cancel := context.WithTimeout(s.requestContext(), s.ownershipTiming().operation)
	defer cancel()
	var task DeployTask
	err := s.db.WithContext(ctx).Select("status").First(&task, taskID).Error
	if s.ownership != nil {
		s.ownership.fail(err)
	}
	return task.Status, err
}

// Durations are immutable after service construction; tests shorten both
// horizons without changing the production protocol or the persisted schema.
type ownershipTiming struct{ attempt, physical, cadence, operation, margin time.Duration }

func (s *DeployService) ownershipTiming() ownershipTiming {
	if s.leaseTiming != nil {
		return *s.leaseTiming
	}
	return ownershipTiming{5 * time.Minute, 30 * time.Minute, time.Minute, 5 * time.Second, 10 * time.Second}
}

type executionOwnership struct {
	taskID, taskHostID, physicalHostID, attemptID uint64
	workerID, taskLeaseOwner                      string
	lifecycle                                     sync.Mutex // joins renewal with local retirement; never used by watchdog
	mu                                            sync.Mutex
	retired                                       bool
	deadline                                      time.Time // carries Go's monotonic clock
	failure                                       error
	cancel                                        context.CancelCauseFunc
	lost                                          context.Context
	cancelLost                                    context.CancelCauseFunc
	result                                        *MarkHostResultRequest
	summary                                       deployExecutionSummary
}

func (o *executionOwnership) fail(err error) {
	if err == nil {
		return
	}
	o.mu.Lock()
	if o.failure == nil {
		o.failure = err
		if o.cancelLost != nil {
			o.cancelLost(err)
		}
		if o.cancel != nil {
			o.cancel(err)
		}
	}
	o.mu.Unlock()
}
func (o *executionOwnership) check() error {
	o.mu.Lock()
	err, deadline, retired := o.failure, o.deadline, o.retired
	o.mu.Unlock()
	if err != nil {
		return err
	}
	if retired {
		return errExecutionOwnership
	}
	if !deadline.IsZero() && !time.Now().Before(deadline) {
		o.fail(errExecutionOwnership)
		return errExecutionOwnership
	}
	return nil
}

// Only a successful, timely reply may extend the last confirmed deadline.
func (o *executionOwnership) confirm(start time.Time, remaining time.Duration, timing ownershipTiming) error {
	received := time.Now()
	deadline := start.Add(timing.attempt).Add(-timing.margin)
	observed := received.Add(remaining - received.Sub(start) - timing.margin)
	if observed.Before(deadline) {
		deadline = observed
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.failure != nil {
		return o.failure
	}
	if o.retired {
		return errExecutionOwnership
	}
	if (!o.deadline.IsZero() && !received.Before(o.deadline)) || !received.Before(deadline) {
		o.failure = errExecutionOwnership
		if o.cancelLost != nil {
			o.cancelLost(o.failure)
		}
		if o.cancel != nil {
			o.cancel(o.failure)
		}
		return o.failure
	}
	o.deadline = deadline
	return nil
}

type deploymentTxKey struct{}

var errExecutionInterrupted = errors.New("deploy execution interrupted")

// Projection isolation is set at the OUTERMOST BEGIN, before any read.
// An unmarked existing SQL transaction is rejected, never silently repaired
// by nested GORM options. The marker lives on this transaction's context only.
func (s *DeployService) deploymentTransaction(write func(*gorm.DB) error) error {
	if s.ownership != nil {
		if err := s.ownership.check(); err != nil {
			return err
		}
	}
	ctx, cancel := context.WithTimeout(s.requestContext(), s.ownershipTiming().operation)
	if s.ownership != nil {
		if s.ownership.lost != nil {
			lostJoined := make(chan struct{})
			stopLost := context.AfterFunc(s.ownership.lost, func() { defer close(lostJoined); cancel() })
			defer func() {
				if !stopLost() {
					<-lostJoined
				}
			}()
		}
		joined := make(chan struct{})
		stop := context.AfterFunc(ctx, func() { defer close(joined); s.ownership.fail(ctx.Err()) })
		defer func() {
			if !stop() {
				<-joined
			}
			cancel()
		}()
	} else {
		defer cancel()
	}
	db := s.db.WithContext(biztx.WithReadCommittedContext(ctx, s.db))
	if _, nested := db.Statement.ConnPool.(gorm.TxCommitter); nested {
		if !biztx.IsReadCommittedTransaction(db) {
			return errors.New("deploy.projection.transactionIsolationUnknown")
		}
		err := biztx.ReadCommittedTransaction(ctx, db, write, nil)
		if s.ownership != nil && !errors.Is(err, errExecutionInterrupted) {
			s.ownership.fail(err)
		}
		return err
	}
	ctx = context.WithValue(ctx, deploymentTxKey{}, true)
	var hooks *biztx.ReadCommittedTransactionHooks
	if s.ownership != nil {
		hooks = &biztx.ReadCommittedTransactionHooks{BeforeCommit: s.ownership.check, OnError: s.ownership.fail}
	}
	err := biztx.ReadCommittedTransaction(ctx, db, func(tx *gorm.DB) error {
		if s.ownership != nil {
			if err := s.ownership.check(); err != nil {
				return err
			}
		}
		if err := write(tx); err != nil {
			if s.ownership != nil && !errors.Is(err, errExecutionInterrupted) {
				s.ownership.fail(err)
			}
			return err
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if s.ownership != nil {
			return s.ownership.check()
		}
		return nil
	}, hooks)
	// A driver may return a success after a canceled request. It cannot restore
	// local execution ownership, even if the DB outcome was ambiguous.
	if err == nil {
		err = ctx.Err()
	}
	if s.ownership != nil && !errors.Is(err, errExecutionInterrupted) {
		s.ownership.fail(err)
	}
	return err
}

type lockedDeployment struct {
	task     DeployTask
	hosts    []DeployTaskHost
	attempts []DeployTaskAttempt
	leases   []DeployHostLease
}

// All writers and reconcile paths use task -> hosts -> attempts -> physical
// leases. Pure renewal below omits task entirely and never locks it later.
func lockDeployment(tx *gorm.DB, taskID uint64, reserve ...bool) (lockedDeployment, error) {
	var state lockedDeployment
	lock := clause.Locking{Strength: "UPDATE"}
	if err := tx.Clauses(lock).First(&state.task, taskID).Error; err != nil {
		return state, err
	}
	if err := tx.Clauses(lock).Where("task_id = ?", taskID).Order("id ASC").Find(&state.hosts).Error; err != nil {
		return state, err
	}
	if err := tx.Clauses(lock).Where("task_id = ?", taskID).Order("task_host_id ASC, attempt_no ASC, id ASC").Find(&state.attempts).Error; err != nil {
		return state, err
	}
	ids := make([]uint64, 0, len(state.hosts))
	for _, host := range state.hosts {
		ids = append(ids, host.HostID)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	for _, id := range ids {
		// A terminal replay must not recreate a released physical lease row.
		if len(reserve) > 0 && reserve[0] && !isTerminalTaskStatus(state.task.Status) {
			now, err := databaseNow(tx)
			if err != nil {
				return state, err
			}
			if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&DeployHostLease{HostID: id, TaskID: 0, Owner: "", ExpiresAt: now, CreatedAt: now, UpdatedAt: now}).Error; err != nil {
				return state, err
			}
		}
		var rows []DeployHostLease
		if err := tx.Clauses(lock).Where("host_id = ?", id).Find(&rows).Error; err != nil {
			return state, err
		}
		state.leases = append(state.leases, rows...)
	}

	return state, nil
}

func databaseNow(tx *gorm.DB) (time.Time, error) {
	var now time.Time
	err := tx.Raw("SELECT CURRENT_TIMESTAMP(6)").Scan(&now).Error
	return now, err
}

func (o *executionOwnership) fence(tx *gorm.DB, state lockedDeployment) error {
	if o.workerID == "" || o.taskLeaseOwner != deployHostLeaseOwner(o.taskID) {
		return errExecutionOwnership
	}
	if state.task.ID != o.taskID {
		return errExecutionOwnership
	}
	var host DeployTaskHost
	for _, h := range state.hosts {
		if h.ID == o.taskHostID {
			host = h
		}
	}
	if host.TaskID != o.taskID || host.HostID != o.physicalHostID {
		return errExecutionOwnership
	}
	var latest DeployTaskAttempt
	for _, a := range state.attempts {
		if a.TaskHostID == o.taskHostID && (a.AttemptNo > latest.AttemptNo || (a.AttemptNo == latest.AttemptNo && a.ID > latest.ID)) {
			latest = a
		}
	}
	if latest.ID != o.attemptID || latest.TaskID != o.taskID || latest.WorkerID != o.workerID || latest.Status != TaskHostStatusRunning || latest.LeaseExpiresAt == nil {
		return errExecutionOwnership
	}
	now, err := databaseNow(tx)
	if err != nil {
		return err
	}
	if !latest.LeaseExpiresAt.After(now) {
		return errExecutionOwnership
	}
	for _, lease := range state.leases {
		if lease.HostID == o.physicalHostID && lease.TaskID == o.taskID && lease.Owner == o.taskLeaseOwner && lease.ExpiresAt.After(now) {
			return nil
		}
	}
	return errExecutionOwnership
}

func (s *DeployService) fencedTransaction(taskID uint64, write func(*gorm.DB, lockedDeployment) error) error {
	return s.deploymentTransaction(func(tx *gorm.DB) error {
		state, err := lockDeployment(tx, taskID)
		if err != nil {
			return err
		}
		if s.ownership != nil {
			if err := s.ownership.fence(tx, state); err != nil {
				return err
			}
		}
		if err := write(tx, state); err != nil {
			return err
		}
		if s.ownership != nil {
			return s.ownership.fence(tx, state)
		}
		return nil
	})
}

func (s *DeployService) boundCMDB(tx *gorm.DB, hostIDs []uint64) (cmdb.DeployCMDBCapability, error) {
	owner, ok := s.cmdbCapability.(cmdb.TargetSelectionCapability)
	if !ok {
		return nil, errors.New("business.cmdb.selection.notConfigured")
	}
	bound, ok := owner.WithTargetTransaction(tx).(cmdb.DeployCMDBCapability)
	if !ok {
		return nil, errors.New("business.cmdb.selection.notConfigured")
	}
	ids := append([]uint64(nil), hostIDs...)
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	if len(ids) > 0 {
		rows, err := bound.ResolveDeployTargets(cmdb.DeployHostResolveRequest{TargetType: TargetTypeHost, TargetIDs: ids})
		if err != nil {
			return nil, err
		}
		if len(rows) != len(ids) {
			return nil, errDeployTaskForbidden
		}
	}
	return bound, nil
}

// No sync.Map is copied. Each attempt gets an immutable service view carrying
// its token all the way through SSH, providers and the void log-writer API.
func (s *DeployService) ownedService(o *executionOwnership) *DeployService {
	inner := NewDeployService(s.db, s.cmdbCapability, s.bizScopeReader)
	inner.ownership = o
	inner.leaseTiming = s.leaseTiming
	inner.executionRoot = s
	if s.executionRoot != nil {
		inner.executionRoot = s.executionRoot
	}
	inner.hostReader = s.hostReader
	inner.serviceState = s.serviceState
	inner.serviceStateTransaction = s.serviceStateTransaction
	inner.serviceReader = s.serviceReader
	inner.serviceReaderTransaction = s.serviceReaderTransaction
	inner.sshRunnerFactory = s.sshRunnerFactory
	if _, ssh := s.executor.(*sshDeployExecutor); !ssh {
		inner.executor = s.executor
	}
	return inner
}

func tokenForAttempt(a DeployTaskAttempt, h DeployTaskHost) *executionOwnership {
	lost, cancelLost := context.WithCancelCause(context.Background())
	return &executionOwnership{taskID: a.TaskID, taskHostID: a.TaskHostID, physicalHostID: h.HostID, attemptID: a.ID, workerID: a.WorkerID, taskLeaseOwner: deployHostLeaseOwner(a.TaskID), lost: lost, cancelLost: cancelLost}
}

func (s *DeployService) renewOwnership(o *executionOwnership) error {
	// Keep a queued local renewal from treating our own committed finish as
	// ownership loss. Cross-service exclusion remains the persisted row locks.
	o.lifecycle.Lock()
	defer o.lifecycle.Unlock()
	o.mu.Lock()
	retired := o.retired
	o.mu.Unlock()
	if retired {
		return errExecutionOwnership
	}
	if _, nested := s.db.Statement.ConnPool.(gorm.TxCommitter); nested {
		err := errors.New("deploy.renewal.requiresOutermostTransaction")
		o.fail(err)
		return err
	}
	start := time.Now()
	timing := s.ownershipTiming()
	var remaining time.Duration
	err := s.deploymentTransaction(func(tx *gorm.DB) error {
		var h DeployTaskHost
		var attempts []DeployTaskAttempt
		var lease DeployHostLease
		lock := clause.Locking{Strength: "UPDATE"}
		if err := tx.Clauses(lock).First(&h, o.taskHostID).Error; err != nil {
			return err
		}
		if err := tx.Clauses(lock).Where("task_host_id = ?", o.taskHostID).Order("attempt_no ASC, id ASC").Find(&attempts).Error; err != nil {
			return err
		}
		if err := tx.Clauses(lock).Where("host_id = ?", o.physicalHostID).First(&lease).Error; err != nil {
			return err
		}
		state := lockedDeployment{task: DeployTask{ID: o.taskID}, hosts: []DeployTaskHost{h}, attempts: attempts, leases: []DeployHostLease{lease}}
		if err := o.fence(tx, state); err != nil {
			return err
		}
		now, err := databaseNow(tx)
		if err != nil {
			return err
		}
		nextAttempt, nextHost := now.Add(timing.attempt), now.Add(timing.physical)
		a := tx.Model(&DeployTaskAttempt{}).Where("id = ? AND worker_id = ? AND status = 'running' AND lease_expires_at > CURRENT_TIMESTAMP(6)", o.attemptID, o.workerID).Updates(map[string]any{"lease_expires_at": nextAttempt, "updated_at": now})
		if a.Error != nil {
			return a.Error
		}
		if a.RowsAffected != 1 {
			return errExecutionOwnership
		}
		l := tx.Model(&DeployHostLease{}).Where("host_id = ? AND task_id = ? AND owner = ? AND expires_at > CURRENT_TIMESTAMP(6)", o.physicalHostID, o.taskID, o.taskLeaseOwner).Updates(map[string]any{"expires_at": nextHost, "updated_at": now})
		if l.Error != nil {
			return l.Error
		}
		if l.RowsAffected != 1 {
			return errExecutionOwnership
		}
		observed, err := databaseNow(tx)
		if err != nil {
			return err
		}
		remaining = nextAttempt.Sub(observed)
		if r := nextHost.Sub(observed); r < remaining {
			remaining = r
		}
		return nil
	})
	if err != nil {
		o.fail(err)
		return err
	}
	return o.confirm(start, remaining, timing)
}

func (s *DeployService) maintainOwnership(o *executionOwnership) func() {
	stop := make(chan struct{})
	var wg sync.WaitGroup
	timing := s.ownershipTiming()
	wg.Add(2)
	go func() {
		defer wg.Done()
		ticker := time.NewTicker(timing.cadence)
		defer ticker.Stop()
		for {
			select {
			case <-stop:
				return
			case <-ticker.C:
				if s.renewOwnership(o) != nil {
					return
				}
				status, err := s.executionTaskStatus(o.taskID)
				if err != nil {
					return
				}
				if status != TaskStatusRunning {
					if o.cancel != nil {
						o.cancel(errExecutionInterrupted)
					}
					return
				}

			}
		}
	}()
	go func() {
		defer wg.Done()
		for {
			o.mu.Lock()
			deadline, failed := o.deadline, o.failure
			o.mu.Unlock()
			if failed != nil {
				return
			}
			wait := time.Until(deadline)
			if wait <= 0 {
				o.fail(errExecutionOwnership)
				return
			}
			// Wake after a confirmed renewal without depending on the DB goroutine.
			if wait > timing.cadence {
				wait = timing.cadence
			}
			timer := time.NewTimer(wait)
			select {
			case <-stop:
				timer.Stop()
				return
			case <-timer.C:
			}
		}
	}()
	return func() { close(stop); wg.Wait() }
}

func (s *DeployService) assertCommandOwnership() error {
	if s.ownership == nil {
		return nil
	}
	return s.fencedTransaction(s.ownership.taskID, func(_ *gorm.DB, state lockedDeployment) error {
		if state.task.Status != TaskStatusRunning {
			return fmt.Errorf("%w: %s", errExecutionInterrupted, state.task.Status)
		}
		return nil
	})
}

func appendTaskEventTx(tx *gorm.DB, event taskEventInput) error {
	var sequence uint64
	if err := tx.Model(&DeployTaskEvent{}).Where("task_id = ?", event.taskID).Select("COALESCE(MAX(sequence),0)").Scan(&sequence).Error; err != nil {
		return err
	}
	return tx.Create(&DeployTaskEvent{TaskID: event.taskID, TaskHostID: event.taskHostID, TaskStepID: event.taskStepID, Sequence: sequence + 1, EventType: event.eventType, FromStatus: event.fromStatus, ToStatus: event.toStatus, Actor: event.actor, RequestID: event.requestID, Reason: event.reason, CreatedAt: time.Now()}).Error
}

func (s *DeployService) runOwnedHost(parent context.Context, task DeployTask, host DeployTaskHost, req StartTaskRequest, actor, worker string, retry bool) (bool, error) {
	// A canceled worker must stop its claim even while waiting for a task lock.
	a, err := s.targetRequestService(parent).claimOwnedAttempt(task.ID, host.ID, worker)
	if err != nil {
		return false, err
	}
	o := tokenForAttempt(*a, host)
	ctx, cancel := context.WithCancelCause(parent)
	defer cancel(context.Canceled)
	o.cancel = cancel
	inner := s.ownedService(o)
	// Confirm both leases before any executor/SSH side effect.
	if err := inner.renewOwnership(o); err != nil {
		return false, err
	}
	stop := inner.maintainOwnership(o)
	defer stop()
	// A panicking executor would otherwise take down the whole server: the
	// attempt runs on a background goroutine, outside gin's recovery. Convert
	// it to a failed host result so the task stays visible and settleable.
	runErr := func() (err error) {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("deploy executor panic recovered", "task_id", task.ID, "task_host_id", host.ID, "panic", r, "stack", string(debug.Stack()))
				err = fmt.Errorf("business.deploy.executor.panic: %v", r)
			}
		}()
		return inner.executeAsyncHostAttempt(ctx, task, host, req, actor)
	}()
	if err := o.check(); err != nil {
		return false, err
	}
	o.mu.Lock()
	result := o.result
	o.mu.Unlock()
	failed := runErr != nil
	if result != nil {
		failed = result.Status != TaskHostStatusSuccess
		if failed && runErr == nil {
			runErr = errors.New(result.ErrorMessage)
		}
	}
	status := "success"
	if failed {
		status = TaskHostStatusFailed
	}
	if err := inner.completeOwnedAttempt(status, runErr, actor, retry); err != nil {
		return failed, err
	}
	// Retired tokens cannot issue a post-finish callback.
	o.fail(errExecutionOwnership)
	return failed, nil
}

func appendHostTraceTx(tx *gorm.DB, taskID, hostID uint64, steps []map[string]any) error {
	var host DeployTaskHost
	if err := tx.Where("id = ? AND task_id = ?", hostID, taskID).First(&host).Error; err != nil {
		return err
	}
	var current []map[string]any
	if len(host.TraceSteps) > 0 {
		if err := json.Unmarshal(host.TraceSteps, &current); err != nil {
			return err
		}
	}
	payload, err := json.Marshal(append(current, steps...))
	if err != nil {
		return err
	}
	return tx.Model(&DeployTaskHost{}).Where("id = ? AND task_id = ?", hostID, taskID).Update("trace_steps", datatypes.JSON(payload)).Error
}

func (s *DeployService) commitHostResult(tx *gorm.DB, host DeployTaskHost, task *DeployTask, req MarkHostResultRequest, actor string, summary deployExecutionSummary, now time.Time) error {
	applied, err := s.applyMarkedHostResult(host, req, req.ReportKey, actor, now)
	if err != nil {
		return err
	}
	if !applied {
		return errExecutionOwnership
	}
	if err := appendHostTraceTx(tx, host.TaskID, host.ID, []map[string]any{{"at": now.Format(time.RFC3339), "phase": "result", "message": fmt.Sprintf("status=%s", req.Status)}}); err != nil {
		return err
	}
	return s.writebackMarkedHostResult(host, task, req, actor, summary, now)
}

func (s *DeployService) recomputeTaskStatusTx(tx *gorm.DB, task DeployTask, actor string, result ...MarkHostResultRequest) error {
	var hosts []DeployTaskHost
	if err := tx.Where("task_id = ?", task.ID).Order("id ASC").Find(&hosts).Error; err != nil {
		return err
	}
	var active int64
	if err := tx.Model(&DeployTaskAttempt{}).Where("task_id = ? AND status = 'running' AND lease_expires_at > CURRENT_TIMESTAMP(6)", task.ID).Count(&active).Error; err != nil {
		return err
	}
	if active > 0 {
		return nil
	}
	status := terminalTaskStatus(hosts)
	if task.Status == TaskStatusPauseRequested {
		status = TaskStatusPaused
	}
	if task.Status == TaskStatusCancelRequested {
		status = TaskStatusCanceled
	}
	if status == "" || status == task.Status || (task.Status != TaskStatusRunning && task.Status != TaskStatusPauseRequested && task.Status != TaskStatusCancelRequested) {
		return nil
	}
	now, err := databaseNow(tx)
	if err != nil {
		return err
	}
	if status == TaskStatusCanceled && task.Status == TaskStatusCancelRequested {
		// Task/hosts/attempts/physical leases are already locked in order, and
		// the active-owner check above excludes live runners. Settle hosts that
		// never claimed an attempt and release only this task's unused leases.
		if err := tx.Model(&DeployTaskHost{}).Where("task_id = ? AND status IN ?", task.ID, []string{TaskHostStatusPending, TaskHostStatusRunning}).Updates(map[string]any{"status": TaskHostStatusSkipped, "finished_at": now, "updated_at": now, "updated_by": actor}).Error; err != nil {
			return err
		}
		for _, host := range hosts {
			if err := releaseHostLease(tx, host.HostID, deployHostLeaseOwner(task.ID)); err != nil {
				return err
			}
		}
	}
	updates := map[string]any{"status": status, "updated_at": now, "updated_by": actor}
	if status != TaskStatusPaused {
		updates["finished_at"] = now
	}
	if err := tx.Model(&DeployTask{}).Where("id = ?", task.ID).Updates(updates).Error; err != nil {
		return err
	}
	if isTerminalTaskStatus(status) && (task.ServiceID != 0 || task.ServiceInstanceID != 0) {
		ids := make([]uint64, 0, len(hosts))
		for _, host := range hosts {
			ids = append(ids, host.HostID)
		}
		provider, err := s.boundCMDB(tx, ids)
		if err != nil {
			return err
		}
		completed := task
		completed.Status = status
		var report MarkHostResultRequest
		if len(result) > 0 {
			report = result[0]
		}
		if err := s.targetTransactionService(tx, provider).finishServiceInstanceState(completed, report, actor, nil); err != nil {
			return err
		}
	}
	eventType := "task.completed"
	if status == TaskStatusPaused {
		eventType = "task.paused"
	}
	if status == TaskStatusCanceled && task.Status == TaskStatusCancelRequested {
		eventType = "task.canceled"
	}
	return appendTaskEventTx(tx, taskEventInput{taskID: task.ID, eventType: eventType, fromStatus: task.Status, toStatus: status, actor: actor})
}

func (s *DeployService) claimOwnedAttempt(taskID, hostID uint64, worker string) (*DeployTaskAttempt, error) {
	if worker == "" {
		return nil, errExecutionOwnership
	}
	var attempt DeployTaskAttempt
	err := s.deploymentTransaction(func(tx *gorm.DB) error {
		state, err := lockDeployment(tx, taskID, true)
		if err != nil {
			return err
		}
		if state.task.Status != TaskStatusRunning {
			return errors.New("business.deploy.attempt.active")
		}
		if err := validateRetireRemoteIntent(state.task); err != nil {
			return err
		}
		var host DeployTaskHost
		for _, h := range state.hosts {
			if h.ID == hostID {
				host = h
			}
		}
		if host.ID == 0 || isTerminalTaskHostStatus(host.Status) {
			return errExecutionOwnership
		}
		now, err := databaseNow(tx)
		if err != nil {
			return err
		}
		count, maxNo := 0, 0
		for _, a := range state.attempts {
			if a.TaskHostID != hostID {
				continue
			}
			if a.AttemptNo > maxNo {
				maxNo = a.AttemptNo
			}
			if a.Status != TaskStatusPaused {
				count++
			}
			if a.Status == TaskHostStatusRunning && a.LeaseExpiresAt != nil && a.LeaseExpiresAt.After(now) {
				return errors.New("business.deploy.attempt.active")
			}
			if a.Status == TaskHostStatusRunning {
				if err := tx.Model(&DeployTaskAttempt{}).Where("id = ?", a.ID).Updates(map[string]any{"status": "retryable", "finished_at": now, "lease_expires_at": nil, "error_message": "worker lease expired", "updated_at": now}).Error; err != nil {
					return err
				}
			}
		}
		if count >= deployAttemptMax {
			return errors.New(msgDeployAttemptExhausted)
		}
		if err := acquireHostLease(tx, host.HostID, taskID, deployHostLeaseOwner(taskID), now); err != nil {
			return err
		}
		lease := now.Add(s.ownershipTiming().attempt)
		attempt = DeployTaskAttempt{TaskID: taskID, TaskHostID: hostID, AttemptNo: maxNo + 1, Status: TaskHostStatusRunning, WorkerID: worker, LeaseExpiresAt: &lease, StartedAt: &now, CreatedAt: now, UpdatedAt: now}
		return tx.Create(&attempt).Error
	})
	if err != nil {
		return nil, err
	}
	return &attempt, nil
}

// Finish, component projection, retry disposition and physical release are one
// transaction while the original attempt is still fenced as running.
func (s *DeployService) completeOwnedAttempt(status string, runErr error, actor string, retry bool) error {
	o := s.ownership
	if o == nil {
		return errExecutionOwnership
	}
	var finishedTask DeployTask
	var finishedHost DeployTaskHost
	var settled bool
	var err error
	func() {
		o.lifecycle.Lock()
		defer o.lifecycle.Unlock()
		err = s.fencedTransaction(o.taskID, func(tx *gorm.DB, state lockedDeployment) error {
			var host DeployTaskHost
			chargeable := 0
			for _, h := range state.hosts {
				if h.ID == o.taskHostID {
					host = h
				}
			}
			for _, a := range state.attempts {
				if a.TaskHostID == o.taskHostID && a.Status != TaskStatusPaused {
					chargeable++
				}
			}
			now, err := databaseNow(tx)
			if err != nil {
				return err
			}
			o.mu.Lock()
			result, summary := o.result, o.summary
			o.mu.Unlock()
			switch state.task.Status {
			case TaskStatusPauseRequested, TaskStatusPaused:
				status = TaskStatusPaused
				retry = false
				result = nil
				if err := tx.Model(&DeployTaskHost{}).Where("id = ?", host.ID).Updates(map[string]any{"status": TaskHostStatusRunning, "finished_at": nil, "reported_at": nil}).Error; err != nil {
					return err
				}
				var steps []DeployTaskStep
				if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("task_host_id = ? AND status IN ?", host.ID, []string{StepStatusRunning, StepStatusCanceled}).Order("id ASC").Find(&steps).Error; err != nil {
					return err
				}
				for _, step := range steps {
					if err := tx.Model(&DeployTaskStep{}).Where("id = ?", step.ID).Updates(map[string]any{"status": StepStatusPending, "error_message": "", "finished_at": nil, "state_version": gorm.Expr("state_version + 1")}).Error; err != nil {
						return err
					}
					if err := tx.First(&step, step.ID).Error; err != nil {
						return err
					}
					if err := appendStepCheckpointHistory(tx, step); err != nil {
						return err
					}
				}
			case TaskStatusCancelRequested, TaskStatusCanceled:
				status = TaskStatusCanceled
				retry = false
				result = nil
				if err := tx.Model(&DeployTaskHost{}).Where("id = ? AND status <> ?", host.ID, TaskHostStatusSuccess).Updates(map[string]any{"status": TaskHostStatusSkipped, "finished_at": now}).Error; err != nil {
					return err
				}
			case TaskStatusRunning:
				if result == nil {
					req := MarkHostResultRequest{Status: TaskHostStatusSuccess}
					if runErr != nil {
						req.Status = TaskHostStatusFailed
						req.ErrorMessage = truncateDeployLog(runErr.Error(), 480)
					}
					result = &req
				}
				provider, err := s.boundCMDB(tx, []uint64{host.HostID})
				if err != nil {
					return err
				}
				inner := s.targetTransactionService(tx, provider)
				if err := inner.commitHostResult(tx, host, &state.task, *result, actor, summary, now); err != nil {
					return err
				}
				status = "success"
				if result.Status != TaskHostStatusSuccess {
					status = TaskHostStatusFailed
					if runErr == nil {
						runErr = errors.New(result.ErrorMessage)
					}
				}
				retry = retry && status == TaskHostStatusFailed && chargeable < deployAttemptMax
			default:
				return errExecutionOwnership
			}
			updates := map[string]any{"status": status, "finished_at": now, "lease_expires_at": nil, "updated_at": now}
			if runErr != nil {
				updates["error_message"] = truncateDeployLog(runErr.Error(), 480)
			}
			if err := tx.Model(&DeployTaskAttempt{}).Where("id = ? AND status = 'running' AND worker_id = ? AND lease_expires_at > CURRENT_TIMESTAMP(6)", o.attemptID, o.workerID).Updates(updates).Error; err != nil {
				return err
			}
			if retry {
				if err := tx.Model(&DeployTaskHost{}).Where("id = ?", host.ID).Updates(map[string]any{"status": TaskHostStatusRunning, "error_message": "", "finished_at": nil, "reported_at": nil, "updated_at": now}).Error; err != nil {
					return err
				}
			} else if err := releaseHostLease(tx, o.physicalHostID, o.taskLeaseOwner); err != nil {
				return err
			}
			var report MarkHostResultRequest
			if result != nil {
				report = *result
			}
			if err := s.recomputeTaskStatusTx(tx, state.task, actor, report); err != nil {
				return err
			}
			var finalTask DeployTask
			if err := tx.First(&finalTask, state.task.ID).Error; err != nil {
				return err
			}
			finishedTask = finalTask
			settled = isTerminalTaskStatus(finalTask.Status) && finalTask.Status != state.task.Status
			if result != nil {
				finishedHost = host
				finishedHost.Status = result.Status
			}
			return nil
		})
		if err == nil {
			o.mu.Lock()
			o.retired = true
			o.mu.Unlock()
		}
	}()
	if err == nil {
		if finishedHost.ID > 0 {
			recordHostResult(&finishedHost, finishedTask.ExecutorType)
		}
		if settled {
			root := s.executionRoot
			if root == nil {
				root = s
			}
			root.recordTaskFinishMetrics(finishedTask.ID, finishedTask.Status)
		}
	}
	return err
}

func (s *DeployService) reconcileOwnership() (int64, error) {
	var ids []uint64
	ctx, cancel := context.WithTimeout(s.requestContext(), s.ownershipTiming().operation)
	defer cancel()
	if err := s.db.WithContext(ctx).Model(&DeployTaskAttempt{}).Where("status = 'running'").Distinct("task_id").Order("task_id ASC").Pluck("task_id", &ids).Error; err != nil {
		return 0, err
	}
	var total int64
	for _, id := range ids {
		var changed int64
		err := s.fencedTransaction(id, func(tx *gorm.DB, state lockedDeployment) error {
			now, err := databaseNow(tx)
			if err != nil {
				return err
			}
			for _, a := range state.attempts {
				if a.Status != TaskHostStatusRunning {
					continue
				}
				var host DeployTaskHost
				for _, h := range state.hosts {
					if h.ID == a.TaskHostID {
						host = h
					}
				}
				latest := a
				for _, other := range state.attempts {
					if other.TaskHostID == a.TaskHostID && (other.AttemptNo > latest.AttemptNo || (other.AttemptNo == latest.AttemptNo && other.ID > latest.ID)) {
						latest = other
					}
				}
				newerLive := latest.ID != a.ID && latest.Status == TaskHostStatusRunning && latest.LeaseExpiresAt != nil && latest.LeaseExpiresAt.After(now)
				owned := latest.ID == a.ID && a.WorkerID != "" && a.LeaseExpiresAt != nil && a.LeaseExpiresAt.After(now)
				physical := false
				for _, l := range state.leases {
					if l.HostID == host.HostID && l.TaskID == id && l.Owner == deployHostLeaseOwner(id) && l.ExpiresAt.After(now) {
						physical = true
					}
				}
				if owned && physical {
					continue
				}
				status := "retryable"
				if state.task.Status == TaskStatusPauseRequested || state.task.Status == TaskStatusPaused {
					status = TaskStatusPaused
				}
				if state.task.Status == TaskStatusCancelRequested || state.task.Status == TaskStatusCanceled {
					status = TaskStatusCanceled
				}
				if err := tx.Model(&DeployTaskAttempt{}).Where("id = ? AND status = 'running'", a.ID).Updates(map[string]any{"status": status, "error_message": "worker lease expired", "finished_at": now, "lease_expires_at": nil, "updated_at": now}).Error; err != nil {
					return err
				}
				changed++
				count := 0
				for _, prior := range state.attempts {
					if prior.TaskHostID == host.ID && prior.Status != TaskStatusPaused {
						count++
					}
				}
				if status == "retryable" && count >= deployAttemptMax && !newerLive {
					if err := tx.Model(&DeployTaskHost{}).Where("id = ? AND status <> ?", host.ID, TaskHostStatusSuccess).Updates(map[string]any{"status": TaskHostStatusFailed, "error_message": msgDeployAttemptExhausted, "finished_at": now}).Error; err != nil {
						return err
					}
				}
				if !newerLive {
					if err := releaseHostLease(tx, host.HostID, deployHostLeaseOwner(id)); err != nil {
						return err
					}
				}
			}
			return s.recomputeTaskStatusTx(tx, state.task, "system")
		})
		if err != nil {
			return total, err
		}
		total += changed
	}
	return total, nil
}
