package deploy

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
)

// componentPlan is the rendered execution plan for a fixed component step. The
// install/uninstall scripts are generated from the built-in fixed templates;
// health check commands are probes that determine whether the component is
// already in the desired state (idempotency guard).
type componentPlan struct {
	InstallScript               string
	UninstallScript             string
	HealthCheckCommand          string
	UninstallHealthCheckCommand string
}

// resolveComponentPlan renders the fixed-template install/uninstall scripts and
// idempotency probes for a component step. The host argument is used to select
// the platform script variant when the template declares one.
func (s *DeployService) resolveComponentPlan(step deployExecutionStep, task DeployTask, host cmdbHostSnapshot) (componentPlan, error) {
	templateCode, ok := componentTypeToTemplateCode[strings.TrimSpace(step.StepType)]
	if !ok {
		return componentPlan{}, errors.New(errDeployTaskTemplateInvalid)
	}
	// Build the render view with secrets resolved in-memory (never persisted).
	params := resolveStepParamsSecretSentinels(step.TemplateParams, s.renderSecrets(task.ID))
	params = cloneDeployConfig(params)

	pkg := step.Package
	pkg.TemplateCode = templateCode

	installScript, err := s.renderComponentScript(pkg, task, step, params, TaskActionInstall)
	if err != nil {
		return componentPlan{}, err
	}
	uninstallScript, err := s.renderComponentScript(pkg, task, step, params, TaskActionUninstall)
	if err != nil {
		return componentPlan{}, err
	}
	return componentPlan{
		InstallScript:               installScript,
		UninstallScript:             uninstallScript,
		HealthCheckCommand:          buildComponentHealthCheck(templateCode, params),
		UninstallHealthCheckCommand: buildComponentUninstallHealthCheck(templateCode, params),
	}, nil
}

// renderComponentScript renders a fixed template script with the given action
// applied to the parameter view.
func (s *DeployService) renderComponentScript(pkg DeployPackage, task DeployTask, step deployExecutionStep, params map[string]any, action string) (string, error) {
	renderTask := task
	renderTask.Action = action
	renderParams := cloneDeployConfig(params)
	renderParams["action"] = action
	paramsJSON, _ := json.Marshal(renderParams)
	renderTask.TemplateParams = paramsJSON
	return renderFixedTemplateScript(pkg, renderTask)
}

// buildComponentHealthCheck builds the idempotency probe for a component type.
// A probe that exits zero means the component is already in the desired state.
func buildComponentHealthCheck(templateCode string, params map[string]any) string {
	serviceName := strings.TrimSpace(anyToString(params["serviceName"]))
	installRoot := strings.TrimSpace(anyToString(params["installRoot"]))
	port := strings.TrimSpace(anyToString(params["port"]))
	webPort := strings.TrimSpace(anyToString(params["webPort"]))
	apiPort := strings.TrimSpace(anyToString(params["apiPort"]))
	httpPort := strings.TrimSpace(anyToString(params["httpPort"]))

	switch templateCode {
	case TemplateCodeNginxSystemd:
		if serviceName == "" {
			serviceName = defaultNginxServiceName
		}
		return fmt.Sprintf("systemctl is-active --quiet %s", serviceName)
	case TemplateCodeRedisSystemd:
		if port == "" {
			port = "6379"
		}
		return fmt.Sprintf("redis-cli -p %s ping | grep -q PONG", port)
	case TemplateCodeJDKSystemd:
		if installRoot == "" {
			installRoot = "/data/jdk"
		}
		return fmt.Sprintf("test -x %s/bin/java && %s/bin/java -version", installRoot, installRoot)
	case TemplateCodeMySQLSystemd:
		if port == "" {
			port = "3306"
		}
		return fmt.Sprintf("mysqladmin --protocol=tcp -h 127.0.0.1 -P %s ping >/dev/null 2>&1", port)
	case TemplateCodeMinIOSystemd:
		if apiPort == "" {
			apiPort = "9000"
		}
		return fmt.Sprintf("curl -fsS http://127.0.0.1:%s/minio/health/live >/dev/null 2>&1", apiPort)
	case TemplateCodeHarborOffline:
		if httpPort == "" {
			httpPort = "80"
		}
		return fmt.Sprintf("curl -fsS http://127.0.0.1:%s/api/v2.0/health >/dev/null 2>&1", httpPort)
	case TemplateCodePrometheusSystemd:
		if webPort == "" {
			webPort = "9090"
		}
		return fmt.Sprintf("curl -fsS http://127.0.0.1:%s/-/healthy >/dev/null 2>&1", webPort)
	default:
		return ""
	}
}

// buildComponentUninstallHealthCheck builds the probe that determines whether
// an uninstall has already completed (i.e. the service/process is gone).
func buildComponentUninstallHealthCheck(templateCode string, params map[string]any) string {
	serviceName := strings.TrimSpace(anyToString(params["serviceName"]))
	switch templateCode {
	case TemplateCodeNginxSystemd, TemplateCodeHarborOffline:
		if serviceName == "" {
			serviceName = defaultNginxServiceName
		}
		return fmt.Sprintf("! systemctl is-active --quiet %s", serviceName)
	case TemplateCodeRedisSystemd, TemplateCodeMySQLSystemd, TemplateCodeMinIOSystemd, TemplateCodePrometheusSystemd:
		if serviceName == "" {
			return "true"
		}
		return fmt.Sprintf("! systemctl is-active --quiet %s", serviceName)
	default:
		return "true"
	}
}

