package observability

import (
	"time"

	"gorm.io/gorm"
)

const (
	observabilityModuleKey = "business.observability"
	observabilityMenuPath  = "/business/observability"
	observabilityMenuTitle = "operations.observability.menu"
	monitoringMenuPath     = "/business/observability/monitoring"
	alertCenterMenuPath    = "/business/observability/alerts"
	logsMenuPath           = "/business/observability/logs"
	tracesMenuPath         = "/business/observability/traces"
)

type obsMenuSeed struct {
	Key        string
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
	ParentPath string
}

// obsMenuSeeds mirrors the frontend ObservabilityModule routes/menus exactly.
// The paths must match the frontend route declarations (singular segments);
// the stale seed rows previously used plural paths (/metric-sources, /alert-rules,
// /alert-records) which produced client-side 404s on menu click.
var obsMenuSeeds = []obsMenuSeed{
	{Key: "observability", TitleKey: observabilityMenuTitle, Path: observabilityMenuPath, Type: "M", Icon: "radar", RouteName: "observability", Module: observabilityModuleKey, Sort: 5},
	{Key: "observability-monitoring", ParentPath: observabilityMenuPath, TitleKey: "operations.observability.monitoring.menu", Path: monitoringMenuPath, Component: "business/observability/MonitoringCenter", PagePerm: "business:observability:metric_source:view", Type: "C", Icon: "storage", RouteName: "observability-monitoring", Module: observabilityModuleKey, Sort: 1},
	{Key: "observability-alerts", ParentPath: observabilityMenuPath, TitleKey: "operations.observability.alertCenter.menu", Path: alertCenterMenuPath, Component: "business/observability/AlertCenter", PagePerm: "business:observability:alert_rule:view", Type: "C", Icon: "book", RouteName: "observability-alert-center", Module: observabilityModuleKey, Sort: 2},
	{Key: "observability-logs", ParentPath: observabilityMenuPath, TitleKey: "operations.observability.logs.menu", Path: logsMenuPath, Component: "business/observability/LogExplorer", PagePerm: "business:observability:logs:view", Type: "C", Icon: "tags", RouteName: "observability-logs", Module: observabilityModuleKey, Sort: 3},
	{Key: "observability-traces", ParentPath: observabilityMenuPath, TitleKey: "operations.observability.traces.menu", Path: tracesMenuPath, Component: "business/observability/TraceExplorer", PagePerm: "business:observability:traces:view", Type: "C", Icon: "branch", RouteName: "observability-traces", Module: observabilityModuleKey, Sort: 4},
	{Key: "observability-monitoring-create", ParentPath: monitoringMenuPath, TitleKey: "business.observability.permission.metricSourceCreate", Perms: "business:observability:metric_source:create", Type: "F", Module: observabilityModuleKey, Sort: 1},
	{Key: "observability-monitoring-update", ParentPath: monitoringMenuPath, TitleKey: "business.observability.permission.metricSourceUpdate", Perms: "business:observability:metric_source:update", Type: "F", Module: observabilityModuleKey, Sort: 2},
	{Key: "observability-monitoring-delete", ParentPath: monitoringMenuPath, TitleKey: "business.observability.permission.metricSourceDelete", Perms: "business:observability:metric_source:delete", Type: "F", Module: observabilityModuleKey, Sort: 3},
	{Key: "observability-alert-create", ParentPath: alertCenterMenuPath, TitleKey: "business.observability.permission.alertRuleCreate", Perms: "business:observability:alert_rule:create", Type: "F", Module: observabilityModuleKey, Sort: 1},
	{Key: "observability-alert-update", ParentPath: alertCenterMenuPath, TitleKey: "business.observability.permission.alertRuleUpdate", Perms: "business:observability:alert_rule:update", Type: "F", Module: observabilityModuleKey, Sort: 2},
	{Key: "observability-alert-delete", ParentPath: alertCenterMenuPath, TitleKey: "business.observability.permission.alertRuleDelete", Perms: "business:observability:alert_rule:delete", Type: "F", Module: observabilityModuleKey, Sort: 3},
	{Key: "observability-notification-update", ParentPath: alertCenterMenuPath, TitleKey: "business.observability.permission.notificationUpdate", Perms: "business:observability:notification:update", Type: "F", Module: observabilityModuleKey, Sort: 4},
}

