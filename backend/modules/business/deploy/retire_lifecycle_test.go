package deploy

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	bizcap "github.com/duanxldragon/pantheon-base/backend/modules/business/capability"
	"github.com/duanxldragon/pantheon-base/backend/modules/business/cmdb"
	serviceowner "github.com/duanxldragon/pantheon-base/backend/modules/business/service"
	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// Raw JSON keeps these regressions compilable before the production schema adds
// kind. No existing executable snapshot is converted by the fixture's APIs.
const retireMetadataSnapshot = `{"kind":"metadata_only","templateId":0,"templateName":"","templateVersion":"","steps":[]}`

type retireCMDBRecorder struct {
	cmdb.DeployCMDBCapability
	cmdb.TargetSelectionCapability
	writes *atomic.Int32
}

func (r *retireCMDBRecorder) WriteDeployHostResult(req cmdb.DeployHostWritebackRequest) error {
	r.writes.Add(1)
	return r.DeployCMDBCapability.WriteDeployHostResult(req)
}

func (r *retireCMDBRecorder) WithTargetTransaction(tx *gorm.DB) cmdb.TargetSelectionCapability {
	bound := r.TargetSelectionCapability.WithTargetTransaction(tx)
	return &retireCMDBRecorder{DeployCMDBCapability: bound.(cmdb.DeployCMDBCapability), TargetSelectionCapability: bound, writes: r.writes}
}

type retireStateCalls struct {
	mu    sync.Mutex
	calls []bizcap.ServiceInstanceStateTransition
}

type retireStateRecorder struct {
	bizcap.ServiceInstanceStateApplier
	record *retireStateCalls
}

func (r *retireStateRecorder) ApplyServiceInstanceState(ctx context.Context, req bizcap.ServiceInstanceStateTransition, actor string, scope *common.DataScopeReq) error {
	r.record.mu.Lock()
	r.record.calls = append(r.record.calls, req)
	r.record.mu.Unlock()
	// Delegate to the real owner on the exact transaction provided by Deploy.
	return r.ServiceInstanceStateApplier.ApplyServiceInstanceState(ctx, req, actor, scope)
}

func (r *retireStateCalls) snapshot() []bizcap.ServiceInstanceStateTransition {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]bizcap.ServiceInstanceStateTransition(nil), r.calls...)
}

func recordRetireOwners(s *DeployService) (*atomic.Int32, *retireStateCalls) {
	writes := &atomic.Int32{}
	s.cmdbCapability = &retireCMDBRecorder{DeployCMDBCapability: s.cmdbCapability, TargetSelectionCapability: s.cmdbCapability.(cmdb.TargetSelectionCapability), writes: writes}
	record := &retireStateCalls{}
	owner, factory := s.serviceState, s.serviceStateTransaction
	s.SetServiceInstanceStateApplierWithTransaction(owner, func(tx *gorm.DB) bizcap.ServiceInstanceStateApplier {
		return &retireStateRecorder{ServiceInstanceStateApplier: factory(tx), record: record}
	})
	return writes, record
}

func retireJSON(t *testing.T, value any) datatypes.JSON {
	t.Helper()
	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func retireCreateFixture(t *testing.T) (*gorm.DB, *DeployService, CreateTaskRequest, serviceowner.ServiceInstance) {
	t.Helper()
	db, s, draft, _, instance := serviceLinkDraftFixture(t)
	if err := db.Model(&instance).Updates(map[string]any{"desired_state": "stopped", "observed_state": "stopped"}).Error; err != nil {
		t.Fatal(err)
	}
	return db, s, CreateTaskRequest{Name: "retire-metadata", PackageID: draft.PackageID, BusinessScopeID: draft.BusinessScopeID, ServiceID: instance.ServiceID, ServiceInstanceID: instance.ID, TargetType: TargetTypeHost, TargetIDs: draft.TargetIDs, Action: TaskActionRetire, ExecutorType: ExecutorTypeManual}, instance
}

// Model a genuinely admitted metadata task for completion REDs, independently
// of Create's missing marker. Target/owner binding and the reservation are real.
func retireAdmittedFixture(t *testing.T) (*gorm.DB, *DeployService, DeployTask, DeployTaskHost, serviceowner.ServiceInstance) {
	t.Helper()
	db, s, task, host, instance := linkedServiceFixture(t, false)
	instance.DesiredState, instance.ObservedState = "stopped", "stopped"
	instance.DesiredVersion, instance.CurrentVersion, instance.RollbackVersion = "keep-desired", "keep-current", "keep-rollback"
	if err := db.Save(&instance).Error; err != nil {
		t.Fatal(err)
	}
	task.Action, task.ExecutorType, task.TargetType = TaskActionRetire, ExecutorTypeManual, TargetTypeHost
	task.TargetIDs = retireJSON(t, []uint64{host.HostID})
	task.PackageVersion = "reference-version-must-not-write"
	task.ExecutionSnapshot = datatypes.JSON(retireMetadataSnapshot)
	setLinkedFixtureSnapshot(t, &task, host, instance)
	if err := db.Save(&task).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Where("task_id = ?", task.ID).Delete(&DeployTaskStep{}).Error; err != nil {
		t.Fatal(err)
	}
	components := `[{"name":"owned","version":"keep-current","deployedAt":"2026-08-18T00:00:00Z","deployTaskId":91,"deployTaskName":"original-install","executorType":"ssh"},{"name":"unrelated","version":"3"}]`
	if err := db.Table("biz_cmdb_host").Where("id = ?", host.HostID).Update("installed_components", components).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&DeployHostLease{HostID: host.HostID, TaskID: task.ID, Owner: deployHostLeaseOwner(task.ID), ExpiresAt: time.Now().Add(time.Hour)}).Error; err != nil {
		t.Fatal(err)
	}
	// MySQL JSON normalizes serialization. Compare immutable intent against its
	// persisted representation, never against bytes from before the write.
	if err := db.First(&task, task.ID).Error; err != nil {
		t.Fatal(err)
	}
	return db, s, task, host, instance
}

