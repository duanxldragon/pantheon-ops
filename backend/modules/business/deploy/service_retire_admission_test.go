package deploy

import (
	"context"
	"database/sql"
	"testing"

	bizcap "github.com/duanxldragon/pantheon-base/backend/modules/business/capability"
	serviceowner "github.com/duanxldragon/pantheon-base/backend/modules/business/service"
	"gorm.io/gorm"
)

func TestServiceRetireAdmissionCallerRepeatableReadSnapshot(t *testing.T) {
	db, s, req, instance := retireCreateFixture(t)
	owner := retireAdmissionOwner(t, s)
	task, err := s.CreateTask(req, "1", nil)
	if err != nil {
		t.Fatal(err)
	}
	var beforeOwner serviceowner.ServiceInstance
	if err := db.First(&beforeOwner, instance.ID).Error; err != nil {
		t.Fatal(err)
	}
	if beforeOwner.ObservedState != serviceowner.ObservedStateStopped {
		t.Fatal("expected stopped Service owner before competing Start")
	}

	// A nonlocking read establishes the caller's consistent snapshot before a
	// legitimate Start commits on another connection. Service owner row locks
	// acquired later must not let admission trust this stale Deploy snapshot.
	tx := db.Begin(&sql.TxOptions{Isolation: sql.LevelRepeatableRead})
	if tx.Error != nil {
		t.Fatal(tx.Error)
	}
	defer tx.Rollback()
	var snapshotCount int64
	if err := tx.Model(&DeployTask{}).Where("id = ? AND status = ?", task.ID, TaskStatusRunning).Count(&snapshotCount).Error; err != nil {
		t.Fatal(err)
	}
	if snapshotCount != 0 {
		t.Fatal("new caller snapshot unexpectedly contains an executing task")
	}
	started, err := s.StartTask(task.ID, StartTaskRequest{IdempotencyKey: "retire-caller-snapshot-start"}, "1", nil)
	if err != nil {
		t.Fatalf("separate legitimate Start failed: %v", err)
	}
	if started.Status != TaskStatusRunning || len(started.Hosts) != 1 {
		t.Fatal("separate Start did not commit an executing linked task")
	}
	var committedCount int64
	if err := db.Model(&DeployTask{}).Where("id = ? AND status = ?", task.ID, TaskStatusRunning).Count(&committedCount).Error; err != nil {
		t.Fatal(err)
	}
	if committedCount != 1 {
		t.Fatal("separate Start is not visible outside the caller transaction")
	}
	if err := tx.Model(&DeployTask{}).Where("id = ? AND status = ?", task.ID, TaskStatusRunning).Count(&snapshotCount).Error; err != nil {
		t.Fatal(err)
	}
	if snapshotCount != 0 {
		t.Fatal("caller did not retain its established repeatable-read snapshot")
	}
	beforeRetire := linkedServiceFacts(t, db)
	err = owner.WithServiceInstanceStateTransaction(tx).ApplyServiceInstanceState(context.Background(), bizcap.ServiceInstanceStateTransition{
		InstanceID: instance.ID, Action: serviceowner.TransitionActionRetire, ObservedState: serviceowner.ObservedStateRetired, CorrelationID: "caller-snapshot-retire",
	}, "2", nil)
	if err == nil || err.Error() != "service.state.transactionIsolationUnknown" {
		t.Errorf("unverified caller snapshot did not fail closed: %v", err)
	}
	var callerOwner serviceowner.ServiceInstance
	if err := tx.First(&callerOwner, instance.ID).Error; err != nil {
		t.Fatal(err)
	}
	if callerOwner.ObservedState != beforeOwner.ObservedState || callerOwner.DesiredState != beforeOwner.DesiredState || callerOwner.LifecycleVersion != beforeOwner.LifecycleVersion {
		t.Errorf("rejected Retire changed owner inside caller transaction: observed=%s desired=%s lifecycleVersion=%d", callerOwner.ObservedState, callerOwner.DesiredState, callerOwner.LifecycleVersion)
	}
	if err := tx.Rollback().Error; err != nil {
		t.Fatal(err)
	}
	assertRetireFactsUnchanged(t, db, beforeRetire)
	var stored DeployTask
	if err := db.First(&stored, task.ID).Error; err != nil {
		t.Fatal(err)
	}
	if stored.Status != TaskStatusRunning {
		t.Fatal("caller rollback lost the separately committed Start")
	}
}

