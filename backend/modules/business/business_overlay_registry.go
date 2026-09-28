package business

import (
	opsinit "github.com/duanxldragon/pantheon-base/backend/modules/business/opsinit"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func initOverlayBusinessModules(r *gin.RouterGroup, db *gorm.DB) {
	opsinit.Init(r, db)
}
