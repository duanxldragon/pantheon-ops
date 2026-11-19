package service

import (
	"time"

	"gorm.io/gorm"
)

const (
	menuTitleKeyService  = "operations.service.menu"
	permKeyServiceView   = "business.service.permission.view"
	permKeyServiceCreate = "business.service.permission.create"
	permKeyServiceUpdate = "business.service.permission.update"
	permKeyServiceDelete = "business.service.permission.delete"
	wherePathEq          = "path = ?"

	// Current + upgraded-legacy parent paths. The legacy /business/service row
	// (old id, no registered page route -> 404) is converged onto the current
	// parent path while keeping the old id, so existing role-menu bindings
	// survive an upgrade.
	serviceMenuPathCurrent = "/business/service-catalog"
	serviceMenuPathLegacy  = "/business/service"
)

type menuSeed struct {
	Key, ParentPath, ParentKey, TitleKey, Path, Component, PagePerm, Perms, Type, Icon, RouteName, Module string
	Sort                                                                                                  int
}

const (
	menuTitleKeyServiceApplication = "operations.service.application.menu"
	menuTitleKeyServiceService     = "operations.service.service.menu"
	menuTitleKeyServiceInstance    = "operations.service.instance.menu"
)

var menuSeeds = []menuSeed{
	{Key: "service", ParentPath: "", TitleKey: menuTitleKeyService, Path: "/business/service-catalog", Component: "business/service/ServiceCatalog", PagePerm: "business:service:list", Type: "C", Icon: "apps", RouteName: "service-list", Module: serviceModuleKey, Sort: 3},
	// Children are tab anchors for the single ServiceCatalog page; only the
	// parent owns the page permission so the RBAC permission tree keeps one
	// pagePerm per C-node (see RoleList pagePermissionTree keying).
	{Key: "service-applications", ParentKey: "service", TitleKey: menuTitleKeyServiceApplication, Path: "/business/service-catalog/applications", Component: "business/service/ServiceCatalog", Type: "C", Icon: "apps", RouteName: "service-applications", Module: serviceModuleKey, Sort: 3},
	{Key: "service-services", ParentKey: "service", TitleKey: menuTitleKeyServiceService, Path: "/business/service-catalog/services", Component: "business/service/ServiceCatalog", Type: "C", Icon: "apps", RouteName: "service-services", Module: serviceModuleKey, Sort: 4},
	{Key: "service-instances", ParentKey: "service", TitleKey: menuTitleKeyServiceInstance, Path: "/business/service-catalog/instances", Component: "business/service/ServiceCatalog", Type: "C", Icon: "apps", RouteName: "service-instances", Module: serviceModuleKey, Sort: 5},
	{Key: "service-view", ParentKey: "service", TitleKey: permKeyServiceView, Perms: "business:service:view", Type: "F", Module: serviceModuleKey, Sort: 1},
	{Key: "service-create", ParentKey: "service", TitleKey: permKeyServiceCreate, Perms: "business:service:create", Type: "F", Module: serviceModuleKey, Sort: 2},
	{Key: "service-update", ParentKey: "service", TitleKey: permKeyServiceUpdate, Perms: "business:service:update", Type: "F", Module: serviceModuleKey, Sort: 3},
	{Key: "service-delete", ParentKey: "service", TitleKey: permKeyServiceDelete, Perms: "business:service:delete", Type: "F", Module: serviceModuleKey, Sort: 4},
}

func seedMenus(db *gorm.DB) error {
	if db == nil || !db.Migrator().HasTable("system_menu") {
		return nil
	}
	return db.Transaction(func(tx *gorm.DB) error {
		if err := mergeLegacyServiceMenu(tx); err != nil {
			return err
		}
		ids := map[string]uint64{}
		for _, seed := range menuSeeds {
			id := findSeedMenuID(tx, seed)
			parentID := seedMenuParentID(tx, seed, ids)
			id, err := upsertSeedMenu(tx, seed, parentID, id)
			if err != nil {
				return err
			}
			if seed.Key != "" {
				ids[seed.Key] = id
			}
		}
		return nil
	})
}