// SeedMenus aligns the backend menu tree with the frontend ObservabilityModule
// routes and grants admin the list permission. It also removes the stale
// plural-path menu rows left by the earlier shell implementation.
func SeedMenus(db *gorm.DB) error {
	if db == nil {
		return nil
	}
	if db.Migrator().HasTable("system_menu") {
		if err := db.Transaction(func(tx *gorm.DB) error {
			// Remove stale observability menus whose paths no longer match any
			// frontend route (they render a client-side 404 on click).
			for _, stale := range []string{
				observabilityMenuPath + "/metric-sources",
				observabilityMenuPath + "/alert-rules",
				observabilityMenuPath + "/alert-records",
				observabilityMenuPath + "/metric-source",
				observabilityMenuPath + "/alert-rule",
				observabilityMenuPath + "/alert-record",
			} {
				if err := tx.Table("system_menu").Where("path = ?", stale).Delete(nil).Error; err != nil {
					return err
				}
			}

			keyToID := make(map[string]uint64, len(obsMenuSeeds))
			for _, seed := range obsMenuSeeds {
				var menuID uint64
				lookup := tx.Table("system_menu").Select("id")
				if seed.Path != "" {
					lookup = lookup.Where("path = ?", seed.Path)
				} else {
					lookup = lookup.Where("perms = ?", seed.Perms)
				}
				if err := lookup.Limit(1).Pluck("id", &menuID).Error; err != nil {
					return err
				}
				parentID := uint64(0)
				if seed.ParentPath != "" {
					if err := tx.Table("system_menu").Select("id").Where("path = ?", seed.ParentPath).Limit(1).Pluck("id", &parentID).Error; err != nil {
						return err
					}
				}
				payload := map[string]any{
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
						return err
					}
					lookup = tx.Table("system_menu").Select("id")
					if seed.Path != "" {
						lookup = lookup.Where("path = ?", seed.Path)
					} else {
						lookup = lookup.Where("perms = ?", seed.Perms)
					}
					if err := lookup.Limit(1).Pluck("id", &menuID).Error; err != nil {
						return err
					}
				} else if err := tx.Table("system_menu").Where("id = ?", menuID).Updates(payload).Error; err != nil {
					return err
				}
				keyToID[seed.Key] = menuID
				if err := bindObservabilityAdmin(tx, menuID, seed); err != nil {
					return err
				}
			}
			return nil
		}); err != nil {
			return err
		}
	}
	return seedObservabilityI18n(db)
}

