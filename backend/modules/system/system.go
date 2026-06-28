package system

import (
	"pantheon-ops/backend/pkg/contracts"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// InitSystemModule 初始化系统模块
func InitSystemModule(r *gin.RouterGroup, db *gorm.DB) {
	// 清理历史废弃菜单（/workspace、/operations 等）
	if err := CleanupObsoleteMenus(db); err != nil {
		panic(err)
	}

	deps := newSystemModuleDependencies(db)
	modules := make([]contracts.BackendModule, 0, 12)
	modules = append(modules, initRefreshSyncModules(deps)...)
	modules = append(modules, initIAMModules(deps)...)
	modules = append(modules, initOrgModules(deps)...)
	modules = append(modules, initConfigModules(deps)...)
	modules = append(modules, initI18nModules(deps)...)
	modules = append(modules, initAuditModules(deps)...)

	// 注册底座模块
	contracts.RegisterBackendModules(r, db, modules...)
	InitGeneratedSystemModules(r, db)
}