// persistDeployTaskSecrets encrypts and stores the write-only secret params of
// a task. Existing secrets for the task are replaced atomically.
func (s *DeployService) persistDeployTaskSecrets(taskID uint64, secretParams map[string]string) error {
	if taskID == 0 {
		return errors.New(errDeployTaskSecretParamInvalid)
	}
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("task_id = ?", taskID).Delete(&DeployTaskSecret{}).Error; err != nil {
			return err
		}
		for key, value := range secretParams {
			if strings.TrimSpace(key) == "" || strings.TrimSpace(value) == "" {
				return errors.New(errDeployTaskSecretParamInvalid)
			}
			encrypted, err := encryptDeployCredential(value)
			if err != nil {
				return err
			}
			if err := tx.Create(&DeployTaskSecret{
				TaskID:          taskID,
				SecretKey:       strings.TrimSpace(key),
				SecretEncrypted: encrypted,
			}).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// loadDeployTaskSecrets loads and decrypts the secret params for a task.
func (s *DeployService) loadDeployTaskSecrets(taskID uint64) (map[string]string, error) {
	var rows []DeployTaskSecret
	if err := s.db.Where("task_id = ?", taskID).Find(&rows).Error; err != nil {
		return nil, err
	}
	secrets := make(map[string]string, len(rows))
	for _, row := range rows {
		decrypted, err := decryptDeployCredential(row.SecretEncrypted)
		if err != nil {
			continue // skip unreadable rows instead of failing the whole task
		}
		secrets[row.SecretKey] = decrypted
	}
	return secrets, nil
}

// deployTaskSecretPresence reports which secret keys exist for a task without
// exposing any value.
func (s *DeployService) deployTaskSecretPresence(taskID uint64) []SecretPresenceItem {
	var rows []DeployTaskSecret
	if err := s.db.Select("secret_key").Where("task_id = ?", taskID).Find(&rows).Error; err != nil {
		return []SecretPresenceItem{}
	}
	items := make([]SecretPresenceItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, SecretPresenceItem{Key: row.SecretKey, Present: true})
	}
	return items
}

// RollbackTask creates a rollback task from a completed deploy task. The
// rollback action is derived from the original action (install -> uninstall,
// start -> stop). The returned task is in draft state and must be started
// explicitly.
func (s *DeployService) RollbackTask(ctx context.Context, taskID uint64, userID string) (*TaskResponse, error) {
	var original DeployTask
	if err := s.db.First(&original, taskID).Error; err != nil {
		return nil, err
	}
	rollbackAction := DetermineRollbackAction(original.Action)
	if rollbackAction == "" {
		return nil, errors.New("business.deploy.task.rollbackNotSupported")
	}
	if original.Status != TaskStatusSuccess && original.Status != TaskStatusFailed {
		return nil, errors.New("business.deploy.task.rollbackInvalidState")
	}

	// Reuse the frozen snapshot: create a new task that replays the same
	// plan with the inverse action.
	req := CreateTaskRequest{
		Name:              fmt.Sprintf("%s-rollback", strings.TrimSuffix(original.Name, "-rollback")),
		TemplateID:        original.TemplateID,
		PackageID:         original.PackageID,
		BusinessScopeID:   original.BusinessScopeID,
		ServiceID:         original.ServiceID,
		ServiceInstanceID: original.ServiceInstanceID,
		Action:            rollbackAction,
		TargetType:        TargetTypeHost,
		TargetIDs:         parseUint64JSON(original.TargetIDs),
		ExecutorType:      original.ExecutorType,
		TemplateParams:    decodeJSONMap(original.TemplateParams),
		Remark:            fmt.Sprintf("rollback of task %d", taskID),
	}
	rollbackTask, err := s.CreateTask(req, userID, nil)
	if err != nil {
		return nil, err
	}
	// Link the rollback chain.
	if err := s.db.Model(&DeployTask{}).Where("id = ?", rollbackTask.ID).Updates(map[string]any{
		"rollback_task_id": original.ID,
		"updated_at":       time.Now(),
		"updated_by":       userID,
	}).Error; err != nil {
		return nil, err
	}
	return rollbackTask, nil
}

// GetRollbackTask returns the rollback task linked to the given task, or
// gorm.ErrRecordNotFound when no rollback exists.
func (s *DeployService) GetRollbackTask(ctx context.Context, taskID uint64) (*TaskResponse, error) {
	var rollback DeployTask
	err := s.db.Where("rollback_task_id = ?", taskID).First(&rollback).Error
	if err != nil {
		return nil, err
	}
	return s.GetTask(rollback.ID, nil)
}

// TriggerAutoRollback creates and immediately starts a rollback task for a
// failed task when auto-rollback is enabled.
func (s *DeployService) TriggerAutoRollback(ctx context.Context, task *DeployTask) error {
	if task == nil {
		return errors.New("business.deploy.task.notFound")
	}
	rollbackTask, err := s.RollbackTask(ctx, task.ID, "system")
	if err != nil {
		return err
	}
	_, err = s.StartTask(rollbackTask.ID, StartTaskRequest{}, "system", nil)
	return err
}
