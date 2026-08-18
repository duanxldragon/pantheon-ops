package bizscope

import (
	"strings"
	"time"

	"pantheon-base/pkg/contracts"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const (
	bizScopeModuleKey         = "business.bizscope"
	operationsBizScopeMenuKey = "operations.bizscope.menu"
	bizScopePermViewKey       = "business.bizscope.permission.view"
	bizScopePermCreateKey     = "business.bizscope.permission.create"
	bizScopePermUpdateKey     = "business.bizscope.permission.update"
	bizScopePermDeleteKey     = "business.bizscope.permission.delete"
	bizScopePermExportKey     = "business.bizscope.permission.export"
	bizScopePermImportKey     = "business.bizscope.permission.import"
	menuPathWhereClause       = "path = ?"
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
		ParentPath: "",
		TitleKey:   operationsBizScopeMenuKey,
		Path:       "/business/business-scope",
		Component:  "business/bizscope/BizScopeList",
		PagePerm:   "business:bizscope:list",
		Type:       "C",
		Icon:       "apps",
		RouteName:  "bizscope-list",
		Module:     bizScopeModuleKey,
		Sort:       2,
	},
	{
		Key:       "bizscope-view",
		ParentKey: "bizscope",
		TitleKey:  bizScopePermViewKey,
		Perms:     "business:bizscope:view",
		Type:      "F",
		Module:    bizScopeModuleKey,
		Sort:      1,
	},
	{
		Key:       "bizscope-create",
		ParentKey: "bizscope",
		TitleKey:  bizScopePermCreateKey,
		Perms:     "business:bizscope:create",
		Type:      "F",
		Module:    bizScopeModuleKey,
		Sort:      2,
	},
	{
		Key:       "bizscope-update",
		ParentKey: "bizscope",
		TitleKey:  bizScopePermUpdateKey,
		Perms:     "business:bizscope:update",
		Type:      "F",
		Module:    bizScopeModuleKey,
		Sort:      3,
	},
	{
		Key:       "bizscope-delete",
		ParentKey: "bizscope",
		TitleKey:  bizScopePermDeleteKey,
		Perms:     "business:bizscope:delete",
		Type:      "F",
		Module:    bizScopeModuleKey,
		Sort:      4,
	},
	{
		Key:       "bizscope-export",
		ParentKey: "bizscope",
		TitleKey:  bizScopePermExportKey,
		Perms:     "business:bizscope:export",
		Type:      "F",
		Module:    bizScopeModuleKey,
		Sort:      5,
	},
	{
		Key:       "bizscope-import",
		ParentKey: "bizscope",
		TitleKey:  bizScopePermImportKey,
		Perms:     "business:bizscope:import",
		Type:      "F",
		Module:    bizScopeModuleKey,
		Sort:      6,
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
	{Module: bizScopeModuleKey, Locale: "zh-CN", Group: "menu", Key: operationsBizScopeMenuKey, Value: "业务域"},
	{Module: bizScopeModuleKey, Locale: "en-US", Group: "menu", Key: operationsBizScopeMenuKey, Value: "Business Scope"},
	{Module: bizScopeModuleKey, Locale: "zh-CN", Group: "page", Key: "operations.bizscope.detail", Value: "业务域详情"},
	{Module: bizScopeModuleKey, Locale: "en-US", Group: "page", Key: "operations.bizscope.detail", Value: "Business Scope Detail"},
	{Module: bizScopeModuleKey, Locale: "zh-CN", Group: "page", Key: "business.bizscope.title", Value: "业务域"},
	{Module: bizScopeModuleKey, Locale: "en-US", Group: "page", Key: "business.bizscope.title", Value: "Business Scope"},
	{Module: bizScopeModuleKey, Locale: "zh-CN", Group: "page", Key: "business.bizscope.hero.eyebrow", Value: "运维平台 / 业务域"},
	{Module: bizScopeModuleKey, Locale: "en-US", Group: "page", Key: "business.bizscope.hero.eyebrow", Value: "Operations / Business Scope"},
	{Module: bizScopeModuleKey, Locale: "zh-CN", Group: "page", Key: "business.bizscope.hero.title", Value: "独立治理业务域，并作为主机分配与部署信任来源"},
	{Module: bizScopeModuleKey, Locale: "en-US", Group: "page", Key: "business.bizscope.hero.title", Value: "Govern business scopes as the source of host assignment and deployment trust"},
	{Module: bizScopeModuleKey, Locale: "zh-CN", Group: "page", Key: "business.bizscope.hero.total", Value: "业务域总数"},
	{Module: bizScopeModuleKey, Locale: "en-US", Group: "page", Key: "business.bizscope.hero.total", Value: "Total Scopes"},
	{Module: bizScopeModuleKey, Locale: "zh-CN", Group: "page", Key: "business.bizscope.hero.active", Value: "启用中"},
	{Module: bizScopeModuleKey, Locale: "en-US", Group: "page", Key: "business.bizscope.hero.active", Value: "Active"},
	{Module: bizScopeModuleKey, Locale: "zh-CN", Group: "page", Key: "business.bizscope.hero.prod", Value: "生产域"},
	{Module: bizScopeModuleKey, Locale: "en-US", Group: "page", Key: "business.bizscope.hero.prod", Value: "Production"},
	{Module: bizScopeModuleKey, Locale: "zh-CN", Group: "page", Key: "business.bizscope.field.code", Value: "业务域编码"},
	{Module: bizScopeModuleKey, Locale: "en-US", Group: "page", Key: "business.bizscope.field.code", Value: "Scope Code"},
	{Module: bizScopeModuleKey, Locale: "zh-CN", Group: "placeholder", Key: "business.bizscope.field.codePlaceholder", Value: "请输入业务域编码"},
	{Module: bizScopeModuleKey, Locale: "en-US", Group: "placeholder", Key: "business.bizscope.field.codePlaceholder", Value: "Enter scope code"},
	{Module: bizScopeModuleKey, Locale: "zh-CN", Group: "page", Key: "business.bizscope.field.name", Value: "业务域名称"},
	{Module: bizScopeModuleKey, Locale: "en-US", Group: "page", Key: "business.bizscope.field.name", Value: "Scope Name"},
	{Module: bizScopeModuleKey, Locale: "zh-CN", Group: "placeholder", Key: "business.bizscope.field.namePlaceholder", Value: "请输入业务域名称"},
	{Module: bizScopeModuleKey, Locale: "en-US", Group: "placeholder", Key: "business.bizscope.field.namePlaceholder", Value: "Enter scope name"},
	{Module: bizScopeModuleKey, Locale: "zh-CN", Group: "page", Key: "business.bizscope.field.owner", Value: "负责人"},
	{Module: bizScopeModuleKey, Locale: "en-US", Group: "page", Key: "business.bizscope.field.owner", Value: "Owner"},
	{Module: bizScopeModuleKey, Locale: "zh-CN", Group: "placeholder", Key: "business.bizscope.field.ownerPlaceholder", Value: "请输入负责人"},
	{Module: bizScopeModuleKey, Locale: "en-US", Group: "placeholder", Key: "business.bizscope.field.ownerPlaceholder", Value: "Enter owner"},
	{Module: bizScopeModuleKey, Locale: "zh-CN", Group: "page", Key: "business.bizscope.field.environment", Value: "环境"},
	{Module: bizScopeModuleKey, Locale: "en-US", Group: "page", Key: "business.bizscope.field.environment", Value: "Environment"},
	{Module: bizScopeModuleKey, Locale: "zh-CN", Group: "page", Key: "business.bizscope.environment.dev", Value: "开发"},
	{Module: bizScopeModuleKey, Locale: "en-US", Group: "page", Key: "business.bizscope.environment.dev", Value: "Development"},
	{Module: bizScopeModuleKey, Locale: "zh-CN", Group: "page", Key: "business.bizscope.environment.test", Value: "测试"},
	{Module: bizScopeModuleKey, Locale: "en-US", Group: "page", Key: "business.bizscope.environment.test", Value: "Testing"},
	{Module: bizScopeModuleKey, Locale: "zh-CN", Group: "page", Key: "business.bizscope.environment.prod", Value: "生产"},
	{Module: bizScopeModuleKey, Locale: "en-US", Group: "page", Key: "business.bizscope.environment.prod", Value: "Production"},
	{Module: bizScopeModuleKey, Locale: "zh-CN", Group: "page", Key: "business.bizscope.field.status", Value: "状态"},
	{Module: bizScopeModuleKey, Locale: "en-US", Group: "page", Key: "business.bizscope.field.status", Value: "Status"},
	{Module: bizScopeModuleKey, Locale: "zh-CN", Group: "page", Key: "business.bizscope.status.active", Value: "启用"},
	{Module: bizScopeModuleKey, Locale: "en-US", Group: "page", Key: "business.bizscope.status.active", Value: "Active"},
	{Module: bizScopeModuleKey, Locale: "zh-CN", Group: "page", Key: "business.bizscope.status.inactive", Value: "停用"},
	{Module: bizScopeModuleKey, Locale: "en-US", Group: "page", Key: "business.bizscope.status.inactive", Value: "Inactive"},
	{Module: bizScopeModuleKey, Locale: "zh-CN", Group: "page", Key: "business.bizscope.field.remark", Value: "备注"},
	{Module: bizScopeModuleKey, Locale: "en-US", Group: "page", Key: "business.bizscope.field.remark", Value: "Remark"},
	{Module: bizScopeModuleKey, Locale: "zh-CN", Group: "placeholder", Key: "business.bizscope.field.remarkPlaceholder", Value: "请输入备注"},
	{Module: bizScopeModuleKey, Locale: "en-US", Group: "placeholder", Key: "business.bizscope.field.remarkPlaceholder", Value: "Enter remark"},
	{Module: bizScopeModuleKey, Locale: "zh-CN", Group: "page", Key: "business.bizscope.createTitle", Value: "新增业务域"},
	{Module: bizScopeModuleKey, Locale: "en-US", Group: "page", Key: "business.bizscope.createTitle", Value: "Create Business Scope"},
	{Module: bizScopeModuleKey, Locale: "zh-CN", Group: "page", Key: "business.bizscope.editTitle", Value: "编辑业务域"},
	{Module: bizScopeModuleKey, Locale: "en-US", Group: "page", Key: "business.bizscope.editTitle", Value: "Edit Business Scope"},
	{Module: bizScopeModuleKey, Locale: "zh-CN", Group: "page", Key: "business.bizscope.deleteConfirm", Value: "确认删除该业务域？已绑定主机时不能删除。"},
	{Module: bizScopeModuleKey, Locale: "en-US", Group: "page", Key: "business.bizscope.deleteConfirm", Value: "Delete this business scope? Scopes bound to hosts cannot be deleted."},
	{Module: bizScopeModuleKey, Locale: "zh-CN", Group: "page", Key: "business.bizscope.empty", Value: "暂无业务域"},
	{Module: bizScopeModuleKey, Locale: "en-US", Group: "page", Key: "business.bizscope.empty", Value: "No business scopes"},
	{Module: bizScopeModuleKey, Locale: "zh-CN", Group: "permission", Key: bizScopePermViewKey, Value: "查看业务域"},
	{Module: bizScopeModuleKey, Locale: "en-US", Group: "permission", Key: bizScopePermViewKey, Value: "View business scope"},
	{Module: bizScopeModuleKey, Locale: "zh-CN", Group: "permission", Key: bizScopePermCreateKey, Value: "新增业务域"},
	{Module: bizScopeModuleKey, Locale: "en-US", Group: "permission", Key: bizScopePermCreateKey, Value: "Create business scope"},
	{Module: bizScopeModuleKey, Locale: "zh-CN", Group: "permission", Key: bizScopePermUpdateKey, Value: "编辑业务域"},
	{Module: bizScopeModuleKey, Locale: "en-US", Group: "permission", Key: bizScopePermUpdateKey, Value: "Update business scope"},
	{Module: bizScopeModuleKey, Locale: "zh-CN", Group: "permission", Key: bizScopePermDeleteKey, Value: "删除业务域"},
	{Module: bizScopeModuleKey, Locale: "en-US", Group: "permission", Key: bizScopePermDeleteKey, Value: "Delete business scope"},
	{Module: bizScopeModuleKey, Locale: "zh-CN", Group: "permission", Key: bizScopePermExportKey, Value: "导出业务域"},
	{Module: bizScopeModuleKey, Locale: "en-US", Group: "permission", Key: bizScopePermExportKey, Value: "Export business scope"},
	{Module: bizScopeModuleKey, Locale: "zh-CN", Group: "permission", Key: bizScopePermImportKey, Value: "导入业务域"},
	{Module: bizScopeModuleKey, Locale: "en-US", Group: "permission", Key: bizScopePermImportKey, Value: "Import business scope"},
	{Module: bizScopeModuleKey, Locale: "zh-CN", Group: "audit", Key: "business.bizscope.audit.create", Value: "新增业务域"},
	{Module: bizScopeModuleKey, Locale: "en-US", Group: "audit", Key: "business.bizscope.audit.create", Value: "Create business scope"},
	{Module: bizScopeModuleKey, Locale: "zh-CN", Group: "audit", Key: "business.bizscope.audit.update", Value: "编辑业务域"},
	{Module: bizScopeModuleKey, Locale: "en-US", Group: "audit", Key: "business.bizscope.audit.update", Value: "Update business scope"},
	{Module: bizScopeModuleKey, Locale: "zh-CN", Group: "audit", Key: "business.bizscope.audit.delete", Value: "删除业务域"},
	{Module: bizScopeModuleKey, Locale: "en-US", Group: "audit", Key: "business.bizscope.audit.delete", Value: "Delete business scope"},
	{Module: bizScopeModuleKey, Locale: "zh-CN", Group: "error", Key: bizScopeCodeExistsKey, Value: "业务域编码已存在"},
	{Module: bizScopeModuleKey, Locale: "en-US", Group: "error", Key: bizScopeCodeExistsKey, Value: "Business scope code already exists"},
	{Module: bizScopeModuleKey, Locale: "zh-CN", Group: "error", Key: bizScopeInUseKey, Value: "业务域已绑定主机，不能删除"},
	{Module: bizScopeModuleKey, Locale: "en-US", Group: "error", Key: bizScopeInUseKey, Value: "The business scope is bound to hosts and cannot be deleted"},
	{Module: bizScopeModuleKey, Locale: "zh-CN", Group: "error", Key: bizScopeNotFoundKey, Value: "业务域不存在"},
	{Module: bizScopeModuleKey, Locale: "en-US", Group: "error", Key: bizScopeNotFoundKey, Value: "Business scope does not exist"},
	{Module: bizScopeModuleKey, Locale: "zh-CN", Group: "error", Key: "bizscope.export_failed", Value: "导出业务域失败"},
	{Module: bizScopeModuleKey, Locale: "en-US", Group: "error", Key: "bizscope.export_failed", Value: "Failed to export business scope"},
	{Module: bizScopeModuleKey, Locale: "zh-CN", Group: "error", Key: "bizscope.import_failed", Value: "导入业务域失败"},
	{Module: bizScopeModuleKey, Locale: "en-US", Group: "error", Key: "bizscope.import_failed", Value: "Failed to import business scope"},
	{Module: bizScopeModuleKey, Locale: "zh-CN", Group: "error", Key: "business.bizscope.code_required", Value: "业务域编码不能为空"},
	{Module: bizScopeModuleKey, Locale: "en-US", Group: "error", Key: "business.bizscope.code_required", Value: "Scope code is required"},
	{Module: bizScopeModuleKey, Locale: "zh-CN", Group: "error", Key: "business.bizscope.name_required", Value: "业务域名称不能为空"},
	{Module: bizScopeModuleKey, Locale: "en-US", Group: "error", Key: "business.bizscope.name_required", Value: "Scope name is required"},
	{Module: bizScopeModuleKey, Locale: "zh-CN", Group: "error", Key: "business.bizscope.environment_required", Value: "环境类型不能为空"},
	{Module: bizScopeModuleKey, Locale: "en-US", Group: "error", Key: "business.bizscope.environment_required", Value: "Environment is required"},
	{Module: bizScopeModuleKey, Locale: "zh-CN", Group: "error", Key: "business.bizscope.environment_invalid", Value: "环境类型必须是 dev/test/prod 之一"},
	{Module: bizScopeModuleKey, Locale: "en-US", Group: "error", Key: "business.bizscope.environment_invalid", Value: "Environment must be one of dev/test/prod"},
	{Module: bizScopeModuleKey, Locale: "zh-CN", Group: "error", Key: "business.bizscope.status_invalid", Value: "状态必须是 active/inactive 之一"},
	{Module: bizScopeModuleKey, Locale: "en-US", Group: "error", Key: "business.bizscope.status_invalid", Value: "Status must be one of active/inactive"},
}

var legacyBizScopeI18nKeys = []string{
	"bizscope.code_exists",
	"bizscope.in_use",
	"bizscope.not_found",
}

func InitBizScopeModule(r *gin.RouterGroup, db *gorm.DB, dependencies ...ServiceDependencies) *Service {
	service := NewService(db, dependencies...)
	handler := NewHandler(service)

	contracts.RegisterBackendModules(r, db, contracts.FuncModule{
		ModuleName:    bizScopeModuleKey,
		MigrateFunc:   func(_ *gorm.DB) error { return service.Migrate() },
		SeedMenusFunc: seedMenus,
		SeedI18nFunc:  seedI18n,
		Register: func(r *gin.RouterGroup) {
			protected := contracts.DataScopedGroup(r, "/business/bizscope", db)
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
				protected.GET("/export", handler.Export)
				protected.GET("/import-template", handler.DownloadImportTemplate)
				protected.POST("/import", handler.Import)
			}
		},
	})
	return service
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
	menuID, err := findMenuSeedID(tx, seed)
	if err != nil {
		return 0, err
	}

	parentID, err := resolveMenuSeedParentID(tx, keyToID, seed)
	if err != nil {
		return 0, err
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
		if menuID, err = findMenuSeedID(tx, seed); err != nil {
			return 0, err
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

// findMenuSeedID returns the existing menu id for a seed, matching by path or
// perms, or 0 when no matching row exists.
func findMenuSeedID(tx *gorm.DB, seed menuSeed) (uint64, error) {
	var menuID uint64
	if seed.Path != "" {
		if err := tx.Table("system_menu").Select("id").Where(menuPathWhereClause, seed.Path).Limit(1).Pluck("id", &menuID).Error; err != nil {
			return 0, err
		}
	} else if seed.Perms != "" {
		if err := tx.Table("system_menu").Select("id").Where("perms = ?", seed.Perms).Limit(1).Pluck("id", &menuID).Error; err != nil {
			return 0, err
		}
	}
	return menuID, nil
}

// resolveMenuSeedParentID resolves a seed's parent menu id from the already
// inserted keys or the parent path.
func resolveMenuSeedParentID(tx *gorm.DB, keyToID map[string]uint64, seed menuSeed) (uint64, error) {
	parentID := uint64(0)
	if seed.ParentKey != "" {
		parentID = keyToID[seed.ParentKey]
	}
	if parentID == 0 && seed.ParentPath != "" {
		if err := tx.Table("system_menu").Select("id").Where(menuPathWhereClause, seed.ParentPath).Limit(1).Pluck("id", &parentID).Error; err != nil {
			return 0, err
		}
	}
	return parentID, nil
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
	if err := bindAdminMenu(tx, adminRoleID, menuID, seed); err != nil {
		return err
	}
	return bindAdminPermission(tx, adminRoleID, seed)
}

func bindAdminMenu(tx *gorm.DB, adminRoleID, menuID uint64, seed menuSeed) error {
	if seed.Type != "C" || !tx.Migrator().HasTable("system_role_menu") {
		return nil
	}
	var count int64
	if err := tx.Table("system_role_menu").Where("role_id = ? AND menu_id = ?", adminRoleID, menuID).Count(&count).Error; err != nil {
		return err
	}
	if count != 0 {
		return nil
	}
	return tx.Exec("INSERT INTO system_role_menu (role_id, menu_id) VALUES (?, ?)", adminRoleID, menuID).Error
}

func bindAdminPermission(tx *gorm.DB, adminRoleID uint64, seed menuSeed) error {
	if !tx.Migrator().HasTable("system_role_permission") {
		return nil
	}
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
	return nil
}

func seedI18n(db *gorm.DB) error {
	if db == nil || !db.Migrator().HasTable("system_i18n") {
		return nil
	}
	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(
			"DELETE FROM system_i18n WHERE module = ? AND `key` IN ?",
			bizScopeModuleKey,
			legacyBizScopeI18nKeys,
		).Error; err != nil {
			return err
		}

		for _, seed := range i18nSeeds {
			var count int64
			if err := tx.Table("system_i18n").Where("module = ? AND locale = ? AND `key` = ?", seed.Module, seed.Locale, seed.Key).Count(&count).Error; err != nil {
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
				if err := tx.Table("system_i18n").Create(payload).Error; err != nil {
					return err
				}
				continue
			}
			if err := tx.Table("system_i18n").Where("module = ? AND locale = ? AND `key` = ?", seed.Module, seed.Locale, seed.Key).Updates(payload).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
