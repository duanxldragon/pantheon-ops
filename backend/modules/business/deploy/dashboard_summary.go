package deploy

import (
	"context"
	"strings"
	"time"

	bizcap "github.com/duanxldragon/pantheon-base/backend/modules/business/capability"
	"github.com/duanxldragon/pantheon-base/backend/modules/business/shared/dashboard"
	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
	"gorm.io/gorm"
)

// DashboardSummary reads a bounded recent sample, never full task/host detail or execution secrets.
func (s *DeployService) DashboardSummary(ctx context.Context, limit int, status string, scope *common.DataScopeReq) (*dashboard.Snapshot, error) {
	limit = dashboard.BoundLimit(limit)
	if !dashboard.ScopeAllowed(scope) {
		return nil, common.NewForbidden("permission.denied")
	}
	if s.db == nil {
		return nil, common.ErrDatabaseNotInitialized
	}
	ctx, cancel := context.WithTimeout(ctx, dashboard.QueryTimeout)
	defer cancel()
	end := time.Now().UTC()
	start := end.Add(-24 * time.Hour)
	q := s.db.WithContext(ctx).Model(&DeployTask{}).Where("created_at >= ? AND created_at < ?", start, end)
	if strings.TrimSpace(status) != "" {
		q = q.Where("status = ?", strings.TrimSpace(status))
	}
	var rows []dashboard.Row
	if err := q.Select("id, name, status, business_scope_id, updated_at AS observed_at").Order("id DESC").Limit(limit + 1).Find(&rows).Error; err != nil {
		return nil, err
	}
	partial := len(rows) > limit
	if requiresScopedTaskVisibility(scope) {
		visible, err := s.dashboardVisibleTasks(ctx, rows, scope)
		if err != nil {
			return nil, err
		}
		rows = visible
		// Restricted readers see only authorized persisted host membership from the bounded sample.
		// Unstarted address-only targets are omitted rather than expanding an unbounded target set.
		partial = true
	}
	snapshot := dashboard.NewSnapshot(rows, limit, partial)
	snapshot.RangeStart, snapshot.RangeEnd = &start, &end
	return snapshot, nil
}

// applyTaskTimeRange keeps dashboard drill-down on the same half-open UTC range.
func applyTaskTimeRange(db *gorm.DB, query TaskQuery) (*gorm.DB, error) {
	if query.CreatedFrom == "" && query.CreatedTo == "" {
		return db, nil
	}
	start, err := time.Parse(time.RFC3339Nano, query.CreatedFrom)
	if err != nil {
		return nil, common.NewBadRequest("common.param_invalid")
	}
	end, err := time.Parse(time.RFC3339Nano, query.CreatedTo)
	if err != nil || !start.Before(end) || end.Sub(start) > 31*24*time.Hour {
		return nil, common.NewBadRequest("common.param_invalid")
	}
	return db.Where("created_at >= ? AND created_at < ?", start.UTC(), end.UTC()), nil
}

func (s *DeployService) dashboardVisibleTasks(ctx context.Context, rows []dashboard.Row, scope *common.DataScopeReq) ([]dashboard.Row, error) {
	if len(rows) == 0 {
		return rows, nil
	}
	if s.hostReader == nil {
		return nil, common.NewForbidden("permission.denied")
	}
	ids := make([]uint64, len(rows))
	for i := range rows {
		ids[i] = rows[i].ID
	}
	var hosts []DeployTaskHost
	if err := s.db.WithContext(ctx).Model(&DeployTaskHost{}).Select("task_id, host_id").
		Where("task_id IN ?", ids).Order("id DESC").Limit(128).Find(&hosts).Error; err != nil {
		return nil, err
	}
	if len(hosts) == 0 {
		return []dashboard.Row{}, nil
	}
	hostIDs := make([]uint64, len(hosts))
	for i := range hosts {
		hostIDs[i] = hosts[i].HostID
	}
	page, err := s.hostReader.GetByIDs(ctx, bizcap.HostIDsQuery{HostIDs: hostIDs, DataScope: scope})
	if err != nil {
		return nil, err
	}
	allowedHosts, allowedTasks := map[uint64]bool{}, map[uint64]bool{}
	for _, host := range page.Items {
		allowedHosts[host.ID] = true
	}
	for _, host := range hosts {
		if allowedHosts[host.HostID] {
			allowedTasks[host.TaskID] = true
		}
	}
	visible := make([]dashboard.Row, 0, len(rows))
	for _, row := range rows {
		if allowedTasks[row.ID] {
			visible = append(visible, row)
		}
	}
	return visible, nil
}
