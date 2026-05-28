package system

import (
	"strings"

	"gorm.io/gorm"
)

type menuSeed struct {
	Key        string
	ParentKey  string
	TitleKey   string
	Path       string
	Component  string
	PagePerm   string
	Perms      string
	Type       string
	Icon       string
	RouteName  string
	Module     string
	Sort       int
	IsCache    int
	IsExternal int
	ActiveMenu string
}

func seedAuditModuleMenus(db *gorm.DB) error {
	return ensureMenuSeeds(db, append(baseMenuGroupSeeds(), auditMenuSeeds()...))
}

func seedMenuModuleMenus(db *gorm.DB) error {
	return ensureMenuSeeds(db, append(baseMenuGroupSeeds(), coreMenuSeeds()...))
}

func seedDeptModuleMenus(db *gorm.DB) error {
	return ensureMenuSeeds(db, append(baseMenuGroupSeeds(), deptMenuSeeds()...))
}

func seedPostModuleMenus(db *gorm.DB) error {
	return ensureMenuSeeds(db, append(baseMenuGroupSeeds(), postMenuSeeds()...))
}

func seedPermissionModuleMenus(db *gorm.DB) error {
	return ensureMenuSeeds(db, append(baseMenuGroupSeeds(), permissionMenuSeeds()...))
}

func seedSettingModuleMenus(db *gorm.DB) error {
	return ensureMenuSeeds(db, append(baseMenuGroupSeeds(), append(settingMenuSeeds(), platformToolMenuSeeds()...)...))
}

func seedDictModuleMenus(db *gorm.DB) error {
	return ensureMenuSeeds(db, append(baseMenuGroupSeeds(), dictMenuSeeds()...))
}

func seedI18nModuleMenus(db *gorm.DB) error {
	return ensureMenuSeeds(db, append(baseMenuGroupSeeds(), i18nMenuSeeds()...))
}

func ensureMenuSeeds(db *gorm.DB, seeds []menuSeed) error {
	if db == nil {
		return nil
	}
	if err := cleanupObsoleteMenus(db); err != nil {
		return err
	}
	for _, seed := range seeds {
		if err := ensureSingleMenuSeed(db, seed); err != nil {
			return err
		}
	}
	return cleanupActionMenuRoleBindings(db)
}

type obsoleteMenuRule struct {
	TitleKeys  []string
	Paths      []string
	RouteNames []string
	Components []string
	PagePerms  []string
	Perms      []string
}

var obsoleteMenuRules = []obsoleteMenuRule{
	{
		TitleKeys:  []string{"system.menu-matrix", "system.menu.matrix"},
		Paths:      []string{"/system/menu-matrix"},
		RouteNames: []string{"system-menu-matrix"},
		Components: []string{"system/menu/MenuMatrix"},
		PagePerms:  []string{"system:menu:matrix"},
		Perms:      []string{"system:menu:matrix"},
	},
}

func baseMenuGroupSeeds() []menuSeed {
	return []menuSeed{
		{
			Key:       "access",
			TitleKey:  "system.menu.access",
			Path:      "/system/access",
			Type:      "M",
			Icon:      "idcard",
			Module:    "system.iam",
			RouteName: "system-access",
			Sort:      20,
		},
		{
			Key:       "org",
			TitleKey:  "system.menu.org",
			Path:      "/system/org",
			Type:      "M",
			Icon:      "storage",
			Module:    "system.org",
			RouteName: "system-org",
			Sort:      30,
		},
		{
			Key:       "config",
			TitleKey:  "system.menu.config",
			Path:      "/system/config",
			Type:      "M",
			Icon:      "tool",
			Module:    "system.config",
			RouteName: "system-config",
			Sort:      50,
		},
		{
			Key:       "lowcode",
			TitleKey:  "system.menu.lowcode",
			Path:      "/system/lowcode",
			Type:      "M",
			Icon:      "code",
			Module:    "system.lowcode",
			RouteName: "system-lowcode",
			Sort:      45,
		},
		{
			Key:       "security",
			TitleKey:  "system.menu.security",
			Path:      "/system/security",
			Type:      "M",
			Icon:      "safe",
			Module:    "system.auth",
			RouteName: "system-security",
			Sort:      40,
		},
	}
}

