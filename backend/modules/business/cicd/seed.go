package cicd

import (
	"time"

	"gorm.io/gorm"
)

const (
	cicdModuleKey        = "business.cicd"
	argocdMenuTitleKey   = "operations.cicd.argocd.menu"
	argocdMenuPath       = "/business/cicd/argocd"
	argocdRouteName      = "cicd-argocd-list"
	argocdComponentKey   = "business/cicd/argocd/ArgoCDList"
	argocdPagePermission = "business:cicd:argocd:list"
	// i18nRowWhere locates one i18n entry. The column names are fixed SQL text, the
	// values reach the statement only through bind parameters.
	i18nRowWhere = "module = ? AND locale = ? AND `key` = ?"
)

type argocdI18nSeed struct {
	Locale string
	Group  string
	Key    string
	Value  string
}

type argocdMenuSeed struct {
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

var argocdMenuSeeds = []argocdMenuSeed{
	{Key: "cicd-argocd", TitleKey: argocdMenuTitleKey, Path: argocdMenuPath, Component: argocdComponentKey, PagePerm: argocdPagePermission, Type: "C", Icon: "code", RouteName: argocdRouteName, Module: cicdModuleKey, Sort: 60},
}

// SeedMenus seeds the ArgoCD menu, admin role binding and menu i18n entries.
func SeedMenus(db *gorm.DB) error {
	if db == nil || !db.Migrator().HasTable("system_menu") {
		return nil
	}
	if err := db.Transaction(func(tx *gorm.DB) error {
		for _, seed := range argocdMenuSeeds {
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
			if err := ensureArgocdAdminBinding(tx, menuID, seed.PagePerm); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return err
	}
	return seedArgocdI18n(db)
}

func seedArgocdI18n(db *gorm.DB) error {
	if db == nil || !db.Migrator().HasTable("system_i18n") {
		return nil
	}
	seeds := []argocdI18nSeed{
		{Locale: "zh-CN", Group: "menu", Key: argocdMenuTitleKey, Value: "ArgoCD 应用发布"},
		{Locale: "en-US", Group: "menu", Key: argocdMenuTitleKey, Value: "ArgoCD GitOps"},
	}
	return db.Transaction(func(tx *gorm.DB) error {
		for _, item := range seeds {
			payload := map[string]any{
				"module":           cicdModuleKey,
				"locale":           item.Locale,
				"group_name":       item.Group,
				"key":              item.Key,
				"value":            item.Value,
				"lifecycle_status": "active",
				"updated_at":       time.Now(),
			}
			var count int64
			if err := tx.Table("system_i18n").Where(i18nRowWhere, cicdModuleKey, item.Locale, item.Key).Count(&count).Error; err != nil {
				return err
			}
			if count == 0 {
				payload["created_at"] = time.Now()
				if err := tx.Table("system_i18n").Create(payload).Error; err != nil {
					return err
				}
			} else if err := tx.Table("system_i18n").Where(i18nRowWhere, cicdModuleKey, item.Locale, item.Key).Updates(payload).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// SeedPerms seeds the ArgoCD permission keys used by the frontend.
func SeedPerms(db *gorm.DB) error {
	if db == nil || !db.Migrator().HasTable("system_i18n") {
		return nil
	}
	seeds := []argocdI18nSeed{
		{Locale: "zh-CN", Group: "permission", Key: "business.cicd.argocd.permission.list", Value: "查看 ArgoCD 应用"},
		{Locale: "en-US", Group: "permission", Key: "business.cicd.argocd.permission.list", Value: "View ArgoCD apps"},
		{Locale: "zh-CN", Group: "permission", Key: "business.cicd.argocd.permission.create", Value: "创建 ArgoCD 应用"},
		{Locale: "en-US", Group: "permission", Key: "business.cicd.argocd.permission.create", Value: "Create ArgoCD apps"},
		{Locale: "zh-CN", Group: "permission", Key: "business.cicd.argocd.permission.delete", Value: "删除 ArgoCD 应用"},
		{Locale: "en-US", Group: "permission", Key: "business.cicd.argocd.permission.delete", Value: "Delete ArgoCD apps"},
	}
	return db.Transaction(func(tx *gorm.DB) error {
		for _, item := range seeds {
			var count int64
			if err := tx.Table("system_i18n").Where(i18nRowWhere, cicdModuleKey, item.Locale, item.Key).Count(&count).Error; err != nil {
				return err
			}
			if count != 0 {
				continue
			}
			if err := tx.Table("system_i18n").Create(map[string]any{
				"module":           cicdModuleKey,
				"locale":           item.Locale,
				"group_name":       item.Group,
				"key":              item.Key,
				"value":            item.Value,
				"lifecycle_status": "active",
				"created_at":       time.Now(),
				"updated_at":       time.Now(),
			}).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func ensureArgocdAdminBinding(db *gorm.DB, menuID uint64, pagePermission string) error {
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
