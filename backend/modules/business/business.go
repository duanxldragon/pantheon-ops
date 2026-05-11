package business

import (
	"pantheon-ops/backend/modules/business/cmdb"
	"pantheon-ops/backend/modules/business/deploy"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func InitBusinessModules(r *gin.RouterGroup, db *gorm.DB) {
	if err := cleanupRetiredBusinessModules(db); err != nil {
		panic(err)
	}
	InitGeneratedBusinessModules(r, db)
	cmdb.InitCmdbModule(r, db)
	deploy.InitDeployModule(r, db)
}