func coreMenuSeeds() []menuSeed {
	return []menuSeed{
		{
			Key:       "workspace",
			TitleKey:  "app.workspace",
			Path:      "/workspace",
			Type:      "D",
			Icon:      "dashboard",
			RouteName: "workspace",
			Module:    "platform",
			Sort:      10,
		},
		{
			Key:       "dashboard",
			ParentKey: "workspace",
			TitleKey:  "system.menu.dashboard",
			Path:      "/dashboard",
			Component: "dashboard",
			PagePerm:  "platform:dashboard:view",
			Type:      "C",
			Icon:      "dashboard",
			RouteName: "dashboard",
			Module:    "platform",
			Sort:      1,
		},
		{
			Key:       "operations",
			ParentKey: "workspace",
			TitleKey:  "operations.menu",
			Path:      "/operations",
			Type:      "M",
			Icon:      "desktop",
			RouteName: "operations",
			Module:    "platform",
			Sort:      15,
		},
		{
			Key:       "user",
			ParentKey: "access",
			TitleKey:  "system.menu.user",
			Path:      "/system/user",
			Component: "system/user/UserList",
			PagePerm:  "system:user:list",
			Perms:     "",
			Type:      "C",
			Icon:      "user",
			RouteName: "system-user",
			Module:    "system.iam",
			Sort:      10,
		},
		{
			Key:       "role",
			ParentKey: "access",
			TitleKey:  "system.menu.role",
			Path:      "/system/role",
			Component: "system/role/RoleList",
			PagePerm:  "system:role:list",
			Perms:     "",
			Type:      "C",
			Icon:      "user-group",
			RouteName: "system-role",
			Module:    "system.iam",
			Sort:      20,
		},
		{
			Key:       "menu",
			ParentKey: "access",
			TitleKey:  "system.menu.menu",
			Path:      "/system/menu",
			Component: "system/menu/MenuList",
			PagePerm:  "system:menu:list",
			Perms:     "",
			Type:      "C",
			Icon:      "menu",
			RouteName: "system-menu",
			Module:    "system.iam",
			Sort:      40,
		},
	}
}

func deptMenuSeeds() []menuSeed {
	return []menuSeed{
		{
			Key:       "dept",
			ParentKey: "org",
			TitleKey:  "system.menu.dept",
			Path:      "/system/dept",
			Component: "system/dept/DeptList",
			PagePerm:  "system:dept:list",
			Perms:     "",
			Type:      "C",
			Icon:      "branch",
			RouteName: "system-dept",
			Module:    "system.org",
			Sort:      10,
		},
		{Key: "dept-create", ParentKey: "dept", TitleKey: "system.permission.dept.create", Perms: "system:dept:create", Type: "F", Sort: 1},
		{Key: "dept-update", ParentKey: "dept", TitleKey: "system.permission.dept.update", Perms: "system:dept:update", Type: "F", Sort: 2},
		{Key: "dept-delete", ParentKey: "dept", TitleKey: "system.permission.dept.delete", Perms: "system:dept:delete", Type: "F", Sort: 3},
		{Key: "dept-export", ParentKey: "dept", TitleKey: "system.permission.dept.export", Perms: "system:dept:export", Type: "F", Sort: 4},
		{Key: "dept-import", ParentKey: "dept", TitleKey: "system.permission.dept.import", Perms: "system:dept:import", Type: "F", Sort: 5},
		{Key: "dept-batch-update", ParentKey: "dept", TitleKey: "system.permission.dept.batch_update", Perms: "system:dept:batch-update", Type: "F", Sort: 6},
		{Key: "dept-batch-delete", ParentKey: "dept", TitleKey: "system.permission.dept.batch_delete", Perms: "system:dept:batch-delete", Type: "F", Sort: 7},
	}
}

