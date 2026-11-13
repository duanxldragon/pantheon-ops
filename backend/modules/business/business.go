package business

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func InitBusinessModules(r *gin.RouterGroup, db *gorm.DB) {
	initializeBusinessLifecycle(db)
	initOverlayBusinessModules(r, db)
	InitGeneratedBusinessModules(r, db)
}

func initializeBusinessLifecycle(db *gorm.DB) {
	if err := businessInitializationPreflight.run(db); err != nil {
		panic(err)
	}
	if err := cleanupRetiredBusinessModules(db); err != nil {
		panic(err)
	}
}
