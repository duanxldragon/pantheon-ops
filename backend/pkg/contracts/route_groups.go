package contracts

import (
	"pantheon-ops/backend/internal/middleware"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func ProtectedGroup(r *gin.RouterGroup, path string) gin.IRoutes {
	return r.Group(path).
		Use(middleware.JWTAuthMiddleware()).
		Use(middleware.CasbinMiddleware())
}

func DataScopedGroup(r *gin.RouterGroup, path string, db *gorm.DB) gin.IRoutes {
	return ProtectedGroup(r, path).
		Use(middleware.DataScopeMiddleware(db))
}
