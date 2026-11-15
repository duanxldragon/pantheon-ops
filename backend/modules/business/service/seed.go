package service

import (
	"strings"
	"time"

	"gorm.io/gorm"
)

const (
	menuTitleKeyService     = "operations.service.menu"
	permKeyServiceView      = "business.service.permission.view"
	permKeyServiceCreate    = "business.service.permission.create"
	permKeyServiceUpdate    = "business.service.permission.update"
	permKeyServiceDelete    = "business.service.permission.delete"
	wherePathEq             = "path = ?"
	menuTitleKeyApplication = "operations.service.application.menu"
	menuTitleKeyServiceList = "operations.service.service.menu"
	menuTitleKeyInstance    = "operations.service.instance.menu"
)

type menuSeed struct {
	Key, ParentPath, ParentKey, TitleKey, Path, Component, PagePerm, Perms, Type, Icon, RouteName, Module string
	Sort                                                                                                  int
}

func bindServiceAdmin(tx *gorm.DB, menuID uint64, seed menuSeed) error {
	if menuID == 0 || !tx.Migrator().HasTable("system_role") {
		return nil
	}
	var roleID uint64
	if err := tx.Table("system_role").Select("id").Where("role_key = ?", "admin").Limit(1).Pluck("id", &roleID).Error; err != nil {
		return err
	}
	if roleID == 0 {
		return nil
	}
	if seed.Type == "C" && tx.Migrator().HasTable("system_role_menu") {
		var count int64
		if err := tx.Table("system_role_menu").Where("role_id = ? AND menu_id = ?", roleID, menuID).Count(&count).Error; err != nil {
			return err
		}
		if count == 0 {
			if err := tx.Exec("INSERT INTO system_role_menu (role_id, menu_id) VALUES (?, ?)", roleID, menuID).Error; err != nil {
				return err
			}
		}
	}
	if !tx.Migrator().HasTable("system_role_permission") {
		return nil
	}
	for _, key := range []string{strings.TrimSpace(seed.PagePerm), strings.TrimSpace(seed.Perms)} {
		if key == "" {
			continue
		}
		var count int64
		if err := tx.Table("system_role_permission").Where("role_id = ? AND permission_key = ?", roleID, key).Count(&count).Error; err != nil {
			return err
		}
		if count == 0 {
			if err := tx.Exec("INSERT INTO system_role_permission (role_id, permission_key) VALUES (?, ?)", roleID, key).Error; err != nil {
				return err
			}
		}
	}
	return nil
}

var menuSeeds = []menuSeed{
	{Key: "service", ParentPath: "", TitleKey: menuTitleKeyService, Path: "/business/service-catalog", Type: "M", Icon: "apps", RouteName: "service-catalog", Module: serviceModuleKey, Sort: 2},
	{Key: "service-applications", ParentKey: "service", TitleKey: menuTitleKeyApplication, Path: "/business/service-catalog/applications", Component: "business/service/ApplicationList", PagePerm: "business:service:list", Type: "C", RouteName: "service-application-list", Module: serviceModuleKey, Sort: 1},
	{Key: "service-services", ParentKey: "service", TitleKey: menuTitleKeyServiceList, Path: "/business/service-catalog/services", Component: "business/service/ServiceList", PagePerm: "business:service:list", Type: "C", RouteName: "service-service-list", Module: serviceModuleKey, Sort: 2},
	{Key: "service-instances", ParentKey: "service", TitleKey: menuTitleKeyInstance, Path: "/business/service-catalog/instances", Component: "business/serviceinstance/ServiceInstanceList", PagePerm: "business:service:list", Type: "C", RouteName: "service-instance-list", Module: serviceModuleKey, Sort: 3},
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
			if err := bindServiceAdmin(tx, id, seed); err != nil {
				return err
			}
		}
		return nil
	})
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
		{Locale: "zh-CN", Group: "menu", Key: menuTitleKeyApplication, Value: "应用"},
		{Locale: "en-US", Group: "menu", Key: menuTitleKeyApplication, Value: "Applications"},
		{Locale: "zh-CN", Group: "menu", Key: menuTitleKeyServiceList, Value: "服务"},
		{Locale: "en-US", Group: "menu", Key: menuTitleKeyServiceList, Value: "Services"},
		{Locale: "zh-CN", Group: "menu", Key: menuTitleKeyInstance, Value: "服务实例"},
		{Locale: "en-US", Group: "menu", Key: menuTitleKeyInstance, Value: "Service Instances"},
		{Locale: "zh-CN", Group: "permission", Key: permKeyServiceView, Value: "查看服务"},
		{Locale: "en-US", Group: "permission", Key: permKeyServiceView, Value: "View services"},
		{Locale: "zh-CN", Group: "permission", Key: permKeyServiceCreate, Value: "创建服务"},
		{Locale: "en-US", Group: "permission", Key: permKeyServiceCreate, Value: "Create services"},
		{Locale: "zh-CN", Group: "permission", Key: permKeyServiceUpdate, Value: "编辑服务"},
		{Locale: "en-US", Group: "permission", Key: permKeyServiceUpdate, Value: "Update services"},
		{Locale: "zh-CN", Group: "permission", Key: permKeyServiceDelete, Value: "删除服务"},
		{Locale: "en-US", Group: "permission", Key: permKeyServiceDelete, Value: "Delete services"},
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
