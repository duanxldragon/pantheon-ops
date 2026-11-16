package deploy

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	cmdbhost "github.com/duanxldragon/pantheon-base/backend/modules/business/cmdb/host"
	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
	"gorm.io/gorm"
)

func rollbackFixture(t *testing.T) (*gorm.DB, *DeployService, *TaskResponse) {
	t.Helper()
	db, svc, initial, hostID := previewFixture(t, false)
	if err := db.Model(&cmdbhost.Host{}).Where("id = ?", hostID).Update("status", "online").Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&DeployPackage{}).Where("id = ?", initial.PackageID).Update("uninstall_command", "echo frozen-uninstall").Error; err != nil {
		t.Fatal(err)
	}
	template, err := svc.CreateTemplate(CreateTemplateRequest{Name: "rollback-frozen-template", Version: "1.0.0", PackageID: initial.PackageID, Status: TemplateStatusEnabled}, "1")
	if err != nil {
		t.Fatal(err)
	}
	params := map[string]any{"serviceName": "frozen-service", "rootPassword": "{{secret.password}}"}
	secrets := map[string]string{"password": "rollback-private-fixture"}
	task, err := svc.UpdateTask(initial.ID, UpdateTaskRequest{TemplateID: &template.ID, TemplateParams: &params, SecretParams: &secrets}, "1", nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&DeployTask{}).Where("id = ?", task.ID).Update("status", TaskStatusSuccess).Error; err != nil {
		t.Fatal(err)
	}
	return db, svc, task
}

func TestFrozenRollbackPreservesDefinitionAndOwnReview(t *testing.T) {
	db, svc, original := rollbackFixture(t)
	var before DeployTask
	if err := db.First(&before, original.ID).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&DeployTemplate{}).Where("id = ?", original.TemplateID).Updates(map[string]any{"name": "changed-template", "version": "99.0", "status": "disabled"}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&DeployPackage{}).Where("id = ?", original.PackageID).Updates(map[string]any{"version": "99.0", "uninstall_command": "echo changed-dangerous-command", "status": "disabled"}).Error; err != nil {
		t.Fatal(err)
	}
	rollback, err := svc.RollbackTask(context.Background(), original.ID, "1", nil)
	if err != nil {
		t.Fatal(err)
	}
	if rollback.Status != TaskStatusDraft || rollback.Action != TaskActionUninstall || rollback.SourceTaskID == nil || *rollback.SourceTaskID != original.ID {
		t.Fatalf("wrong draft: %+v", rollback)
	}
	var saved DeployTask
	if err := db.First(&saved, rollback.ID).Error; err != nil {
		t.Fatal(err)
	}
	plan, err := decodeExecutionSnapshot(saved.ExecutionSnapshot)
	if err != nil {
		t.Fatal(err)
	}
	if plan.TemplateName != "rollback-frozen-template" || plan.TemplateVersion != "1.0.0" || plan.Steps[0].Package.Version != "2.0.0" || plan.Steps[0].Package.UninstallCommand != "echo frozen-uninstall" || plan.Steps[0].Action != TaskActionUninstall {
		t.Fatalf("live definition replaced frozen plan: %+v", plan)
	}
	if strings.Contains(string(saved.ExecutionSnapshot), "rollback-private-fixture") {
		t.Fatal("plaintext in rollback snapshot")
	}
	material, complete, err := svc.loadDeployTaskSecretMaterial(rollback.ID)
	if err != nil || !complete || material["password"] != "rollback-private-fixture" {
		t.Fatalf("secret inheritance: %v", err)
	}
	preview, err := svc.PreviewTask(rollback.ID, StartTaskRequest{}, "1", nil)
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(preview)
	if strings.Contains(string(raw), "rollback-private-fixture") || strings.Contains(string(raw), "frozen-uninstall") {
		t.Fatal("private definition in preview DTO")
	}
	if _, err := svc.StartTask(rollback.ID, StartTaskRequest{RequirePreview: true}, "1", nil); !errors.Is(err, ErrTaskPreviewRequired) {
		t.Fatalf("rollback bypassed review: %v", err)
	}
	if _, err := svc.StartTask(rollback.ID, StartTaskRequest{RequirePreview: true, IntentVersion: preview.IntentVersion}, "1", nil); err != nil {
		t.Fatalf("reviewed rollback rejected: %v", err)
	}
	var after DeployTask
	if err := db.First(&after, original.ID).Error; err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(before, after) {
		t.Fatal("source task mutated")
	}
}

