package cmdb

import (
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

const (
	cmdbModuleKey          = "business.cmdb"
	cmdbHostMenuKey        = "operations.cmdb.host.menu"
	cmdbHostPermDetailKey  = "business.cmdb.host.permission.detail"
	cmdbHostPermCreateKey  = "business.cmdb.host.permission.create"
	cmdbHostPermUpdateKey  = "business.cmdb.host.permission.update"
	cmdbHostPermDeleteKey  = "business.cmdb.host.permission.delete"
	cmdbHostPermCollectKey = "business.cmdb.host.permission.collect"
	cmdbHostPermStatusKey  = "business.cmdb.host.permission.status"
	cmdbHostPermExportKey  = "business.cmdb.host.permission.export"
	cmdbHostPermImportKey  = "business.cmdb.host.permission.import"
	cmdbGroupMenuKey       = "operations.cmdb.group.menu"
	cmdbGroupPermDetailKey = "business.cmdb.group.permission.detail"
	cmdbGroupPermCreateKey = "business.cmdb.group.permission.create"
	cmdbGroupPermUpdateKey = "business.cmdb.group.permission.update"
	cmdbGroupPermDeleteKey = "business.cmdb.group.permission.delete"
	cmdbGroupPermExportKey = "business.cmdb.group.permission.export"
	cmdbGroupPermImportKey = "business.cmdb.group.permission.import"
	cmdbLabelMenuKey       = "operations.cmdb.label.menu"
	cmdbLabelPermCreateKey = "business.cmdb.label.permission.create"
	cmdbLabelPermUpdateKey = "business.cmdb.label.permission.update"
	cmdbLabelPermDeleteKey = "business.cmdb.label.permission.delete"
	cmdbLabelPermExportKey = "business.cmdb.label.permission.export"
	cmdbLabelPermImportKey = "business.cmdb.label.permission.import"
	cmdbMenuKey            = "operations.cmdb.menu"
	cmdbHostListRoute      = "cmdb-host-list"
	cmdbGroupListRoute     = "cmdb-group-list"
	cmdbLabelListRoute     = "cmdb-label-list"
)

type cmdbMenuSeed struct {
	Key       string
	ParentKey string
	TitleKey  string
	Path      string
	Component string
	PagePerm  string
	Perms     string
	Type      string
	Icon      string
	RouteName string
	Module    string
	Sort      int
	IsCache   int
}

type i18nSeed struct {
	Module string
	Locale string
	Group  string
	Key    string
	Value  string
}

func hostMenuSeeds() []cmdbMenuSeed {
	return []cmdbMenuSeed{
		{
			Key:       "operations-cmdb-host",
			ParentKey: "cmdb",
			TitleKey:  cmdbHostMenuKey,
			Path:      "/business/cmdb/host",
			Component: "business/cmdb/host/CmdbHostList",
			PagePerm:  "business:cmdb:host:view",
			Type:      "C",
			Icon:      "cloud",
			Module:    cmdbModuleKey,
			RouteName: cmdbHostListRoute,
			Sort:      1,
		},
		{Key: "operations-cmdb-host-detail", ParentKey: cmdbHostListRoute, TitleKey: cmdbHostPermDetailKey, Perms: "business:cmdb:host:detail", Type: "F", Module: cmdbModuleKey, Sort: 1},
		{Key: "operations-cmdb-host-create", ParentKey: cmdbHostListRoute, TitleKey: cmdbHostPermCreateKey, Perms: "business:cmdb:host:create", Type: "F", Module: cmdbModuleKey, Sort: 2},
		{Key: "operations-cmdb-host-update", ParentKey: cmdbHostListRoute, TitleKey: cmdbHostPermUpdateKey, Perms: "business:cmdb:host:update", Type: "F", Module: cmdbModuleKey, Sort: 3},
		{Key: "operations-cmdb-host-delete", ParentKey: cmdbHostListRoute, TitleKey: cmdbHostPermDeleteKey, Perms: "business:cmdb:host:delete", Type: "F", Module: cmdbModuleKey, Sort: 4},
		{Key: "operations-cmdb-host-collect", ParentKey: cmdbHostListRoute, TitleKey: cmdbHostPermCollectKey, Perms: "business:cmdb:host:collect", Type: "F", Module: cmdbModuleKey, Sort: 5},
		{Key: "operations-cmdb-host-status", ParentKey: cmdbHostListRoute, TitleKey: cmdbHostPermStatusKey, Perms: "business:cmdb:host:status", Type: "F", Module: cmdbModuleKey, Sort: 6},
		{Key: "operations-cmdb-host-export", ParentKey: cmdbHostListRoute, TitleKey: cmdbHostPermExportKey, Perms: "business:cmdb:host:export", Type: "F", Module: cmdbModuleKey, Sort: 7},
		{Key: "operations-cmdb-host-import", ParentKey: cmdbHostListRoute, TitleKey: cmdbHostPermImportKey, Perms: "business:cmdb:host:import", Type: "F", Module: cmdbModuleKey, Sort: 8},
		{
			Key:       "operations-cmdb-group",
			ParentKey: "cmdb",
			TitleKey:  cmdbGroupMenuKey,
			Path:      "/business/cmdb/group",
			Component: "business/cmdb/group/CmdbGroupList",
			PagePerm:  "business:cmdb:group:view",
			Type:      "C",
			Icon:      "apps",
			Module:    cmdbModuleKey,
			RouteName: cmdbGroupListRoute,
			Sort:      2,
		},
		{Key: "operations-cmdb-group-detail", ParentKey: cmdbGroupListRoute, TitleKey: cmdbGroupPermDetailKey, Perms: "business:cmdb:group:detail", Type: "F", Module: cmdbModuleKey, Sort: 1},
		{Key: "operations-cmdb-group-create", ParentKey: cmdbGroupListRoute, TitleKey: cmdbGroupPermCreateKey, Perms: "business:cmdb:group:create", Type: "F", Module: cmdbModuleKey, Sort: 2},
		{Key: "operations-cmdb-group-update", ParentKey: cmdbGroupListRoute, TitleKey: cmdbGroupPermUpdateKey, Perms: "business:cmdb:group:update", Type: "F", Module: cmdbModuleKey, Sort: 3},
		{Key: "operations-cmdb-group-delete", ParentKey: cmdbGroupListRoute, TitleKey: cmdbGroupPermDeleteKey, Perms: "business:cmdb:group:delete", Type: "F", Module: cmdbModuleKey, Sort: 4},
		{Key: "operations-cmdb-group-export", ParentKey: cmdbGroupListRoute, TitleKey: cmdbGroupPermExportKey, Perms: "business:cmdb:group:export", Type: "F", Module: cmdbModuleKey, Sort: 5},
		{Key: "operations-cmdb-group-import", ParentKey: cmdbGroupListRoute, TitleKey: cmdbGroupPermImportKey, Perms: "business:cmdb:group:import", Type: "F", Module: cmdbModuleKey, Sort: 6},
		{
			Key:       "operations-cmdb-label",
			ParentKey: "cmdb",
			TitleKey:  cmdbLabelMenuKey,
			Path:      "/business/cmdb/label",
			Component: "business/cmdb/label/CmdbLabelSchemaList",
			PagePerm:  "business:cmdb:label:view",
			Type:      "C",
			Icon:      "tags",
			Module:    cmdbModuleKey,
			RouteName: cmdbLabelListRoute,
			Sort:      3,
		},
		{Key: "operations-cmdb-label-create", ParentKey: cmdbLabelListRoute, TitleKey: cmdbLabelPermCreateKey, Perms: "business:cmdb:label:create", Type: "F", Module: cmdbModuleKey, Sort: 1},
		{Key: "operations-cmdb-label-update", ParentKey: cmdbLabelListRoute, TitleKey: cmdbLabelPermUpdateKey, Perms: "business:cmdb:label:update", Type: "F", Module: cmdbModuleKey, Sort: 2},
		{Key: "operations-cmdb-label-delete", ParentKey: cmdbLabelListRoute, TitleKey: cmdbLabelPermDeleteKey, Perms: "business:cmdb:label:delete", Type: "F", Module: cmdbModuleKey, Sort: 3},
		{Key: "operations-cmdb-label-export", ParentKey: cmdbLabelListRoute, TitleKey: cmdbLabelPermExportKey, Perms: "business:cmdb:label:export", Type: "F", Module: cmdbModuleKey, Sort: 4},
		{Key: "operations-cmdb-label-import", ParentKey: cmdbLabelListRoute, TitleKey: cmdbLabelPermImportKey, Perms: "business:cmdb:label:import", Type: "F", Module: cmdbModuleKey, Sort: 5},
	}
}

func topLevelMenuSeeds() []cmdbMenuSeed {
	return []cmdbMenuSeed{
		{
			Key:       "operations-cmdb",
			ParentKey: "",
			TitleKey:  cmdbMenuKey,
			Path:      "/business/cmdb",
			Type:      "M",
			Module:    cmdbModuleKey,
			Icon:      "storage",
			RouteName: "cmdb",
			Sort:      1,
		},
	}
}

func seedHostMenus(db *gorm.DB) error {
	if db == nil {
		return nil
	}
	if err := seedCmdbDicts(db); err != nil {
		return err
	}
	if err := seedCmdbLabelSchemas(db); err != nil {
		return err
	}
	return ensureCmdbMenuSeeds(db, append(topLevelMenuSeeds(), hostMenuSeeds()...))
}

func seedHostPermissions(db *gorm.DB) error { return nil }

func seedGroupMenus(db *gorm.DB) error       { return nil }
func seedGroupPermissions(db *gorm.DB) error { return nil }
func seedLabelMenus(db *gorm.DB) error       { return nil }
func seedLabelI18n(db *gorm.DB) error        { return nil }

func ensureCmdbMenuSeeds(db *gorm.DB, seeds []cmdbMenuSeed) error {
	for _, seed := range seeds {
		var menuID uint64
		if seed.Path != "" {
			if err := db.Table("system_menu").Select("id").Where("path = ?", seed.Path).Limit(1).Pluck("id", &menuID).Error; err != nil {
				return err
			}
		} else if seed.Perms != "" {
			if err := db.Table("system_menu").Select("id").Where("perms = ?", seed.Perms).Limit(1).Pluck("id", &menuID).Error; err != nil {
				return err
			}
		}
		parentID, err := resolveCmdbMenuParentID(db, seed.ParentKey)
		if err != nil {
			return err
		}
		if menuID == 0 {
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
				"created_at": time.Now(),
				"updated_at": time.Now(),
			}
			if err := db.Table("system_menu").Create(payload).Error; err != nil {
				return err
			}
			if seed.Path != "" {
				if err := db.Table("system_menu").Select("id").Where("path = ?", seed.Path).Limit(1).Pluck("id", &menuID).Error; err != nil {
					return err
				}
			} else if seed.Perms != "" {
				if err := db.Table("system_menu").Select("id").Where("perms = ?", seed.Perms).Limit(1).Pluck("id", &menuID).Error; err != nil {
					return err
				}
			}
		} else {
			updates := map[string]interface{}{
				"parent_id":  parentID,
				"title_key":  seed.TitleKey,
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
			updates["path"] = seed.Path
			if err := db.Table("system_menu").Where(idWhereClause, menuID).Updates(updates).Error; err != nil {
				return err
			}
		}
		if err := ensureCmdbAdminBindings(db, menuID, seed); err != nil {
			return err
		}
	}
	return nil
}

func resolveCmdbMenuParentID(db *gorm.DB, parentKey string) (uint64, error) {
	if parentKey == "" {
		return 0, nil
	}
	var parentID uint64
	if err := db.Table("system_menu").Select("id").Where("route_name = ?", parentKey).Limit(1).Pluck("id", &parentID).Error; err != nil {
		return 0, err
	}
	return parentID, nil
}

func ensureCmdbAdminBindings(db *gorm.DB, menuID uint64, seed cmdbMenuSeed) error {
	if menuID == 0 || !db.Migrator().HasTable("system_role") {
		return nil
	}
	var adminRoleID uint64
	if err := db.Table("system_role").Select("id").Where("role_key = ?", "admin").Limit(1).Pluck("id", &adminRoleID).Error; err != nil {
		return err
	}
	if adminRoleID == 0 {
		return nil
	}
	if seed.Type != "F" && db.Migrator().HasTable("system_role_menu") {
		var count int64
		if err := db.Table("system_role_menu").Where("role_id = ? AND menu_id = ?", adminRoleID, menuID).Count(&count).Error; err != nil {
			return err
		}
		if count == 0 {
			if err := db.Exec("INSERT INTO system_role_menu (role_id, menu_id) VALUES (?, ?)", adminRoleID, menuID).Error; err != nil {
				return err
			}
		}
	}
	if err := ensureCmdbAdminPermission(db, adminRoleID, seed.PagePerm); err != nil {
		return err
	}
	return ensureCmdbAdminPermission(db, adminRoleID, seed.Perms)
}

func ensureCmdbAdminPermission(db *gorm.DB, adminRoleID uint64, permissionKey string) error {
	if permissionKey == "" || !db.Migrator().HasTable("system_role_permission") {
		return nil
	}
	var count int64
	if err := db.Table("system_role_permission").Where("role_id = ? AND permission_key = ?", adminRoleID, permissionKey).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	return db.Exec("INSERT INTO system_role_permission (role_id, permission_key) VALUES (?, ?)", adminRoleID, permissionKey).Error
}

func seedHostI18n(db *gorm.DB) error {
	if db == nil {
		return nil
	}
	i18nEntries := []i18nSeed{
		{Module: cmdbModuleKey, Locale: "zh-CN", Group: "menu", Key: "operations.menu", Value: "运维平台"},
		{Module: cmdbModuleKey, Locale: "en-US", Group: "menu", Key: "operations.menu", Value: "Operations"},
		{Module: cmdbModuleKey, Locale: "zh-CN", Group: "menu", Key: cmdbMenuKey, Value: "CMDB"},
		{Module: cmdbModuleKey, Locale: "en-US", Group: "menu", Key: cmdbMenuKey, Value: "CMDB"},
		{Module: cmdbModuleKey, Locale: "zh-CN", Group: "menu", Key: cmdbHostMenuKey, Value: "主机管理"},
		{Module: cmdbModuleKey, Locale: "en-US", Group: "menu", Key: cmdbHostMenuKey, Value: "Host Management"},
		{Module: cmdbModuleKey, Locale: "zh-CN", Group: "page", Key: "operations.cmdb.host.detail", Value: "主机详情"},
		{Module: cmdbModuleKey, Locale: "en-US", Group: "page", Key: "operations.cmdb.host.detail", Value: "Host Detail"},
		{Module: cmdbModuleKey, Locale: "zh-CN", Group: "menu", Key: cmdbGroupMenuKey, Value: "主机分组"},
		{Module: cmdbModuleKey, Locale: "en-US", Group: "menu", Key: cmdbGroupMenuKey, Value: "Host Groups"},
		{Module: cmdbModuleKey, Locale: "zh-CN", Group: "menu", Key: cmdbLabelMenuKey, Value: "标签管理"},
		{Module: cmdbModuleKey, Locale: "en-US", Group: "menu", Key: cmdbLabelMenuKey, Value: "Label Management"},
		{Module: cmdbModuleKey, Locale: "zh-CN", Group: "page", Key: "business.cmdb.host.title", Value: "主机管理"},
		{Module: cmdbModuleKey, Locale: "en-US", Group: "page", Key: "business.cmdb.host.title", Value: "Host Management"},
		{Module: cmdbModuleKey, Locale: "zh-CN", Group: "page", Key: "business.cmdb.host.hero.eyebrow", Value: "运维平台 / 主机台账"},
		{Module: cmdbModuleKey, Locale: "en-US", Group: "page", Key: "business.cmdb.host.hero.eyebrow", Value: "Operations / Host Inventory"},
		{Module: cmdbModuleKey, Locale: "zh-CN", Group: "page", Key: "business.cmdb.host.hero.title", Value: "在统一视图中管理主机、标签与配置采集"},
		{Module: cmdbModuleKey, Locale: "en-US", Group: "page", Key: "business.cmdb.host.hero.title", Value: "Manage hosts, labels, and collection in one view"},
		{Module: cmdbModuleKey, Locale: "zh-CN", Group: "page", Key: "business.cmdb.host.hero.total", Value: "主机总数"},
		{Module: cmdbModuleKey, Locale: "en-US", Group: "page", Key: "business.cmdb.host.hero.total", Value: "Total Hosts"},
		{Module: cmdbModuleKey, Locale: "zh-CN", Group: "page", Key: "business.cmdb.host.hero.totalHint", Value: "当前筛选条件下的主机总量。"},
		{Module: cmdbModuleKey, Locale: "en-US", Group: "page", Key: "business.cmdb.host.hero.totalHint", Value: "Total hosts under the current filter."},
		{Module: cmdbModuleKey, Locale: "zh-CN", Group: "page", Key: "business.cmdb.host.hero.online", Value: "可运维主机"},
		{Module: cmdbModuleKey, Locale: "en-US", Group: "page", Key: "business.cmdb.host.hero.online", Value: "Operable Hosts"},
		{Module: cmdbModuleKey, Locale: "zh-CN", Group: "page", Key: "business.cmdb.host.hero.onlineHint", Value: "运维状态为可运维的主机数量。"},
		{Module: cmdbModuleKey, Locale: "en-US", Group: "page", Key: "business.cmdb.host.hero.onlineHint", Value: "Hosts whose operations status is operable."},
		{Module: cmdbModuleKey, Locale: "zh-CN", Group: "page", Key: "business.cmdb.host.hero.maintenance", Value: "维护中"},
		{Module: cmdbModuleKey, Locale: "en-US", Group: "page", Key: "business.cmdb.host.hero.maintenance", Value: "Under Maintenance"},
		{Module: cmdbModuleKey, Locale: "zh-CN", Group: "page", Key: "business.cmdb.host.hero.maintenanceHint", Value: "状态为维护中的主机数量。"},
		{Module: cmdbModuleKey, Locale: "en-US", Group: "page", Key: "business.cmdb.host.hero.maintenanceHint", Value: "Hosts whose status is maintenance."},
		{Module: cmdbModuleKey, Locale: "zh-CN", Group: "page", Key: "business.cmdb.host.hero.scope", Value: "数据范围"},
		{Module: cmdbModuleKey, Locale: "en-US", Group: "page", Key: "business.cmdb.host.hero.scope", Value: "Data Scope"},
		{Module: cmdbModuleKey, Locale: "zh-CN", Group: "page", Key: "business.cmdb.host.hero.scopeValue", Value: "按当前登录主体可见数据"},
		{Module: cmdbModuleKey, Locale: "en-US", Group: "page", Key: "business.cmdb.host.hero.scopeValue", Value: "Visible to the current login context"},
		{Module: cmdbModuleKey, Locale: "zh-CN", Group: "page", Key: "business.cmdb.host.hero.scopeHint", Value: "主机列表和详情遵循系统域数据范围。"},
		{Module: cmdbModuleKey, Locale: "en-US", Group: "page", Key: "business.cmdb.host.hero.scopeHint", Value: "Host lists and details follow the system data scope."},
		{Module: cmdbModuleKey, Locale: "zh-CN", Group: "page", Key: "business.cmdb.host.hero.statusHint", Value: "当前主机运维状态，不代表实时连通性。"},
		{Module: cmdbModuleKey, Locale: "en-US", Group: "page", Key: "business.cmdb.host.hero.statusHint", Value: "Current operations status, not real-time connectivity."},
		{Module: cmdbModuleKey, Locale: "zh-CN", Group: "page", Key: "business.cmdb.host.hero.osHint", Value: "当前操作系统类型。"},
		{Module: cmdbModuleKey, Locale: "en-US", Group: "page", Key: "business.cmdb.host.hero.osHint", Value: "Current operating system type."},
		{Module: cmdbModuleKey, Locale: "zh-CN", Group: "page", Key: "business.cmdb.host.hero.osSummary", Value: "操作系统分布"},
		{Module: cmdbModuleKey, Locale: "en-US", Group: "page", Key: "business.cmdb.host.hero.osSummary", Value: "OS Distribution"},
		{Module: cmdbModuleKey, Locale: "zh-CN", Group: "page", Key: "business.cmdb.host.hero.osSummaryHint", Value: "当前页主机的 Linux / Windows 数量。"},
		{Module: cmdbModuleKey, Locale: "en-US", Group: "page", Key: "business.cmdb.host.hero.osSummaryHint", Value: "Linux / Windows count on the current page."},
		{Module: cmdbModuleKey, Locale: "zh-CN", Group: "page", Key: "business.cmdb.host.hero.labelsHint", Value: "当前主机标签数量。"},
		{Module: cmdbModuleKey, Locale: "en-US", Group: "page", Key: "business.cmdb.host.hero.labelsHint", Value: "Current host label count."},
		{Module: cmdbModuleKey, Locale: "zh-CN", Group: "page", Key: "business.cmdb.host.hero.componentsHint", Value: "当前主机已装组件数量。"},
		{Module: cmdbModuleKey, Locale: "en-US", Group: "page", Key: "business.cmdb.host.hero.componentsHint", Value: "Installed component count on the current host."},
		{Module: cmdbModuleKey, Locale: "zh-CN", Group: "page", Key: "business.cmdb.host.baseInfo", Value: "基础信息"},
		{Module: cmdbModuleKey, Locale: "en-US", Group: "page", Key: "business.cmdb.host.baseInfo", Value: "Base Information"},
		{Module: cmdbModuleKey, Locale: "zh-CN", Group: "page", Key: "business.cmdb.host.systemConfig", Value: "系统配置"},
		{Module: cmdbModuleKey, Locale: "en-US", Group: "page", Key: "business.cmdb.host.systemConfig", Value: "System Configuration"},
		{Module: cmdbModuleKey, Locale: "zh-CN", Group: "page", Key: "business.cmdb.host.labelsEmpty", Value: "暂无标签"},
		{Module: cmdbModuleKey, Locale: "en-US", Group: "page", Key: "business.cmdb.host.labelsEmpty", Value: "No labels"},
		{Module: cmdbModuleKey, Locale: "zh-CN", Group: "page", Key: "business.cmdb.host.componentsEmpty", Value: "暂无已装组件"},
		{Module: cmdbModuleKey, Locale: "en-US", Group: "page", Key: "business.cmdb.host.componentsEmpty", Value: "No installed components"},
		{Module: cmdbModuleKey, Locale: "zh-CN", Group: "page", Key: "business.cmdb.host.collectSshUserPlaceholder", Value: "root"},
		{Module: cmdbModuleKey, Locale: "en-US", Group: "page", Key: "business.cmdb.host.collectSshUserPlaceholder", Value: "root"},
		{Module: cmdbModuleKey, Locale: "zh-CN", Group: "page", Key: "business.cmdb.host.collectPrivateKeyPlaceholder", Value: "粘贴 SSH 私钥内容"},
		{Module: cmdbModuleKey, Locale: "en-US", Group: "page", Key: "business.cmdb.host.collectPrivateKeyPlaceholder", Value: "Paste SSH private key content"},
		{Module: cmdbModuleKey, Locale: "zh-CN", Group: "menu", Key: "business.cmdb.host.os.linux", Value: "Linux"},
		{Module: cmdbModuleKey, Locale: "en-US", Group: "menu", Key: "business.cmdb.host.os.linux", Value: "Linux"},
		{Module: cmdbModuleKey, Locale: "zh-CN", Group: "menu", Key: "business.cmdb.host.os.windows", Value: "Windows"},
		{Module: cmdbModuleKey, Locale: "en-US", Group: "menu", Key: "business.cmdb.host.os.windows", Value: "Windows"},
		{Module: cmdbModuleKey, Locale: "zh-CN", Group: "menu", Key: "business.cmdb.host.status.maintenance", Value: "维护中"},
		{Module: cmdbModuleKey, Locale: "en-US", Group: "menu", Key: "business.cmdb.host.status.maintenance", Value: "Maintenance"},
		{Module: cmdbModuleKey, Locale: "zh-CN", Group: "page", Key: "business.cmdb.group.hero.eyebrow", Value: "运维平台 / 主机分组"},
		{Module: cmdbModuleKey, Locale: "en-US", Group: "page", Key: "business.cmdb.group.hero.eyebrow", Value: "Operations / Host Groups"},
		{Module: cmdbModuleKey, Locale: "zh-CN", Group: "page", Key: "business.cmdb.group.hero.title", Value: "通过标签条件管理可复用的主机集合"},
		{Module: cmdbModuleKey, Locale: "en-US", Group: "page", Key: "business.cmdb.group.hero.title", Value: "Manage reusable host sets with label conditions"},
		{Module: cmdbModuleKey, Locale: "zh-CN", Group: "page", Key: "business.cmdb.group.hero.total", Value: "分组总数"},
		{Module: cmdbModuleKey, Locale: "en-US", Group: "page", Key: "business.cmdb.group.hero.total", Value: "Total Groups"},
		{Module: cmdbModuleKey, Locale: "zh-CN", Group: "page", Key: "business.cmdb.group.hero.totalHint", Value: "当前可见的主机分组数量。"},
		{Module: cmdbModuleKey, Locale: "en-US", Group: "page", Key: "business.cmdb.group.hero.totalHint", Value: "All visible host groups under the current scope."},
		{Module: cmdbModuleKey, Locale: "zh-CN", Group: "page", Key: "business.cmdb.group.hero.members", Value: "选中分组成员"},
		{Module: cmdbModuleKey, Locale: "en-US", Group: "page", Key: "business.cmdb.group.hero.members", Value: "Selected Group Members"},
		{Module: cmdbModuleKey, Locale: "zh-CN", Group: "page", Key: "business.cmdb.group.hero.membersHint", Value: "当前选中分组的成员数量。"},
		{Module: cmdbModuleKey, Locale: "en-US", Group: "page", Key: "business.cmdb.group.hero.membersHint", Value: "Member count of the selected group."},
		{Module: cmdbModuleKey, Locale: "zh-CN", Group: "page", Key: "business.cmdb.group.hero.scope", Value: "数据范围"},
		{Module: cmdbModuleKey, Locale: "en-US", Group: "page", Key: "business.cmdb.group.hero.scope", Value: "Data Scope"},
		{Module: cmdbModuleKey, Locale: "zh-CN", Group: "page", Key: "business.cmdb.group.hero.scopeValue", Value: "按当前登录主体可见数据"},
		{Module: cmdbModuleKey, Locale: "en-US", Group: "page", Key: "business.cmdb.group.hero.scopeValue", Value: "Visible to the current login context"},
		{Module: cmdbModuleKey, Locale: "zh-CN", Group: "page", Key: "business.cmdb.group.hero.scopeHint", Value: "成员计算遵循当前请求的数据范围。"},
		{Module: cmdbModuleKey, Locale: "en-US", Group: "page", Key: "business.cmdb.group.hero.scopeHint", Value: "Member computation follows the current request scope."},
		{Module: cmdbModuleKey, Locale: "zh-CN", Group: "page", Key: "business.cmdb.group.hero.rules", Value: "筛选规则"},
		{Module: cmdbModuleKey, Locale: "en-US", Group: "page", Key: "business.cmdb.group.hero.rules", Value: "Filter Rules"},
		{Module: cmdbModuleKey, Locale: "zh-CN", Group: "page", Key: "business.cmdb.group.hero.rulesHint", Value: "当前选中分组的规则条数。"},
		{Module: cmdbModuleKey, Locale: "en-US", Group: "page", Key: "business.cmdb.group.hero.rulesHint", Value: "Rule count of the selected group."},
		{Module: cmdbModuleKey, Locale: "zh-CN", Group: "page", Key: "business.cmdb.group.tree.title", Value: "分组树"},
		{Module: cmdbModuleKey, Locale: "en-US", Group: "page", Key: "business.cmdb.group.tree.title", Value: "Group Tree"},
		{Module: cmdbModuleKey, Locale: "zh-CN", Group: "page", Key: "business.cmdb.group.parent", Value: "上级分组"},
		{Module: cmdbModuleKey, Locale: "en-US", Group: "page", Key: "business.cmdb.group.parent", Value: "Parent Group"},
		{Module: cmdbModuleKey, Locale: "zh-CN", Group: "page", Key: "business.cmdb.group.noParent", Value: "无上级分组"},
		{Module: cmdbModuleKey, Locale: "en-US", Group: "page", Key: "business.cmdb.group.noParent", Value: "No Parent Group"},
		{Module: cmdbModuleKey, Locale: "zh-CN", Group: "page", Key: "business.cmdb.group.createChild", Value: "新增子分组"},
		{Module: cmdbModuleKey, Locale: "en-US", Group: "page", Key: "business.cmdb.group.createChild", Value: "Add Subgroup"},
		{Module: cmdbModuleKey, Locale: "zh-CN", Group: "page", Key: "business.cmdb.group.childCount", Value: "子分组"},
		{Module: cmdbModuleKey, Locale: "en-US", Group: "page", Key: "business.cmdb.group.childCount", Value: "Subgroups"},
		{Module: cmdbModuleKey, Locale: "zh-CN", Group: "page", Key: "business.cmdb.group.condition.ruleIndex", Value: "条件 {{count}}"},
		{Module: cmdbModuleKey, Locale: "en-US", Group: "page", Key: "business.cmdb.group.condition.ruleIndex", Value: "Condition {{count}}"},
		{Module: cmdbModuleKey, Locale: "zh-CN", Group: "permission", Key: cmdbHostPermDetailKey, Value: "查看主机详情"},
		{Module: cmdbModuleKey, Locale: "en-US", Group: "permission", Key: cmdbHostPermDetailKey, Value: "View host detail"},
		{Module: cmdbModuleKey, Locale: "zh-CN", Group: "permission", Key: cmdbHostPermCreateKey, Value: "新增主机"},
		{Module: cmdbModuleKey, Locale: "en-US", Group: "permission", Key: cmdbHostPermCreateKey, Value: "Create hosts"},
		{Module: cmdbModuleKey, Locale: "zh-CN", Group: "permission", Key: cmdbHostPermUpdateKey, Value: "编辑主机"},
		{Module: cmdbModuleKey, Locale: "en-US", Group: "permission", Key: cmdbHostPermUpdateKey, Value: "Update hosts"},
		{Module: cmdbModuleKey, Locale: "zh-CN", Group: "permission", Key: cmdbHostPermDeleteKey, Value: "删除主机"},
		{Module: cmdbModuleKey, Locale: "en-US", Group: "permission", Key: cmdbHostPermDeleteKey, Value: "Delete hosts"},
		{Module: cmdbModuleKey, Locale: "zh-CN", Group: "permission", Key: cmdbHostPermCollectKey, Value: "采集主机配置"},
		{Module: cmdbModuleKey, Locale: "en-US", Group: "permission", Key: cmdbHostPermCollectKey, Value: "Collect host config"},
		{Module: cmdbModuleKey, Locale: "zh-CN", Group: "permission", Key: cmdbHostPermStatusKey, Value: "更新主机状态"},
		{Module: cmdbModuleKey, Locale: "en-US", Group: "permission", Key: cmdbHostPermStatusKey, Value: "Update host status"},
		{Module: cmdbModuleKey, Locale: "zh-CN", Group: "permission", Key: cmdbHostPermExportKey, Value: "导出主机资产"},
		{Module: cmdbModuleKey, Locale: "en-US", Group: "permission", Key: cmdbHostPermExportKey, Value: "Export host assets"},
		{Module: cmdbModuleKey, Locale: "zh-CN", Group: "permission", Key: cmdbHostPermImportKey, Value: "导入主机"},
		{Module: cmdbModuleKey, Locale: "en-US", Group: "permission", Key: cmdbHostPermImportKey, Value: "Import hosts"},
		{Module: cmdbModuleKey, Locale: "zh-CN", Group: "permission", Key: cmdbGroupPermDetailKey, Value: "查看主机分组"},
		{Module: cmdbModuleKey, Locale: "en-US", Group: "permission", Key: cmdbGroupPermDetailKey, Value: "View host groups"},
		{Module: cmdbModuleKey, Locale: "zh-CN", Group: "permission", Key: cmdbGroupPermCreateKey, Value: "新增主机分组"},
		{Module: cmdbModuleKey, Locale: "en-US", Group: "permission", Key: cmdbGroupPermCreateKey, Value: "Create host groups"},
		{Module: cmdbModuleKey, Locale: "zh-CN", Group: "permission", Key: cmdbGroupPermUpdateKey, Value: "编辑主机分组"},
		{Module: cmdbModuleKey, Locale: "en-US", Group: "permission", Key: cmdbGroupPermUpdateKey, Value: "Update host groups"},
		{Module: cmdbModuleKey, Locale: "zh-CN", Group: "permission", Key: cmdbGroupPermDeleteKey, Value: "删除主机分组"},
		{Module: cmdbModuleKey, Locale: "en-US", Group: "permission", Key: cmdbGroupPermDeleteKey, Value: "Delete host groups"},
		{Module: cmdbModuleKey, Locale: "zh-CN", Group: "permission", Key: cmdbGroupPermExportKey, Value: "导出主机分组"},
		{Module: cmdbModuleKey, Locale: "en-US", Group: "permission", Key: cmdbGroupPermExportKey, Value: "Export host groups"},
		{Module: cmdbModuleKey, Locale: "zh-CN", Group: "permission", Key: cmdbGroupPermImportKey, Value: "导入主机分组"},
		{Module: cmdbModuleKey, Locale: "en-US", Group: "permission", Key: cmdbGroupPermImportKey, Value: "Import host groups"},
		{Module: cmdbModuleKey, Locale: "zh-CN", Group: "page", Key: "business.cmdb.label.schema.title", Value: "标签管理"},
		{Module: cmdbModuleKey, Locale: "en-US", Group: "page", Key: "business.cmdb.label.schema.title", Value: "Label Management"},
		{Module: cmdbModuleKey, Locale: "zh-CN", Group: "page", Key: "business.cmdb.label.hero.eyebrow", Value: "运维平台 / 标签管理"},
		{Module: cmdbModuleKey, Locale: "en-US", Group: "page", Key: "business.cmdb.label.hero.eyebrow", Value: "Operations / Label Management"},
		{Module: cmdbModuleKey, Locale: "zh-CN", Group: "page", Key: "business.cmdb.label.hero.title", Value: "按分组治理主机标签键和值模板，确保分组和后续运维目标稳定"},
		{Module: cmdbModuleKey, Locale: "en-US", Group: "page", Key: "business.cmdb.label.hero.title", Value: "Govern host labels by category and presets so groups and operations targets stay reliable"},
		{Module: cmdbModuleKey, Locale: "zh-CN", Group: "permission", Key: cmdbLabelPermCreateKey, Value: "新增标签"},
		{Module: cmdbModuleKey, Locale: "en-US", Group: "permission", Key: cmdbLabelPermCreateKey, Value: "Create labels"},
		{Module: cmdbModuleKey, Locale: "zh-CN", Group: "permission", Key: cmdbLabelPermUpdateKey, Value: "编辑标签"},
		{Module: cmdbModuleKey, Locale: "en-US", Group: "permission", Key: cmdbLabelPermUpdateKey, Value: "Update labels"},
		{Module: cmdbModuleKey, Locale: "zh-CN", Group: "permission", Key: cmdbLabelPermDeleteKey, Value: "删除标签"},
		{Module: cmdbModuleKey, Locale: "en-US", Group: "permission", Key: cmdbLabelPermDeleteKey, Value: "Delete labels"},
		{Module: cmdbModuleKey, Locale: "zh-CN", Group: "permission", Key: cmdbLabelPermExportKey, Value: "导出标签"},
		{Module: cmdbModuleKey, Locale: "en-US", Group: "permission", Key: cmdbLabelPermExportKey, Value: "Export labels"},
		{Module: cmdbModuleKey, Locale: "zh-CN", Group: "permission", Key: cmdbLabelPermImportKey, Value: "导入标签"},
		{Module: cmdbModuleKey, Locale: "en-US", Group: "permission", Key: cmdbLabelPermImportKey, Value: "Import labels"},
		{Module: cmdbModuleKey, Locale: "zh-CN", Group: "error", Key: "cmdblabel.key_exists", Value: "标签键已存在"},
		{Module: cmdbModuleKey, Locale: "en-US", Group: "error", Key: "cmdblabel.key_exists", Value: "Label key already exists"},
		{Module: cmdbModuleKey, Locale: "zh-CN", Group: "error", Key: "cmdblabel.invalid", Value: "标签配置无效"},
		{Module: cmdbModuleKey, Locale: "en-US", Group: "error", Key: "cmdblabel.invalid", Value: "Label configuration is invalid"},
		{Module: cmdbModuleKey, Locale: "zh-CN", Group: "error", Key: "cmdblabel.in_use", Value: "该标签已被主机或分组引用，不能删除"},
		{Module: cmdbModuleKey, Locale: "en-US", Group: "error", Key: "cmdblabel.in_use", Value: "This label is used by hosts or groups and cannot be deleted."},
		{Module: cmdbModuleKey, Locale: "zh-CN", Group: "error", Key: "cmdblabel.not_found", Value: "标签不存在"},
		{Module: cmdbModuleKey, Locale: "en-US", Group: "error", Key: "cmdblabel.not_found", Value: "Label does not exist"},
		{Module: cmdbModuleKey, Locale: "zh-CN", Group: "error", Key: "cmdbgroup.has_children", Value: "该分组存在子分组，不能删除"},
		{Module: cmdbModuleKey, Locale: "en-US", Group: "error", Key: "cmdbgroup.has_children", Value: "Delete subgroups before deleting this group."},
		{Module: cmdbModuleKey, Locale: "zh-CN", Group: "error", Key: "cmdbgroup.parent_cycle", Value: "上级分组不能选择自身或下级分组"},
		{Module: cmdbModuleKey, Locale: "en-US", Group: "error", Key: "cmdbgroup.parent_cycle", Value: "The parent group cannot be itself or a descendant group."},
		{Module: cmdbModuleKey, Locale: "zh-CN", Group: "error", Key: "cmdbgroup.parent_not_found", Value: "上级分组不存在"},
		{Module: cmdbModuleKey, Locale: "en-US", Group: "error", Key: "cmdbgroup.parent_not_found", Value: "Parent group does not exist."},
		{Module: cmdbModuleKey, Locale: "zh-CN", Group: "error", Key: "cmdbgroup.export_failed", Value: "导出主机分组失败"},
		{Module: cmdbModuleKey, Locale: "en-US", Group: "error", Key: "cmdbgroup.export_failed", Value: "Failed to export host groups"},
		{Module: cmdbModuleKey, Locale: "zh-CN", Group: "error", Key: "cmdbgroup.import_failed", Value: "导入主机分组失败"},
		{Module: cmdbModuleKey, Locale: "en-US", Group: "error", Key: "cmdbgroup.import_failed", Value: "Failed to import host groups"},
		{Module: cmdbModuleKey, Locale: "zh-CN", Group: "error", Key: "business.cmdb.group.name_required", Value: "分组名称不能为空"},
		{Module: cmdbModuleKey, Locale: "en-US", Group: "error", Key: "business.cmdb.group.name_required", Value: "Group name is required"},
		{Module: cmdbModuleKey, Locale: "zh-CN", Group: "error", Key: "business.cmdb.group.conditions_invalid_json", Value: "筛选条件不是有效的 JSON"},
		{Module: cmdbModuleKey, Locale: "en-US", Group: "error", Key: "business.cmdb.group.conditions_invalid_json", Value: "Conditions must be valid JSON"},
		{Module: cmdbModuleKey, Locale: "zh-CN", Group: "error", Key: "cmdblabel.key_required", Value: "标签键不能为空"},
		{Module: cmdbModuleKey, Locale: "en-US", Group: "error", Key: "cmdblabel.key_required", Value: "Label key is required"},
		{Module: cmdbModuleKey, Locale: "zh-CN", Group: "error", Key: "cmdblabel.name_required", Value: "标签名称不能为空"},
		{Module: cmdbModuleKey, Locale: "en-US", Group: "error", Key: "cmdblabel.name_required", Value: "Label name is required"},
		{Module: cmdbModuleKey, Locale: "zh-CN", Group: "error", Key: "cmdblabel.category_required", Value: "分类不能为空"},
		{Module: cmdbModuleKey, Locale: "en-US", Group: "error", Key: "cmdblabel.category_required", Value: "Category is required"},
		{Module: cmdbModuleKey, Locale: "zh-CN", Group: "error", Key: "cmdblabel.valueMode_required", Value: "值模式不能为空"},
		{Module: cmdbModuleKey, Locale: "en-US", Group: "error", Key: "cmdblabel.valueMode_required", Value: "Value mode is required"},
		{Module: cmdbModuleKey, Locale: "zh-CN", Group: "error", Key: "cmdblabel.category.invalid", Value: "分类值无效（base/network/business/custom）"},
		{Module: cmdbModuleKey, Locale: "en-US", Group: "error", Key: "cmdblabel.category.invalid", Value: "Invalid category (base/network/business/custom)"},
		{Module: cmdbModuleKey, Locale: "zh-CN", Group: "error", Key: "cmdblabel.valueMode.invalid", Value: "值模式无效（free/enum/dict）"},
		{Module: cmdbModuleKey, Locale: "en-US", Group: "error", Key: "cmdblabel.valueMode.invalid", Value: "Invalid value mode (free/enum/dict)"},
		{Module: cmdbModuleKey, Locale: "zh-CN", Group: "error", Key: "cmdblabel.status.invalid", Value: "状态值无效（enabled/disabled）"},
		{Module: cmdbModuleKey, Locale: "en-US", Group: "error", Key: "cmdblabel.status.invalid", Value: "Invalid status (enabled/disabled)"},
		{Module: cmdbModuleKey, Locale: "zh-CN", Group: "error", Key: "cmdblabel.options_required_for_enum", Value: "枚举模式下选项不能为空"},
		{Module: cmdbModuleKey, Locale: "en-US", Group: "error", Key: "cmdblabel.options_required_for_enum", Value: "Options are required for enum mode"},
		{Module: cmdbModuleKey, Locale: "zh-CN", Group: "error", Key: "cmdblabel.dictCode_required_for_dict", Value: "字典模式下字典编码不能为空"},
		{Module: cmdbModuleKey, Locale: "en-US", Group: "error", Key: "cmdblabel.dictCode_required_for_dict", Value: "Dict code is required for dict mode"},
		{Module: cmdbModuleKey, Locale: "zh-CN", Group: "error", Key: "cmdblabel.export_failed", Value: "导出标签失败"},
		{Module: cmdbModuleKey, Locale: "en-US", Group: "error", Key: "cmdblabel.export_failed", Value: "Failed to export labels"},
		{Module: cmdbModuleKey, Locale: "zh-CN", Group: "error", Key: "cmdblabel.import_failed", Value: "导入标签失败"},
		{Module: cmdbModuleKey, Locale: "en-US", Group: "error", Key: "cmdblabel.import_failed", Value: "Failed to import labels"},
		{Module: cmdbModuleKey, Locale: "zh-CN", Group: "error", Key: "cmdbhost.hostname_required", Value: "主机名不能为空"},
		{Module: cmdbModuleKey, Locale: "en-US", Group: "error", Key: "cmdbhost.hostname_required", Value: "Hostname is required"},
		{Module: cmdbModuleKey, Locale: "zh-CN", Group: "error", Key: "cmdbhost.ip_required", Value: "IP 地址不能为空"},
		{Module: cmdbModuleKey, Locale: "en-US", Group: "error", Key: "cmdbhost.ip_required", Value: "IP address is required"},
		{Module: cmdbModuleKey, Locale: "zh-CN", Group: "error", Key: "cmdbhost.os_required", Value: "操作系统不能为空"},
		{Module: cmdbModuleKey, Locale: "en-US", Group: "error", Key: "cmdbhost.os_required", Value: "Operating system is required"},
		{Module: cmdbModuleKey, Locale: "zh-CN", Group: "error", Key: "cmdbhost.status.invalid", Value: "主机状态无效"},
		{Module: cmdbModuleKey, Locale: "en-US", Group: "error", Key: "cmdbhost.status.invalid", Value: "Host status is invalid"},
		{Module: cmdbModuleKey, Locale: "zh-CN", Group: "error", Key: "cmdbhost.export_failed", Value: "导出主机资产失败"},
		{Module: cmdbModuleKey, Locale: "en-US", Group: "error", Key: "cmdbhost.export_failed", Value: "Failed to export host assets"},
		{Module: cmdbModuleKey, Locale: "zh-CN", Group: "error", Key: "cmdbhost.import_failed", Value: "导入主机失败"},
		{Module: cmdbModuleKey, Locale: "en-US", Group: "error", Key: "cmdbhost.import_failed", Value: "Failed to import hosts"},
	}
	return seedCmdbRecords(db, i18nEntries)
}

func seedGroupI18n(db *gorm.DB) error { return nil }

func seedCmdbDicts(db *gorm.DB) error {
	if db == nil {
		return nil
	}
	dictTypes := []map[string]interface{}{
		{"dict_code": "cmdb_host_status", "dict_name": "主机状态", "module": cmdbModuleKey, "status": 1, "created_at": time.Now(), "updated_at": time.Now()},
		{"dict_code": "cmdb_os_type", "dict_name": "操作系统类型", "module": cmdbModuleKey, "status": 1, "created_at": time.Now(), "updated_at": time.Now()},
		{"dict_code": "cmdb_label_key", "dict_name": "预置标签键", "module": cmdbModuleKey, "status": 1, "created_at": time.Now(), "updated_at": time.Now()},
		{"dict_code": "cmdb_env", "dict_name": "CMDB 环境", "module": cmdbModuleKey, "status": 1, "created_at": time.Now(), "updated_at": time.Now()},
	}
	for _, dt := range dictTypes {
		var count int64
		db.Table("system_dict_type").Where("dict_code = ?", dt["dict_code"]).Count(&count)
		if count == 0 {
			if err := db.Table("system_dict_type").Create(dt).Error; err != nil {
				return err
			}
		}
	}
	dictItems := []map[string]interface{}{
		{"dict_code": "cmdb_host_status", "item_label_key": "待上线", "item_value": "pending", "sort": 1, "status": 1, "created_at": time.Now(), "updated_at": time.Now()},
		{"dict_code": "cmdb_host_status", "item_label_key": "已分配", "item_value": "assigned", "sort": 2, "status": 1, "created_at": time.Now(), "updated_at": time.Now()},
		{"dict_code": "cmdb_host_status", "item_label_key": "可运维", "item_value": "online", "sort": 3, "status": 1, "created_at": time.Now(), "updated_at": time.Now()},
		{"dict_code": "cmdb_host_status", "item_label_key": "已下线", "item_value": "offline", "sort": 4, "status": 1, "created_at": time.Now(), "updated_at": time.Now()},
		{"dict_code": "cmdb_host_status", "item_label_key": "维护中", "item_value": "maintenance", "sort": 5, "status": 1, "created_at": time.Now(), "updated_at": time.Now()},
		{"dict_code": "cmdb_os_type", "item_label_key": "Linux", "item_value": "linux", "sort": 1, "status": 1, "created_at": time.Now(), "updated_at": time.Now()},
		{"dict_code": "cmdb_os_type", "item_label_key": "Windows", "item_value": "windows", "sort": 2, "status": 1, "created_at": time.Now(), "updated_at": time.Now()},
		{"dict_code": "cmdb_label_key", "item_label_key": "环境", "item_value": "env", "sort": 1, "status": 1, "created_at": time.Now(), "updated_at": time.Now()},
		{"dict_code": "cmdb_label_key", "item_label_key": "业务系统", "item_value": "biz", "sort": 2, "status": 1, "created_at": time.Now(), "updated_at": time.Now()},
		{"dict_code": "cmdb_label_key", "item_label_key": "集群", "item_value": "cluster", "sort": 3, "status": 1, "created_at": time.Now(), "updated_at": time.Now()},
		{"dict_code": "cmdb_label_key", "item_label_key": "区域", "item_value": "region", "sort": 4, "status": 1, "created_at": time.Now(), "updated_at": time.Now()},
		{"dict_code": "cmdb_label_key", "item_label_key": "数据库类型", "item_value": "db_type", "sort": 5, "status": 1, "created_at": time.Now(), "updated_at": time.Now()},
		{"dict_code": "cmdb_label_key", "item_label_key": "中间件", "item_value": "middleware", "sort": 6, "status": 1, "created_at": time.Now(), "updated_at": time.Now()},
		{"dict_code": "cmdb_env", "item_label_key": "开发", "item_value": "dev", "sort": 1, "status": 1, "created_at": time.Now(), "updated_at": time.Now()},
		{"dict_code": "cmdb_env", "item_label_key": "测试", "item_value": "test", "sort": 2, "status": 1, "created_at": time.Now(), "updated_at": time.Now()},
		{"dict_code": "cmdb_env", "item_label_key": "生产", "item_value": "prod", "sort": 3, "status": 1, "created_at": time.Now(), "updated_at": time.Now()},
	}
	for _, di := range dictItems {
		var existingID uint64
		db.Table("system_dict_item").Select("id").Where("dict_code = ? AND item_value = ?", di["dict_code"], di["item_value"]).Limit(1).Pluck("id", &existingID)
		if existingID == 0 {
			if err := db.Table("system_dict_item").Create(di).Error; err != nil {
				return err
			}
		} else if err := db.Table("system_dict_item").Where(idWhereClause, existingID).Updates(map[string]interface{}{
			"item_label_key": di["item_label_key"],
			"sort":           di["sort"],
			"status":         di["status"],
			"updated_at":     time.Now(),
		}).Error; err != nil {
			return err
		}
	}
	return nil
}

func seedCmdbLabelSchemas(db *gorm.DB) error {
	if db == nil || !db.Migrator().HasTable("biz_cmdb_label_schema") {
		return nil
	}
	if err := db.Table("biz_cmdb_label_schema").
		Where("category IS NULL OR TRIM(category) = ''").
		Updates(map[string]interface{}{
			"category":   "base",
			"updated_at": time.Now(),
		}).Error; err != nil {
		return err
	}
	schemas := []map[string]interface{}{
		{"key": "env", "name": "环境", "category": "environment", "value_mode": "dict", "dict_code": "cmdb_env", "options": datatypes.JSON([]byte(`["dev","test","prod"]`)), "required": false, "status": "enabled", "description": "部署环境，如 dev/test/prod", "created_at": time.Now(), "updated_at": time.Now()},
		{"key": "biz", "name": "业务系统", "category": "business", "value_mode": "enum", "dict_code": "", "options": datatypes.JSON([]byte(`["order-center","user-center","ops-platform"]`)), "required": false, "status": "enabled", "description": "业务系统或应用归属", "created_at": time.Now(), "updated_at": time.Now()},
		{"key": "cluster", "name": "集群", "category": "topology", "value_mode": "free", "dict_code": "", "options": datatypes.JSON([]byte(`[]`)), "required": false, "status": "enabled", "description": "集群或资源池归属", "created_at": time.Now(), "updated_at": time.Now()},
		{"key": "region", "name": "区域", "category": "topology", "value_mode": "enum", "dict_code": "", "options": datatypes.JSON([]byte(`["cn-east-1","cn-north-1","edge-idc-a"]`)), "required": false, "status": "enabled", "description": "区域、机房或可用区", "created_at": time.Now(), "updated_at": time.Now()},
		{"key": "db_type", "name": "数据库类型", "category": "database", "value_mode": "enum", "dict_code": "", "options": datatypes.JSON([]byte(`["mysql","postgresql"]`)), "required": false, "status": "enabled", "description": "主机承载的数据库类型", "created_at": time.Now(), "updated_at": time.Now()},
		{"key": "middleware", "name": "中间件", "category": "middleware", "value_mode": "enum", "dict_code": "", "options": datatypes.JSON([]byte(`["nginx","redis","minio","harbor"]`)), "required": false, "status": "enabled", "description": "主机承载的中间件类型", "created_at": time.Now(), "updated_at": time.Now()},
	}
	for _, schema := range schemas {
		var existingID uint64
		db.Table("biz_cmdb_label_schema").Select("id").Where("`key` = ?", schema["key"]).Limit(1).Pluck("id", &existingID)
		if existingID == 0 {
			if err := db.Table("biz_cmdb_label_schema").Create(schema).Error; err != nil {
				return err
			}
		} else if err := db.Table("biz_cmdb_label_schema").Where(idWhereClause, existingID).Updates(map[string]interface{}{
			"name":        schema["name"],
			"category":    schema["category"],
			"value_mode":  schema["value_mode"],
			"dict_code":   schema["dict_code"],
			"options":     schema["options"],
			"status":      schema["status"],
			"description": schema["description"],
			"updated_at":  time.Now(),
		}).Error; err != nil {
			return err
		}
	}
	return nil
}

func seedCmdbRecords(db *gorm.DB, records []i18nSeed) error {
	for _, seed := range records {
		var existingID uint64
		if err := db.Table("system_i18n").Select("id").Where("`key` = ? AND locale = ?",
			seed.Key, seed.Locale).Limit(1).Pluck("id", &existingID).Error; err != nil {
			return err
		}
		if existingID > 0 {
			update := map[string]interface{}{
				"value":      seed.Value,
				"module":     seed.Module,
				"updated_at": time.Now(),
			}
			if err := db.Table("system_i18n").Where(idWhereClause, existingID).Updates(update).Error; err != nil {
				return err
			}
			continue
		}
		payload := map[string]interface{}{
			"module":           seed.Module,
			"group_name":       seed.Group,
			"key":              seed.Key,
			"locale":           seed.Locale,
			"value":            seed.Value,
			"lifecycle_status": "active",
			"created_at":       time.Now(),
			"updated_at":       time.Now(),
		}
		if err := db.Table("system_i18n").Create(payload).Error; err != nil {
			return err
		}
	}
	return nil
}
