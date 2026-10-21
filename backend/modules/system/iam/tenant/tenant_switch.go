package tenant

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// SwitchTenantRequest is the input for switching tenant context
type SwitchTenantRequest struct {
	TenantID uint64 `json:"tenant_id" binding:"required"`
}

// SwitchHandler handles tenant context switching
type SwitchHandler struct {
	service *Service
}

// NewSwitchHandler creates a new tenant switch handler
func NewSwitchHandler(service *Service) *SwitchHandler {
	return &SwitchHandler{service: service}
}

// SwitchTenant godoc
// @Summary Switch current tenant context
// @Description Switch authenticated user to a different tenant (user must be member)
// @Tags Tenant
// @Accept json
// @Produce json
// @Param body body SwitchTenantRequest true "Target tenant ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Router /api/v1/tenants/switch [post]
func (h *SwitchHandler) SwitchTenant(c *gin.Context) {
	// Extract user ID from context
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

	var req SwitchTenantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Verify user is member of target tenant (single bounded lookup, no
	// full member list materialization)
	memberRole, isMember, err := h.service.ActiveMembershipRole(req.TenantID, uid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if !isMember {
		c.JSON(http.StatusForbidden, gin.H{"error": "user is not a member of this tenant"})
		return
	}

	// Get tenant details
	tenant, err := h.service.GetTenantByID(req.TenantID)
	if err != nil {
		if err == ErrTenantNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "tenant not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Check tenant status
	if tenant.Status != "active" {
		c.JSON(http.StatusForbidden, gin.H{"error": "tenant is not active"})
		return
	}

	// Return new session data (caller should re-authenticate with new tenant)
	c.JSON(http.StatusOK, gin.H{
		"message":   "tenant switch successful, please re-login with new tenant_id",
		"tenant_id": req.TenantID,
		"tenant": gin.H{
			"id":   tenant.ID,
			"code": tenant.Code,
			"name": tenant.Name,
		},
		"role": memberRole,
	})
}

// ListSwitchableTenants godoc
// @Summary List tenants user can switch to
// @Description Get all active tenants the current user is a member of
// @Tags Tenant
// @Produce json
// @Success 200 {array} map[string]interface{}
// @Router /api/v1/tenants/switchable [get]
func (h *SwitchHandler) ListSwitchableTenants(c *gin.Context) {
	// Extract user ID from context
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

	// Get user's tenants
	tenants, err := h.service.GetUserTenants(uid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Resolve the user's role in each tenant with one bounded query instead
	// of re-listing every tenant's full membership.
	tenantIDs := make([]uint64, 0, len(tenants))
	for _, tenant := range tenants {
		tenantIDs = append(tenantIDs, tenant.ID)
	}
	rolesByTenant, err := h.service.ActiveMembershipRolesByTenant(uid, tenantIDs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Build response with role information
	result := make([]map[string]interface{}, 0, len(tenants))
	for _, tenant := range tenants {
		userRole := rolesByTenant[tenant.ID]

		result = append(result, map[string]interface{}{
			"id":     tenant.ID,
			"code":   tenant.Code,
			"name":   tenant.Name,
			"status": tenant.Status,
			"plan":   tenant.Plan,
			"role":   userRole,
		})
	}

	c.JSON(http.StatusOK, result)
}

// GetCurrentTenant godoc
// @Summary Get current tenant context
// @Description Get the tenant context from current session
// @Tags Tenant
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/tenants/current [get]
func (h *SwitchHandler) GetCurrentTenant(c *gin.Context) {
	// Extract tenant ID from context (set by TenantContextMiddleware)
	tenantID, exists := c.Get("tenant_id")
	if !exists {
		c.JSON(http.StatusOK, gin.H{
			"mode":      "compat",
			"tenant_id": 0,
			"message":   "running in compat mode",
		})
		return
	}

	tid, ok := tenantID.(uint64)
	if !ok || tid == 0 {
		c.JSON(http.StatusOK, gin.H{
			"mode":      "compat",
			"tenant_id": 0,
			"message":   "running in compat mode",
		})
		return
	}

	// Get tenant details
	tenant, err := h.service.GetTenantByID(tid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"mode":      "multi",
		"tenant_id": tenant.ID,
		"tenant": gin.H{
			"id":     tenant.ID,
			"code":   tenant.Code,
			"name":   tenant.Name,
			"status": tenant.Status,
			"plan":   tenant.Plan,
		},
	})
}

// RegisterSwitchRoutes registers tenant switching routes
func RegisterSwitchRoutes(r *gin.RouterGroup, handler *SwitchHandler) {
	r.POST("/tenants/switch", handler.SwitchTenant)
	r.GET("/tenants/switchable", handler.ListSwitchableTenants)
	r.GET("/tenants/current", handler.GetCurrentTenant)
}
