package business

import (
	bizscope "pantheon-base/modules/business/bizscope"
	cmdb "pantheon-base/modules/business/cmdb"
	deploy "pantheon-base/modules/business/deploy"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func initOverlayBusinessModules(r *gin.RouterGroup, db *gorm.DB) {
	bizscope.InitBizScopeModule(r, db)
	cmdb.InitCmdbModule(r, db)
	deploy.InitDeployModule(r, db)
}
