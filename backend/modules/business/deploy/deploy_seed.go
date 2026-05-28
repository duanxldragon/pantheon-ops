package deploy

import (
	"time"

	"gorm.io/gorm"
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

func seedDeployMenus(db *gorm.DB) error {
	return ensureDeployMenuSeeds(db, []deployMenuSeed{
		{Key: "operations-deploy", ParentKey: "operations", TitleKey: "operations.deploy.menu", Path: "/operations/deploy", Type: "M", Module: "business.deploy", Icon: "tool", RouteName: "deploy", Sort: 3},
		{Key: "operations-deploy-package", ParentKey: "deploy", TitleKey: "operations.deploy.package.menu", Path: "/operations/deploy/package", Component: "business/deploy/package/DeployPackageList", PagePerm: "business:deploy:package:list", Type: "C", Module: "business.deploy", RouteName: "deploy-package-list", Sort: 1},
		{Key: "operations-deploy-template", ParentKey: "deploy", TitleKey: "operations.deploy.template.menu", Path: "/operations/deploy/template", Component: "business/deploy/template/DeployTemplateList", PagePerm: "business:deploy:template:list", Type: "C", Module: "business.deploy", RouteName: "deploy-template-list", Sort: 2},
		{Key: "operations-deploy-package-create", ParentKey: "deploy-package-list", TitleKey: "business.deploy.package.permission.create", Perms: "business:deploy:package:create", Type: "F", Module: "business.deploy", Sort: 1},
		{Key: "operations-deploy-package-update", ParentKey: "deploy-package-list", TitleKey: "business.deploy.package.permission.update", Perms: "business:deploy:package:update", Type: "F", Module: "business.deploy", Sort: 2},
		{Key: "operations-deploy-package-delete", ParentKey: "deploy-package-list", TitleKey: "business.deploy.package.permission.delete", Perms: "business:deploy:package:delete", Type: "F", Module: "business.deploy", Sort: 3},
		{Key: "operations-deploy-task", ParentKey: "deploy", TitleKey: "operations.deploy.task.menu", Path: "/operations/deploy/task", Component: "business/deploy/task/DeployTaskList", PagePerm: "business:deploy:task:list", Type: "C", Module: "business.deploy", RouteName: "deploy-task-list", Sort: 4},
		{Key: "operations-deploy-task-detail", ParentKey: "deploy-task-list", TitleKey: "business.deploy.task.permission.detail", Perms: "business:deploy:task:detail", Type: "F", Module: "business.deploy", Sort: 1},
		{Key: "operations-deploy-task-create", ParentKey: "deploy-task-list", TitleKey: "business.deploy.task.permission.create", Perms: "business:deploy:task:create", Type: "F", Module: "business.deploy", Sort: 2},
		{Key: "operations-deploy-task-update", ParentKey: "deploy-task-list", TitleKey: "business.deploy.task.permission.update", Perms: "business:deploy:task:update", Type: "F", Module: "business.deploy", Sort: 3},
		{Key: "operations-deploy-task-start", ParentKey: "deploy-task-list", TitleKey: "business.deploy.task.permission.start", Perms: "business:deploy:task:start", Type: "F", Module: "business.deploy", Sort: 4},
		{Key: "operations-deploy-task-cancel", ParentKey: "deploy-task-list", TitleKey: "business.deploy.task.permission.cancel", Perms: "business:deploy:task:cancel", Type: "F", Module: "business.deploy", Sort: 5},
		{Key: "operations-deploy-task-mark-result", ParentKey: "deploy-task-list", TitleKey: "business.deploy.task.permission.markResult", Perms: "business:deploy:task:mark-result", Type: "F", Module: "business.deploy", Sort: 6},
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
		} else if err := db.Table("system_menu").Where("id = ?", menuID).Updates(payload).Error; err != nil {
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
	entries := []map[string]interface{}{
		{"module": "business.deploy", "locale": "zh-CN", "group_name": "menu", "key": "operations.deploy.menu", "value": "安装部署", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.deploy", "locale": "en-US", "group_name": "menu", "key": "operations.deploy.menu", "value": "Deployment", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.deploy", "locale": "zh-CN", "group_name": "menu", "key": "operations.deploy.package.menu", "value": "软件组件", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.deploy", "locale": "en-US", "group_name": "menu", "key": "operations.deploy.package.menu", "value": "Software", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.deploy", "locale": "zh-CN", "group_name": "menu", "key": "operations.deploy.template.menu", "value": "任务模板", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.deploy", "locale": "en-US", "group_name": "menu", "key": "operations.deploy.template.menu", "value": "Task Templates", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.deploy", "locale": "zh-CN", "group_name": "menu", "key": "operations.deploy.task.menu", "value": "部署任务", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.deploy", "locale": "en-US", "group_name": "menu", "key": "operations.deploy.task.menu", "value": "Tasks", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.deploy", "locale": "zh-CN", "group_name": "page", "key": "operations.deploy.task.detail", "value": "任务详情", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.deploy", "locale": "en-US", "group_name": "page", "key": "operations.deploy.task.detail", "value": "Task Detail", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.deploy", "locale": "zh-CN", "group_name": "permission", "key": "business.deploy.package.permission.create", "value": "新增软件组件", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.deploy", "locale": "en-US", "group_name": "permission", "key": "business.deploy.package.permission.create", "value": "Create software components", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.deploy", "locale": "zh-CN", "group_name": "permission", "key": "business.deploy.package.permission.update", "value": "编辑软件组件", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.deploy", "locale": "en-US", "group_name": "permission", "key": "business.deploy.package.permission.update", "value": "Update software components", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.deploy", "locale": "zh-CN", "group_name": "permission", "key": "business.deploy.package.permission.delete", "value": "删除软件组件", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.deploy", "locale": "en-US", "group_name": "permission", "key": "business.deploy.package.permission.delete", "value": "Delete software components", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.deploy", "locale": "zh-CN", "group_name": "permission", "key": "business.deploy.task.permission.detail", "value": "查看任务详情", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.deploy", "locale": "en-US", "group_name": "permission", "key": "business.deploy.task.permission.detail", "value": "View task detail", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.deploy", "locale": "zh-CN", "group_name": "permission", "key": "business.deploy.task.permission.create", "value": "新增部署任务", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.deploy", "locale": "en-US", "group_name": "permission", "key": "business.deploy.task.permission.create", "value": "Create deployment tasks", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.deploy", "locale": "zh-CN", "group_name": "permission", "key": "business.deploy.task.permission.update", "value": "编辑部署任务", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.deploy", "locale": "en-US", "group_name": "permission", "key": "business.deploy.task.permission.update", "value": "Update deployment tasks", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.deploy", "locale": "zh-CN", "group_name": "permission", "key": "business.deploy.task.permission.start", "value": "启动部署任务", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.deploy", "locale": "en-US", "group_name": "permission", "key": "business.deploy.task.permission.start", "value": "Start deployment tasks", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.deploy", "locale": "zh-CN", "group_name": "permission", "key": "business.deploy.task.permission.cancel", "value": "取消部署任务", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.deploy", "locale": "en-US", "group_name": "permission", "key": "business.deploy.task.permission.cancel", "value": "Cancel deployment tasks", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.deploy", "locale": "zh-CN", "group_name": "permission", "key": "business.deploy.task.permission.markResult", "value": "标记执行结果", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.deploy", "locale": "en-US", "group_name": "permission", "key": "business.deploy.task.permission.markResult", "value": "Mark execution result", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
	}
	for _, record := range entries {
		var existingID uint64
		if err := db.Table("system_i18n").Select("id").Where("`key` = ? AND locale = ?", record["key"], record["locale"]).Limit(1).Pluck("id", &existingID).Error; err != nil {
			return err
		}
		if existingID == 0 {
			if err := db.Table("system_i18n").Create(record).Error; err != nil {
				return err
			}
		} else if err := db.Table("system_i18n").Where("id = ?", existingID).Updates(map[string]interface{}{
			"value":      record["value"],
			"module":     record["module"],
			"updated_at": record["updated_at"],
		}).Error; err != nil {
			return err
		}
	}
	return nil
}
