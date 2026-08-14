package deploy

import (
	"time"

	"gorm.io/gorm"
)

const (
	deployModuleKey = "business.deploy"

	deployMenuTitleKey         = "operations.deploy.menu"
	deployPackageMenuTitleKey  = "operations.deploy.package.menu"
	deployTemplateMenuTitleKey = "operations.deploy.template.menu"
	deployTaskMenuTitleKey     = "operations.deploy.task.menu"

	deployPackageListRoute  = "deploy-package-list"
	deployTemplateListRoute = "deploy-template-list"
	deployTaskListRoute     = "deploy-task-list"

	deployTemplatePermCreateKey = "business.deploy.template.permission.create"
	deployTemplatePermUpdateKey = "business.deploy.template.permission.update"
	deployTemplatePermDeleteKey = "business.deploy.template.permission.delete"

	deployPackagePermCreateKey = "business.deploy.package.permission.create"
	deployPackagePermUpdateKey = "business.deploy.package.permission.update"
	deployPackagePermDeleteKey = "business.deploy.package.permission.delete"

	deployTaskPermDetailKey     = "business.deploy.task.permission.detail"
	deployTaskPermCreateKey     = "business.deploy.task.permission.create"
	deployTaskPermUpdateKey     = "business.deploy.task.permission.update"
	deployTaskPermDeleteKey     = "business.deploy.task.permission.delete"
	deployTaskPermStartKey      = "business.deploy.task.permission.start"
	deployTaskPermCancelKey     = "business.deploy.task.permission.cancel"
	deployTaskPermMarkResultKey = "business.deploy.task.permission.markResult"
)

type deployMenuSeed struct {
	Key       string
	ParentKey string
	TitleKey  string
	Path      string
	Component string
	PagePerm  string
	Perms     string
	Type      string
	Icon      string
	RouteName string
	Module    string
	Sort      int
}

type i18nSeed struct {
	Module string
	Locale string
	Group  string
	Key    string
	Value  string
}

