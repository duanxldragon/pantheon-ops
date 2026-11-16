package deploy

import (
	"time"

	"errors"
	bizcap "github.com/duanxldragon/pantheon-base/backend/modules/business/capability"
	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
	"gorm.io/gorm"
)

// DeployStepHistory is an append-only observation of a committed checkpoint.
// It records no commands, configuration, credentials or error text, and is not
// a reconstructed execution timeline or a replacement for recovery state.
//
//nolint:revive // Model name retains the deploy domain prefix.
type DeployStepHistory struct {
	ID           uint64 `gorm:"primaryKey;autoIncrement"`
	TaskID       uint64 `gorm:"not null;index:idx_deploy_step_history_task"`
	TaskHostID   uint64 `gorm:"not null;index:idx_deploy_step_history_task"`
	StepID       uint64 `gorm:"not null;uniqueIndex:uk_deploy_step_history_version"`
	StepKey      string `gorm:"size:128;not null"`
	StepOrder    int    `gorm:"not null"`
	AttemptNo    int    `gorm:"not null"`
	Status       string `gorm:"size:32;not null"`
	StateVersion uint64 `gorm:"not null;uniqueIndex:uk_deploy_step_history_version"`
	StartedAt    *time.Time
	FinishedAt   *time.Time
	CapturedAt   time.Time `gorm:"not null"`
}

// TableName returns the persistence table for step history checkpoints.
func (DeployStepHistory) TableName() string { return "biz_deploy_step_history" }

func appendStepCheckpointHistory(tx *gorm.DB, step DeployTaskStep) error {
	if step.AttemptNo <= 0 {
		return nil
	}
	return tx.Create(&DeployStepHistory{
		TaskID: step.TaskID, TaskHostID: step.TaskHostID, StepID: step.ID, StepKey: step.StepKey, StepOrder: step.StepOrder,
		AttemptNo: step.AttemptNo, Status: step.Status, StateVersion: step.StateVersion,
		StartedAt: step.StartedAt, FinishedAt: step.FinishedAt, CapturedAt: time.Now(),
	}).Error
}

const stepHistoryResponseLimit = 500

func (s *DeployService) loadStepHistory(taskID uint64, hosts []DeployTaskHost) ([]TaskStepHistoryResponse, bool, error) {
	result := make([]TaskStepHistoryResponse, 0)
	if len(hosts) == 0 {
		return result, false, nil
	}
	hostIDs := make([]uint64, 0, len(hosts))
	for _, host := range hosts {
		hostIDs = append(hostIDs, host.ID)
	}
	// Both snapshot and live step must belong to this authorized task/host.
	// Filter before grouping: a malformed foreign snapshot cannot shadow a valid one.
	eligible := s.db.Model(&DeployStepHistory{}).Select("biz_deploy_step_history.step_id, biz_deploy_step_history.attempt_no, MAX(biz_deploy_step_history.state_version) AS state_version").
		Joins("JOIN biz_deploy_task_step AS step ON step.id = biz_deploy_step_history.step_id AND step.task_id = biz_deploy_step_history.task_id AND step.task_host_id = biz_deploy_step_history.task_host_id").
		Where("biz_deploy_step_history.task_id = ? AND biz_deploy_step_history.task_host_id IN ? AND biz_deploy_step_history.attempt_no > 0", taskID, hostIDs).
		Group("biz_deploy_step_history.step_id, biz_deploy_step_history.attempt_no")
	var rows []DeployStepHistory
	err := s.db.Table("biz_deploy_step_history AS history").Select("history.*").
		Joins("JOIN (?) AS latest ON latest.step_id = history.step_id AND latest.attempt_no = history.attempt_no AND latest.state_version = history.state_version", eligible).
		Where("history.task_id = ? AND history.task_host_id IN ?", taskID, hostIDs).
		Order("history.attempt_no DESC, history.state_version DESC, history.task_host_id ASC, history.step_order ASC, history.step_id ASC").Limit(stepHistoryResponseLimit + 1).Find(&rows).Error
	if err != nil {
		return nil, false, err
	}
	limited := len(rows) > stepHistoryResponseLimit
	if limited {
		rows = rows[:stepHistoryResponseLimit]
	}
	for _, row := range rows {
		result = append(result, TaskStepHistoryResponse{TaskStepResponse: TaskStepResponse{
			ID: row.StepID, TaskHostID: row.TaskHostID, StepKey: row.StepKey, StepOrder: row.StepOrder,
			AttemptNo: row.AttemptNo, Status: row.Status, StateVersion: row.StateVersion, StartedAt: row.StartedAt, FinishedAt: row.FinishedAt,
		}, CapturedAt: row.CapturedAt})
	}
	return result, limited, nil
}

// visibleTaskHosts derives host visibility from the CMDB-owned reader.
// Task visibility alone must not authorize the other hosts in a mixed-scope task.
func (s *DeployService) visibleTaskHosts(hosts []DeployTaskHost, scope *common.DataScopeReq) ([]DeployTaskHost, error) {
	if !requiresScopedTaskVisibility(scope) || len(hosts) == 0 {
		return hosts, nil
	}
	if s.hostReader == nil {
		return nil, errors.New("business.cmdb.readerNotConfigured")
	}
	ids := make([]uint64, 0, len(hosts))
	for _, host := range hosts {
		ids = append(ids, host.HostID)
	}
	page, err := s.hostReader.GetByIDs(s.requestContext(), bizcap.HostIDsQuery{HostIDs: ids, DataScope: scope})
	if err != nil {
		return nil, err
	}
	allowed := make(map[uint64]bool, len(page.Items))
	for _, host := range page.Items {
		allowed[host.ID] = true
	}
	result := make([]DeployTaskHost, 0, len(hosts))
	for _, host := range hosts {
		if allowed[host.HostID] {
			result = append(result, host)
		}
	}
	return result, nil
}
