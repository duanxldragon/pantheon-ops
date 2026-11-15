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

func (s *DeployService) CancelTask(id uint64, actor string, dataScope *common.DataScopeReq) (*TaskResponse, error) {
now := time.Now()
task, err := s.loadVisibleTask(id, dataScope)
if err != nil {
return nil, err
}
allowed := map[string]bool{
TaskStatusPending: true,
TaskStatusRunning: true,
TaskStatusPauseRequested: true,
TaskStatusPaused: true,
TaskStatusCancelRequested: task.ExecutorType == ExecutorTypeManual,
}
if !allowed[task.Status] {
return nil, errors.New(errDeployTaskInvalidCancelState)
}
nextStatus := TaskStatusCancelRequested
// Manual tasks have no worker to acknowledge cancellation. Complete both
// a new cancel and a legacy stranded request through the existing CAS path.
if task.Status == TaskStatusPending || task.Status == TaskStatusPaused || task.ExecutorType == ExecutorTypeManual {
nextStatus = TaskStatusCanceled
}
updates := map[string]interface{}{
"status": nextStatus,
"updated_by": actor,
"updated_at": now,
}
if nextStatus == TaskStatusCanceled {
updates["finished_at"] = &now
}
transition := s.db.Model(&DeployTask{}).
Where(idAndStatusWhereClause, task.ID, task.Status).
Updates(updates)
if transition.Error != nil {
return nil, transition.Error
}
if transition.RowsAffected == 0 {
return nil, errors.New(errDeployTaskInvalidCancelState)
}
if err := s.db.Model(&DeployTaskHost{}).Where(taskIDAndStatusInWhereClause, id, []string{TaskHostStatusPending, TaskHostStatusRunning}).Updates(map[string]interface{}{
"status": TaskHostStatusSkipped,
"finished_at": &now,
"updated_by": actor,
"updated_at": now,
}).Error; err != nil {
return nil, err
}
if cancel, ok := s.taskCancels.Load(id); ok {
if cancelFunc, ok := cancel.(context.CancelFunc); ok {
cancelFunc()
}
}
eventType := "task.cancel_requested"
if nextStatus == TaskStatusCanceled {
eventType = "task.canceled"
}
if err := s.appendTaskEvent(taskEventInput{taskID: task.ID, eventType: eventType, fromStatus: task.Status, toStatus: nextStatus, actor: actor}); err != nil {
return nil, err
}
return s.GetTask(id, dataScope)
}

// PauseTask requests cooperative pause for a running task. The task enters
// pause_requested; the executor must finish the current step, then the worker
// transitions the task to paused.
func (s *DeployService) PauseTask(id uint64, actor string, dataScope *common.DataScopeReq) (*TaskResponse, error) {
task, err := s.loadVisibleTask(id, dataScope)
if err != nil {
return nil, err
}
if task.Status != TaskStatusRunning {
return nil, errors.New(errDeployTaskInvalidPauseState)
}
now := time.Now()
transition := s.db.Model(&DeployTask{}).
Where(idAndStatusWhereClause, task.ID, TaskStatusRunning).
Updates(map[string]interface{}{
"status": TaskStatusPauseRequested,
"updated_by": actor,
"updated_at": now,
})
if transition.Error != nil {
return nil, transition.Error
}
if transition.RowsAffected == 0 {
return nil, errors.New(errDeployTaskInvalidPauseState)
}
if err := s.appendTaskEvent(taskEventInput{taskID: task.ID, eventType: "task.pause_requested", fromStatus: TaskStatusRunning, toStatus: TaskStatusPauseRequested, actor: actor}); err != nil {
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
if s.asyncExecution {
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
if s.asyncExecution {
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

// PauseTask suspends the deployment task identified by the path id.
func (h *DeployHandler) PauseTask(c *gin.Context) {
common.SetAuditMetadata(c, "暂停部署任务", common.BusinessUpdate)
id, ok := parseIDParam(c)
if !ok {
return
}
var req TaskCommandRequest
if err := c.ShouldBindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
return
}
resp, err := h.svc.ControlTask(c.Request.Context(), id, currentActor(c), common.GetDataScope(c), "pause", req)
if err != nil {
failDeployTaskError(c, err, "deploytask.pause_failed")
return
}
common.Success(c, resp)
}

// ResumeTask continues the deployment task identified by the path id.
func (h *DeployHandler) ResumeTask(c *gin.Context) {
common.SetAuditMetadata(c, "恢复部署任务", common.BusinessUpdate)
id, ok := parseIDParam(c)
if !ok {
return
}
var req TaskCommandRequest
if err := c.ShouldBindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
return
}
resp, err := h.svc.ControlTask(c.Request.Context(), id, currentActor(c), common.GetDataScope(c), "resume", req)
if err != nil {
failDeployTaskError(c, err, "deploytask.resume_failed")
return
}
common.Success(c, resp)
}

// RetryTask re-runs the deployment task identified by the path id.
func (h *DeployHandler) RetryTask(c *gin.Context) {
common.SetAuditMetadata(c, "重试部署任务", common.BusinessUpdate)
id, ok := parseIDParam(c)
if !ok {
return
}
var req TaskCommandRequest
if err := c.ShouldBindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
return
}
resp, err := h.svc.ControlTask(c.Request.Context(), id, currentActor(c), common.GetDataScope(c), "retry", req)
if err != nil {
failDeployTaskError(c, err, "deploytask.retry_failed")
return
}
common.Success(c, resp)
}

func (h *DeployHandler) CancelTask(c *gin.Context) {
common.SetAuditMetadata(c, "取消部署任务", common.BusinessUpdate)
id, ok := parseIDParam(c)
if !ok {
return
}
var req TaskCommandRequest
if err := c.ShouldBindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
return
}
resp, err := h.svc.ControlTask(c.Request.Context(), id, currentActor(c), common.GetDataScope(c), "cancel", req)
if err != nil {
failDeployTaskError(c, err, "deploytask.cancel_failed")
return
}
common.Success(c, resp)
}
