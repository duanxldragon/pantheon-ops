package cluster

import (
	"context"

	"github.com/duanxldragon/pantheon-base/backend/modules/business/shared/dashboard"
	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
	"github.com/duanxldragon/pantheon-base/backend/pkg/database"
)

// DashboardSummary reads stored registration health only, without loading credentials or contacting Kubernetes.
func (s *ClusterService) DashboardSummary(ctx context.Context, limit int, scope *common.DataScopeReq) (*dashboard.Snapshot, error) {
	limit = dashboard.BoundLimit(limit)
	if !dashboard.ScopeAllowed(scope) {
		return nil, common.NewForbidden("permission.denied")
	}
	if s.db == nil {
		return nil, common.ErrDatabaseNotInitialized
	}
	ctx, cancel := context.WithTimeout(ctx, dashboard.QueryTimeout)
	defer cancel()
	var rows []dashboard.Row
	err := s.db.WithContext(ctx).Model(&Cluster{}).Scopes(database.WithDataScope(scope)).
		Select("id, name, status, business_scope_id, last_synced_at AS observed_at").
		Order("id DESC").Limit(limit + 1).Find(&rows).Error
	if err != nil {
		return nil, err
	}
	return dashboard.NewSnapshot(rows, limit, false), nil
}
