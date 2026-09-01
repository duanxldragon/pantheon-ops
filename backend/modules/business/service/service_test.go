package service

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	bizcap "pantheon-base/modules/business/capability"
	"pantheon-base/pkg/common"
	"pantheon-base/pkg/testmysql"
)

func TestInstanceRefUsesPublicJSONContract(t *testing.T) {
	payload, err := json.Marshal(InstanceRef{ID: 42, ServiceID: 7, LifecycleVersion: 3})
	if err != nil {
		t.Fatalf("marshal instance ref: %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("unmarshal instance ref: %v", err)
	}
	if decoded["id"] != float64(42) || decoded["serviceId"] != float64(7) || decoded["lifecycleVersion"] != float64(3) {
		t.Fatalf("unexpected public JSON contract: %s", payload)
	}
	if _, exists := decoded["ID"]; exists {
		t.Fatalf("internal Go field name leaked into JSON: %s", payload)
	}
}

type fakeBizScopeReader struct {
	ref bizcap.BizScopeRef
}

func TestServiceCatalogDeleteLifecycle(t *testing.T) {
	manager := setupServiceTestDB(t)
	scope := &common.DataScopeReq{Mode: common.DataScopeModeDept, DeptID: 7}
	application, err := manager.CreateApplication(
		CreateApplicationRequest{Code: "app-delete", Name: "Delete App", BusinessScopeID: 10},
		"1",
		scope,
	)
	if err != nil {
		t.Fatalf("create application: %v", err)
	}
	service, err := manager.CreateService(
		CreateServiceRequest{ApplicationID: application.ID, Code: "api-delete", Name: "Delete API", RuntimeType: "vm"},
		"1",
		scope,
	)
	if err != nil {
		t.Fatalf("create service: %v", err)
	}
	instance, err := manager.CreateInstance(
		context.Background(),
		CreateInstanceRequest{ServiceID: service.ID, Environment: "prod", TargetType: TargetTypeVM, HostID: 55},
		"1",
		scope,
	)
	if err != nil {
		t.Fatalf("create instance: %v", err)
	}

	if err := manager.DeleteInstance(instance.ID, scope); err != nil {
		t.Fatalf("delete instance: %v", err)
	}
	if err := manager.DeleteService(service.ID, scope); err != nil {
		t.Fatalf("delete service after instance: %v", err)
	}
	if err := manager.DeleteApplication(application.ID, scope); err != nil {
		t.Fatalf("delete application after service: %v", err)
	}
}

func TestServiceInstanceReconciliationMarksStuckTransitionFailed(t *testing.T) {
	manager := setupServiceTestDB(t)
	scope := &common.DataScopeReq{Mode: common.DataScopeModeDept, DeptID: 7}
	app, err := manager.CreateApplication(CreateApplicationRequest{Code: "app-reconcile", Name: "Reconcile App", BusinessScopeID: 10}, "1", scope)
	if err != nil {
		t.Fatalf("create app: %v", err)
	}
	svc, err := manager.CreateService(CreateServiceRequest{ApplicationID: app.ID, Code: "api", Name: "API", RuntimeType: "vm"}, "1", scope)
	if err != nil {
		t.Fatalf("create service: %v", err)
	}
	instance, err := manager.CreateInstance(context.Background(), CreateInstanceRequest{ServiceID: svc.ID, Environment: "prod", TargetType: TargetTypeVM, HostID: 55}, "1", scope)
	if err != nil {
		t.Fatalf("create instance: %v", err)
	}
	updated, err := manager.UpdateInstance(instance.ID, UpdateInstanceRequest{CurrentVersion: stringPtr("1.0")}, "1", scope)
	if err != nil {
		t.Fatalf("seed version: %v", err)
	}
	installing, err := manager.TransitionState(context.Background(), InstanceStateTransitionRequest{
		InstanceID: instance.ID, Action: TransitionActionInstall, ExpectedLifecycleVersion: updated.LifecycleVersion,
		ObservedState: ObservedStateInstalling, CorrelationID: "reconcile-begin",
	}, "1", scope)
	if err != nil {
		t.Fatalf("begin install: %v", err)
	}
	old := time.Now().Add(-time.Hour)
	if err := manager.db.Model(&ServiceInstance{}).Where("id = ?", instance.ID).Updates(map[string]any{
		"last_transition_at": old,
	}).Error; err != nil {
		t.Fatalf("age transition: %v", err)
	}
	reconciled, err := manager.ReconcileInstanceState(context.Background(), instance.ID, ReconcileInstanceRequest{MaxAgeSeconds: 60}, "system", scope)
	if err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	if reconciled.ObservedState != ObservedStateFailed || reconciled.HealthState != HealthStateDegraded {
		t.Fatalf("unexpected reconciled state: %+v", reconciled)
	}
	if reconciled.LifecycleVersion <= installing.LifecycleVersion {
		t.Fatalf("expected reconciliation CAS increment, got %+v", reconciled)
	}
}

func TestServiceInstanceStateTransitionsUseCASAndRejectIllegalMoves(t *testing.T) {
	manager := setupServiceTestDB(t)
	scope := &common.DataScopeReq{Mode: common.DataScopeModeDept, DeptID: 7}
	app, err := manager.CreateApplication(CreateApplicationRequest{Code: "app-state", Name: "State App", BusinessScopeID: 10}, "1", scope)
	if err != nil {
		t.Fatalf("create app: %v", err)
	}
	svc, err := manager.CreateService(CreateServiceRequest{ApplicationID: app.ID, Code: "api", Name: "API", RuntimeType: "vm"}, "1", scope)
	if err != nil {
		t.Fatalf("create service: %v", err)
	}
	instance, err := manager.CreateInstance(context.Background(), CreateInstanceRequest{
		ServiceID: svc.ID, Environment: "prod", TargetType: TargetTypeVM, HostID: 55,
	}, "1", scope)
	if err != nil {
		t.Fatalf("create instance: %v", err)
	}
	if instance.DesiredState != DesiredStateStopped || instance.ObservedState != ObservedStateUnknown || instance.HealthState != HealthStateUnknown {
		t.Fatalf("unexpected initial state: %+v", instance)
	}

	_, err = manager.TransitionState(context.Background(), InstanceStateTransitionRequest{
		InstanceID: instance.ID, Action: TransitionActionStart, ObservedState: ObservedStateStarting,
	}, "1", scope)
	if err == nil || err.Error() != errStateInvalid {
		t.Fatalf("expected illegal start rejection, got %v", err)
	}

	updated, err := manager.UpdateInstance(instance.ID, UpdateInstanceRequest{CurrentVersion: stringPtr("1.0")}, "1", scope)
	if err != nil {
		t.Fatalf("set installed version: %v", err)
	}
	installing, err := manager.TransitionState(context.Background(), InstanceStateTransitionRequest{
		InstanceID: instance.ID, Action: TransitionActionInstall,
		ExpectedLifecycleVersion: updated.LifecycleVersion, ObservedState: ObservedStateInstalling,
		CorrelationID: "install-1",
	}, "1", scope)
	if err != nil {
		t.Fatalf("install transition: %v", err)
	}
	stopped, err := manager.TransitionState(context.Background(), InstanceStateTransitionRequest{
		InstanceID: instance.ID, Action: TransitionActionInstall,
		ExpectedLifecycleVersion: installing.LifecycleVersion, ObservedState: ObservedStateStopped,
		CorrelationID: "install-1-done",
	}, "1", scope)
	if err != nil {
		t.Fatalf("install completion: %v", err)
	}
	if stopped.ObservedState != ObservedStateStopped {
		t.Fatalf("expected stopped state, got %+v", stopped)
	}

	_, err = manager.TransitionState(context.Background(), InstanceStateTransitionRequest{
		InstanceID: instance.ID, Action: TransitionActionStart, ExpectedLifecycleVersion: installing.LifecycleVersion,
		ObservedState: ObservedStateStarting,
	}, "1", scope)
	if err == nil || err.Error() != errStateStale {
		t.Fatalf("expected stale transition rejection, got %v", err)
	}
	starting, err := manager.TransitionState(context.Background(), InstanceStateTransitionRequest{
		InstanceID: instance.ID, Action: TransitionActionStart,
		ExpectedLifecycleVersion: stopped.LifecycleVersion, ObservedState: ObservedStateStarting,
		CorrelationID: "start-1",
	}, "1", scope)
	if err != nil {
		t.Fatalf("start transition: %v", err)
	}
	running, err := manager.TransitionState(context.Background(), InstanceStateTransitionRequest{
		InstanceID: instance.ID, Action: TransitionActionStart,
		ExpectedLifecycleVersion: starting.LifecycleVersion, ObservedState: ObservedStateRunning,
		HealthState: HealthStateUnknown, CorrelationID: "start-1-done",
	}, "1", scope)
	if err != nil {
		t.Fatalf("start completion: %v", err)
	}
	if running.DesiredState != DesiredStateRunning || running.ObservedState != ObservedStateRunning {
		t.Fatalf("unexpected running state: %+v", running)
	}
}

func TestServiceInstanceUpgradeRequiresHealthyResult(t *testing.T) {
	manager := setupServiceTestDB(t)
	scope := &common.DataScopeReq{Mode: common.DataScopeModeDept, DeptID: 7}
	app, err := manager.CreateApplication(CreateApplicationRequest{Code: "app-upgrade", Name: "Upgrade App", BusinessScopeID: 10}, "1", scope)
	if err != nil {
		t.Fatalf("create app: %v", err)
	}
	svc, err := manager.CreateService(CreateServiceRequest{ApplicationID: app.ID, Code: "api", Name: "API", RuntimeType: "vm"}, "1", scope)
	if err != nil {
		t.Fatalf("create service: %v", err)
	}
	instance, err := manager.CreateInstance(context.Background(), CreateInstanceRequest{ServiceID: svc.ID, Environment: "prod", TargetType: TargetTypeVM, HostID: 55}, "1", scope)
	if err != nil {
		t.Fatalf("create instance: %v", err)
	}
	installed, err := manager.UpdateInstance(instance.ID, UpdateInstanceRequest{CurrentVersion: stringPtr("1.0"), DesiredVersion: stringPtr("1.0")}, "1", scope)
	if err != nil {
		t.Fatalf("seed installed version: %v", err)
	}
	installing, err := manager.TransitionState(context.Background(), InstanceStateTransitionRequest{
		InstanceID: instance.ID, Action: TransitionActionInstall, ExpectedLifecycleVersion: installed.LifecycleVersion,
		ObservedState: ObservedStateInstalling, CorrelationID: "upgrade-install-begin",
	}, "1", scope)
	if err != nil {
		t.Fatalf("install begin: %v", err)
	}
	stopped, err := manager.TransitionState(context.Background(), InstanceStateTransitionRequest{
		InstanceID: instance.ID, Action: TransitionActionInstall, ExpectedLifecycleVersion: installing.LifecycleVersion,
		ObservedState: ObservedStateStopped, CorrelationID: "upgrade-install-done",
	}, "1", scope)
	if err != nil {
		t.Fatalf("install done: %v", err)
	}
	upgrading, err := manager.TransitionState(context.Background(), InstanceStateTransitionRequest{
		InstanceID: instance.ID, Action: TransitionActionUpgrade, ExpectedLifecycleVersion: stopped.LifecycleVersion,
		DesiredVersion: "2.0", ObservedState: ObservedStateUpgrading, CorrelationID: "upgrade-begin",
	}, "1", scope)
	if err != nil {
		t.Fatalf("upgrade begin: %v", err)
	}
	if _, err := manager.TransitionState(context.Background(), InstanceStateTransitionRequest{
		InstanceID: instance.ID, Action: TransitionActionUpgrade, ExpectedLifecycleVersion: upgrading.LifecycleVersion,
		DesiredVersion: "2.0", CurrentVersion: "2.0", ObservedState: ObservedStateRunning,
		HealthState: HealthStateUnknown, CorrelationID: "upgrade-unknown-health",
	}, "1", scope); err == nil || err.Error() != errStateInvalid {
		t.Fatalf("expected unhealthy upgrade completion rejection, got %v", err)
	}
	healthy, err := manager.TransitionState(context.Background(), InstanceStateTransitionRequest{
		InstanceID: instance.ID, Action: TransitionActionUpgrade, ExpectedLifecycleVersion: upgrading.LifecycleVersion,
		DesiredVersion: "2.0", CurrentVersion: "2.0", ObservedState: ObservedStateRunning,
		HealthState: HealthStateHealthy, HealthRevision: "check-2", CorrelationID: "upgrade-healthy",
	}, "1", scope)
	if err != nil {
		t.Fatalf("healthy upgrade completion: %v", err)
	}
	if healthy.CurrentVersion != "2.0" || healthy.HealthState != HealthStateHealthy {
		t.Fatalf("unexpected healthy upgrade result: %+v", healthy)
	}
}

func stringPtr(value string) *string {
	return &value
}

func (f fakeBizScopeReader) GetActive(context.Context, uint64, *common.DataScopeReq) (bizcap.BizScopeRef, error) {
	return f.ref, nil
}

func (f fakeBizScopeReader) ResolveActiveByCodes(context.Context, []string, *common.DataScopeReq) (map[string]bizcap.BizScopeRef, error) {
	return map[string]bizcap.BizScopeRef{f.ref.Code: f.ref}, nil
}

type fakeHostReader struct {
	ref bizcap.HostRef
}

func (f fakeHostReader) GetByIDs(context.Context, bizcap.HostIDsQuery) (bizcap.HostPage, error) {
	return bizcap.HostPage{Items: []bizcap.HostRef{f.ref}, Total: 1}, nil
}

func (f fakeHostReader) ListByBusinessScope(context.Context, bizcap.HostScopeQuery) (bizcap.HostPage, error) {
	return bizcap.HostPage{}, nil
}

func (f fakeHostReader) ListAvailable(context.Context, bizcap.AvailableHostQuery) (bizcap.HostPage, error) {
	return bizcap.HostPage{}, nil
}

func (f fakeHostReader) HasBusinessScopeReferences(context.Context, uint64) (bool, error) {
	return false, nil
}

type fakeK8sReader struct {
	ref bizcap.K8sTargetRef
	err error
}

func (f fakeK8sReader) ResolveTarget(context.Context, bizcap.K8sTargetQuery) (bizcap.K8sTargetRef, error) {
	return f.ref, f.err
}

func setupServiceTestDB(t *testing.T) *Manager {
	t.Helper()
	db := testmysql.Open(t)
	manager := NewManager(db, Dependencies{
		BizScopeReader: fakeBizScopeReader{ref: bizcap.BizScopeRef{ID: 10, Code: "scope-a", Name: "Scope A", Status: StatusActive, DeptID: 7}},
		CMDBReader:     fakeHostReader{ref: bizcap.HostRef{ID: 55, BusinessScopeID: 10, DeptID: 7}},
		K8sReader:      fakeK8sReader{ref: bizcap.K8sTargetRef{ClusterID: 99, Namespace: "prod", WorkloadKind: "Deployment", WorkloadName: "api", BusinessScopeID: 10, DeptID: 7}},
	})
	if err := manager.Migrate(); err != nil {
		t.Fatalf("migrate service schema: %v", err)
	}
	return manager
}

func TestServiceFoundationRejectsDuplicateCodesAndMixedTargets(t *testing.T) {
	manager := setupServiceTestDB(t)
	scope := &common.DataScopeReq{Mode: common.DataScopeModeDept, DeptID: 7}
	app, err := manager.CreateApplication(CreateApplicationRequest{Code: "app-a", Name: "App A", BusinessScopeID: 10}, "1", scope)
	if err != nil {
		t.Fatalf("create app: %v", err)
	}
	if _, err := manager.CreateApplication(CreateApplicationRequest{Code: "app-a", Name: "Duplicate", BusinessScopeID: 10}, "1", scope); err == nil || err.Error() != errApplicationCodeExist {
		t.Fatalf("expected duplicate application code, got %v", err)
	}
	svc, err := manager.CreateService(CreateServiceRequest{ApplicationID: app.ID, Code: "api", Name: "API", RuntimeType: "vm"}, "1", scope)
	if err != nil {
		t.Fatalf("create service: %v", err)
	}
	if _, err := manager.CreateService(CreateServiceRequest{ApplicationID: app.ID, Code: "api", Name: "Duplicate", RuntimeType: "vm"}, "1", scope); err == nil || err.Error() != errServiceCodeExist {
		t.Fatalf("expected duplicate service code, got %v", err)
	}
	_, err = manager.CreateInstance(context.Background(), CreateInstanceRequest{
		ServiceID: svc.ID, Environment: "prod", TargetType: TargetTypeVM,
		HostID: 55, Namespace: "prod", WorkloadName: "api",
	}, "1", scope)
	if err == nil || err.Error() != errTargetInvalid {
		t.Fatalf("expected mixed target rejection, got %v", err)
	}
}

func TestServiceFoundationValidatesVMAndK8sTargets(t *testing.T) {
	manager := setupServiceTestDB(t)
	scope := &common.DataScopeReq{Mode: common.DataScopeModeDept, DeptID: 7}
	app, err := manager.CreateApplication(CreateApplicationRequest{Code: "app-b", Name: "App B", BusinessScopeID: 10}, "1", scope)
	if err != nil {
		t.Fatalf("create app: %v", err)
	}
	svc, err := manager.CreateService(CreateServiceRequest{ApplicationID: app.ID, Code: "api", Name: "API", RuntimeType: "hybrid"}, "1", scope)
	if err != nil {
		t.Fatalf("create service: %v", err)
	}
	vm, err := manager.CreateInstance(context.Background(), CreateInstanceRequest{ServiceID: svc.ID, Environment: "prod", TargetType: TargetTypeVM, HostID: 55}, "1", scope)
	if err != nil {
		t.Fatalf("create vm instance: %v", err)
	}
	if vm.TargetType != TargetTypeVM || vm.BusinessScopeID != 10 {
		t.Fatalf("unexpected vm ref: %+v", vm)
	}
	k8s, err := manager.CreateInstance(context.Background(), CreateInstanceRequest{ServiceID: svc.ID, Environment: "prod", TargetType: TargetTypeK8s, K8sClusterID: 99, Namespace: "prod", WorkloadKind: "deployment", WorkloadName: "api"}, "1", scope)
	if err != nil {
		t.Fatalf("create k8s instance: %v", err)
	}
	if k8s.WorkloadKind != "Deployment" {
		t.Fatalf("expected normalized workload kind, got %q", k8s.WorkloadKind)
	}
	if _, err := manager.CreateInstance(context.Background(), CreateInstanceRequest{ServiceID: svc.ID, Environment: "prod", TargetType: TargetTypeVM, HostID: 55}, "1", scope); err == nil || err.Error() != errTargetConflict {
		t.Fatalf("expected duplicate target conflict, got %v", err)
	}
}

func TestServiceRuntimeTypeRejectsIncompatibleInstanceTarget(t *testing.T) {
	manager := setupServiceTestDB(t)
	scope := &common.DataScopeReq{Mode: common.DataScopeModeDept, DeptID: 7}
	app, err := manager.CreateApplication(CreateApplicationRequest{Code: "app-runtime", Name: "Runtime App", BusinessScopeID: 10}, "1", scope)
	if err != nil {
		t.Fatalf("create app: %v", err)
	}
	vmService, err := manager.CreateService(CreateServiceRequest{ApplicationID: app.ID, Code: "vm-api", Name: "VM API", RuntimeType: RuntimeTypeVM}, "1", scope)
	if err != nil {
		t.Fatalf("create service: %v", err)
	}
	_, err = manager.CreateInstance(context.Background(), CreateInstanceRequest{
		ServiceID: vmService.ID, Environment: "prod", TargetType: TargetTypeK8s,
		K8sClusterID: 99, Namespace: "prod", WorkloadKind: "Deployment", WorkloadName: "api",
	}, "1", scope)
	if err == nil || err.Error() != errTargetInvalid {
		t.Fatalf("expected runtime/target mismatch rejection, got %v", err)
	}

	invalidRuntime := "container"
	if _, err := manager.UpdateService(vmService.ID, UpdateServiceRequest{RuntimeType: &invalidRuntime}, "1", scope); err == nil || err.Error() != errTargetInvalid {
		t.Fatalf("expected invalid runtime rejection, got %v", err)
	}
}

func TestServiceUpdatesReturnPersistedValues(t *testing.T) {
	manager := setupServiceTestDB(t)
	scope := &common.DataScopeReq{Mode: common.DataScopeModeDept, DeptID: 7}
	app, err := manager.CreateApplication(CreateApplicationRequest{Code: "app-update", Name: "App", BusinessScopeID: 10}, "1", scope)
	if err != nil {
		t.Fatalf("create app: %v", err)
	}
	svc, err := manager.CreateService(CreateServiceRequest{ApplicationID: app.ID, Code: "api", Name: "API", RuntimeType: "vm"}, "1", scope)
	if err != nil {
		t.Fatalf("create service: %v", err)
	}
	newName := "API v2"
	updatedService, err := manager.UpdateService(svc.ID, UpdateServiceRequest{Name: &newName}, "2", scope)
	if err != nil {
		t.Fatalf("update service: %v", err)
	}
	if updatedService.Name != newName {
		t.Fatalf("expected persisted service name %q, got %q", newName, updatedService.Name)
	}

	instance, err := manager.CreateInstance(context.Background(), CreateInstanceRequest{ServiceID: svc.ID, Environment: "prod", TargetType: TargetTypeVM, HostID: 55}, "1", scope)
	if err != nil {
		t.Fatalf("create instance: %v", err)
	}
	version := "1.2.0"
	updatedInstance, err := manager.UpdateInstance(instance.ID, UpdateInstanceRequest{DesiredVersion: &version}, "2", scope)
	if err != nil {
		t.Fatalf("update instance: %v", err)
	}
	if updatedInstance.DesiredVersion != version || updatedInstance.LifecycleVersion != 1 {
		t.Fatalf("expected persisted instance update, got %+v", updatedInstance)
	}
}
