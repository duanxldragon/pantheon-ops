package system

import (
	"pantheon-ops/backend/internal/middleware"
	authsession "pantheon-ops/backend/modules/auth/session"
	audit "pantheon-ops/backend/modules/system/audit"
	dict "pantheon-ops/backend/modules/system/config/dict"
	setting "pantheon-ops/backend/modules/system/config/setting"
	i18n "pantheon-ops/backend/modules/system/i18n"
	menu "pantheon-ops/backend/modules/system/iam/menu"
	permission "pantheon-ops/backend/modules/system/iam/permission"
	role "pantheon-ops/backend/modules/system/iam/role"
	user "pantheon-ops/backend/modules/system/iam/user"
	dept "pantheon-ops/backend/modules/system/org/dept"
	post "pantheon-ops/backend/modules/system/org/post"
	"pantheon-ops/backend/pkg/contracts"
	"pantheon-ops/backend/pkg/database"
	uploadpkg "pantheon-ops/backend/pkg/upload"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type systemModuleDependencies struct {
	db *gorm.DB

	refreshSyncSvc     *RefreshSyncService
	refreshSyncHandler *RefreshSyncHandler

	userSvc     *user.UserService
	userHandler *user.UserHandler

	menuSvc     *menu.MenuService
	menuHandler *menu.MenuHandler

	roleSvc     *role.RoleService
	roleHandler *role.RoleHandler

	permissionSvc     *permission.PermissionService
	permissionHandler *permission.PermissionHandler

	deptSvc     *dept.DeptService
	deptHandler *dept.DeptHandler

	postSvc     *post.PostService
	postHandler *post.PostHandler

	dictSvc     *dict.DictService
	dictHandler *dict.DictHandler

	settingSvc     *setting.SettingService
	settingHandler *setting.SettingHandler

	i18nSvc     *i18n.I18nService
	i18nHandler *i18n.I18nHandler

	auditSvc     *audit.AuditService
	auditHandler *audit.AuditHandler
}

func newSystemModuleDependencies(db *gorm.DB) *systemModuleDependencies {
	refreshSyncSvc := NewRefreshSyncService(db)

	userSvc := user.NewUserService(db, user.WithSessionLifecycle(func(db *gorm.DB) user.SessionLifecycle {
		return authsession.NewLifecycleService(db)
	}))
	menuSvc := menu.NewMenuService(db)
	roleSvc := role.NewRoleService(db)
	permissionSvc := permission.NewPermissionService(db)
	deptSvc := dept.NewDeptService(db)
	postSvc := post.NewPostService(db)
	dictSvc := dict.NewDictService(db)
	settingSvc := setting.NewSettingService(db)
	uploadSvc := uploadpkg.NewService(settingSvc)
	i18nSvc := i18n.NewI18nService(db)
	auditSvc := audit.NewAuditService(db)

	return &systemModuleDependencies{
		db:                 db,
		refreshSyncSvc:     refreshSyncSvc,
		refreshSyncHandler: NewRefreshSyncHandler(refreshSyncSvc),
		userSvc:            userSvc,
		userHandler:        user.NewUserHandler(userSvc),
		menuSvc:            menuSvc,
		menuHandler:        menu.NewMenuHandler(menuSvc),
		roleSvc:            roleSvc,
		roleHandler:        role.NewRoleHandler(roleSvc),
		permissionSvc:      permissionSvc,
		permissionHandler:  permission.NewPermissionHandler(permissionSvc),
		deptSvc:            deptSvc,
		deptHandler:        dept.NewDeptHandler(deptSvc),
		postSvc:            postSvc,
		postHandler:        post.NewPostHandler(postSvc),
		dictSvc:            dictSvc,
		dictHandler:        dict.NewDictHandler(dictSvc),
		settingSvc:         settingSvc,
		settingHandler:     setting.NewSettingHandler(settingSvc, uploadSvc),
		i18nSvc:            i18nSvc,
		i18nHandler:        i18n.NewI18nHandler(i18nSvc),
		auditSvc:           auditSvc,
		auditHandler:       audit.NewAuditHandler(auditSvc),
	}
}

func initRefreshSyncModules(deps *systemModuleDependencies) []contracts.BackendModule {
	return []contracts.BackendModule{
		contracts.FuncModule{
			ModuleName:  "refresh-sync",
			MigrateFunc: func(_ *gorm.DB) error { return deps.refreshSyncSvc.Migrate() },
			Register: func(r *gin.RouterGroup) {
				systemAuth := r.Group("/system").Use(middleware.TokenAuthMiddleware(database.RDB))
				{
					systemAuth.GET("/refresh/state", deps.refreshSyncHandler.GetState)
				}
			},
		},
	}
}

func initIAMModules(deps *systemModuleDependencies) []contracts.BackendModule {
	return []contracts.BackendModule{
		contracts.FuncModule{
			ModuleName:    "user",
			MigrateFunc:   func(_ *gorm.DB) error { return deps.userSvc.Migrate() },
			BootstrapFunc: func(_ *gorm.DB) error { return deps.userSvc.Bootstrap() },
			Register: func(r *gin.RouterGroup) {
				systemProtected := r.Group("/system").Use(middleware.TokenAuthMiddleware(database.RDB)).Use(middleware.CasbinMiddleware()).Use(RefreshSyncMiddleware(deps.refreshSyncSvc))
				systemDataScoped := r.Group("/system").Use(middleware.TokenAuthMiddleware(database.RDB)).Use(middleware.CasbinMiddleware()).Use(middleware.DataScopeMiddleware(deps.db)).Use(RefreshSyncMiddleware(deps.refreshSyncSvc))
				{
					systemProtected.GET("/profile", deps.userHandler.GetProfile)
					systemProtected.PUT("/profile", deps.userHandler.UpdateProfile)
					systemProtected.GET("/user/import-template", deps.userHandler.DownloadImportTemplate)
					systemProtected.GET("/user/:id", deps.userHandler.GetUserDetail)
					systemProtected.POST("/user", deps.userHandler.CreateUser)
					systemProtected.POST("/user/import", deps.userHandler.ImportUsers)
					systemProtected.POST("/user/batch-status", deps.userHandler.BatchUpdateUserStatus)
					systemProtected.POST("/user/batch-delete", middleware.SecureActionMiddleware(), deps.userHandler.BatchDeleteUsers)
					systemProtected.PUT("/user/:id", deps.userHandler.UpdateUser)
					systemProtected.PUT("/user/:id/reset-password", deps.userHandler.ResetPassword)
					systemProtected.DELETE("/user/:id", deps.userHandler.DeleteUser)
				}
				{
					systemDataScoped.GET("/user/list", deps.userHandler.GetUserList)
					systemDataScoped.POST("/user/export", deps.userHandler.ExportUsers)
				}
			},
		},
		contracts.FuncModule{
			ModuleName:    "menu",
			MigrateFunc:   func(_ *gorm.DB) error { return deps.menuSvc.Migrate() },
			SeedMenusFunc: seedMenuModuleMenus,
			Register: func(r *gin.RouterGroup) {
				systemProtected := r.Group("/system").Use(middleware.TokenAuthMiddleware(database.RDB)).Use(middleware.CasbinMiddleware()).Use(RefreshSyncMiddleware(deps.refreshSyncSvc))
				{
					systemProtected.GET("/menu/tree", deps.menuHandler.GetMenuTree)
					systemProtected.POST("/menu", deps.menuHandler.CreateMenu)
					systemProtected.PUT("/menu/:id", deps.menuHandler.UpdateMenu)
					systemProtected.DELETE("/menu/:id", deps.menuHandler.DeleteMenu)
				}
			},
		},
		contracts.FuncModule{
			ModuleName:    "role",
			MigrateFunc:   func(_ *gorm.DB) error { return deps.roleSvc.Migrate() },
			BootstrapFunc: func(_ *gorm.DB) error { return deps.roleSvc.Bootstrap() },
			Register: func(r *gin.RouterGroup) {
				systemProtected := r.Group("/system").Use(middleware.TokenAuthMiddleware(database.RDB)).Use(middleware.CasbinMiddleware()).Use(RefreshSyncMiddleware(deps.refreshSyncSvc))
				{
					systemProtected.GET("/role/list", deps.roleHandler.GetRoleList)
					systemProtected.GET("/role/:id/users", deps.roleHandler.GetRoleMembers)
					systemProtected.GET("/role/:id/user-candidates", deps.roleHandler.GetRoleMemberCandidates)
					systemProtected.POST("/role", deps.roleHandler.CreateRole)
					systemProtected.POST("/role/export", deps.roleHandler.ExportRoles)
					systemProtected.POST("/role/batch-status", deps.roleHandler.BatchUpdateRoleStatus)
					systemProtected.POST("/role/batch-delete", middleware.SecureActionMiddleware(), deps.roleHandler.BatchDeleteRoles)
					systemProtected.POST("/role/:id/users", deps.roleHandler.AddRoleMembers)
					systemProtected.POST("/role/:id/users/remove", deps.roleHandler.RemoveRoleMembers)
					systemProtected.PUT("/role/:id", deps.roleHandler.UpdateRole)
					systemProtected.DELETE("/role/:id", deps.roleHandler.DeleteRole)
				}
			},
		},
		contracts.FuncModule{
			ModuleName: "permission",
			MigrateFunc: func(_ *gorm.DB) error {
				if err := middleware.MigrateDataScopePolicy(deps.db); err != nil {
					return err
				}
				return deps.permissionSvc.Migrate()
			},
			BootstrapFunc: func(_ *gorm.DB) error { return deps.permissionSvc.Bootstrap() },
			SeedMenusFunc: seedPermissionModuleMenus,
			Register: func(r *gin.RouterGroup) {
				systemProtected := r.Group("/system").Use(middleware.TokenAuthMiddleware(database.RDB)).Use(middleware.CasbinMiddleware()).Use(RefreshSyncMiddleware(deps.refreshSyncSvc))
				{
					systemProtected.GET("/permission/workbench", deps.permissionHandler.GetWorkbench)
					systemProtected.GET("/permission/workbench/remediation", deps.permissionHandler.ListWorkbenchRemediationEvents)
					systemProtected.GET("/permission/workbench/export", deps.permissionHandler.ExportWorkbench)
					systemProtected.POST("/permission/workbench/remediate", middleware.SecureActionMiddleware(), deps.permissionHandler.RemediateWorkbenchPolicies)
					systemProtected.GET("/permission/data-scope", deps.permissionHandler.ListDataScopePolicies)
					systemProtected.PUT("/permission/data-scope/:roleKey", middleware.SecureActionMiddleware(), deps.permissionHandler.UpdateDataScopePolicy)
					systemProtected.GET("/permission/list", deps.permissionHandler.GetPolicyList)
					systemProtected.GET("/permission/import-template", deps.permissionHandler.DownloadImportTemplate)
					systemProtected.POST("/permission", deps.permissionHandler.CreatePolicy)
					systemProtected.POST("/permission/export", deps.permissionHandler.ExportPolicies)
					systemProtected.POST("/permission/import", deps.permissionHandler.ImportPolicies)
					systemProtected.POST("/permission/batch-delete", middleware.SecureActionMiddleware(), deps.permissionHandler.BatchDeletePolicies)
					systemProtected.PUT("/permission/:id", deps.permissionHandler.UpdatePolicy)
					systemProtected.DELETE("/permission/:id", deps.permissionHandler.DeletePolicy)
				}
			},
		},
	}
}

func initOrgModules(deps *systemModuleDependencies) []contracts.BackendModule {
	return []contracts.BackendModule{
		contracts.FuncModule{
			ModuleName:    "dept",
			MigrateFunc:   func(_ *gorm.DB) error { return deps.deptSvc.Migrate() },
			BootstrapFunc: func(_ *gorm.DB) error { return deps.deptSvc.Bootstrap() },
			SeedMenusFunc: seedDeptModuleMenus,
			Register: func(r *gin.RouterGroup) {
				systemProtected := r.Group("/system").Use(middleware.TokenAuthMiddleware(database.RDB)).Use(middleware.CasbinMiddleware()).Use(RefreshSyncMiddleware(deps.refreshSyncSvc))
				{
					systemProtected.GET("/dept/overview", deps.deptHandler.GetDeptOverview)
					systemProtected.GET("/dept/governance/tasks", deps.deptHandler.GetGovernanceTasks)
					systemProtected.GET("/dept/tree", deps.deptHandler.GetDeptTree)
					systemProtected.GET("/dept/:id/leader-candidates", deps.deptHandler.GetDeptLeaderCandidates)
					systemProtected.GET("/dept/import-template", deps.deptHandler.DownloadImportTemplate)
					systemProtected.POST("/dept", deps.deptHandler.CreateDept)
					systemProtected.POST("/dept/export", deps.deptHandler.ExportDepts)
					systemProtected.POST("/dept/governance/export", deps.deptHandler.ExportGovernanceTasks)
					systemProtected.POST("/dept/import", deps.deptHandler.ImportDepts)
					systemProtected.POST("/dept/batch-status", deps.deptHandler.BatchUpdateDeptStatus)
					systemProtected.POST("/dept/batch-leader", deps.deptHandler.BatchUpdateDeptLeader)
					systemProtected.POST("/dept/batch-delete", middleware.SecureActionMiddleware(), deps.deptHandler.BatchDeleteDepts)
					systemProtected.PUT("/dept/:id", deps.deptHandler.UpdateDept)
					systemProtected.DELETE("/dept/:id", deps.deptHandler.DeleteDept)
				}
			},
		},
		contracts.FuncModule{
			ModuleName:    "post",
			MigrateFunc:   func(_ *gorm.DB) error { return deps.postSvc.Migrate() },
			BootstrapFunc: func(_ *gorm.DB) error { return deps.postSvc.Bootstrap() },
			SeedMenusFunc: seedPostModuleMenus,
			Register: func(r *gin.RouterGroup) {
				systemProtected := r.Group("/system").Use(middleware.TokenAuthMiddleware(database.RDB)).Use(middleware.CasbinMiddleware()).Use(RefreshSyncMiddleware(deps.refreshSyncSvc))
				{
					systemProtected.GET("/post/list", deps.postHandler.GetPostList)
					systemProtected.GET("/post/import-template", deps.postHandler.DownloadImportTemplate)
					systemProtected.POST("/post", deps.postHandler.CreatePost)
					systemProtected.POST("/post/export", deps.postHandler.ExportPosts)
					systemProtected.POST("/post/import", deps.postHandler.ImportPosts)
					systemProtected.POST("/post/batch-status", deps.postHandler.BatchUpdatePostStatus)
					systemProtected.POST("/post/batch-delete", middleware.SecureActionMiddleware(), deps.postHandler.BatchDeletePosts)
					systemProtected.PUT("/post/:id", deps.postHandler.UpdatePost)
					systemProtected.DELETE("/post/:id", deps.postHandler.DeletePost)
				}
			},
		},
	}
}

func initConfigModules(deps *systemModuleDependencies) []contracts.BackendModule {
	return []contracts.BackendModule{
		contracts.FuncModule{
			ModuleName:    "dict",
			MigrateFunc:   func(_ *gorm.DB) error { return deps.dictSvc.Migrate() },
			BootstrapFunc: func(_ *gorm.DB) error { return deps.dictSvc.Bootstrap() },
			SeedMenusFunc: seedDictModuleMenus,
			Register: func(r *gin.RouterGroup) {
				systemPublic := r.Group("/system")
				{
					systemPublic.GET("/dict/options", deps.dictHandler.GetDictOptions)
				}

				systemProtected := r.Group("/system").Use(middleware.TokenAuthMiddleware(database.RDB)).Use(middleware.CasbinMiddleware())
				{
					systemProtected.GET("/dict/type/list", deps.dictHandler.GetDictTypeList)
					systemProtected.GET("/dict/type/import-template", deps.dictHandler.DownloadDictTypeImportTemplate)
					systemProtected.POST("/dict/type/export", deps.dictHandler.ExportDictTypes)
					systemProtected.POST("/dict/type/import", deps.dictHandler.ImportDictTypes)
					systemProtected.POST("/dict/cache/refresh", RefreshSyncMiddleware(deps.refreshSyncSvc), deps.dictHandler.RefreshDictOptionsCache)
					systemProtected.POST("/dict/type", RefreshSyncMiddleware(deps.refreshSyncSvc), deps.dictHandler.CreateDictType)
					systemProtected.POST("/dict/type/batch-status", RefreshSyncMiddleware(deps.refreshSyncSvc), deps.dictHandler.BatchUpdateDictTypeStatus)
					systemProtected.POST("/dict/type/batch-delete", RefreshSyncMiddleware(deps.refreshSyncSvc), middleware.SecureActionMiddleware(), deps.dictHandler.BatchDeleteDictTypes)
					systemProtected.PUT("/dict/type/:id", RefreshSyncMiddleware(deps.refreshSyncSvc), deps.dictHandler.UpdateDictType)
					systemProtected.DELETE("/dict/type/:id", RefreshSyncMiddleware(deps.refreshSyncSvc), deps.dictHandler.DeleteDictType)
					systemProtected.GET("/dict/item/list", deps.dictHandler.GetDictItemList)
					systemProtected.GET("/dict/usage", deps.dictHandler.AnalyzeDictUsage)
					systemProtected.GET("/dict/item/import-template", deps.dictHandler.DownloadDictItemImportTemplate)
					systemProtected.POST("/dict/item/export", deps.dictHandler.ExportDictItems)
					systemProtected.POST("/dict/item/import", deps.dictHandler.ImportDictItems)
					systemProtected.POST("/dict/item", RefreshSyncMiddleware(deps.refreshSyncSvc), deps.dictHandler.CreateDictItem)
					systemProtected.POST("/dict/item/batch-status", RefreshSyncMiddleware(deps.refreshSyncSvc), deps.dictHandler.BatchUpdateDictItemStatus)
					systemProtected.POST("/dict/item/batch-delete", RefreshSyncMiddleware(deps.refreshSyncSvc), middleware.SecureActionMiddleware(), deps.dictHandler.BatchDeleteDictItems)
					systemProtected.PUT("/dict/item/:id", RefreshSyncMiddleware(deps.refreshSyncSvc), deps.dictHandler.UpdateDictItem)
					systemProtected.PUT("/dict/item/:id/reorder", RefreshSyncMiddleware(deps.refreshSyncSvc), deps.dictHandler.ReorderDictItem)
					systemProtected.DELETE("/dict/item/:id", RefreshSyncMiddleware(deps.refreshSyncSvc), deps.dictHandler.DeleteDictItem)
				}
			},
		},
		contracts.FuncModule{
			ModuleName:    "setting",
			MigrateFunc:   func(_ *gorm.DB) error { return deps.settingSvc.Migrate() },
			BootstrapFunc: func(_ *gorm.DB) error { return deps.settingSvc.Bootstrap() },
			SeedMenusFunc: seedSettingModuleMenus,
			Register: func(r *gin.RouterGroup) {
				systemPublic := r.Group("/system")
				{
					systemPublic.GET("/setting/public", deps.settingHandler.GetPublicSettings)
					systemPublic.GET("/upload/files/*filepath", deps.settingHandler.ServeUploadedFile)
				}

				systemAuth := r.Group("/system").Use(middleware.TokenAuthMiddleware(database.RDB))
				{
					systemAuth.POST("/upload", RefreshSyncMiddleware(deps.refreshSyncSvc), deps.settingHandler.UploadFile)
				}

				systemProtected := r.Group("/system").Use(middleware.TokenAuthMiddleware(database.RDB)).Use(middleware.CasbinMiddleware())
				{
					systemProtected.GET("/setting/overview", deps.settingHandler.GetSettingOverview)
					systemProtected.GET("/setting/list", deps.settingHandler.GetSettingList)
					systemProtected.GET("/setting/audit/list", deps.settingHandler.GetSettingAuditList)
					systemProtected.POST("/setting/audit/export", deps.settingHandler.ExportSettingAudit)
					systemProtected.POST("/setting/cache/refresh", RefreshSyncMiddleware(deps.refreshSyncSvc), deps.settingHandler.RefreshSettingCache)
					systemProtected.GET("/setting/group/:groupKey", deps.settingHandler.GetSettingGroup)
					systemProtected.PUT("/setting/group/:groupKey", RefreshSyncMiddleware(deps.refreshSyncSvc), middleware.SecureActionMiddleware(), deps.settingHandler.UpdateSettingGroup)
				}
			},
		},
	}
}

func initI18nModules(deps *systemModuleDependencies) []contracts.BackendModule {
	return []contracts.BackendModule{
		contracts.FuncModule{
			ModuleName:    "i18n",
			MigrateFunc:   func(_ *gorm.DB) error { return deps.i18nSvc.Migrate() },
			BootstrapFunc: func(_ *gorm.DB) error { return deps.i18nSvc.Bootstrap() },
			SeedMenusFunc: seedI18nModuleMenus,
			SeedI18nFunc:  func(db *gorm.DB) error { return deps.i18nSvc.SeedI18nModuleI18n(db) },
			Register: func(r *gin.RouterGroup) {
				sysPublic := r.Group("/system")
				{
					sysPublic.GET("/i18n/pack", deps.i18nHandler.GetLangPack)
				}

				sysProtected := r.Group("/system").Use(middleware.TokenAuthMiddleware(database.RDB)).Use(middleware.CasbinMiddleware())
				{
					sysProtected.GET("/i18n/overview", deps.i18nHandler.GetOverview)
					sysProtected.GET("/i18n/audit", deps.i18nHandler.GetAudit)
					sysProtected.GET("/i18n/missing-locales", deps.i18nHandler.GetMissingLocales)
					sysProtected.POST("/i18n/cleanup-unused", RefreshSyncMiddleware(deps.refreshSyncSvc), deps.i18nHandler.CleanupUnusedKeys)
					sysProtected.POST("/i18n/lifecycle/observe", RefreshSyncMiddleware(deps.refreshSyncSvc), deps.i18nHandler.StartUnusedObservation)
					sysProtected.POST("/i18n/lifecycle/archive", RefreshSyncMiddleware(deps.refreshSyncSvc), deps.i18nHandler.ArchiveObservedUnusedKeys)
					sysProtected.POST("/i18n/lifecycle/delete", RefreshSyncMiddleware(deps.refreshSyncSvc), deps.i18nHandler.DeleteArchivedUnusedKeys)
					sysProtected.POST("/i18n/rename/preview", deps.i18nHandler.PreviewRenameKey)
					sysProtected.POST("/i18n/rename", RefreshSyncMiddleware(deps.refreshSyncSvc), deps.i18nHandler.RenameKey)
					sysProtected.POST("/i18n/fill-missing-locales", RefreshSyncMiddleware(deps.refreshSyncSvc), deps.i18nHandler.FillMissingLocales)
					sysProtected.POST("/i18n/hydrate-builtin-locales", RefreshSyncMiddleware(deps.refreshSyncSvc), deps.i18nHandler.HydrateBuiltinLocales)
					sysProtected.POST("/i18n", RefreshSyncMiddleware(deps.refreshSyncSvc), deps.i18nHandler.Create)
					sysProtected.GET("/i18n/list", deps.i18nHandler.List)
					sysProtected.GET("/i18n/import-template", deps.i18nHandler.DownloadImportTemplate)
					sysProtected.GET("/i18n/:id", deps.i18nHandler.Get)
					sysProtected.PUT("/i18n/:id", RefreshSyncMiddleware(deps.refreshSyncSvc), deps.i18nHandler.Update)
					sysProtected.DELETE("/i18n/:id", RefreshSyncMiddleware(deps.refreshSyncSvc), deps.i18nHandler.Delete)
					sysProtected.POST("/i18n/batch-delete", RefreshSyncMiddleware(deps.refreshSyncSvc), deps.i18nHandler.DeleteBatch)
					sysProtected.POST("/i18n/export", deps.i18nHandler.Export)
					sysProtected.POST("/i18n/import", RefreshSyncMiddleware(deps.refreshSyncSvc), deps.i18nHandler.Import)
					sysProtected.POST("/i18n/cache/refresh", RefreshSyncMiddleware(deps.refreshSyncSvc), deps.i18nHandler.ReloadCache)
					sysProtected.POST("/i18n/sync-keys", RefreshSyncMiddleware(deps.refreshSyncSvc), deps.i18nHandler.SyncMissingKeys)
				}
			},
		},
	}
}

func initAuditModules(deps *systemModuleDependencies) []contracts.BackendModule {
	return []contracts.BackendModule{
		contracts.FuncModule{
			ModuleName:    "audit",
			MigrateFunc:   func(_ *gorm.DB) error { return deps.auditSvc.Migrate() },
			BootstrapFunc: func(_ *gorm.DB) error { return deps.auditSvc.Bootstrap() },
			SeedMenusFunc: seedAuditModuleMenus,
			Register: func(r *gin.RouterGroup) {
				systemProtected := r.Group("/system").Use(middleware.TokenAuthMiddleware(database.RDB)).Use(middleware.CasbinMiddleware())
				{
					systemProtected.GET("/operation-log/list", deps.auditHandler.GetOperationLogList)
					systemProtected.GET("/operation-log/:id", deps.auditHandler.GetOperationLog)
					systemProtected.POST("/operation-log/export", deps.auditHandler.ExportOperationLogs)
					systemProtected.DELETE("/operation-log/:id", deps.auditHandler.DeleteOperationLog)
					systemProtected.POST("/operation-log/cleanup", middleware.SecureActionMiddleware(), deps.auditHandler.CleanupOperationLogs)
					systemProtected.POST("/operation-log/batch-delete", middleware.SecureActionMiddleware(), deps.auditHandler.BatchDeleteOperationLogs)
				}
			},
		},
	}
}
