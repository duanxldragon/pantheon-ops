package deploy

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"
)

const (
	taskPreviewStepLimit      = 100
	taskPreviewParameterLimit = 200
	taskPreviewValueLimit     = 128
)

// TaskPreviewPlan projects frozen intent, not a measured current configuration.
// Execution definitions and arbitrary configuration values never enter this DTO.
type TaskPreviewPlan struct {
	TemplateID      uint64                `json:"templateId"`
	TemplateName    string                `json:"templateName"`
	TemplateVersion string                `json:"templateVersion"`
	Steps           []TaskPreviewPlanStep `json:"steps"`
	StepCount       int                   `json:"stepCount"`
	ParameterCount  int                   `json:"parameterCount"`
	Truncated       bool                  `json:"truncated"`
}

type TaskPreviewPlanStep struct {
	ID                    string                 `json:"id"`
	Name                  string                 `json:"name"`
	Type                  string                 `json:"type"`
	Action                string                 `json:"action"`
	PackageName           string                 `json:"packageName"`
	PackageVersion        string                 `json:"packageVersion"`
	FailurePolicy         string                 `json:"failurePolicy"`
	ContinueOnHostFailure bool                   `json:"continueOnHostFailure"`
	Parameters            []TaskPreviewParameter `json:"parameters"`
}

type TaskPreviewParameter struct {
	Key    string `json:"key"`
	Value  string `json:"value,omitempty"`
	Hidden bool   `json:"hidden"`
}

func (s *DeployService) taskPreviewPlan(task *DeployTask) (*TaskPreviewPlan, error) {
	snapshot, err := decodeExecutionSnapshot(task.ExecutionSnapshot)
	if err != nil {
		return nil, err
	}
	secrets, complete, err := s.loadDeployTaskSecretMaterial(task.ID)
	if err != nil {
		return nil, err
	}
	return projectTaskPreviewPlan(snapshot, secrets, complete), nil
}

func projectTaskPreviewPlan(snapshot deployExecutionSnapshot, secrets map[string]string, complete bool) *TaskPreviewPlan {
	public := func(value string) string { return taskPreviewPublicText(value, secrets, complete) }
	plan := &TaskPreviewPlan{TemplateID: snapshot.TemplateID, TemplateName: public(snapshot.TemplateName), TemplateVersion: public(snapshot.TemplateVersion), Steps: []TaskPreviewPlanStep{}, StepCount: len(snapshot.Steps)}
	remaining := taskPreviewParameterLimit
	for i, step := range snapshot.Steps {
		plan.ParameterCount += len(step.TemplateParams)
		if i >= taskPreviewStepLimit {
			plan.Truncated = true
			continue
		}
		item := TaskPreviewPlanStep{ID: fmt.Sprintf("step:%d", i+1), Name: public(step.StepName), Type: public(step.StepType), Action: public(step.Action), PackageName: public(step.Package.Name), PackageVersion: public(step.Package.Version), FailurePolicy: public(effectiveFailurePolicy(step.FailurePolicy, step.ContinueOnHostFailure)), ContinueOnHostFailure: step.ContinueOnHostFailure, Parameters: []TaskPreviewParameter{}}
		keys := make([]string, 0, len(step.TemplateParams))
		for key := range step.TemplateParams {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for index, key := range keys {
			if remaining == 0 {
				plan.Truncated = true
				break
			}
			label := public(key)
			if label == "" || !taskPreviewIdentifier.MatchString(label) {
				label = fmt.Sprintf("parameter:%d", index+1)
			}
			value := taskPreviewParameterValue(key, step.TemplateParams[key], secrets, complete)
			item.Parameters = append(item.Parameters, TaskPreviewParameter{Key: label, Value: value, Hidden: value == ""})
			remaining--
		}
		plan.Steps = append(plan.Steps, item)
	}
	return plan
}

var taskPreviewIdentifier = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.@+-]*$`)
var taskPreviewPath = regexp.MustCompile(`^/[A-Za-z0-9_./-]+$`)
var taskPreviewRetention = regexp.MustCompile(`^[0-9]+(ms|s|m|h|d|w|y|B|KB|MB|GB|TB)$`)

func taskPreviewPublicText(value string, secrets map[string]string, complete bool) string {
	if !complete || len(value) > taskPreviewValueLimit || strings.Contains(value, secretSentinelPrefix) || strings.IndexFunc(value, unicode.IsControl) >= 0 || redactDeploySecrets(value, secrets) != value {
		return ""
	}
	return value
}

// Minimal-complexity: only the existing fixed renderer's public structural
// fields are displayable. Freeform params need a future explicit public schema;
// checking only a secret-looking key would allow nested or mislabeled secrets.
func taskPreviewParameterValue(key string, value any, secrets map[string]string, complete bool) string {
	if isDeployTaskSecretParam(key) {
		return ""
	}
	switch value.(type) {
	case string, float64, bool:
	default:
		return ""
	}
	text := taskPreviewPublicText(anyToString(value), secrets, complete)
	if text == "" {
		return ""
	}
	switch key {
	case "port", "apiPort", "consolePort", "httpPort", "webPort":
		port, err := strconv.Atoi(text)
		if err == nil && port > 0 && port <= 65535 {
			return text
		}
	case "installRoot", "dataRoot":
		if taskPreviewPath.MatchString(text) {
			return text
		}
	case "serviceName", "hostname", "version":
		if taskPreviewIdentifier.MatchString(text) {
			return text
		}
	case "storageRetentionPolicy":
		if taskPreviewRetention.MatchString(text) {
			return text
		}
	case "action":
		if validTaskAction(text) {
			return text
		}
	}
	return ""
}
