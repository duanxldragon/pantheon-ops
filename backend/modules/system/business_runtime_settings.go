package system

import (
	"github.com/duanxldragon/pantheon-base/backend/modules/system/config/setting"
	"github.com/duanxldragon/pantheon-base/backend/pkg/contracts"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func prepareBusinessRuntimeSettings(r *gin.RouterGroup, db *gorm.DB, service *setting.SettingService, definitions []contracts.BusinessSettingDefinition) error {
	if len(definitions) == 0 {
		return nil
	}
	return contracts.RegisterRequiredBackendModules(r, db, contracts.FuncModule{
		ModuleName:    "business-runtime-settings",
		MigrateFunc:   func(_ *gorm.DB) error { return service.MigrateBusinessSettings(definitions) },
		BootstrapFunc: func(_ *gorm.DB) error { return service.BootstrapBusinessSettings(definitions) },
	})
}