func postMenuSeeds() []menuSeed {
	return []menuSeed{
		{
			Key:       "post",
			ParentKey: "org",
			TitleKey:  "system.menu.post",
			Path:      "/system/post",
			Component: "system/post/PostList",
			PagePerm:  "system:post:list",
			Perms:     "",
			Type:      "C",
			Icon:      "tags",
			RouteName: "system-post",
			Module:    "system.org",
			Sort:      20,
		},
		{Key: "post-create", ParentKey: "post", TitleKey: "system.permission.post.create", Perms: "system:post:create", Type: "F", Sort: 1},
		{Key: "post-update", ParentKey: "post", TitleKey: "system.permission.post.update", Perms: "system:post:update", Type: "F", Sort: 2},
		{Key: "post-delete", ParentKey: "post", TitleKey: "system.permission.post.delete", Perms: "system:post:delete", Type: "F", Sort: 3},
		{Key: "post-export", ParentKey: "post", TitleKey: "system.permission.post.export", Perms: "system:post:export", Type: "F", Sort: 4},
		{Key: "post-import", ParentKey: "post", TitleKey: "system.permission.post.import", Perms: "system:post:import", Type: "F", Sort: 5},
		{Key: "post-batch-update", ParentKey: "post", TitleKey: "system.permission.post.batch_update", Perms: "system:post:batch-update", Type: "F", Sort: 6},
		{Key: "post-batch-delete", ParentKey: "post", TitleKey: "system.permission.post.batch_delete", Perms: "system:post:batch-delete", Type: "F", Sort: 7},
	}
}

func permissionMenuSeeds() []menuSeed {
	return []menuSeed{
		{
			Key:       "permission",
			ParentKey: "access",
			TitleKey:  "system.menu.permission",
			Path:      "/system/permission",
			Component: "system/permission/PermissionList",
			PagePerm:  "system:permission:list",
			Perms:     "",
			Type:      "C",
			Icon:      "lock",
			RouteName: "system-permission",
			Module:    "system.iam",
			Sort:      30,
		},
		{Key: "user-view", ParentKey: "user", TitleKey: "system.permission.user.view", Perms: "system:user:view", Type: "F", Sort: 1},
		{Key: "user-create", ParentKey: "user", TitleKey: "system.permission.user.create", Perms: "system:user:create", Type: "F", Sort: 2},
		{Key: "user-update", ParentKey: "user", TitleKey: "system.permission.user.update", Perms: "system:user:update", Type: "F", Sort: 3},
		{Key: "user-delete", ParentKey: "user", TitleKey: "system.permission.user.delete", Perms: "system:user:delete", Type: "F", Sort: 4},
		{Key: "user-reset", ParentKey: "user", TitleKey: "system.permission.user.reset", Perms: "system:user:reset", Type: "F", Sort: 5},
		{Key: "user-export", ParentKey: "user", TitleKey: "system.permission.user.export", Perms: "system:user:export", Type: "F", Sort: 6},
		{Key: "user-import", ParentKey: "user", TitleKey: "system.permission.user.import", Perms: "system:user:import", Type: "F", Sort: 7},
		{Key: "user-batch-update", ParentKey: "user", TitleKey: "system.permission.user.batch_update", Perms: "system:user:batch-update", Type: "F", Sort: 8},
		{Key: "user-batch-delete", ParentKey: "user", TitleKey: "system.permission.user.batch_delete", Perms: "system:user:batch-delete", Type: "F", Sort: 9},
		{Key: "role-create", ParentKey: "role", TitleKey: "system.permission.role.create", Perms: "system:role:create", Type: "F", Sort: 1},
		{Key: "role-update", ParentKey: "role", TitleKey: "system.permission.role.update", Perms: "system:role:update", Type: "F", Sort: 2},
		{Key: "role-delete", ParentKey: "role", TitleKey: "system.permission.role.delete", Perms: "system:role:delete", Type: "F", Sort: 3},
		{Key: "role-batch-update", ParentKey: "role", TitleKey: "system.permission.role.batch_update", Perms: "system:role:batch-update", Type: "F", Sort: 4},
		{Key: "role-batch-delete", ParentKey: "role", TitleKey: "system.permission.role.batch_delete", Perms: "system:role:batch-delete", Type: "F", Sort: 5},
		{Key: "role-export", ParentKey: "role", TitleKey: "system.permission.role.export", Perms: "system:role:export", Type: "F", Sort: 6},
		{Key: "menu-create", ParentKey: "menu", TitleKey: "system.permission.menu.create", Perms: "system:menu:create", Type: "F", Sort: 1},
		{Key: "menu-update", ParentKey: "menu", TitleKey: "system.permission.menu.update", Perms: "system:menu:update", Type: "F", Sort: 2},
		{Key: "menu-delete", ParentKey: "menu", TitleKey: "system.permission.menu.delete", Perms: "system:menu:delete", Type: "F", Sort: 3},
		{Key: "permission-create", ParentKey: "permission", TitleKey: "system.permission.policy.create", Perms: "system:permission:create", Type: "F", Sort: 1},
		{Key: "permission-update", ParentKey: "permission", TitleKey: "system.permission.policy.update", Perms: "system:permission:update", Type: "F", Sort: 2},
		{Key: "permission-delete", ParentKey: "permission", TitleKey: "system.permission.policy.delete", Perms: "system:permission:delete", Type: "F", Sort: 3},
		{Key: "permission-batch-delete", ParentKey: "permission", TitleKey: "system.permission.policy.batch_delete", Perms: "system:permission:batch-delete", Type: "F", Sort: 4},
		{Key: "permission-export", ParentKey: "permission", TitleKey: "system.permission.policy.export", Perms: "system:permission:export", Type: "F", Sort: 5},
		{Key: "permission-import", ParentKey: "permission", TitleKey: "system.permission.policy.import", Perms: "system:permission:import", Type: "F", Sort: 6},
	}
}

