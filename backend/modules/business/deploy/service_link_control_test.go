package deploy

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	bizcap "github.com/duanxldragon/pantheon-base/backend/modules/business/capability"
	serviceowner "github.com/duanxldragon/pantheon-base/backend/modules/business/service"
	"gorm.io/gorm"
)

func startedServiceLink(t *testing.T) (*gorm.DB, *DeployService, *TaskResponse, *serviceowner.Manager, serviceowner.ServiceInstance) {
	t.Helper()
	db, svc, task, owner, instance := serviceLinkDraftFixture(t)
	linkDraft(t, svc, task, instance)
	started, err := svc.StartTask(task.ID, StartTaskRequest{}, "1", nil)
	if err != nil {
		t.Fatal(err)
	}
	return db, svc, started, owner, instance
}

func TestServiceLinkControlRetryRejectsDeletedOrChangedOwner(t *testing.T) {
	for _, kind := range []string{"deleted", "host", "disabled"} {
		t.Run(kind, func(t *testing.T) {
			db, svc, task, owner, instance := startedServiceLink(t)
			if _, err := svc.MarkHostResult(task.Hosts[0].ID, MarkHostResultRequest{Status: TaskHostStatusFailed, ErrorMessage: "first failed", ExecutionVersion: task.Hosts[0].ExecutionVersion}, "1", nil); err != nil {
				t.Fatal(err)
			}
			switch kind {
			case "deleted":
				if err := owner.DeleteInstance(instance.ID, nil); err != nil {
					t.Fatal(err)
				}
			case "host":
				if err := db.Model(&instance).Update("host_id", instance.HostID+100).Error; err != nil {
					t.Fatal(err)
				}
			case "disabled":
				status := "disabled"
				if _, err := owner.UpdateInstance(instance.ID, serviceowner.UpdateInstanceRequest{Status: &status}, "2", nil); err != nil {
					t.Fatal(err)
				}
			}
			before := linkedServiceFacts(t, db)
			if _, err := svc.RetryTask(task.ID, "1", nil); err == nil {
				t.Fatal("retry admitted an unavailable/mismatched Service owner")
			}
			if linkedServiceFacts(t, db) != before {
				t.Fatal("rejected retry changed persisted facts")
			}
		})
	}
}

func TestServiceLinkControlCompletionPreservesNewerOwner(t *testing.T) {
	for _, change := range []string{"update", "reconcile"} {
		t.Run(change, func(t *testing.T) {
			db, svc, task, owner, instance := startedServiceLink(t)
			if change == "update" {
				version := "operator-new-version"
				if _, err := owner.UpdateInstance(instance.ID, serviceowner.UpdateInstanceRequest{DesiredVersion: &version}, "2", nil); err != nil {
					t.Fatal(err)
				}
			} else {
				if err := db.Model(&instance).Update("last_transition_at", time.Now().Add(-time.Hour)).Error; err != nil {
					t.Fatal(err)
				}
				if _, err := owner.ReconcileInstanceState(context.Background(), instance.ID, serviceowner.ReconcileInstanceRequest{MaxAgeSeconds: 1}, "2", nil); err != nil {
					t.Fatal(err)
				}
			}
			before := linkedServiceFacts(t, db)
			_, err := svc.MarkHostResult(task.Hosts[0].ID, MarkHostResultRequest{Status: TaskHostStatusSuccess, ExecutionVersion: task.Hosts[0].ExecutionVersion}, "1", nil)
			if err == nil || err.Error() != stateStaleKey {
				t.Fatalf("late owner completion: %v", err)
			}
			if linkedServiceFacts(t, db) != before {
				t.Fatal("late completion overwrote newer owner or published CMDB/task result")
			}
		})
	}
}

func TestServiceLinkControlMalformedReferencesStillProtectInstance(t *testing.T) {
	for _, status := range []string{TaskStatusRunning, TaskStatusPauseRequested, TaskStatusPaused, TaskStatusCancelRequested} {
		t.Run(status, func(t *testing.T) {
			db, svc, task, owner, instance := serviceLinkDraftFixture(t)
			if err := db.Model(&DeployTask{}).Where("id = ?", task.ID).Updates(map[string]any{"service_id": instance.ServiceID + 100, "service_instance_id": instance.ID, "status": status}).Error; err != nil {
				t.Fatal(err)
			}
			before := linkedServiceFacts(t, db)
			if err := owner.DeleteInstance(instance.ID, nil); err == nil || err.Error() != "business.service.instance_in_use" {
				t.Fatalf("malformed executing reference: %v", err)
			}
			if linkedServiceFacts(t, db) != before {
				t.Fatal("guarded deletion changed history")
			}
			svc.SetServiceReader(nil)
		})
	}
}

