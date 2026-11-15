package workload

import (
	"context"
	"errors"
	"time"

	"github.com/duanxldragon/pantheon-base/backend/modules/business/shared/dashboard"
	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
	"github.com/duanxldragon/pantheon-base/backend/pkg/contracts/businessauth"
	"github.com/gin-gonic/gin"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type dashboardWorkloadRow struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Status    string `json:"status"`
	Namespace string `json:"namespace"`
}
type dashboardWorkloadSnapshot struct {
	Items           []dashboardWorkloadRow `json:"items"`
	ClusterID       uint64                 `json:"clusterId,omitempty"`
	BusinessScopeID uint64                 `json:"businessScopeId,omitempty"`
	Partial         bool                   `json:"partial"`
	SampledAt       time.Time              `json:"sampledAt"`
}

// DashboardSummary reads one Deployment page from the newest visible cluster.
// The result is an explicit sample, never an all-cluster health or workload count.
func (s *WorkloadService) DashboardSummary(ctx context.Context, limit int, scope *common.DataScopeReq) (*dashboardWorkloadSnapshot, error) {
	if !dashboard.ScopeAllowed(scope) {
		return nil, common.NewForbidden("permission.denied")
	}
	if s.clusterSvc == nil {
		return nil, common.ErrDatabaseNotInitialized
	}
	ctx, cancel := context.WithTimeout(ctx, dashboard.QueryTimeout)
	defer cancel()
	clusters, err := s.clusterSvc.DashboardSummary(ctx, 1, scope)
	if err != nil {
		return nil, err
	}
	result := &dashboardWorkloadSnapshot{Items: []dashboardWorkloadRow{}, Partial: true, SampledAt: time.Now().UTC()}
	if len(clusters.Items) == 0 {
		return result, nil
	}
	selected := clusters.Items[0]
	client, err := s.clusterSvc.GetClientsetWithContext(ctx, selected.ID, scope)
	if err != nil {
		return nil, errors.New("k8s.workload.list_failed")
	}
	limit = dashboard.BoundLimit(limit)
	items, _, err := listDeployments(ctx, client, metav1.NamespaceAll, metav1.ListOptions{Limit: int64(limit)})
	if err != nil {
		return nil, err
	}
	result.ClusterID, result.BusinessScopeID = selected.ID, selected.BusinessScopeID
	for _, item := range items {
		if len(result.Items) >= limit {
			break
		}
		result.Items = append(result.Items, dashboardWorkloadRow{ID: item.Namespace + "/" + item.Name,
			Name: item.Name, Namespace: item.Namespace, Status: item.Status})
	}
	result.SampledAt = time.Now().UTC()
	return result, nil
}

func (h *WorkloadHandler) dashboardSummary(c *gin.Context) {
	authorized, err := businessauth.Reauthorize(c, nil)
	if err != nil {
		common.Fail(c, common.CodeForbidden, "permission.denied")
		return
	}
	result, err := h.svc.DashboardSummary(c.Request.Context(), dashboard.Limit(c.Query("limit")), authorized.DataScope)
	if err != nil {
		common.Fail(c, common.CodeError, "k8s.workload.list_failed")
		return
	}
	current, err := businessauth.Reauthorize(c, nil)
	if err != nil {
		common.Fail(c, common.CodeForbidden, "permission.denied")
		return
	}
	if result.ClusterID > 0 {
		// Re-read visibility/credential state after the provider query; late revocation
		// cannot publish a completed snapshot from the old authorization.
		ctx, cancel := context.WithTimeout(c.Request.Context(), dashboard.QueryTimeout)
		defer cancel()
		if _, err := h.svc.clusterSvc.GetClientsetWithContext(ctx, result.ClusterID, current.DataScope); err != nil {
			common.Fail(c, common.CodeForbidden, "permission.denied")
			return
		}
	}
	common.Success(c, result)
}