func TestFrozenRollbackRejectsScopeAndCancellationWithoutRows(t *testing.T) {
	db, svc, original := rollbackFixture(t)
	scope := &common.DataScopeReq{Mode: common.DataScopeModeSelf, UserID: 999}
	if _, err := svc.RollbackTask(context.Background(), original.ID, "999", scope); !errors.Is(err, errDeployTaskForbidden) {
		t.Fatalf("invisible rollback create: %v", err)
	}
	if _, err := svc.GetRollbackTask(context.Background(), original.ID, scope); !errors.Is(err, errDeployTaskForbidden) {
		t.Fatalf("invisible rollback read: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := svc.RollbackTask(ctx, original.ID, "1", nil); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation ignored: %v", err)
	}
	var count int64
	if err := db.Model(&DeployTask{}).Where("rollback_task_id = ?", original.ID).Count(&count).Error; err != nil || count != 0 {
		t.Fatalf("denied request created rows: %d %v", count, err)
	}
	rollback, err := svc.RollbackTask(context.Background(), original.ID, "1", nil)
	if err != nil {
		t.Fatal(err)
	}
	found, err := svc.GetRollbackTask(context.Background(), original.ID, nil)
	if err != nil || found.ID != rollback.ID {
		t.Fatalf("authorized rollback read failed: %v", err)
	}
	if _, err := svc.GetRollbackTask(context.Background(), original.ID, scope); !errors.Is(err, errDeployTaskForbidden) {
		t.Fatalf("created rollback became disclosed: %v", err)
	}
}

func TestFrozenRollbackRejectsUnsupportedAndUnreadablePlans(t *testing.T) {
	for _, scenario := range []string{"malformed", "multiple", "script", "conditional", "no-inverse", "unreadable-secret"} {
		t.Run(scenario, func(t *testing.T) {
			db, svc, task := rollbackFixture(t)
			var original DeployTask
			if err := db.First(&original, task.ID).Error; err != nil {
				t.Fatal(err)
			}
			plan, err := decodeExecutionSnapshot(original.ExecutionSnapshot)
			if err != nil {
				t.Fatal(err)
			}
			switch scenario {
			case "malformed":
				original.ExecutionSnapshot = []byte("[]")
			case "multiple":
				plan.Steps = append(plan.Steps, plan.Steps[0])
			case "script":
				plan.Steps[0].StepType = TemplateStepTypeScript
			case "conditional":
				plan.Steps[0].Condition = map[string]any{"type": "on_success"}
			case "no-inverse":
				plan.Steps[0].Package.UninstallCommand = ""
			case "unreadable-secret":
				if err := db.Model(&DeployTaskSecret{}).Where("task_id = ?", task.ID).Update("secret_encrypted", "invalid").Error; err != nil {
					t.Fatal(err)
				}
			}
			if scenario != "malformed" {
				original.ExecutionSnapshot, err = json.Marshal(plan)
				if err != nil {
					t.Fatal(err)
				}
			}
			if err := db.Model(&DeployTask{}).Where("id = ?", task.ID).Update("execution_snapshot", string(original.ExecutionSnapshot)).Error; err != nil {
				t.Fatal(err)
			}
			if _, err := svc.RollbackTask(context.Background(), task.ID, "1", nil); err == nil {
				t.Fatal("unsafe plan accepted")
			}
			var count int64
			if err := db.Model(&DeployTask{}).Where("rollback_task_id = ?", task.ID).Count(&count).Error; err != nil || count != 0 {
				t.Fatalf("failed create left rows: %d %v", count, err)
			}
		})
	}
}

func TestFrozenRollbackHandlersPreserveDataScope(t *testing.T) {
	db, svc, task := rollbackFixture(t)
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(common.DataScopeContextKey, &common.DataScopeReq{Mode: common.DataScopeModeSelf, UserID: 999})
	})
	handler := NewDeployHandler(svc)
	router.POST("/tasks/:id/rollback", handler.RollbackTask)
	router.GET("/tasks/:id/rollback", handler.GetRollbackInfo)
	for _, method := range []string{http.MethodPost, http.MethodGet} {
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, httptest.NewRequest(method, fmt.Sprintf("/tasks/%d/rollback", task.ID), nil))
		var body struct {
			Code int `json:"code"`
		}
		if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		if body.Code != common.CodeForbidden {
			t.Fatalf("%s ignored scope: %d %s", method, recorder.Code, recorder.Body.String())
		}
	}
	var count int64
	if err := db.Model(&DeployTask{}).Where("rollback_task_id = ?", task.ID).Count(&count).Error; err != nil || count != 0 {
		t.Fatalf("denied handler created rows: %d %v", count, err)
	}
}
