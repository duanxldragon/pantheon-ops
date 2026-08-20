package business

import (
	"pantheon-base/modules/business/observability"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func InitBusinessModules(r *gin.RouterGroup, db *gorm.DB) {
	if err := cleanupRetiredBusinessModules(db); err != nil {
		panic(err)
	}
	initOverlayBusinessModules(r, db)
	InitGeneratedBusinessModules(r, db)

	// TODO: Register service module (Week 2)
	// initServiceModule(r, db)

	// Register observability module
	initObservabilityModule(r, db)
}

// initServiceModule initializes the service module.
// func initServiceModule(r *gin.RouterGroup, db *gorm.DB) {
// 	repo := service.NewRepository(db)
// 	svc := service.NewService(repo)
// 	handler := service.NewHandler(svc)
// 	service.RegisterRoutes(r, handler)
// }

// initObservabilityModule initializes the observability module.
func initObservabilityModule(r *gin.RouterGroup, db *gorm.DB) {
	repo := observability.NewRepository(db)
	svc := observability.NewService(repo)
	handler := observability.NewHandler(svc)

	observability.RegisterRoutes(r, handler)
}
