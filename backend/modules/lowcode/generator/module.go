package generator

import (
	"time"

	"github.com/duanxldragon/pantheon-base/backend/internal/middleware"
	"github.com/duanxldragon/pantheon-base/backend/modules/lowcode/dynamicmodule"
	"github.com/duanxldragon/pantheon-base/backend/pkg/contracts"
	"github.com/duanxldragon/pantheon-base/backend/pkg/database"
	"github.com/duanxldragon/pantheon-base/backend/pkg/tenant"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func InitGeneratorModule(r *gin.RouterGroup, db *gorm.DB) {
	// AutoMigrate handled by versioned migrations or contracts system
	service := NewGeneratorService(db)
	handler := NewGeneratorHandler(service)
	tenantModeLoader := tenant.NewModeLoader(func() string {
		if db == nil {
			return tenant.ModeCompat
		}
		var value string
		if err := db.Table("system_setting").Where("setting_key = ?", "platform.tenant_mode").Pluck("setting_value", &value).Error; err != nil {
			return tenant.ModeCompat
		}
		return value
	}, 5*time.Second.Nanoseconds())

	contracts.RegisterBackendModules(r, db, contracts.FuncModule{
		ModuleName: "generator",
		Register: func(r *gin.RouterGroup) {
			tokenMiddleware := middleware.TokenAuthMiddleware(database.RDB)
			readAPI := r.Group("/lowcode/generator").
				Use(tokenMiddleware).
				Use(middleware.TenantContextMiddleware(tenantModeLoader, db)).
				Use(middleware.CasbinMiddleware())
			{
				readAPI.GET("/datasources", handler.ListDatasources)
				readAPI.GET("/tables", handler.ListTables)
				readAPI.GET("/table-schema", handler.PreviewTable)
				readAPI.POST("/preview-files", dynamicmodule.DynamicModuleEnvGuard(), handler.PreviewGeneratedFiles)
				readAPI.POST("/download-source", dynamicmodule.DynamicModuleEnvGuard(), handler.DownloadGeneratedSource)
			}

			writeAPI := r.Group("/lowcode/generator").
				Use(tokenMiddleware).
				Use(middleware.TenantContextMiddleware(tenantModeLoader, db)).
				Use(middleware.CasbinMiddleware()).
				Use(middleware.SecureActionMiddleware())
			{
				writeAPI.POST("/datasources", handler.CreateDatasource)
				writeAPI.PUT("/datasources/:id", handler.UpdateDatasource)
				writeAPI.DELETE("/datasources/:id", handler.DeleteDatasource)
				writeAPI.POST("/datasources/:id/test", handler.TestDatasource)
			}
		},
	})
}
