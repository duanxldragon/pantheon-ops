//nolint:revive // Dynamic module lifecycle methods are part of the module facade.
package dynamicmodule

import (
	"errors"
	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
	"log/slog"
	"strings"
	"time"

	"github.com/duanxldragon/pantheon-base/backend/internal/scaffold"
	systemi18n "github.com/duanxldragon/pantheon-base/backend/modules/system/i18n"

	"gorm.io/gorm"
)

// UnregisterModule 卸载模块
// 1. 删除菜单/权限
// 2. 可选删除数据表
// 3. 从注册表标记为卸载
func (s *DynamicModuleService) UnregisterModule(moduleName string, dropTable bool, purgeSource bool) (*ModuleI18nLifecycleSummary, error) {
	if s.db == nil {
		return buildModuleI18nLifecycleSummary(moduleName, purgeSource, nil), nil
	}

	registration, err := s.loadModuleRegistration(moduleName)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(registration.ModelTableName) == "" {
		return nil, common.NewForbidden(msgModuleBuiltinForbidden)
	}

	// 菜单/授权清理与状态更新收进同一事务：中途失败整体回滚，
	// 不再留下"菜单已删但注册状态未变"的半成品状态。
	if err := s.markModuleUninstalled(moduleName); err != nil {
		return nil, err
	}

	// MySQL DDL 自带隐式提交，DROP TABLE 无法参与上面的事务，放到提交后执行。
	// 失败时模块已是 uninstalled 状态且注册记录仍在，PurgeModule 会重试删表。
	if s.shouldDropManagedTable(registration, dropTable) {
		if err := s.dropManagedModuleTable(registration.Scope, registration.ModelTableName); err != nil {
			// 日志不带 module/table（请求输入，CodeQL go/log-injection 污点源；
			// 与 rate_limit 中间件同一先例），上下文由调用方错误响应与审计日志承载。
			slog.Warn("dynamic module table drop failed; retry via purge", "registration_id", registration.ID, "error", err)
			return nil, err
		}
	}

	return s.FinalizeUnregister(moduleName, purgeSource)
}

func (s *DynamicModuleService) loadModuleRegistration(moduleName string) (ModuleRegistration, error) {
	var registration ModuleRegistration
	if err := s.db.Where(condNameEquals, moduleName).First(&registration).Error; err != nil {
		// 无注册记录的模块一律拒绝卸载：防止对任意磁盘目录（尤其 system/*）执行清理/删除。
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ModuleRegistration{}, common.NewNotFound(msgModuleNotFound)
		}
		return ModuleRegistration{}, err
	}
	return registration, nil
}

func (s *DynamicModuleService) markModuleUninstalled(moduleName string) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := s.deleteModuleNavigationArtifacts(tx, moduleName); err != nil {
			return err
		}
		return tx.Table("system_module_registration").
			Where(condNameEquals, moduleName).
			Updates(map[string]interface{}{
				"status":         ModuleStatusUninstalled,
				"uninstalled_at": time.Now().Format(time.RFC3339),
			}).Error
	})
}

func (s *DynamicModuleService) DeleteModuleRecord(moduleName string) error {
	if s.db == nil {
		return nil
	}

	var registration ModuleRegistration
	if err := s.db.Where(condNameEquals, moduleName).First(&registration).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return common.NewNotFound(msgModuleNotFound)
		}
		return err
	}
	if strings.TrimSpace(registration.ModelTableName) == "" {
		return common.NewForbidden(msgModuleBuiltinForbidden)
	}
	if registration.Status != ModuleStatusUninstalled {
		return common.NewBadRequest("module.delete_record.requires_uninstalled")
	}

	if err := s.db.Delete(&registration).Error; err != nil {
		return err
	}

	if strings.TrimSpace(s.workspaceRoot) == "" {
		return nil
	}
	_, err := s.refreshGeneratedWorkspaceArtifactsIfAvailable()
	return err
}

