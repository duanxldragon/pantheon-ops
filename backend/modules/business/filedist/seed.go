package filedist

import (
	"time"

	"gorm.io/gorm"
)

const (
	fileDistModuleKey      = "business.filedist"
	fileDistMenuTitleKey   = "operations.filedist.menu"
	fileDistMenuPath       = "/business/filedist/files"
	fileDistRouteName      = "filedist-file-list"
	fileDistComponentKey   = "business/filedist/FiledistList"
	fileDistPagePermission = "business:filedist:list"
)

type fileDistI18nSeed struct {
	Locale string
	Group  string
	Key    string
	Value  string
}

type fileDistMenuSeed struct {
	Key       string
	TitleKey  string
	Path      string
	Component string
	PagePerm  string
	Type      string
	Icon      string
	RouteName string
	Module    string
	Sort      int
}

var fileDistMenuSeeds = []fileDistMenuSeed{
	{Key: "filedist-files", TitleKey: fileDistMenuTitleKey, Path: fileDistMenuPath, Component: fileDistComponentKey, PagePerm: fileDistPagePermission, Type: "C", Icon: "storage", RouteName: fileDistRouteName, Module: fileDistModuleKey, Sort: 50},
}

func seedFileDistMenusAndI18n(db *gorm.DB) error {
	if db == nil || !db.Migrator().HasTable("system_menu") {
		return nil
	}
	if err := db.Transaction(func(tx *gorm.DB) error {
		for _, seed := range fileDistMenuSeeds {
			var menuID uint64
			if err := tx.Table("system_menu").Select("id").Where("path = ?", seed.Path).Limit(1).Pluck("id", &menuID).Error; err != nil {
				return err
			}
			payload := map[string]any{
				"parent_id":  0,
				"title_key":  seed.TitleKey,
				"path":       seed.Path,
				"component":  seed.Component,
				"page_perm":  seed.PagePerm,
				"perms":      "",
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
				if err := tx.Table("system_menu").Create(payload).Error; err != nil {
					return err
				}
				if err := tx.Table("system_menu").Select("id").Where("path = ?", seed.Path).Limit(1).Pluck("id", &menuID).Error; err != nil {
					return err
				}
			} else if err := tx.Table("system_menu").Where("id = ?", menuID).Updates(payload).Error; err != nil {
				return err
			}
			if err := ensureFileDistAdminBinding(tx, menuID, seed.PagePerm); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return err
	}
	if !db.Migrator().HasTable("system_i18n") {
		return nil
	}
	return db.Transaction(func(tx *gorm.DB) error {
		for _, item := range []fileDistI18nSeed{
			{Locale: "zh-CN", Group: "menu", Key: fileDistMenuTitleKey, Value: "文件分发"},
			{Locale: "en-US", Group: "menu", Key: fileDistMenuTitleKey, Value: "File Distribution"},
		} {
			payload := map[string]any{
				"module":           fileDistModuleKey,
				"locale":           item.Locale,
				"group_name":       item.Group,
				"key":              item.Key,
				"value":            item.Value,
				"lifecycle_status": "active",
				"updated_at":       time.Now(),
			}
			var count int64
			if err := tx.Table("system_i18n").Where("module = ? AND locale = ? AND `key` = ?", fileDistModuleKey, item.Locale, item.Key).Count(&count).Error; err != nil {
				return err
			}
			if count == 0 {
				payload["created_at"] = time.Now()
				if err := tx.Table("system_i18n").Create(payload).Error; err != nil {
					return err
				}
			} else if err := tx.Table("system_i18n").Where("module = ? AND locale = ? AND `key` = ?", fileDistModuleKey, item.Locale, item.Key).Updates(payload).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func ensureFileDistAdminBinding(db *gorm.DB, menuID uint64, pagePermission string) error {
	if menuID == 0 || !db.Migrator().HasTable("system_role") {
		return nil
	}
	var roleID uint64
	if err := db.Table("system_role").Select("id").Where("role_key = ?", "admin").Limit(1).Pluck("id", &roleID).Error; err != nil {
		return err
	}
	if roleID == 0 {
		return nil
	}
	if db.Migrator().HasTable("system_role_menu") {
		var count int64
		if err := db.Table("system_role_menu").Where("role_id = ? AND menu_id = ?", roleID, menuID).Count(&count).Error; err != nil {
			return err
		}
		if count == 0 {
			if err := db.Exec("INSERT INTO system_role_menu (role_id, menu_id) VALUES (?, ?)", roleID, menuID).Error; err != nil {
				return err
			}
		}
	}
	if pagePermission == "" || !db.Migrator().HasTable("system_role_permission") {
		return nil
	}
	var count int64
	if err := db.Table("system_role_permission").Where("role_id = ? AND permission_key = ?", roleID, pagePermission).Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return db.Exec("INSERT INTO system_role_permission (role_id, permission_key) VALUES (?, ?)", roleID, pagePermission).Error
	}
	return nil
}