func seedObservabilityI18n(db *gorm.DB) error {
	if !db.Migrator().HasTable("system_i18n") {
		return nil
	}
	seeds := []struct {
		Locale string
		Group  string
		Key    string
		Value  string
	}{
		{Locale: "zh-CN", Group: "menu", Key: observabilityMenuTitle, Value: "可观测中心"},
		{Locale: "en-US", Group: "menu", Key: observabilityMenuTitle, Value: "Observability"},
		{Locale: "zh-CN", Group: "menu", Key: "operations.observability.monitoring.menu", Value: "监控中心"},
		{Locale: "en-US", Group: "menu", Key: "operations.observability.monitoring.menu", Value: "Monitoring Center"},
		{Locale: "zh-CN", Group: "menu", Key: "operations.observability.alertCenter.menu", Value: "告警中心"},
		{Locale: "en-US", Group: "menu", Key: "operations.observability.alertCenter.menu", Value: "Alert Center"},
		{Locale: "zh-CN", Group: "menu", Key: "operations.observability.logs.menu", Value: "日志中心"},
		{Locale: "en-US", Group: "menu", Key: "operations.observability.logs.menu", Value: "Log Center"},
		{Locale: "zh-CN", Group: "menu", Key: "operations.observability.traces.menu", Value: "链路追踪"},
		{Locale: "en-US", Group: "menu", Key: "operations.observability.traces.menu", Value: "Distributed Tracing"},
		{Locale: "zh-CN", Group: "permission", Key: "business.observability.permission.metricSourceCreate", Value: "创建可观测数据源"},
		{Locale: "en-US", Group: "permission", Key: "business.observability.permission.metricSourceCreate", Value: "Create observability data sources"},
		{Locale: "zh-CN", Group: "permission", Key: "business.observability.permission.metricSourceUpdate", Value: "更新可观测数据源"},
		{Locale: "en-US", Group: "permission", Key: "business.observability.permission.metricSourceUpdate", Value: "Update observability data sources"},
		{Locale: "zh-CN", Group: "permission", Key: "business.observability.permission.metricSourceDelete", Value: "删除可观测数据源"},
		{Locale: "en-US", Group: "permission", Key: "business.observability.permission.metricSourceDelete", Value: "Delete observability data sources"},
		{Locale: "zh-CN", Group: "permission", Key: "business.observability.permission.alertRuleCreate", Value: "创建告警规则"},
		{Locale: "en-US", Group: "permission", Key: "business.observability.permission.alertRuleCreate", Value: "Create alert rules"},
		{Locale: "zh-CN", Group: "permission", Key: "business.observability.permission.alertRuleUpdate", Value: "更新告警规则"},
		{Locale: "en-US", Group: "permission", Key: "business.observability.permission.alertRuleUpdate", Value: "Update alert rules"},
		{Locale: "zh-CN", Group: "permission", Key: "business.observability.permission.alertRuleDelete", Value: "删除告警规则"},
		{Locale: "en-US", Group: "permission", Key: "business.observability.permission.alertRuleDelete", Value: "Delete alert rules"},
		{Locale: "zh-CN", Group: "permission", Key: "business.observability.permission.notificationUpdate", Value: "管理通知渠道"},
		{Locale: "en-US", Group: "permission", Key: "business.observability.permission.notificationUpdate", Value: "Manage notification channels"},
	}
	return db.Transaction(func(tx *gorm.DB) error {
		for _, item := range seeds {
			payload := map[string]any{
				"module":           observabilityModuleKey,
				"locale":           item.Locale,
				"group_name":       item.Group,
				"key":              item.Key,
				"value":            item.Value,
				"lifecycle_status": "active",
				"updated_at":       time.Now(),
			}
			var count int64
			where := tx.Table("system_i18n").Where("module = ? AND locale = ? AND `key` = ?", observabilityModuleKey, item.Locale, item.Key)
			if err := where.Count(&count).Error; err != nil {
				return err
			}
			if count == 0 {
				payload["created_at"] = time.Now()
				if err := tx.Table("system_i18n").Create(payload).Error; err != nil {
					return err
				}
			} else if err := where.Updates(payload).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func bindObservabilityAdmin(tx *gorm.DB, menuID uint64, seed obsMenuSeed) error {
	if menuID == 0 || !tx.Migrator().HasTable("system_role") {
		return nil
	}
	var roleID uint64
	if err := tx.Table("system_role").Select("id").Where("role_key = ?", "admin").Limit(1).Pluck("id", &roleID).Error; err != nil {
		return err
	}
	if roleID == 0 {
		return nil
	}
	if seed.Type == "C" && tx.Migrator().HasTable("system_role_menu") {
		var count int64
		if err := tx.Table("system_role_menu").Where("role_id = ? AND menu_id = ?", roleID, menuID).Count(&count).Error; err != nil {
			return err
		}
		if count == 0 {
			if err := tx.Exec("INSERT INTO system_role_menu (role_id, menu_id) VALUES (?, ?)", roleID, menuID).Error; err != nil {
				return err
			}
		}
	}
	if !tx.Migrator().HasTable("system_role_permission") {
		return nil
	}
	for _, permissionKey := range []string{seed.PagePerm, seed.Perms} {
		if permissionKey == "" {
			continue
		}
		var count int64
		if err := tx.Table("system_role_permission").Where("role_id = ? AND permission_key = ?", roleID, permissionKey).Count(&count).Error; err != nil {
			return err
		}
		if count == 0 {
			if err := tx.Exec("INSERT INTO system_role_permission (role_id, permission_key) VALUES (?, ?)", roleID, permissionKey).Error; err != nil {
				return err
			}
		}
	}
	return nil
}
