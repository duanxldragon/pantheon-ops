package deploy

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/duanxldragon/pantheon-base/backend/modules/business/bizscope"
	"github.com/duanxldragon/pantheon-base/backend/modules/business/cmdb"
	cmdbgroup "github.com/duanxldragon/pantheon-base/backend/modules/business/cmdb/group"
	cmdbhost "github.com/duanxldragon/pantheon-base/backend/modules/business/cmdb/host"
	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
	"gorm.io/gorm"
	"reflect"
	"strings"
	"testing"
	"time"
)

func previewFixture(t *testing.T, selected bool) (*gorm.DB, *DeployService, *TaskResponse, uint64) {
	t.Helper()
	t.Setenv(deployCredentialKeyEnv, strings.Repeat("12", 32))
	db := setupDeployTestDB(t)
	if err := db.AutoMigrate(&cmdbhost.Host{}, &cmdbgroup.Group{}); err != nil {
		t.Fatal(err)
	}
	reader := bizscope.NewService(db)
	svc := NewDeployService(db, cmdb.NewDeployCMDBCapability(db, reader), reader)
	scopeID, hostID := seedDeployReliabilityFixture(t, db, "preview-host", "10.40.9.1")
	pkg, err := svc.CreatePackage(CreatePackageRequest{Name: "preview-package", Version: "2.0.0", InstallCommand: "echo install", Status: PackageStatusEnabled}, "1")
	if err != nil {
		t.Fatal(err)
	}
	req := CreateTaskRequest{Name: "preview-task", PackageID: pkg.ID, BusinessScopeID: scopeID, TargetType: TargetTypeHost, TargetIDs: []uint64{hostID}, ExecutorType: ExecutorTypeManual}
	if selected {
		selection := cmdb.TargetSelection{BusinessScopeID: scopeID, HostIDs: []uint64{hostID}}
		resolved, err := svc.ResolveTargetSelection(context.Background(), selection, nil)
		if err != nil {
			t.Fatal(err)
		}
		req.TargetSelection = &selection
		req.ScopeVersion = resolved.ScopeVersion
	}
	task, err := svc.CreateTask(req, "1", nil)
	if err != nil {
		t.Fatal(err)
	}
	return db, svc, task, hostID
}
func TestPreviewTicketRejectsForgeryExpiryAndBudget(t *testing.T) {
	t.Setenv(deployCredentialKeyEnv, strings.Repeat("12", 32))
	valid := taskPreviewTicket{TaskID: 7, Actor: "1", Revision: "facts", Expires: time.Now().Add(time.Minute).Unix()}
	token, err := signPreviewTicket(valid)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := parsePreviewTicket(token)
	if err != nil || parsed != valid {
		t.Fatalf("valid ticket: %v", err)
	}
	expired := valid
	expired.Expires = time.Now().Add(-time.Minute).Unix()
	expiredToken, _ := signPreviewTicket(expired)
	for _, bad := range []string{"", token + "x", strings.Repeat("x", 1025), expiredToken, "forged.signature"} {
		if _, err := parsePreviewTicket(bad); !errors.Is(err, ErrTaskPreviewConflict) {
			t.Fatalf("accepted invalid ticket: %v", err)
		}
	}
}
func TestPreviewRequiredValidStartAndReplay(t *testing.T) {
	db, svc, task, _ := previewFixture(t, true)
	if _, err := svc.StartTask(task.ID, StartTaskRequest{RequirePreview: true}, "1", nil); !errors.Is(err, ErrTaskPreviewRequired) {
		t.Fatalf("missing preview accepted: %v", err)
	}
	preview, err := svc.PreviewTask(task.ID, StartTaskRequest{}, "1", nil)
	if err != nil {
		t.Fatal(err)
	}
	if preview.TargetCount != 1 || len(preview.Changes) != 1 || preview.Changes[0].Kind != "added" {
		t.Fatalf("wrong preview: %+v", preview)
	}
	req := StartTaskRequest{IntentVersion: preview.IntentVersion, RequirePreview: true, IdempotencyKey: "preview-start"}
	started, err := svc.StartTask(task.ID, req, "1", nil)
	if err != nil {
		t.Fatal(err)
	}
	if started.Status != TaskStatusRunning || len(started.Hosts) != 1 {
		t.Fatalf("start did not claim: %+v", started)
	}
	replay, err := svc.StartTask(task.ID, req, "1", nil)
	if err != nil || !replay.StartRequestReused {
		t.Fatalf("replay: %v", err)
	}
	var count int64
	db.Model(&DeployTaskHost{}).Where("task_id = ?", task.ID).Count(&count)
	if count != 1 {
		t.Fatalf("duplicate host: %d", count)
	}
}
func TestPreviewRejectsActorRequestAndPersistedDrift(t *testing.T) {
	for _, kind := range []string{"actor", "request", "task", "host", "secret", "scope", "permission"} {
		t.Run(kind, func(t *testing.T) {
			db, svc, task, hostID := previewFixture(t, true)
			preview, err := svc.PreviewTask(task.ID, StartTaskRequest{}, "1", nil)
			if err != nil {
				t.Fatal(err)
			}
			req := StartTaskRequest{RequirePreview: true, IntentVersion: preview.IntentVersion}
			actor := "1"
			var scope *common.DataScopeReq
			switch kind {
			case "actor":
				actor = "2"
			case "request":
				req.TimeoutSeconds = 42
			case "task":
				err = db.Model(&DeployTask{}).Where("id = ?", task.ID).Update("remark", "changed").Error
			case "host":
				err = db.Model(&cmdbhost.Host{}).Where("id = ?", hostID).Update("ip", "10.40.9.2").Error
			case "secret":
				err = db.Create(&DeployTaskSecret{TaskID: task.ID, SecretKey: "password", SecretEncrypted: "ciphertext-change"}).Error
			case "scope":
				err = db.Model(&deployTestBizScope{}).Where("id = ?", task.BusinessScopeID).Update("status", "disabled").Error
			case "permission":
				scope = &common.DataScopeReq{Mode: common.DataScopeModeSelf, UserID: 999}
			}
			if err != nil {
				t.Fatal(err)
			}
			if _, err = svc.StartTask(task.ID, req, actor, scope); err == nil {
				t.Fatal("stale preview started")
			}
			var current DeployTask
			if err = db.First(&current, task.ID).Error; err != nil {
				t.Fatal(err)
			}
			if current.Status == TaskStatusRunning {
				t.Fatal("failed validation mutated task")
			}
			var count int64
			db.Model(&DeployTaskHost{}).Where("task_id = ?", task.ID).Count(&count)
			if count != 0 {
				t.Fatal("failed validation persisted host")
			}
		})
	}
}
func TestSelectedCreateRejectsStaleResolution(t *testing.T) {
	db, svc, task, hostID := previewFixture(t, true)
	selected := cmdb.TargetSelection{BusinessScopeID: task.BusinessScopeID, HostIDs: []uint64{hostID}}
	resolved, err := svc.ResolveTargetSelection(context.Background(), selected, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err = db.Model(&cmdbhost.Host{}).Where("id = ?", hostID).Update("hostname", "changed-name").Error; err != nil {
		t.Fatal(err)
	}
	_, err = svc.CreateTask(CreateTaskRequest{Name: "stale-create", PackageID: task.PackageID, TargetSelection: &selected, ScopeVersion: resolved.ScopeVersion, ExecutorType: ExecutorTypeManual}, "1", nil)
	if !errors.Is(err, cmdb.ErrTargetSelectionConflict) {
		t.Fatalf("accepted stale resolution: %v", err)
	}
	var count int64
	db.Model(&DeployTask{}).Where("name = ?", "stale-create").Count(&count)
	if count != 0 {
		t.Fatal("stale create persisted")
	}
}
func TestPreviewSecretMaterialIsNeverReturned(t *testing.T) {
	db, svc, task, _ := previewFixture(t, false)
	if err := db.Create(&DeployTaskSecret{TaskID: task.ID, SecretKey: "password", SecretEncrypted: "never-expose-ciphertext"}).Error; err != nil {
		t.Fatal(err)
	}
	preview, err := svc.PreviewTask(task.ID, StartTaskRequest{}, "1", nil)
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(preview)
	if strings.Contains(string(raw), "never-expose") || strings.Contains(string(raw), "echo install") {
		t.Fatal("preview leaked sensitive execution material")
	}
}

func TestSelectedUpdateKeepsNewProvenanceAndInvalidatesPreview(t *testing.T) {
	db, svc, task, hostID := previewFixture(t, true)
	preview, err := svc.PreviewTask(task.ID, StartTaskRequest{}, "1", nil)
	if err != nil {
		t.Fatal(err)
	}
	selection := cmdb.TargetSelection{BusinessScopeID: task.BusinessScopeID, Manual: []string{"10.40.9.1"}}
	result, err := svc.ResolveTargetSelection(context.Background(), selection, nil)
	if err != nil {
		t.Fatal(err)
	}
	updated, err := svc.UpdateTask(task.ID, UpdateTaskRequest{TargetSelection: &selection, ScopeVersion: result.ScopeVersion}, "1", nil)
	if err != nil {
		t.Fatal(err)
	}
	if updated.TargetSelection == nil || len(updated.TargetSelection.Manual) != 1 || updated.TargetIDs[0] != hostID {
		t.Fatal("update lost provenance")
	}
	if _, err := svc.StartTask(task.ID, StartTaskRequest{IntentVersion: preview.IntentVersion, RequirePreview: true}, "1", nil); err == nil {
		t.Fatal("old preview started changed selection")
	}
	var count int64
	db.Model(&DeployTaskHost{}).Where("task_id = ?", task.ID).Count(&count)
	if count != 0 {
		t.Fatal("failed start leaked host")
	}
	if _, err := svc.UpdateTask(task.ID, UpdateTaskRequest{TargetIDs: []uint64{hostID}}, "1", nil); !errors.Is(err, cmdb.ErrTargetSelectionInvalid) {
		t.Fatalf("legacy target edit bypassed selection: %v", err)
	}
}
func TestPreviewCredentialVersionChangesInvalidate(t *testing.T) {
	db, svc, task, _ := previewFixture(t, false)
	if err := db.Model(&DeployTask{}).Where("id = ?", task.ID).Update("executor_type", ExecutorTypeSSH).Error; err != nil {
		t.Fatal(err)
	}
	credential := setupDeployTestCredential(t, svc)
	req := StartTaskRequest{CredentialRefID: credential, HostFingerprint: "SHA256:preview-fixture", RequirePreview: true}
	preview, err := svc.PreviewTask(task.ID, req, "1", nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&DeployCredentialRef{}).Where("id = ?", credential).Update("version", 2).Error; err != nil {
		t.Fatal(err)
	}
	req.IntentVersion = preview.IntentVersion
	if _, err := svc.StartTask(task.ID, req, "1", nil); !errors.Is(err, ErrTaskPreviewConflict) {
		t.Fatalf("changed credential accepted: %v", err)
	}
}

func TestPreviewSigningConfigurationAndRotation(t *testing.T) {
	t.Setenv(deployCredentialKeyEnv, strings.Repeat("12", 32))
	ticket := taskPreviewTicket{TaskID: 1, Actor: "1", Revision: "revision", Expires: time.Now().Add(time.Minute).Unix()}
	first, err := signPreviewTicket(ticket)
	if err != nil {
		t.Fatal(err)
	}
	second, err := signPreviewTicket(ticket)
	if err != nil || first != second {
		t.Fatal("shared secret did not yield a stable signature")
	}
	t.Setenv(deployCredentialKeyEnv, strings.Repeat("34", 32))
	if _, err := parsePreviewTicket(first); !errors.Is(err, ErrTaskPreviewConflict) {
		t.Fatalf("rotation accepted old signature: %v", err)
	}
	for _, key := range []string{"", "malformed"} {
		t.Setenv(deployCredentialKeyEnv, key)
		if _, err := signPreviewTicket(ticket); err == nil {
			t.Fatal("missing/invalid config accepted")
		}
	}
}
func TestPreviewCorruptSnapshotCannotDowngradeEnforcement(t *testing.T) {
	db, svc, task, _ := previewFixture(t, false)
	if err := db.Model(&DeployTask{}).Where("id = ?", task.ID).UpdateColumn("target_snapshot", "[]").Error; err != nil {
		t.Fatal(err)
	}
	if _, err := svc.StartTask(task.ID, StartTaskRequest{}, "1", nil); !errors.Is(err, ErrTaskPreviewConflict) {
		t.Fatalf("corrupt snapshot bypassed gate: %v", err)
	}
}

func TestSelectedUpdateCannotMoveInvisibleTaskIntoVisibleScope(t *testing.T) {
	db, svc, task, hostID := previewFixture(t, true)
	// The replacement target is visible, while the existing task target is not.
	visible := cmdbhost.Host{Hostname: "visible-replacement", IP: "10.40.9.3", BusinessScopeID: task.BusinessScopeID, Status: "assigned", StateVersion: 1, DeptID: 9}
	visible.CreatedBy = "2"
	if err := db.Create(&visible).Error; err != nil {
		t.Fatal(err)
	}
	scope := &common.DataScopeReq{Mode: common.DataScopeModeDept, DeptID: 9}
	selection := cmdb.TargetSelection{BusinessScopeID: task.BusinessScopeID, HostIDs: []uint64{visible.ID}}
	// Scope itself must be visible so the regression reaches task visibility.
	if err := db.Model(&deployTestBizScope{}).Where("id = ?", task.BusinessScopeID).Update("dept_id", 9).Error; err != nil {
		t.Fatal(err)
	}
	resolved, err := svc.ResolveTargetSelection(context.Background(), selection, scope)
	if err != nil {
		t.Fatal(err)
	}
	before := DeployTask{}
	if err := db.First(&before, task.ID).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := svc.UpdateTask(task.ID, UpdateTaskRequest{TargetSelection: &selection, ScopeVersion: resolved.ScopeVersion}, "2", scope); err == nil {
		t.Fatal("invisible task became editable through visible replacement")
	}
	after := DeployTask{}
	if err := db.First(&after, task.ID).Error; err != nil {
		t.Fatal(err)
	}
	if string(before.TargetSnapshot) != string(after.TargetSnapshot) || string(before.TargetIDs) != string(after.TargetIDs) {
		t.Fatal("denied update mutated provenance/targets")
	}
	if after.TargetIDs == nil || hostID == visible.ID {
		t.Fatal("invalid fixture")
	}
}
func TestCanceledSelectionRequestsDoNotMutate(t *testing.T) {
	db, svc, task, hostID := previewFixture(t, true)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	selection := cmdb.TargetSelection{BusinessScopeID: task.BusinessScopeID, HostIDs: []uint64{hostID}}
	resolved, err := svc.ResolveTargetSelection(context.Background(), selection, nil)
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name string
		call func() error
	}{
		{"create", func() error {
			_, err := svc.CreateTaskContext(ctx, CreateTaskRequest{Name: "canceled-create", PackageID: task.PackageID, TargetSelection: &selection, ScopeVersion: resolved.ScopeVersion}, "1", nil)
			return err
		}},
		{"update", func() error {
			_, err := svc.UpdateTaskContext(ctx, task.ID, UpdateTaskRequest{TargetSelection: &selection, ScopeVersion: resolved.ScopeVersion}, "1", nil)
			return err
		}},
		{"preview", func() error { _, err := svc.PreviewTaskContext(ctx, task.ID, StartTaskRequest{}, "1", nil); return err }},
		{"start", func() error {
			_, err := svc.StartTaskContext(ctx, task.ID, StartTaskRequest{RequirePreview: true}, "1", nil)
			return err
		}},
	}
	for _, c := range cases {
		if err := c.call(); !errors.Is(err, context.Canceled) {
			t.Fatalf("%s ignored cancellation: %v", c.name, err)
		}
	}
	var count int64
	if err := db.Model(&DeployTask{}).Where("name = ?", "canceled-create").Count(&count).Error; err != nil || count != 0 {
		t.Fatal("canceled create persisted")
	}
	var current DeployTask
	if err := db.First(&current, task.ID).Error; err != nil || current.Status != TaskStatusDraft {
		t.Fatal("canceled start changed draft")
	}
}

