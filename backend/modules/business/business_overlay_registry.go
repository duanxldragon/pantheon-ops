package business

import (
	opsinit "github.com/duanxldragon/pantheon-base/backend/modules/business/opsinit"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func init() {
	if err := RegisterBusinessInitializationPreflight(preflightOverlayBusinessModules); err != nil {
		panic(err)
	}
}

func preflightOverlayBusinessModules(db *gorm.DB) error {
	if err := opsinit.Preflight(db); err != nil {
		return err
	}
	return nil
}

func initOverlayBusinessModules(r *gin.RouterGroup, db *gorm.DB) {
	opsinit.Init(r, db)
}
