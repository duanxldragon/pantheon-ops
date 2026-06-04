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

var retiredBusinessModules = []retiredModuleSpec{
	{
		ModuleNames: []string{
			"business.cmdb",
			"business.cmdb.host",
			"business.cmdbhostqa",
			"business.cmdb.group",
			"business.cmdb.label",
			"cmdb",
			"cmdb.host",
			"cmdbhostqa",
			"cmdb.group",
		},
		PermissionPrefixes: []string{
			"business:cmdb:",
			"business:cmdbhostqa:",
			"cmdb:",
		},
		MenuPaths: []string{
			"/business/cmdb",
			"/business/cmdb/host",
			"/business/cmdb/host/:id",
			"/business/cmdbhostqa",
			"/business/cmdbhostqa/:id",
			"/business/cmdb/group",
			"/business/cmdb/label",
			"/operations/cmdb",
			"/operations/cmdb/host",
			"/operations/cmdb/host/:id",
			"/operations/cmdb/group",
			"/operations/cmdb/label",
		},
		ComponentKeys: []string{
			"business/cmdb/host/CmdbHostList",
			"business/cmdb/host/CmdbHostDetail",
			"business/cmdbhostqa/CmdbhostqaList",
			"business/cmdbhostqa/CmdbhostqaDetail",
			"business/cmdb/group/CmdbGroupList",
			"business/cmdb/label/CmdbLabelSchemaList",
		},
	},
}

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

func cleanupRetiredModule(tx *gorm.DB, spec retiredModuleSpec) error {
	menuIDs, err := collectRetiredMenuIDs(tx, spec)
	if err != nil {
		return err
	}
	if len(menuIDs) > 0 {
		if tx.Migrator().HasTable("system_role_menu") {
			if err := tx.Table("system_role_menu").Where("menu_id IN ?", menuIDs).Delete(nil).Error; err != nil {
				return err
			}
		}
		if tx.Migrator().HasTable("system_menu") {
			if err := tx.Table("system_menu").Where("id IN ?", menuIDs).Delete(nil).Error; err != nil {
				return err
			}
		}
	}

	if tx.Migrator().HasTable("system_role_permission") {
		for _, prefix := range spec.PermissionPrefixes {
			if err := tx.Table("system_role_permission").Where("permission_key LIKE ?", prefix+"%").Delete(nil).Error; err != nil {
				return err
			}
		}
	}

	if tx.Migrator().HasTable("system_i18n") && len(spec.ModuleNames) > 0 {
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
	}

	if tx.Migrator().HasTable("system_module_registration") && len(spec.ModuleNames) > 0 {
		if err := tx.Table("system_module_registration").Where("name IN ?", spec.ModuleNames).Delete(nil).Error; err != nil {
			return err
		}
	}

	return nil
}

func collectRetiredMenuIDs(tx *gorm.DB, spec retiredModuleSpec) ([]uint64, error) {
	if !tx.Migrator().HasTable("system_menu") {
		return nil, nil
	}
	query := tx.Table("system_menu").Distinct("id")
	applied := false

	if len(spec.ModuleNames) > 0 {
		query = query.Where("module IN ?", spec.ModuleNames)
		applied = true
	}
	if len(spec.MenuPaths) > 0 {
		if applied {
			query = query.Or("path IN ?", spec.MenuPaths)
		} else {
			query = query.Where("path IN ?", spec.MenuPaths)
			applied = true
		}
	}
	if len(spec.ComponentKeys) > 0 {
		if applied {
			query = query.Or("component IN ?", spec.ComponentKeys)
		} else {
			query = query.Where("component IN ?", spec.ComponentKeys)
			applied = true
		}
	}
	for _, prefix := range spec.PermissionPrefixes {
		if applied {
			query = query.Or("page_perm LIKE ?", prefix+"%").Or("perms LIKE ?", prefix+"%")
		} else {
			query = query.Where("page_perm LIKE ?", prefix+"%").Or("perms LIKE ?", prefix+"%")
			applied = true
		}
	}
	if !applied {
		return nil, nil
	}

	var menuIDs []uint64
	if err := query.Pluck("id", &menuIDs).Error; err != nil {
		return nil, err
	}
	return menuIDs, nil
}
