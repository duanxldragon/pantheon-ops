package contracts

import (
	"pantheon-ops/backend/internal/middleware"
	"pantheon-ops/backend/pkg/database"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

func ProtectedGroup(r *gin.RouterGroup, path string) gin.IRoutes {
	return ProtectedGroupWithRedis(r, path, database.RDB)
}

func ProtectedGroupWithRedis(r *gin.RouterGroup, path string, rdb *redis.Client) gin.IRoutes {
	return r.Group(path).
		Use(middleware.TokenAuthMiddleware(rdb)).
		Use(middleware.CasbinMiddleware())
}

func DataScopedGroup(r *gin.RouterGroup, path string, db *gorm.DB) gin.IRoutes {
	return ProtectedGroup(r, path).
		Use(middleware.DataScopeMiddleware(db))
}
