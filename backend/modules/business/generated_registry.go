package business

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	bizscope "pantheon-ops/backend/modules/business/bizscope"
)

func InitGeneratedBusinessModules(r *gin.RouterGroup, db *gorm.DB) {
	bizscope.InitBizScopeModule(r, db)
}
