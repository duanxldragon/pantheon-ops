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
)

type menuSeed struct {
	Key, ParentPath, ParentKey, TitleKey, Path, Component, PagePerm, Perms, Type, Icon, RouteName, Module string
	Sort                                                                                                  int
}

var menuSeeds = []menuSeed{
	{Key: "service", ParentPath: "", TitleKey: menuTitleKeyService, Path: "/business/service", Component: "business/service/ServiceCatalog", PagePerm: "business:service:list", Type: "C", Icon: "apps", RouteName: "service-list", Module: serviceModuleKey, Sort: 2},
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