func settingMenuSeeds() []menuSeed {
	return []menuSeed{
		{
			Key:       "setting",
			ParentKey: "config",
			TitleKey:  "system.menu.setting",
			Path:      "/system/setting",
			Component: "system/setting/SettingOverviewPage",
			PagePerm:  "system:setting:list",
			Perms:     "",
			Type:      "C",
			Icon:      "settings",
			RouteName: "system-setting",
			Module:    "system.config",
			IsCache:   1,
			Sort:      20,
		},
		{Key: "setting-update", ParentKey: "setting", TitleKey: "system.permission.setting.update", Perms: "system:setting:update", Type: "F", Sort: 1},
		{Key: "setting-refresh", ParentKey: "setting", TitleKey: "system.permission.setting.refresh", Perms: "system:setting:refresh", Type: "F", Sort: 2},
		{Key: "setting-export", ParentKey: "setting", TitleKey: "system.permission.setting.export", Perms: "system:setting:export", Type: "F", Sort: 3},
	}
}

func platformToolMenuSeeds() []menuSeed {
	return []menuSeed{
		{
			Key:       "modules",
			ParentKey: "lowcode",
			TitleKey:  "system.menu.modules",
			Path:      "/system/modules",
			Component: "system/dynamicmodule/ModuleManager",
			PagePerm:  "system:module:list",
			Type:      "C",
			Icon:      "apps",
			RouteName: "system-modules",
			Module:    "system.lowcode",
			Sort:      35,
		},
		{Key: "module-register", ParentKey: "modules", TitleKey: "system.permission.module.register", Perms: "system:module:register", Type: "F", Sort: 1},
		{Key: "module-unregister", ParentKey: "modules", TitleKey: "system.permission.module.unregister", Perms: "system:module:unregister", Type: "F", Sort: 2},
		{Key: "module-delete-record", ParentKey: "modules", TitleKey: "system.permission.module.deleteRecord", Perms: "system:module:delete_record", Type: "F", Sort: 3},
		{Key: "module-purge", ParentKey: "modules", TitleKey: "system.permission.module.purge", Perms: "system:module:purge", Type: "F", Sort: 4},
		{Key: "module-repair", ParentKey: "modules", TitleKey: "system.permission.module.repair", Perms: "system:module:repair", Type: "F", Sort: 5},
		{
			Key:       "generator",
			ParentKey: "lowcode",
			TitleKey:  "system.menu.generator",
			Path:      "/system/generator",
			Component: "system/generator/ModuleWizard",
			PagePerm:  "system:generator:use",
			Type:      "C",
			Icon:      "code",
			RouteName: "system-generator",
			Module:    "system.lowcode",
			Sort:      40,
		},
		{Key: "module-generate", ParentKey: "generator", TitleKey: "system.permission.module.generate", Perms: "system:module:generate", Type: "F", Sort: 1},
		{Key: "generator-datasource-manage", ParentKey: "generator", TitleKey: "system.permission.generator.datasourceManage", Perms: "system:generator:datasource:manage", Type: "F", Sort: 2},
	}
}

