package dynamicmodule

import (
	"errors"
	"strings"
	"time"

	"pantheon-ops/backend/internal/scaffold"

	"gorm.io/gorm"
)

// UnregisterModule 卸载模块
// 1. 删除菜单/权限
// 2. 可选删除数据表
// 3. 从注册表标记为卸载
func (s *DynamicModuleService) UnregisterModule(moduleName string, dropTable bool, purgeSource bool) error {
	if s.db == nil {
		return nil
	}

	var registration ModuleRegistration
	if err := s.db.Where("name = ?", moduleName).First(&registration).Error; err == nil {
		if strings.TrimSpace(registration.ModelTableName) == "" {
			return errors.New("module.unregister.builtin_forbidden")
		}
	}

	scope, shortName, err := splitModuleKey(moduleName)
	if err != nil {
		return err
	}

	if s.db.Migrator().HasTable("system_menu") {
		if err := s.db.Table("system_menu").
			Where("module = ?", moduleName).
			Delete(nil).Error; err != nil {
			return err
		}
	}

	if s.db.Migrator().HasTable("system_role_permission") {
		if err := s.db.Table("system_role_permission").
			Where("permission_key LIKE ?", scope+":"+shortName+":%").
			Delete(nil).Error; err != nil {
			return err
		}
	}

	if s.shouldDropManagedTable(registration, dropTable) {
		if err := s.dropManagedModuleTable(scope, registration.ModelTableName); err != nil {
			return err
		}
	}

	if err := s.db.Table("system_module_registration").
		Where("name = ?", moduleName).
		Updates(map[string]interface{}{
			"status":         ModuleStatusUninstalled,
			"uninstalled_at": time.Now().Format(time.RFC3339),
		}).Error; err != nil {
		return err
	}

	return s.FinalizeUnregister(moduleName, purgeSource)
}

func (s *DynamicModuleService) DeleteModuleRecord(moduleName string) error {
	if s.db == nil {
		return nil
	}

	var registration ModuleRegistration
	if err := s.db.Where("name = ?", moduleName).First(&registration).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("module.not_found")
		}
		return err
	}
	if strings.TrimSpace(registration.ModelTableName) == "" {
		return errors.New("module.unregister.builtin_forbidden")
	}
	if registration.Status != ModuleStatusUninstalled {
		return errors.New("module.delete_record.requires_uninstalled")
	}

	if err := s.db.Delete(&registration).Error; err != nil {
		return err
	}

	if strings.TrimSpace(s.workspaceRoot) == "" {
		return nil
	}
	refs, err := s.listGeneratedModuleRefs()
	if err != nil {
		return err
	}
	return scaffold.WriteGeneratedRegistries(s.workspaceRoot, refs)
}

func (s *DynamicModuleService) PurgeModule(moduleName string, dropTable bool, purgeSource bool) error {
	if s.db == nil {
		return nil
	}

	var registration ModuleRegistration
	if err := s.db.Where("name = ?", moduleName).First(&registration).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("module.not_found")
		}
		return err
	}
	if isBuiltInModuleRegistration(registration) {
		return errors.New("module.unregister.builtin_forbidden")
	}
	if strings.TrimSpace(registration.ModelTableName) == "" {
		if err := s.deleteModuleNavigationArtifacts(moduleName); err != nil {
			return err
		}
		if err := s.db.Delete(&registration).Error; err != nil {
			return err
		}
		return s.rewriteGeneratedRegistriesIfWorkspaceAvailable()
	}

	if registration.Status != ModuleStatusUninstalled {
		if err := s.UnregisterModule(moduleName, dropTable, false); err != nil {
			return err
		}
	} else if s.shouldDropManagedTable(registration, dropTable) {
		if err := s.dropManagedModuleTable(registration.Scope, registration.ModelTableName); err != nil {
			return err
		}
	}

	if err := s.db.Delete(&registration).Error; err != nil {
		return err
	}
	return s.FinalizeUnregister(moduleName, purgeSource)
}

func (s *DynamicModuleService) deleteModuleNavigationArtifacts(moduleName string) error {
	if s.db == nil {
		return nil
	}
	if s.db.Migrator().HasTable("system_menu") {
		if err := s.db.Table("system_menu").Where("module = ?", moduleName).Delete(nil).Error; err != nil {
			return err
		}
	}
	if s.db.Migrator().HasTable("system_i18n") {
		if err := s.db.Table("system_i18n").Where("module = ?", moduleName).Delete(nil).Error; err != nil {
			return err
		}
	}
	return nil
}

func (s *DynamicModuleService) rewriteGeneratedRegistriesIfWorkspaceAvailable() error {
	if strings.TrimSpace(s.workspaceRoot) == "" {
		return nil
	}
	refs, err := s.listGeneratedModuleRefs()
	if err != nil {
		return err
	}
	return scaffold.WriteGeneratedRegistries(s.workspaceRoot, refs)
}

// ListRegisteredModules 获取已注册模块列表
func (s *DynamicModuleService) ListRegisteredModules() ([]ModuleRegistrationResp, error) {
	if err := s.syncModuleRegistrationRecords(false); err != nil {
		return nil, err
	}
	var modules []ModuleRegistration
	if err := s.db.Table("system_module_registration").
		Order("scope ASC").
		Order("name ASC").
		Find(&modules).Error; err != nil {
		return nil, err
	}
	for index := range modules {
		modules[index].BuiltIn = isBuiltInModuleRegistration(modules[index])
	}
	resp := make([]ModuleRegistrationResp, 0, len(modules))
	for _, module := range modules {
		resp = append(resp, toModuleRegistrationResp(module))
	}
	return resp, nil
}

func (s *DynamicModuleService) dropManagedModuleTable(scope string, tableName string) error {
	if err := scaffold.ValidateManagedTableName(scope, tableName); err != nil {
		return err
	}
	return s.db.Migrator().DropTable(strings.TrimSpace(tableName))
}

func (s *DynamicModuleService) shouldDropManagedTable(registration ModuleRegistration, requested bool) bool {
	if strings.TrimSpace(registration.ModelTableName) == "" {
		return false
	}
	return requested || registration.AutoRecycle
}

// GetModuleStatus 获取模块状态
func (s *DynamicModuleService) GetModuleStatus(moduleName string) (*ModuleRegistrationResp, error) {
	if err := s.syncModuleRegistrationRecords(false); err != nil {
		return nil, err
	}
	var module ModuleRegistration
	if err := s.db.Table("system_module_registration").
		Where("name = ?", moduleName).
		First(&module).Error; err != nil {
		return nil, err
	}
	module.BuiltIn = isBuiltInModuleRegistration(module)
	resp := toModuleRegistrationResp(module)
	return &resp, nil
}

func isBuiltInModuleRegistration(module ModuleRegistration) bool {
	if strings.TrimSpace(module.ModelTableName) != "" {
		return false
	}
	return strings.TrimSpace(module.Scope) != "business"
}

func (s *DynamicModuleService) FinalizeUnregister(moduleName string, purgeSource bool) error {
	scope, shortName, err := splitModuleKey(moduleName)
	if err != nil {
		return err
	}
	if strings.TrimSpace(s.workspaceRoot) == "" {
		return errors.New("workspace.not_found")
	}
	refs, err := s.listGeneratedModuleRefs()
	if err != nil {
		return err
	}
	if err := scaffold.WriteGeneratedRegistries(s.workspaceRoot, refs); err != nil {
		return err
	}
	if purgeSource {
		if err := scaffold.RemoveGeneratedModuleSource(s.workspaceRoot, scope, shortName); err != nil {
			return err
		}
	}
	return nil
}
