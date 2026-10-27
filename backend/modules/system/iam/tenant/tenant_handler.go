package tenant

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/duanxldragon/pantheon-base/backend/pkg/common"

	"github.com/gin-gonic/gin"
)

// Handler handles HTTP requests for tenant management.
//
// Response contract (two shapes by design):
//   - Success paths keep the historical raw-JSON shape (bare Tenant, ListResponse,
//     {"message": ...}) because out-of-band consumers (runbooks, curl smoke
//     scripts, swagger docs) parse them directly and the frontend never reads
//     this surface: the console consumes the PlatformHandler envelope on
//     /system/tenant/* (see tenant_platform_handler.go).
//   - Error paths emit the platform envelope (common.Fail / FailWithError:
//     {code, data, message}) so every failure is classified through the same
//     pipeline as the rest of the system modules and no internal error string
//     (GORM/driver detail, sentinel text) is ever echoed to the client.
type Handler struct {
	service *Service
}

// NewHandler creates a new tenant handler instance
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// failServiceError maps tenant service sentinel errors onto the platform
// envelope codes and i18n message keys, mirroring
// PlatformHandler.failServiceError so both tenant surfaces agree. Unknown
// internal errors collapse to CodeError + "request.failed"; FailWithError
// logs the real cause server-side and never leaks it in the response.
func (h *Handler) failServiceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, common.ErrBadRequest):
		common.FailWithError(c, common.CodeParamInvalid, err, errRequestFailed)
	case errors.Is(err, ErrTenantNotFound), errors.Is(err, ErrMembershipNotFound):
		common.FailWithError(c, common.CodeNotFound, err, "tenant.not_found")
	case errors.Is(err, ErrTenantCodeExists):
		// Duplicate key: the input is unacceptable as sent (the envelope code
		// set has no 409 sentinel), so it reports as a caller error.
		common.FailWithError(c, common.CodeParamInvalid, err, "tenant.code_taken")
	case errors.Is(err, ErrTenantCodeReserved):
		common.FailWithError(c, common.CodeParamInvalid, err, "tenant.code_reserved")
	case errors.Is(err, ErrMembershipExists):
		common.FailWithError(c, common.CodeParamInvalid, err, "tenant.member_exists")
	case errors.Is(err, ErrLastOwner):
		common.FailWithError(c, common.CodeParamInvalid, err, "tenant.member.last_owner")
	case errors.Is(err, common.ErrConflict):
		common.FailWithError(c, common.CodeParamInvalid, err, errRequestFailed)
	case errors.Is(err, ErrTenantHasActiveUsers):
		common.FailWithError(c, common.CodeParamInvalid, err, "tenant.has_active_members")
	case errors.Is(err, ErrTenantAdminUsernameTaken):
		common.FailWithError(c, common.CodeParamInvalid, err, "user.create.error.username_exists")
	default:
		common.FailWithError(c, common.CodeError, err, errRequestFailed)
	}
}

// CreateTenant godoc
// @Summary Create a new tenant
// @Description Create a new tenant (platform_ops role required)
// @Tags Tenant
// @Accept json
// @Produce json
// @Param body body CreateTenantDTO true "Tenant creation data"
// @Success 201 {object} Tenant
// @Failure 400 {object} common.Response
// @Failure 409 {object} common.Response
// @Router /api/v1/tenants [post]
func (h *Handler) CreateTenant(c *gin.Context) {
	var dto CreateTenantDTO
	if err := c.ShouldBindJSON(&dto); err != nil {
		common.Fail(c, common.CodeParamInvalid, errParamInvalid)
		return
	}

	tenant, err := h.service.CreateTenant(dto)
	if err != nil {
		h.failServiceError(c, err)
		return
	}

	c.JSON(http.StatusCreated, tenant)
}

// GetTenant godoc
// @Summary Get tenant by ID
// @Description Get tenant details by ID
// @Tags Tenant
// @Produce json
// @Param id path int true "Tenant ID"
// @Success 200 {object} Tenant
// @Failure 404 {object} common.Response
// @Router /api/v1/tenants/{id} [get]
func (h *Handler) GetTenant(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		common.Fail(c, common.CodeParamInvalid, errParamInvalid)
		return
	}

	tenant, err := h.service.GetTenantByID(id)
	if err != nil {
		h.failServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, tenant)
}

// ListTenants godoc
// @Summary List tenants
// @Description List tenants with pagination and filters
// @Tags Tenant
// @Produce json
// @Param status query string false "Filter by status (active, suspended, deleted)"
// @Param plan query string false "Filter by plan (free, basic, professional, enterprise)"
// @Param page query int false "Page number (default 1)"
// @Param size query int false "Page size (default 20, max 100)"
// @Success 200 {object} ListResponse
// @Router /api/v1/tenants [get]
func (h *Handler) ListTenants(c *gin.Context) {
	var filter Filter
	if err := c.ShouldBindQuery(&filter); err != nil {
		common.Fail(c, common.CodeParamInvalid, errParamInvalid)
		return
	}

	response, err := h.service.ListTenants(filter)
	if err != nil {
		h.failServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, response)
}

