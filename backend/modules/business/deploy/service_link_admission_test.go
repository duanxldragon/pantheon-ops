package deploy

import (
	"context"
	"errors"
	"testing"
	"time"

	bizcap "github.com/duanxldragon/pantheon-base/backend/modules/business/capability"
	serviceowner "github.com/duanxldragon/pantheon-base/backend/modules/business/service"
	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func configureServiceLinkFixture(t *testing.T, db *gorm.DB, svc *DeployService, scopeID, hostID uint64) (*serviceowner.Manager, serviceowner.ServiceInstance) {
	t.Helper()
	owner := serviceowner.NewManager(db, serviceowner.Dependencies{})
	if err := owner.Migrate(); err != nil {
		t.Fatal(err)
	}
	app := serviceowner.Application{Code: "admission-app", Name: "admission-app", BusinessScopeID: scopeID, DeptID: 7, Status: "active"}
	if err := db.Create(&app).Error; err != nil {
		t.Fatal(err)
	}
	service := serviceowner.Service{ApplicationID: app.ID, Code: "admission-service", Name: "admission-service", RuntimeType: "vm", Status: "active"}
	if err := db.Create(&service).Error; err != nil {
		t.Fatal(err)
	}
	instance := serviceowner.ServiceInstance{ServiceID: service.ID, Environment: "test", TargetType: "vm", HostID: hostID, Status: "active", CurrentVersion: "old", LifecycleVersion: 1}
	if err := db.Create(&instance).Error; err != nil {
		t.Fatal(err)
	}
	svc.SetServiceReader(owner)
	svc.SetServiceInstanceStateApplierWithTransaction(owner, owner.WithServiceInstanceStateTransaction)
	owner.SetDeploymentReferenceReader(svc)
	return owner, instance
}

func serviceLinkDraftFixture(t *testing.T) (*gorm.DB, *DeployService, *TaskResponse, *serviceowner.Manager, serviceowner.ServiceInstance) {
	t.Helper()
	db, svc, task, hostID := previewFixture(t, false)
	owner, instance := configureServiceLinkFixture(t, db, svc, task.BusinessScopeID, hostID)
	return db, svc, task, owner, instance
}

func linkDraft(t *testing.T, svc *DeployService, task *TaskResponse, instance serviceowner.ServiceInstance) {
	t.Helper()
	if _, err := svc.UpdateTask(task.ID, UpdateTaskRequest{ServiceID: &instance.ServiceID, ServiceInstanceID: &instance.ID}, "1", nil); err != nil {
		t.Fatal(err)
	}
}

func TestServiceLinkAdmissionOwnerMismatchAtEveryMutation(t *testing.T) {
	for _, kind := range []string{"host", "service", "scope", "inactive-app", "inactive-service", "inactive-instance", "retired", "deleted", "kubernetes", "invisible"} {
		t.Run(kind, func(t *testing.T) {
			db, svc, task, _, instance := serviceLinkDraftFixture(t)
			requestedServiceID := instance.ServiceID
			scope := (*common.DataScopeReq)(nil)
			switch kind {
			case "host":
				otherHost := deployTestHost{Hostname: "other-visible-host", IP: "10.40.9.2", OS: "linux", Status: "assigned", BusinessScopeID: task.BusinessScopeID}
				if err := db.Create(&otherHost).Error; err != nil {
					t.Fatal(err)
				}
				instance.HostID = otherHost.ID
			case "service":
				var original serviceowner.Service
				if err := db.First(&original, instance.ServiceID).Error; err != nil {
					t.Fatal(err)
				}
				original.ID, original.Code = 0, "other-visible-service"
				if err := db.Create(&original).Error; err != nil {
					t.Fatal(err)
				}
				instance.ServiceID = original.ID
			case "scope":
				if err := db.Model(&serviceowner.Application{}).Where("code = ?", "admission-app").Update("business_scope_id", task.BusinessScopeID+100).Error; err != nil {
					t.Fatal(err)
				}
			case "inactive-app":
				if err := db.Model(&serviceowner.Application{}).Where("code = ?", "admission-app").Update("status", "inactive").Error; err != nil {
					t.Fatal(err)
				}
			case "inactive-service":
				if err := db.Model(&serviceowner.Service{}).Where("id = ?", instance.ServiceID).Update("status", "inactive").Error; err != nil {
					t.Fatal(err)
				}
			case "inactive-instance":
				instance.Status = "disabled"
			case "retired":
				instance.DesiredState = "retired"
			case "deleted":
				instance.DeletedAt = gorm.DeletedAt{Time: time.Now(), Valid: true}
			case "kubernetes":
				instance.TargetType, instance.HostID, instance.K8sClusterID, instance.Namespace = "k8s", 0, 7, "test"
			case "invisible":
				scope = &common.DataScopeReq{Mode: common.DataScopeModeDept, DeptID: 999}
			}
			if err := db.Save(&instance).Error; err != nil {
				t.Fatal(err)
			}
			before := linkedServiceFacts(t, db)
			_, err := svc.CreateTask(CreateTaskRequest{Name: "bad-link", PackageID: task.PackageID, BusinessScopeID: task.BusinessScopeID,
				TargetType: TargetTypeHost, TargetIDs: task.TargetIDs, ServiceID: requestedServiceID, ServiceInstanceID: instance.ID}, "1", scope)
			if err == nil {
				t.Fatal("invalid create accepted")
			}
			if _, err := svc.UpdateTask(task.ID, UpdateTaskRequest{ServiceID: &requestedServiceID, ServiceInstanceID: &instance.ID}, "1", scope); err == nil {
				t.Fatal("invalid update accepted")
			}
			if after := linkedServiceFacts(t, db); after != before {
				t.Fatal("rejected create/update mutated task, secrets, host rows, leases or instance")
			}
			// Historical/mutated draft associations must be revalidated on start.
			if err := db.Model(&DeployTask{}).Where("id = ?", task.ID).Updates(map[string]any{"service_id": requestedServiceID, "service_instance_id": instance.ID}).Error; err != nil {
				t.Fatal(err)
			}
			before = linkedServiceFacts(t, db)
			if _, err := svc.StartTask(task.ID, StartTaskRequest{}, "1", scope); err == nil {
				t.Fatal("invalid start accepted")
			}
			if after := linkedServiceFacts(t, db); after != before {
				t.Fatal("rejected start left committed effects")
			}
		})
	}
}

func TestServiceLinkAdmissionBeginRefusalRollsBackClaim(t *testing.T) {
	db, svc, task, _, instance := serviceLinkDraftFixture(t)
	linkDraft(t, svc, task, instance)
	refusal := errors.New("owner refuses begin")
	fake := &recordingServiceStateApplier{err: refusal}
	svc.SetServiceInstanceStateApplierWithTransaction(fake, fake.WithServiceInstanceStateTransaction)
	before := linkedServiceFacts(t, db)
	if _, err := svc.StartTask(task.ID, StartTaskRequest{}, "1", nil); !errors.Is(err, refusal) {
		t.Fatalf("begin refusal: %v", err)
	}
	if len(fake.transitions) != 1 {
		t.Fatal("begin was not attempted")
	}
	if after := linkedServiceFacts(t, db); after != before {
		t.Fatal("begin refusal did not roll back claim/lease/state")
	}
}

func TestServiceLinkAdmissionCompletionReplayAndReferenceProtection(t *testing.T) {
	for _, status := range []string{TaskHostStatusSuccess, TaskHostStatusFailed} {
		t.Run(status, func(t *testing.T) {
			db, svc, task, owner, instance := serviceLinkDraftFixture(t)
			other := instance
			other.ID, other.HostID, other.Environment = 0, instance.HostID+100, "other"
			if err := db.Create(&other).Error; err != nil {
				t.Fatal(err)
			}
			linkDraft(t, svc, task, instance)
			started, err := svc.StartTask(task.ID, StartTaskRequest{IdempotencyKey: "linked-start"}, "1", nil)
			if err != nil {
				t.Fatal(err)
			}
			beforeReplay := linkedServiceFacts(t, db)
			if _, err := svc.StartTask(task.ID, StartTaskRequest{IdempotencyKey: "linked-start"}, "1", nil); err != nil {
				t.Fatal(err)
			}
			if after := linkedServiceFacts(t, db); after != beforeReplay {
				t.Fatal("start replay changed state")
			}
			if err := owner.DeleteInstance(instance.ID, nil); err == nil {
				t.Fatal("executing instance deletion accepted")
			}
			if err := owner.DeleteService(instance.ServiceID, nil); err == nil {
				t.Fatal("service with instances deletion accepted")
			}
			report := MarkHostResultRequest{Status: status, ReportKey: "linked-finish", ErrorMessage: "fixture result", ExecutionVersion: started.Hosts[0].ExecutionVersion}
			if _, err := svc.MarkHostResult(started.Hosts[0].ID, report, "1", nil); err != nil {
				t.Fatal(err)
			}
			var current, untouched serviceowner.ServiceInstance
			if err := db.First(&current, instance.ID).Error; err != nil {
				t.Fatal(err)
			}
			if err := db.First(&untouched, other.ID).Error; err != nil {
				t.Fatal(err)
			}
			want := "stopped"
			if status == TaskHostStatusFailed {
				want = "failed"
			}
			if current.ObservedState != want || current.LifecycleVersion != instance.LifecycleVersion+2 || untouched.LifecycleVersion != other.LifecycleVersion {
				t.Fatalf("wrong instance projection: current=%+v other=%+v", current, untouched)
			}
			beforeReplay = linkedServiceFacts(t, db)
			if _, err := svc.MarkHostResult(started.Hosts[0].ID, report, "1", nil); err != nil {
				t.Fatal(err)
			}
			late := report
			late.Status, late.ReportKey = TaskHostStatusFailed, "out-of-order"
			if _, err := svc.MarkHostResult(started.Hosts[0].ID, late, "1", nil); err == nil {
				t.Fatal("out-of-order result accepted")
			}
			if after := linkedServiceFacts(t, db); after != beforeReplay {
				t.Fatal("duplicate/out-of-order completion changed history or instance")
			}
			if err := owner.DeleteInstance(instance.ID, nil); err != nil {
				t.Fatal(err)
			}
			var stored DeployTask
			if err := db.First(&stored, task.ID).Error; err != nil {
				t.Fatal(err)
			}
			if stored.ServiceInstanceID != instance.ID || len(stored.TargetSnapshot) == 0 {
				t.Fatal("terminal history lost association/snapshot")
			}
		})
	}
}

type gatedServiceLinkReader struct {
	serviceowner.Reader
	locked     chan struct{}
	release    chan struct{}
	beforeRead chan struct{}
}

func (r *gatedServiceLinkReader) GetInstance(ctx context.Context, id uint64, scope *common.DataScopeReq) (serviceowner.InstanceRef, error) {
	if r.beforeRead != nil {
		select {
		case r.beforeRead <- struct{}{}:
		default:
		}
	}
	ref, err := r.Reader.GetInstance(ctx, id, scope)
	if err != nil {
		return ref, err
	}
	select {
	case r.locked <- struct{}{}:
	default:
	}
	select {
	case <-r.release:
		return ref, nil
	case <-ctx.Done():
		return ref, ctx.Err()
	}
}

func TestServiceLinkAdmissionFinalOwnerChangeRollsBackReservations(t *testing.T) {
	db, svc, task, owner, instance := serviceLinkDraftFixture(t)
	linkDraft(t, svc, task, instance)
	otherHost := deployTestHost{Hostname: "current-owner-host", IP: "10.40.9.2", OS: "linux", Status: "assigned", BusinessScopeID: task.BusinessScopeID}
	if err := db.Create(&otherHost).Error; err != nil {
		t.Fatal(err)
	}
	ready, release, finalRead := make(chan struct{}, 1), make(chan struct{}), make(chan struct{}, 1)
	defer func() {
		select {
		case <-release:
		default:
			close(release)
		}
	}()
	blockerDone, startDone := make(chan error, 1), make(chan error, 1)
	go func() {
		blockerDone <- db.Transaction(func(tx *gorm.DB) error {
			var current serviceowner.ServiceInstance
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&current, instance.ID).Error; err != nil {
				return err
			}
			if err := tx.Model(&current).Update("host_id", otherHost.ID).Error; err != nil {
				return err
			}
			ready <- struct{}{}
			select {
			case <-release:
				return nil
			case <-time.After(10 * time.Second):
				return errors.New("owner blocker expired")
			}
		})
	}()
	awaitServiceLinkGate(t, ready)
	svc.serviceReaderTransaction = func(tx *gorm.DB) serviceowner.Reader {
		return &gatedServiceLinkReader{Reader: owner.WithReaderTransaction(tx), beforeRead: finalRead, locked: make(chan struct{}, 1), release: release}
	}
	go func() { _, err := svc.StartTask(task.ID, StartTaskRequest{}, "1", nil); startDone <- err }()
	awaitServiceLinkGate(t, finalRead) // discovery accepted old committed owner; final admission reached
	close(release)
	if err := awaitServiceLinkResult(t, blockerDone); err != nil {
		t.Fatal(err)
	}
	if err := awaitServiceLinkResult(t, startDone); err == nil {
		t.Fatal("final admission used stale owner host")
	}
	var stored DeployTask
	if err := db.First(&stored, task.ID).Error; err != nil {
		t.Fatal(err)
	}
	if stored.Status != TaskStatusDraft || stored.StartedAt != nil || len(stored.TargetSnapshot) != 0 {
		t.Fatal("rejected final admission persisted task start")
	}
	for _, model := range []any{&DeployTaskHost{}, &DeployHostLease{}, &DeployTaskEvent{}} {
		var count int64
		if err := db.Model(model).Where("task_id = ?", task.ID).Count(&count).Error; err != nil {
			t.Fatal(err)
		}
		if count != 0 {
			t.Fatalf("final rejection persisted %T", model)
		}
	}
}

