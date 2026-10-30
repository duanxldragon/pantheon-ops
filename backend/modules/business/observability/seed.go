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

	// Menu title keys are shared verbatim by the menu rows below and the i18n rows in
	// seedObservabilityI18n, so each is declared once. The strings are a contract with the
	// frontend menu catalogue and must not drift.
	monitoringMenuTitle  = "operations.observability.monitoring.menu"
	alertCenterMenuTitle = "operations.observability.alertCenter.menu"
	logsMenuTitle        = "operations.observability.logs.menu"
	tracesMenuTitle      = "operations.observability.traces.menu"

	// Permission title keys of the button-level menu rows, shared with the i18n seed for the
	// same reason.
	metricSourceCreatePermTitle = "business.observability.permission.metricSourceCreate"
	metricSourceUpdatePermTitle = "business.observability.permission.metricSourceUpdate"
	metricSourceDeletePermTitle = "business.observability.permission.metricSourceDelete"
	alertRuleCreatePermTitle    = "business.observability.permission.alertRuleCreate"
	alertRuleUpdatePermTitle    = "business.observability.permission.alertRuleUpdate"
	alertRuleDeletePermTitle    = "business.observability.permission.alertRuleDelete"
	notificationUpdatePermTitle = "business.observability.permission.notificationUpdate"

	// Fixed SQL text of the menu lookups; seed data only ever enters through bind parameters.
	menuPathWhereClause  = "path = ?"
	menuPermsWhereClause = "perms = ?"
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
	{Key: "observability-monitoring", ParentPath: observabilityMenuPath, TitleKey: monitoringMenuTitle, Path: monitoringMenuPath, Component: "business/observability/MonitoringCenter", PagePerm: "business:observability:metric_source:view", Type: "C", Icon: "storage", RouteName: "observability-monitoring", Module: observabilityModuleKey, Sort: 1},
	{Key: "observability-alerts", ParentPath: observabilityMenuPath, TitleKey: alertCenterMenuTitle, Path: alertCenterMenuPath, Component: "business/observability/AlertCenter", PagePerm: "business:observability:alert_rule:view", Type: "C", Icon: "book", RouteName: "observability-alert-center", Module: observabilityModuleKey, Sort: 2},
	{Key: "observability-logs", ParentPath: observabilityMenuPath, TitleKey: logsMenuTitle, Path: logsMenuPath, Component: "business/observability/LogExplorer", PagePerm: "business:observability:logs:view", Type: "C", Icon: "tags", RouteName: "observability-logs", Module: observabilityModuleKey, Sort: 3},
	{Key: "observability-traces", ParentPath: observabilityMenuPath, TitleKey: tracesMenuTitle, Path: tracesMenuPath, Component: "business/observability/TraceExplorer", PagePerm: "business:observability:traces:view", Type: "C", Icon: "branch", RouteName: "observability-traces", Module: observabilityModuleKey, Sort: 4},
	{Key: "observability-monitoring-create", ParentPath: monitoringMenuPath, TitleKey: metricSourceCreatePermTitle, Perms: "business:observability:metric_source:create", Type: "F", Module: observabilityModuleKey, Sort: 1},
	{Key: "observability-monitoring-update", ParentPath: monitoringMenuPath, TitleKey: metricSourceUpdatePermTitle, Perms: "business:observability:metric_source:update", Type: "F", Module: observabilityModuleKey, Sort: 2},
	{Key: "observability-monitoring-delete", ParentPath: monitoringMenuPath, TitleKey: metricSourceDeletePermTitle, Perms: "business:observability:metric_source:delete", Type: "F", Module: observabilityModuleKey, Sort: 3},
	{Key: "observability-alert-create", ParentPath: alertCenterMenuPath, TitleKey: alertRuleCreatePermTitle, Perms: "business:observability:alert_rule:create", Type: "F", Module: observabilityModuleKey, Sort: 1},
	{Key: "observability-alert-update", ParentPath: alertCenterMenuPath, TitleKey: alertRuleUpdatePermTitle, Perms: "business:observability:alert_rule:update", Type: "F", Module: observabilityModuleKey, Sort: 2},
	{Key: "observability-alert-delete", ParentPath: alertCenterMenuPath, TitleKey: alertRuleDeletePermTitle, Perms: "business:observability:alert_rule:delete", Type: "F", Module: observabilityModuleKey, Sort: 3},
	{Key: "observability-notification-update", ParentPath: alertCenterMenuPath, TitleKey: notificationUpdatePermTitle, Perms: "business:observability:notification:update", Type: "F", Module: observabilityModuleKey, Sort: 4},
}

// SeedMenus aligns the backend menu tree with the frontend ObservabilityModule
// routes and grants admin the list permission. It also removes the stale
// plural-path menu rows left by the earlier shell implementation.
func SeedMenus(db *gorm.DB) error {
	if db == nil {
		return nil
	}
	if db.Migrator().HasTable("system_menu") {
		if err := db.Transaction(seedObservabilityMenus); err != nil {
			return err
		}
	}
	return seedObservabilityI18n(db)
}

// seedObservabilityMenus walks the declared menu tree inside one transaction, upserting every
// row and binding admin to it, in the order the rows are declared.
func seedObservabilityMenus(tx *gorm.DB) error {
	if err := deleteStaleObservabilityMenus(tx); err != nil {
		return err
	}
	keyToID := make(map[string]uint64, len(obsMenuSeeds))
	for _, seed := range obsMenuSeeds {
		menuID, err := ensureObservabilityMenu(tx, seed)
		if err != nil {
			return err
		}
		keyToID[seed.Key] = menuID
		if err := bindObservabilityAdmin(tx, menuID, seed); err != nil {
			return err
		}
	}
	return nil
}