// mergeLegacyServiceMenu converges the upgraded legacy single-row service
// parent onto the current path while keeping the OLD menu id, so pre-existing
// role-menu bindings (incl. tenant_id) survive. Only module=business.service
// rows are ever touched, and it never deletes a same-named custom menu.
// Idempotent over consecutive seeds/starts.
func mergeLegacyServiceMenu(tx *gorm.DB) error {
	type row struct{ ID uint64 }
	canonical, legacy := row{}, row{}
	if err := tx.Table("system_menu").Where("path = ? AND module = ?", serviceMenuPathCurrent, serviceModuleKey).Limit(1).Scan(&canonical).Error; err != nil {
		return err
	}
	if err := tx.Table("system_menu").Where("path = ? AND module = ?", serviceMenuPathLegacy, serviceModuleKey).Limit(1).Scan(&legacy).Error; err != nil {
		return err
	}

	// Both coexist and differ -> reparent ALL children (incl user-custom ones,
	// so none dangles under the soon-deleted parent) + absorb bindings onto legacy.
	if legacy.ID != 0 && canonical.ID != 0 && legacy.ID != canonical.ID {
		if err := tx.Table("system_menu").
			Where("parent_id = ?", canonical.ID).
			Update("parent_id", legacy.ID).Error; err != nil {
			return err
		}
		// Composite PK (tenant_id, role_id, menu_id) — keep tenant_id.
		if err := tx.Exec("INSERT INTO system_role_menu (tenant_id, role_id, menu_id) "+
			"SELECT s.tenant_id, s.role_id, ? FROM system_role_menu s "+
			"WHERE s.menu_id = ? AND NOT EXISTS (SELECT 1 FROM system_role_menu d "+
			"WHERE d.tenant_id = s.tenant_id AND d.role_id = s.role_id AND d.menu_id = ?)",
			legacy.ID, canonical.ID, legacy.ID).Error; err != nil {
			return err
		}
		if err := tx.Table("system_role_menu").Where("menu_id = ?", canonical.ID).Delete(nil).Error; err != nil {
			return err
		}
		if err := tx.Table("system_menu").Where("id = ? AND module = ?", canonical.ID, serviceModuleKey).Delete(nil).Error; err != nil {
			return err
		}
		canonical.ID = 0
	}

	// A surviving legacy row (whether promoted alone or the merge target) owns
	// the canonical path going forward so findSeedMenuID resolves to its old id.
	if legacy.ID != 0 && canonical.ID == 0 {
		if err := tx.Table("system_menu").
			Where("id = ? AND module = ?", legacy.ID, serviceModuleKey).
			Update("path", serviceMenuPathCurrent).Error; err != nil {
			return err
		}
	}
	return nil
}

func findSeedMenuID(tx *gorm.DB, seed menuSeed) uint64 {
	var id uint64
	q := tx.Table("system_menu").Select("id")
	if seed.Path != "" {
		q = q.Where(wherePathEq, seed.Path)
	} else {
		q = q.Where("perms = ?", seed.Perms)
	}
	_ = q.Limit(1).Pluck("id", &id).Error
	return id
}

func seedMenuParentID(tx *gorm.DB, seed menuSeed, ids map[string]uint64) uint64 {
	parentID := uint64(0)
	if seed.ParentKey != "" {
		parentID = ids[seed.ParentKey]
	}
	if parentID == 0 && seed.ParentPath != "" {
		_ = tx.Table("system_menu").Select("id").Where(wherePathEq, seed.ParentPath).Limit(1).Pluck("id", &parentID).Error
	}
	return parentID
}

func upsertSeedMenu(tx *gorm.DB, seed menuSeed, parentID uint64, id uint64) (uint64, error) {
	moduleKey := seed.Module
	if moduleKey == "" {
		moduleKey = serviceModuleKey
	}
	payload := map[string]any{"parent_id": parentID, "title_key": seed.TitleKey, "path": seed.Path, "component": seed.Component, "page_perm": seed.PagePerm, "perms": seed.Perms, "type": seed.Type, "icon": seed.Icon, "route_name": seed.RouteName, "module": moduleKey, "sort": seed.Sort, "is_visible": 1, "is_cache": 0, "updated_at": time.Now()}
	if id != 0 {
		if err := tx.Table("system_menu").Where("id = ?", id).Updates(payload).Error; err != nil {
			return id, err
		}
		return id, nil
	}
	payload["created_at"] = time.Now()
	if err := tx.Table("system_menu").Create(payload).Error; err != nil {
		return id, err
	}
	_ = tx.Table("system_menu").Select("id").Where(wherePathEq, seed.Path).Limit(1).Pluck("id", &id).Error
	return id, nil
}