func seedDeployMenus(db *gorm.DB) error {
	return ensureDeployMenuSeeds(db, []deployMenuSeed{
		{Key: "operations-deploy", ParentKey: "operations", TitleKey: deployMenuTitleKey, Path: "/operations/deploy", Type: "M", Module: deployModuleKey, Icon: "tool", RouteName: "deploy", Sort: 3},
		{Key: "operations-deploy-package", ParentKey: "deploy", TitleKey: deployPackageMenuTitleKey, Path: "/operations/deploy/package", Component: "business/deploy/package/DeployPackageList", PagePerm: "business:deploy:package:view", Type: "C", Module: deployModuleKey, RouteName: deployPackageListRoute, Sort: 1},
		{Key: "operations-deploy-template", ParentKey: "deploy", TitleKey: deployTemplateMenuTitleKey, Path: "/operations/deploy/template", Component: "business/deploy/template/DeployTemplateList", PagePerm: "business:deploy:template:list", Type: "C", Module: deployModuleKey, RouteName: deployTemplateListRoute, Sort: 2},
		{Key: "operations-deploy-template-create", ParentKey: deployTemplateListRoute, TitleKey: deployTemplatePermCreateKey, Perms: "business:deploy:template:create", Type: "F", Module: deployModuleKey, Sort: 1},
		{Key: "operations-deploy-template-update", ParentKey: deployTemplateListRoute, TitleKey: deployTemplatePermUpdateKey, Perms: "business:deploy:template:update", Type: "F", Module: deployModuleKey, Sort: 2},
		{Key: "operations-deploy-template-delete", ParentKey: deployTemplateListRoute, TitleKey: deployTemplatePermDeleteKey, Perms: "business:deploy:template:delete", Type: "F", Module: deployModuleKey, Sort: 3},
		{Key: "operations-deploy-package-create", ParentKey: deployPackageListRoute, TitleKey: deployPackagePermCreateKey, Perms: "business:deploy:package:create", Type: "F", Module: deployModuleKey, Sort: 1},
		{Key: "operations-deploy-package-update", ParentKey: deployPackageListRoute, TitleKey: deployPackagePermUpdateKey, Perms: "business:deploy:package:update", Type: "F", Module: deployModuleKey, Sort: 2},
		{Key: "operations-deploy-package-delete", ParentKey: deployPackageListRoute, TitleKey: deployPackagePermDeleteKey, Perms: "business:deploy:package:delete", Type: "F", Module: deployModuleKey, Sort: 3},
		{Key: "operations-deploy-task", ParentKey: "deploy", TitleKey: deployTaskMenuTitleKey, Path: "/operations/deploy/task", Component: "business/deploy/task/DeployTaskList", PagePerm: "business:deploy:task:view", Type: "C", Module: deployModuleKey, RouteName: deployTaskListRoute, Sort: 4},
		{Key: "operations-deploy-task-detail", ParentKey: deployTaskListRoute, TitleKey: deployTaskPermDetailKey, Perms: "business:deploy:task:detail", Type: "F", Module: deployModuleKey, Sort: 1},
		{Key: "operations-deploy-task-create", ParentKey: deployTaskListRoute, TitleKey: deployTaskPermCreateKey, Perms: "business:deploy:task:create", Type: "F", Module: deployModuleKey, Sort: 2},
		{Key: "operations-deploy-task-update", ParentKey: deployTaskListRoute, TitleKey: deployTaskPermUpdateKey, Perms: "business:deploy:task:update", Type: "F", Module: deployModuleKey, Sort: 3},
		{Key: "operations-deploy-task-delete", ParentKey: deployTaskListRoute, TitleKey: deployTaskPermDeleteKey, Perms: "business:deploy:task:delete", Type: "F", Module: deployModuleKey, Sort: 4},
		{Key: "operations-deploy-task-start", ParentKey: deployTaskListRoute, TitleKey: deployTaskPermStartKey, Perms: "business:deploy:task:start", Type: "F", Module: deployModuleKey, Sort: 5},
		{Key: "operations-deploy-task-cancel", ParentKey: deployTaskListRoute, TitleKey: deployTaskPermCancelKey, Perms: "business:deploy:task:cancel", Type: "F", Module: deployModuleKey, Sort: 6},
		{Key: "operations-deploy-task-mark-result", ParentKey: deployTaskListRoute, TitleKey: deployTaskPermMarkResultKey, Perms: "business:deploy:task:mark-result", Type: "F", Module: deployModuleKey, Sort: 7},
	})
}

func ensureDeployMenuSeeds(db *gorm.DB, seeds []deployMenuSeed) error {
	if db == nil {
		return nil
	}
	for _, seed := range seeds {
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
		parentID, err := resolveDeployMenuParentID(db, seed.ParentKey)
		if err != nil {
			return err
		}
		payload := map[string]interface{}{
			"parent_id":  parentID,
			"title_key":  seed.TitleKey,
			"path":       seed.Path,
			"component":  seed.Component,
			"page_perm":  seed.PagePerm,
			"perms":      seed.Perms,
			"type":       seed.Type,
			"icon":       seed.Icon,
			"route_name": seed.RouteName,
			"module":     seed.Module,
			"sort":       seed.Sort,
			"is_visible": 1,
			"is_cache":   0,
			"updated_at": time.Now(),
		}
		if menuID == 0 {
			payload["created_at"] = time.Now()
			if err := db.Table("system_menu").Create(payload).Error; err != nil {
				return err
			}
			if seed.Path != "" {
				_ = db.Table("system_menu").Select("id").Where("path = ?", seed.Path).Limit(1).Pluck("id", &menuID).Error
			} else if seed.Perms != "" {
				_ = db.Table("system_menu").Select("id").Where("perms = ?", seed.Perms).Limit(1).Pluck("id", &menuID).Error
			}
		} else if err := db.Table("system_menu").Where(idWhereClause, menuID).Updates(payload).Error; err != nil {
			return err
		}
		if err := ensureDeployAdminBindings(db, menuID, seed); err != nil {
			return err
		}
	}
	return nil
}

