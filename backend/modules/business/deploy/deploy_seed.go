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
	deployTemplatePermExportKey = "business.deploy.template.permission.export"
	deployTemplatePermImportKey = "business.deploy.template.permission.import"

	deployPackagePermCreateKey = "business.deploy.package.permission.create"
	deployPackagePermUpdateKey = "business.deploy.package.permission.update"
	deployPackagePermDeleteKey = "business.deploy.package.permission.delete"
	deployPackagePermExportKey = "business.deploy.package.permission.export"
	deployPackagePermImportKey = "business.deploy.package.permission.import"

	deployTaskPermDetailKey     = "business.deploy.task.permission.detail"
	deployTaskPermCreateKey     = "business.deploy.task.permission.create"
	deployTaskPermUpdateKey     = "business.deploy.task.permission.update"
	deployTaskPermDeleteKey     = "business.deploy.task.permission.delete"
	deployTaskPermStartKey      = "business.deploy.task.permission.start"
	deployTaskPermPauseKey      = "business.deploy.task.permission.pause"
	deployTaskPermResumeKey     = "business.deploy.task.permission.resume"
	deployTaskPermRetryKey      = "business.deploy.task.permission.retry"
	deployTaskPermCancelKey     = "business.deploy.task.permission.cancel"
	deployTaskPermMarkResultKey = "business.deploy.task.permission.markResult"
	deployTaskPermExportKey     = "business.deploy.task.permission.export"
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
		{Key: "business-deploy", ParentKey: "", TitleKey: deployMenuTitleKey, Path: "/business/deploy", Type: "M", Module: deployModuleKey, Icon: "tool", RouteName: "deploy", Sort: 4},
		{Key: "business-deploy-package", ParentKey: "deploy", TitleKey: deployPackageMenuTitleKey, Path: "/business/deploy/package", Component: "business/deploy/package/DeployPackageList", PagePerm: "business:deploy:package:view", Type: "C", Icon: "file", Module: deployModuleKey, RouteName: deployPackageListRoute, Sort: 1},
		{Key: "business-deploy-template", ParentKey: "deploy", TitleKey: deployTemplateMenuTitleKey, Path: "/business/deploy/template", Component: "business/deploy/template/DeployTemplateList", PagePerm: "business:deploy:template:list", Type: "C", Icon: "book", Module: deployModuleKey, RouteName: deployTemplateListRoute, Sort: 2},
		{Key: "operations-deploy-template-create", ParentKey: deployTemplateListRoute, TitleKey: deployTemplatePermCreateKey, Perms: "business:deploy:template:create", Type: "F", Module: deployModuleKey, Sort: 1},
		{Key: "operations-deploy-template-update", ParentKey: deployTemplateListRoute, TitleKey: deployTemplatePermUpdateKey, Perms: "business:deploy:template:update", Type: "F", Module: deployModuleKey, Sort: 2},
		{Key: "operations-deploy-template-delete", ParentKey: deployTemplateListRoute, TitleKey: deployTemplatePermDeleteKey, Perms: "business:deploy:template:delete", Type: "F", Module: deployModuleKey, Sort: 3},
		{Key: "operations-deploy-template-export", ParentKey: deployTemplateListRoute, TitleKey: deployTemplatePermExportKey, Perms: "business:deploy:template:export", Type: "F", Module: deployModuleKey, Sort: 4},
		{Key: "operations-deploy-template-import", ParentKey: deployTemplateListRoute, TitleKey: deployTemplatePermImportKey, Perms: "business:deploy:template:import", Type: "F", Module: deployModuleKey, Sort: 5},
		{Key: "operations-deploy-package-create", ParentKey: deployPackageListRoute, TitleKey: deployPackagePermCreateKey, Perms: "business:deploy:package:create", Type: "F", Module: deployModuleKey, Sort: 1},
		{Key: "operations-deploy-package-update", ParentKey: deployPackageListRoute, TitleKey: deployPackagePermUpdateKey, Perms: "business:deploy:package:update", Type: "F", Module: deployModuleKey, Sort: 2},
		{Key: "operations-deploy-package-delete", ParentKey: deployPackageListRoute, TitleKey: deployPackagePermDeleteKey, Perms: "business:deploy:package:delete", Type: "F", Module: deployModuleKey, Sort: 3},
		{Key: "operations-deploy-package-export", ParentKey: deployPackageListRoute, TitleKey: deployPackagePermExportKey, Perms: "business:deploy:package:export", Type: "F", Module: deployModuleKey, Sort: 4},
		{Key: "operations-deploy-package-import", ParentKey: deployPackageListRoute, TitleKey: deployPackagePermImportKey, Perms: "business:deploy:package:import", Type: "F", Module: deployModuleKey, Sort: 5},
		{Key: "business-deploy-task", ParentKey: "deploy", TitleKey: deployTaskMenuTitleKey, Path: "/business/deploy/task", Component: "business/deploy/task/DeployTaskList", PagePerm: "business:deploy:task:view", Type: "C", Icon: "clock", Module: deployModuleKey, RouteName: deployTaskListRoute, Sort: 4},
		{Key: "operations-deploy-task-detail", ParentKey: deployTaskListRoute, TitleKey: deployTaskPermDetailKey, Perms: "business:deploy:task:detail", Type: "F", Module: deployModuleKey, Sort: 1},
		{Key: "operations-deploy-task-create", ParentKey: deployTaskListRoute, TitleKey: deployTaskPermCreateKey, Perms: "business:deploy:task:create", Type: "F", Module: deployModuleKey, Sort: 2},
		{Key: "operations-deploy-task-update", ParentKey: deployTaskListRoute, TitleKey: deployTaskPermUpdateKey, Perms: "business:deploy:task:update", Type: "F", Module: deployModuleKey, Sort: 3},
		{Key: "operations-deploy-task-delete", ParentKey: deployTaskListRoute, TitleKey: deployTaskPermDeleteKey, Perms: "business:deploy:task:delete", Type: "F", Module: deployModuleKey, Sort: 4},
		{Key: "operations-deploy-task-start", ParentKey: deployTaskListRoute, TitleKey: deployTaskPermStartKey, Perms: "business:deploy:task:start", Type: "F", Module: deployModuleKey, Sort: 5},
		{Key: "operations-deploy-task-pause", ParentKey: deployTaskListRoute, TitleKey: deployTaskPermPauseKey, Perms: "business:deploy:task:pause", Type: "F", Module: deployModuleKey, Sort: 6},
		{Key: "operations-deploy-task-resume", ParentKey: deployTaskListRoute, TitleKey: deployTaskPermResumeKey, Perms: "business:deploy:task:resume", Type: "F", Module: deployModuleKey, Sort: 7},
		{Key: "operations-deploy-task-retry", ParentKey: deployTaskListRoute, TitleKey: deployTaskPermRetryKey, Perms: "business:deploy:task:retry", Type: "F", Module: deployModuleKey, Sort: 8},
		{Key: "operations-deploy-task-cancel", ParentKey: deployTaskListRoute, TitleKey: deployTaskPermCancelKey, Perms: "business:deploy:task:cancel", Type: "F", Module: deployModuleKey, Sort: 9},
		{Key: "operations-deploy-task-mark-result", ParentKey: deployTaskListRoute, TitleKey: deployTaskPermMarkResultKey, Perms: "business:deploy:task:mark-result", Type: "F", Module: deployModuleKey, Sort: 10},
		{Key: "operations-deploy-task-export", ParentKey: deployTaskListRoute, TitleKey: deployTaskPermExportKey, Perms: "business:deploy:task:export", Type: "F", Module: deployModuleKey, Sort: 11},
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
		{Module: deployModuleKey, Locale: "zh-CN", Group: "error", Key: errDeployServiceLinkInvalid, Value: "服务与实例必须有效、属于同一业务域并匹配实际部署主机"},
		{Module: deployModuleKey, Locale: "en-US", Group: "error", Key: errDeployServiceLinkInvalid, Value: "Service and instance must be active, belong to the same business scope and match the deployment host"},
		{Module: deployModuleKey, Locale: "en-US", Group: "page", Key: "business.deploy.task.commandConflict", Value: "The task changed. Refresh completed; review its current state before trying the action again."},
		{Module: deployModuleKey, Locale: "en-US", Group: "page", Key: "business.deploy.task.commandVersionRequired", Value: "Refresh the task before performing this action."},
		{Module: deployModuleKey, Locale: "en-US", Group: "page", Key: "business.deploy.task.commandPartialFailure", Value: "Some selected tasks could not be canceled. The list has refreshed; review the remaining selection before trying again."},
		{Module: deployModuleKey, Locale: "zh-CN", Group: "page", Key: "business.deploy.task.commandConflict", Value: "任务状态已变化，请刷新并确认当前状态后重新操作。"},
		{Module: deployModuleKey, Locale: "zh-CN", Group: "page", Key: "business.deploy.task.commandVersionRequired", Value: "请先刷新任务，再执行此操作。"},
		{Module: deployModuleKey, Locale: "zh-CN", Group: "page", Key: "business.deploy.task.commandPartialFailure", Value: "部分任务未能取消。列表已刷新，请确认剩余选择后重新操作。"},
		{Module: deployModuleKey, Locale: "zh-CN", Group: "page", Key: "business.deploy.preview.search", Value: "搜索目标和版本变化"},
		{Module: deployModuleKey, Locale: "en-US", Group: "page", Key: "business.deploy.preview.search", Value: "Search target and version changes"},
		{Module: deployModuleKey, Locale: "zh-CN", Group: "page", Key: "business.deploy.preview.title", Value: "执行前预览"},
		{Module: deployModuleKey, Locale: "en-US", Group: "page", Key: "business.deploy.preview.title", Value: "Review before execution"},
		{Module: deployModuleKey, Locale: "zh-CN", Group: "page", Key: "business.deploy.preview.changes", Value: "目标版本变化"},
		{Module: deployModuleKey, Locale: "en-US", Group: "page", Key: "business.deploy.preview.changes", Value: "Target version changes"},
		{Module: deployModuleKey, Locale: "zh-CN", Group: "page", Key: "business.deploy.preview.scopeSummary", Value: "业务范围：{{scope}}；目标数：{{count}}"},
		{Module: deployModuleKey, Locale: "en-US", Group: "page", Key: "business.deploy.preview.scopeSummary", Value: "Scope: {{scope}}; targets: {{count}}"},
		{Module: deployModuleKey, Locale: "zh-CN", Group: "page", Key: "business.deploy.preview.loadFailed", Value: "预览加载失败，请重试"},
		{Module: deployModuleKey, Locale: "en-US", Group: "page", Key: "business.deploy.preview.loadFailed", Value: "Preview unavailable; retry"},
		{Module: deployModuleKey, Locale: "zh-CN", Group: "page", Key: "business.deploy.preview.forbidden", Value: "无权查看或执行此任务"},
		{Module: deployModuleKey, Locale: "en-US", Group: "page", Key: "business.deploy.preview.forbidden", Value: "Task preview or execution access denied"},
		{Module: deployModuleKey, Locale: "zh-CN", Group: "page", Key: "business.deploy.preview.stale", Value: "预览已过期或数据已变化，请刷新预览"},
		{Module: deployModuleKey, Locale: "en-US", Group: "page", Key: "business.deploy.preview.stale", Value: "Preview expired or changed; refresh before execution"},
		{Module: deployModuleKey, Locale: "zh-CN", Group: "page", Key: "business.deploy.preview.partial", Value: "部分目标信息不可用"},
		{Module: deployModuleKey, Locale: "en-US", Group: "page", Key: "business.deploy.preview.partial", Value: "Some target details are unavailable"},
		{Module: deployModuleKey, Locale: "zh-CN", Group: "page", Key: "business.deploy.preview.ready", Value: "预览已就绪"},
		{Module: deployModuleKey, Locale: "en-US", Group: "page", Key: "business.deploy.preview.ready", Value: "Preview ready"},
		{Module: deployModuleKey, Locale: "zh-CN", Group: "page", Key: "business.deploy.preview.added", Value: "新增"},
		{Module: deployModuleKey, Locale: "en-US", Group: "page", Key: "business.deploy.preview.added", Value: "Added"},
		{Module: deployModuleKey, Locale: "zh-CN", Group: "page", Key: "business.deploy.preview.removed", Value: "移除"},
		{Module: deployModuleKey, Locale: "en-US", Group: "page", Key: "business.deploy.preview.removed", Value: "Removed"},
		{Module: deployModuleKey, Locale: "zh-CN", Group: "page", Key: "business.deploy.preview.modified", Value: "修改"},
		{Module: deployModuleKey, Locale: "en-US", Group: "page", Key: "business.deploy.preview.modified", Value: "Modified"},
		{Module: deployModuleKey, Locale: "zh-CN", Group: "page", Key: "business.deploy.preview.unchanged", Value: "未改变"},
		{Module: deployModuleKey, Locale: "en-US", Group: "page", Key: "business.deploy.preview.unchanged", Value: "Unchanged"},
		{Module: deployModuleKey, Locale: "zh-CN", Group: "page", Key: "business.deploy.preview.conflict", Value: "当前状态冲突，请重新预览"},
		{Module: deployModuleKey, Locale: "en-US", Group: "page", Key: "business.deploy.preview.conflict", Value: "Current state conflicts; preview again"},
		{Module: deployModuleKey, Locale: "zh-CN", Group: "page", Key: "business.deploy.preview.before", Value: "当前版本"},
		{Module: deployModuleKey, Locale: "en-US", Group: "page", Key: "business.deploy.preview.before", Value: "Current version"},
		{Module: deployModuleKey, Locale: "zh-CN", Group: "page", Key: "business.deploy.preview.after", Value: "目标版本"},
		{Module: deployModuleKey, Locale: "en-US", Group: "page", Key: "business.deploy.preview.after", Value: "Target version"},
		{Module: deployModuleKey, Locale: "zh-CN", Group: "page", Key: "business.deploy.preview.sensitive", Value: "敏感内容已隐藏"},
		{Module: deployModuleKey, Locale: "en-US", Group: "page", Key: "business.deploy.preview.sensitive", Value: "Sensitive content hidden"},
		{Module: deployModuleKey, Locale: "zh-CN", Group: "page", Key: "business.deploy.preview.unchangedHidden", Value: "未改变的目标已折叠"},
		{Module: deployModuleKey, Locale: "en-US", Group: "page", Key: "business.deploy.preview.unchangedHidden", Value: "Unchanged targets are collapsed"},
		{Module: deployModuleKey, Locale: "zh-CN", Group: "page", Key: "business.deploy.preview.expandUnchanged", Value: "显示未改变项"},
		{Module: deployModuleKey, Locale: "en-US", Group: "page", Key: "business.deploy.preview.expandUnchanged", Value: "Show unchanged"},
		{Module: deployModuleKey, Locale: "zh-CN", Group: "page", Key: "business.deploy.preview.collapseUnchanged", Value: "折叠未改变项"},
		{Module: deployModuleKey, Locale: "en-US", Group: "page", Key: "business.deploy.preview.collapseUnchanged", Value: "Hide unchanged"},
		{Module: deployModuleKey, Locale: "zh-CN", Group: "page", Key: "business.deploy.preview.refresh", Value: "刷新预览"},
		{Module: deployModuleKey, Locale: "en-US", Group: "page", Key: "business.deploy.preview.refresh", Value: "Refresh preview"},
		{Module: deployModuleKey, Locale: "zh-CN", Group: "page", Key: "business.deploy.preview.confirm", Value: "确认并执行"},
		{Module: deployModuleKey, Locale: "en-US", Group: "page", Key: "business.deploy.preview.confirm", Value: "Confirm and execute"},
		{Module: deployModuleKey, Locale: "zh-CN", Group: "page", Key: "business.deploy.preview.healthUnknown", Value: "目标健康状态尚未验证"},
		{Module: deployModuleKey, Locale: "en-US", Group: "page", Key: "business.deploy.preview.healthUnknown", Value: "Target health has not been verified"},
		{Module: deployModuleKey, Locale: "zh-CN", Group: "page", Key: "business.deploy.preview.rollbackManual", Value: "此任务需按人工回退预案处理"},
		{Module: deployModuleKey, Locale: "en-US", Group: "page", Key: "business.deploy.preview.rollbackManual", Value: "This task requires the manual rollback plan"},
		{Module: deployModuleKey, Locale: "zh-CN", Group: "page", Key: "business.deploy.preview.largeScope", Value: "本次操作涉及超过 100 个目标"},
		{Module: deployModuleKey, Locale: "en-US", Group: "page", Key: "business.deploy.preview.largeScope", Value: "This operation affects more than 100 targets"},
		{Module: deployModuleKey, Locale: "zh-CN", Group: "page", Key: "business.deploy.preview.disruptive", Value: "此操作可能中断服务"},
		{Module: deployModuleKey, Locale: "en-US", Group: "page", Key: "business.deploy.preview.disruptive", Value: "This operation may interrupt service"},
		{Module: deployModuleKey, Locale: "zh-CN", Group: "page", Key: "business.deploy.attempt.exhausted", Value: "三次执行尝试已耗尽，请查看保留的失败记录并按支持的恢复预案处理。"},
		{Module: deployModuleKey, Locale: "en-US", Group: "page", Key: "business.deploy.attempt.exhausted", Value: "The three execution attempts are exhausted. Review the retained failure and use the supported recovery plan."},
		{Module: deployModuleKey, Locale: "zh-CN", Group: "page", Key: "business.deploy.task.rollbackDraft", Value: "创建回退草稿"},
		{Module: deployModuleKey, Locale: "en-US", Group: "page", Key: "business.deploy.task.rollbackDraft", Value: "Create rollback draft"},
		{Module: deployModuleKey, Locale: "zh-CN", Group: "page", Key: "business.deploy.task.rollbackDraftConfirm", Value: "按冻结计划创建回退草稿？执行前仍需确认该草稿的预览。"},
		{Module: deployModuleKey, Locale: "en-US", Group: "page", Key: "business.deploy.task.rollbackDraftConfirm", Value: "Create a rollback draft from the frozen plan? Execution requires its own reviewed preview."},
		{Module: deployModuleKey, Locale: "zh-CN", Group: "page", Key: "business.deploy.task.refreshStale", Value: "任务刷新失败，当前显示保留数据，将继续尝试刷新。"},
		{Module: deployModuleKey, Locale: "en-US", Group: "page", Key: "business.deploy.task.refreshStale", Value: "Task refresh failed; showing retained data. Refresh will retry."},
		{Module: deployModuleKey, Locale: "zh-CN", Group: "page", Key: "business.deploy.task.rollbackNotSupported", Value: "冻结计划没有可安全执行的回退操作，请使用经确认的回退预案"},
		{Module: deployModuleKey, Locale: "en-US", Group: "page", Key: "business.deploy.task.rollbackNotSupported", Value: "This frozen plan has no supported safe inverse. Use the approved recovery plan."},
		{Module: deployModuleKey, Locale: "zh-CN", Group: "page", Key: "business.deploy.task.rollbackInvalidState", Value: "仅已成功或失败的任务可以创建回退任务"},
		{Module: deployModuleKey, Locale: "en-US", Group: "page", Key: "business.deploy.task.rollbackInvalidState", Value: "Only completed or failed tasks can create a rollback draft."},
		{Module: deployModuleKey, Locale: "zh-CN", Group: "page", Key: "business.deploy.task.rollbackFailed", Value: "创建回退任务失败"},
		{Module: deployModuleKey, Locale: "en-US", Group: "page", Key: "business.deploy.task.rollbackFailed", Value: "Failed to create the rollback draft."},
		{Module: deployModuleKey, Locale: "zh-CN", Group: "page", Key: "business.deploy.task.rollbackLookupFailed", Value: "读取回退任务失败"},
		{Module: deployModuleKey, Locale: "en-US", Group: "page", Key: "business.deploy.task.rollbackLookupFailed", Value: "Failed to load the rollback task."},
		{Module: deployModuleKey, Locale: "zh-CN", Group: "page", Key: "business.deploy.preview.required", Value: "执行前必须先获取并确认预览"},
		{Module: deployModuleKey, Locale: "en-US", Group: "page", Key: "business.deploy.preview.required", Value: "Obtain and confirm a preview before execution"},
		{Module: deployModuleKey, Locale: "zh-CN", Group: "page", Key: "business.deploy.selection.title", Value: "选择主机目标"},
		{Module: deployModuleKey, Locale: "en-US", Group: "page", Key: "business.deploy.selection.title", Value: "Select host targets"},
		{Module: deployModuleKey, Locale: "zh-CN", Group: "page", Key: "business.deploy.selection.search", Value: "搜索当前业务范围内的主机"},
		{Module: deployModuleKey, Locale: "en-US", Group: "page", Key: "business.deploy.selection.search", Value: "Search hosts in the current scope"},
		{Module: deployModuleKey, Locale: "zh-CN", Group: "page", Key: "business.deploy.selection.filterPage", Value: "筛选当前页候选项"},
		{Module: deployModuleKey, Locale: "en-US", Group: "page", Key: "business.deploy.selection.filterPage", Value: "Filter candidates on this page"},
		{Module: deployModuleKey, Locale: "zh-CN", Group: "page", Key: "business.deploy.selection.available", Value: "可选主机"},
		{Module: deployModuleKey, Locale: "en-US", Group: "page", Key: "business.deploy.selection.available", Value: "Available hosts"},
		{Module: deployModuleKey, Locale: "zh-CN", Group: "page", Key: "business.deploy.selection.selected", Value: "明确选中的主机"},
		{Module: deployModuleKey, Locale: "en-US", Group: "page", Key: "business.deploy.selection.selected", Value: "Explicit host selections"},
		{Module: deployModuleKey, Locale: "zh-CN", Group: "page", Key: "business.deploy.selection.add", Value: "添加"},
		{Module: deployModuleKey, Locale: "en-US", Group: "page", Key: "business.deploy.selection.add", Value: "Add"},
		{Module: deployModuleKey, Locale: "zh-CN", Group: "page", Key: "business.deploy.selection.remove", Value: "移除"},
		{Module: deployModuleKey, Locale: "en-US", Group: "page", Key: "business.deploy.selection.remove", Value: "Remove"},
		{Module: deployModuleKey, Locale: "zh-CN", Group: "page", Key: "business.deploy.selection.excluded", Value: "排除项"},
		{Module: deployModuleKey, Locale: "en-US", Group: "page", Key: "business.deploy.selection.excluded", Value: "Exclusions"},
		{Module: deployModuleKey, Locale: "zh-CN", Group: "page", Key: "business.deploy.selection.stale", Value: "目标已变化"},
		{Module: deployModuleKey, Locale: "en-US", Group: "page", Key: "business.deploy.selection.stale", Value: "Target changed"},
		{Module: deployModuleKey, Locale: "zh-CN", Group: "page", Key: "business.deploy.selection.groups", Value: "按条件分组添加目标"},
		{Module: deployModuleKey, Locale: "en-US", Group: "page", Key: "business.deploy.selection.groups", Value: "Add targets from condition groups"},
		{Module: deployModuleKey, Locale: "zh-CN", Group: "page", Key: "business.deploy.selection.includeScope", Value: "包含当前业务范围的全部主机"},
		{Module: deployModuleKey, Locale: "en-US", Group: "page", Key: "business.deploy.selection.includeScope", Value: "Include all hosts in the current scope"},
		{Module: deployModuleKey, Locale: "zh-CN", Group: "page", Key: "business.deploy.selection.manual", Value: "每行输入一个主机 IP 或 ID"},
		{Module: deployModuleKey, Locale: "en-US", Group: "page", Key: "business.deploy.selection.manual", Value: "Enter one host IP or ID per line"},
		{Module: deployModuleKey, Locale: "zh-CN", Group: "page", Key: "business.deploy.selection.exclude", Value: "从候选主机中排除"},
		{Module: deployModuleKey, Locale: "en-US", Group: "page", Key: "business.deploy.selection.exclude", Value: "Exclude candidate hosts"},
		{Module: deployModuleKey, Locale: "zh-CN", Group: "page", Key: "business.deploy.selection.resolve", Value: "解析并核对目标"},
		{Module: deployModuleKey, Locale: "en-US", Group: "page", Key: "business.deploy.selection.resolve", Value: "Resolve and review targets"},
		{Module: deployModuleKey, Locale: "zh-CN", Group: "page", Key: "business.deploy.selection.loadFailed", Value: "目标加载失败，请重试"},
		{Module: deployModuleKey, Locale: "en-US", Group: "page", Key: "business.deploy.selection.loadFailed", Value: "Targets unavailable; retry"},
		{Module: deployModuleKey, Locale: "zh-CN", Group: "page", Key: "business.deploy.selection.budget", Value: "选择输入超过 10000 项，请缩小范围"},
		{Module: deployModuleKey, Locale: "en-US", Group: "page", Key: "business.deploy.selection.budget", Value: "Selection exceeds 10,000 inputs; narrow the scope"},
		{Module: deployModuleKey, Locale: "zh-CN", Group: "page", Key: "business.deploy.selection.summary", Value: "已解析 {{count}} 个目标；去重 {{duplicate}}，不可用 {{unavailable}}，状态不匹配 {{ineligible}}，排除 {{excluded}}"},
		{Module: deployModuleKey, Locale: "en-US", Group: "page", Key: "business.deploy.selection.summary", Value: "Resolved {{count}}; duplicates {{duplicate}}, unavailable {{unavailable}}, ineligible {{ineligible}}, excluded {{excluded}}"},
		{Module: deployModuleKey, Locale: "zh-CN", Group: "page", Key: "business.deploy.selection.invalid", Value: "目标无效或状态不允许，请调整选择后重新解析"},
		{Module: deployModuleKey, Locale: "en-US", Group: "page", Key: "business.deploy.selection.invalid", Value: "Some targets are invalid or ineligible; adjust and resolve again"},
		{Module: deployModuleKey, Locale: "zh-CN", Group: "page", Key: "business.deploy.selection.required", Value: "请先解析并核对当前业务范围的有效目标"},
		{Module: deployModuleKey, Locale: "en-US", Group: "page", Key: "business.deploy.selection.required", Value: "Resolve valid targets in the current scope first"},
		{Module: deployModuleKey, Locale: "zh-CN", Group: "page", Key: "business.deploy.logs.attempts", Value: "尝试次数"},
		{Module: deployModuleKey, Locale: "en-US", Group: "page", Key: "business.deploy.logs.attempts", Value: "Attempts"},
		{Module: deployModuleKey, Locale: "zh-CN", Group: "page", Key: "business.deploy.logs.allSteps", Value: "查看全部步骤日志"},
		{Module: deployModuleKey, Locale: "en-US", Group: "page", Key: "business.deploy.logs.allSteps", Value: "Show all step logs"},
		{Module: deployModuleKey, Locale: "zh-CN", Group: "page", Key: "business.deploy.logs.loading", Value: "正在连接日志"},
		{Module: deployModuleKey, Locale: "en-US", Group: "page", Key: "business.deploy.logs.loading", Value: "Connecting to logs"},
		{Module: deployModuleKey, Locale: "zh-CN", Group: "page", Key: "business.deploy.logs.empty", Value: "暂无日志"},
		{Module: deployModuleKey, Locale: "en-US", Group: "page", Key: "business.deploy.logs.empty", Value: "No logs yet"},
		{Module: deployModuleKey, Locale: "zh-CN", Group: "page", Key: "business.deploy.logs.error", Value: "日志加载失败，可重新连接"},
		{Module: deployModuleKey, Locale: "en-US", Group: "page", Key: "business.deploy.logs.error", Value: "Logs unavailable; reconnect to retry"},
		{Module: deployModuleKey, Locale: "zh-CN", Group: "page", Key: "business.deploy.logs.forbidden", Value: "没有日志查看权限"},
		{Module: deployModuleKey, Locale: "en-US", Group: "page", Key: "business.deploy.logs.forbidden", Value: "Log access denied"},
		{Module: deployModuleKey, Locale: "zh-CN", Group: "page", Key: "business.deploy.logs.stale", Value: "连接中断，保留最近日志"},
		{Module: deployModuleKey, Locale: "en-US", Group: "page", Key: "business.deploy.logs.stale", Value: "Disconnected; showing retained logs"},
		{Module: deployModuleKey, Locale: "zh-CN", Group: "page", Key: "business.deploy.logs.partial", Value: "部分日志不可用"},
		{Module: deployModuleKey, Locale: "en-US", Group: "page", Key: "business.deploy.logs.partial", Value: "Some logs are unavailable"},
		{Module: deployModuleKey, Locale: "zh-CN", Group: "page", Key: "business.deploy.logs.ready", Value: "实时日志"},
		{Module: deployModuleKey, Locale: "en-US", Group: "page", Key: "business.deploy.logs.ready", Value: "Live logs"},
		{Module: deployModuleKey, Locale: "zh-CN", Group: "page", Key: "business.deploy.logs.search", Value: "搜索已保留日志"},
		{Module: deployModuleKey, Locale: "en-US", Group: "page", Key: "business.deploy.logs.search", Value: "Search retained logs"},
		{Module: deployModuleKey, Locale: "zh-CN", Group: "page", Key: "business.deploy.logs.pause", Value: "暂停追尾"},
		{Module: deployModuleKey, Locale: "en-US", Group: "page", Key: "business.deploy.logs.pause", Value: "Pause tail"},
		{Module: deployModuleKey, Locale: "zh-CN", Group: "page", Key: "business.deploy.logs.resume", Value: "恢复追尾"},
		{Module: deployModuleKey, Locale: "en-US", Group: "page", Key: "business.deploy.logs.resume", Value: "Resume tail"},
		{Module: deployModuleKey, Locale: "zh-CN", Group: "page", Key: "business.deploy.logs.wrap", Value: "自动换行"},
		{Module: deployModuleKey, Locale: "en-US", Group: "page", Key: "business.deploy.logs.wrap", Value: "Wrap lines"},
		{Module: deployModuleKey, Locale: "zh-CN", Group: "page", Key: "business.deploy.logs.rows", Value: "显示 {count} 条日志"},
		{Module: deployModuleKey, Locale: "en-US", Group: "page", Key: "business.deploy.logs.rows", Value: "Showing {count} log rows"},
		{Module: deployModuleKey, Locale: "zh-CN", Group: "page", Key: "business.deploy.logs.stream", Value: "日志流"},
		{Module: deployModuleKey, Locale: "en-US", Group: "page", Key: "business.deploy.logs.stream", Value: "Stream"},
		{Module: deployModuleKey, Locale: "zh-CN", Group: "page", Key: "business.deploy.logs.fullscreen", Value: "展开日志"},
		{Module: deployModuleKey, Locale: "en-US", Group: "page", Key: "business.deploy.logs.fullscreen", Value: "Expand logs"},
		{Module: deployModuleKey, Locale: "zh-CN", Group: "page", Key: "business.deploy.logs.exitFullscreen", Value: "退出展开"},
		{Module: deployModuleKey, Locale: "en-US", Group: "page", Key: "business.deploy.logs.exitFullscreen", Value: "Exit expanded view"},
		{Module: deployModuleKey, Locale: "zh-CN", Group: "page", Key: "business.deploy.logs.paused", Value: "已暂停追尾"},
		{Module: deployModuleKey, Locale: "en-US", Group: "page", Key: "business.deploy.logs.paused", Value: "Tail paused"},
		{Module: deployModuleKey, Locale: "zh-CN", Group: "page", Key: "business.deploy.logs.context", Value: "日志来源"},
		{Module: deployModuleKey, Locale: "en-US", Group: "page", Key: "business.deploy.logs.context", Value: "Log source"},
		{Module: deployModuleKey, Locale: "zh-CN", Group: "page", Key: "business.deploy.logs.jumpFailure", Value: "定位首个失败目标"},
		{Module: deployModuleKey, Locale: "en-US", Group: "page", Key: "business.deploy.logs.jumpFailure", Value: "Open first failed target"},
		{Module: deployModuleKey, Locale: "zh-CN", Group: "page", Key: "business.deploy.logs.host", Value: "执行目标"},
		{Module: deployModuleKey, Locale: "en-US", Group: "page", Key: "business.deploy.logs.host", Value: "Execution target"},
		{Module: deployModuleKey, Locale: "zh-CN", Group: "page", Key: "business.deploy.logs.steps", Value: "执行步骤"},
		{Module: deployModuleKey, Locale: "en-US", Group: "page", Key: "business.deploy.logs.steps", Value: "Execution steps"},
		{Module: deployModuleKey, Locale: "zh-CN", Group: "page", Key: "business.deploy.logs.jumpStep", Value: "定位步骤"},
		{Module: deployModuleKey, Locale: "en-US", Group: "page", Key: "business.deploy.logs.jumpStep", Value: "Open step"},
		{Module: deployModuleKey, Locale: "zh-CN", Group: "page", Key: "business.deploy.logs.reconnect", Value: "重新连接"},
		{Module: deployModuleKey, Locale: "en-US", Group: "page", Key: "business.deploy.logs.reconnect", Value: "Reconnect"},
		{Module: deployModuleKey, Locale: "zh-CN", Group: "page", Key: "business.deploy.logs.copy", Value: "复制可见日志"},
		{Module: deployModuleKey, Locale: "en-US", Group: "page", Key: "business.deploy.logs.copy", Value: "Copy visible logs"},
		{Module: deployModuleKey, Locale: "zh-CN", Group: "page", Key: "business.deploy.logs.copied", Value: "已复制日志"},
		{Module: deployModuleKey, Locale: "en-US", Group: "page", Key: "business.deploy.logs.copied", Value: "Logs copied"},
		{Module: deployModuleKey, Locale: "zh-CN", Group: "page", Key: "business.deploy.logs.copyFailed", Value: "无法复制日志"},
		{Module: deployModuleKey, Locale: "en-US", Group: "page", Key: "business.deploy.logs.copyFailed", Value: "Unable to copy logs"},
		{Module: deployModuleKey, Locale: "zh-CN", Group: "page", Key: "business.deploy.logs.download", Value: "下载日志快照"},
		{Module: deployModuleKey, Locale: "en-US", Group: "page", Key: "business.deploy.logs.download", Value: "Download log snapshot"},
		{Module: deployModuleKey, Locale: "zh-CN", Group: "page", Key: "business.deploy.logs.downloadFailed", Value: "无法下载日志"},
		{Module: deployModuleKey, Locale: "en-US", Group: "page", Key: "business.deploy.logs.downloadFailed", Value: "Unable to download logs"},
		{Module: deployModuleKey, Locale: "zh-CN", Group: "page", Key: "business.deploy.logs.downloadLimited", Value: "日志达到上限，本快照仅包含部分记录。"},
		{Module: deployModuleKey, Locale: "en-US", Group: "page", Key: "business.deploy.logs.downloadLimited", Value: "This snapshot is partial because a log limit was reached."},
		{Module: deployModuleKey, Locale: "zh-CN", Group: "page", Key: "business.deploy.logs.knownRedaction", Value: "已遮蔽已知任务秘密，应用输出仍可能含其他敏感内容。"},
		{Module: deployModuleKey, Locale: "en-US", Group: "page", Key: "business.deploy.logs.knownRedaction", Value: "Known task secrets are masked. Other sensitive application output may remain."},
		{Module: deployModuleKey, Locale: "zh-CN", Group: "page", Key: "business.deploy.logs.limited", Value: "仅显示最近 500 条日志"},
		{Module: deployModuleKey, Locale: "en-US", Group: "page", Key: "business.deploy.logs.limited", Value: "Showing the latest 500 log rows"},
		{Module: deployModuleKey, Locale: "zh-CN", Group: "page", Key: "business.deploy.logs.truncated", Value: "日志窗口已截断；重连可能重放或缺失"},
		{Module: deployModuleKey, Locale: "en-US", Group: "page", Key: "business.deploy.logs.truncated", Value: "Window truncated; reconnection may replay or omit logs"},
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
		{Module: deployModuleKey, Locale: "zh-CN", Group: "permission", Key: deployTaskPermPauseKey, Value: "暂停部署任务"},
		{Module: deployModuleKey, Locale: "en-US", Group: "permission", Key: deployTaskPermPauseKey, Value: "Pause deployment tasks"},
		{Module: deployModuleKey, Locale: "zh-CN", Group: "permission", Key: deployTaskPermResumeKey, Value: "恢复部署任务"},
		{Module: deployModuleKey, Locale: "en-US", Group: "permission", Key: deployTaskPermResumeKey, Value: "Resume deployment tasks"},
		{Module: deployModuleKey, Locale: "zh-CN", Group: "permission", Key: deployTaskPermRetryKey, Value: "重试部署任务"},
		{Module: deployModuleKey, Locale: "en-US", Group: "permission", Key: deployTaskPermRetryKey, Value: "Retry deployment tasks"},
		{Module: deployModuleKey, Locale: "zh-CN", Group: "permission", Key: deployTaskPermCancelKey, Value: "取消部署任务"},
		{Module: deployModuleKey, Locale: "en-US", Group: "permission", Key: deployTaskPermCancelKey, Value: "Cancel deployment tasks"},
		{Module: deployModuleKey, Locale: "zh-CN", Group: "permission", Key: deployTaskPermMarkResultKey, Value: "标记执行结果"},
		{Module: deployModuleKey, Locale: "en-US", Group: "permission", Key: deployTaskPermMarkResultKey, Value: "Mark execution result"},
		{Module: deployModuleKey, Locale: "zh-CN", Group: "permission", Key: deployTemplatePermExportKey, Value: "导出任务模板"},
		{Module: deployModuleKey, Locale: "en-US", Group: "permission", Key: deployTemplatePermExportKey, Value: "Export deployment templates"},
		{Module: deployModuleKey, Locale: "zh-CN", Group: "permission", Key: deployTemplatePermImportKey, Value: "导入任务模板"},
		{Module: deployModuleKey, Locale: "en-US", Group: "permission", Key: deployTemplatePermImportKey, Value: "Import deployment templates"},
		{Module: deployModuleKey, Locale: "zh-CN", Group: "permission", Key: deployPackagePermExportKey, Value: "导出部署包"},
		{Module: deployModuleKey, Locale: "en-US", Group: "permission", Key: deployPackagePermExportKey, Value: "Export deployment packages"},
		{Module: deployModuleKey, Locale: "zh-CN", Group: "permission", Key: deployPackagePermImportKey, Value: "导入部署包"},
		{Module: deployModuleKey, Locale: "en-US", Group: "permission", Key: deployPackagePermImportKey, Value: "Import deployment packages"},
		{Module: deployModuleKey, Locale: "zh-CN", Group: "permission", Key: deployTaskPermExportKey, Value: "导出部署任务"},
		{Module: deployModuleKey, Locale: "en-US", Group: "permission", Key: deployTaskPermExportKey, Value: "Export deployment tasks"},
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
			"lifecycle_status": bizScopeStatusActive,
			"created_at":       time.Now(),
			"updated_at":       time.Now(),
		}
		if err := db.Table("system_i18n").Create(payload).Error; err != nil {
			return err
		}
	}
	return nil
}