func retireResourceFacts(t *testing.T, db *gorm.DB, hostID uint64) string {
	t.Helper()
	var rows []map[string]any
	if err := db.Table("biz_cmdb_host").Where("id = ?", hostID).Find(&rows).Error; err != nil {
		t.Fatal(err)
	}
	return string(retireJSON(t, rows))
}

func assertRetireFactsUnchanged(t *testing.T, db *gorm.DB, before string) {
	t.Helper()
	if after := linkedServiceFacts(t, db); after != before {
		t.Error("rejected Retire changed task, snapshot, host, attempt, lease, event or Service facts")
	}
}

func TestRetireMetadataRepresentationAndAdmission(t *testing.T) {
	t.Run("resolver-marker-contract", func(t *testing.T) {
		step := `{"stepCode":"legacy","stepType":"package","action":"install","package":{"id":3,"name":"legacy","version":"1","installCommand":"echo install"}}`
		for _, tc := range []struct {
			name, raw, action, executor string
			linked, valid               bool
		}{
			{"metadata", retireMetadataSnapshot, TaskActionRetire, ExecutorTypeManual, true, true},
			{"missing-steps", `{"kind":"metadata_only"}`, TaskActionRetire, ExecutorTypeManual, true, false},
			{"null-steps", `{"kind":"metadata_only","steps":null}`, TaskActionRetire, ExecutorTypeManual, true, false},
			{"polluted-marker", `{"kind":"metadata_only","steps":[` + step + `]}`, TaskActionRetire, ExecutorTypeManual, true, false},
			{"unknown-kind", `{"kind":"future_kind","steps":[` + step + `]}`, TaskActionInstall, ExecutorTypeManual, true, false},
			{"wrong-action", retireMetadataSnapshot, TaskActionInstall, ExecutorTypeManual, true, false},
			{"wrong-executor", retireMetadataSnapshot, TaskActionRetire, ExecutorTypeSSH, true, false},
			{"standalone-marker", retireMetadataSnapshot, TaskActionRetire, ExecutorTypeManual, false, false},
			{"legacy-empty", `{"steps":[]}`, TaskActionRetire, ExecutorTypeManual, true, false},
			{"legacy-retire", `{"steps":[` + step + `]}`, TaskActionRetire, ExecutorTypeManual, true, false},
			{"legacy-install", `{"steps":[` + step + `]}`, TaskActionInstall, ExecutorTypeManual, true, true},
		} {
			t.Run(tc.name, func(t *testing.T) {
				task := DeployTask{Action: tc.action, ExecutorType: tc.executor, ExecutionSnapshot: datatypes.JSON(tc.raw)}
				if tc.linked {
					task.ServiceID, task.ServiceInstanceID = 1, 2
				}
				plan, err := (&DeployService{}).resolveTaskExecutionPlan(task)
				if tc.valid {
					if err != nil || (tc.action == TaskActionRetire && len(plan) != 0) {
						t.Fatalf("valid frozen intent rejected or made executable: err=%v steps=%d", err, len(plan))
					}
				} else if err == nil {
					t.Fatal("invalid metadata/legacy Retire intent admitted")
				}
			})
		}
	})
	t.Run("create-references-do-not-grant-execution", func(t *testing.T) {
		db, s, req, _ := retireCreateFixture(t)
		template, err := s.CreateTemplate(CreateTemplateRequest{Name: "retire-reference", Version: "1", ExecutionMode: ExecutionModeOrchestrated, DefaultAction: TaskActionInstall, PackageID: req.PackageID, Status: TemplateStatusEnabled, Steps: []TemplateStepPayload{{StepCode: "referenced-script", StepName: "Referenced script", StepType: TemplateStepTypeScript, Action: TaskActionInstall, StepConfig: map[string]any{"script": "echo reference-command-must-not-be-copied"}}}}, "1")
		if err != nil {
			t.Fatal(err)
		}
		req.TemplateID = template.ID
		created, err := s.CreateTask(req, "1", nil)
		if err != nil {
			t.Fatal(err)
		}
		var task DeployTask
		if err := db.First(&task, created.ID).Error; err != nil {
			t.Fatal(err)
		}
		var snapshot map[string]json.RawMessage
		if err := json.Unmarshal(task.ExecutionSnapshot, &snapshot); err != nil {
			t.Fatal(err)
		}
		if string(snapshot["kind"]) != `"metadata_only"` || string(snapshot["steps"]) != "[]" {
			t.Fatalf("new Manual Retire did not freeze explicit metadata-only intent: %s", task.ExecutionSnapshot)
		}
		if task.PackageID != req.PackageID || task.TemplateID != template.ID || task.TemplateName != template.Name || task.TemplateVersion != template.Version {
			t.Fatal("metadata identity references were lost")
		}
		for _, forbidden := range []string{"installCommand", "uninstallCommand", "stepConfig", "templateParams", "reference-command-must-not-be-copied"} {
			if bytes.Contains(task.ExecutionSnapshot, []byte(forbidden)) {
				t.Errorf("metadata snapshot copied executable definition: %s", forbidden)
			}
		}
		preview, err := s.PreviewTask(task.ID, StartTaskRequest{}, "1", nil)
		if err != nil {
			t.Fatal(err)
		}
		if preview.Plan == nil || preview.Plan.StepCount != 0 || len(preview.Plan.Steps) != 0 {
			t.Fatal("metadata preview exposed execution steps")
		}
		for _, change := range preview.Changes {
			if change.Kind == "removed" || change.Before != change.After {
				t.Fatal("metadata preview promised a software component change")
			}
		}
	})
	for _, kind := range []string{"ssh", "standalone", "script", "precheckCommand", "postcheckCommand", "healthCheckCommand"} {
		t.Run("create-rejects-"+kind, func(t *testing.T) {
			db, s, req, _ := retireCreateFixture(t)
			switch kind {
			case "ssh":
				req.ExecutorType = ExecutorTypeSSH
			case "standalone":
				req.ServiceID, req.ServiceInstanceID = 0, 0
			default:
				req.TemplateParams = map[string]any{kind: "echo explicit-execution-intent"}
			}
			before := linkedServiceFacts(t, db)
			if _, err := s.CreateTask(req, "1", nil); err == nil {
				t.Error("unsupported Retire executor, owner or explicit command input accepted")
			}
			assertRetireFactsUnchanged(t, db, before)
		})
	}
	t.Run("nonretire-template-cannot-hide-retire-step", func(t *testing.T) {
		db, s, req, _ := retireCreateFixture(t)
		template, err := s.CreateTemplate(CreateTemplateRequest{Name: "explicit-retire-step", Version: "1", ExecutionMode: ExecutionModeOrchestrated, DefaultAction: TaskActionInstall, PackageID: req.PackageID, Status: TemplateStatusEnabled, Steps: []TemplateStepPayload{{StepCode: "retire-step", StepName: "Retire step", StepType: TemplateStepTypeScript, Action: TaskActionRetire, StepConfig: map[string]any{"script": "echo explicit-retire-step"}}}}, "1")
		if err != nil {
			t.Fatal(err)
		}
		req.Action, req.TemplateID = TaskActionInstall, template.ID
		before := linkedServiceFacts(t, db)
		if _, err := s.CreateTask(req, "1", nil); err == nil {
			t.Error("non-Retire task silently discarded a Retire template step action")
		}
		assertRetireFactsUnchanged(t, db, before)
	})
	t.Run("old-update-cannot-launder-snapshot", func(t *testing.T) {
		db, s, req, _ := retireCreateFixture(t)
		var task DeployTask
		if err := db.Where("name = ?", "preview-task").First(&task).Error; err != nil {
			t.Fatal(err)
		}
		task.Action, task.ExecutorType = TaskActionRetire, ExecutorTypeSSH
		task.ServiceID, task.ServiceInstanceID = req.ServiceID, req.ServiceInstanceID
		if err := db.Save(&task).Error; err != nil {
			t.Fatal(err)
		}
		if err := db.First(&task, task.ID).Error; err != nil {
			t.Fatal(err)
		}
		frozen := append([]byte(nil), task.ExecutionSnapshot...)
		before := linkedServiceFacts(t, db)
		manual := ExecutorTypeManual
		if _, err := s.UpdateTask(task.ID, UpdateTaskRequest{ExecutorType: &manual}, "1", nil); err == nil {
			t.Error("executor-only update laundered a legacy executable Retire")
		}
		assertRetireFactsUnchanged(t, db, before)
		if err := db.First(&task, task.ID).Error; err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(frozen, task.ExecutionSnapshot) {
			t.Fatal("rejected update rewrote the historical snapshot bytes")
		}
	})
	t.Run("template-default-update-cannot-introduce-marker", func(t *testing.T) {
		db, s, req, _ := retireCreateFixture(t)
		var task DeployTask
		if err := db.Where("name = ?", "preview-task").First(&task).Error; err != nil {
			t.Fatal(err)
		}
		template, err := s.CreateTemplate(CreateTemplateRequest{Name: "retire-default-reference", Version: "1", ExecutionMode: ExecutionModeOrchestrated, DefaultAction: TaskActionRetire, PackageID: req.PackageID, Status: TemplateStatusEnabled}, "1")
		if err != nil {
			t.Fatal(err)
		}
		before := linkedServiceFacts(t, db)
		if _, err := s.UpdateTask(task.ID, UpdateTaskRequest{TemplateID: &template.ID, ServiceID: &req.ServiceID, ServiceInstanceID: &req.ServiceInstanceID}, "1", nil); err == nil {
			t.Error("template default laundered an existing executable task into metadata")
		}
		assertRetireFactsUnchanged(t, db, before)
	})
}

