package deploy

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
	"strings"

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
		if err := tx.Where(taskIDWhereClause, taskID).Delete(&DeployTaskSecret{}).Error; err != nil {
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
	secrets, _, err := s.loadDeployTaskSecretMaterial(taskID)
	return secrets, err
}

// loadDeployTaskSecretMaterial derives completeness from the same row set as
// decryption. Preview must hide values if legacy duplicates or unreadable rows
// prevent reliable redaction; execution retains its existing skip behavior.
func (s *DeployService) loadDeployTaskSecretMaterial(taskID uint64) (map[string]string, bool, error) {
	ctx, cancel := context.WithTimeout(s.requestContext(), s.ownershipTiming().operation)
	defer cancel()
	var rows []DeployTaskSecret
	if err := s.db.WithContext(ctx).Where(taskIDWhereClause, taskID).Find(&rows).Error; err != nil {
		if s.ownership != nil {
			s.ownership.fail(err)
		}
		return nil, false, err
	}
	secrets := make(map[string]string, len(rows))
	complete := true
	for _, row := range rows {
		decrypted, err := decryptDeployCredential(row.SecretEncrypted)
		if err != nil {
			complete = false
			continue
		}
		if _, duplicate := secrets[row.SecretKey]; duplicate {
			complete = false
		}
		secrets[row.SecretKey] = decrypted
	}
	return secrets, complete, nil
}

// deployTaskSecretPresence reports which secret keys exist for a task without
// exposing any value.
func (s *DeployService) deployTaskSecretPresence(taskID uint64) []SecretPresenceItem {
	var rows []DeployTaskSecret
	if err := s.db.Select("secret_key").Where(taskIDWhereClause, taskID).Find(&rows).Error; err != nil {
		return []SecretPresenceItem{}
	}
	items := make([]SecretPresenceItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, SecretPresenceItem{Key: row.SecretKey, Present: true})
	}
	return items
}

// RollbackTask creates a scoped draft from a supported frozen rollback plan.
// Trusted worker callers may omit scope; HTTP callers must supply it.
func (s *DeployService) RollbackTask(ctx context.Context, taskID uint64, userID string, scopes ...*common.DataScopeReq) (*TaskResponse, error) {
	var scope *common.DataScopeReq
	if len(scopes) > 0 {
		scope = scopes[0]
	}
	return s.createFrozenRollback(ctx, taskID, userID, scope)
}

// GetRollbackTask authorizes the source and the derived task independently.
func (s *DeployService) GetRollbackTask(ctx context.Context, taskID uint64, scopes ...*common.DataScopeReq) (*TaskResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	var scope *common.DataScopeReq
	if len(scopes) > 0 {
		scope = scopes[0]
	}
	inner := s.targetRequestService(ctx)
	if _, err := inner.loadVisibleTask(taskID, scope); err != nil {
		return nil, err
	}
	var rollback DeployTask
	if err := inner.db.Where("rollback_task_id = ?", taskID).Order("id DESC").First(&rollback).Error; err != nil {
		return nil, err
	}
	return inner.GetTask(rollback.ID, scope)
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