func TestPreviewFrozenPlanIgnoresLiveDefinitionsAndRejectsSnapshotDrift(t *testing.T) {
	db, svc, initial, hostID := previewFixture(t, false)
	template, err := svc.CreateTemplate(CreateTemplateRequest{Name: "frozen-template", Version: "1.0.0", PackageID: initial.PackageID, Status: TemplateStatusEnabled}, "1")
	if err != nil {
		t.Fatal(err)
	}
	task, err := svc.CreateTask(CreateTaskRequest{Name: "frozen-plan-task", TemplateID: template.ID, BusinessScopeID: initial.BusinessScopeID, TargetType: TargetTypeHost, TargetIDs: []uint64{hostID}, ExecutorType: ExecutorTypeManual, TemplateParams: map[string]any{"serviceName": "frozen-service", "port": "8080", "custom": "private-freeform", "rootPassword": "{{secret.password}}"}, SecretParams: map[string]string{"password": "super-private-value"}}, "1", nil)
	if err != nil {
		t.Fatal(err)
	}
	preview, err := svc.PreviewTask(task.ID, StartTaskRequest{}, "1", nil)
	if err != nil {
		t.Fatal(err)
	}
	if preview.Plan.TemplateName != "frozen-template" || preview.Plan.TemplateVersion != "1.0.0" || preview.Plan.Steps[0].PackageVersion != "2.0.0" {
		t.Fatalf("wrong frozen identities: %+v", preview.Plan)
	}
	values := map[string]TaskPreviewParameter{}
	for _, p := range preview.Plan.Steps[0].Parameters {
		values[p.Key] = p
	}
	if values["serviceName"].Value != "frozen-service" || values["port"].Value != "8080" || !values["rootPassword"].Hidden || !values["custom"].Hidden {
		t.Fatalf("unsafe parameters: %+v", values)
	}
	raw, _ := json.Marshal(preview)
	for _, private := range []string{"private-freeform", "super-private-value", "echo install", "{{secret.password}}"} {
		if strings.Contains(string(raw), private) {
			t.Fatalf("leaked %s", private)
		}
	}
	if err := db.Model(&DeployTemplate{}).Where("id = ?", template.ID).UpdateColumn("version", "9.0.0").Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&DeployPackage{}).Where("id = ?", initial.PackageID).Updates(map[string]any{"version": "99.0.0", "install_command": "echo changed-live"}).Error; err != nil {
		t.Fatal(err)
	}
	again, err := svc.PreviewTask(task.ID, StartTaskRequest{}, "1", nil)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(preview.Plan, again.Plan) || preview.CurrentVersion != again.CurrentVersion || preview.TargetVersion != again.TargetVersion {
		t.Fatal("live definition changed frozen review")
	}
	var stored DeployTask
	if err := db.First(&stored, task.ID).Error; err != nil {
		t.Fatal(err)
	}
	snapshot, err := decodeExecutionSnapshot(stored.ExecutionSnapshot)
	if err != nil {
		t.Fatal(err)
	}
	snapshot.Steps[0].TemplateParams["port"] = "9090"
	changed, _ := json.Marshal(snapshot)
	if err := db.Model(&stored).UpdateColumn("execution_snapshot", string(changed)).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := svc.StartTask(task.ID, StartTaskRequest{RequirePreview: true, IntentVersion: preview.IntentVersion}, "1", nil); !errors.Is(err, ErrTaskPreviewConflict) {
		t.Fatalf("changed frozen plan started: %v", err)
	}
	if err := db.First(&stored, task.ID).Error; err != nil {
		t.Fatal(err)
	}
	if stored.Status != TaskStatusDraft {
		t.Fatal("conflict mutated state")
	}
	fresh, err := svc.PreviewTask(task.ID, StartTaskRequest{}, "1", nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.StartTask(task.ID, StartTaskRequest{RequirePreview: true, IntentVersion: fresh.IntentVersion}, "1", nil); err != nil {
		t.Fatalf("fresh frozen review rejected: %v", err)
	}
}