func retireExecutableSteps() map[string]deployExecutionStep {
	pkg := DeployPackage{ID: 3, Name: "retire-component", Version: "1.26.2", InstallCommand: "echo install-must-not-run", UninstallCommand: "echo uninstall-must-not-run", ExecutionMode: ExecutionModeOrchestrated}
	params := map[string]any{"installRoot": "/data/nginx", "serviceName": "nginx"}
	fixed := pkg
	fixed.ExecutionMode, fixed.TemplateCode = ExecutionModeFixed, TemplateCodeNginxSystemd
	return map[string]deployExecutionStep{
		"package":   {StepCode: "package", StepType: TemplateStepTypePackage, Action: TaskActionInstall, Package: pkg},
		"script":    {StepCode: "script", StepType: TemplateStepTypeScript, Action: TaskActionInstall, StepConfig: map[string]any{"script": "echo custom-must-not-run"}},
		"fixed":     {StepCode: "fixed", StepType: TemplateStepTypePackage, Action: TaskActionInstall, Package: fixed, TemplateParams: params},
		"component": {StepCode: "component", StepType: defaultNginxServiceName, Action: TaskActionInstall, Package: pkg, TemplateParams: params},
	}
}

type retireCommandRecorder struct{ calls atomic.Int32 }

func (r *retireCommandRecorder) RunScript(context.Context, string) (string, string, error) {
	r.calls.Add(1)
	// Refuse rather than pretend software execution succeeded.
	return "", "", errors.New("retire test transport refuses software execution")
}

func (*retireCommandRecorder) Close() error { return nil }

