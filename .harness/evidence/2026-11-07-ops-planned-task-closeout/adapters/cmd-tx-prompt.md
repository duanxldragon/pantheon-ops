Read-only source review; no tools. Report proved P1/P2 defects with minimal fix, otherwise APPROVE supplied scope only. Max 220 words. This business O2 repair requires public HTTP viewed task versions. List/Get add stateVersion from taskCommandVersion(task, grouped task-level MAX(sequence)) after visibility/paging. Task schema is unchanged (datetime(3)); SHA includes task status/facts and task-level event sequence; log/host/step sequences excluded. targetTransactionService constructs a new service with transaction-bound CMDB/BizScope readers, asyncExecution=false and empty worker maps. Original dispatcher cancellation is captured under task lock and invoked post-commit; resume/retry use original dispatcher post-commit. Tests on real MySQL cover state matrix, same-version concurrency, ABA, scope before validation, context cancellation, chunked HTTP missing/stale/version, event/host failures, post-commit cancel. Omitted engine/lease/dependency/permission rules are unchanged. UI API errors expose numeric code/status. Alerts persist until next explicit action; 409 refresh does not replay; failed batch selections retained. Review only supplied source, not whole repository.
package deploy

import (
"context"
"crypto/sha256"
"encoding/hex"
"encoding/json"
"errors"
"strings"
"time"

    "github.com/duanxldragon/pantheon-base/backend/modules/business/cmdb"
    "github.com/duanxldragon/pantheon-base/backend/pkg/common"
    "gorm.io/gorm"
    "gorm.io/gorm/clause"

)

var (
ErrTaskCommandConflict = errors.New("business.deploy.task.commandConflict")
ErrTaskCommandVersionRequired = errors.New("business.deploy.task.commandVersionRequired")
)

// TaskCommandRequest binds an action to the task facts the caller reviewed.
// This is a precondition, not an idempotency token; rejected actions are not replayed.
type TaskCommandRequest struct {
StateVersion string `json:"stateVersion"`
}

// Task events share the task-row sequence lock. Host/step events and log output
// are deliberately excluded so streaming output cannot invalidate controls.
func (s *DeployService) taskCommandSequences(ids []uint64) (map[uint64]uint64, error) {
result := make(map[uint64]uint64, len(ids))
if len(ids) == 0 {
return result, nil
}
var rows []struct {
TaskID uint64
Sequence uint64
}
err := s.db.Model(&DeployTaskEvent{}).Select("task_id, MAX(sequence) AS sequence").
Where("task_id IN ? AND task_host_id = 0 AND task_step_id = 0 AND event_type LIKE ?", ids, "task.%").Group("task_id").Scan(&rows).Error
if err != nil {
return nil, err
}
for _, row := range rows {
result[row.TaskID] = row.Sequence
}
return result, nil
}

func taskCommandVersion(task *DeployTask, sequence uint64) string {
// Primitive strings preserve even malformed legacy snapshot bytes. Only the
// digest leaves the server; no snapshot/credential material is exposed.
facts := []any{"deploy-command-v1", task.ID, task.Status, task.UpdatedAt.UTC().Format(time.RFC3339Nano), sequence,
task.TemplateID, task.PackageID, task.BusinessScopeID, task.ExecutorType, task.ExecutionMode, task.Action, task.TargetType,
string(task.TargetIDs), string(task.TemplateParams), string(task.ExecutionSnapshot), string(task.TargetSnapshot),
task.CredentialRefID, task.CredentialRefVersion, task.ExecutionTimeoutSeconds}
raw, _ := json.Marshal(facts)
digest := sha256.Sum256(raw)
return hex.EncodeToString(digest[:])
}

// ControlTask commits the task, hosts and command event under one row lock.
// All request-bound visibility reads use this transaction. Worker side effects
// belong to the original long-lived dispatcher and only run after commit.
func (s *DeployService) ControlTask(ctx context.Context, id uint64, actor string, scope *common.DataScopeReq, action string, req TaskCommandRequest) (*TaskResponse, error) {
var response *TaskResponse
var workerCancel context.CancelFunc
err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
var task DeployTask
if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&task, id).Error; err != nil {
if errors.Is(err, gorm.ErrRecordNotFound) {
return errDeployTaskNotFound
}
return err
}
provider := s.cmdbCapability
if owner, ok := provider.(cmdb.TargetSelectionCapability); ok {
if bound, ok := owner.WithTargetTransaction(tx).(cmdb.DeployCMDBCapability); ok {
provider = bound
}
}
inner := s.targetTransactionService(tx, provider)
visible, err := inner.taskVisible(&task, scope)
if err != nil {
return err
}
if !visible {
return errDeployTaskForbidden
}
if len(req.StateVersion) != 64 || req.StateVersion != strings.ToLower(req.StateVersion) {
return ErrTaskCommandVersionRequired
}
if _, err := hex.DecodeString(req.StateVersion); err != nil {
return ErrTaskCommandVersionRequired
}
sequences, err := inner.taskCommandSequences([]uint64{id})
if err != nil {
return err
}
if req.StateVersion != taskCommandVersion(&task, sequences[id]) {
return ErrTaskCommandConflict
}
if action == "pause" || action == "cancel" {
if cancel, ok := s.taskCancels.Load(id); ok {
workerCancel, _ = cancel.(context.CancelFunc)
}
}
// inner has no worker maps or async execution, so these calls cannot cancel
// or dispatch a worker before the transaction has committed.
switch action {
case "pause":
response, err = inner.PauseTask(id, actor, scope)
case "resume":
response, err = inner.ResumeTask(id, actor, scope)
case "retry":
response, err = inner.RetryTask(id, actor, scope)
case "cancel":
response, err = inner.CancelTask(id, actor, scope)
default:
return errors.New(msgParamInvalid)
}
return err
})
if err != nil {
return nil, err
}
switch action {
case "pause", "cancel":
if workerCancel != nil {
workerCancel()
}
case "resume", "retry":
if s.asyncExecution {
go s.runAsyncTask(id, actor)
}
}
return response, nil
}