func TestPreviewPlanRedactionAndBudgets(t *testing.T) {
	snapshot := deployExecutionSnapshot{TemplateName: "template", Steps: []deployStepSnapshot{{StepName: "safe", Package: deployPackageSnapshot{Name: "package", Version: "1.0.0", InstallCommand: "command-private", SourceURL: "url-private"}, StepConfig: map[string]any{"command": "config-private"}, Condition: map[string]any{"value": "condition-private"}, TemplateParams: map[string]any{
		"port": "8080", "dataRoot": "/data/private-value", "serviceName": "{{secret.name}}", "version": strings.Repeat("v", 129), "secret.password": "secret-private", "rootPassword": "password-private", "custom": map[string]any{"password": "nested-private"}, "script": "script-private", "line\nprivate": "key-private",
	}}}}
	plan := projectTaskPreviewPlan(snapshot, map[string]string{"name": "private-value"}, true)
	raw, _ := json.Marshal(plan)
	for _, value := range []string{"private-value", "{{secret.name}}", "secret-private", "password-private", "nested-private", "script-private", "command-private", "url-private", "config-private", "condition-private", "key-private", "line\nprivate"} {
		if strings.Contains(string(raw), value) {
			t.Fatalf("preview leaked %q", value)
		}
	}
	if len(plan.Steps[0].Parameters) != 9 {
		t.Fatal("hidden parameters missing")
	}
	for _, p := range plan.Steps[0].Parameters {
		if p.Key == "port" && (p.Hidden || p.Value != "8080") {
			t.Fatal("public port missing")
		}
	}
	incomplete := projectTaskPreviewPlan(snapshot, nil, false)
	for _, p := range incomplete.Steps[0].Parameters {
		if !p.Hidden || p.Value != "" {
			t.Fatal("incomplete secrets allowed a value")
		}
	}
	snapshot.Steps = make([]deployStepSnapshot, 101)
	for i := range snapshot.Steps {
		snapshot.Steps[i] = deployStepSnapshot{StepName: "step", TemplateParams: map[string]any{fmt.Sprintf("param-%03d", i): "hidden", "port": "8080", "serviceName": "service"}}
	}
	bounded := projectTaskPreviewPlan(snapshot, nil, true)
	if !bounded.Truncated || bounded.StepCount != 101 || len(bounded.Steps) != 100 || bounded.ParameterCount != 303 {
		t.Fatalf("wrong budget: %+v", bounded)
	}
	count := 0
	for _, step := range bounded.Steps {
		count += len(step.Parameters)
	}
	if count != 200 {
		t.Fatalf("parameter cap violated: %d", count)
	}
	first, _ := json.Marshal(bounded)
	second, _ := json.Marshal(projectTaskPreviewPlan(snapshot, nil, true))
	if string(first) != string(second) {
		t.Fatal("unstable parameter order")
	}
}