func TestRetireExecutableBypassesHaveNoSoftwareEffects(t *testing.T) {
	db, s, task, host, _ := retireAdmittedFixture(t)
	for name, original := range retireExecutableSteps() {
		for _, mismatch := range []string{"task-retire-step-install", "task-install-step-retire"} {
			t.Run(name+"/"+mismatch, func(t *testing.T) {
				step, candidate := original, task
				if mismatch == "task-install-step-retire" {
					candidate.Action, step.Action = TaskActionInstall, TaskActionRetire
				}
				candidate.ExecutionSnapshot = retireJSON(t, deployExecutionSnapshot{Steps: []deployStepSnapshot{{StepCode: step.StepCode, StepType: step.StepType, Action: step.Action, Package: packageToSnapshot(step.Package), TemplateParams: step.TemplateParams, StepConfig: step.StepConfig}}})
				resourceBefore := retireResourceFacts(t, db, host.HostID)
				if _, err := s.resolveTaskExecutionPlan(candidate); err == nil {
					t.Error("legacy/mismatched Retire executable snapshot resolved")
				}
				if script, err := s.renderExecutionStepScript(step, candidate, cmdbHostSnapshot{ID: host.HostID, OS: "linux"}); err == nil || script != "" {
					t.Error("Retire renderer produced a software command")
				}
				runner, execution := &retireCommandRecorder{}, &taskHostExecution{}
				run := taskStepHostRun{task: candidate, taskHost: taskHostToResponse(&host), target: cmdbHostSnapshot{ID: host.HostID, OS: "linux"}, runner: runner, execution: execution}
				stop, _ := s.executeTaskStep(context.Background(), run, step)
				if !stop || execution.executionErr == nil || runner.calls.Load() != 0 {
					t.Errorf("Retire remote-step bypass reached script/probe: stop=%v err=%v calls=%d", stop, execution.executionErr, runner.calls.Load())
				}
				execution.recordComponent(step, candidate)
				if len(execution.summary.InstalledComponents) != 0 || len(execution.summary.RemovedComponentNames) != 0 {
					t.Error("Retire manufactured component execution facts")
				}
				if retireResourceFacts(t, db, host.HostID) != resourceBefore {
					t.Error("Retire bypass changed CMDB software/resource facts")
				}
			})
		}
	}
	for _, key := range []string{"precheckCommand", "postcheckCommand"} {
		t.Run("direct-"+key, func(t *testing.T) {
			runner, execution := &retireCommandRecorder{}, &taskHostExecution{}
			step := deployExecutionStep{StepCode: "check", StepType: TemplateStepTypeScript, Action: TaskActionInstall, StepConfig: map[string]any{key: "echo check-must-not-run"}}
			run := taskStepHostRun{task: task, taskHost: taskHostToResponse(&host), runner: runner, execution: execution}
			if !s.runTaskStepCheck(context.Background(), run, step, taskStepCheckConfig{configKey: key, phase: key}) || execution.executionErr == nil || runner.calls.Load() != 0 {
				t.Errorf("Retire check bypass called transport: err=%v calls=%d", execution.executionErr, runner.calls.Load())
			}
		})
	}
	t.Run("generic-and-fixed-package-renderers", func(t *testing.T) {
		for name, step := range retireExecutableSteps() {
			if step.Package.ID == 0 {
				continue
			}
			candidate := task
			candidate.TemplateParams = retireJSON(t, step.TemplateParams)
			if script, err := s.resolveInstallScript(step.Package, candidate); err == nil || script != "" {
				t.Errorf("%s generic/fixed Retire resolver produced software commands", name)
			}
		}
	})
	for _, entry := range []string{"start", "resume", "retry", "claim"} {
		t.Run("legacy-"+entry, func(t *testing.T) {
			db, s, task, host, _ := retireAdmittedFixture(t)
			step := retireExecutableSteps()["package"]
			task.ExecutionSnapshot = retireJSON(t, deployExecutionSnapshot{Steps: []deployStepSnapshot{{StepCode: step.StepCode, StepType: step.StepType, Action: step.Action, Package: packageToSnapshot(step.Package)}}})
			switch entry {
			case "start":
				task.Status = TaskStatusDraft
			case "resume":
				task.Status = TaskStatusPaused
			case "retry":
				task.Status, host.Status = TaskStatusFailed, TaskHostStatusFailed
				if err := db.Save(&host).Error; err != nil {
					t.Fatal(err)
				}
			}
			if err := db.Save(&task).Error; err != nil {
				t.Fatal(err)
			}
			before := linkedServiceFacts(t, db)
			var err error
			switch entry {
			case "start":
				_, err = s.StartTask(task.ID, StartTaskRequest{}, "1", nil)
			case "resume":
				_, err = s.ResumeTask(task.ID, "1", nil)
			case "retry":
				_, err = s.RetryTask(task.ID, "1", nil)
			case "claim":
				_, err = s.claimOwnedAttempt(task.ID, host.ID, "retire-must-not-claim")
			}
			if err == nil {
				t.Error("legacy executable Retire admitted through " + entry)
			}
			assertRetireFactsUnchanged(t, db, before)
		})
	}
	t.Run("metadata-cannot-acquire-worker-attempt", func(t *testing.T) {
		db, s, task, host, _ := retireAdmittedFixture(t)
		before := linkedServiceFacts(t, db)
		if _, err := s.claimOwnedAttempt(task.ID, host.ID, "metadata-must-not-claim"); err == nil {
			t.Error("metadata-only Retire acquired a remote worker attempt")
		}
		assertRetireFactsUnchanged(t, db, before)
	})
}

