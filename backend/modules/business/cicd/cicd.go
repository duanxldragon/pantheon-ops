package cicd

import (
	"pantheon-base/modules/business/cicd/argocd"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// InitCICDModule initializes the CICD business module and returns the ArgoCD service.
func InitCICDModule(r *gin.RouterGroup, db *gorm.DB) *argocd.ArgoCDService {
	argocdSvc := argocd.NewArgoCDService(db)
	argocdHandler := argocd.NewArgoCDHandler(argocdSvc)

	// Migrate database
	if err := argocdSvc.Migrate(); err != nil {
		panic(err)
	}

	// Register routes
	argocdHandler.RegisterRoutes(r)

	return argocdSvc
}
