package observability

import (
	"strings"
	"time"

	"gorm.io/gorm"
)

// Seed and request values reach this statement only through its bind parameter.
const idWhereClause = "id = ?"

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
	{Key: "observability", TitleKey: observabilityMenuTitle, Path: observabilityMenuPath, Type: "M", Icon: "radar", RouteName: "observability", Module: observabilityModuleKey, Sort: 8},
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
	{Key: "observability-history-view", ParentPath: monitoringMenuPath, TitleKey: "business.observability.permission.queryHistoryView", Perms: "business:observability:query-history:view", Type: "F", Module: observabilityModuleKey, Sort: 10},
	{Key: "observability-history-delete", ParentPath: monitoringMenuPath, TitleKey: "business.observability.permission.queryHistoryDelete", Perms: "business:observability:query-history:delete", Type: "F", Module: observabilityModuleKey, Sort: 11},
	{Key: "observability-history-restore", ParentPath: monitoringMenuPath, TitleKey: "business.observability.permission.queryHistoryRestore", Perms: "business:observability:query-history:restore", Type: "F", Module: observabilityModuleKey, Sort: 12},
	{Key: "observability-view-view", ParentPath: monitoringMenuPath, TitleKey: "business.observability.permission.savedViewView", Perms: "business:observability:saved-view:view", Type: "F", Module: observabilityModuleKey, Sort: 13},
	{Key: "observability-view-create", ParentPath: monitoringMenuPath, TitleKey: "business.observability.permission.savedViewCreate", Perms: "business:observability:saved-view:create", Type: "F", Module: observabilityModuleKey, Sort: 14},
	{Key: "observability-view-update", ParentPath: monitoringMenuPath, TitleKey: "business.observability.permission.savedViewUpdate", Perms: "business:observability:saved-view:update", Type: "F", Module: observabilityModuleKey, Sort: 15},
	{Key: "observability-view-delete", ParentPath: monitoringMenuPath, TitleKey: "business.observability.permission.savedViewDelete", Perms: "business:observability:saved-view:delete", Type: "F", Module: observabilityModuleKey, Sort: 16},
	{Key: "observability-view-restore", ParentPath: monitoringMenuPath, TitleKey: "business.observability.permission.savedViewRestore", Perms: "business:observability:saved-view:restore", Type: "F", Module: observabilityModuleKey, Sort: 17},
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
		{Locale: "zh-CN", Group: "workspace", Key: "observability.query_upgrade_required", Value: "请升级客户端并使用 POST 查询"},
		{Locale: "en-US", Group: "workspace", Key: "observability.query_upgrade_required", Value: "Upgrade the client to use POST queries"},
		{Locale: "zh-CN", Group: "workspace", Key: "observability.workspace_unavailable", Value: "查询工作区暂不可用"},
		{Locale: "en-US", Group: "workspace", Key: "observability.workspace_unavailable", Value: "Query workspace is unavailable"},
		{Locale: "zh-CN", Group: "workspace", Key: "observability.query_unavailable", Value: "查询暂不可用"},
		{Locale: "en-US", Group: "workspace", Key: "observability.query_unavailable", Value: "Query is unavailable"},
		{Locale: "zh-CN", Group: "workspace", Key: "observability.workspace_conflict", Value: "内容已改变，请重新加载"},
		{Locale: "en-US", Group: "workspace", Key: "observability.workspace_conflict", Value: "The item changed; reload it"},
		{Locale: "zh-CN", Group: "workspace", Key: "observability.workspace_quota", Value: "已达到查询工作区上限"},
		{Locale: "en-US", Group: "workspace", Key: "observability.workspace_quota", Value: "Query workspace limit reached"},
		{Locale: "zh-CN", Group: "workspace", Key: "observability.query_busy", Value: "查询并发已达到上限"},
		{Locale: "en-US", Group: "workspace", Key: "observability.query_busy", Value: "Query concurrency limit reached"},
		{Locale: "zh-CN", Group: "workspace", Key: "business.observability.workspace.savedViewLabel", Value: "保存的视图 {{id}}"},
		{Locale: "en-US", Group: "workspace", Key: "business.observability.workspace.savedViewLabel", Value: "Saved view {{id}}"},
		{Locale: "zh-CN", Group: "permission", Key: "business.observability.permission.queryHistoryView", Value: "查看自己的查询历史"},
		{Locale: "en-US", Group: "permission", Key: "business.observability.permission.queryHistoryView", Value: "View own query history"},
		{Locale: "zh-CN", Group: "permission", Key: "business.observability.permission.queryHistoryDelete", Value: "删除自己的查询历史"},
		{Locale: "en-US", Group: "permission", Key: "business.observability.permission.queryHistoryDelete", Value: "Delete own query history"},
		{Locale: "zh-CN", Group: "permission", Key: "business.observability.permission.queryHistoryRestore", Value: "恢复自己的查询条件"},
		{Locale: "en-US", Group: "permission", Key: "business.observability.permission.queryHistoryRestore", Value: "Restore own query intent"},
		{Locale: "zh-CN", Group: "permission", Key: "business.observability.permission.savedViewView", Value: "查看自己的保存视图"},
		{Locale: "en-US", Group: "permission", Key: "business.observability.permission.savedViewView", Value: "View own saved views"},
		{Locale: "zh-CN", Group: "permission", Key: "business.observability.permission.savedViewCreate", Value: "创建自己的保存视图"},
		{Locale: "en-US", Group: "permission", Key: "business.observability.permission.savedViewCreate", Value: "Create own saved views"},
		{Locale: "zh-CN", Group: "permission", Key: "business.observability.permission.savedViewUpdate", Value: "更新自己的保存视图"},
		{Locale: "en-US", Group: "permission", Key: "business.observability.permission.savedViewUpdate", Value: "Update own saved views"},
		{Locale: "zh-CN", Group: "permission", Key: "business.observability.permission.savedViewDelete", Value: "删除自己的保存视图"},
		{Locale: "en-US", Group: "permission", Key: "business.observability.permission.savedViewDelete", Value: "Delete own saved views"},
		{Locale: "zh-CN", Group: "permission", Key: "business.observability.permission.savedViewRestore", Value: "恢复自己的保存视图"},
		{Locale: "en-US", Group: "permission", Key: "business.observability.permission.savedViewRestore", Value: "Restore own saved views"},
		{Locale: "zh-CN", Group: "logs", Key: "business.observability.logs.startUTC", Value: "开始时间（UTC）"},
		{Locale: "en-US", Group: "logs", Key: "business.observability.logs.startUTC", Value: "Start time (UTC)"},
		{Locale: "zh-CN", Group: "logs", Key: "business.observability.logs.endUTC", Value: "结束时间（UTC）"},
		{Locale: "en-US", Group: "logs", Key: "business.observability.logs.endUTC", Value: "End time (UTC)"},
		{Locale: "zh-CN", Group: "logs", Key: "business.observability.logs.limit", Value: "最多日志条数"},
		{Locale: "en-US", Group: "logs", Key: "business.observability.logs.limit", Value: "Maximum log rows"},
		{Locale: "zh-CN", Group: "logs", Key: "business.observability.logs.stop", Value: "停止查询"},
		{Locale: "en-US", Group: "logs", Key: "business.observability.logs.stop", Value: "Stop query"},
		{Locale: "zh-CN", Group: "logs", Key: "business.observability.logs.running", Value: "查询中，已有结果来自上一次查询"},
		{Locale: "en-US", Group: "logs", Key: "business.observability.logs.running", Value: "Query running; retained rows are from the previous query"},
		{Locale: "zh-CN", Group: "logs", Key: "business.observability.logs.stopped", Value: "查询已停止，已有结果来自上一次查询"},
		{Locale: "en-US", Group: "logs", Key: "business.observability.logs.stopped", Value: "Query stopped; retained rows are from the previous query"},
		{Locale: "zh-CN", Group: "logs", Key: "business.observability.logs.invalidRange", Value: "请选择不超过 24 小时的有效 UTC 时间范围"},
		{Locale: "en-US", Group: "logs", Key: "business.observability.logs.invalidRange", Value: "Choose a valid UTC range of up to 24 hours"},
		{Locale: "zh-CN", Group: "logs", Key: "business.observability.logs.budget", Value: "单次范围最多 24 小时，响应最多 8 MiB；按所选条数截取结果"},
		{Locale: "en-US", Group: "logs", Key: "business.observability.logs.budget", Value: "Up to 24 hours and 8 MiB per query; results are capped at the selected row limit"},
		{Locale: "zh-CN", Group: "logs", Key: "business.observability.logs.forbidden", Value: "没有日志查询权限"},
		{Locale: "en-US", Group: "logs", Key: "business.observability.logs.forbidden", Value: "No permission to query logs"},
		{Locale: "zh-CN", Group: "traces", Key: "business.observability.traces.budget", Value: "单次范围最多 24 小时，响应最多 8 MiB，最多展示 100 条链路"},
		{Locale: "en-US", Group: "traces", Key: "business.observability.traces.budget", Value: "Up to 24 hours and 8 MiB per query; at most 100 traces are shown"},
		{Locale: "zh-CN", Group: "traces", Key: "business.observability.traces.forbidden", Value: "没有链路追踪查询权限"},
		{Locale: "en-US", Group: "traces", Key: "business.observability.traces.forbidden", Value: "No permission to query traces"},
		{Locale: "zh-CN", Group: "traces", Key: "business.observability.traces.running", Value: "查询中，已有链路来自上一次查询"},
		{Locale: "en-US", Group: "traces", Key: "business.observability.traces.running", Value: "Query running; retained traces are from the previous query"},
		{Locale: "zh-CN", Group: "traces", Key: "business.observability.traces.stopped", Value: "查询已停止，已有链路来自上一次查询"},
		{Locale: "en-US", Group: "traces", Key: "business.observability.traces.stopped", Value: "Query stopped; retained traces are from the previous query"},
		{Locale: "zh-CN", Group: "traces", Key: "business.observability.traces.inputsChanged", Value: "查询条件已改变，当前显示上一次链路结果；请重新运行。"},
		{Locale: "en-US", Group: "traces", Key: "business.observability.traces.inputsChanged", Value: "Query inputs changed; the previous trace result is shown. Run again."},
		{Locale: "zh-CN", Group: "traces", Key: "business.observability.traces.resultContext", Value: "结果来源：{{datasource}} · UTC {{start}} 至 {{end}} · 最多 100 条"},
		{Locale: "en-US", Group: "traces", Key: "business.observability.traces.resultContext", Value: "Result source: {{datasource}} · UTC {{start}} to {{end}} · up to 100 traces"},
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
	// Shared definitions are not grants. Private O4 writes require explicit IAM assignment.
	if strings.Contains(seed.Perms, ":query-history:") || strings.Contains(seed.Perms, ":saved-view:") {
		return nil
	}
	if menuID == 0 || !tx.Migrator().HasTable("system_role") {
		return nil
	}
	var roleID uint64
	if err := tx.Table("system_role").Select("id").Where("tenant_id = 0 AND role_key = ?", "admin").Limit(1).Pluck("id", &roleID).Error; err != nil {
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