func TestRetireMetadataCompletionIsAtomicAndVersionNeutral(t *testing.T) {
	t.Run("full-create-preview-start-complete-replay", func(t *testing.T) {
		db, s, req, instance := retireCreateFixture(t)
		instance.DesiredState, instance.ObservedState = "stopped", "stopped"
		instance.DesiredVersion, instance.CurrentVersion, instance.RollbackVersion = "flow-desired", "flow-current", "flow-rollback"
		if err := db.Save(&instance).Error; err != nil {
			t.Fatal(err)
		}
		writes, record := recordRetireOwners(s)
		var credentialReads, runnerCreates atomic.Int32
		if err := db.Callback().Query().Before("gorm:query").Register("retire_no_credentials", func(tx *gorm.DB) {
			if tx.Statement.Table == "biz_deploy_credential_ref" {
				credentialReads.Add(1)
			}
		}); err != nil {
			t.Fatal(err)
		}
		defer db.Callback().Query().Remove("retire_no_credentials")
		s.sshRunnerFactory = func(context.Context, cmdbHostSnapshot, StartTaskRequest) (deploySSHRunner, error) {
			runnerCreates.Add(1)
			return nil, errors.New("metadata lifecycle must not create a remote runner")
		}
		resourceBefore := retireResourceFacts(t, db, req.TargetIDs[0])
		created, err := s.CreateTask(req, "1", nil)
		if err != nil {
			t.Fatal(err)
		}
		preview, err := s.PreviewTask(created.ID, StartTaskRequest{}, "1", nil)
		if err != nil {
			t.Fatal(err)
		}
		if preview.Plan == nil || preview.Plan.StepCount != 0 || len(preview.Plan.Steps) != 0 {
			t.Error("full metadata lifecycle preview contains executable steps")
		}
		for _, change := range preview.Changes {
			if change.Kind == "removed" || change.Before != change.After {
				t.Error("full metadata lifecycle preview promises a software change")
			}
		}
		started, err := s.StartTask(created.ID, StartTaskRequest{RequirePreview: true, IntentVersion: preview.IntentVersion}, "1", nil)
		if err != nil {
			t.Fatal(err)
		}
		if len(started.Hosts) != 1 || started.Hosts[0].ExecutionVersion == "" {
			t.Fatal("actual Start did not admit a bound metadata host")
		}
		var steps, attempts int64
		if err := db.Model(&DeployTaskStep{}).Where("task_id = ?", created.ID).Count(&steps).Error; err != nil {
			t.Fatal(err)
		}
		if err := db.Model(&DeployTaskAttempt{}).Where("task_id = ?", created.ID).Count(&attempts).Error; err != nil {
			t.Fatal(err)
		}
		if steps != 0 || attempts != 0 || len(record.snapshot()) != 0 {
			t.Error("metadata Start created execution steps/attempts or began the Service owner")
		}
		report := MarkHostResultRequest{Status: TaskHostStatusSuccess, ExecutionVersion: started.Hosts[0].ExecutionVersion, ReportKey: "full-metadata-flow"}
		if _, err := s.MarkHostResult(started.Hosts[0].ID, report, "1", nil); err != nil {
			t.Fatal(err)
		}
		settled := linkedServiceFacts(t, db)
		if _, err := s.MarkHostResult(started.Hosts[0].ID, report, "1", nil); err != nil {
			t.Fatal(err)
		}
		var after serviceowner.ServiceInstance
		if err := db.First(&after, instance.ID).Error; err != nil {
			t.Fatal(err)
		}
		if credentialReads.Load() != 0 || runnerCreates.Load() != 0 || writes.Load() != 0 || retireResourceFacts(t, db, req.TargetIDs[0]) != resourceBefore {
			t.Errorf("actual metadata lifecycle performed execution/projection: credential=%d runner=%d cmdb=%d", credentialReads.Load(), runnerCreates.Load(), writes.Load())
		}
		if after.DesiredVersion != instance.DesiredVersion || after.CurrentVersion != instance.CurrentVersion || after.RollbackVersion != instance.RollbackVersion {
			t.Error("actual metadata lifecycle changed software versions")
		}
		if after.DesiredState != "retired" || after.ObservedState != "retired" || after.LifecycleVersion != instance.LifecycleVersion+1 || len(record.snapshot()) != 1 || linkedServiceFacts(t, db) != settled {
			t.Error("actual metadata lifecycle did not retire exactly once with immutable replay")
		}
	})
	t.Run("success-no-cmdb-or-version-write-and-replay", func(t *testing.T) {
		db, s, task, host, instance := retireAdmittedFixture(t)
		writes, record := recordRetireOwners(s)
		resourceBefore := retireResourceFacts(t, db, host.HostID)
		frozen := append([]byte(nil), task.ExecutionSnapshot...)
		req := MarkHostResultRequest{Status: TaskHostStatusSuccess, ExecutionVersion: taskExecutionVersion(&task), ReportKey: "metadata-retire-finish"}
		if _, err := s.MarkHostResult(host.ID, req, "1", nil); err != nil {
			t.Fatal(err)
		}
		if writes.Load() != 0 || retireResourceFacts(t, db, host.HostID) != resourceBefore {
			t.Errorf("metadata completion invoked CMDB or changed software/resource facts: calls=%d", writes.Load())
		}
		var after serviceowner.ServiceInstance
		if err := db.First(&after, instance.ID).Error; err != nil {
			t.Fatal(err)
		}
		if after.DesiredVersion != instance.DesiredVersion || after.CurrentVersion != instance.CurrentVersion || after.RollbackVersion != instance.RollbackVersion {
			t.Error("metadata completion changed desired/current/rollback software versions")
		}
		calls := record.snapshot()
		if len(calls) != 1 || calls[0].Action != TaskActionRetire || calls[0].DesiredVersion != "" || calls[0].CurrentVersion != "" || after.ObservedState != "retired" || after.DesiredState != "retired" || after.LifecycleVersion != instance.LifecycleVersion+1 {
			t.Errorf("metadata did not issue exactly one version-neutral owner Retire: calls=%+v owner=%+v", calls, after)
		}
		var leases int64
		if err := db.Model(&DeployHostLease{}).Where("host_id = ? AND task_id = ?", host.HostID, task.ID).Count(&leases).Error; err != nil || leases != 0 {
			t.Fatalf("metadata reservation did not settle: err=%v leases=%d", err, leases)
		}
		settled := linkedServiceFacts(t, db)
		if _, err := s.MarkHostResult(host.ID, req, "1", nil); err != nil {
			t.Fatal(err)
		}
		if linkedServiceFacts(t, db) != settled || len(record.snapshot()) != 1 || writes.Load() != 0 {
			t.Error("metadata terminal replay mutated facts or invoked an owner again")
		}
		if err := db.First(&task, task.ID).Error; err != nil || !bytes.Equal(task.ExecutionSnapshot, frozen) {
			t.Fatal("metadata completion/replay changed the frozen snapshot")
		}
	})
	for _, kind := range []string{"installed", "removed", "mixed", "legacy-success"} {
		t.Run("rejects-"+kind+"-atomically", func(t *testing.T) {
			db, s, task, host, _ := retireAdmittedFixture(t)
			summary := deployExecutionSummary{}
			if kind == "installed" || kind == "mixed" {
				summary.InstalledComponents = []cmdb.InstalledComponentUpsert{{Name: "unexpected-install", Version: "2"}}
			}
			if kind == "removed" || kind == "mixed" {
				summary.RemovedComponentNames = []string{"owned"}
			}
			if kind == "legacy-success" {
				task.ExecutionSnapshot = retireJSON(t, deployExecutionSnapshot{Steps: []deployStepSnapshot{{StepCode: "legacy", StepType: TemplateStepTypePackage, Action: TaskActionRetire}}})
				if err := db.Save(&task).Error; err != nil {
					t.Fatal(err)
				}
			}
			writes, record := recordRetireOwners(s)
			before := linkedServiceFacts(t, db)
			_, err := s.markHostResultWithSummary(host.ID, MarkHostResultRequest{Status: TaskHostStatusSuccess, ExecutionVersion: taskExecutionVersion(&task)}, "1", nil, summary)
			if err == nil || writes.Load() != 0 || len(record.snapshot()) != 0 {
				t.Errorf("invalid Retire report escaped before owner/projection: err=%v cmdb=%d state=%d", err, writes.Load(), len(record.snapshot()))
			}
			assertRetireFactsUnchanged(t, db, before)
		})
	}
	t.Run("failed-no-begin-retry-and-stale-replay", func(t *testing.T) {
		db, s, task, host, instance := retireAdmittedFixture(t)
		writes, record := recordRetireOwners(s)
		resourceBefore := retireResourceFacts(t, db, host.HostID)
		frozen := append([]byte(nil), task.ExecutionSnapshot...)
		old := MarkHostResultRequest{Status: TaskHostStatusFailed, ErrorMessage: "metadata acknowledgement failed", ExecutionVersion: taskExecutionVersion(&task), ReportKey: "retire-failed"}
		if _, err := s.MarkHostResult(host.ID, old, "1", nil); err != nil {
			t.Fatal(err)
		}
		retried, err := s.RetryTask(task.ID, "1", nil)
		if err != nil {
			t.Fatal(err)
		}
		if len(retried.Hosts) != 1 || retried.Hosts[0].ExecutionVersion == "" || retried.Hosts[0].ExecutionVersion == old.ExecutionVersion {
			t.Fatal("metadata Retry did not rotate execution-generation binding")
		}
		var current serviceowner.ServiceInstance
		if err := db.First(&current, instance.ID).Error; err != nil {
			t.Fatal(err)
		}
		if current.LifecycleVersion != instance.LifecycleVersion || len(record.snapshot()) != 0 || writes.Load() != 0 || retireResourceFacts(t, db, host.HostID) != resourceBefore {
			t.Fatal("metadata failure/retry began owner lifecycle or changed CMDB")
		}
		before := linkedServiceFacts(t, db)
		if _, err := s.MarkHostResult(host.ID, old, "1", nil); err == nil {
			t.Error("old metadata acknowledgement settled retried intent")
		}
		assertRetireFactsUnchanged(t, db, before)
		if err := db.First(&task, task.ID).Error; err != nil || !bytes.Equal(task.ExecutionSnapshot, frozen) {
			t.Fatal("metadata Retry rewrote the frozen lifecycle intent")
		}
	})
	t.Run("actual-owner-failure-rolls-back", func(t *testing.T) {
		db, s, task, host, _ := retireAdmittedFixture(t)
		writes, record := recordRetireOwners(s)
		injected := errors.New("actual Retire owner SQL refused")
		if err := db.Callback().Update().Before("gorm:update").Register("retire_owner_refusal", func(tx *gorm.DB) {
			if tx.Statement.Table == "biz_service_instance" {
				tx.AddError(injected)
			}
		}); err != nil {
			t.Fatal(err)
		}
		defer db.Callback().Update().Remove("retire_owner_refusal")
		before := linkedServiceFacts(t, db)
		_, err := s.MarkHostResult(host.ID, MarkHostResultRequest{Status: TaskHostStatusSuccess, ExecutionVersion: taskExecutionVersion(&task)}, "1", nil)
		if !errors.Is(err, injected) || len(record.snapshot()) != 1 || writes.Load() != 0 {
			t.Errorf("real owner fault did not roll back without a CMDB call: err=%v owner=%d cmdb=%d", err, len(record.snapshot()), writes.Load())
		}
		assertRetireFactsUnchanged(t, db, before)
	})
}

