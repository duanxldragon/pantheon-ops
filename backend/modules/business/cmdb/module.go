package cmdb

import (
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
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

func hostMenuSeeds() []cmdbMenuSeed {
	return []cmdbMenuSeed{
		{
			Key:       "operations-cmdb-host",
			ParentKey: "cmdb",
			TitleKey:  "operations.cmdb.host.menu",
			Path:      "/operations/cmdb/host",
			Component: "business/cmdb/host/CmdbHostList",
			PagePerm:  "business:cmdb:host:list",
			Type:      "C",
			Module:    "business.cmdb",
			RouteName: "cmdb-host-list",
			Sort:      1,
		},
		{Key: "operations-cmdb-host-detail", ParentKey: "cmdb-host-list", TitleKey: "business.cmdb.host.permission.detail", Perms: "business:cmdb:host:detail", Type: "F", Module: "business.cmdb", Sort: 1},
		{Key: "operations-cmdb-host-create", ParentKey: "cmdb-host-list", TitleKey: "business.cmdb.host.permission.create", Perms: "business:cmdb:host:create", Type: "F", Module: "business.cmdb", Sort: 2},
		{Key: "operations-cmdb-host-update", ParentKey: "cmdb-host-list", TitleKey: "business.cmdb.host.permission.update", Perms: "business:cmdb:host:update", Type: "F", Module: "business.cmdb", Sort: 3},
		{Key: "operations-cmdb-host-delete", ParentKey: "cmdb-host-list", TitleKey: "business.cmdb.host.permission.delete", Perms: "business:cmdb:host:delete", Type: "F", Module: "business.cmdb", Sort: 4},
		{Key: "operations-cmdb-host-collect", ParentKey: "cmdb-host-list", TitleKey: "business.cmdb.host.permission.collect", Perms: "business:cmdb:host:collect", Type: "F", Module: "business.cmdb", Sort: 5},
		{Key: "operations-cmdb-host-status", ParentKey: "cmdb-host-list", TitleKey: "business.cmdb.host.permission.status", Perms: "business:cmdb:host:status", Type: "F", Module: "business.cmdb", Sort: 6},
		{
			Key:       "operations-cmdb-group",
			ParentKey: "cmdb",
			TitleKey:  "operations.cmdb.group.menu",
			Path:      "/operations/cmdb/group",
			Component: "business/cmdb/group/CmdbGroupList",
			PagePerm:  "business:cmdb:group:list",
			Type:      "C",
			Module:    "business.cmdb",
			RouteName: "cmdb-group-list",
			Sort:      2,
		},
		{Key: "operations-cmdb-group-detail", ParentKey: "cmdb-group-list", TitleKey: "business.cmdb.group.permission.detail", Perms: "business:cmdb:group:detail", Type: "F", Module: "business.cmdb", Sort: 1},
		{Key: "operations-cmdb-group-create", ParentKey: "cmdb-group-list", TitleKey: "business.cmdb.group.permission.create", Perms: "business:cmdb:group:create", Type: "F", Module: "business.cmdb", Sort: 2},
		{Key: "operations-cmdb-group-update", ParentKey: "cmdb-group-list", TitleKey: "business.cmdb.group.permission.update", Perms: "business:cmdb:group:update", Type: "F", Module: "business.cmdb", Sort: 3},
		{Key: "operations-cmdb-group-delete", ParentKey: "cmdb-group-list", TitleKey: "business.cmdb.group.permission.delete", Perms: "business:cmdb:group:delete", Type: "F", Module: "business.cmdb", Sort: 4},
		{
			Key:       "operations-cmdb-label",
			ParentKey: "cmdb",
			TitleKey:  "operations.cmdb.label.menu",
			Path:      "/operations/cmdb/label",
			Component: "business/cmdb/label/CmdbLabelSchemaList",
			PagePerm:  "business:cmdb:label:list",
			Type:      "C",
			Module:    "business.cmdb",
			RouteName: "cmdb-label-list",
			Sort:      3,
		},
		{Key: "operations-cmdb-label-create", ParentKey: "cmdb-label-list", TitleKey: "business.cmdb.label.permission.create", Perms: "business:cmdb:label:create", Type: "F", Module: "business.cmdb", Sort: 1},
		{Key: "operations-cmdb-label-update", ParentKey: "cmdb-label-list", TitleKey: "business.cmdb.label.permission.update", Perms: "business:cmdb:label:update", Type: "F", Module: "business.cmdb", Sort: 2},
		{Key: "operations-cmdb-label-delete", ParentKey: "cmdb-label-list", TitleKey: "business.cmdb.label.permission.delete", Perms: "business:cmdb:label:delete", Type: "F", Module: "business.cmdb", Sort: 3},
	}
}

func topLevelMenuSeeds() []cmdbMenuSeed {
	return []cmdbMenuSeed{
		{
			Key:       "operations",
			TitleKey:  "operations.menu",
			Path:      "/operations",
			Type:      "D",
			Module:    "business.cmdb",
			Icon:      "desktop",
			RouteName: "operations",
			Sort:      1,
		},
		{
			Key:       "operations-cmdb",
			ParentKey: "operations",
			TitleKey:  "operations.cmdb.menu",
			Path:      "/operations/cmdb",
			Type:      "M",
			Module:    "business.cmdb",
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
			if err := db.Table("system_menu").Where("id = ?", menuID).Updates(updates).Error; err != nil {
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
	i18nEntries := []map[string]interface{}{
		{"module": "business.cmdb", "locale": "zh-CN", "group_name": "menu", "key": "operations.menu", "value": "运维平台", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.cmdb", "locale": "en-US", "group_name": "menu", "key": "operations.menu", "value": "Operations", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.cmdb", "locale": "zh-CN", "group_name": "menu", "key": "operations.cmdb.menu", "value": "CMDB", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.cmdb", "locale": "en-US", "group_name": "menu", "key": "operations.cmdb.menu", "value": "CMDB", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.cmdb", "locale": "zh-CN", "group_name": "menu", "key": "operations.cmdb.host.menu", "value": "主机管理", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.cmdb", "locale": "en-US", "group_name": "menu", "key": "operations.cmdb.host.menu", "value": "Host Management", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.cmdb", "locale": "zh-CN", "group_name": "page", "key": "operations.cmdb.host.detail", "value": "主机详情", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.cmdb", "locale": "en-US", "group_name": "page", "key": "operations.cmdb.host.detail", "value": "Host Detail", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.cmdb", "locale": "zh-CN", "group_name": "menu", "key": "operations.cmdb.group.menu", "value": "主机分组", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.cmdb", "locale": "en-US", "group_name": "menu", "key": "operations.cmdb.group.menu", "value": "Host Groups", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.cmdb", "locale": "zh-CN", "group_name": "menu", "key": "operations.cmdb.label.menu", "value": "标签规范", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.cmdb", "locale": "en-US", "group_name": "menu", "key": "operations.cmdb.label.menu", "value": "Label Schema", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.cmdb", "locale": "zh-CN", "group_name": "page", "key": "business.cmdb.host.title", "value": "主机管理", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.cmdb", "locale": "en-US", "group_name": "page", "key": "business.cmdb.host.title", "value": "Host Management", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.cmdb", "locale": "zh-CN", "group_name": "page", "key": "business.cmdb.host.hero.eyebrow", "value": "运维平台 / 主机台账", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.cmdb", "locale": "en-US", "group_name": "page", "key": "business.cmdb.host.hero.eyebrow", "value": "Operations / Host Inventory", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.cmdb", "locale": "zh-CN", "group_name": "page", "key": "business.cmdb.host.hero.title", "value": "在统一视图中管理主机、标签与配置采集", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.cmdb", "locale": "en-US", "group_name": "page", "key": "business.cmdb.host.hero.title", "value": "Manage hosts, labels, and collection in one view", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.cmdb", "locale": "zh-CN", "group_name": "page", "key": "business.cmdb.host.hero.total", "value": "主机总数", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.cmdb", "locale": "en-US", "group_name": "page", "key": "business.cmdb.host.hero.total", "value": "Total Hosts", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.cmdb", "locale": "zh-CN", "group_name": "page", "key": "business.cmdb.host.hero.totalHint", "value": "当前筛选条件下的主机总量。", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.cmdb", "locale": "en-US", "group_name": "page", "key": "business.cmdb.host.hero.totalHint", "value": "Total hosts under the current filter.", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.cmdb", "locale": "zh-CN", "group_name": "page", "key": "business.cmdb.host.hero.online", "value": "可运维主机", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.cmdb", "locale": "en-US", "group_name": "page", "key": "business.cmdb.host.hero.online", "value": "Operable Hosts", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.cmdb", "locale": "zh-CN", "group_name": "page", "key": "business.cmdb.host.hero.onlineHint", "value": "运维状态为可运维的主机数量。", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.cmdb", "locale": "en-US", "group_name": "page", "key": "business.cmdb.host.hero.onlineHint", "value": "Hosts whose operations status is operable.", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.cmdb", "locale": "zh-CN", "group_name": "page", "key": "business.cmdb.host.hero.maintenance", "value": "维护中", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.cmdb", "locale": "en-US", "group_name": "page", "key": "business.cmdb.host.hero.maintenance", "value": "Under Maintenance", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.cmdb", "locale": "zh-CN", "group_name": "page", "key": "business.cmdb.host.hero.maintenanceHint", "value": "状态为维护中的主机数量。", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.cmdb", "locale": "en-US", "group_name": "page", "key": "business.cmdb.host.hero.maintenanceHint", "value": "Hosts whose status is maintenance.", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.cmdb", "locale": "zh-CN", "group_name": "page", "key": "business.cmdb.host.hero.scope", "value": "数据范围", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.cmdb", "locale": "en-US", "group_name": "page", "key": "business.cmdb.host.hero.scope", "value": "Data Scope", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.cmdb", "locale": "zh-CN", "group_name": "page", "key": "business.cmdb.host.hero.scopeValue", "value": "按当前登录主体可见数据", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.cmdb", "locale": "en-US", "group_name": "page", "key": "business.cmdb.host.hero.scopeValue", "value": "Visible to the current login context", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.cmdb", "locale": "zh-CN", "group_name": "page", "key": "business.cmdb.host.hero.scopeHint", "value": "主机列表和详情遵循系统域数据范围。", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.cmdb", "locale": "en-US", "group_name": "page", "key": "business.cmdb.host.hero.scopeHint", "value": "Host lists and details follow the system data scope.", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.cmdb", "locale": "zh-CN", "group_name": "page", "key": "business.cmdb.host.hero.statusHint", "value": "当前主机运维状态，不代表实时连通性。", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.cmdb", "locale": "en-US", "group_name": "page", "key": "business.cmdb.host.hero.statusHint", "value": "Current operations status, not real-time connectivity.", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.cmdb", "locale": "zh-CN", "group_name": "page", "key": "business.cmdb.host.hero.osHint", "value": "当前操作系统类型。", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.cmdb", "locale": "en-US", "group_name": "page", "key": "business.cmdb.host.hero.osHint", "value": "Current operating system type.", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.cmdb", "locale": "zh-CN", "group_name": "page", "key": "business.cmdb.host.hero.osSummary", "value": "操作系统分布", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.cmdb", "locale": "en-US", "group_name": "page", "key": "business.cmdb.host.hero.osSummary", "value": "OS Distribution", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.cmdb", "locale": "zh-CN", "group_name": "page", "key": "business.cmdb.host.hero.osSummaryHint", "value": "当前页主机的 Linux / Windows 数量。", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.cmdb", "locale": "en-US", "group_name": "page", "key": "business.cmdb.host.hero.osSummaryHint", "value": "Linux / Windows count on the current page.", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.cmdb", "locale": "zh-CN", "group_name": "page", "key": "business.cmdb.host.hero.labelsHint", "value": "当前主机标签数量。", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.cmdb", "locale": "en-US", "group_name": "page", "key": "business.cmdb.host.hero.labelsHint", "value": "Current host label count.", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.cmdb", "locale": "zh-CN", "group_name": "page", "key": "business.cmdb.host.hero.componentsHint", "value": "当前主机已装组件数量。", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.cmdb", "locale": "en-US", "group_name": "page", "key": "business.cmdb.host.hero.componentsHint", "value": "Installed component count on the current host.", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.cmdb", "locale": "zh-CN", "group_name": "page", "key": "business.cmdb.host.baseInfo", "value": "基础信息", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.cmdb", "locale": "en-US", "group_name": "page", "key": "business.cmdb.host.baseInfo", "value": "Base Information", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.cmdb", "locale": "zh-CN", "group_name": "page", "key": "business.cmdb.host.systemConfig", "value": "系统配置", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.cmdb", "locale": "en-US", "group_name": "page", "key": "business.cmdb.host.systemConfig", "value": "System Configuration", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.cmdb", "locale": "zh-CN", "group_name": "page", "key": "business.cmdb.host.labelsEmpty", "value": "暂无标签", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.cmdb", "locale": "en-US", "group_name": "page", "key": "business.cmdb.host.labelsEmpty", "value": "No labels", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.cmdb", "locale": "zh-CN", "group_name": "page", "key": "business.cmdb.host.componentsEmpty", "value": "暂无已装组件", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.cmdb", "locale": "en-US", "group_name": "page", "key": "business.cmdb.host.componentsEmpty", "value": "No installed components", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.cmdb", "locale": "zh-CN", "group_name": "page", "key": "business.cmdb.host.collectSshUserPlaceholder", "value": "root", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.cmdb", "locale": "en-US", "group_name": "page", "key": "business.cmdb.host.collectSshUserPlaceholder", "value": "root", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.cmdb", "locale": "zh-CN", "group_name": "page", "key": "business.cmdb.host.collectPrivateKeyPlaceholder", "value": "-----BEGIN OPENSSH PRIVATE KEY-----", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.cmdb", "locale": "en-US", "group_name": "page", "key": "business.cmdb.host.collectPrivateKeyPlaceholder", "value": "-----BEGIN OPENSSH PRIVATE KEY-----", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.cmdb", "locale": "zh-CN", "group_name": "menu", "key": "business.cmdb.host.os.linux", "value": "Linux", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.cmdb", "locale": "en-US", "group_name": "menu", "key": "business.cmdb.host.os.linux", "value": "Linux", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.cmdb", "locale": "zh-CN", "group_name": "menu", "key": "business.cmdb.host.os.windows", "value": "Windows", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.cmdb", "locale": "en-US", "group_name": "menu", "key": "business.cmdb.host.os.windows", "value": "Windows", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.cmdb", "locale": "zh-CN", "group_name": "menu", "key": "business.cmdb.host.status.maintenance", "value": "维护中", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.cmdb", "locale": "en-US", "group_name": "menu", "key": "business.cmdb.host.status.maintenance", "value": "Maintenance", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.cmdb", "locale": "zh-CN", "group_name": "page", "key": "business.cmdb.group.hero.eyebrow", "value": "运维平台 / 主机分组", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.cmdb", "locale": "en-US", "group_name": "page", "key": "business.cmdb.group.hero.eyebrow", "value": "Operations / Host Groups", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.cmdb", "locale": "zh-CN", "group_name": "page", "key": "business.cmdb.group.hero.title", "value": "通过标签条件管理可复用的主机集合", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.cmdb", "locale": "en-US", "group_name": "page", "key": "business.cmdb.group.hero.title", "value": "Manage reusable host sets with label conditions", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.cmdb", "locale": "zh-CN", "group_name": "page", "key": "business.cmdb.group.hero.total", "value": "分组总数", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.cmdb", "locale": "en-US", "group_name": "page", "key": "business.cmdb.group.hero.total", "value": "Total Groups", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.cmdb", "locale": "zh-CN", "group_name": "page", "key": "business.cmdb.group.hero.totalHint", "value": "当前可见的主机分组数量。", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.cmdb", "locale": "en-US", "group_name": "page", "key": "business.cmdb.group.hero.totalHint", "value": "All visible host groups under the current scope.", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.cmdb", "locale": "zh-CN", "group_name": "page", "key": "business.cmdb.group.hero.members", "value": "选中分组成员", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.cmdb", "locale": "en-US", "group_name": "page", "key": "business.cmdb.group.hero.members", "value": "Selected Group Members", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.cmdb", "locale": "zh-CN", "group_name": "page", "key": "business.cmdb.group.hero.membersHint", "value": "当前选中分组的成员数量。", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.cmdb", "locale": "en-US", "group_name": "page", "key": "business.cmdb.group.hero.membersHint", "value": "Member count of the selected group.", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.cmdb", "locale": "zh-CN", "group_name": "page", "key": "business.cmdb.group.hero.scope", "value": "数据范围", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.cmdb", "locale": "en-US", "group_name": "page", "key": "business.cmdb.group.hero.scope", "value": "Data Scope", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.cmdb", "locale": "zh-CN", "group_name": "page", "key": "business.cmdb.group.hero.scopeValue", "value": "按当前登录主体可见数据", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.cmdb", "locale": "en-US", "group_name": "page", "key": "business.cmdb.group.hero.scopeValue", "value": "Visible to the current login context", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.cmdb", "locale": "zh-CN", "group_name": "page", "key": "business.cmdb.group.hero.scopeHint", "value": "成员计算遵循当前请求的数据范围。", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.cmdb", "locale": "en-US", "group_name": "page", "key": "business.cmdb.group.hero.scopeHint", "value": "Member computation follows the current request scope.", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.cmdb", "locale": "zh-CN", "group_name": "page", "key": "business.cmdb.group.hero.rules", "value": "筛选规则", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.cmdb", "locale": "en-US", "group_name": "page", "key": "business.cmdb.group.hero.rules", "value": "Filter Rules", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.cmdb", "locale": "zh-CN", "group_name": "page", "key": "business.cmdb.group.hero.rulesHint", "value": "当前选中分组的规则条数。", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.cmdb", "locale": "en-US", "group_name": "page", "key": "business.cmdb.group.hero.rulesHint", "value": "Rule count of the selected group.", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.cmdb", "locale": "zh-CN", "group_name": "page", "key": "business.cmdb.group.tree.title", "value": "分组树", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.cmdb", "locale": "en-US", "group_name": "page", "key": "business.cmdb.group.tree.title", "value": "Group Tree", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.cmdb", "locale": "zh-CN", "group_name": "page", "key": "business.cmdb.group.parent", "value": "上级分组", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.cmdb", "locale": "en-US", "group_name": "page", "key": "business.cmdb.group.parent", "value": "Parent Group", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.cmdb", "locale": "zh-CN", "group_name": "page", "key": "business.cmdb.group.noParent", "value": "无上级分组", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.cmdb", "locale": "en-US", "group_name": "page", "key": "business.cmdb.group.noParent", "value": "No Parent Group", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.cmdb", "locale": "zh-CN", "group_name": "page", "key": "business.cmdb.group.createChild", "value": "新增子分组", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.cmdb", "locale": "en-US", "group_name": "page", "key": "business.cmdb.group.createChild", "value": "Add Subgroup", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.cmdb", "locale": "zh-CN", "group_name": "page", "key": "business.cmdb.group.childCount", "value": "子分组", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.cmdb", "locale": "en-US", "group_name": "page", "key": "business.cmdb.group.childCount", "value": "Subgroups", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.cmdb", "locale": "zh-CN", "group_name": "page", "key": "business.cmdb.group.condition.ruleIndex", "value": "条件 {{count}}", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.cmdb", "locale": "en-US", "group_name": "page", "key": "business.cmdb.group.condition.ruleIndex", "value": "Condition {{count}}", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.cmdb", "locale": "zh-CN", "group_name": "permission", "key": "business.cmdb.host.permission.detail", "value": "查看主机详情", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.cmdb", "locale": "en-US", "group_name": "permission", "key": "business.cmdb.host.permission.detail", "value": "View host detail", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.cmdb", "locale": "zh-CN", "group_name": "permission", "key": "business.cmdb.host.permission.create", "value": "新增主机", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.cmdb", "locale": "en-US", "group_name": "permission", "key": "business.cmdb.host.permission.create", "value": "Create hosts", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.cmdb", "locale": "zh-CN", "group_name": "permission", "key": "business.cmdb.host.permission.update", "value": "编辑主机", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.cmdb", "locale": "en-US", "group_name": "permission", "key": "business.cmdb.host.permission.update", "value": "Update hosts", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.cmdb", "locale": "zh-CN", "group_name": "permission", "key": "business.cmdb.host.permission.delete", "value": "删除主机", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.cmdb", "locale": "en-US", "group_name": "permission", "key": "business.cmdb.host.permission.delete", "value": "Delete hosts", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.cmdb", "locale": "zh-CN", "group_name": "permission", "key": "business.cmdb.host.permission.collect", "value": "采集主机配置", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.cmdb", "locale": "en-US", "group_name": "permission", "key": "business.cmdb.host.permission.collect", "value": "Collect host config", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.cmdb", "locale": "zh-CN", "group_name": "permission", "key": "business.cmdb.host.permission.status", "value": "更新主机状态", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.cmdb", "locale": "en-US", "group_name": "permission", "key": "business.cmdb.host.permission.status", "value": "Update host status", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.cmdb", "locale": "zh-CN", "group_name": "permission", "key": "business.cmdb.group.permission.detail", "value": "查看主机分组", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.cmdb", "locale": "en-US", "group_name": "permission", "key": "business.cmdb.group.permission.detail", "value": "View host groups", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.cmdb", "locale": "zh-CN", "group_name": "permission", "key": "business.cmdb.group.permission.create", "value": "新增主机分组", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.cmdb", "locale": "en-US", "group_name": "permission", "key": "business.cmdb.group.permission.create", "value": "Create host groups", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.cmdb", "locale": "zh-CN", "group_name": "permission", "key": "business.cmdb.group.permission.update", "value": "编辑主机分组", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.cmdb", "locale": "en-US", "group_name": "permission", "key": "business.cmdb.group.permission.update", "value": "Update host groups", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.cmdb", "locale": "zh-CN", "group_name": "permission", "key": "business.cmdb.group.permission.delete", "value": "删除主机分组", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.cmdb", "locale": "en-US", "group_name": "permission", "key": "business.cmdb.group.permission.delete", "value": "Delete host groups", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.cmdb", "locale": "zh-CN", "group_name": "page", "key": "business.cmdb.label.schema.title", "value": "标签规范", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.cmdb", "locale": "en-US", "group_name": "page", "key": "business.cmdb.label.schema.title", "value": "Label Schema", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.cmdb", "locale": "zh-CN", "group_name": "page", "key": "business.cmdb.label.hero.eyebrow", "value": "运维平台 / 标签规范", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.cmdb", "locale": "en-US", "group_name": "page", "key": "business.cmdb.label.hero.eyebrow", "value": "Operations / Label Schema", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.cmdb", "locale": "zh-CN", "group_name": "page", "key": "business.cmdb.label.hero.title", "value": "治理主机标签键，确保分组和后续运维目标稳定", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.cmdb", "locale": "en-US", "group_name": "page", "key": "business.cmdb.label.hero.title", "value": "Govern host label keys so groups and operations targets stay reliable", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.cmdb", "locale": "zh-CN", "group_name": "permission", "key": "business.cmdb.label.permission.create", "value": "新增标签规范", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.cmdb", "locale": "en-US", "group_name": "permission", "key": "business.cmdb.label.permission.create", "value": "Create label schemas", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.cmdb", "locale": "zh-CN", "group_name": "permission", "key": "business.cmdb.label.permission.update", "value": "编辑标签规范", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.cmdb", "locale": "en-US", "group_name": "permission", "key": "business.cmdb.label.permission.update", "value": "Update label schemas", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.cmdb", "locale": "zh-CN", "group_name": "permission", "key": "business.cmdb.label.permission.delete", "value": "删除标签规范", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.cmdb", "locale": "en-US", "group_name": "permission", "key": "business.cmdb.label.permission.delete", "value": "Delete label schemas", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.cmdb", "locale": "zh-CN", "group_name": "error", "key": "cmdblabel.key_exists", "value": "标签键已存在", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.cmdb", "locale": "en-US", "group_name": "error", "key": "cmdblabel.key_exists", "value": "Label key already exists", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.cmdb", "locale": "zh-CN", "group_name": "error", "key": "cmdblabel.invalid", "value": "标签规范配置无效", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.cmdb", "locale": "en-US", "group_name": "error", "key": "cmdblabel.invalid", "value": "Label schema configuration is invalid", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.cmdb", "locale": "zh-CN", "group_name": "error", "key": "cmdblabel.in_use", "value": "该标签已被主机或分组引用，不能删除", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.cmdb", "locale": "en-US", "group_name": "error", "key": "cmdblabel.in_use", "value": "This label is used by hosts or groups and cannot be deleted.", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.cmdb", "locale": "zh-CN", "group_name": "error", "key": "cmdblabel.not_found", "value": "标签规范不存在", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.cmdb", "locale": "en-US", "group_name": "error", "key": "cmdblabel.not_found", "value": "Label schema does not exist", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.cmdb", "locale": "zh-CN", "group_name": "error", "key": "cmdbgroup.has_children", "value": "该分组存在子分组，不能删除", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.cmdb", "locale": "en-US", "group_name": "error", "key": "cmdbgroup.has_children", "value": "Delete subgroups before deleting this group.", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.cmdb", "locale": "zh-CN", "group_name": "error", "key": "cmdbgroup.parent_cycle", "value": "上级分组不能选择自身或下级分组", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.cmdb", "locale": "en-US", "group_name": "error", "key": "cmdbgroup.parent_cycle", "value": "The parent group cannot be itself or a descendant group.", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.cmdb", "locale": "zh-CN", "group_name": "error", "key": "cmdbgroup.parent_not_found", "value": "上级分组不存在", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
		{"module": "business.cmdb", "locale": "en-US", "group_name": "error", "key": "cmdbgroup.parent_not_found", "value": "Parent group does not exist.", "lifecycle_status": "active", "created_at": time.Now(), "updated_at": time.Now()},
	}
	return seedCmdbRecords(db, "system_i18n", i18nEntries)
}

func seedGroupI18n(db *gorm.DB) error { return nil }

func seedCmdbDicts(db *gorm.DB) error {
	if db == nil {
		return nil
	}
	dictTypes := []map[string]interface{}{
		{"dict_code": "cmdb_host_status", "dict_name": "主机状态", "module": "business.cmdb", "status": 1, "created_at": time.Now(), "updated_at": time.Now()},
		{"dict_code": "cmdb_os_type", "dict_name": "操作系统类型", "module": "business.cmdb", "status": 1, "created_at": time.Now(), "updated_at": time.Now()},
		{"dict_code": "cmdb_label_key", "dict_name": "预置标签键", "module": "business.cmdb", "status": 1, "created_at": time.Now(), "updated_at": time.Now()},
		{"dict_code": "cmdb_env", "dict_name": "CMDB 环境", "module": "business.cmdb", "status": 1, "created_at": time.Now(), "updated_at": time.Now()},
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
		{"dict_code": "cmdb_host_status", "item_label_key": "可运维", "item_value": "online", "sort": 2, "status": 1, "created_at": time.Now(), "updated_at": time.Now()},
		{"dict_code": "cmdb_host_status", "item_label_key": "已下线", "item_value": "offline", "sort": 3, "status": 1, "created_at": time.Now(), "updated_at": time.Now()},
		{"dict_code": "cmdb_host_status", "item_label_key": "维护中", "item_value": "maintenance", "sort": 4, "status": 1, "created_at": time.Now(), "updated_at": time.Now()},
		{"dict_code": "cmdb_os_type", "item_label_key": "Linux", "item_value": "linux", "sort": 1, "status": 1, "created_at": time.Now(), "updated_at": time.Now()},
		{"dict_code": "cmdb_os_type", "item_label_key": "Windows", "item_value": "windows", "sort": 2, "status": 1, "created_at": time.Now(), "updated_at": time.Now()},
		{"dict_code": "cmdb_label_key", "item_label_key": "环境", "item_value": "env", "sort": 1, "status": 1, "created_at": time.Now(), "updated_at": time.Now()},
		{"dict_code": "cmdb_label_key", "item_label_key": "业务系统", "item_value": "biz", "sort": 2, "status": 1, "created_at": time.Now(), "updated_at": time.Now()},
		{"dict_code": "cmdb_label_key", "item_label_key": "集群", "item_value": "cluster", "sort": 3, "status": 1, "created_at": time.Now(), "updated_at": time.Now()},
		{"dict_code": "cmdb_label_key", "item_label_key": "区域", "item_value": "region", "sort": 4, "status": 1, "created_at": time.Now(), "updated_at": time.Now()},
		{"dict_code": "cmdb_label_key", "item_label_key": "数据库类型", "item_value": "db_type", "sort": 5, "status": 1, "created_at": time.Now(), "updated_at": time.Now()},
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
		} else if err := db.Table("system_dict_item").Where("id = ?", existingID).Updates(map[string]interface{}{
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
	schemas := []map[string]interface{}{
		{"key": "env", "name": "环境", "value_mode": "dict", "dict_code": "cmdb_env", "options": datatypes.JSON([]byte(`["dev","test","prod"]`)), "required": false, "status": "enabled", "description": "部署环境，如 dev/test/prod", "created_at": time.Now(), "updated_at": time.Now()},
		{"key": "biz", "name": "业务系统", "value_mode": "enum", "dict_code": "", "options": datatypes.JSON([]byte(`["Prometheus","Nginx","MySQL"]`)), "required": false, "status": "enabled", "description": "业务系统或应用归属", "created_at": time.Now(), "updated_at": time.Now()},
		{"key": "cluster", "name": "集群", "value_mode": "free", "dict_code": "", "options": datatypes.JSON([]byte(`[]`)), "required": false, "status": "enabled", "description": "集群或资源池归属", "created_at": time.Now(), "updated_at": time.Now()},
		{"key": "region", "name": "区域", "value_mode": "enum", "dict_code": "", "options": datatypes.JSON([]byte(`["西安开发环境","西安测试环境"]`)), "required": false, "status": "enabled", "description": "区域、机房或可用区", "created_at": time.Now(), "updated_at": time.Now()},
		{"key": "db_type", "name": "数据库类型", "value_mode": "enum", "dict_code": "", "options": datatypes.JSON([]byte(`["mysql","postgresql","redis"]`)), "required": false, "status": "enabled", "description": "主机承载的数据库类型", "created_at": time.Now(), "updated_at": time.Now()},
	}
	for _, schema := range schemas {
		var existingID uint64
		db.Table("biz_cmdb_label_schema").Select("id").Where("`key` = ?", schema["key"]).Limit(1).Pluck("id", &existingID)
		if existingID == 0 {
			if err := db.Table("biz_cmdb_label_schema").Create(schema).Error; err != nil {
				return err
			}
		} else if err := db.Table("biz_cmdb_label_schema").Where("id = ?", existingID).Updates(map[string]interface{}{
			"name":        schema["name"],
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

func seedCmdbRecords(db *gorm.DB, table string, records []map[string]interface{}) error {
	for _, record := range records {
		var existingID uint64
		db.Table(table).Select("id").Where("`key` = ? AND locale = ?",
			record["key"], record["locale"]).Limit(1).Pluck("id", &existingID)
		if existingID > 0 {
			update := map[string]interface{}{
				"value":      record["value"],
				"module":     record["module"],
				"updated_at": record["updated_at"],
			}
			if err := db.Table(table).Where("id = ?", existingID).Updates(update).Error; err != nil {
				return err
			}
		} else {
			if err := db.Table(table).Create(record).Error; err != nil {
				return err
			}
		}
	}
	return nil
}
