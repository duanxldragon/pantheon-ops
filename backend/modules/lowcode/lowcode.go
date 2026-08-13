package lowcode

import (
	"pantheon-base/modules/lowcode/dynamicmodule"
	"pantheon-base/modules/lowcode/generator"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func InitLowcodeModule(r *gin.RouterGroup, db *gorm.DB) {
	dynamicmodule.InitDynamicModule(r, db)
	generator.InitGeneratorModule(r, db)
}