func TestRetireUninstallAndHistoricalCancellationCompatibility(t *testing.T) {
	t.Run("real-uninstall-stop-retire-and-component-removal", func(t *testing.T) {
		db, s, task, host, instance := linkedServiceFixture(t, true)
		writes, record := recordRetireOwners(s)
		if _, err := s.MarkHostResult(host.ID, MarkHostResultRequest{Status: TaskHostStatusSuccess, ExecutionVersion: taskExecutionVersion(&task)}, "1", nil); err != nil {
			t.Fatal(err)
		}
		calls := record.snapshot()
		if len(calls) != 2 || calls[0].Action != TaskActionStop || calls[1].Action != TaskActionRetire || calls[0].CorrelationID != fmt.Sprintf("deploy:%d:finish:success", task.ID) || calls[1].CorrelationID != fmt.Sprintf("deploy:%d:retire", task.ID) || calls[1].ExpectedLifecycleVersion != instance.LifecycleVersion+1 {
			t.Fatalf("Uninstall STOP-to-Retire contract changed: %+v", calls)
		}
		var after serviceowner.ServiceInstance
		if err := db.First(&after, instance.ID).Error; err != nil || after.LifecycleVersion != instance.LifecycleVersion+2 || after.ObservedState != "retired" {
			t.Fatalf("real Uninstall owner did not retire: err=%v owner=%+v", err, after)
		}
		var resource deployTestHost
		if err := db.First(&resource, host.HostID).Error; err != nil {
			t.Fatal(err)
		}
		var components []any
		if err := json.Unmarshal(resource.InstalledComponents, &components); err != nil || len(components) != 0 || writes.Load() != 1 {
			t.Fatalf("Uninstall no longer removed software exactly once: err=%v components=%s calls=%d", err, resource.InstalledComponents, writes.Load())
		}
	})
	t.Run("legacy-read-and-cancel-preserve-snapshot", func(t *testing.T) {
		db, s, task, _, _ := retireAdmittedFixture(t)
		task.ExecutionSnapshot = retireJSON(t, deployExecutionSnapshot{Steps: []deployStepSnapshot{{StepCode: "legacy-retire", StepType: TemplateStepTypeScript, Action: TaskActionRetire, StepConfig: map[string]any{"script": "echo historical-install"}}}})
		if err := db.Save(&task).Error; err != nil {
			t.Fatal(err)
		}
		if err := db.First(&task, task.ID).Error; err != nil {
			t.Fatal(err)
		}
		frozen := append([]byte(nil), task.ExecutionSnapshot...)
		if _, err := s.GetTask(task.ID, nil); err != nil {
			t.Fatal(err)
		}
		writes, record := recordRetireOwners(s)
		if _, err := s.CancelTask(task.ID, "1", nil); err != nil {
			t.Fatal(err)
		}
		if err := db.First(&task, task.ID).Error; err != nil || task.Status != TaskStatusCanceled || !bytes.Equal(task.ExecutionSnapshot, frozen) {
			t.Fatalf("legacy Retire cancellation destroyed historical intent: err=%v status=%s", err, task.Status)
		}
		if writes.Load() != 0 || len(record.snapshot()) != 0 || !strings.Contains(string(task.ExecutionSnapshot), "historical-install") {
			t.Fatal("legacy cancellation executed/projected Retire or rewrote history")
		}
	})
}

