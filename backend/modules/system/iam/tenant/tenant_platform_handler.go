package tenant

import (
	"errors"
	"strconv"

	"github.com/duanxldragon/pantheon-base/backend/pkg/common"

	"github.com/gin-gonic/gin"
)

const (
	errParamInvalid       = "param.invalid"
	errRequestFailed      = "request.failed"
	tenantAuditPrefix     = "tenant."
	tenantAuditSuffix     = ".title"
	tenantListErrorKey    = "tenant.list.error"
	tenantMembersErrorKey = "tenant.members.error"
)

// PlatformHandler exposes tenant management through the platform common
// envelope (common.Success / common.Fail) with audit metadata, so the system
// console consumes the same response shape as every other system module.
// The legacy raw-JSON Handler is kept intact for API compatibility.
type PlatformHandler struct {
	service *Service
}

// NewPlatformHandler builds the platform-scope tenant handler on top of the
// shared tenant service.
func NewPlatformHandler(service *Service) *PlatformHandler {
	return &PlatformHandler{service: service}
}

func tenantAuditTitle(action string) string {
	return tenantAuditPrefix + action + tenantAuditSuffix
}

func parseTenantID(c *gin.Context) (uint64, bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		common.Fail(c, common.CodeParamInvalid, errParamInvalid)
		return 0, false
	}
	return id, true
}

func (h *PlatformHandler) failServiceError(c *gin.Context, err error, fallback string) {
	switch {
	case errors.Is(err, common.ErrBadRequest):
		// Invalid input is the caller's to fix, not a server fault. Without this
		// branch a rejected field reports as a 500, which both hides the cause
		// and tells the operator to retry something that cannot succeed.
		common.FailWithError(c, common.CodeParamInvalid, err, fallback)
	case errors.Is(err, ErrTenantNotFound), errors.Is(err, ErrMembershipNotFound):
		// The subject of the request does not exist. Reporting that as 500
		// blames the server for a caller mistake and invites blind retries.
		common.FailWithError(c, common.CodeNotFound, err, "tenant.not_found")
	case errors.Is(err, ErrTenantCodeExists):
		// Duplicate key: the input is unacceptable as sent, which is the
		// caller's to fix (the envelope code set has no 409 sentinel).
		common.FailWithError(c, common.CodeParamInvalid, err, "tenant.code_taken")
	case errors.Is(err, ErrTenantCodeReserved):
		// Same HTTP class, different remedy: the code is held by a soft-deleted
		// tenant, which the default list does not show. Saying so is the whole
		// point — without it the operator sees an empty list and an
		// inexplicable conflict.
		common.FailWithError(c, common.CodeParamInvalid, err, "tenant.code_reserved")
	case errors.Is(err, ErrMembershipExists):
		common.FailWithError(c, common.CodeParamInvalid, err, "tenant.member_exists")
	case errors.Is(err, ErrLastOwner):
		// Contract §2.2: removing the last active owner would leave a tenant
		// nobody can administer. Same HTTP class as a rejected input.
		common.FailWithError(c, common.CodeParamInvalid, err, "tenant.member.last_owner")
	case errors.Is(err, common.ErrConflict):
		// Lifecycle conflicts (archived is terminal, contract §2.1) carry their
		// own i18n key; the envelope has no 409 sentinel, so they report as a
		// caller error instead of a server fault.
		common.FailWithError(c, common.CodeParamInvalid, err, fallback)
	case errors.Is(err, ErrTenantHasActiveUsers):
		// A state conflict, not a fault: DELETE /tenant is refused while an
		// active membership remains (contract §2.2) and the tenant is not
		// archived. The remedy is the archive-then-delete path, not member
		// removal — the last owner cannot be removed, so pointing the operator
		// at that path would be a dead end.
		common.FailWithError(c, common.CodeParamInvalid, err, "tenant.has_active_members")
	case errors.Is(err, ErrPlanNotFound):
		common.FailWithError(c, common.CodeNotFound, err, "tenant.plan.not_found")
	case errors.Is(err, ErrPlanCodeExists):
		common.FailWithError(c, common.CodeParamInvalid, err, "tenant.plan.code_taken")
	case errors.Is(err, ErrPlanInUse):
		// Retiring a plan tenants still sit on would strand them on a limit
		// nobody chose, so the remedy the console can offer is "reassign first".
		common.FailWithError(c, common.CodeParamInvalid, err, "tenant.plan.in_use")
	default:
		common.FailWithError(c, common.CodeError, err, fallback)
	}
}