func resolveDeployMenuParentID(db *gorm.DB, parentKey string) (uint64, error) {
	if parentKey == "" {
		return 0, nil
	}
	var parentID uint64
	if err := db.Table("system_menu").Select("id").Where("route_name = ?", parentKey).Limit(1).Pluck("id", &parentID).Error; err != nil {
		return 0, err
	}
	return parentID, nil
}

func ensureDeployAdminBindings(db *gorm.DB, menuID uint64, seed deployMenuSeed) error {
	if menuID == 0 || !db.Migrator().HasTable("system_role") {
		return nil
	}
	var adminRoleID uint64
	if err := db.Table("system_role").Select("id").Where("role_key = ?", "admin").Limit(1).Pluck("id", &adminRoleID).Error; err != nil {
		return err
	}
	if adminRoleID == 0 {
		return nil
	}
	if seed.Type != "F" && db.Migrator().HasTable("system_role_menu") {
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
	if err := ensureDeployAdminPermission(db, adminRoleID, seed.PagePerm); err != nil {
		return err
	}
	return ensureDeployAdminPermission(db, adminRoleID, seed.Perms)
}

func ensureDeployAdminPermission(db *gorm.DB, adminRoleID uint64, permissionKey string) error {
	if permissionKey == "" || !db.Migrator().HasTable("system_role_permission") {
		return nil
	}
	var count int64
	if err := db.Table("system_role_permission").Where("role_id = ? AND permission_key = ?", adminRoleID, permissionKey).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	return db.Exec("INSERT INTO system_role_permission (role_id, permission_key) VALUES (?, ?)", adminRoleID, permissionKey).Error
}

func seedDeployI18n(db *gorm.DB) error {
	if db == nil {
		return nil
	}
	entries := []i18nSeed{
		{Module: deployModuleKey, Locale: "zh-CN", Group: "menu", Key: deployMenuTitleKey, Value: "安装部署"},
		{Module: deployModuleKey, Locale: "en-US", Group: "menu", Key: deployMenuTitleKey, Value: "Deployment"},
		{Module: deployModuleKey, Locale: "zh-CN", Group: "menu", Key: deployPackageMenuTitleKey, Value: "软件组件"},
		{Module: deployModuleKey, Locale: "en-US", Group: "menu", Key: deployPackageMenuTitleKey, Value: "Software"},
		{Module: deployModuleKey, Locale: "zh-CN", Group: "menu", Key: deployTemplateMenuTitleKey, Value: "任务模板"},
		{Module: deployModuleKey, Locale: "en-US", Group: "menu", Key: deployTemplateMenuTitleKey, Value: "Task Templates"},
		{Module: deployModuleKey, Locale: "zh-CN", Group: "menu", Key: deployTaskMenuTitleKey, Value: "部署任务"},
		{Module: deployModuleKey, Locale: "en-US", Group: "menu", Key: deployTaskMenuTitleKey, Value: "Tasks"},
		{Module: deployModuleKey, Locale: "zh-CN", Group: "page", Key: "operations.deploy.task.detail", Value: "任务详情"},
		{Module: deployModuleKey, Locale: "en-US", Group: "page", Key: "operations.deploy.task.detail", Value: "Task Detail"},
		{Module: deployModuleKey, Locale: "zh-CN", Group: "permission", Key: deployPackagePermCreateKey, Value: "新增软件组件"},
		{Module: deployModuleKey, Locale: "en-US", Group: "permission", Key: deployPackagePermCreateKey, Value: "Create software components"},
		{Module: deployModuleKey, Locale: "zh-CN", Group: "permission", Key: deployPackagePermUpdateKey, Value: "编辑软件组件"},
		{Module: deployModuleKey, Locale: "en-US", Group: "permission", Key: deployPackagePermUpdateKey, Value: "Update software components"},
		{Module: deployModuleKey, Locale: "zh-CN", Group: "permission", Key: deployPackagePermDeleteKey, Value: "删除软件组件"},
		{Module: deployModuleKey, Locale: "en-US", Group: "permission", Key: deployPackagePermDeleteKey, Value: "Delete software components"},
		{Module: deployModuleKey, Locale: "zh-CN", Group: "permission", Key: deployTemplatePermCreateKey, Value: "新增任务模板"},
		{Module: deployModuleKey, Locale: "en-US", Group: "permission", Key: deployTemplatePermCreateKey, Value: "Create task templates"},
		{Module: deployModuleKey, Locale: "zh-CN", Group: "permission", Key: deployTemplatePermUpdateKey, Value: "编辑任务模板"},
		{Module: deployModuleKey, Locale: "en-US", Group: "permission", Key: deployTemplatePermUpdateKey, Value: "Update task templates"},
		{Module: deployModuleKey, Locale: "zh-CN", Group: "permission", Key: deployTemplatePermDeleteKey, Value: "删除任务模板"},
		{Module: deployModuleKey, Locale: "en-US", Group: "permission", Key: deployTemplatePermDeleteKey, Value: "Delete task templates"},
		{Module: deployModuleKey, Locale: "zh-CN", Group: "permission", Key: deployTaskPermDetailKey, Value: "查看任务详情"},
		{Module: deployModuleKey, Locale: "en-US", Group: "permission", Key: deployTaskPermDetailKey, Value: "View task detail"},
		{Module: deployModuleKey, Locale: "zh-CN", Group: "permission", Key: deployTaskPermCreateKey, Value: "新增部署任务"},
		{Module: deployModuleKey, Locale: "en-US", Group: "permission", Key: deployTaskPermCreateKey, Value: "Create deployment tasks"},
		{Module: deployModuleKey, Locale: "zh-CN", Group: "permission", Key: deployTaskPermUpdateKey, Value: "编辑部署任务"},
		{Module: deployModuleKey, Locale: "en-US", Group: "permission", Key: deployTaskPermUpdateKey, Value: "Update deployment tasks"},
		{Module: deployModuleKey, Locale: "zh-CN", Group: "permission", Key: deployTaskPermDeleteKey, Value: "删除部署任务"},
		{Module: deployModuleKey, Locale: "en-US", Group: "permission", Key: deployTaskPermDeleteKey, Value: "Delete deployment tasks"},
		{Module: deployModuleKey, Locale: "zh-CN", Group: "permission", Key: deployTaskPermStartKey, Value: "启动部署任务"},
		{Module: deployModuleKey, Locale: "en-US", Group: "permission", Key: deployTaskPermStartKey, Value: "Start deployment tasks"},
		{Module: deployModuleKey, Locale: "zh-CN", Group: "permission", Key: deployTaskPermCancelKey, Value: "取消部署任务"},
		{Module: deployModuleKey, Locale: "en-US", Group: "permission", Key: deployTaskPermCancelKey, Value: "Cancel deployment tasks"},
		{Module: deployModuleKey, Locale: "zh-CN", Group: "permission", Key: deployTaskPermMarkResultKey, Value: "标记执行结果"},
		{Module: deployModuleKey, Locale: "en-US", Group: "permission", Key: deployTaskPermMarkResultKey, Value: "Mark execution result"},
	}
	for _, seed := range entries {
		var existingID uint64
		if err := db.Table("system_i18n").Select("id").Where("`key` = ? AND locale = ?", seed.Key, seed.Locale).Limit(1).Pluck("id", &existingID).Error; err != nil {
			return err
		}
		if existingID > 0 {
			update := map[string]interface{}{
				"value":      seed.Value,
				"module":     seed.Module,
				"updated_at": time.Now(),
			}
			if err := db.Table("system_i18n").Where(idWhereClause, existingID).Updates(update).Error; err != nil {
				return err
			}
			continue
		}
		payload := map[string]interface{}{
			"module":           seed.Module,
			"group_name":       seed.Group,
			"key":              seed.Key,
			"locale":           seed.Locale,
			"value":            seed.Value,
			"lifecycle_status": "active",
			"created_at":       time.Now(),
			"updated_at":       time.Now(),
		}
		if err := db.Table("system_i18n").Create(payload).Error; err != nil {
			return err
		}
	}
	return nil
}