func TestPreviewUnreadableSecretHidesAllNewValues(t *testing.T) {
	db, svc, task, _ := previewFixture(t, false)
	if err := db.Create(&DeployTaskSecret{TaskID: task.ID, SecretKey: "password", SecretEncrypted: "unreadable"}).Error; err != nil {
		t.Fatal(err)
	}
	preview, err := svc.PreviewTask(task.ID, StartTaskRequest{}, "1", nil)
	if err != nil {
		t.Fatal(err)
	}
	if preview.Plan.Steps[0].Name != "" || preview.Plan.Steps[0].PackageName != "" {
		t.Fatal("incomplete secret set exposed new strings")
	}
}

func TestPreviewLegacyDuplicateSecretHidesNewValues(t *testing.T) {
	db, svc, task, _ := previewFixture(t, false)
	// Only this test-owned schema is changed to emulate an incompatible legacy table.
	if err := db.Migrator().DropIndex(&DeployTaskSecret{}, "uk_deploy_task_secret"); err != nil {
		t.Fatal(err)
	}
	first, err := encryptDeployCredential("first-private")
	if err != nil {
		t.Fatal(err)
	}
	second, err := encryptDeployCredential("second-private")
	if err != nil {
		t.Fatal(err)
	}
	rows := []DeployTaskSecret{{TaskID: task.ID, SecretKey: "password", SecretEncrypted: first}, {TaskID: task.ID, SecretKey: "password", SecretEncrypted: second}}
	if err := db.Create(&rows).Error; err != nil {
		t.Fatal(err)
	}
	_, complete, err := svc.loadDeployTaskSecretMaterial(task.ID)
	if err != nil || complete {
		t.Fatalf("legacy duplicates certified complete: %v", err)
	}
	preview, err := svc.PreviewTask(task.ID, StartTaskRequest{}, "1", nil)
	if err != nil {
		t.Fatal(err)
	}
	if preview.Plan.Steps[0].Name != "" || preview.Plan.Steps[0].PackageVersion != "" {
		t.Fatal("legacy duplicate secrets exposed new values")
	}
}
