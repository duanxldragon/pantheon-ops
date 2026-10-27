package system

import (
	"time"

	"github.com/duanxldragon/pantheon-base/backend/internal/middleware"
	"github.com/duanxldragon/pantheon-base/backend/modules/auth/session"
	"github.com/duanxldragon/pantheon-base/backend/modules/system/audit"
	"github.com/duanxldragon/pantheon-base/backend/modules/system/config/dict"
	"github.com/duanxldragon/pantheon-base/backend/modules/system/config/setting"
	"github.com/duanxldragon/pantheon-base/backend/modules/system/i18n"
	"github.com/duanxldragon/pantheon-base/backend/modules/system/iam/menu"
	"github.com/duanxldragon/pantheon-base/backend/modules/system/iam/permission"
	"github.com/duanxldragon/pantheon-base/backend/modules/system/iam/role"
	tenantiam "github.com/duanxldragon/pantheon-base/backend/modules/system/iam/tenant"
	"github.com/duanxldragon/pantheon-base/backend/modules/system/iam/user"
	"github.com/duanxldragon/pantheon-base/backend/modules/system/org/dept"
	"github.com/duanxldragon/pantheon-base/backend/modules/system/org/post"
	"github.com/duanxldragon/pantheon-base/backend/pkg/contracts"
	"github.com/duanxldragon/pantheon-base/backend/pkg/database"
	"github.com/duanxldragon/pantheon-base/backend/pkg/maintenance"
	"github.com/duanxldragon/pantheon-base/backend/pkg/tenant"
	uploadpkg "github.com/duanxldragon/pantheon-base/backend/pkg/upload"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const (
	routeGroupSystem = "/system"
	routeUserByID    = "/user/:id"
	routeI18nByID    = "/i18n/:id"
)

// tenantModeLoaderTTL bounds staleness of the cached tenant mode flag.
const tenantModeLoaderTTL = 5 * time.Second

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

	tenantSvc     *tenantiam.Service
	tenantHandler *tenantiam.PlatformHandler

	deptSvc     *dept.DeptService
	deptHandler *dept.DeptHandler

	postSvc     *post.PostService
	postHandler *post.PostHandler

	dictSvc     *dict.DictService
	dictHandler *dict.DictHandler

	// tenantModeLoader drives the dict canary flag (nil-safe middleware).
	tenantModeLoader *tenant.ModeLoader

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
		return session.NewLifecycleService(db)
	}))
	menuSvc := menu.NewMenuService(db)
	roleSvc := role.NewRoleService(db)
	permissionSvc := permission.NewPermissionService(db)
	tenantSvc := tenantiam.NewService(db)
	deptSvc := dept.NewDeptService(db)
	postSvc := post.NewPostService(db)
	dictSvc := dict.NewDictService(db)
	settingSvc := setting.NewSettingService(db)
	// Canary flag loader (contract §6): cached read of platform.tenant_mode.
	tenantModeLoader := tenant.NewSettingModeLoader(settingSvc, int64(tenantModeLoaderTTL))
	uploadSvc := uploadpkg.NewService(settingSvc)
	i18nSvc := i18n.NewI18nService(db)
	auditSvc := audit.NewAuditService(db)
	// Register the operation-log retention sweep with the background
	// maintenance runner instead of purging inline on list/export requests
	// (task 2026-09-22-request-path-maintenance).
	auditSvc.RegisterMaintenanceTasks(maintenance.Default())

	return &systemModuleDependencies{
		db:                 db,
		refreshSyncSvc:     refreshSyncSvc,
		refreshSyncHandler: NewRefreshSyncHandler(refreshSyncSvc),
		tenantModeLoader:   tenantModeLoader,
		userSvc:            userSvc,
		userHandler:        user.NewUserHandler(userSvc),
		menuSvc:            menuSvc,
		menuHandler:        menu.NewMenuHandler(menuSvc),
		roleSvc:            roleSvc,
		roleHandler:        role.NewRoleHandler(roleSvc),
		permissionSvc:      permissionSvc,
		permissionHandler:  permission.NewPermissionHandler(permissionSvc),
		tenantSvc:          tenantSvc,
		tenantHandler:      tenantiam.NewPlatformHandler(tenantSvc),
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
				systemAuth := r.Group(routeGroupSystem).Use(middleware.TokenAuthMiddleware(database.RDB))
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
				// The IAM console is shared by platform operators (tenant 0) and
				// tenant administrators. Declaring the group platform-scoped is
				// what lets a claim-less platform operator resolve to the
				// platform-global context in multi mode, while a tenant subject
				// still resolves to its own tenant claim (contract §3.1/§3.3).
				systemProtected := r.Group(routeGroupSystem).Use(middleware.TokenAuthMiddleware(database.RDB)).Use(middleware.PlatformScopeTenantContextMiddleware(deps.tenantModeLoader, deps.db)).Use(middleware.CasbinMiddleware()).Use(RefreshSyncMiddleware(deps.refreshSyncSvc))
				systemDataScoped := r.Group(routeGroupSystem).Use(middleware.TokenAuthMiddleware(database.RDB)).Use(middleware.PlatformScopeTenantContextMiddleware(deps.tenantModeLoader, deps.db)).Use(middleware.CasbinMiddleware()).Use(middleware.DataScopeMiddleware(deps.db)).Use(RefreshSyncMiddleware(deps.refreshSyncSvc))
				{
					systemProtected.GET("/profile", deps.userHandler.GetProfile)
					systemProtected.PUT("/profile", deps.userHandler.UpdateProfile)
					systemProtected.GET("/user/import-template", deps.userHandler.DownloadImportTemplate)
					systemProtected.GET(routeUserByID, deps.userHandler.GetUserDetail)
					systemProtected.POST("/user", deps.userHandler.CreateUser)
					systemProtected.POST("/user/import", deps.userHandler.ImportUsers)
					systemProtected.POST("/user/batch-status", deps.userHandler.BatchUpdateUserStatus)
					systemProtected.POST("/user/batch-delete", middleware.SecureActionMiddleware(), deps.userHandler.BatchDeleteUsers)
					systemProtected.PUT(routeUserByID, deps.userHandler.UpdateUser)
					systemProtected.PUT("/user/:id/reset-password", middleware.SecureActionMiddleware(), deps.userHandler.ResetPassword)
					systemProtected.DELETE(routeUserByID, middleware.SecureActionMiddleware(), deps.userHandler.DeleteUser)
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
				systemProtected := r.Group(routeGroupSystem).Use(middleware.TokenAuthMiddleware(database.RDB)).Use(middleware.PlatformScopeTenantContextMiddleware(deps.tenantModeLoader, deps.db)).Use(middleware.CasbinMiddleware()).Use(RefreshSyncMiddleware(deps.refreshSyncSvc))
				// Menu writes mutate the platform-global `system_menu` table.
				// A tenant administrator's `/api/v1/*` tenant-domain Casbin
				// grant satisfies the route check above, so authoring the
				// shared menu tree must additionally require a platform
				// capability (same contract §3.3 gate as the tenant console
				// below). The tree read stays tenant-accessible: tenants need
				// their own navigation.
				{
					systemProtected.GET("/menu/tree", deps.menuHandler.GetMenuTree)
					systemProtected.POST("/menu", middleware.PlatformCapabilityMiddleware(), deps.menuHandler.CreateMenu)
					systemProtected.PUT("/menu/:id", middleware.PlatformCapabilityMiddleware(), deps.menuHandler.UpdateMenu)
					systemProtected.DELETE("/menu/:id", middleware.PlatformCapabilityMiddleware(), middleware.SecureActionMiddleware(), deps.menuHandler.DeleteMenu)
				}
			},
		},
		contracts.FuncModule{
			ModuleName:    "role",
			MigrateFunc:   func(_ *gorm.DB) error { return deps.roleSvc.Migrate() },
			BootstrapFunc: func(_ *gorm.DB) error { return deps.roleSvc.Bootstrap() },
			Register: func(r *gin.RouterGroup) {
				systemProtected := r.Group(routeGroupSystem).Use(middleware.TokenAuthMiddleware(database.RDB)).Use(middleware.PlatformScopeTenantContextMiddleware(deps.tenantModeLoader, deps.db)).Use(middleware.CasbinMiddleware()).Use(RefreshSyncMiddleware(deps.refreshSyncSvc))
				{
					systemProtected.GET("/role/list", deps.roleHandler.GetRoleList)
					systemProtected.GET("/role/:id/users", deps.roleHandler.GetRoleMembers)
					systemProtected.GET("/role/:id/user-candidates", deps.roleHandler.GetRoleMemberCandidates)
					systemProtected.POST("/role", deps.roleHandler.CreateRole)
					systemProtected.POST("/role/export", deps.roleHandler.ExportRoles)
					systemProtected.POST("/role/batch-status", deps.roleHandler.BatchUpdateRoleStatus)
					systemProtected.POST("/role/batch-delete", middleware.SecureActionMiddleware(), deps.roleHandler.BatchDeleteRoles)
					systemProtected.GET("/role/import-template", deps.roleHandler.DownloadImportTemplate)
					systemProtected.POST("/role/import", deps.roleHandler.ImportRoles)
					systemProtected.POST("/role/:id/users", deps.roleHandler.AddRoleMembers)
					systemProtected.POST("/role/:id/users/remove", deps.roleHandler.RemoveRoleMembers)
					systemProtected.PUT("/role/:id", deps.roleHandler.UpdateRole)
					systemProtected.DELETE("/role/:id", middleware.SecureActionMiddleware(), deps.roleHandler.DeleteRole)
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
				systemProtected := r.Group(routeGroupSystem).Use(middleware.TokenAuthMiddleware(database.RDB)).Use(middleware.PlatformScopeTenantContextMiddleware(deps.tenantModeLoader, deps.db)).Use(middleware.CasbinMiddleware()).Use(RefreshSyncMiddleware(deps.refreshSyncSvc))
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
		contracts.FuncModule{
			// Tenant management (platform-scoped): tables come from the tenant
			// migrations (000013..000020); only the console surface + menu seeds
			// are wired here. Visibility is gated by platform.tenant_mode via the
			// frontend capability filter (same pattern as system.org / org.enabled).
			ModuleName: "tenant",
			// MigrateFunc mirrors the permission module. RegisterBackendModules
			// runs either Migrate() (PANTHEON_AUTO_MIGRATE=true, dev) or
			// Bootstrap() (versioned-migration mode) — never both. Migrate adds
			// the plan-definition table (migration 000030 owns it in versioned
			// mode) and then runs the same repairs. The permission
			// bootstrap purges casbin policies whose subject is not a live
			// system_role.role_key, which includes every tenant-domain subject
			// `role:<key>@tenant:<id>`. Without this repair on the Migrate path a
			// dev-mode boot strips every tenant role policy and each tenant user
			// answers 403 permission.denied on every page. Idempotent/additive.
			MigrateFunc:   func(_ *gorm.DB) error { return deps.tenantSvc.Migrate() },
			BootstrapFunc: func(_ *gorm.DB) error { return deps.tenantSvc.Bootstrap() },
			SeedMenusFunc: seedTenantModuleMenus,
			Register: func(r *gin.RouterGroup) {
				// Platform-scoped by declaration (contract §3.3): the console
				// reads and writes `tenants` across every tenant, which is the
				// "平台管理 API（显式声明，要求平台权限，落审计）" path. The
				// declaration is what lets a tenant-less platform operator
				// resolve to the platform-global context (tenant 0) in multi
				// mode; Casbin still requires `system:tenant:*` on top.
				//
				// PlatformCapabilityMiddleware is the second half of that
				// contract: a tenant administrator's `/api/v1/*` tenant-domain
				// grant satisfies Casbin, so without the gate it could list
				// every tenant. The console stays a platform capability.
				systemProtected := r.Group(routeGroupSystem).Use(middleware.TokenAuthMiddleware(database.RDB)).Use(middleware.PlatformScopeTenantContextMiddleware(deps.tenantModeLoader, deps.db)).Use(middleware.CasbinMiddleware()).Use(middleware.PlatformCapabilityMiddleware()).Use(RefreshSyncMiddleware(deps.refreshSyncSvc))
				{
					systemProtected.GET("/tenant/list", deps.tenantHandler.ListTenants)
					systemProtected.GET("/tenant/:id", deps.tenantHandler.GetTenant)
					systemProtected.POST("/tenant", deps.tenantHandler.CreateTenant)
					systemProtected.PUT("/tenant/:id", deps.tenantHandler.UpdateTenant)
					systemProtected.DELETE("/tenant/:id", middleware.SecureActionMiddleware(), deps.tenantHandler.DeleteTenant)
					systemProtected.GET("/tenant/:id/members", deps.tenantHandler.ListTenantMembers)
					systemProtected.POST("/tenant/:id/members", deps.tenantHandler.AddTenantMember)
					systemProtected.DELETE("/tenant/:id/members/:userId", middleware.SecureActionMiddleware(), deps.tenantHandler.RemoveTenantMember)
					// Tenant-ops console (contract §2.5): the platform-global
					// rollup, the cross-tenant member listing and the plan
					// editor. Separate path segment from /tenant/:id so no static
					// route competes with the wildcard.
					systemProtected.GET("/tenant-ops/overview", deps.tenantHandler.GetOverview)
					systemProtected.GET("/tenant-ops/member", deps.tenantHandler.ListAllMembers)
					systemProtected.GET("/tenant-ops/plan/list", deps.tenantHandler.ListPlans)
					systemProtected.POST("/tenant-ops/plan", deps.tenantHandler.CreatePlan)
					systemProtected.PUT("/tenant-ops/plan/:code", deps.tenantHandler.UpdatePlan)
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
				systemProtected := r.Group(routeGroupSystem).Use(middleware.TokenAuthMiddleware(database.RDB)).Use(middleware.PlatformScopeTenantContextMiddleware(deps.tenantModeLoader, deps.db)).Use(middleware.CasbinMiddleware()).Use(RefreshSyncMiddleware(deps.refreshSyncSvc))
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
				systemProtected := r.Group(routeGroupSystem).Use(middleware.TokenAuthMiddleware(database.RDB)).Use(middleware.PlatformScopeTenantContextMiddleware(deps.tenantModeLoader, deps.db)).Use(middleware.CasbinMiddleware()).Use(RefreshSyncMiddleware(deps.refreshSyncSvc))
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
			SeedMenusFunc: seedDictModuleMenus, Register: func(r *gin.RouterGroup) {
				systemPublic := r.Group(routeGroupSystem)
				{
					systemPublic.GET("/dict/options", deps.dictHandler.GetDictOptions)
				}

				systemProtected := r.Group(routeGroupSystem).Use(middleware.TokenAuthMiddleware(database.RDB)).Use(middleware.PlatformScopeTenantContextMiddleware(deps.tenantModeLoader, deps.db)).Use(middleware.CasbinMiddleware())
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
			ModuleName: "setting",
			MigrateFunc: func(_ *gorm.DB) error {
				if err := deps.settingSvc.Migrate(); err != nil {
					return err
				}
				// Cross-instance invalidation: drop this process's setting
				// caches whenever any instance publishes settings:refresh.
				deps.settingSvc.WatchSettingsInvalidation()
				return nil
			},
			BootstrapFunc: func(_ *gorm.DB) error { return deps.settingSvc.Bootstrap() },
			SeedMenusFunc: seedSettingModuleMenus,
			Register: func(r *gin.RouterGroup) {
				systemPublic := r.Group(routeGroupSystem)
				{
					systemPublic.GET("/setting/public", deps.settingHandler.GetPublicSettings)
				}
				uploadProtected := r.Group(routeGroupSystem).Use(middleware.TokenAuthMiddleware(database.RDB)).Use(middleware.PlatformScopeTenantContextMiddleware(deps.tenantModeLoader, deps.db))
				{
					uploadProtected.GET("/upload/files/*filepath", deps.settingHandler.ServeUploadedFile)
				}

				// TenantContextMiddleware is REQUIRED on the upload write path too:
				// uploadScope() derives the `t{id}/` object-key prefix from the
				// resolved tenant context — without it a multi-mode upload would
				// silently fall back to the legacy (tenantless) keyspace and let
				// tenants collide into the global namespace (queue-6 matrix
				// finding #2, 2026-09-13).
				systemAuth := r.Group(routeGroupSystem).Use(middleware.TokenAuthMiddleware(database.RDB)).Use(middleware.PlatformScopeTenantContextMiddleware(deps.tenantModeLoader, deps.db))
				{
					systemAuth.POST("/upload", RefreshSyncMiddleware(deps.refreshSyncSvc), deps.settingHandler.UploadFile)
				}

				systemProtected := r.Group(routeGroupSystem).Use(middleware.TokenAuthMiddleware(database.RDB)).Use(middleware.PlatformScopeTenantContextMiddleware(deps.tenantModeLoader, deps.db)).Use(middleware.CasbinMiddleware())
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
				sysPublic := r.Group(routeGroupSystem)
				{
					sysPublic.GET("/i18n/pack", deps.i18nHandler.GetLangPack)
				}

				// The i18n console writes the platform-global `sys_i18n`
				// catalogue; a tenant administrator's blanket `/api/v1/*`
				// tenant-domain Casbin grant would otherwise let it rewrite
				// shared language packs, so the whole management group is a
				// platform capability (contract §3.3, same gate as the tenant
				// console). The public `/i18n/pack` read stays open — every
				// tenant's UI loads its language pack from it.
				sysProtected := r.Group(routeGroupSystem).Use(middleware.TokenAuthMiddleware(database.RDB)).Use(middleware.PlatformScopeTenantContextMiddleware(deps.tenantModeLoader, deps.db)).Use(middleware.CasbinMiddleware()).Use(middleware.PlatformCapabilityMiddleware())
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
					sysProtected.GET(routeI18nByID, deps.i18nHandler.Get)
					sysProtected.PUT(routeI18nByID, RefreshSyncMiddleware(deps.refreshSyncSvc), deps.i18nHandler.Update)
					sysProtected.DELETE(routeI18nByID, RefreshSyncMiddleware(deps.refreshSyncSvc), deps.i18nHandler.Delete)
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
				// TenantContextMiddleware is REQUIRED here: without it the audit
				// handler sees a nil tenant context and WithTenantScope no-ops,
				// letting every tenant read the full operation-log population
				// (found by the queue-6 live two-tenant matrix 2026-09-13).
				systemProtected := r.Group(routeGroupSystem).Use(middleware.TokenAuthMiddleware(database.RDB)).Use(middleware.PlatformScopeTenantContextMiddleware(deps.tenantModeLoader, deps.db)).Use(middleware.CasbinMiddleware())
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
