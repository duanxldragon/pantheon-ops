package observability

import (
	"context"

	"github.com/duanxldragon/pantheon-base/backend/modules/business/shared/dashboard"
	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
	"github.com/duanxldragon/pantheon-base/backend/pkg/contracts/businessauth"
	"github.com/duanxldragon/pantheon-base/backend/pkg/database"
	"github.com/duanxldragon/pantheon-base/backend/pkg/tenant"
	"github.com/gin-gonic/gin"
)

// DashboardSummary projects only stored health metadata for the current tenant and scope.
func (s *DatasourceService) DashboardSummary(ctx context.Context, limit int, scope *common.DataScopeReq) (*dashboard.Snapshot, error) {
	limit = dashboard.BoundLimit(limit)
	if !s.tenantBound || !dashboard.ScopeAllowed(scope) {
		return nil, common.NewForbidden("permission.denied")
	}
	if s.db == nil {
		return nil, common.ErrDatabaseNotInitialized
	}
	ctx, cancel := context.WithTimeout(ctx, dashboard.QueryTimeout)
	defer cancel()
	var rows []dashboard.Row
	err := s.tenantDB().WithContext(ctx).Model(&PrometheusDatasource{}).Scopes(database.WithDataScope(scope)).
		Select("id, name, status, type AS kind, last_checked AS observed_at").
		Order("id DESC").Limit(limit + 1).Find(&rows).Error
	if err != nil {
		return nil, err
	}
	return dashboard.NewSnapshot(rows, limit, false), nil
}

// DashboardAlertRules lists recorded evaluation errors from visible legacy alert sources.
// Alert configuration remains compat-only; this does not claim tenant-ready alert storage.
func (s *AlertRuleService) DashboardAlertRules(ctx context.Context, limit int, scope *common.DataScopeReq) (*dashboard.Snapshot, error) {
	if !dashboard.ScopeAllowed(scope) {
		return nil, common.NewForbidden("permission.denied")
	}
	if s.db == nil {
		return nil, common.ErrDatabaseNotInitialized
	}
	ctx, cancel := context.WithTimeout(ctx, dashboard.QueryTimeout)
	defer cancel()
	limit = dashboard.BoundLimit(limit)
	sources := s.db.WithContext(ctx).Model(&PrometheusDatasource{}).Where("tenant_id = 0").
		Scopes(database.WithDataScope(scope)).Select("id")
	var rows []dashboard.Row
	err := s.db.WithContext(ctx).Model(&AlertRule{}).Scopes(database.WithDataScope(scope)).
		Where("datasource_id IN (?) AND status = ?", sources, "error").
		Select("id, name, status, last_evaluation AS observed_at").Order("id DESC").Limit(limit + 1).Find(&rows).Error
	if err != nil {
		return nil, err
	}
	return dashboard.NewSnapshot(rows, limit, false), nil
}

func (h *ObservabilityHandler) dashboardAlertRules(c *gin.Context) {
	tc := tenant.FromGin(c)
	if tc == nil || tc.Mode != tenant.ModeCompat {
		queryFailure(c, ErrWorkspaceDenied)
		return
	}
	authorized, err := businessauth.Reauthorize(c, nil)
	if err != nil {
		queryFailure(c, err)
		return
	}
	resp, err := h.alertRuleService.DashboardAlertRules(c.Request.Context(), dashboard.Limit(c.Query("limit")), authorized.DataScope)
	if err != nil {
		queryFailure(c, err)
		return
	}
	if _, err := businessauth.Reauthorize(c, nil); err != nil {
		queryFailure(c, err)
		return
	}
	common.Success(c, resp)
}

func (h *ObservabilityHandler) dashboardSummary(state *queryRequestState) {
	c := state.c
	limit := dashboard.Limit(c.Query("limit"))
	if _, err := state.datasource.DashboardSummary(c.Request.Context(), limit, common.GetDataScope(c)); err != nil {
		queryFailure(c, err)
		return
	}
	if err := state.refresh(); err != nil {
		queryFailure(c, err)
		return
	}
	resp, err := state.datasource.DashboardSummary(c.Request.Context(), limit, common.GetDataScope(c))
	if err != nil {
		queryFailure(c, err)
		return
	}
	common.Success(c, resp)
}
