package business

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func InitBusinessModules(r *gin.RouterGroup, db *gorm.DB) {
	if err := cleanupRetiredBusinessModules(db); err != nil {
		panic(err)
	}
	InitGeneratedBusinessModules(r, db)
}
