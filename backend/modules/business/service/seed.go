package service

import (
	"time"

	"gorm.io/gorm"
)

type menuSeed struct {
	Key, ParentPath, ParentKey, TitleKey, Path, Component, PagePerm, Perms, Type, Icon, RouteName, Module string
	Sort                                                                                                  int
}

var menuSeeds = []menuSeed{
	{Key: "service", ParentPath: "", TitleKey: "operations.service.menu", Path: "/business/service", Component: "business/service/ServiceList", PagePerm: "business:service:list", Type: "C", Icon: "apps", RouteName: "service-list", Module: "business.service", Sort: 30},
	{Key: "service-view", ParentKey: "service", TitleKey: "business.service.permission.view", Perms: "business:service:view", Type: "F", Module: "business.service", Sort: 1},
	{Key: "service-create", ParentKey: "service", TitleKey: "business.service.permission.create", Perms: "business:service:create", Type: "F", Module: "business.service", Sort: 2},
	{Key: "service-update", ParentKey: "service", TitleKey: "business.service.permission.update", Perms: "business:service:update", Type: "F", Module: "business.service", Sort: 3},
	{Key: "service-delete", ParentKey: "service", TitleKey: "business.service.permission.delete", Perms: "business:service:delete", Type: "F", Module: "business.service", Sort: 4},
}

func seedMenus(db *gorm.DB) error {
	if db == nil || !db.Migrator().HasTable("system_menu") {
		return nil
	}
	return db.Transaction(func(tx *gorm.DB) error {
		ids := map[string]uint64{}
		for _, seed := range menuSeeds {
			var id uint64
			q := tx.Table("system_menu").Select("id")
			if seed.Path != "" {
				q = q.Where("path = ?", seed.Path)
			} else {
				q = q.Where("perms = ?", seed.Perms)
			}
			_ = q.Limit(1).Pluck("id", &id).Error
			parentID := uint64(0)
			if seed.ParentKey != "" {
				parentID = ids[seed.ParentKey]
			}
			if parentID == 0 && seed.ParentPath != "" {
				_ = tx.Table("system_menu").Select("id").Where("path = ?", seed.ParentPath).Limit(1).Pluck("id", &parentID).Error
			}
			moduleKey := seed.Module
			if moduleKey == "" {
				moduleKey = serviceModuleKey
			}
			payload := map[string]any{"parent_id": parentID, "title_key": seed.TitleKey, "path": seed.Path, "component": seed.Component, "page_perm": seed.PagePerm, "perms": seed.Perms, "type": seed.Type, "icon": seed.Icon, "route_name": seed.RouteName, "module": moduleKey, "sort": seed.Sort, "is_visible": 1, "is_cache": 0, "updated_at": time.Now()}
			if id == 0 {
				payload["created_at"] = time.Now()
				if err := tx.Table("system_menu").Create(payload).Error; err != nil {
					return err
				}
				_ = tx.Table("system_menu").Select("id").Where("path = ?", seed.Path).Limit(1).Pluck("id", &id).Error
			} else if err := tx.Table("system_menu").Where("id = ?", id).Updates(payload).Error; err != nil {
				return err
			}
			if seed.Key != "" {
				ids[seed.Key] = id
			}
		}
		return nil
	})
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
		{Locale: "zh-CN", Group: "menu", Key: "operations.service.menu", Value: "服务目录"},
		{Locale: "en-US", Group: "menu", Key: "operations.service.menu", Value: "Service Catalog"},
		{Locale: "zh-CN", Group: "permission", Key: "business.service.permission.view", Value: "查看服务"},
		{Locale: "en-US", Group: "permission", Key: "business.service.permission.view", Value: "View services"},
		{Locale: "zh-CN", Group: "permission", Key: "business.service.permission.create", Value: "创建服务"},
		{Locale: "en-US", Group: "permission", Key: "business.service.permission.create", Value: "Create services"},
		{Locale: "zh-CN", Group: "permission", Key: "business.service.permission.update", Value: "编辑服务"},
		{Locale: "en-US", Group: "permission", Key: "business.service.permission.update", Value: "Update services"},
		{Locale: "zh-CN", Group: "permission", Key: "business.service.permission.delete", Value: "删除服务"},
		{Locale: "en-US", Group: "permission", Key: "business.service.permission.delete", Value: "Delete services"},
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