func dictMenuSeeds() []menuSeed {
	return []menuSeed{
		{
			Key:       "dict",
			ParentKey: "config",
			TitleKey:  "system.menu.dict",
			Path:      "/system/dict",
			Component: "system/dict/DictPage",
			PagePerm:  "system:dict:list",
			Perms:     "",
			Type:      "C",
			Icon:      "book",
			RouteName: "system-dict",
			Module:    "system.config",
			IsCache:   1,
			Sort:      10,
		},
		{Key: "dict-create", ParentKey: "dict", TitleKey: "system.permission.dict.create", Perms: "system:dict:create", Type: "F", Sort: 1},
		{Key: "dict-update", ParentKey: "dict", TitleKey: "system.permission.dict.update", Perms: "system:dict:update", Type: "F", Sort: 2},
		{Key: "dict-delete", ParentKey: "dict", TitleKey: "system.permission.dict.delete", Perms: "system:dict:delete", Type: "F", Sort: 3},
		{Key: "dict-refresh", ParentKey: "dict", TitleKey: "system.permission.dict.refresh", Perms: "system:dict:refresh", Type: "F", Sort: 4},
		{Key: "dict-export", ParentKey: "dict", TitleKey: "system.permission.dict.export", Perms: "system:dict:export", Type: "F", Sort: 5},
		{Key: "dict-import", ParentKey: "dict", TitleKey: "system.permission.dict.import", Perms: "system:dict:import", Type: "F", Sort: 6},
		{Key: "dict-batch-update", ParentKey: "dict", TitleKey: "system.permission.dict.batch_update", Perms: "system:dict:batch-update", Type: "F", Sort: 7},
		{Key: "dict-batch-delete", ParentKey: "dict", TitleKey: "system.permission.dict.batch_delete", Perms: "system:dict:batch-delete", Type: "F", Sort: 8},
	}
}

func i18nMenuSeeds() []menuSeed {
	return []menuSeed{
		{
			Key:       "i18n",
			ParentKey: "config",
			TitleKey:  "system.menu.i18n",
			Path:      "/system/i18n",
			Component: "system/i18n/I18nList",
			PagePerm:  "system:i18n:list",
			Perms:     "",
			Type:      "C",
			Icon:      "language",
			RouteName: "system-i18n",
			Module:    "system.config",
			IsCache:   1,
			Sort:      30,
		},
		{Key: "i18n-create", ParentKey: "i18n", TitleKey: "system.permission.i18n.create", Perms: "system:i18n:create", Type: "F", Sort: 1},
		{Key: "i18n-update", ParentKey: "i18n", TitleKey: "system.permission.i18n.update", Perms: "system:i18n:update", Type: "F", Sort: 2},
		{Key: "i18n-delete", ParentKey: "i18n", TitleKey: "system.permission.i18n.delete", Perms: "system:i18n:delete", Type: "F", Sort: 3},
		{Key: "i18n-batch-delete", ParentKey: "i18n", TitleKey: "system.permission.i18n.batch_delete", Perms: "system:i18n:delete", Type: "F", Sort: 4},
		{Key: "i18n-export", ParentKey: "i18n", TitleKey: "system.permission.i18n.export", Perms: "system:i18n:export", Type: "F", Sort: 5},
		{Key: "i18n-import", ParentKey: "i18n", TitleKey: "system.permission.i18n.import", Perms: "system:i18n:import", Type: "F", Sort: 6},
		{Key: "i18n-refresh", ParentKey: "i18n", TitleKey: "system.permission.i18n.refresh", Perms: "system:i18n:refresh", Type: "F", Sort: 7},
	}
}

