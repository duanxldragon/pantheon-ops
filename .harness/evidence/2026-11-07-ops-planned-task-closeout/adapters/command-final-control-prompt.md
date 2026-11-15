Read-only bounded source review; no tools. Report only proved P1/P2 defects and minimal repair or APPROVE supplied scope only. Maximum 200 words. Do not approve omitted helpers or repository. ControlTask uses a SQL transaction and task FOR UPDATE. targetTransactionService rebinds providers with asyncExecution=false and empty worker maps. appendTaskEventTx locks the task and allocates persistent monotonic sequence. Get/List versions follow DataScope. Manual tasks pause directly because no worker acknowledges pause; resume/retry dispatch only SSH async tasks. Worker/lease internals omitted.
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
if s.asyncExecution && response.ExecutorType == ExecutorTypeSSH {
go s.runAsyncTask(id, actor)
}
}
return response, nil
}

func (s *DeployService) PauseTask(id uint64, actor string, dataScope *common.DataScopeReq) (*TaskResponse, error) {
task, err := s.loadVisibleTask(id, dataScope)
if err != nil {
return nil, err
}
if task.Status != TaskStatusRunning {
return nil, errors.New(errDeployTaskInvalidPauseState)
}
nextStatus := TaskStatusPauseRequested
eventType := "task.pause_requested"
// Manual execution has no worker to acknowledge a cooperative request.
if task.ExecutorType == ExecutorTypeManual {
nextStatus = TaskStatusPaused
eventType = "task.paused"
}
now := time.Now()
transition := s.db.Model(&DeployTask{}).
Where(idAndStatusWhereClause, task.ID, TaskStatusRunning).
Updates(map[string]interface{}{
"status": nextStatus,
"updated_by": actor,
"updated_at": now,
})
if transition.Error != nil {
return nil, transition.Error
}
if transition.RowsAffected == 0 {
return nil, errors.New(errDeployTaskInvalidPauseState)
}
if err := s.appendTaskEvent(taskEventInput{taskID: task.ID, eventType: eventType, fromStatus: TaskStatusRunning, toStatus: nextStatus, actor: actor}); err != nil {
return nil, err
}
// Propagate cancellation to the current executor step so it finishes
// promptly; the worker will transition to paused after the step returns.
if cancel, ok := s.taskCancels.Load(id); ok {
if cancelFunc, ok := cancel.(context.CancelFunc); ok {
cancelFunc()
}
}
return s.GetTask(id, dataScope)
}

// ResumeTask resumes a paused task. The task re-enters running and the worker
// re-schedules unfinished hosts from the last checkpoint.
func (s *DeployService) ResumeTask(id uint64, actor string, dataScope *common.DataScopeReq) (*TaskResponse, error) {
task, err := s.loadVisibleTask(id, dataScope)
if err != nil {
return nil, err
}
if task.Status != TaskStatusPaused {
return nil, errors.New(errDeployTaskInvalidResumeState)
}
now := time.Now()
transition := s.db.Model(&DeployTask{}).
Where(idAndStatusWhereClause, task.ID, TaskStatusPaused).
Updates(map[string]interface{}{
"status": TaskStatusRunning,
"updated_by": actor,
"updated_at": now,
})
if transition.Error != nil {
return nil, transition.Error
}
if transition.RowsAffected == 0 {
return nil, errors.New(errDeployTaskInvalidResumeState)
}
if err := s.appendTaskEvent(taskEventInput{taskID: task.ID, eventType: "task.resumed", fromStatus: TaskStatusPaused, toStatus: TaskStatusRunning, actor: actor}); err != nil {
return nil, err
}
// Re-run the async worker to pick up unfinished hosts from the checkpoint.
if s.asyncExecution && task.ExecutorType == ExecutorTypeSSH {
go s.runAsyncTask(task.ID, actor)
}
return s.GetTask(id, dataScope)
}

// RetryTask creates a new attempt for a failed task. Successful prior host
// results are retained; only failed, canceled, or not-started hosts are
// scheduled again.
func (s *DeployService) RetryTask(id uint64, actor string, dataScope *common.DataScopeReq) (*TaskResponse, error) {
task, err := s.loadVisibleTask(id, dataScope)
if err != nil {
return nil, err
}
if task.Status != TaskStatusFailed {
return nil, errors.New(errDeployTaskInvalidRetryState)
}
now := time.Now()
transition := s.db.Model(&DeployTask{}).
Where(idAndStatusWhereClause, task.ID, TaskStatusFailed).
Updates(map[string]interface{}{
"status": TaskStatusRunning,
"finished_at": nil,
"updated_by": actor,
"updated_at": now,
})
if transition.Error != nil {
return nil, transition.Error
}
if transition.RowsAffected == 0 {
return nil, errors.New(errDeployTaskInvalidRetryState)
}
if err := s.appendTaskEvent(taskEventInput{taskID: task.ID, eventType: "task.retried", fromStatus: TaskStatusFailed, toStatus: TaskStatusRunning, actor: actor}); err != nil {
return nil, err
}
// Reset failed/canceled hosts to running so the worker re-executes them.
if err := s.db.Model(&DeployTaskHost{}).
Where(taskIDAndStatusInWhereClause, task.ID, []string{TaskHostStatusFailed, TaskHostStatusSkipped}).
Updates(map[string]interface{}{
"status": TaskHostStatusRunning,
"error_message": "",
"finished_at": nil,
"reported_at": nil,
"updated_at": now,
}).Error; err != nil {
return nil, err
}
if s.asyncExecution && task.ExecutorType == ExecutorTypeSSH {
go s.runAsyncTask(task.ID, actor)
}
return s.GetTask(id, dataScope)
}

// --- FZ-03: Event append and step persistence ---

// taskEventInput is the payload of an append-only task event record. Omitted
// fields carry their zero value.
type taskEventInput struct {
taskID uint64
taskHostID uint64
taskStepID uint64
eventType string
fromStatus string
toStatus string
actor string
requestID string
reason string
}

// appendTaskEvent writes an append-only event record for a task transition.
// The sequence is monotonically increasing per task and never reused.
func (s *DeployService) appendTaskEvent(event taskEventInput) error {
err := s.withTaskSequence(event.taskID, &DeployTaskEvent{}, func(sequence uint64, tx *gorm.DB) error {
return tx.Create(&DeployTaskEvent{
TaskID: event.taskID, TaskHostID: event.taskHostID, TaskStepID: event.taskStepID,
Sequence: sequence, EventType: event.eventType, FromStatus: event.fromStatus,
ToStatus: event.toStatus, Actor: event.actor, RequestID: event.requestID,
Reason: event.reason, CreatedAt: time.Now(),
}).Error
})
if err != nil {
slog.Error("append deploy task event failed", "task_id", event.taskID, "event_type", event.eventType, "error", err)
}
return err
}

func (s *DeployService) withTaskSequence(taskID uint64, model any, write func(uint64, *gorm.DB) error) error {
return s.db.Transaction(func(tx *gorm.DB) error {
var task DeployTask
if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Select("id").First(&task, taskID).Error; err != nil {
return err
}
var maxSequence uint64
if err := tx.Model(model).Where(taskIDWhereClause, taskID).Select("COALESCE(MAX(sequence), 0)").Scan(&maxSequence).Error; err != nil {
return err
}
return write(maxSequence+1, tx)
})
}
