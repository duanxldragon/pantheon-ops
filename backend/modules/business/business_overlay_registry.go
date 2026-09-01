package business

import (
	opsinit "pantheon-base/modules/business/opsinit"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func initOverlayBusinessModules(r *gin.RouterGroup, db *gorm.DB) {
	opsinit.Init(r, db)
}