func auditMenuSeeds() []menuSeed {
	return []menuSeed{
		{
			Key:       "operation-log",
			ParentKey: "security",
			TitleKey:  "system.menu.operationLog",
			Path:      "/system/operation-log",
			Component: "system/audit/OperationLogList",
			PagePerm:  "system:operation-log:list",
			Perms:     "",
			Type:      "C",
			Icon:      "file",
			RouteName: "system-operation-log",
			Module:    "system.audit",
			Sort:      30,
		},
		{Key: "operation-log-delete", ParentKey: "operation-log", TitleKey: "system.permission.operation_log.delete", Perms: "system:operation-log:delete", Type: "F", Sort: 1},
		{Key: "operation-log-clear", ParentKey: "operation-log", TitleKey: "system.permission.operation_log.clear", Perms: "system:operation-log:clear", Type: "F", Sort: 2},
		{Key: "operation-log-export", ParentKey: "operation-log", TitleKey: "system.permission.operation_log.export", Perms: "system:operation-log:export", Type: "F", Sort: 3},
	}
}

func ensureSingleMenuSeed(db *gorm.DB, seed menuSeed) error {
	var menuID uint64
	if seed.Path != "" {
		if err := db.Table("system_menu").Select("id").Where("path = ?", seed.Path).Limit(1).Pluck("id", &menuID).Error; err != nil {
			return err
		}
	} else if seed.Perms != "" {
		if err := db.Table("system_menu").Select("id").Where("perms = ?", seed.Perms).Limit(1).Pluck("id", &menuID).Error; err != nil {
			return err
		}
	}

	parentID, err := resolveMenuParentID(db, seed.ParentKey)
	if err != nil {
		return err
	}
	if menuID == 0 {
		payload := map[string]interface{}{
			"parent_id":   parentID,
			"title_key":   seed.TitleKey,
			"path":        seed.Path,
			"component":   seed.Component,
			"page_perm":   seed.PagePerm,
			"perms":       seed.Perms,
			"type":        normalizeSeedMenuType(seed.Type),
			"icon":        seed.Icon,
			"route_name":  strings.TrimSpace(seed.RouteName),
			"module":      normalizeSeedMenuModule(seed.Module),
			"sort":        seed.Sort,
			"is_visible":  1,
			"is_cache":    normalizeSeedMenuFlag(seed.IsCache),
			"is_external": normalizeSeedMenuFlag(seed.IsExternal),
			"active_menu": strings.TrimSpace(seed.ActiveMenu),
		}
		if err := db.Table("system_menu").Create(payload).Error; err != nil {
			return err
		}
		if seed.Path != "" {
			if err := db.Table("system_menu").Select("id").Where("path = ?", seed.Path).Limit(1).Pluck("id", &menuID).Error; err != nil {
				return err
			}
		} else if seed.Perms != "" {
			if err := db.Table("system_menu").Select("id").Where("perms = ?", seed.Perms).Limit(1).Pluck("id", &menuID).Error; err != nil {
				return err
			}
		}
	} else {
		updates := map[string]interface{}{
			"parent_id":   parentID,
			"title_key":   seed.TitleKey,
			"component":   seed.Component,
			"page_perm":   seed.PagePerm,
			"icon":        seed.Icon,
			"route_name":  strings.TrimSpace(seed.RouteName),
			"module":      normalizeSeedMenuModule(seed.Module),
			"sort":        seed.Sort,
			"type":        normalizeSeedMenuType(seed.Type),
			"is_visible":  1,
			"is_cache":    normalizeSeedMenuFlag(seed.IsCache),
			"is_external": normalizeSeedMenuFlag(seed.IsExternal),
			"active_menu": strings.TrimSpace(seed.ActiveMenu),
		}
		updates["path"] = seed.Path
		updates["perms"] = seed.Perms
		if err := db.Table("system_menu").Where("id = ?", menuID).Updates(updates).Error; err != nil {
			return err
		}
	}

	if menuID == 0 {
		return nil
	}

	var adminRoleID uint64
	if err := db.Table("system_role").Select("id").Where("role_key = ?", "admin").Limit(1).Pluck("id", &adminRoleID).Error; err != nil {
		return err
	}
	if adminRoleID == 0 {
		return nil
	}

	if normalizeSeedMenuType(seed.Type) != "F" {
		var count int64
		if err := db.Table("system_role_menu").Where("role_id = ? AND menu_id = ?", adminRoleID, menuID).Count(&count).Error; err != nil {
			return err
		}
		if count == 0 {
			if err := db.Exec("INSERT INTO system_role_menu (role_id, menu_id) VALUES (?, ?)", adminRoleID, menuID).Error; err != nil {
				return err
			}
		}
	}
	if err := ensureAdminPermissionSeed(db, adminRoleID, seed.PagePerm); err != nil {
		return err
	}
	if err := ensureAdminPermissionSeed(db, adminRoleID, seed.Perms); err != nil {
		return err
	}
	return nil
}

