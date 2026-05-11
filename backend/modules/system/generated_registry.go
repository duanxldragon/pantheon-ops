package system

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func InitGeneratedSystemModules(r *gin.RouterGroup, db *gorm.DB) {
	_ = r
	_ = db
}
