package tenant

import (
	"github.com/duanxldragon/pantheon-base/backend/pkg/common"

	"github.com/gin-gonic/gin"
)

// The tenant-ops console surface (contract §2.5): a platform-global rollup, a
// cross-tenant member listing, and the plan editor.
//
// These live under /system/tenant-ops/* rather than /system/tenant/* for one
// concrete reason: `GET /system/tenant/:id` already owns the wildcard child of
// that group, so a static sibling like /tenant/overview is a routing-shape
// decision waiting to panic on a Gin upgrade. A sibling group keeps the console
// paths free of that constraint.
//
// Every route here is a declared platform-scoped read or write: it spans tenants
// by design, which the tenant console's PlatformCapabilityMiddleware is what
// makes safe. The permission keys all stay under the `system:tenant:` prefix so
// pkg/tenant.IsPlatformOnlyPermissionKey classifies them without a second list.

const (
	tenantOverviewErrorKey = "tenant.overview.error"
	tenantPlanErrorKey     = "tenant.plan.error"
)

// GetOverview handles GET /system/tenant-ops/overview.
func (h *PlatformHandler) GetOverview(c *gin.Context) {
	var filter Filter
	if err := c.ShouldBindQuery(&filter); err != nil {
		common.Fail(c, common.CodeParamInvalid, errParamInvalid)
		return
	}
	resp, err := h.service.Overview(filter)
	if err != nil {
		h.failServiceError(c, err, tenantOverviewErrorKey)
		return
	}
	common.Success(c, resp)
}

// ListAllMembers handles GET /system/tenant-ops/member.
func (h *PlatformHandler) ListAllMembers(c *gin.Context) {
	var filter MemberRowFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		common.Fail(c, common.CodeParamInvalid, errParamInvalid)
		return
	}
	resp, err := h.service.ListMembersAcrossPages(filter)
	if err != nil {
		h.failServiceError(c, err, tenantMembersErrorKey)
		return
	}
	common.Success(c, resp)
}

// ListPlans handles GET /system/tenant-ops/plan/list.
func (h *PlatformHandler) ListPlans(c *gin.Context) {
	resp, err := h.service.ListPlans()
	if err != nil {
		h.failServiceError(c, err, tenantPlanErrorKey)
		return
	}
	common.Success(c, gin.H{"items": resp})
}

// CreatePlan handles POST /system/tenant-ops/plan.
func (h *PlatformHandler) CreatePlan(c *gin.Context) {
	common.SetAuditMetadata(c, tenantAuditTitle("plan_create"), common.BusinessInsert)

	var req PlanCreateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, errParamInvalid)
		return
	}
	plan, err := h.service.Plans().Create(req)
	if err != nil {
		h.failServiceError(c, err, tenantPlanErrorKey)
		return
	}
	common.Success(c, plan)
}

// UpdatePlan handles PUT /system/tenant-ops/plan/:code.
func (h *PlatformHandler) UpdatePlan(c *gin.Context) {
	common.SetAuditMetadata(c, tenantAuditTitle("plan_update"), common.BusinessUpdate)

	code := normalizePlanCode(c.Param("code"))
	if code == "" {
		common.Fail(c, common.CodeParamInvalid, errParamInvalid)
		return
	}
	var req PlanUpdateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, errParamInvalid)
		return
	}
	plan, err := h.service.Plans().Update(code, req)
	if err != nil {
		h.failServiceError(c, err, tenantPlanErrorKey)
		return
	}
	common.Success(c, plan)
}