func seedI18n(db *gorm.DB) error {
	if db == nil || !db.Migrator().HasTable("system_i18n") {
		return nil
	}
	seeds := []struct {
		Locale string
		Group  string
		Key    string
		Value  string
	}{
		{Locale: "zh-CN", Group: "menu", Key: menuTitleKeyService, Value: "服务目录"},
		{Locale: "en-US", Group: "menu", Key: menuTitleKeyService, Value: "Service Catalog"},
		{Locale: "zh-CN", Group: "menu", Key: menuTitleKeyServiceApplication, Value: "应用"},
		{Locale: "en-US", Group: "menu", Key: menuTitleKeyServiceApplication, Value: "Applications"},
		{Locale: "zh-CN", Group: "menu", Key: menuTitleKeyServiceService, Value: "服务"},
		{Locale: "en-US", Group: "menu", Key: menuTitleKeyServiceService, Value: "Services"},
		{Locale: "zh-CN", Group: "menu", Key: menuTitleKeyServiceInstance, Value: "服务实例"},
		{Locale: "en-US", Group: "menu", Key: menuTitleKeyServiceInstance, Value: "Service Instances"},
		{Locale: "zh-CN", Group: "permission", Key: permKeyServiceView, Value: "查看服务"},
		{Locale: "en-US", Group: "permission", Key: permKeyServiceView, Value: "View services"},
		{Locale: "zh-CN", Group: "permission", Key: permKeyServiceCreate, Value: "创建服务"},
		{Locale: "en-US", Group: "permission", Key: permKeyServiceCreate, Value: "Create services"},
		{Locale: "zh-CN", Group: "permission", Key: permKeyServiceUpdate, Value: "编辑服务"},
		{Locale: "en-US", Group: "permission", Key: permKeyServiceUpdate, Value: "Update services"},
		{Locale: "zh-CN", Group: "permission", Key: permKeyServiceDelete, Value: "删除服务"},
		{Locale: "en-US", Group: "permission", Key: permKeyServiceDelete, Value: "Delete services"},
		{Locale: "zh-CN", Group: "error", Key: errStateStale, Value: "实例状态已变化，请刷新列表并重新核对后操作。"},
		{Locale: "en-US", Group: "error", Key: errStateStale, Value: "The instance state has changed. Refresh the list and review it before retrying."},
		{Locale: "zh-CN", Group: "error", Key: errStateInvalid, Value: "当前实例状态不允许此操作。"},
		{Locale: "en-US", Group: "error", Key: errStateInvalid, Value: "The current instance state does not allow this operation."},
		{Locale: "zh-CN", Group: "error", Key: errTargetInvalid, Value: "目标信息无效，请按服务类型选择完整且属于同一业务域的目标。"},
		{Locale: "en-US", Group: "error", Key: errTargetInvalid, Value: "Invalid target. Select a complete target compatible with the service in the same business scope."},
		{Locale: "zh-CN", Group: "error", Key: errTargetNotFound, Value: "目标不可用或无权访问，请重新选择。"},
		{Locale: "en-US", Group: "error", Key: errTargetNotFound, Value: "The target is unavailable or inaccessible. Select another target."},
		{Locale: "zh-CN", Group: "error", Key: errTargetConflict, Value: "目标绑定存在冲突或当前不能安全更换，请检查已有绑定、运行状态和部署引用。"},
		{Locale: "en-US", Group: "error", Key: errTargetConflict, Value: "The binding conflicts or cannot be safely changed. Check existing bindings, lifecycle state, and deployment references."},
		{Locale: "zh-CN", Group: "error", Key: "business.service.instance_in_use", Value: "实例仍被执行中的部署引用，不能删除或退役"},
		{Locale: "en-US", Group: "error", Key: "business.service.instance_in_use", Value: "The instance is referenced by an executing deployment and cannot be deleted or retired"},
		{Locale: "zh-CN", Group: "error", Key: "business.service.deployment_reader_not_configured", Value: "部署引用校验未配置，不能删除或退役"},
		{Locale: "en-US", Group: "error", Key: "business.service.deployment_reader_not_configured", Value: "Deployment reference validation is unavailable; deletion and retirement are blocked"},
	}
	return db.Transaction(func(tx *gorm.DB) error {
		for _, item := range seeds {
			var count int64
			if err := tx.Table("system_i18n").Where("module = ? AND locale = ? AND `key` = ?", serviceModuleKey, item.Locale, item.Key).Count(&count).Error; err != nil {
				return err
			}
			payload := map[string]any{"module": serviceModuleKey, "locale": item.Locale, "group_name": item.Group, "key": item.Key, "value": item.Value, "lifecycle_status": "active", "updated_at": time.Now()}
			if count == 0 {
				payload["created_at"] = time.Now()
				if err := tx.Table("system_i18n").Create(payload).Error; err != nil {
					return err
				}
			} else if err := tx.Table("system_i18n").Where("module = ? AND locale = ? AND `key` = ?", serviceModuleKey, item.Locale, item.Key).Updates(payload).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
