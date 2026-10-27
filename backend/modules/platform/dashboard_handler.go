package platform

import (
	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
	"github.com/duanxldragon/pantheon-base/backend/pkg/tenant"

	"github.com/gin-gonic/gin"
)

type DashboardHandler struct {
	service *DashboardService
}

func NewDashboardHandler(service *DashboardService) *DashboardHandler {
	return &DashboardHandler{service: service}
}

func (h *DashboardHandler) GetSummary(c *gin.Context) {
	if _, ok := c.Get("userId"); !ok {
		common.Fail(c, common.CodeUnauthorized, "token.invalid")
		return
	}

	summary, err := h.service.WithTenantContext(tenant.FromGin(c)).GetSummary()
	if err != nil {
		common.FailWithError(c, common.CodeError, err, "dashboard.summary.error")
		return
	}
	common.Success(c, summary)
}