func TestServiceLinkAdmissionGroupPreservesScopeAndExactHost(t *testing.T) {
	db, svc, task, _, instance := serviceLinkDraftFixture(t)
	if err := db.Model(&deployTestHost{}).Where("id = ?", instance.HostID).Update("label_values", `[{"key":"role","val":"linked"}]`).Error; err != nil {
		t.Fatal(err)
	}
	group := deployTestGroup{Name: "linked-host", Conditions: datatypes.JSON([]byte(`{"operator":"AND","rules":[{"key":"role","op":"eq","val":"linked"}]}`))}
	if err := db.Create(&group).Error; err != nil {
		t.Fatal(err)
	}
	created, err := svc.CreateTask(CreateTaskRequest{Name: "linked-group", PackageID: task.PackageID, BusinessScopeID: task.BusinessScopeID, ServiceID: instance.ServiceID, ServiceInstanceID: instance.ID, TargetType: TargetTypeGroup, TargetIDs: []uint64{group.ID}}, "1", nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.UpdateTask(created.ID, UpdateTaskRequest{Name: stringPtr("updated-group")}, "1", nil); err != nil {
		t.Fatal(err)
	}
	started, err := svc.StartTask(created.ID, StartTaskRequest{}, "1", nil)
	if err != nil {
		t.Fatal(err)
	}
	if started.BusinessScopeID != task.BusinessScopeID || len(started.Hosts) != 1 || started.Hosts[0].HostID != instance.HostID {
		t.Fatalf("group admission mismatch: %+v", started)
	}
}

func TestServiceLinkAdmissionIndependentNeverCallsState(t *testing.T) {
	_, svc, task, _, _ := serviceLinkDraftFixture(t)
	fake := &recordingServiceStateApplier{err: errors.New("must not call")}
	svc.SetServiceInstanceStateApplierWithTransaction(fake, fake.WithServiceInstanceStateTransaction)
	started, err := svc.StartTask(task.ID, StartTaskRequest{}, "1", nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.MarkHostResult(started.Hosts[0].ID, MarkHostResultRequest{Status: TaskHostStatusSuccess}, "1", nil); err != nil {
		t.Fatal(err)
	}
	if len(fake.transitions) != 0 {
		t.Fatal("independent task called Service state")
	}
}

type gatedDeploymentReferences struct {
	bizcap.DeploymentReferenceReader
	checked chan struct{}
	release chan struct{}
}

func (r *gatedDeploymentReferences) HasExecutingServiceReferences(ctx context.Context, serviceID, instanceID uint64) (bool, error) {
	active, err := r.DeploymentReferenceReader.HasExecutingServiceReferences(ctx, serviceID, instanceID)
	if err != nil {
		return active, err
	}
	select {
	case r.checked <- struct{}{}:
	default:
	}
	select {
	case <-r.release:
		return active, nil
	case <-ctx.Done():
		return false, ctx.Err()
	}
}

type gatedDeploymentOwner struct {
	*DeployService
	checked chan struct{}
	release chan struct{}
}

func (r *gatedDeploymentOwner) WithDeploymentReferenceTransaction(tx *gorm.DB) bizcap.DeploymentReferenceReader {
	return &gatedDeploymentReferences{DeploymentReferenceReader: r.DeployService.WithDeploymentReferenceTransaction(tx), checked: r.checked, release: r.release}
}

func awaitServiceLinkGate(t *testing.T, ch <-chan struct{}) {
	t.Helper()
	select {
	case <-ch:
	case <-time.After(10 * time.Second):
		t.Fatal("owner gate not reached")
	}
}

func awaitServiceLinkResult(t *testing.T, ch <-chan error) error {
	t.Helper()
	select {
	case err := <-ch:
		return err
	case <-time.After(10 * time.Second):
		t.Fatal("operation did not settle")
		return nil
	}
}

func TestServiceLinkAdmissionConcurrentDelete(t *testing.T) {
	for _, winner := range []string{"start", "delete"} {
		t.Run(winner, func(t *testing.T) {
			db, svc, task, owner, instance := serviceLinkDraftFixture(t)
			linkDraft(t, svc, task, instance)
			gate, release := make(chan struct{}, 1), make(chan struct{})
			defer func() {
				select {
				case <-release:
				default:
					close(release)
				}
			}()
			startDone, deleteDone := make(chan error, 1), make(chan error, 1)
			start := func() { _, err := svc.StartTask(task.ID, StartTaskRequest{}, "1", nil); startDone <- err }
			remove := func() { deleteDone <- owner.DeleteInstance(instance.ID, nil) }
			if winner == "start" {
				svc.serviceReaderTransaction = func(tx *gorm.DB) serviceowner.Reader {
					return &gatedServiceLinkReader{Reader: owner.WithReaderTransaction(tx), locked: gate, release: release}
				}
				go start()
				awaitServiceLinkGate(t, gate)
				go remove()
				close(release)
				if err := awaitServiceLinkResult(t, startDone); err != nil {
					t.Fatal(err)
				}
				if err := awaitServiceLinkResult(t, deleteDone); err == nil {
					t.Fatal("delete accepted after admitted start")
				}
			} else {
				owner.SetDeploymentReferenceReader(&gatedDeploymentOwner{DeployService: svc, checked: gate, release: release})
				go remove()
				awaitServiceLinkGate(t, gate)
				go start()
				close(release)
				if err := awaitServiceLinkResult(t, deleteDone); err != nil {
					t.Fatal(err)
				}
				if err := awaitServiceLinkResult(t, startDone); err == nil {
					t.Fatal("start admitted deleted instance")
				}
				var current DeployTask
				if err := db.First(&current, task.ID).Error; err != nil {
					t.Fatal(err)
				}
				for _, model := range []any{&DeployTaskHost{}, &DeployHostLease{}, &DeployTaskEvent{}} {
					var count int64
					if err := db.Model(model).Where("task_id = ?", task.ID).Count(&count).Error; err != nil {
						t.Fatal(err)
					}
					if count != 0 {
						t.Fatalf("rejected start persisted %T rows", model)
					}
				}
				if current.Status != TaskStatusDraft {
					t.Fatalf("deleted owner left task %s", current.Status)
				}
			}
		})
	}
}

func TestServiceLinkAdmissionRejectsInvalidCreateBeforeMutation(t *testing.T) {
	cases := []struct {
		name       string
		serviceID  uint64
		instanceID uint64
	}{
		{name: "service without instance", serviceID: 901},
		{name: "instance without service", instanceID: 902},
		{name: "unconfigured owner cannot validate association", serviceID: 901, instanceID: 902},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			db, svc, task, hostID := previewFixture(t, false)
			var before int64
			if err := db.Model(&DeployTask{}).Count(&before).Error; err != nil {
				t.Fatal(err)
			}
			created, err := svc.CreateTask(CreateTaskRequest{
				Name: "invalid-associated-task", PackageID: task.PackageID,
				BusinessScopeID: task.BusinessScopeID, TargetType: TargetTypeHost,
				TargetIDs: []uint64{hostID}, ExecutorType: ExecutorTypeManual,
				ServiceID: tc.serviceID, ServiceInstanceID: tc.instanceID,
			}, "1", nil)
			if err == nil {
				t.Errorf("invalid association accepted: task=%v", created)
			}
			var after int64
			if err := db.Model(&DeployTask{}).Count(&after).Error; err != nil {
				t.Fatal(err)
			}
			if after != before {
				t.Errorf("rejected create must not persist a task: before=%d after=%d", before, after)
			}
		})
	}
}

func TestServiceLinkAdmissionRejectsInvalidUpdateBeforeMutation(t *testing.T) {
	cases := []struct {
		name       string
		serviceID  uint64
		instanceID uint64
	}{
		{name: "service without instance", serviceID: 901},
		{name: "instance without service", instanceID: 902},
		{name: "unconfigured owner cannot validate association", serviceID: 901, instanceID: 902},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			db, svc, task, _ := previewFixture(t, false)
			_, err := svc.UpdateTask(task.ID, UpdateTaskRequest{ServiceID: &tc.serviceID, ServiceInstanceID: &tc.instanceID}, "1", nil)
			if err == nil {
				t.Error("invalid association update accepted")
			}
			var stored DeployTask
			if err := db.First(&stored, task.ID).Error; err != nil {
				t.Fatal(err)
			}
			if stored.ServiceID != 0 || stored.ServiceInstanceID != 0 {
				t.Errorf("rejected update changed association: service=%d instance=%d", stored.ServiceID, stored.ServiceInstanceID)
			}
		})
	}
}