func ensureAdminPermissionSeed(db *gorm.DB, adminRoleID uint64, permissionKey string) error {
	permissionKey = strings.TrimSpace(permissionKey)
	if permissionKey == "" {
		return nil
	}
	var count int64
	if err := db.Table("system_role_permission").
		Where("role_id = ? AND permission_key = ?", adminRoleID, permissionKey).
		Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	return db.Exec("INSERT INTO system_role_permission (role_id, permission_key) VALUES (?, ?)", adminRoleID, permissionKey).Error
}

func cleanupActionMenuRoleBindings(db *gorm.DB) error {
	return db.Exec(`
DELETE FROM system_role_menu
WHERE menu_id IN (
	SELECT id FROM system_menu WHERE type = 'F'
)`).Error
}

func cleanupObsoleteMenus(db *gorm.DB) error {
	if db == nil || !db.Migrator().HasTable("system_menu") {
		return nil
	}

	return db.Transaction(func(tx *gorm.DB) error {
		obsoleteIDs := make(map[uint64]struct{})
		for _, rule := range obsoleteMenuRules {
			ids, err := collectObsoleteMenuIDs(tx, rule)
			if err != nil {
				return err
			}
			for _, id := range ids {
				obsoleteIDs[id] = struct{}{}
			}
		}

		if len(obsoleteIDs) == 0 {
			return nil
		}

		menuIDs := make([]uint64, 0, len(obsoleteIDs))
		for id := range obsoleteIDs {
			menuIDs = append(menuIDs, id)
		}

		if tx.Migrator().HasTable("system_role_menu") {
			if err := tx.Table("system_role_menu").Where("menu_id IN ?", menuIDs).Delete(nil).Error; err != nil {
				return err
			}
		}

		if tx.Migrator().HasTable("system_role_permission") {
			for _, rule := range obsoleteMenuRules {
				if len(rule.PagePerms) > 0 {
					if err := tx.Table("system_role_permission").Where("permission_key IN ?", rule.PagePerms).Delete(nil).Error; err != nil {
						return err
					}
				}
				if len(rule.Perms) > 0 {
					if err := tx.Table("system_role_permission").Where("permission_key IN ?", rule.Perms).Delete(nil).Error; err != nil {
						return err
					}
				}
			}
		}

		return tx.Table("system_menu").Where("id IN ?", menuIDs).Delete(nil).Error
	})
}

