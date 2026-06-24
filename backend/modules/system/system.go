package system

import (
	"pantheon-ops/backend/internal/middleware"

	audit "pantheon-ops/backend/modules/system/audit"
	dict "pantheon-ops/backend/modules/system/config/dict"
	setting "pantheon-ops/backend/modules/system/config/setting"
	"pantheon-ops/backend/modules/system/dynamicmodule"
	generator "pantheon-ops/backend/modules/system/generator"
	i18n "pantheon-ops/backend/modules/system/i18n"
	menu "pantheon-ops/backend/modules/system/iam/menu"
	permission "pantheon-ops/backend/modules/system/iam/permission"
	role "pantheon-ops/backend/modules/system/iam/role"
	user "pantheon-ops/backend/modules/system/iam/user"
	dept "pantheon-ops/backend/modules/system/org/dept"
	post "pantheon-ops/backend/modules/system/org/post"
	"pantheon-ops/backend/pkg/contracts"
	uploadpkg "pantheon-ops/backend/pkg/upload"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// InitSystemModule 初始化系统模块
func InitSystemModule(r *gin.RouterGroup, db *gorm.DB) {
	// 清理历史废弃菜单（/workspace、/operations 等）
	if err := CleanupObsoleteMenus(db); err != nil {
		panic(err)
	}

	// 用户模块注入
	userSvc := user.NewUserService(db)
	userHandler := user.NewUserHandler(userSvc)

	// 菜单模块注入
	menuSvc := menu.NewMenuService(db)
	menuHandler := menu.NewMenuHandler(menuSvc)

	// 部门模块注入
	deptSvc := dept.NewDeptService(db)
	deptHandler := dept.NewDeptHandler(deptSvc)

	// 字典模块注入
	dictSvc := dict.NewDictService(db)
	dictHandler := dict.NewDictHandler(dictSvc)

	// 岗位模块注入
	postSvc := post.NewPostService(db)
	postHandler := post.NewPostHandler(postSvc)

	// 权限模块注入
	permissionSvc := permission.NewPermissionService(db)
	permissionHandler := permission.NewPermissionHandler(permissionSvc)

	// 角色模块注入
	roleSvc := role.NewRoleService(db)
	roleHandler := role.NewRoleHandler(roleSvc)

	// 设置模块注入
	settingSvc := setting.NewSettingService(db)
	uploadSvc := uploadpkg.NewService(settingSvc)
	settingHandler := setting.NewSettingHandler(settingSvc, uploadSvc)

	// 平台刷新同步注入
	refreshSyncSvc := NewRefreshSyncService(db)
	refreshSyncHandler := NewRefreshSyncHandler(refreshSyncSvc)

	// 多语言模块注入
	i18nSvc := i18n.NewI18nService(db)
	i18nHandler := i18n.NewI18nHandler(i18nSvc)

	// 审计模块注入
	auditSvc := audit.NewAuditService(db)
	auditHandler := audit.NewAuditHandler(auditSvc)

	modules := []contracts.BackendModule{
		contracts.FuncModule{
			ModuleName:  "refresh-sync",
			MigrateFunc: func(_ *gorm.DB) error { return refreshSyncSvc.Migrate() },
			Register: func(r *gin.RouterGroup) {
				systemAuth := r.Group("/system").Use(middleware.JWTAuthMiddleware())
				{
					systemAuth.GET("/refresh/state", refreshSyncHandler.GetState)
				}
			},
		},
		contracts.FuncModule{
			ModuleName:    "user",
			MigrateFunc:   func(_ *gorm.DB) error { return userSvc.Migrate() },
			BootstrapFunc: func(_ *gorm.DB) error { return userSvc.Bootstrap() },
			Register: func(r *gin.RouterGroup) {
				systemProtected := r.Group("/system").Use(middleware.JWTAuthMiddleware()).Use(middleware.CasbinMiddleware()).Use(RefreshSyncMiddleware(refreshSyncSvc))
				systemDataScoped := r.Group("/system").Use(middleware.JWTAuthMiddleware()).Use(middleware.CasbinMiddleware()).Use(middleware.DataScopeMiddleware(db)).Use(RefreshSyncMiddleware(refreshSyncSvc))
				{
					systemProtected.GET("/profile", userHandler.GetProfile)
					systemProtected.PUT("/profile", userHandler.UpdateProfile)
					systemProtected.GET("/user/import-template", userHandler.DownloadImportTemplate)
					systemProtected.GET("/user/:id", userHandler.GetUserDetail)
					systemProtected.POST("/user", userHandler.CreateUser)
					systemProtected.POST("/user/import", userHandler.ImportUsers)
					systemProtected.POST("/user/batch-status", userHandler.BatchUpdateUserStatus)
					systemProtected.POST("/user/batch-delete", middleware.SecureActionMiddleware(), userHandler.BatchDeleteUsers)
					systemProtected.PUT("/user/:id", userHandler.UpdateUser)
					systemProtected.PUT("/user/:id/reset-password", userHandler.ResetPassword)
					systemProtected.DELETE("/user/:id", userHandler.DeleteUser)
				}
				{
					systemDataScoped.GET("/user/list", userHandler.GetUserList)
					systemDataScoped.POST("/user/export", userHandler.ExportUsers)
				}
			},
		},
		contracts.FuncModule{
			ModuleName:    "menu",
			MigrateFunc:   func(_ *gorm.DB) error { return menuSvc.Migrate() },
			SeedMenusFunc: seedMenuModuleMenus,
			Register: func(r *gin.RouterGroup) {
				systemProtected := r.Group("/system").Use(middleware.JWTAuthMiddleware()).Use(middleware.CasbinMiddleware()).Use(RefreshSyncMiddleware(refreshSyncSvc))
				{
					systemProtected.GET("/menu/tree", menuHandler.GetMenuTree)
					systemProtected.POST("/menu", menuHandler.CreateMenu)
					systemProtected.PUT("/menu/:id", menuHandler.UpdateMenu)
					systemProtected.DELETE("/menu/:id", menuHandler.DeleteMenu)
				}
			},
		},
		contracts.FuncModule{
			ModuleName:    "role",
			MigrateFunc:   func(_ *gorm.DB) error { return roleSvc.Migrate() },
			BootstrapFunc: func(_ *gorm.DB) error { return roleSvc.Bootstrap() },
			Register: func(r *gin.RouterGroup) {
				systemProtected := r.Group("/system").Use(middleware.JWTAuthMiddleware()).Use(middleware.CasbinMiddleware()).Use(RefreshSyncMiddleware(refreshSyncSvc))
				{
					systemProtected.GET("/role/list", roleHandler.GetRoleList)
					systemProtected.GET("/role/:id/users", roleHandler.GetRoleMembers)
					systemProtected.GET("/role/:id/user-candidates", roleHandler.GetRoleMemberCandidates)
					systemProtected.POST("/role", roleHandler.CreateRole)
					systemProtected.POST("/role/export", roleHandler.ExportRoles)
					systemProtected.POST("/role/batch-status", roleHandler.BatchUpdateRoleStatus)
					systemProtected.POST("/role/batch-delete", middleware.SecureActionMiddleware(), roleHandler.BatchDeleteRoles)
					systemProtected.POST("/role/:id/users", roleHandler.AddRoleMembers)
					systemProtected.POST("/role/:id/users/remove", roleHandler.RemoveRoleMembers)
					systemProtected.PUT("/role/:id", roleHandler.UpdateRole)
					systemProtected.DELETE("/role/:id", roleHandler.DeleteRole)
				}
			},
		},
		contracts.FuncModule{
			ModuleName:    "dept",
			MigrateFunc:   func(_ *gorm.DB) error { return deptSvc.Migrate() },
			BootstrapFunc: func(_ *gorm.DB) error { return deptSvc.Bootstrap() },
			SeedMenusFunc: seedDeptModuleMenus,
			Register: func(r *gin.RouterGroup) {
				systemProtected := r.Group("/system").Use(middleware.JWTAuthMiddleware()).Use(middleware.CasbinMiddleware()).Use(RefreshSyncMiddleware(refreshSyncSvc))
				{
					systemProtected.GET("/dept/overview", deptHandler.GetDeptOverview)
					systemProtected.GET("/dept/governance/tasks", deptHandler.GetGovernanceTasks)
					systemProtected.GET("/dept/tree", deptHandler.GetDeptTree)
					systemProtected.GET("/dept/:id/leader-candidates", deptHandler.GetDeptLeaderCandidates)
					systemProtected.GET("/dept/import-template", deptHandler.DownloadImportTemplate)
					systemProtected.POST("/dept", deptHandler.CreateDept)
					systemProtected.POST("/dept/export", deptHandler.ExportDepts)
					systemProtected.POST("/dept/governance/export", deptHandler.ExportGovernanceTasks)
					systemProtected.POST("/dept/import", deptHandler.ImportDepts)
					systemProtected.POST("/dept/batch-status", deptHandler.BatchUpdateDeptStatus)
					systemProtected.POST("/dept/batch-leader", deptHandler.BatchUpdateDeptLeader)
					systemProtected.POST("/dept/batch-delete", middleware.SecureActionMiddleware(), deptHandler.BatchDeleteDepts)
					systemProtected.PUT("/dept/:id", deptHandler.UpdateDept)
					systemProtected.DELETE("/dept/:id", deptHandler.DeleteDept)
				}
			},
		},
		contracts.FuncModule{
			ModuleName:    "post",
			MigrateFunc:   func(_ *gorm.DB) error { return postSvc.Migrate() },
			BootstrapFunc: func(_ *gorm.DB) error { return postSvc.Bootstrap() },
			SeedMenusFunc: seedPostModuleMenus,
			Register: func(r *gin.RouterGroup) {
				systemProtected := r.Group("/system").Use(middleware.JWTAuthMiddleware()).Use(middleware.CasbinMiddleware()).Use(RefreshSyncMiddleware(refreshSyncSvc))
				{
					systemProtected.GET("/post/list", postHandler.GetPostList)
					systemProtected.GET("/post/import-template", postHandler.DownloadImportTemplate)
					systemProtected.POST("/post", postHandler.CreatePost)
					systemProtected.POST("/post/export", postHandler.ExportPosts)
					systemProtected.POST("/post/import", postHandler.ImportPosts)
					systemProtected.POST("/post/batch-status", postHandler.BatchUpdatePostStatus)
					systemProtected.POST("/post/batch-delete", middleware.SecureActionMiddleware(), postHandler.BatchDeletePosts)
					systemProtected.PUT("/post/:id", postHandler.UpdatePost)
					systemProtected.DELETE("/post/:id", postHandler.DeletePost)
				}
			},
		},
		contracts.FuncModule{
			ModuleName: "permission",
			MigrateFunc: func(_ *gorm.DB) error {
				if err := middleware.MigrateDataScopePolicy(db); err != nil {
					return err
				}
				return permissionSvc.Migrate()
			},
			BootstrapFunc: func(_ *gorm.DB) error { return permissionSvc.Bootstrap() },
			SeedMenusFunc: seedPermissionModuleMenus,
			Register: func(r *gin.RouterGroup) {
				systemProtected := r.Group("/system").Use(middleware.JWTAuthMiddleware()).Use(middleware.CasbinMiddleware()).Use(RefreshSyncMiddleware(refreshSyncSvc))
				{
					systemProtected.GET("/permission/workbench", permissionHandler.GetWorkbench)
					systemProtected.GET("/permission/workbench/remediation", permissionHandler.ListWorkbenchRemediationEvents)
					systemProtected.GET("/permission/workbench/export", permissionHandler.ExportWorkbench)
					systemProtected.POST("/permission/workbench/remediate", middleware.SecureActionMiddleware(), permissionHandler.RemediateWorkbenchPolicies)
					systemProtected.GET("/permission/data-scope", permissionHandler.ListDataScopePolicies)
					systemProtected.PUT("/permission/data-scope/:roleKey", middleware.SecureActionMiddleware(), permissionHandler.UpdateDataScopePolicy)
					systemProtected.GET("/permission/list", permissionHandler.GetPolicyList)
					systemProtected.GET("/permission/import-template", permissionHandler.DownloadImportTemplate)
					systemProtected.POST("/permission", permissionHandler.CreatePolicy)
					systemProtected.POST("/permission/export", permissionHandler.ExportPolicies)
					systemProtected.POST("/permission/import", permissionHandler.ImportPolicies)
					systemProtected.POST("/permission/batch-delete", middleware.SecureActionMiddleware(), permissionHandler.BatchDeletePolicies)
					systemProtected.PUT("/permission/:id", permissionHandler.UpdatePolicy)
					systemProtected.DELETE("/permission/:id", permissionHandler.DeletePolicy)
				}
			},
		},
		contracts.FuncModule{
			ModuleName:    "dict",
			MigrateFunc:   func(_ *gorm.DB) error { return dictSvc.Migrate() },
			BootstrapFunc: func(_ *gorm.DB) error { return dictSvc.Bootstrap() },
			SeedMenusFunc: seedDictModuleMenus,
			Register: func(r *gin.RouterGroup) {
				systemPublic := r.Group("/system")
				{
					systemPublic.GET("/dict/options", dictHandler.GetDictOptions)
				}

				systemProtected := r.Group("/system").Use(middleware.JWTAuthMiddleware()).Use(middleware.CasbinMiddleware())
				{
					systemProtected.GET("/dict/type/list", dictHandler.GetDictTypeList)
					systemProtected.GET("/dict/type/import-template", dictHandler.DownloadDictTypeImportTemplate)
					systemProtected.POST("/dict/type/export", dictHandler.ExportDictTypes)
					systemProtected.POST("/dict/type/import", dictHandler.ImportDictTypes)
					systemProtected.POST("/dict/cache/refresh", RefreshSyncMiddleware(refreshSyncSvc), dictHandler.RefreshDictOptionsCache)
					systemProtected.POST("/dict/type", RefreshSyncMiddleware(refreshSyncSvc), dictHandler.CreateDictType)
					systemProtected.POST("/dict/type/batch-status", RefreshSyncMiddleware(refreshSyncSvc), dictHandler.BatchUpdateDictTypeStatus)
					systemProtected.POST("/dict/type/batch-delete", RefreshSyncMiddleware(refreshSyncSvc), middleware.SecureActionMiddleware(), dictHandler.BatchDeleteDictTypes)
					systemProtected.PUT("/dict/type/:id", RefreshSyncMiddleware(refreshSyncSvc), dictHandler.UpdateDictType)
					systemProtected.DELETE("/dict/type/:id", RefreshSyncMiddleware(refreshSyncSvc), dictHandler.DeleteDictType)
					systemProtected.GET("/dict/item/list", dictHandler.GetDictItemList)
					systemProtected.GET("/dict/usage", dictHandler.AnalyzeDictUsage)
					systemProtected.GET("/dict/item/import-template", dictHandler.DownloadDictItemImportTemplate)
					systemProtected.POST("/dict/item/export", dictHandler.ExportDictItems)
					systemProtected.POST("/dict/item/import", dictHandler.ImportDictItems)
					systemProtected.POST("/dict/item", RefreshSyncMiddleware(refreshSyncSvc), dictHandler.CreateDictItem)
					systemProtected.POST("/dict/item/batch-status", RefreshSyncMiddleware(refreshSyncSvc), dictHandler.BatchUpdateDictItemStatus)
					systemProtected.POST("/dict/item/batch-delete", RefreshSyncMiddleware(refreshSyncSvc), middleware.SecureActionMiddleware(), dictHandler.BatchDeleteDictItems)
					systemProtected.PUT("/dict/item/:id", RefreshSyncMiddleware(refreshSyncSvc), dictHandler.UpdateDictItem)
					systemProtected.PUT("/dict/item/:id/reorder", RefreshSyncMiddleware(refreshSyncSvc), dictHandler.ReorderDictItem)
					systemProtected.DELETE("/dict/item/:id", RefreshSyncMiddleware(refreshSyncSvc), dictHandler.DeleteDictItem)
				}
			},
		},
		contracts.FuncModule{
			ModuleName:    "setting",
			MigrateFunc:   func(_ *gorm.DB) error { return settingSvc.Migrate() },
			BootstrapFunc: func(_ *gorm.DB) error { return settingSvc.Bootstrap() },
			SeedMenusFunc: seedSettingModuleMenus,
			Register: func(r *gin.RouterGroup) {
				systemPublic := r.Group("/system")
				{
					systemPublic.GET("/setting/public", settingHandler.GetPublicSettings)
					systemPublic.GET("/upload/files/*filepath", settingHandler.ServeUploadedFile)
				}

				systemAuth := r.Group("/system").Use(middleware.JWTAuthMiddleware())
				{
					systemAuth.POST("/upload", RefreshSyncMiddleware(refreshSyncSvc), settingHandler.UploadFile)
				}

				systemProtected := r.Group("/system").Use(middleware.JWTAuthMiddleware()).Use(middleware.CasbinMiddleware())
				{
					systemProtected.GET("/setting/overview", settingHandler.GetSettingOverview)
					systemProtected.GET("/setting/list", settingHandler.GetSettingList)
					systemProtected.GET("/setting/audit/list", settingHandler.GetSettingAuditList)
					systemProtected.POST("/setting/audit/export", settingHandler.ExportSettingAudit)
					systemProtected.POST("/setting/cache/refresh", RefreshSyncMiddleware(refreshSyncSvc), settingHandler.RefreshSettingCache)
					systemProtected.GET("/setting/group/:groupKey", settingHandler.GetSettingGroup)
					systemProtected.PUT("/setting/group/:groupKey", RefreshSyncMiddleware(refreshSyncSvc), middleware.SecureActionMiddleware(), settingHandler.UpdateSettingGroup)
				}
			},
		},
		contracts.FuncModule{
			ModuleName:    "i18n",
			MigrateFunc:   func(_ *gorm.DB) error { return i18nSvc.Migrate() },
			BootstrapFunc: func(_ *gorm.DB) error { return i18nSvc.Bootstrap() },
			SeedMenusFunc: seedI18nModuleMenus,
			SeedI18nFunc:  func(db *gorm.DB) error { return i18nSvc.SeedI18nModuleI18n(db) },
			Register: func(r *gin.RouterGroup) {
				sysPublic := r.Group("/system")
				{
					sysPublic.GET("/i18n/pack", i18nHandler.GetLangPack)
				}

				sysProtected := r.Group("/system").Use(middleware.JWTAuthMiddleware()).Use(middleware.CasbinMiddleware())
				{
					sysProtected.GET("/i18n/overview", i18nHandler.GetOverview)
					sysProtected.GET("/i18n/audit", i18nHandler.GetAudit)
					sysProtected.GET("/i18n/missing-locales", i18nHandler.GetMissingLocales)
					sysProtected.POST("/i18n/cleanup-unused", RefreshSyncMiddleware(refreshSyncSvc), i18nHandler.CleanupUnusedKeys)
					sysProtected.POST("/i18n/lifecycle/observe", RefreshSyncMiddleware(refreshSyncSvc), i18nHandler.StartUnusedObservation)
					sysProtected.POST("/i18n/lifecycle/archive", RefreshSyncMiddleware(refreshSyncSvc), i18nHandler.ArchiveObservedUnusedKeys)
					sysProtected.POST("/i18n/lifecycle/delete", RefreshSyncMiddleware(refreshSyncSvc), i18nHandler.DeleteArchivedUnusedKeys)
					sysProtected.POST("/i18n/rename/preview", i18nHandler.PreviewRenameKey)
					sysProtected.POST("/i18n/rename", RefreshSyncMiddleware(refreshSyncSvc), i18nHandler.RenameKey)
					sysProtected.POST("/i18n/fill-missing-locales", RefreshSyncMiddleware(refreshSyncSvc), i18nHandler.FillMissingLocales)
					sysProtected.POST("/i18n/hydrate-builtin-locales", RefreshSyncMiddleware(refreshSyncSvc), i18nHandler.HydrateBuiltinLocales)
					sysProtected.POST("/i18n", RefreshSyncMiddleware(refreshSyncSvc), i18nHandler.Create)
					sysProtected.GET("/i18n/list", i18nHandler.List)
					sysProtected.GET("/i18n/import-template", i18nHandler.DownloadImportTemplate)
					sysProtected.GET("/i18n/:id", i18nHandler.Get)
					sysProtected.PUT("/i18n/:id", RefreshSyncMiddleware(refreshSyncSvc), i18nHandler.Update)
					sysProtected.DELETE("/i18n/:id", RefreshSyncMiddleware(refreshSyncSvc), i18nHandler.Delete)
					sysProtected.POST("/i18n/batch-delete", RefreshSyncMiddleware(refreshSyncSvc), i18nHandler.DeleteBatch)
					sysProtected.POST("/i18n/export", i18nHandler.Export)
					sysProtected.POST("/i18n/import", RefreshSyncMiddleware(refreshSyncSvc), i18nHandler.Import)
					sysProtected.POST("/i18n/cache/refresh", RefreshSyncMiddleware(refreshSyncSvc), i18nHandler.ReloadCache)
					sysProtected.POST("/i18n/sync-keys", RefreshSyncMiddleware(refreshSyncSvc), i18nHandler.SyncMissingKeys)
				}
			},
		},
		contracts.FuncModule{
			ModuleName:    "audit",
			MigrateFunc:   func(_ *gorm.DB) error { return auditSvc.Migrate() },
			BootstrapFunc: func(_ *gorm.DB) error { return auditSvc.Bootstrap() },
			SeedMenusFunc: seedAuditModuleMenus,
			Register: func(r *gin.RouterGroup) {
				systemProtected := r.Group("/system").Use(middleware.JWTAuthMiddleware()).Use(middleware.CasbinMiddleware())
				{
					systemProtected.GET("/operation-log/list", auditHandler.GetOperationLogList)
					systemProtected.GET("/operation-log/:id", auditHandler.GetOperationLog)
					systemProtected.POST("/operation-log/export", auditHandler.ExportOperationLogs)
					systemProtected.DELETE("/operation-log/:id", auditHandler.DeleteOperationLog)
					systemProtected.POST("/operation-log/cleanup", middleware.SecureActionMiddleware(), auditHandler.CleanupOperationLogs)
					systemProtected.POST("/operation-log/batch-delete", middleware.SecureActionMiddleware(), auditHandler.BatchDeleteOperationLogs)
				}
			},
		},
	}

	// 注册动态模块管理
	dynamicmodule.InitDynamicModule(r, db)
	generator.InitGeneratorModule(r, db)

	contracts.RegisterBackendModules(r, db, modules...)
	InitGeneratedSystemModules(r, db)
}