func retireAdmissionOwner(t *testing.T, s *DeployService) *serviceowner.Manager {
	t.Helper()
	owner, ok := s.serviceReader.(*serviceowner.Manager)
	if !ok {
		t.Fatal("expected real Service owner fixture")
	}
	return owner
}

func TestServiceRetireAdmissionExecutingStatuses(t *testing.T) {
	for _, status := range []string{TaskStatusRunning, TaskStatusPauseRequested, TaskStatusPaused, TaskStatusCancelRequested,
		TaskStatusDraft, TaskStatusPending, TaskStatusSuccess, TaskStatusFailed, TaskStatusCanceled} {
		t.Run(status, func(t *testing.T) {
			db, s, req, instance := retireCreateFixture(t)
			owner := retireAdmissionOwner(t, s)
			other := DeployTask{Name: "retire-referenced-" + status, PackageID: req.PackageID, ServiceID: instance.ServiceID, ServiceInstanceID: instance.ID, Status: status, ExecutorType: ExecutorTypeManual}
			if err := db.Create(&other).Error; err != nil {
				t.Fatal(err)
			}
			before := linkedServiceFacts(t, db)
			_, err := owner.TransitionState(context.Background(), serviceowner.InstanceStateTransitionRequest{
				InstanceID: instance.ID, Action: serviceowner.TransitionActionRetire, ObservedState: serviceowner.ObservedStateRetired, CorrelationID: "reference-retire",
			}, "2", nil)
			executing := status == TaskStatusRunning || status == TaskStatusPauseRequested || status == TaskStatusPaused || status == TaskStatusCancelRequested
			if executing {
				if err == nil || err.Error() != "business.service.instance_in_use" {
					t.Fatalf("executing %s did not refuse Retire: %v", status, err)
				}
				assertRetireFactsUnchanged(t, db, before)
			} else {
				if err != nil {
					t.Fatalf("nonexecuting %s incorrectly blocked Retire: %v", status, err)
				}
				var current serviceowner.ServiceInstance
				if err := db.First(&current, instance.ID).Error; err != nil {
					t.Fatal(err)
				}
				if current.ObservedState != serviceowner.ObservedStateRetired {
					t.Fatal("legal Retire did not finish")
				}
			}
		})
	}
}

func TestServiceRetireAdmissionCompletionSelfAndForeign(t *testing.T) {
	for _, otherStatus := range []string{"", TaskStatusRunning, TaskStatusPaused} {
		name := otherStatus
		if name == "" {
			name = "terminal-self"
		}
		t.Run(name, func(t *testing.T) {
			db, s, req, instance := retireCreateFixture(t)
			task, err := s.CreateTask(req, "1", nil)
			if err != nil {
				t.Fatal(err)
			}
			started, err := s.StartTask(task.ID, StartTaskRequest{IdempotencyKey: "retire-self-start"}, "1", nil)
			if err != nil {
				t.Fatal(err)
			}
			if otherStatus != "" {
				other := DeployTask{Name: "foreign-retire-reference", PackageID: req.PackageID, ServiceID: instance.ServiceID, ServiceInstanceID: instance.ID, Status: otherStatus, ExecutorType: ExecutorTypeManual}
				if err := db.Create(&other).Error; err != nil {
					t.Fatal(err)
				}
			}
			before := linkedServiceFacts(t, db)
			report := MarkHostResultRequest{Status: TaskHostStatusSuccess, ReportKey: "retire-self-finish", ExecutionVersion: started.Hosts[0].ExecutionVersion}
			_, err = s.MarkHostResult(started.Hosts[0].ID, report, "1", nil)
			if otherStatus != "" {
				if err == nil || err.Error() != "business.service.instance_in_use" {
					t.Fatalf("foreign reference did not roll back completion: %v", err)
				}
				assertRetireFactsUnchanged(t, db, before)
				return
			}
			if err != nil {
				t.Fatalf("own terminal reference incorrectly blocked completion: %v", err)
			}
			var current serviceowner.ServiceInstance
			var stored DeployTask
			if err := db.First(&current, instance.ID).Error; err != nil {
				t.Fatal(err)
			}
			if err := db.First(&stored, task.ID).Error; err != nil {
				t.Fatal(err)
			}
			if current.ObservedState != serviceowner.ObservedStateRetired || stored.Status != TaskStatusSuccess {
				t.Fatal("self completion did not commit task and owner atomically")
			}
		})
	}
}

