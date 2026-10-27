package tenant

import (
	"github.com/gin-gonic/gin"
)

// RegisterRoutes registers tenant management routes
// Routes are protected by platform_ops role (configured in main router)
func RegisterRoutes(r *gin.RouterGroup, handler *Handler) {
	tenants := r.Group("/tenants")
	{
		// Public/authenticated routes
		tenants.GET("/my", handler.GetMyTenants) // Get current user's tenants

		// Platform ops routes (require platform_ops role - enforced by middleware)
		tenants.POST("", handler.CreateTenant)
		tenants.GET("", handler.ListTenants)
		tenants.GET("/:id", handler.GetTenant)
		tenants.PUT("/:id", handler.UpdateTenant)
		tenants.DELETE("/:id", handler.DeleteTenant)

		// Tenant member management
		tenants.POST("/:id/members", handler.AddTenantMember)
		tenants.GET("/:id/members", handler.ListTenantMembers)
		tenants.GET("/:id/members/page", handler.ListTenantMembersPage)
		tenants.DELETE("/:id/members/:user_id", handler.RemoveTenantMember)
	}
}
