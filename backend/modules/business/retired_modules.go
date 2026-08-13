package business

import (
	"strings"

	"gorm.io/gorm"
)

type retiredModuleSpec struct {
	ModuleNames        []string
	PermissionPrefixes []string
	MenuPaths          []string
	ComponentKeys      []string
}

// retiredBusinessModules 在 pantheon-base 留作通用 hook，business/* 退役条目
// 属于 ops 业务语义，应由 pantheon-ops 维护。base 不再内置 cmdb 等业务模块。
var retiredBusinessModules = []retiredModuleSpec{}

func CleanupRetiredBusinessModules(db *gorm.DB) error {
	return cleanupRetiredBusinessModules(db)
}

func cleanupRetiredBusinessModules(db *gorm.DB) error {
	if db == nil {
		return nil
	}
	return db.Transaction(func(tx *gorm.DB) error {
		for _, spec := range retiredBusinessModules {
			if err := cleanupRetiredModule(tx, spec); err != nil {
				return err
			}
		}
		return nil
	})
}

// cleanupRetiredModule 清理单个退役模块：先归集待删菜单 ID，再按固定顺序删除
// 角色-菜单关联、菜单、角色权限、i18n 文案与模块注册信息。删除顺序与判定逻辑
// 与原实现完全一致，以保证引用完整性与幂等性。
func cleanupRetiredModule(tx *gorm.DB, spec retiredModuleSpec) error {
	menuIDs, err := collectRetiredMenuIDs(tx, spec)
	if err != nil {
		return err
	}
	if err := deleteRetiredRoleMenuLinks(tx, menuIDs); err != nil {
		return err
	}
	if err := deleteRetiredMenus(tx, menuIDs); err != nil {
		return err
	}
	if err := deleteRetiredRolePermissions(tx, spec); err != nil {
		return err
	}
	if err := deleteRetiredI18n(tx, spec); err != nil {
		return err
	}
	if err := deleteRetiredModuleRegistrations(tx, spec); err != nil {
		return err
	}
	return nil
}

// deleteRetiredRoleMenuLinks 删除引用待退役菜单的角色-菜单关联行。
func deleteRetiredRoleMenuLinks(tx *gorm.DB, menuIDs []uint64) error {
	if len(menuIDs) == 0 {
		return nil
	}
	if !tx.Migrator().HasTable("system_role_menu") {
		return nil
	}
	if err := tx.Table("system_role_menu").Where("menu_id IN ?", menuIDs).Delete(nil).Error; err != nil {
		return err
	}
	return nil
}

// deleteRetiredMenus 删除待退役菜单本身。
func deleteRetiredMenus(tx *gorm.DB, menuIDs []uint64) error {
	if len(menuIDs) == 0 {
		return nil
	}
	if !tx.Migrator().HasTable("system_menu") {
		return nil
	}
	if err := tx.Table("system_menu").Where("id IN ?", menuIDs).Delete(nil).Error; err != nil {
		return err
	}
	return nil
}

// deleteRetiredRolePermissions 按权限前缀删除角色权限条目。
func deleteRetiredRolePermissions(tx *gorm.DB, spec retiredModuleSpec) error {
	if !tx.Migrator().HasTable("system_role_permission") {
		return nil
	}
	for _, prefix := range spec.PermissionPrefixes {
		if err := tx.Table("system_role_permission").Where("permission_key LIKE ?", prefix+"%").Delete(nil).Error; err != nil {
			return err
		}
	}
	return nil
}

// deleteRetiredI18n 删除模块级 i18n 文案及 system.config 作用域下的模块键文案。
func deleteRetiredI18n(tx *gorm.DB, spec retiredModuleSpec) error {
	if !tx.Migrator().HasTable("system_i18n") || len(spec.ModuleNames) == 0 {
		return nil
	}
	if err := tx.Table("system_i18n").Where("module IN ?", spec.ModuleNames).Delete(nil).Error; err != nil {
		return err
	}
	for _, moduleName := range spec.ModuleNames {
		trimmed := strings.TrimSpace(moduleName)
		if trimmed == "" {
			continue
		}
		if err := tx.Table("system_i18n").
			Where("module = ? AND (`key` = ? OR `key` LIKE ?)", "system.config", trimmed, trimmed+".%").
			Delete(nil).Error; err != nil {
			return err
		}
	}
	return nil
}

// deleteRetiredModuleRegistrations 删除模块注册信息。
func deleteRetiredModuleRegistrations(tx *gorm.DB, spec retiredModuleSpec) error {
	if !tx.Migrator().HasTable("system_module_registration") || len(spec.ModuleNames) == 0 {
		return nil
	}
	if err := tx.Table("system_module_registration").Where("name IN ?", spec.ModuleNames).Delete(nil).Error; err != nil {
		return err
	}
	return nil
}

// collectRetiredMenuIDs 归集待退役菜单 ID：依据模块名、菜单路径、组件键与权限前缀
// 构造去重查询。首个条件用 WHERE，后续条件用 OR 串联（与原实现等价）；若无任何条件
// 则返回空。查询构造逻辑与原实现逐行一致。
func collectRetiredMenuIDs(tx *gorm.DB, spec retiredModuleSpec) ([]uint64, error) {
	if !tx.Migrator().HasTable("system_menu") {
		return nil, nil
	}
	query := tx.Table("system_menu").Distinct("id")
	applied := false

	query, applied = applyMenuFieldFilter(query, applied, "module", spec.ModuleNames)
	query, applied = applyMenuFieldFilter(query, applied, "path", spec.MenuPaths)
	query, applied = applyMenuFieldFilter(query, applied, "component", spec.ComponentKeys)
	query, applied = appendPermissionPrefixFilters(query, applied, spec.PermissionPrefixes)

	if !applied {
		return nil, nil
	}

	var menuIDs []uint64
	if err := query.Pluck("id", &menuIDs).Error; err != nil {
		return nil, err
	}
	return menuIDs, nil
}

// applyMenuFieldFilter 为单个字段（模块名/路径/组件）追加 IN 过滤条件：首个条件用
// WHERE，后续条件用 OR 串联，返回更新后的 (query, applied)。等价于原 collectRetiredMenuIDs
// 中内联的 if applied/else 分支。
func applyMenuFieldFilter(query *gorm.DB, applied bool, column string, values []string) (*gorm.DB, bool) {
	if len(values) == 0 {
		return query, applied
	}
	if applied {
		return query.Or(column+" IN ?", values), true
	}
	return query.Where(column+" IN ?", values), true
}

// appendPermissionPrefixFilters 按权限前缀追加 page_perm/perms 的 OR 过滤条件，
// 并返回更新后的 (query, applied)。该逻辑为原 collectRetiredMenuIDs 循环体的等价提取。
func appendPermissionPrefixFilters(query *gorm.DB, applied bool, prefixes []string) (*gorm.DB, bool) {
	for _, prefix := range prefixes {
		if applied {
			query = query.Or("page_perm LIKE ?", prefix+"%").Or("perms LIKE ?", prefix+"%")
		} else {
			query = query.Where("page_perm LIKE ?", prefix+"%").Or("perms LIKE ?", prefix+"%")
			applied = true
		}
	}
	return query, applied
}
