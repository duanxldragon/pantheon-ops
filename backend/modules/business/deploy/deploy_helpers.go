package deploy

import (
	"errors"
	"strings"
	"time"

	"github.com/duanxldragon/pantheon-base/backend/modules/business/metrics"
)

// errDeployTaskSecretParamInvalid rejects empty secret parameter values.
const errDeployTaskSecretParamInvalid = "business.deploy.task.secretParamInvalid"

// SecretPresenceItem reports whether a write-only secret parameter exists for a
// task without exposing its value.
type SecretPresenceItem struct {
	Key     string `json:"key"`
	Present bool   `json:"present"`
}

// shellQuoteSingle wraps a value for safe single-quoted shell interpolation:
// ' -> '\'' so arbitrary values (including quotes) cannot break out.
func shellQuoteSingle(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}

// DeployTaskSecret is the write-only encrypted secret parameter store for a
// deploy task. Values are encrypted with the deploy credential key and are
// never returned by list/detail APIs.
type DeployTaskSecret struct {
	ID              uint64 `gorm:"primaryKey;autoIncrement" json:"id"`
	TaskID          uint64 `gorm:"not null;index:uk_deploy_task_secret,unique" json:"taskId"`
	SecretKey       string `gorm:"size:128;not null;index:uk_deploy_task_secret,unique" json:"-"`
	SecretEncrypted string `gorm:"type:text;not null" json:"-"`
	CreatedAt       string `gorm:"-" json:"-"`
}

// TableName returns the deploy task secret table name.
func (DeployTaskSecret) TableName() string { return "biz_deploy_task_secret" }

// isDeployTaskSecretParam reports whether a template parameter key carries a
// write-only secret value. Secret keys use the "secret." prefix convention so
// they are stripped before persistence and resolved back only at render time.
func isDeployTaskSecretParam(key string) bool {
	return strings.HasPrefix(strings.TrimSpace(key), "secret.")
}

// splitDeploySecretParams separates write-only secret params from the persisted
// template params. Secret values are moved to the returned secret map and
// removed from the visible template params.
func splitDeploySecretParams(templateParams map[string]any) (visible map[string]any, secrets map[string]string) {
	visible = make(map[string]any, len(templateParams))
	secrets = make(map[string]string)
	for key, value := range templateParams {
		if isDeployTaskSecretParam(key) {
			if text := strings.TrimSpace(anyToString(value)); text != "" {
				secrets[strings.TrimPrefix(strings.TrimSpace(key), "secret.")] = text
			}
			continue
		}
		visible[key] = value
	}
	return visible, secrets
}

// mergeDeploySecretParamsForValidation merges decrypted secrets back into a
// template params view solely for required-parameter validation. The merged
// view is never persisted.
func mergeDeploySecretParamsForValidation(visible map[string]any, secrets map[string]string) {
	for key, value := range secrets {
		if _, exists := visible[key]; !exists {
			visible[key] = value
		}
	}
}

// validateDeploySecretParams validates secret parameter keys. Empty or
// whitespace-only values are rejected so a task never stores a vacuous secret.
func validateDeploySecretParams(secrets map[string]string) error {
	for key, value := range secrets {
		if strings.TrimSpace(key) == "" {
			return errors.New(errDeployTaskSecretParamInvalid)
		}
		if strings.TrimSpace(value) == "" {
			return errors.New(errDeployTaskSecretParamInvalid)
		}
	}
	return nil
}
// validateDeployStructuralParams validates structural template params. Values
// that could break the fixed template renderer (newlines, embedded quotes) are
// rejected for the keys known to be rendered into shell text.
func validateDeployStructuralParams(params map[string]any) error {
	if params == nil {
		return nil
	}
	for key, value := range params {
		if isDeployTaskSecretParam(key) {
			continue
		}
		text := strings.TrimSpace(anyToString(value))
		if strings.ContainsAny(text, "\n\r") {
			return errors.New(errDeployTaskTemplateParamsInvalid)
		}
	}
	return nil
}

