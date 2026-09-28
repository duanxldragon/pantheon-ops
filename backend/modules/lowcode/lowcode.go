package lowcode

import (
	"github.com/duanxldragon/pantheon-base/backend/modules/lowcode/dynamicmodule"
	"github.com/duanxldragon/pantheon-base/backend/modules/lowcode/generator"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func InitLowcodeModule(r *gin.RouterGroup, db *gorm.DB) {
	dynamicmodule.InitDynamicModule(r, db)
	generator.InitGeneratorModule(r, db)
}