func collectObsoleteMenuIDs(tx *gorm.DB, rule obsoleteMenuRule) ([]uint64, error) {
	collected := make(map[uint64]struct{})

	var collect func(ids []uint64) error
	collect = func(ids []uint64) error {
		for _, id := range ids {
			if id == 0 {
				continue
			}
			if _, ok := collected[id]; ok {
				continue
			}
			collected[id] = struct{}{}

			var children []uint64
			if err := tx.Table("system_menu").Select("id").Where("parent_id = ?", id).Pluck("id", &children).Error; err != nil {
				return err
			}
			if err := collect(children); err != nil {
				return err
			}
		}
		return nil
	}

	var directIDs []uint64
	if len(rule.TitleKeys) > 0 {
		if err := tx.Table("system_menu").Select("id").Where("title_key IN ?", rule.TitleKeys).Pluck("id", &directIDs).Error; err != nil {
			return nil, err
		}
		if err := collect(directIDs); err != nil {
			return nil, err
		}
	}
	if len(rule.Paths) > 0 {
		directIDs = directIDs[:0]
		if err := tx.Table("system_menu").Select("id").Where("path IN ?", rule.Paths).Pluck("id", &directIDs).Error; err != nil {
			return nil, err
		}
		if err := collect(directIDs); err != nil {
			return nil, err
		}
	}
	if len(rule.RouteNames) > 0 {
		directIDs = directIDs[:0]
		if err := tx.Table("system_menu").Select("id").Where("route_name IN ?", rule.RouteNames).Pluck("id", &directIDs).Error; err != nil {
			return nil, err
		}
		if err := collect(directIDs); err != nil {
			return nil, err
		}
	}
	if len(rule.Components) > 0 {
		directIDs = directIDs[:0]
		if err := tx.Table("system_menu").Select("id").Where("component IN ?", rule.Components).Pluck("id", &directIDs).Error; err != nil {
			return nil, err
		}
		if err := collect(directIDs); err != nil {
			return nil, err
		}
	}
	if len(rule.PagePerms) > 0 {
		directIDs = directIDs[:0]
		if err := tx.Table("system_menu").Select("id").Where("page_perm IN ?", rule.PagePerms).Pluck("id", &directIDs).Error; err != nil {
			return nil, err
		}
		if err := collect(directIDs); err != nil {
			return nil, err
		}
	}
	if len(rule.Perms) > 0 {
		directIDs = directIDs[:0]
		if err := tx.Table("system_menu").Select("id").Where("perms IN ?", rule.Perms).Pluck("id", &directIDs).Error; err != nil {
			return nil, err
		}
		if err := collect(directIDs); err != nil {
			return nil, err
		}
	}

	result := make([]uint64, 0, len(collected))
	for id := range collected {
		result = append(result, id)
	}
	return result, nil
}

func resolveMenuParentID(db *gorm.DB, parentKey string) (uint64, error) {
	if parentKey == "" {
		return 0, nil
	}
	parentPaths := map[string]string{
		"access":        "/system/access",
		"org":           "/system/org",
		"config":        "/system/config",
		"lowcode":       "/system/lowcode",
		"security":      "/system/security",
		"user":          "/system/user",
		"role":          "/system/role",
		"menu":          "/system/menu",
		"dept":          "/system/dept",
		"post":          "/system/post",
		"permission":    "/system/permission",
		"login-log":     "/system/login-log",
		"session":       "/system/session",
		"setting":       "/system/setting",
		"modules":       "/system/modules",
		"generator":     "/system/generator",
		"dict":          "/system/dict",
		"i18n":          "/system/i18n",
		"operation-log": "/system/operation-log",
	}
	parentPath, ok := parentPaths[parentKey]
	if !ok {
		return 0, nil
	}
	return lookupMenuIDByPath(db, parentPath)
}

func lookupMenuIDByPath(db *gorm.DB, path string) (uint64, error) {
	var menuID uint64
	if err := db.Table("system_menu").Select("id").Where("path = ?", path).Limit(1).Pluck("id", &menuID).Error; err != nil {
		return 0, err
	}
	return menuID, nil
}

func normalizeSeedMenuType(value string) string {
	switch value {
	case "M", "C", "F":
		return value
	default:
		return "C"
	}
}

func normalizeSeedMenuModule(value string) string {
	if strings.TrimSpace(value) == "" {
		return "system"
	}
	return strings.TrimSpace(value)
}

func normalizeSeedMenuFlag(value int) int {
	if value == 1 {
		return 1
	}
	return 0
}