// deleteStaleObservabilityMenus removes observability menus whose paths no longer match any
// frontend route (they render a client-side 404 on click).
func deleteStaleObservabilityMenus(tx *gorm.DB) error {
	for _, stale := range []string{
		observabilityMenuPath + "/metric-sources",
		observabilityMenuPath + "/alert-rules",
		observabilityMenuPath + "/alert-records",
		observabilityMenuPath + "/metric-source",
		observabilityMenuPath + "/alert-rule",
		observabilityMenuPath + "/alert-record",
	} {
		if err := tx.Table("system_menu").Where(menuPathWhereClause, stale).Delete(nil).Error; err != nil {
			return err
		}
	}
	return nil
}

// ensureObservabilityMenu creates the seeded menu row when it does not exist yet, otherwise
// aligns it with the seed, and returns the stored menu id.
func ensureObservabilityMenu(tx *gorm.DB, seed obsMenuSeed) (uint64, error) {
	menuID, err := lookupObservabilityMenuID(tx, seed)
	if err != nil {
		return 0, err
	}
	parentID, err := lookupObservabilityParentID(tx, seed)
	if err != nil {
		return 0, err
	}
	payload := observabilityMenuPayload(seed, parentID)
	if menuID == 0 {
		payload["created_at"] = time.Now()
		if err := tx.Table("system_menu").Create(payload).Error; err != nil {
			return 0, err
		}
		return lookupObservabilityMenuID(tx, seed)
	}
	if err := tx.Table("system_menu").Where(idWhereClause, menuID).Updates(payload).Error; err != nil {
		return 0, err
	}
	return menuID, nil
}

// lookupObservabilityMenuID resolves the id of a seeded menu row: page menus are keyed by
// path, button-level rows have no path and are keyed by their permission string.
func lookupObservabilityMenuID(tx *gorm.DB, seed obsMenuSeed) (uint64, error) {
	var menuID uint64
	lookup := tx.Table("system_menu").Select("id")
	if seed.Path != "" {
		lookup = lookup.Where(menuPathWhereClause, seed.Path)
	} else {
		lookup = lookup.Where(menuPermsWhereClause, seed.Perms)
	}
	if err := lookup.Limit(1).Pluck("id", &menuID).Error; err != nil {
		return 0, err
	}
	return menuID, nil
}

// lookupObservabilityParentID resolves the parent menu id of a seed; a seed without a parent
// path is mounted at the root.
func lookupObservabilityParentID(tx *gorm.DB, seed obsMenuSeed) (uint64, error) {
	if seed.ParentPath == "" {
		return 0, nil
	}
	var parentID uint64
	if err := tx.Table("system_menu").Select("id").Where(menuPathWhereClause, seed.ParentPath).Limit(1).Pluck("id", &parentID).Error; err != nil {
		return 0, err
	}
	return parentID, nil
}

// observabilityMenuPayload is the column set written for one seeded menu row.
func observabilityMenuPayload(seed obsMenuSeed, parentID uint64) map[string]any {
	return map[string]any{
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
		{Locale: "zh-CN", Group: "menu", Key: monitoringMenuTitle, Value: "监控中心"},
		{Locale: "en-US", Group: "menu", Key: monitoringMenuTitle, Value: "Monitoring Center"},
		{Locale: "zh-CN", Group: "menu", Key: alertCenterMenuTitle, Value: "告警中心"},
		{Locale: "en-US", Group: "menu", Key: alertCenterMenuTitle, Value: "Alert Center"},
		{Locale: "zh-CN", Group: "menu", Key: logsMenuTitle, Value: "日志中心"},
		{Locale: "en-US", Group: "menu", Key: logsMenuTitle, Value: "Log Center"},
		{Locale: "zh-CN", Group: "menu", Key: tracesMenuTitle, Value: "链路追踪"},
		{Locale: "en-US", Group: "menu", Key: tracesMenuTitle, Value: "Distributed Tracing"},
		{Locale: "zh-CN", Group: "permission", Key: metricSourceCreatePermTitle, Value: "创建可观测数据源"},
		{Locale: "en-US", Group: "permission", Key: metricSourceCreatePermTitle, Value: "Create observability data sources"},
		{Locale: "zh-CN", Group: "permission", Key: metricSourceUpdatePermTitle, Value: "更新可观测数据源"},
		{Locale: "en-US", Group: "permission", Key: metricSourceUpdatePermTitle, Value: "Update observability data sources"},
		{Locale: "zh-CN", Group: "permission", Key: metricSourceDeletePermTitle, Value: "删除可观测数据源"},
		{Locale: "en-US", Group: "permission", Key: metricSourceDeletePermTitle, Value: "Delete observability data sources"},
		{Locale: "zh-CN", Group: "permission", Key: alertRuleCreatePermTitle, Value: "创建告警规则"},
		{Locale: "en-US", Group: "permission", Key: alertRuleCreatePermTitle, Value: "Create alert rules"},
		{Locale: "zh-CN", Group: "permission", Key: alertRuleUpdatePermTitle, Value: "更新告警规则"},
		{Locale: "en-US", Group: "permission", Key: alertRuleUpdatePermTitle, Value: "Update alert rules"},
		{Locale: "zh-CN", Group: "permission", Key: alertRuleDeletePermTitle, Value: "删除告警规则"},
		{Locale: "en-US", Group: "permission", Key: alertRuleDeletePermTitle, Value: "Delete alert rules"},
		{Locale: "zh-CN", Group: "permission", Key: notificationUpdatePermTitle, Value: "管理通知渠道"},
		{Locale: "en-US", Group: "permission", Key: notificationUpdatePermTitle, Value: "Manage notification channels"},
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