func TestServiceLinkControlCancelSettlesOwner(t *testing.T) {
	db, svc, task, _, instance := startedServiceLink(t)
	if _, err := svc.CancelTask(task.ID, "1", nil); err != nil {
		t.Fatal(err)
	}
	if err := db.First(&instance, instance.ID).Error; err != nil {
		t.Fatal(err)
	}
	if instance.ObservedState != "failed" || instance.LifecycleVersion != 3 {
		t.Fatalf("cancel left owner in flight: %+v", instance)
	}
}

func TestServiceLinkControlRetryRotatesManualExecution(t *testing.T) {
	db, svc, task, _, instance := startedServiceLink(t)
	old := MarkHostResultRequest{Status: TaskHostStatusFailed, ErrorMessage: "first failed", ReportKey: "first-finish", ExecutionVersion: task.Hosts[0].ExecutionVersion}
	if old.ExecutionVersion == "" {
		t.Fatal("linked execution token missing")
	}
	if _, err := svc.MarkHostResult(task.Hosts[0].ID, old, "1", nil); err != nil {
		t.Fatal(err)
	}
	retried, err := svc.RetryTask(task.ID, "1", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(retried.Hosts) != 1 || retried.Hosts[0].ExecutionVersion == "" || retried.Hosts[0].ExecutionVersion == old.ExecutionVersion {
		t.Fatal("retry did not rotate execution binding")
	}
	if err := db.First(&instance, instance.ID).Error; err != nil {
		t.Fatal(err)
	}
	if instance.ObservedState != "installing" || instance.LifecycleVersion != 4 {
		t.Fatalf("retry did not begin the linked owner: %+v", instance)
	}
	before := linkedServiceFacts(t, db)
	for _, stale := range []MarkHostResultRequest{old, {Status: TaskHostStatusSuccess}, {Status: TaskHostStatusSuccess, ExecutionVersion: old.ExecutionVersion}} {
		if _, err := svc.MarkHostResult(task.Hosts[0].ID, stale, "1", nil); err == nil || err.Error() != errDeployTaskHostStaleReport {
			t.Fatalf("prior execution report: %v", err)
		}
		if linkedServiceFacts(t, db) != before {
			t.Fatal("stale execution report changed current execution")
		}
	}
	fresh := MarkHostResultRequest{Status: TaskHostStatusSuccess, ExecutionVersion: retried.Hosts[0].ExecutionVersion, ReportKey: "second-finish"}
	if _, err := svc.MarkHostResult(task.Hosts[0].ID, fresh, "1", nil); err != nil {
		t.Fatal(err)
	}
	settled := linkedServiceFacts(t, db)
	if _, err := svc.MarkHostResult(task.Hosts[0].ID, fresh, "1", nil); err != nil {
		t.Fatal(err)
	}
	if linkedServiceFacts(t, db) != settled {
		t.Fatal("terminal replay changed owner/task/leases")
	}
	if err := db.First(&instance, instance.ID).Error; err != nil {
		t.Fatal(err)
	}
	if instance.ObservedState != "stopped" || instance.LifecycleVersion != 5 {
		t.Fatalf("retry completion: %+v", instance)
	}
}

func TestServiceLinkControlRetireRetryRotatesWithoutOwnerMutation(t *testing.T) {
	db, svc, task, _, instance := serviceLinkDraftFixture(t)
	if err := db.Model(&instance).Updates(map[string]any{"observed_state": "stopped", "desired_state": "stopped"}).Error; err != nil {
		t.Fatal(err)
	}
	retireTask, err := svc.CreateTask(CreateTaskRequest{Name: "metadata-retire-retry", PackageID: task.PackageID, BusinessScopeID: task.BusinessScopeID, TargetType: TargetTypeHost, TargetIDs: task.TargetIDs, ServiceID: instance.ServiceID, ServiceInstanceID: instance.ID, Action: TaskActionRetire, ExecutorType: ExecutorTypeManual}, "1", nil)
	if err != nil {
		t.Fatal(err)
	}
	task = retireTask
	started, err := svc.StartTask(task.ID, StartTaskRequest{}, "1", nil)
	if err != nil {
		t.Fatal(err)
	}
	old := MarkHostResultRequest{Status: TaskHostStatusFailed, ErrorMessage: "retire failed", ExecutionVersion: started.Hosts[0].ExecutionVersion}
	if _, err := svc.MarkHostResult(started.Hosts[0].ID, old, "1", nil); err != nil {
		t.Fatal(err)
	}
	retried, err := svc.RetryTask(task.ID, "1", nil)
	if err != nil {
		t.Fatal(err)
	}
	if retried.Hosts[0].ExecutionVersion == old.ExecutionVersion {
		t.Fatal("retire retry reused prior manual execution")
	}
	if err := db.First(&instance, instance.ID).Error; err != nil {
		t.Fatal(err)
	}
	if instance.LifecycleVersion != 1 {
		t.Fatal("execution rotation mutated owner for a no-begin action")
	}
	before := linkedServiceFacts(t, db)
	if _, err := svc.MarkHostResult(started.Hosts[0].ID, old, "1", nil); err == nil {
		t.Fatal("old retire failure settled retried execution")
	}
	if linkedServiceFacts(t, db) != before {
		t.Fatal("old retire report changed facts")
	}
	fresh := MarkHostResultRequest{Status: TaskHostStatusSuccess, ExecutionVersion: retried.Hosts[0].ExecutionVersion}
	if _, err := svc.MarkHostResult(started.Hosts[0].ID, fresh, "1", nil); err != nil {
		t.Fatal(err)
	}
	settled := linkedServiceFacts(t, db)
	if _, err := svc.MarkHostResult(started.Hosts[0].ID, fresh, "1", nil); err != nil {
		t.Fatal(err)
	}
	if linkedServiceFacts(t, db) != settled {
		t.Fatal("retire terminal replay changed facts")
	}
}

func TestServiceLinkControlResumeRejectsSupersededOwner(t *testing.T) {
	db, svc, task, owner, instance := startedServiceLink(t)
	if _, err := svc.PauseTask(task.ID, "1", nil); err != nil {
		t.Fatal(err)
	}
	version := "new-desired-version"
	if _, err := owner.UpdateInstance(instance.ID, serviceowner.UpdateInstanceRequest{DesiredVersion: &version}, "2", nil); err != nil {
		t.Fatal(err)
	}
	before := linkedServiceFacts(t, db)
	if _, err := svc.ResumeTask(task.ID, "1", nil); err == nil || err.Error() != stateStaleKey {
		t.Fatalf("superseded resume: %v", err)
	}
	if linkedServiceFacts(t, db) != before {
		t.Fatal("rejected resume changed persisted facts")
	}
}

func TestServiceLinkControlSupersededCancellationRemainsLive(t *testing.T) {
	for _, mode := range []string{"manual", "unclaimed-recovery", "owned"} {
		t.Run(mode, func(t *testing.T) {
			db, svc, task, owner, instance := startedServiceLink(t)
			var worker *DeployService
			if mode != "manual" {
				if err := db.Model(&DeployTask{}).Where("id = ?", task.ID).Update("executor_type", ExecutorTypeSSH).Error; err != nil {
					t.Fatal(err)
				}
				if mode == "owned" {
					var storedTask DeployTask
					var host DeployTaskHost
					if err := db.First(&storedTask, task.ID).Error; err != nil {
						t.Fatal(err)
					}
					if err := db.First(&host, task.Hosts[0].ID).Error; err != nil {
						t.Fatal(err)
					}
					worker, _ = ownedClaim(t, svc, storedTask, host, "superseded-cancel")
				}
			}
			version := "new-operator-intent"
			if _, err := owner.UpdateInstance(instance.ID, serviceowner.UpdateInstanceRequest{DesiredVersion: &version}, "2", nil); err != nil {
				t.Fatal(err)
			}
			ownerBefore := serviceOwnerFacts(t, db)
			if _, err := svc.CancelTask(task.ID, "1", nil); err != nil {
				t.Fatal(err)
			}
			if worker != nil {
				if err := worker.completeOwnedAttempt("canceled", nil, "1", false); err != nil {
					t.Fatal(err)
				}
			}
			if _, err := svc.ReconcileDeployQueue("1"); err != nil {
				t.Fatal(err)
			}
			assertCanceledUnclaimed(t, db, task.ID)
			if serviceOwnerFacts(t, db) != ownerBefore {
				t.Fatal("cancellation overwrote superseding owner intent")
			}
			beforeReplay := linkedServiceFacts(t, db)
			if _, err := svc.ReconcileDeployQueue("1"); err != nil {
				t.Fatal(err)
			}
			if linkedServiceFacts(t, db) != beforeReplay {
				t.Fatal("canceled recovery replay changed facts")
			}
		})
	}
}

func serviceOwnerFacts(t *testing.T, db *gorm.DB) string {
	t.Helper()
	var instances []serviceowner.ServiceInstance
	if err := db.Order("id").Find(&instances).Error; err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(instances)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

type refusedDeploymentReferences struct{ mode string }

func (r *refusedDeploymentReferences) HasExecutingServiceReferences(context.Context, uint64, uint64) (bool, error) {
	return false, errors.New("reference query refused")
}

func (r *refusedDeploymentReferences) WithDeploymentReferenceTransaction(*gorm.DB) bizcap.DeploymentReferenceReader {
	switch r.mode {
	case "nil-bound":
		return nil
	case "typednil-bound":
		var nilReader *refusedDeploymentReferences
		return nilReader
	default:
		return r
	}
}

type nonbindingDeploymentReferences struct{}

func (nonbindingDeploymentReferences) HasExecutingServiceReferences(context.Context, uint64, uint64) (bool, error) {
	return false, nil
}

func TestServiceLinkControlDeletionRefusesUnavailableReferenceOwner(t *testing.T) {
	for _, mode := range []string{"missing", "nonbinding", "nil-bound", "typednil-bound", "query-failed"} {
		for _, resource := range []string{"instance", "service"} {
			t.Run(mode+"/"+resource, func(t *testing.T) {
				db, _, _, owner, instance := serviceLinkDraftFixture(t)
				if resource == "service" {
					if err := owner.DeleteInstance(instance.ID, nil); err != nil {
						t.Fatal(err)
					}
				}
				switch mode {
				case "missing":
					owner.SetDeploymentReferenceReader(nil)
				case "nonbinding":
					owner.SetDeploymentReferenceReader(nonbindingDeploymentReferences{})
				default:
					owner.SetDeploymentReferenceReader(&refusedDeploymentReferences{mode: mode})
				}
				before := linkedServiceFacts(t, db)
				var err error
				if resource == "instance" {
					err = owner.DeleteInstance(instance.ID, nil)
				} else {
					err = owner.DeleteService(instance.ServiceID, nil)
				}
				if err == nil {
					t.Fatal("deletion failed open without a trustworthy reference owner")
				}
				if linkedServiceFacts(t, db) != before {
					t.Fatal("refused deletion changed persisted facts")
				}
			})
		}
	}
}

func TestServiceLinkControlCancellationDoesNotIgnoreOwnerFailures(t *testing.T) {
	for _, mode := range []string{"missing-snapshot", "missing-reader", "owner-stale-error"} {
		t.Run(mode, func(t *testing.T) {
			db, svc, task, _, _ := startedServiceLink(t)
			switch mode {
			case "missing-snapshot":
				if err := db.Model(&DeployTask{}).Where("id = ?", task.ID).Update("target_snapshot", "{}").Error; err != nil {
					t.Fatal(err)
				}
			case "missing-reader":
				svc.SetServiceReader(nil)
			case "owner-stale-error":
				fake := &recordingServiceStateApplier{err: errors.New(stateStaleKey)}
				svc.SetServiceInstanceStateApplierWithTransaction(fake, fake.WithServiceInstanceStateTransaction)
			}
			before := linkedServiceFacts(t, db)
			if _, err := svc.CancelTask(task.ID, "1", nil); err == nil {
				t.Fatal("cancellation ignored capability/corrupt snapshot/owner refusal")
			}
			if linkedServiceFacts(t, db) != before {
				t.Fatal("refused cancel committed partial facts")
			}
		})
	}
}

func TestServiceLinkControlSupersededCancelRequiresStateFactory(t *testing.T) {
	for _, mode := range []string{"missing-factory", "nil-bound", "typednil-bound"} {
		t.Run(mode, func(t *testing.T) {
			db, svc, task, owner, instance := startedServiceLink(t)
			version := "superseding-owner-intent"
			if _, err := owner.UpdateInstance(instance.ID, serviceowner.UpdateInstanceRequest{DesiredVersion: &version}, "2", nil); err != nil {
				t.Fatal(err)
			}
			fake := &nonbindingServiceState{}
			switch mode {
			case "missing-factory":
				svc.SetServiceInstanceStateApplier(fake)
			case "nil-bound":
				svc.SetServiceInstanceStateApplierWithTransaction(fake, func(*gorm.DB) bizcap.ServiceInstanceStateApplier { return nil })
			case "typednil-bound":
				svc.SetServiceInstanceStateApplierWithTransaction(fake, func(*gorm.DB) bizcap.ServiceInstanceStateApplier {
					var nilOwner *nonbindingServiceState
					return nilOwner
				})
			}
			before := linkedServiceFacts(t, db)
			if _, err := svc.CancelTask(task.ID, "1", nil); err == nil {
				t.Fatal("superseded cancel bypassed unavailable state factory")
			}
			if linkedServiceFacts(t, db) != before || fake.calls != 0 {
				t.Fatal("unbound superseded cancellation mutated facts or called unsafe owner")
			}
		})
	}
}
