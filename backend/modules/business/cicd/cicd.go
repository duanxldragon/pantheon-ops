package cicd

import (
	"github.com/duanxldragon/pantheon-base/backend/modules/business/cicd/argocd"
	"github.com/duanxldragon/pantheon-base/backend/pkg/contracts"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// InitCICDModule initializes the CICD business module and returns the ArgoCD
// service. Routes are registered under a DataScopedGroup so token + Casbin +
// data-scope middleware apply.
func InitCICDModule(r *gin.RouterGroup, db *gorm.DB) *argocd.ArgoCDService {
	argocdSvc := argocd.NewArgoCDService(db)
	argocdHandler := argocd.NewArgoCDHandler(argocdSvc)

	// Migrate database (AutoMigrate is the module-level schema path; versioned
	// migrations for the ArgoCD tables are part of the v1.2 release baseline).
	if err := argocdSvc.Migrate(); err != nil {
		panic(err)
	}

	contracts.RegisterBackendModules(r, db, contracts.FuncModule{
		ModuleName:    cicdModuleKey,
		MigrateFunc:   func(db *gorm.DB) error { return argocdSvc.Migrate() },
		SeedMenusFunc: SeedMenus,
		SeedPermsFunc: SeedPerms,
		Register: func(r *gin.RouterGroup) {
			// Handler routes are prefixed /cicd/argocd/*, so the group mounts
			// at /business to yield the canonical /business/cicd/argocd/* API.
			argocdHandler.RegisterRoutes(contracts.DataScopedGroup(r, "/business", db))
		},
	})

	return argocdSvc
}