// UpdateTenant godoc
// @Summary Update tenant
// @Description Update tenant properties (platform_ops role required)
// @Tags Tenant
// @Accept json
// @Produce json
// @Param id path int true "Tenant ID"
// @Param body body UpdateTenantDTO true "Tenant update data"
// @Success 200 {object} map[string]string
// @Failure 404 {object} common.Response
// @Router /api/v1/tenants/{id} [put]
func (h *Handler) UpdateTenant(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		common.Fail(c, common.CodeParamInvalid, errParamInvalid)
		return
	}

	var dto UpdateTenantDTO
	if err := c.ShouldBindJSON(&dto); err != nil {
		common.Fail(c, common.CodeParamInvalid, errParamInvalid)
		return
	}

	if err := h.service.UpdateTenant(id, dto); err != nil {
		h.failServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "tenant updated successfully"})
}

// DeleteTenant godoc
// @Summary Delete tenant
// @Description Mark tenant as deleted (platform_ops role required)
// @Tags Tenant
// @Param id path int true "Tenant ID"
// @Success 200 {object} map[string]string
// @Failure 400 {object} common.Response
// @Failure 404 {object} common.Response
// @Router /api/v1/tenants/{id} [delete]
func (h *Handler) DeleteTenant(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		common.Fail(c, common.CodeParamInvalid, errParamInvalid)
		return
	}

	if err := h.service.DeleteTenant(id); err != nil {
		h.failServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "tenant deleted successfully"})
}

// AddTenantMember godoc
// @Summary Add member to tenant
// @Description Add a user as a member of a tenant
// @Tags Tenant
// @Accept json
// @Produce json
// @Param id path int true "Tenant ID"
// @Param body body AddTenantMemberDTO true "Member data"
// @Success 201 {object} map[string]string
// @Failure 400 {object} common.Response
// @Failure 404 {object} common.Response
// @Failure 409 {object} common.Response
// @Router /api/v1/tenants/{id}/members [post]
func (h *Handler) AddTenantMember(c *gin.Context) {
	tenantID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		common.Fail(c, common.CodeParamInvalid, errParamInvalid)
		return
	}

	var dto AddTenantMemberDTO
	if err := c.ShouldBindJSON(&dto); err != nil {
		common.Fail(c, common.CodeParamInvalid, errParamInvalid)
		return
	}

	if err := h.service.AddMember(tenantID, dto.UserID, dto.Role); err != nil {
		h.failServiceError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "member added successfully"})
}

// RemoveTenantMember godoc
// @Summary Remove member from tenant
// @Description Remove a user from a tenant
// @Tags Tenant
// @Param id path int true "Tenant ID"
// @Param user_id path int true "User ID"
// @Success 200 {object} map[string]string
// @Failure 404 {object} common.Response
// @Router /api/v1/tenants/{id}/members/{user_id} [delete]
func (h *Handler) RemoveTenantMember(c *gin.Context) {
	tenantID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		common.Fail(c, common.CodeParamInvalid, errParamInvalid)
		return
	}

	userID, err := strconv.ParseUint(c.Param("user_id"), 10, 64)
	if err != nil {
		common.Fail(c, common.CodeParamInvalid, errParamInvalid)
		return
	}

	if err := h.service.RemoveMember(tenantID, userID); err != nil {
		h.failServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "member removed successfully"})
}

// ListTenantMembers godoc
// @Summary List a page of tenant members
// @Description List active members of a tenant with bounded pagination. The response envelope is shared with the /members/page compatibility alias.
// @Tags Tenant
// @Produce json
// @Param id path int true "Tenant ID"
// @Param page query int false "Page number (>=1, default 1)"
// @Param pageSize query int false "Page size (clamped to 1-100, default 20)"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/tenants/{id}/members [get]
func (h *Handler) ListTenantMembers(c *gin.Context) {
	page, pageSize := parseTenantMemberPagination(c)
	h.writeTenantMembersPage(c, page, pageSize)
}

// ListTenantMembersPage keeps the additive compatibility path while applying the same bounded contract.
func (h *Handler) ListTenantMembersPage(c *gin.Context) {
	h.ListTenantMembers(c)
}

func parseTenantMemberPagination(c *gin.Context) (int, int) {
	page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
	if err != nil {
		page = 1
	}
	pageSize, err := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	if err != nil {
		pageSize = defaultTenantMemberPageSize
	}
	return normalizeTenantMemberPagination(page, pageSize)
}

func (h *Handler) writeTenantMembersPage(c *gin.Context, page, pageSize int) {
	tenantID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		common.Fail(c, common.CodeParamInvalid, errParamInvalid)
		return
	}

	members, total, err := h.service.ListTenantMembers(tenantID, page, pageSize)
	if err != nil {
		h.failServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"items":    members,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	})
}

// GetMyTenants godoc
// @Summary Get current user's tenants
// @Description Get all tenants the current user belongs to
// @Tags Tenant
// @Produce json
// @Success 200 {array} Tenant
// @Router /api/v1/tenants/my [get]
func (h *Handler) GetMyTenants(c *gin.Context) {
	// Extract user ID from context (set by auth middleware)
	userID, exists := c.Get("user_id")
	if !exists {
		common.Fail(c, common.CodeUnauthorized, "token.invalid")
		return
	}

	uid, ok := userID.(uint64)
	if !ok {
		// A malformed auth context is a server-side fault; the concrete type
		// detail must not reach the client.
		common.FailWithError(c, common.CodeError, errors.New("tenant: invalid user id type in auth context"), errRequestFailed)
		return
	}

	tenants, err := h.service.GetUserTenants(uid)
	if err != nil {
		h.failServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, tenants)
}
