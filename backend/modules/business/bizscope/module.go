package bizscope

import (
	"strings"
	"time"

	"pantheon-ops/backend/internal/middleware"
	"pantheon-ops/backend/pkg/database"
	"pantheon-ops/backend/pkg/contracts"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type menuSeed struct {
	Key        string
	ParentPath string
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
}

var seeds = []menuSeed{
	{
		Key:        "bizscope",
		ParentPath: "/operations",
		TitleKey:   "operations.bizscope.menu",
		Path:       "/operations/business-scope",
		Component:  "business/bizscope/BizScopeList",
		PagePerm:   "business:bizscope:list",
		Type:       "C",
		Icon:       "apps",
		RouteName:  "bizscope-list",
		Module:     "business.bizscope",
		Sort:       2,
	},
	{
		Key:       "bizscope-view",
		ParentKey: "bizscope",
		TitleKey:  "business.bizscope.permission.view",
		Perms:     "business:bizscope:view",
		Type:      "F",
		Module:    "business.bizscope",
		Sort:      1,
	},
	{
		Key:       "bizscope-create",
		ParentKey: "bizscope",
		TitleKey:  "business.bizscope.permission.create",
		Perms:     "business:bizscope:create",
		Type:      "F",
		Module:    "business.bizscope",
		Sort:      2,
	},
	{
		Key:       "bizscope-update",
		ParentKey: "bizscope",
		TitleKey:  "business.bizscope.permission.update",
		Perms:     "business:bizscope:update",
		Type:      "F",
		Module:    "business.bizscope",
		Sort:      3,
	},
	{
		Key:       "bizscope-delete",
		ParentKey: "bizscope",
		TitleKey:  "business.bizscope.permission.delete",
		Perms:     "business:bizscope:delete",
		Type:      "F",
		Module:    "business.bizscope",
		Sort:      4,
	},
}

type i18nSeed struct {
	Module string
	Locale string
	Group  string
	Key    string
	Value  string
}

var i18nSeeds = []i18nSeed{
	{Module: "business.bizscope", Locale: "zh-CN", Group: "menu", Key: "operations.bizscope.menu", Value: "业务域"},
	{Module: "business.bizscope", Locale: "en-US", Group: "menu", Key: "operations.bizscope.menu", Value: "Business Scope"},
	{Module: "business.bizscope", Locale: "zh-CN", Group: "page", Key: "operations.bizscope.detail", Value: "业务域详情"},
	{Module: "business.bizscope", Locale: "en-US", Group: "page", Key: "operations.bizscope.detail", Value: "Business Scope Detail"},
	{Module: "business.bizscope", Locale: "zh-CN", Group: "page", Key: "business.bizscope.title", Value: "业务域"},
	{Module: "business.bizscope", Locale: "en-US", Group: "page", Key: "business.bizscope.title", Value: "Business Scope"},
	{Module: "business.bizscope", Locale: "zh-CN", Group: "page", Key: "business.bizscope.hero.eyebrow", Value: "运维平台 / 业务域"},
	{Module: "business.bizscope", Locale: "en-US", Group: "page", Key: "business.bizscope.hero.eyebrow", Value: "Operations / Business Scope"},
	{Module: "business.bizscope", Locale: "zh-CN", Group: "page", Key: "business.bizscope.hero.title", Value: "独立治理业务域，并作为主机分配与部署信任来源"},
	{Module: "business.bizscope", Locale: "en-US", Group: "page", Key: "business.bizscope.hero.title", Value: "Govern business scopes as the source of host assignment and deployment trust"},
	{Module: "business.bizscope", Locale: "zh-CN", Group: "page", Key: "business.bizscope.hero.total", Value: "业务域总数"},
	{Module: "business.bizscope", Locale: "en-US", Group: "page", Key: "business.bizscope.hero.total", Value: "Total Scopes"},
	{Module: "business.bizscope", Locale: "zh-CN", Group: "page", Key: "business.bizscope.hero.active", Value: "启用中"},
	{Module: "business.bizscope", Locale: "en-US", Group: "page", Key: "business.bizscope.hero.active", Value: "Active"},
	{Module: "business.bizscope", Locale: "zh-CN", Group: "page", Key: "business.bizscope.hero.prod", Value: "生产域"},
	{Module: "business.bizscope", Locale: "en-US", Group: "page", Key: "business.bizscope.hero.prod", Value: "Production"},
	{Module: "business.bizscope", Locale: "zh-CN", Group: "page", Key: "business.bizscope.field.code", Value: "业务域编码"},
	{Module: "business.bizscope", Locale: "en-US", Group: "page", Key: "business.bizscope.field.code", Value: "Scope Code"},
	{Module: "business.bizscope", Locale: "zh-CN", Group: "placeholder", Key: "business.bizscope.field.codePlaceholder", Value: "请输入业务域编码"},
	{Module: "business.bizscope", Locale: "en-US", Group: "placeholder", Key: "business.bizscope.field.codePlaceholder", Value: "Enter scope code"},
	{Module: "business.bizscope", Locale: "zh-CN", Group: "page", Key: "business.bizscope.field.name", Value: "业务域名称"},
	{Module: "business.bizscope", Locale: "en-US", Group: "page", Key: "business.bizscope.field.name", Value: "Scope Name"},
	{Module: "business.bizscope", Locale: "zh-CN", Group: "placeholder", Key: "business.bizscope.field.namePlaceholder", Value: "请输入业务域名称"},
	{Module: "business.bizscope", Locale: "en-US", Group: "placeholder", Key: "business.bizscope.field.namePlaceholder", Value: "Enter scope name"},
	{Module: "business.bizscope", Locale: "zh-CN", Group: "page", Key: "business.bizscope.field.owner", Value: "负责人"},
	{Module: "business.bizscope", Locale: "en-US", Group: "page", Key: "business.bizscope.field.owner", Value: "Owner"},
	{Module: "business.bizscope", Locale: "zh-CN", Group: "placeholder", Key: "business.bizscope.field.ownerPlaceholder", Value: "请输入负责人"},
	{Module: "business.bizscope", Locale: "en-US", Group: "placeholder", Key: "business.bizscope.field.ownerPlaceholder", Value: "Enter owner"},
	{Module: "business.bizscope", Locale: "zh-CN", Group: "page", Key: "business.bizscope.field.environment", Value: "环境"},
	{Module: "business.bizscope", Locale: "en-US", Group: "page", Key: "business.bizscope.field.environment", Value: "Environment"},
	{Module: "business.bizscope", Locale: "zh-CN", Group: "page", Key: "business.bizscope.environment.dev", Value: "开发"},
	{Module: "business.bizscope", Locale: "en-US", Group: "page", Key: "business.bizscope.environment.dev", Value: "Development"},
	{Module: "business.bizscope", Locale: "zh-CN", Group: "page", Key: "business.bizscope.environment.test", Value: "测试"},
	{Module: "business.bizscope", Locale: "en-US", Group: "page", Key: "business.bizscope.environment.test", Value: "Testing"},
	{Module: "business.bizscope", Locale: "zh-CN", Group: "page", Key: "business.bizscope.environment.prod", Value: "生产"},
	{Module: "business.bizscope", Locale: "en-US", Group: "page", Key: "business.bizscope.environment.prod", Value: "Production"},
	{Module: "business.bizscope", Locale: "zh-CN", Group: "page", Key: "business.bizscope.field.status", Value: "状态"},
	{Module: "business.bizscope", Locale: "en-US", Group: "page", Key: "business.bizscope.field.status", Value: "Status"},
	{Module: "business.bizscope", Locale: "zh-CN", Group: "page", Key: "business.bizscope.status.active", Value: "启用"},
	{Module: "business.bizscope", Locale: "en-US", Group: "page", Key: "business.bizscope.status.active", Value: "Active"},
	{Module: "business.bizscope", Locale: "zh-CN", Group: "page", Key: "business.bizscope.status.inactive", Value: "停用"},
	{Module: "business.bizscope", Locale: "en-US", Group: "page", Key: "business.bizscope.status.inactive", Value: "Inactive"},
	{Module: "business.bizscope", Locale: "zh-CN", Group: "page", Key: "business.bizscope.field.remark", Value: "备注"},
	{Module: "business.bizscope", Locale: "en-US", Group: "page", Key: "business.bizscope.field.remark", Value: "Remark"},
	{Module: "business.bizscope", Locale: "zh-CN", Group: "placeholder", Key: "business.bizscope.field.remarkPlaceholder", Value: "请输入备注"},
	{Module: "business.bizscope", Locale: "en-US", Group: "placeholder", Key: "business.bizscope.field.remarkPlaceholder", Value: "Enter remark"},
	{Module: "business.bizscope", Locale: "zh-CN", Group: "page", Key: "business.bizscope.createTitle", Value: "新增业务域"},
	{Module: "business.bizscope", Locale: "en-US", Group: "page", Key: "business.bizscope.createTitle", Value: "Create Business Scope"},
	{Module: "business.bizscope", Locale: "zh-CN", Group: "page", Key: "business.bizscope.editTitle", Value: "编辑业务域"},
	{Module: "business.bizscope", Locale: "en-US", Group: "page", Key: "business.bizscope.editTitle", Value: "Edit Business Scope"},
	{Module: "business.bizscope", Locale: "zh-CN", Group: "page", Key: "business.bizscope.deleteConfirm", Value: "确认删除该业务域？已绑定主机时不能删除。"},
	{Module: "business.bizscope", Locale: "en-US", Group: "page", Key: "business.bizscope.deleteConfirm", Value: "Delete this business scope? Scopes bound to hosts cannot be deleted."},
	{Module: "business.bizscope", Locale: "zh-CN", Group: "page", Key: "business.bizscope.empty", Value: "暂无业务域"},
	{Module: "business.bizscope", Locale: "en-US", Group: "page", Key: "business.bizscope.empty", Value: "No business scopes"},
	{Module: "business.bizscope", Locale: "zh-CN", Group: "permission", Key: "business.bizscope.permission.view", Value: "查看业务域"},
	{Module: "business.bizscope", Locale: "en-US", Group: "permission", Key: "business.bizscope.permission.view", Value: "View business scope"},
	{Module: "business.bizscope", Locale: "zh-CN", Group: "permission", Key: "business.bizscope.permission.create", Value: "新增业务域"},
	{Module: "business.bizscope", Locale: "en-US", Group: "permission", Key: "business.bizscope.permission.create", Value: "Create business scope"},
	{Module: "business.bizscope", Locale: "zh-CN", Group: "permission", Key: "business.bizscope.permission.update", Value: "编辑业务域"},
	{Module: "business.bizscope", Locale: "en-US", Group: "permission", Key: "business.bizscope.permission.update", Value: "Update business scope"},
	{Module: "business.bizscope", Locale: "zh-CN", Group: "permission", Key: "business.bizscope.permission.delete", Value: "删除业务域"},
	{Module: "business.bizscope", Locale: "en-US", Group: "permission", Key: "business.bizscope.permission.delete", Value: "Delete business scope"},
	{Module: "business.bizscope", Locale: "zh-CN", Group: "audit", Key: "business.bizscope.audit.create", Value: "新增业务域"},
	{Module: "business.bizscope", Locale: "en-US", Group: "audit", Key: "business.bizscope.audit.create", Value: "Create business scope"},
	{Module: "business.bizscope", Locale: "zh-CN", Group: "audit", Key: "business.bizscope.audit.update", Value: "编辑业务域"},
	{Module: "business.bizscope", Locale: "en-US", Group: "audit", Key: "business.bizscope.audit.update", Value: "Update business scope"},
	{Module: "business.bizscope", Locale: "zh-CN", Group: "audit", Key: "business.bizscope.audit.delete", Value: "删除业务域"},
	{Module: "business.bizscope", Locale: "en-US", Group: "audit", Key: "business.bizscope.audit.delete", Value: "Delete business scope"},
	{Module: "business.bizscope", Locale: "zh-CN", Group: "error", Key: "bizscope.code_exists", Value: "业务域编码已存在"},
	{Module: "business.bizscope", Locale: "en-US", Group: "error", Key: "bizscope.code_exists", Value: "Business scope code already exists"},
	{Module: "business.bizscope", Locale: "zh-CN", Group: "error", Key: "bizscope.in_use", Value: "业务域已绑定主机，不能删除"},
	{Module: "business.bizscope", Locale: "en-US", Group: "error", Key: "bizscope.in_use", Value: "The business scope is bound to hosts and cannot be deleted"},
	{Module: "business.bizscope", Locale: "zh-CN", Group: "error", Key: "bizscope.not_found", Value: "业务域不存在"},
	{Module: "business.bizscope", Locale: "en-US", Group: "error", Key: "bizscope.not_found", Value: "Business scope does not exist"},
}

func InitBizScopeModule(r *gin.RouterGroup, db *gorm.DB) {
	service := NewService(db)
	handler := NewHandler(service)

	contracts.RegisterBackendModules(r, db, contracts.FuncModule{
		ModuleName:    "business.bizscope",
		MigrateFunc:   func(_ *gorm.DB) error { return service.Migrate() },
		SeedMenusFunc: seedMenus,
		SeedI18nFunc:  seedI18n,
		Register: func(r *gin.RouterGroup) {
			protected := r.Group("/business/bizscope").Use(middleware.TokenAuthMiddleware(database.RDB)).Use(middleware.CasbinMiddleware())
			{
				protected.GET("/list", handler.List)
				protected.GET("/options", handler.Options)
				protected.GET("/:id", handler.Detail)
				protected.GET("/:id/hosts", handler.Hosts)
				protected.GET("/:id/available-hosts", handler.AvailableHosts)
				protected.POST("/:id/hosts/bind", handler.BindHosts)
				protected.DELETE("/:id/hosts/:hostId", handler.UnbindHost)
				protected.POST("", handler.Create)
				protected.PUT("/:id", handler.Update)
				protected.DELETE("/:id", handler.Delete)
			}
		},
	})
}

func seedMenus(db *gorm.DB) error {
	if db == nil || !db.Migrator().HasTable("system_menu") {
		return nil
	}
	return db.Transaction(func(tx *gorm.DB) error {
		keyToID := make(map[string]uint64, len(seeds))
		for _, seed := range seeds {
			if _, err := ensureMenuSeed(tx, keyToID, seed); err != nil {
				return err
			}
		}
		return nil
	})
}

func ensureMenuSeed(tx *gorm.DB, keyToID map[string]uint64, seed menuSeed) (uint64, error) {
	var menuID uint64
	if seed.Path != "" {
		if err := tx.Table("system_menu").Select("id").Where("path = ?", seed.Path).Limit(1).Pluck("id", &menuID).Error; err != nil {
			return 0, err
		}
	} else if seed.Perms != "" {
		if err := tx.Table("system_menu").Select("id").Where("perms = ?", seed.Perms).Limit(1).Pluck("id", &menuID).Error; err != nil {
			return 0, err
		}
	}

	parentID := uint64(0)
	if seed.ParentKey != "" {
		parentID = keyToID[seed.ParentKey]
	}
	if parentID == 0 && seed.ParentPath != "" {
		if err := tx.Table("system_menu").Select("id").Where("path = ?", seed.ParentPath).Limit(1).Pluck("id", &parentID).Error; err != nil {
			return 0, err
		}
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
		if err := tx.Table("system_menu").Create(payload).Error; err != nil {
			return 0, err
		}
		if seed.Path != "" {
			if err := tx.Table("system_menu").Select("id").Where("path = ?", seed.Path).Limit(1).Pluck("id", &menuID).Error; err != nil {
				return 0, err
			}
		} else if seed.Perms != "" {
			if err := tx.Table("system_menu").Select("id").Where("perms = ?", seed.Perms).Limit(1).Pluck("id", &menuID).Error; err != nil {
				return 0, err
			}
		}
	} else if err := tx.Table("system_menu").Where("id = ?", menuID).Updates(payload).Error; err != nil {
		return 0, err
	}

	if seed.Key != "" {
		keyToID[seed.Key] = menuID
	}
	if err := bindAdmin(tx, menuID, seed); err != nil {
		return 0, err
	}
	return menuID, nil
}

func bindAdmin(tx *gorm.DB, menuID uint64, seed menuSeed) error {
	if menuID == 0 || !tx.Migrator().HasTable("system_role") {
		return nil
	}
	var adminRoleID uint64
	if err := tx.Table("system_role").Select("id").Where("role_key = ?", "admin").Limit(1).Pluck("id", &adminRoleID).Error; err != nil {
		return err
	}
	if adminRoleID == 0 {
		return nil
	}

	if seed.Type == "C" && tx.Migrator().HasTable("system_role_menu") {
		var count int64
		if err := tx.Table("system_role_menu").Where("role_id = ? AND menu_id = ?", adminRoleID, menuID).Count(&count).Error; err != nil {
			return err
		}
		if count == 0 {
			if err := tx.Exec("INSERT INTO system_role_menu (role_id, menu_id) VALUES (?, ?)", adminRoleID, menuID).Error; err != nil {
				return err
			}
		}
	}

	if tx.Migrator().HasTable("system_role_permission") {
		for _, permissionKey := range []string{strings.TrimSpace(seed.PagePerm), strings.TrimSpace(seed.Perms)} {
			if permissionKey == "" {
				continue
			}
			var count int64
			if err := tx.Table("system_role_permission").Where("role_id = ? AND permission_key = ?", adminRoleID, permissionKey).Count(&count).Error; err != nil {
				return err
			}
			if count == 0 {
				if err := tx.Exec("INSERT INTO system_role_permission (role_id, permission_key) VALUES (?, ?)", adminRoleID, permissionKey).Error; err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func seedI18n(db *gorm.DB) error {
	if db == nil || !db.Migrator().HasTable("system_i18n") {
		return nil
	}
	for _, seed := range i18nSeeds {
		var count int64
		if err := db.Table("system_i18n").Where("module = ? AND locale = ? AND `key` = ?", seed.Module, seed.Locale, seed.Key).Count(&count).Error; err != nil {
			return err
		}
		payload := map[string]interface{}{
			"module":           seed.Module,
			"group_name":       seed.Group,
			"key":              seed.Key,
			"locale":           seed.Locale,
			"value":            seed.Value,
			"lifecycle_status": "active",
			"updated_at":       time.Now(),
		}
		if count == 0 {
			payload["created_at"] = time.Now()
			if err := db.Table("system_i18n").Create(payload).Error; err != nil {
				return err
			}
			continue
		}
		if err := db.Table("system_i18n").Where("module = ? AND locale = ? AND `key` = ?", seed.Module, seed.Locale, seed.Key).Updates(payload).Error; err != nil {
			return err
		}
	}
	return nil
}