type retireQueueLog struct {
	mu sync.Mutex
	bytes.Buffer
}

func (b *retireQueueLog) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.Buffer.Write(p)
}

func (b *retireQueueLog) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.Buffer.String()
}

func captureRetireQueueLog(t *testing.T) *retireQueueLog {
	t.Helper()
	output := &retireQueueLog{}
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(output, nil)))
	t.Cleanup(func() { slog.SetDefault(previous) })
	return output
}

// Join the actual worker, including its deferred reservation release. Query
// entry gates its first task read; no timing sample stands in for worker exit.
func runRetireQueueWorker(t *testing.T, db *gorm.DB, s *DeployService, taskID uint64, direct bool) {
	t.Helper()
	captured, resume := make(chan struct{}), make(chan struct{})
	release := sync.OnceFunc(func() { close(resume) })
	var capture sync.Once
	var slot *taskWorkerSlot
	if err := db.Callback().Query().Before("gorm:query").Register("retire_queue_worker_capture", func(tx *gorm.DB) {
		if tx.Statement.Table == "biz_deploy_task" {
			if current, present := s.taskWorkers.Load(taskID); present {
				capture.Do(func() {
					slot = current.(*taskWorkerSlot)
					close(captured)
					<-resume
				})
			}
		}
	}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		release()
		select {
		case <-captured:
			waitOwnership(t, slot.done, "rejected queue worker cleanup")
		default:
		}
		db.Callback().Query().Remove("retire_queue_worker_capture")
	})
	if direct {
		go s.runAsyncTask(taskID, "retire-queue")
	} else if count, err := s.ReconcileDeployQueue("retire-queue"); err != nil || count != 1 {
		t.Fatalf("real queue did not dispatch one durable task: count=%d err=%v", count, err)
	}
	waitOwnership(t, captured, "capture rejected queue worker")
	release()
	waitOwnership(t, slot.done, "rejected queue worker whole exit")
	if _, present := s.taskWorkers.Load(taskID); present {
		t.Error("rejected queued intent retained worker slot")
	}
	if _, present := s.taskCancels.Load(taskID); present {
		t.Error("rejected queued intent retained cancellation handle")
	}
}