// ListTenants handles GET platform tenant listing with filters.
func (h *PlatformHandler) ListTenants(c *gin.Context) {
	var filter Filter
	if err := c.ShouldBindQuery(&filter); err != nil {
		common.Fail(c, common.CodeParamInvalid, errParamInvalid)
		return
	}
	resp, err := h.service.ListTenants(filter)
	if err != nil {
		common.Fail(c, common.CodeError, tenantListErrorKey)
		return
	}
	common.Success(c, resp)
}

// GetTenant handles GET of a single tenant by id.
func (h *PlatformHandler) GetTenant(c *gin.Context) {
	id, ok := parseTenantID(c)
	if !ok {
		return
	}
	row, err := h.service.GetTenantByID(id)
	if err != nil {
		h.failServiceError(c, err, errRequestFailed)
		return
	}
	common.Success(c, row)
}

// CreateTenant handles POST creation of a tenant with audit metadata.
func (h *PlatformHandler) CreateTenant(c *gin.Context) {
	common.SetAuditMetadata(c, tenantAuditTitle("create"), common.BusinessInsert)

	var dto CreateTenantDTO
	if err := c.ShouldBindJSON(&dto); err != nil {
		common.Fail(c, common.CodeParamInvalid, errParamInvalid)
		return
	}
	row, err := h.service.CreateTenant(dto)
	if err != nil {
		h.failServiceError(c, err, errRequestFailed)
		return
	}
	common.Success(c, row)
}

// UpdateTenant handles PUT updates of a tenant with audit metadata.
func (h *PlatformHandler) UpdateTenant(c *gin.Context) {
	common.SetAuditMetadata(c, tenantAuditTitle("update"), common.BusinessUpdate)

	id, ok := parseTenantID(c)
	if !ok {
		return
	}
	var dto UpdateTenantDTO
	if err := c.ShouldBindJSON(&dto); err != nil {
		common.Fail(c, common.CodeParamInvalid, errParamInvalid)
		return
	}
	if err := h.service.UpdateTenant(id, dto); err != nil {
		h.failServiceError(c, err, errRequestFailed)
		return
	}
	common.Success(c, gin.H{})
}

// DeleteTenant handles DELETE of a tenant with audit metadata.
func (h *PlatformHandler) DeleteTenant(c *gin.Context) {
	common.SetAuditMetadata(c, tenantAuditTitle("delete"), common.BusinessDelete)

	id, ok := parseTenantID(c)
	if !ok {
		return
	}
	if err := h.service.DeleteTenant(id); err != nil {
		h.failServiceError(c, err, errRequestFailed)
		return
	}
	common.Success(c, gin.H{})
}

// ListTenantMembers handles GET of a tenant membership listing.
func (h *PlatformHandler) ListTenantMembers(c *gin.Context) {
	id, ok := parseTenantID(c)
	if !ok {
		return
	}
	page, pageSize := parseTenantMemberPagination(c)
	items, total, err := h.service.ListTenantMembersWithUser(id, page, pageSize)
	if err != nil {
		common.Fail(c, common.CodeError, tenantMembersErrorKey)
		return
	}
	common.Success(c, gin.H{
		"items":    items,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	})
}

// AddTenantMember handles POST addition of a tenant member with audit metadata.
func (h *PlatformHandler) AddTenantMember(c *gin.Context) {
	common.SetAuditMetadata(c, tenantAuditTitle("member_add"), common.BusinessGrant)

	id, ok := parseTenantID(c)
	if !ok {
		return
	}
	var dto AddTenantMemberDTO
	if err := c.ShouldBindJSON(&dto); err != nil {
		common.Fail(c, common.CodeParamInvalid, errParamInvalid)
		return
	}
	if err := h.service.AddMember(id, dto.UserID, dto.Role); err != nil {
		h.failServiceError(c, err, errRequestFailed)
		return
	}
	common.Success(c, gin.H{})
}

// RemoveTenantMember handles DELETE removal of a tenant member with audit metadata.
func (h *PlatformHandler) RemoveTenantMember(c *gin.Context) {
	common.SetAuditMetadata(c, tenantAuditTitle("member_remove"), common.BusinessDelete)

	id, ok := parseTenantID(c)
	if !ok {
		return
	}
	userID, err := strconv.ParseUint(c.Param("userId"), 10, 64)
	if err != nil || userID == 0 {
		common.Fail(c, common.CodeParamInvalid, errParamInvalid)
		return
	}
	if err := h.service.RemoveMember(id, userID); err != nil {
		h.failServiceError(c, err, errRequestFailed)
		return
	}
	common.Success(c, gin.H{})
}
