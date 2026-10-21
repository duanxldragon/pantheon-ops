package tenant

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

const (
	invalidTenantIDMessage = "invalid tenant id"
	tenantNotFoundMessage  = "tenant not found"
)

// Handler handles HTTP requests for tenant management
type Handler struct {
	service *Service
}

// NewHandler creates a new tenant handler instance
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// CreateTenant godoc
// @Summary Create a new tenant
// @Description Create a new tenant (platform_ops role required)
// @Tags Tenant
// @Accept json
// @Produce json
// @Param body body CreateTenantDTO true "Tenant creation data"
// @Success 201 {object} Tenant
// @Failure 400 {object} map[string]string
// @Failure 409 {object} map[string]string
// @Router /api/v1/tenants [post]
func (h *Handler) CreateTenant(c *gin.Context) {
	var dto CreateTenantDTO
	if err := c.ShouldBindJSON(&dto); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tenant, err := h.service.CreateTenant(dto)
	if err != nil {
		if err == ErrTenantCodeExists {
			c.JSON(http.StatusConflict, gin.H{"error": "tenant code already exists"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
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
// @Failure 404 {object} map[string]string
// @Router /api/v1/tenants/{id} [get]
func (h *Handler) GetTenant(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": invalidTenantIDMessage})
		return
	}

	tenant, err := h.service.GetTenantByID(id)
	if err != nil {
		if err == ErrTenantNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": tenantNotFoundMessage})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
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
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	response, err := h.service.ListTenants(filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
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
// @Failure 404 {object} map[string]string
// @Router /api/v1/tenants/{id} [put]
func (h *Handler) UpdateTenant(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": invalidTenantIDMessage})
		return
	}

	var dto UpdateTenantDTO
	if err := c.ShouldBindJSON(&dto); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.service.UpdateTenant(id, dto); err != nil {
		if err == ErrTenantNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": tenantNotFoundMessage})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
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
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /api/v1/tenants/{id} [delete]
func (h *Handler) DeleteTenant(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": invalidTenantIDMessage})
		return
	}

	if err := h.service.DeleteTenant(id); err != nil {
		if err == ErrTenantNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": tenantNotFoundMessage})
			return
		}
		if err == ErrTenantHasActiveUsers {
			c.JSON(http.StatusBadRequest, gin.H{"error": "cannot delete tenant with active members"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
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
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 409 {object} map[string]string
// @Router /api/v1/tenants/{id}/members [post]
func (h *Handler) AddTenantMember(c *gin.Context) {
	tenantID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": invalidTenantIDMessage})
		return
	}

	var dto AddTenantMemberDTO
	if err := c.ShouldBindJSON(&dto); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.service.AddMember(tenantID, dto.UserID, dto.Role); err != nil {
		if err == ErrTenantNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": tenantNotFoundMessage})
			return
		}
		if err == ErrMembershipExists {
			c.JSON(http.StatusConflict, gin.H{"error": "user is already a member"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
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
// @Failure 404 {object} map[string]string
// @Router /api/v1/tenants/{id}/members/{user_id} [delete]
func (h *Handler) RemoveTenantMember(c *gin.Context) {
	tenantID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": invalidTenantIDMessage})
		return
	}

	userID, err := strconv.ParseUint(c.Param("user_id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}

	if err := h.service.RemoveMember(tenantID, userID); err != nil {
		if err == ErrMembershipNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "membership not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
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
		c.JSON(http.StatusBadRequest, gin.H{"error": invalidTenantIDMessage})
		return
	}

	members, total, err := h.service.ListTenantMembers(tenantID, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
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
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	uid, ok := userID.(uint64)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid user id type"})
		return
	}

	tenants, err := h.service.GetUserTenants(uid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, tenants)
}
