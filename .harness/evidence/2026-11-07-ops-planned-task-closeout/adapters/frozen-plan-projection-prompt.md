Independent CODE review. Read-only, no tools. Supplied source only. Report proved P0/P1/P2 defects and minimal fix; otherwise APPROVE. Max 180 words. State scope limits. No release, production or project-completion approval. Current/frozen version binding was previously tested; do not infer missing helpers are defective.

FILE pantheon-ops\backend\modules\business\deploy\task_preview_plan.go
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
taskPreviewStepLimit = 100
taskPreviewParameterLimit = 200
taskPreviewValueLimit = 128
)

// TaskPreviewPlan projects frozen intent, not a measured current configuration.
// Execution definitions and arbitrary configuration values never enter this DTO.
type TaskPreviewPlan struct {
TemplateID uint64 `json:"templateId"`
TemplateName string `json:"templateName"`
TemplateVersion string `json:"templateVersion"`
Steps []TaskPreviewPlanStep `json:"steps"`
StepCount int `json:"stepCount"`
ParameterCount int `json:"parameterCount"`
Truncated bool `json:"truncated"`
}

type TaskPreviewPlanStep struct {
ID string `json:"id"`
Name string `json:"name"`
Type string `json:"type"`
Action string `json:"action"`
PackageName string `json:"packageName"`
PackageVersion string `json:"packageVersion"`
FailurePolicy string `json:"failurePolicy"`
ContinueOnHostFailure bool `json:"continueOnHostFailure"`
Parameters []TaskPreviewParameter `json:"parameters"`
}

type TaskPreviewParameter struct {
Key string `json:"key"`
Value string `json:"value,omitempty"`
Hidden bool `json:"hidden"`
}

func (s *DeployService) taskPreviewPlan(task *DeployTask) (*TaskPreviewPlan, error) {
snapshot, err := decodeExecutionSnapshot(task.ExecutionSnapshot)
if err != nil {
return nil, err
}
secrets, err := s.loadDeployTaskSecrets(task.ID)
if err != nil {
return nil, err
}
// The execution loader skips unreadable ciphertext. For display, an incomplete
// secret set means no string value can be certified safe to return.
var secretCount int64
if err := s.db.Model(&DeployTaskSecret{}).Where(taskIDWhereClause, task.ID).Count(&secretCount).Error; err != nil {
return nil, err
}
return projectTaskPreviewPlan(snapshot, secrets, secretCount == int64(len(secrets))), nil
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

PUBLIC HELPER CONTRACT: taskPreviewPublicText returns empty if incomplete, >128 bytes, controls, a secret sentinel or any known secret occurrence; otherwise unchanged. taskPreviewParameterValue returns empty except public fixed-renderer structural fields with constrained grammar; prior excerpt review APPROVE.

FILE pantheon-ops\backend\modules\business\deploy\deploy_rollback.go
func (s *DeployService) loadDeployTaskSecrets(taskID uint64) (map[string]string, error) {
var rows []DeployTaskSecret
if err := s.db.Where(taskIDWhereClause, taskID).Find(&rows).Error; err != nil {
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
