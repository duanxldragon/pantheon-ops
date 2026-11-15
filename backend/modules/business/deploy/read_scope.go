package deploy

import (
	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
)

// Resolve current reader authority against persisted task membership, never
// against caller-supplied IDs. Every public log/event batch uses this boundary.
func (s *DeployService) taskReadHostIDs(taskID uint64, scope *common.DataScopeReq) ([]uint64, bool, error) {
	if _, err := s.loadVisibleTask(taskID, scope); err != nil {
		return nil, false, err
	}
	var hosts []DeployTaskHost
	if err := s.db.Where(taskIDWhereClause, taskID).Find(&hosts).Error; err != nil {
		return nil, false, err
	}
	visible, err := s.visibleTaskHosts(hosts, scope)
	if err != nil {
		return nil, false, err
	}
	if len(hosts) > 0 && len(visible) == 0 {
		return nil, false, errDeployTaskForbidden
	}
	ids := make([]uint64, 0, len(visible))
	for _, host := range visible {
		ids = append(ids, host.ID)
	}
	return ids, len(visible) < len(hosts), nil
}

func (s *DeployService) loadScopedLogChunks(taskID, after uint64, filter taskLogFilter, scope *common.DataScopeReq) ([]DeployTaskLogChunk, error) {
	ids, _, err := s.taskReadHostIDs(taskID, scope)
	if err != nil {
		return nil, err
	}
	// Explicit host/step selections project to empty just like detail's host/step
	// projection, including foreign and mismatched selections.
	filter.visibleHostIDs = ids
	return s.loadLogChunks(taskID, after, filter)
}

func (s *DeployService) loadScopedTaskEvents(taskID, after uint64, scope *common.DataScopeReq) ([]DeployTaskEvent, error) {
	ids, partial, err := s.taskReadHostIDs(taskID, scope)
	if err != nil {
		return nil, err
	}
	var events []DeployTaskEvent
	steps := s.db.Model(&DeployTaskStep{}).Select("id").Where("task_id = ? AND task_host_id = biz_deploy_task_event.task_host_id", taskID)
	db := s.db.Where("task_id = ? AND sequence > ?", taskID, after).
		Where("(task_host_id = 0 AND task_step_id = 0) OR (task_host_id IN ? AND (task_step_id = 0 OR task_step_id IN (?)))", ids, steps)
	if err := db.Order("sequence ASC").Limit(500).Find(&events).Error; err != nil {
		return nil, err
	}
	if partial {
		for i := range events {
			if events[i].TaskHostID == 0 {
				// Free-form aggregate reasons may embed host output or nested host objects.
				// Keep the task transition, omit its unscoped payload under partial visibility.
				events[i].Reason = ""
			}
		}
	}
	return events, nil
}