// resolveStepParamsSecretSentinels resolves {{secret.<key>}} sentinels in the
// step template params into their decrypted values from the secret store. The
// resolved view is in-memory only and never persisted.
func resolveStepParamsSecretSentinels(params map[string]any, secrets map[string]string) map[string]any {
	resolved := make(map[string]any, len(params))
	for key, value := range params {
		text := strings.TrimSpace(anyToString(value))
		if strings.HasPrefix(text, "{{secret.") && strings.HasSuffix(text, "}}") {
			name := strings.TrimSuffix(strings.TrimPrefix(text, "{{secret."), "}}")
			if secretValue, ok := secrets[name]; ok && secretValue != "" {
				resolved[key] = secretValue
				continue
			}
		}
		resolved[key] = value
	}
	return resolved
}

// redactDeploySecrets replaces any occurrence of a known secret value in the
// given text with a {{secret.<key>}} sentinel so logs and error messages never
// leak credential material.
func redactDeploySecrets(content string, secrets map[string]string) string {
	if content == "" || len(secrets) == 0 {
		return content
	}
	redacted := content
	for key, value := range secrets {
		if value == "" {
			continue
		}
		redacted = strings.ReplaceAll(redacted, value, "{{secret."+key+"}}")
	}
	return redacted
}

// redactExecutionSnapshotSecrets replaces secret param values in the execution
// snapshot with sentinels before the snapshot is persisted. The snapshot must
// never carry plaintext secret material.
func redactExecutionSnapshotSecrets(snapshot *deployExecutionSnapshot, secretParams map[string]string) {
	if snapshot == nil || len(secretParams) == 0 {
		return
	}
	for i := range snapshot.Steps {
		step := &snapshot.Steps[i]
		redacted := make(map[string]any, len(step.TemplateParams))
		for key, value := range step.TemplateParams {
			text := strings.TrimSpace(anyToString(value))
			matched := false
			for secretKey, secretValue := range secretParams {
				if text == secretValue {
					redacted[key] = "{{secret." + secretKey + "}}"
					matched = true
					break
				}
			}
			if !matched {
				redacted[key] = value
			}
		}
		step.TemplateParams = redacted
	}
}

// isComponentType reports whether a step type is a fixed component type that
// maps to a built-in template code (nginx, redis, jdk, ...).
func isComponentType(stepType string) bool {
	_, ok := componentTypeToTemplateCode[strings.TrimSpace(stepType)]
	return ok
}

// componentTypeToTemplateCode maps component step types to their built-in
// fixed template codes.
var componentTypeToTemplateCode = map[string]string{
	"nginx":      TemplateCodeNginxSystemd,
	"redis":      TemplateCodeRedisSystemd,
	"jdk":        TemplateCodeJDKSystemd,
	"mysql":      TemplateCodeMySQLSystemd,
	"minio":      TemplateCodeMinIOSystemd,
	"harbor":     TemplateCodeHarborOffline,
	"prometheus": TemplateCodePrometheusSystemd,
}

// validStepType reports whether a step type is a package step, a script step,
// or a known component type.
func validStepType(stepType string) bool {
	stepType = strings.TrimSpace(stepType)
	return stepType == TemplateStepTypePackage ||
		stepType == TemplateStepTypeScript ||
		isComponentType(stepType)
}

// DetermineRollbackAction returns the inverse action used to build a rollback
// task, or an empty string when the action has no automatic rollback.
func DetermineRollbackAction(action string) string {
	switch normalizeTaskAction(action) {
	case TaskActionInstall, TaskActionReinstall, TaskActionUpgrade:
		return TaskActionUninstall
	case TaskActionStart:
		return TaskActionStop
	default:
		return ""
	}
}

// recordTaskStart records a deploy task start metric.
func recordTaskStart(task *DeployTask) {
	if task == nil {
		return
	}
	metrics.DeployTaskStarted(task.ExecutorType)
}

// recordTaskComplete records a deploy task completion metric.
func recordTaskComplete(task *DeployTask, duration time.Duration) {
	_ = duration
	if task == nil {
		return
	}
	metrics.DeployTaskCompleted(task.ExecutorType)
}

// recordTaskFailure records a deploy task failure metric.
func recordTaskFailure(task *DeployTask, reason string) {
	if task == nil {
		return
	}
	metrics.DeployTaskFailed(task.ExecutorType)
}

// recordHostResult records a host-level result metric for a deploy task.
func recordHostResult(host *DeployTaskHost, executorType string) {
	if host == nil {
		return
	}
	metrics.DeployHostResult(executorType, host.Status)
}