func TestRetireQueuedRejectionSettlesOrReportsFailure(t *testing.T) {
	for _, kind := range []string{"legacy-retire", "unknown-kind-install", "metadata-ssh", "metadata-manual-misdispatch"} {
		t.Run(kind, func(t *testing.T) {
			db, s, task, host, instance := retireAdmittedFixture(t)
			task.ExecutorType = ExecutorTypeSSH
			switch kind {
			case "legacy-retire":
				task.ExecutionSnapshot = retireJSON(t, deployExecutionSnapshot{Steps: []deployStepSnapshot{{StepCode: "legacy", StepType: TemplateStepTypeScript, Action: TaskActionRetire, StepConfig: map[string]any{"script": "echo must-not-run"}}}})
			case "unknown-kind-install":
				task.Action = TaskActionInstall
				task.ExecutionSnapshot = datatypes.JSON(`{"kind":"unexpected","steps":[]}`)
			case "metadata-manual-misdispatch":
				task.ExecutorType = ExecutorTypeManual
			}
			if err := db.Save(&task).Error; err != nil {
				t.Fatal(err)
			}
			if err := db.First(&task, task.ID).Error; err != nil {
				t.Fatal(err)
			}
			frozen := append([]byte(nil), task.ExecutionSnapshot...)
			resourceBefore := retireResourceFacts(t, db, host.HostID)
			ownerBefore := serviceOwnerFacts(t, db)
			writes, record := recordRetireOwners(s)
			var credentials, runners atomic.Int32
			if err := db.Callback().Query().Before("gorm:query").Register("retire_queue_credentials", func(tx *gorm.DB) {
				if tx.Statement.Table == "biz_deploy_credential_ref" {
					credentials.Add(1)
				}
			}); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { db.Callback().Query().Remove("retire_queue_credentials") })
			s.sshRunnerFactory = func(context.Context, cmdbHostSnapshot, StartTaskRequest) (deploySSHRunner, error) {
				runners.Add(1)
				return nil, errors.New("rejected queue intent must not create a remote runner")
			}
			output := captureRetireQueueLog(t)
			runRetireQueueWorker(t, db, s, task.ID, kind == "metadata-manual-misdispatch")
			if !strings.Contains(output.String(), errDeployTaskTemplateInvalid) {
				t.Errorf("queued intent rejection reason was not logged: %s", output.String())
			}
			if err := db.First(&task, task.ID).Error; err != nil {
				t.Fatal(err)
			}
			if err := db.First(&host, host.ID).Error; err != nil {
				t.Fatal(err)
			}
			if task.Status != TaskStatusFailed || host.Status != TaskHostStatusFailed || host.ErrorMessage != errDeployTaskTemplateInvalid || task.FinishedAt == nil || host.FinishedAt == nil {
				t.Errorf("queued rejection did not visibly settle task and host: task=%s host=%s reason=%q", task.Status, host.Status, host.ErrorMessage)
			}
			var leases, attempts, completed int64
			if err := db.Model(&DeployHostLease{}).Where("task_id = ?", task.ID).Count(&leases).Error; err != nil {
				t.Fatal(err)
			}
			if err := db.Model(&DeployTaskAttempt{}).Where("task_id = ?", task.ID).Count(&attempts).Error; err != nil {
				t.Fatal(err)
			}
			if err := db.Model(&DeployTaskEvent{}).Where("task_id = ? AND event_type = ? AND to_status = ?", task.ID, "task.completed", TaskStatusFailed).Count(&completed).Error; err != nil {
				t.Fatal(err)
			}
			if leases != 0 || attempts != 0 || completed != 1 || credentials.Load() != 0 || runners.Load() != 0 || writes.Load() != 0 {
				t.Errorf("rejection escaped queue-only settlement: leases=%d attempts=%d completed=%d credential=%d runner=%d cmdb=%d", leases, attempts, completed, credentials.Load(), runners.Load(), writes.Load())
			}
			if retireResourceFacts(t, db, host.HostID) != resourceBefore || !bytes.Equal(task.ExecutionSnapshot, frozen) {
				t.Error("queued rejection changed software resource facts or frozen intent")
			}
			if kind == "unknown-kind-install" {
				var after serviceowner.ServiceInstance
				if err := db.First(&after, instance.ID).Error; err != nil {
					t.Fatal(err)
				}
				if len(record.snapshot()) != 1 || after.ObservedState != TaskStatusFailed || after.LifecycleVersion != instance.LifecycleVersion+1 {
					t.Error("unknown-kind Install lost its existing real failure owner transition")
				}
			} else if len(record.snapshot()) != 0 || serviceOwnerFacts(t, db) != ownerBefore {
				t.Error("Retire rejection applied owner state or changed desired/current/rollback versions")
			}
			settled := linkedServiceFacts(t, db)
			s.runAsyncTask(task.ID, "retire-queue-replay")
			if linkedServiceFacts(t, db) != settled {
				t.Error("rejected worker replay changed terminal facts")
			}
		})
	}

	t.Run("running-attempt-is-not-overwritten", func(t *testing.T) {
		db, s, task, host, _ := linkedServiceFixture(t, false)
		_, _ = ownedClaim(t, s, task, host, "live-before-rejected-intent")
		task.Action = TaskActionRetire
		task.ExecutionSnapshot = datatypes.JSON(`{"steps":[]}`)
		if err := db.Save(&task).Error; err != nil {
			t.Fatal(err)
		}
		before := linkedServiceFacts(t, db)
		output := captureRetireQueueLog(t)
		runRetireQueueWorker(t, db, s, task.ID, false)
		assertRetireFactsUnchanged(t, db, before)
		if !strings.Contains(output.String(), errDeployTaskTemplateInvalid) || !strings.Contains(output.String(), errExecutionOwnership.Error()) {
			t.Errorf("live attempt refusal swallowed rejection or settlement cause: %s", output.String())
		}
	})

	for _, fault := range []string{"stale-owner", "missing-owner-snapshot", "owner-query-failure"} {
		t.Run(fault, func(t *testing.T) {
			db, s, task, _, instance := retireAdmittedFixture(t)
			task.ExecutorType = ExecutorTypeSSH
			settlementReason := errDeployServiceLinkInvalid
			if fault == "missing-owner-snapshot" {
				task.TargetSnapshot = datatypes.JSON(`{}`)
			}
			if err := db.Save(&task).Error; err != nil {
				t.Fatal(err)
			}
			if fault == "stale-owner" {
				if err := db.Model(&instance).Update("lifecycle_version", instance.LifecycleVersion+1).Error; err != nil {
					t.Fatal(err)
				}
				settlementReason = errServiceExecutionSuperseded.Error()
			}
			var queries atomic.Int32
			if fault == "owner-query-failure" {
				injected := errors.New("queued Retire owner SQL query refused")
				settlementReason = injected.Error()
				if err := db.Callback().Query().Before("gorm:query").Register("retire_queue_owner_query_failure", func(tx *gorm.DB) {
					if _, transaction := tx.Statement.ConnPool.(gorm.TxCommitter); transaction && tx.Statement.Table == "biz_service_instance" {
						queries.Add(1)
						tx.AddError(injected)
					}
				}); err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() { db.Callback().Query().Remove("retire_queue_owner_query_failure") })
			}
			before := linkedServiceFacts(t, db)
			writes, record := recordRetireOwners(s)
			output := captureRetireQueueLog(t)
			runRetireQueueWorker(t, db, s, task.ID, false)
			assertRetireFactsUnchanged(t, db, before)
			if writes.Load() != 0 || len(record.snapshot()) != 0 {
				t.Error("failed queue settlement applied owner or CMDB changes")
			}
			if fault == "owner-query-failure" && queries.Load() == 0 {
				t.Error("queue never reached actual transaction-bound owner query failure")
			}
			if !strings.Contains(output.String(), errDeployTaskTemplateInvalid) || !strings.Contains(output.String(), settlementReason) {
				t.Errorf("queue swallowed rejection or real settlement failure: %s", output.String())
			}
		})
	}
}
