package business

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func InitGeneratedBusinessModules(r *gin.RouterGroup, db *gorm.DB) {
	// Intentionally empty: the low-code module generator rewrites this file and
	// fills in registrations once generated business modules exist.
}