func (s *DynamicModuleService) PurgeModule(moduleName string, dropTable bool, purgeSource bool) (*ModuleI18nLifecycleSummary, error) {
	if s.db == nil {
		return buildModuleI18nLifecycleSummary(moduleName, purgeSource, nil), nil
	}

	registration, err := s.loadModuleRegistration(moduleName)
	if err != nil {
		return nil, err
	}
	if isBuiltInModuleRegistration(registration) {
		return nil, common.NewForbidden(msgModuleBuiltinForbidden)
	}
	if strings.TrimSpace(registration.ModelTableName) == "" {
		if err := s.deleteRegistrationWithoutManagedTable(moduleName, &registration); err != nil {
			return nil, err
		}
		return s.FinalizeUnregister(moduleName, purgeSource)
	}

	if err := s.prepareManagedRegistrationForPurge(moduleName, registration, dropTable); err != nil {
		return nil, err
	}

	if err := s.db.Delete(&registration).Error; err != nil {
		return nil, err
	}
	return s.FinalizeUnregister(moduleName, purgeSource)
}

func (s *DynamicModuleService) deleteRegistrationWithoutManagedTable(moduleName string, registration *ModuleRegistration) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := s.deleteModuleNavigationArtifacts(tx, moduleName); err != nil {
			return err
		}
		return tx.Delete(registration).Error
	})
}

func (s *DynamicModuleService) prepareManagedRegistrationForPurge(moduleName string, registration ModuleRegistration, dropTable bool) error {
	if registration.Status != ModuleStatusUninstalled {
		_, err := s.UnregisterModule(moduleName, dropTable, false)
		return err
	}
	if !s.shouldDropManagedTable(registration, dropTable) {
		return nil
	}
	// 删表失败必须在删除注册记录之前返回：记录保留则 purge 可重试。
	if err := s.dropManagedModuleTable(registration.Scope, registration.ModelTableName); err != nil {
		// 同上：不打请求输入字段，避免 go/log-injection。
		slog.Warn("dynamic module table drop failed; registration kept for retry", "registration_id", registration.ID, "error", err)
		return err
	}
	return nil
}

func (s *DynamicModuleService) deleteModuleNavigationArtifacts(db *gorm.DB, moduleName string) error {
	if db == nil {
		return nil
	}
	scope, shortName, err := splitModuleKey(moduleName)
	if err != nil {
		return err
	}
	if db.Migrator().HasTable("system_menu") {
		if err := s.deleteModuleMenusWithRoleBindings(db, moduleName); err != nil {
			return err
		}
	}
	if db.Migrator().HasTable("system_role_permission") {
		if err := db.Table("system_role_permission").
			Where("permission_key LIKE ?", modulePermissionPrefix(scope, shortName)+":%").
			Delete(nil).Error; err != nil {
			return err
		}
	}
	return nil
}

