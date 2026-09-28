package dynamicmodule

import (
	"strings"
	"time"

	"github.com/duanxldragon/pantheon-base/backend/internal/middleware"
	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
	commonsecurity "github.com/duanxldragon/pantheon-base/backend/pkg/common/security"
	"github.com/duanxldragon/pantheon-base/backend/pkg/contracts"
	"github.com/duanxldragon/pantheon-base/backend/pkg/database"
	"github.com/duanxldragon/pantheon-base/backend/pkg/tenant"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func dynamicModuleEnabled() bool {
	value := strings.ToLower(strings.TrimSpace(commonsecurity.ResolveSecret("PANTHEON_ENABLE_DYNAMIC_MODULES", "")))
	switch value {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	}
	return !commonsecurity.IsProductionEnv()
}

func DynamicModuleEnvGuard() gin.HandlerFunc {
	return func(c *gin.Context) {
		if dynamicModuleEnabled() {
			c.Next()
			return
		}
		common.FailWithCode(c, common.CodeForbidden, "module.dynamic.disabled")
		c.Abort()
	}
}

// InitDynamicModule 初始化动态模块管理
func InitDynamicModule(r *gin.RouterGroup, db *gorm.DB) {
	// AutoMigrate handled by versioned migrations or contracts system

	service := NewDynamicModuleService(db)
	handler := NewDynamicModuleHandler(service)

	tokenMiddleware := middleware.TokenAuthMiddleware(database.RDB)
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

	modules := []contracts.BackendModule{
		contracts.FuncModule{
			ModuleName: "dynamic-module",
			MigrateFunc: func(db *gorm.DB) error {
				return db.AutoMigrate(&ModuleRegistration{})
			},
			Register: func(r *gin.RouterGroup) {
				readAPI := r.Group("/lowcode/dynamic-modules").
					Use(tokenMiddleware).
					Use(middleware.TenantContextMiddleware(tenantModeLoader, db)).
					Use(middleware.CasbinMiddleware()).
					Use(DynamicModuleEnvGuard())
				{
					readAPI.GET("", handler.ListModules)
					readAPI.GET("/schema", handler.GetModuleSchema)
					readAPI.GET("/:name", handler.GetModuleStatus)
				}

				writeAPI := r.Group("/lowcode/dynamic-modules").
					Use(tokenMiddleware).
					Use(middleware.TenantContextMiddleware(tenantModeLoader, db)).
					Use(middleware.CasbinMiddleware()).
					Use(DynamicModuleEnvGuard()).
					Use(middleware.SecureActionMiddleware())
				{
					writeAPI.POST("/generate", handler.GenerateAndRegisterModule)
					writeAPI.POST("/repair", handler.RepairRegistries)
					writeAPI.POST("/activation-audit", handler.AuditPendingActivations)
					writeAPI.POST("", handler.RegisterModule)
					writeAPI.DELETE("/:name", handler.UnregisterModule)
					writeAPI.DELETE("/:name/record", handler.DeleteModuleRecord)
					writeAPI.DELETE("/:name/purge", handler.PurgeModule)
				}
			},
		},
	}

	contracts.RegisterBackendModules(r, db, modules...)
}
