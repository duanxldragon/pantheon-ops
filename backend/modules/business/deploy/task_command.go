package deploy

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"time"

	bizcap "github.com/duanxldragon/pantheon-base/backend/modules/business/capability"
	"github.com/duanxldragon/pantheon-base/backend/modules/business/cmdb"
	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	bizScopeStatusActive = "active"

	taskCommandScopeInvalid = "business.deploy.task.scopeInvalid"
)

var (
	// ErrTaskCommandConflict indicates the state version no longer matches the frozen task state.
	ErrTaskCommandConflict = errors.New("business.deploy.task.commandConflict")
	// ErrTaskCommandVersionRequired indicates the command state version is missing.
	ErrTaskCommandVersionRequired = errors.New("business.deploy.task.commandVersionRequired")
)

type executionScopeError struct{ cause error }

func (e executionScopeError) Error() string { return taskCommandScopeInvalid }
func (e executionScopeError) Unwrap() error { return e.cause }

// Execution admission revalidates the source owner, independently of task
// visibility. Pause/cancel deliberately do not use this guard.
func (s *DeployService) validateActiveExecutionScope(ctx context.Context, task *DeployTask, scope *common.DataScopeReq) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	// Legacy group tasks do not declare a source scope. Keep their existing
	// target contract; do not invent a scope from the current group membership.
	if task != nil && task.TargetType == TargetTypeGroup && task.BusinessScopeID == 0 {
		return nil
	}
	if task == nil || task.BusinessScopeID == 0 {
		return errors.New("business.deploy.task.scopeRequired")
	}
	if s.bizScopeReader == nil {
		return errors.New("business.bizscope.readerNotConfigured")
	}
	owner, err := s.bizScopeReader.GetActive(ctx, task.BusinessScopeID, scope)
	if err != nil {
		return executionScopeError{cause: err}
	}
	if owner.ID != task.BusinessScopeID || owner.Status != bizScopeStatusActive {
		return errors.New(taskCommandScopeInvalid)
	}
	return ctx.Err()
}

// Read the frozen task host set through the current CMDB owner projection.
// This does not resolve a mutable group/selection or execution definition.
func (s *DeployService) validateExecutionHosts(ctx context.Context, task *DeployTask, ids []uint64, scope *common.DataScopeReq) (resultErr error) {
	parent := ctx
	ctx, cancel := context.WithTimeout(ctx, s.ownershipTiming().operation)
	defer cancel()
	defer func() {
		if s.ownership != nil && resultErr != nil && (!errors.Is(resultErr, context.Canceled) || parent.Err() == nil) {
			s.ownership.fail(resultErr)
		}
	}()
	if s.ownership != nil {
		if err := s.ownership.check(); err != nil {
			return err
		}
	}
	if err := s.validateActiveExecutionScope(ctx, task, scope); err != nil {
		return err
	}
	if task.TargetType == TargetTypeGroup && task.BusinessScopeID == 0 {
		return ctx.Err()
	}
	if s.hostReader == nil {
		return errors.New("business.cmdb.readerNotConfigured")
	}
	ids = common.NormalizeUint64IDs(ids)
	if len(ids) == 0 {
		return errors.New("business.deploy.task.emptyResolvedTargets")
	}
	page, err := s.hostReader.GetByIDs(ctx, bizcap.HostIDsQuery{HostIDs: ids, DataScope: scope})
	if err != nil {
		return err
	}
	if len(page.Items) != len(ids) {
		return errDeployTaskForbidden
	}
	wanted := make(map[uint64]bool, len(ids))
	for _, id := range ids {
		wanted[id] = true
	}
	for _, host := range page.Items {
		if !wanted[host.ID] || host.BusinessScopeID != task.BusinessScopeID {
			return errors.New("business.deploy.task.targetOutOfScope")
		}
		delete(wanted, host.ID)
	}
	return ctx.Err()
}

func (s *DeployService) validateTaskExecutionAdmission(task *DeployTask, scope *common.DataScopeReq) error {
	if err := s.validateActiveExecutionScope(s.requestContext(), task, scope); err != nil {
		return err
	}
	var ids []uint64
	if err := s.db.Model(&DeployTaskHost{}).Where(taskIDWhereClause, task.ID).Pluck("host_id", &ids).Error; err != nil {
		return err
	}
	if len(ids) == 0 {
		hosts, err := s.resolveTaskTargets(task, scope)
		if err != nil {
			return err
		}
		for _, host := range hosts {
			ids = append(ids, host.ID)
		}
	}
	return s.validateExecutionHosts(s.requestContext(), task, ids, scope)
}

// Every SSH command, including probes and pre/post checks, gets a fresh source
// and host ownership read. Nil DataScope here is the existing worker contract,
// not a reconstruction of live Base session/role authority.
type executionScopeSSHRunner struct {
	deploySSHRunner
	service *DeployService
	task    DeployTask
	hostID  uint64
}

func (r *executionScopeSSHRunner) RunScript(ctx context.Context, script string) (string, string, error) {
	if err := r.service.assertCommandOwnership(); err != nil {
		return "", "", err
	}
	if err := r.service.validateExecutionHosts(ctx, &r.task, []uint64{r.hostID}, nil); err != nil {
		return "", "", err
	}
	if r.service.ownership != nil {
		if err := r.service.ownership.check(); err != nil {
			return "", "", err
		}
	}
	return runDeployScript(ctx, r.deploySSHRunner, script)
}

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
		TaskID   uint64
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
	err := s.targetRequestService(ctx).deploymentTransaction(func(tx *gorm.DB) error {
		var task DeployTask
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&task, id).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errDeployTaskNotFound
			}
			return err
		}
		if _, err := lockDeployment(tx, id); err != nil {
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
		if action == TaskActionPause || action == TaskActionCancel {
			if cancel, ok := s.taskCancels.Load(id); ok {
				workerCancel, _ = cancel.(context.CancelFunc)
			}
		}
		// inner has no worker maps or async execution, so these calls cannot cancel
		// or dispatch a worker before the transaction has committed.
		switch action {
		case TaskActionPause:
			response, err = inner.PauseTask(id, actor, scope)
		case TaskActionResume:
			response, err = inner.ResumeTask(id, actor, scope)
		case TaskActionRetry:
			response, err = inner.RetryTask(id, actor, scope)
		case TaskActionCancel:
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
	case TaskActionPause, TaskActionCancel:
		if workerCancel != nil {
			workerCancel()
		}
	case TaskActionResume, TaskActionRetry:
		if s.asyncExecution && response.ExecutorType == ExecutorTypeSSH {
			go s.runAsyncTask(id, actor)
		}
	}
	return response, nil
}