// deleteModuleMenusWithRoleBindings 删除模块菜单，并同步清理 system_role_menu 的
// 角色-菜单关联，避免卸载后留下悬挂的 menu_id 孤儿行。
func (s *DynamicModuleService) deleteModuleMenusWithRoleBindings(db *gorm.DB, moduleName string) error {
	var menuIDs []uint64
	if err := db.Table("system_menu").
		Where(condModuleEquals, moduleName).
		Pluck("id", &menuIDs).Error; err != nil {
		return err
	}
	if len(menuIDs) == 0 {
		return nil
	}
	if db.Migrator().HasTable("system_role_menu") {
		if err := db.Table("system_role_menu").
			Where("menu_id IN ?", menuIDs).
			Delete(nil).Error; err != nil {
			return err
		}
	}
	return db.Table("system_menu").
		Where(condModuleEquals, moduleName).
		Delete(nil).Error
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

func (s *DynamicModuleService) dropManagedModuleTable(scope, tableName string) error {
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
		Where(condNameEquals, moduleName).
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

func (s *DynamicModuleService) FinalizeUnregister(moduleName string, purgeSource bool) (*ModuleI18nLifecycleSummary, error) {
	scope, shortName, err := splitModuleKey(moduleName)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(s.workspaceRoot) == "" {
		return nil, common.NewNotFound("workspace.not_found")
	}
	if !purgeSource {
		if _, err := s.refreshGeneratedWorkspaceArtifactsIfAvailable(); err != nil {
			return nil, err
		}
		return buildModuleI18nLifecycleSummary(moduleName, false, nil), nil
	}
	// 纵深防御：源码删除仅允许 business/*，system/* 源码目录不属于生成物管理范围。
	if scope != "business" {
		return nil, common.NewForbidden(msgModuleBuiltinForbidden)
	}
	if err := scaffold.RemoveGeneratedModuleSource(s.workspaceRoot, scope, shortName); err != nil {
		return nil, err
	}
	if _, err := s.refreshGeneratedWorkspaceArtifactsIfAvailable(); err != nil {
		return nil, err
	}
	return s.advanceModuleI18nLifecycle(moduleName)
}

func (s *DynamicModuleService) advanceModuleI18nLifecycle(moduleName string) (*ModuleI18nLifecycleSummary, error) {
	if s.db == nil || !s.db.Migrator().HasTable("system_i18n") {
		return buildModuleI18nLifecycleSummary(moduleName, true, nil), nil
	}

	var directCount int64
	if err := s.db.Table("system_i18n").Where(condModuleEquals, moduleName).Count(&directCount).Error; err != nil {
		return nil, err
	}
	var prefixedConfigCount int64
	if err := s.db.Table("system_i18n").
		Where("module = ? AND (`key` = ? OR `key` LIKE ?)", "system.config", moduleName, moduleName+".%").
		Count(&prefixedConfigCount).Error; err != nil {
		return nil, err
	}
	if directCount == 0 && prefixedConfigCount == 0 {
		return buildModuleI18nLifecycleSummary(moduleName, true, nil), nil
	}

	i18nService := systemi18n.NewI18nService(s.db)
	resp := &systemi18n.I18nUnusedLifecycleAdvanceResp{
		Module:                         moduleName,
		ObservedKeys:                   make([]string, 0),
		ArchivedKeys:                   make([]string, 0),
		DeletedKeys:                    make([]string, 0),
		ArchivedRetentionThresholdDays: systemi18n.I18nArchivedRetentionThresholdDays,
	}
	if directCount > 0 {
		moduleResp, err := i18nService.AdvanceUnusedLifecycle(moduleName)
		if err != nil {
			return nil, err
		}
		mergeI18nLifecycleAdvanceResp(resp, moduleResp)
	}
	if prefixedConfigCount > 0 {
		prefixResp, err := i18nService.StartUnusedObservationByKeyPrefixes("system.config", []string{moduleName})
		if err != nil {
			return nil, err
		}
		if prefixResp != nil {
			resp.ObservedKeys = append(resp.ObservedKeys, prefixResp.AffectedKeys...)
			resp.ObservedRows += prefixResp.AffectedRows
		}
	}
	resp.ObservationOnly = resp.ObservedRows > 0 && resp.ArchivedRows == 0 && resp.DeletedRows == 0
	return buildModuleI18nLifecycleSummary(moduleName, true, resp), nil
}

func mergeI18nLifecycleAdvanceResp(target, source *systemi18n.I18nUnusedLifecycleAdvanceResp) {
	if target == nil || source == nil {
		return
	}
	target.ObservedKeys = append(target.ObservedKeys, source.ObservedKeys...)
	target.ObservedRows += source.ObservedRows
	target.ArchivedKeys = append(target.ArchivedKeys, source.ArchivedKeys...)
	target.ArchivedRows += source.ArchivedRows
	target.DeletedKeys = append(target.DeletedKeys, source.DeletedKeys...)
	target.DeletedRows += source.DeletedRows
	if source.ArchivedRetentionThresholdDays > 0 {
		target.ArchivedRetentionThresholdDays = source.ArchivedRetentionThresholdDays
	}
}