func TestServiceRetireAdmissionConcurrentStart(t *testing.T) {
	for _, winner := range []string{"start", "retire"} {
		t.Run(winner, func(t *testing.T) {
			db, s, req, instance := retireCreateFixture(t)
			owner := retireAdmissionOwner(t, s)
			task, err := s.CreateTask(req, "1", nil)
			if err != nil {
				t.Fatal(err)
			}
			gate, release := make(chan struct{}, 1), make(chan struct{})
			defer func() {
				select {
				case <-release:
				default:
					close(release)
				}
			}()
			startDone, retireDone := make(chan error, 1), make(chan error, 1)
			start := func() {
				_, err := s.StartTask(task.ID, StartTaskRequest{IdempotencyKey: "retire-competing-start"}, "1", nil)
				startDone <- err
			}
			retire := func() {
				_, err := owner.TransitionState(context.Background(), serviceowner.InstanceStateTransitionRequest{
					InstanceID: instance.ID, Action: serviceowner.TransitionActionRetire, ObservedState: serviceowner.ObservedStateRetired, CorrelationID: "concurrent-retire",
				}, "2", nil)
				retireDone <- err
			}
			if winner == "start" {
				s.serviceReaderTransaction = func(tx *gorm.DB) serviceowner.Reader {
					return &gatedServiceLinkReader{Reader: owner.WithReaderTransaction(tx), locked: gate, release: release}
				}
				go start()
				awaitServiceLinkGate(t, gate)
				go retire()
				close(release)
				if err := awaitServiceLinkResult(t, startDone); err != nil {
					t.Fatal(err)
				}
				if err := awaitServiceLinkResult(t, retireDone); err == nil || err.Error() != "business.service.instance_in_use" {
					t.Fatalf("Retire ignored admitted metadata Start: %v", err)
				}
				var current serviceowner.ServiceInstance
				if err := db.First(&current, instance.ID).Error; err != nil {
					t.Fatal(err)
				}
				if current.ObservedState != serviceowner.ObservedStateStopped {
					t.Fatal("Start-first metadata owner became retired")
				}
			} else {
				owner.SetDeploymentReferenceReader(&gatedDeploymentOwner{DeployService: s, checked: gate, release: release})
				go retire()
				select {
				case <-gate:
				case err := <-retireDone:
					t.Fatalf("Retire committed before reference admission gate: %v", err)
				}
				go start()
				close(release)
				if err := awaitServiceLinkResult(t, retireDone); err != nil {
					t.Fatal(err)
				}
				if err := awaitServiceLinkResult(t, startDone); err == nil {
					t.Fatal("Start admitted retired owner")
				}
				var stored DeployTask
				if err := db.First(&stored, task.ID).Error; err != nil {
					t.Fatal(err)
				}
				if stored.Status != TaskStatusDraft || stored.StartedAt != nil || len(stored.TargetSnapshot) != 0 {
					t.Fatal("Retire-first rejection left committed task claim")
				}
				for _, model := range []any{&DeployTaskHost{}, &DeployHostLease{}, &DeployTaskAttempt{}, &DeployTaskEvent{}} {
					var count int64
					if err := db.Model(model).Where("task_id = ?", task.ID).Count(&count).Error; err != nil {
						t.Fatal(err)
					}
					if count != 0 {
						t.Fatalf("retired-owner rejection persisted %T", model)
					}
				}
			}
		})
	}
}
