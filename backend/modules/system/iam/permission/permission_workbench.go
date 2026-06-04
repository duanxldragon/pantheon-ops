package iam

import (
	"errors"
	"sort"
	"strings"
	"time"

	"pantheon-platform/backend/pkg/database"

	"gorm.io/gorm/clause"
)

type permissionWorkbenchRoleRow struct {
	ID       uint64 `gorm:"column:id"`
	RoleName string `gorm:"column:role_name"`
	RoleKey  string `gorm:"column:role_key"`
	Status   int    `gorm:"column:status"`
	Sort     int    `gorm:"column:sort"`
}

type permissionMenuCatalogRow struct {
	ID       uint64 `gorm:"column:id"`
	TitleKey string `gorm:"column:title_key"`
	Path     string `gorm:"column:path"`
	Module   string `gorm:"column:module"`
	PagePerm string `gorm:"column:page_perm"`
	Perms    string `gorm:"column:perms"`
	Type     string `gorm:"column:type"`
}

type permissionRoleMenuPair struct {
	RoleID   uint64 `gorm:"column:role_id"`
	MenuID   uint64 `gorm:"column:menu_id"`
	TitleKey string `gorm:"column:title_key"`
	Path     string `gorm:"column:path"`
	Module   string `gorm:"column:module"`
}

type permissionRoleKeyPair struct {
	RoleID        uint64 `gorm:"column:role_id"`
	PermissionKey string `gorm:"column:permission_key"`
}

type permissionRequiredAPIPolicy struct {
	Path   string
	Method string
}

func (s *PermissionService) GetWorkbench(query *PermissionWorkbenchQuery) (*PermissionWorkbenchResp, error) {
	if s.db == nil {
		return nil, errors.New("database.not_initialized")
	}

	var roles []permissionWorkbenchRoleRow
	db := s.db.Table("system_role").Where("deleted_at IS NULL")
	if query != nil {
		if strings.TrimSpace(query.RoleKey) != "" {
			db = db.Where("role_key LIKE ?", "%"+strings.TrimSpace(query.RoleKey)+"%")
		}
		if query.Status != nil && (*query.Status == 1 || *query.Status == 2) {
			db = db.Where("status = ?", *query.Status)
		}
	}

	if err := db.
		Order(clause.OrderByColumn{Column: clause.Column{Name: "sort"}, Desc: false}).
		Order(clause.OrderByColumn{Column: clause.Column{Name: "id"}, Desc: false}).
		Find(&roles).Error; err != nil {
		return nil, err
	}

	resp := &PermissionWorkbenchResp{
		Overview: PermissionWorkbenchOverviewResp{},
		Roles:    make([]PermissionWorkbenchRoleResp, 0, len(roles)),
	}
	if len(roles) == 0 {
		return resp, nil
	}

	roleIDs := make([]uint64, 0, len(roles))
	roleKeys := make([]string, 0, len(roles))
	roleIndex := make(map[uint64]int, len(roles))
	for index, item := range roles {
		roleIDs = append(roleIDs, item.ID)
		roleKeys = append(roleKeys, item.RoleKey)
		roleIndex[item.ID] = index
		if item.Status == 1 {
			resp.Overview.EnabledRoleCount++
		}
		resp.Roles = append(resp.Roles, PermissionWorkbenchRoleResp{
			ID:                 item.ID,
			RoleName:           item.RoleName,
			RoleKey:            item.RoleKey,
			Status:             item.Status,
			Menus:              []PermissionWorkbenchMenuResp{},
			PagePermissions:    []PermissionWorkbenchPermissionResp{},
			ActionPermissions:  []PermissionWorkbenchPermissionResp{},
			UnknownPermissions: []PermissionWorkbenchPermissionResp{},
			APIPolicies:        []PermissionWorkbenchAPIPolicyResp{},
			MissingAPIPolicies: []PermissionWorkbenchAPIPolicyResp{},
		})
	}
	resp.Overview.RoleCount = len(resp.Roles)

	_, permissionCatalog, err := s.loadPermissionCatalog()
	if err != nil {
		return nil, err
	}

	roleMenus, err := s.loadWorkbenchMenus(roleIDs)
	if err != nil {
		return nil, err
	}
	for roleID, menus := range roleMenus {
		index := roleIndex[roleID]
		resp.Roles[index].Menus = menus
		resp.Roles[index].MenuCount = len(menus)
		resp.Overview.NavigationAssignmentCount += len(menus)
	}

	rolePermissions, err := s.loadWorkbenchPermissions(roleIDs)
	if err != nil {
		return nil, err
	}
	for roleID, permissionKeys := range rolePermissions {
		index := roleIndex[roleID]
		for _, key := range permissionKeys {
			meta, ok := permissionCatalog[key]
			if !ok {
				resp.Roles[index].UnknownPermissions = append(resp.Roles[index].UnknownPermissions, PermissionWorkbenchPermissionResp{
					Key:  key,
					Kind: "unknown",
				})
				continue
			}
			if meta.Kind == "page" {
				resp.Roles[index].PagePermissions = append(resp.Roles[index].PagePermissions, meta)
				continue
			}
			resp.Roles[index].ActionPermissions = append(resp.Roles[index].ActionPermissions, meta)
		}
		resp.Roles[index].PagePermissionCount = len(resp.Roles[index].PagePermissions)
		resp.Roles[index].ActionPermissionCount = len(resp.Roles[index].ActionPermissions)
		resp.Roles[index].UnknownPermissionCount = len(resp.Roles[index].UnknownPermissions)
		resp.Overview.PagePermissionAssignmentCount += resp.Roles[index].PagePermissionCount
		resp.Overview.ActionPermissionAssignmentCount += resp.Roles[index].ActionPermissionCount
		resp.Overview.UnknownPermissionAssignmentCount += resp.Roles[index].UnknownPermissionCount
	}

	rolePolicies, err := s.loadWorkbenchPolicies(roleKeys)
	if err != nil {
		return nil, err
	}
	latestRemediationEvents, err := s.loadLatestWorkbenchRemediationEvents(roleKeys)
	if err != nil {
		return nil, err
	}
	for index := range resp.Roles {
		policies := rolePolicies[resp.Roles[index].RoleKey]
		resp.Roles[index].APIPolicies = policies
		resp.Roles[index].APIPolicyCount = len(policies)
		resp.Roles[index].HasPageGap = resp.Roles[index].MenuCount > 0 && resp.Roles[index].PagePermissionCount == 0
		requiredPolicies := collectRequiredAPIPolicies(resp.Roles[index].PagePermissions, resp.Roles[index].ActionPermissions)
		resp.Roles[index].RequiredAPIPolicyCount = len(requiredPolicies)
		resp.Roles[index].MissingAPIPolicies = diffMissingAPIPolicies(requiredPolicies, policies)
		resp.Roles[index].MissingAPIPolicyCount = len(resp.Roles[index].MissingAPIPolicies)
		resp.Roles[index].HasAPIGap = resp.Roles[index].MissingAPIPolicyCount > 0
		latestEvent := latestRemediationEvents[resp.Roles[index].RoleKey]
		resp.Roles[index].GovernanceStatus = resolveWorkbenchGovernanceStatus(resp.Roles[index], latestEvent)
		if latestEvent != nil {
			resp.Roles[index].LastRemediationAction = latestEvent.Action
			resp.Roles[index].LastRemediationAt = latestEvent.CreatedAt.Format(time.RFC3339)
		}
		resp.Overview.APIActionCount += len(policies)
		sort.Slice(resp.Roles[index].Menus, func(i, j int) bool {
			if resp.Roles[index].Menus[i].Module == resp.Roles[index].Menus[j].Module {
				if resp.Roles[index].Menus[i].Path == resp.Roles[index].Menus[j].Path {
					return resp.Roles[index].Menus[i].ID < resp.Roles[index].Menus[j].ID
				}
				return resp.Roles[index].Menus[i].Path < resp.Roles[index].Menus[j].Path
			}
			return resp.Roles[index].Menus[i].Module < resp.Roles[index].Menus[j].Module
		})
		sort.Slice(resp.Roles[index].PagePermissions, func(i, j int) bool {
			return resp.Roles[index].PagePermissions[i].Key < resp.Roles[index].PagePermissions[j].Key
		})
		sort.Slice(resp.Roles[index].ActionPermissions, func(i, j int) bool {
			return resp.Roles[index].ActionPermissions[i].Key < resp.Roles[index].ActionPermissions[j].Key
		})
		sort.Slice(resp.Roles[index].UnknownPermissions, func(i, j int) bool {
			return resp.Roles[index].UnknownPermissions[i].Key < resp.Roles[index].UnknownPermissions[j].Key
		})
		sort.Slice(resp.Roles[index].APIPolicies, func(i, j int) bool {
			if resp.Roles[index].APIPolicies[i].Path == resp.Roles[index].APIPolicies[j].Path {
				return resp.Roles[index].APIPolicies[i].Method < resp.Roles[index].APIPolicies[j].Method
			}
			return resp.Roles[index].APIPolicies[i].Path < resp.Roles[index].APIPolicies[j].Path
		})
		sort.Slice(resp.Roles[index].MissingAPIPolicies, func(i, j int) bool {
			if resp.Roles[index].MissingAPIPolicies[i].Path == resp.Roles[index].MissingAPIPolicies[j].Path {
				return resp.Roles[index].MissingAPIPolicies[i].Method < resp.Roles[index].MissingAPIPolicies[j].Method
			}
			return resp.Roles[index].MissingAPIPolicies[i].Path < resp.Roles[index].MissingAPIPolicies[j].Path
		})
	}

	if query != nil {
		switch strings.TrimSpace(query.Integrity) {
		case "unknown":
			filtered := make([]PermissionWorkbenchRoleResp, 0, len(resp.Roles))
			for _, role := range resp.Roles {
				if role.UnknownPermissionCount > 0 {
					filtered = append(filtered, role)
				}
			}
			resp.Roles = filtered
		case "clean":
			filtered := make([]PermissionWorkbenchRoleResp, 0, len(resp.Roles))
			for _, role := range resp.Roles {
				if role.UnknownPermissionCount == 0 {
					filtered = append(filtered, role)
				}
			}
			resp.Roles = filtered
		}
		switch strings.TrimSpace(query.Coverage) {
		case "page-gap":
			filtered := make([]PermissionWorkbenchRoleResp, 0, len(resp.Roles))
			for _, role := range resp.Roles {
				if role.HasPageGap {
					filtered = append(filtered, role)
				}
			}
			resp.Roles = filtered
		case "api-gap":
			filtered := make([]PermissionWorkbenchRoleResp, 0, len(resp.Roles))
			for _, role := range resp.Roles {
				if role.HasAPIGap {
					filtered = append(filtered, role)
				}
			}
			resp.Roles = filtered
		case "complete":
			filtered := make([]PermissionWorkbenchRoleResp, 0, len(resp.Roles))
			for _, role := range resp.Roles {
				if !role.HasPageGap && !role.HasAPIGap {
					filtered = append(filtered, role)
				}
			}
			resp.Roles = filtered
		}
	}

	resp.Overview = summarizeWorkbenchOverview(resp.Roles)
	resp.Overview.RecentRemediationCount, err = s.countRecentWorkbenchRemediationEvents(extractWorkbenchRoleKeys(resp.Roles), 20)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func summarizeWorkbenchOverview(roles []PermissionWorkbenchRoleResp) PermissionWorkbenchOverviewResp {
	overview := PermissionWorkbenchOverviewResp{
		RoleCount: len(roles),
	}
	for _, role := range roles {
		if role.Status == 1 {
			overview.EnabledRoleCount++
		}
		overview.NavigationAssignmentCount += role.MenuCount
		overview.PagePermissionAssignmentCount += role.PagePermissionCount
		overview.ActionPermissionAssignmentCount += role.ActionPermissionCount
		overview.APIActionCount += role.APIPolicyCount
		overview.UnknownPermissionAssignmentCount += role.UnknownPermissionCount
		if role.HasPageGap {
			overview.PageGapRoleCount++
		}
		if role.HasAPIGap {
			overview.APIGapRoleCount++
		}
		switch role.GovernanceStatus {
		case "pending":
			overview.PendingRemediationRoleCount++
		case "remediated":
			overview.RemediatedRoleCount++
		}
	}
	return overview
}

func resolveWorkbenchGovernanceStatus(role PermissionWorkbenchRoleResp, latest *PermissionWorkbenchRemediationEvent) string {
	if role.HasPageGap || role.HasAPIGap || role.UnknownPermissionCount > 0 {
		return "pending"
	}
	if latest != nil {
		return "remediated"
	}
	return "clean"
}

func extractWorkbenchRoleKeys(roles []PermissionWorkbenchRoleResp) []string {
	result := make([]string, 0, len(roles))
	for _, role := range roles {
		if strings.TrimSpace(role.RoleKey) == "" {
			continue
		}
		result = append(result, role.RoleKey)
	}
	return result
}

func (s *PermissionService) loadPermissionCatalog() (map[uint64]permissionMenuCatalogRow, map[string]PermissionWorkbenchPermissionResp, error) {
	var rows []permissionMenuCatalogRow
	if err := s.db.Table("system_menu").
		Select("id, title_key, path, module, page_perm, perms, type").
		Find(&rows).Error; err != nil {
		return nil, nil, err
	}

	menuCatalog := make(map[uint64]permissionMenuCatalogRow, len(rows))
	permissionCatalog := make(map[string]PermissionWorkbenchPermissionResp, len(rows)*2)
	for _, row := range rows {
		menuCatalog[row.ID] = row
		if strings.TrimSpace(row.PagePerm) != "" {
			permissionCatalog[strings.TrimSpace(row.PagePerm)] = PermissionWorkbenchPermissionResp{
				Key:      strings.TrimSpace(row.PagePerm),
				TitleKey: row.TitleKey,
				Path:     row.Path,
				Module:   row.Module,
				Kind:     "page",
			}
		}
		if strings.TrimSpace(row.Perms) != "" {
			permissionCatalog[strings.TrimSpace(row.Perms)] = PermissionWorkbenchPermissionResp{
				Key:      strings.TrimSpace(row.Perms),
				TitleKey: row.TitleKey,
				Path:     row.Path,
				Module:   row.Module,
				Kind:     "action",
			}
		}
	}
	return menuCatalog, permissionCatalog, nil
}

func (s *PermissionService) loadWorkbenchMenus(roleIDs []uint64) (map[uint64][]PermissionWorkbenchMenuResp, error) {
	result := make(map[uint64][]PermissionWorkbenchMenuResp, len(roleIDs))
	if len(roleIDs) == 0 {
		return result, nil
	}

	var pairs []permissionRoleMenuPair
	if err := s.db.Table("system_role_menu").
		Select("system_role_menu.role_id, system_menu.id AS menu_id, system_menu.title_key, system_menu.path, system_menu.module").
		Joins("JOIN system_menu ON system_menu.id = system_role_menu.menu_id").
		Where("system_role_menu.role_id IN ? AND system_menu.type <> ?", roleIDs, "F").
		Order("system_menu.sort asc, system_menu.id asc").
		Scan(&pairs).Error; err != nil {
		return nil, err
	}

	for _, pair := range pairs {
		result[pair.RoleID] = append(result[pair.RoleID], PermissionWorkbenchMenuResp{
			ID:       pair.MenuID,
			TitleKey: pair.TitleKey,
			Path:     pair.Path,
			Module:   pair.Module,
		})
	}
	return result, nil
}

func (s *PermissionService) loadWorkbenchPermissions(roleIDs []uint64) (map[uint64][]string, error) {
	result := make(map[uint64][]string, len(roleIDs))
	if len(roleIDs) == 0 {
		return result, nil
	}

	var pairs []permissionRoleKeyPair
	if err := s.db.Table("system_role_permission").
		Select("role_id, permission_key").
		Where("role_id IN ?", roleIDs).
		Order("permission_key asc").
		Scan(&pairs).Error; err != nil {
		return nil, err
	}

	for _, pair := range pairs {
		result[pair.RoleID] = append(result[pair.RoleID], pair.PermissionKey)
	}
	return result, nil
}

func (s *PermissionService) loadWorkbenchPolicies(roleKeys []string) (map[string][]PermissionWorkbenchAPIPolicyResp, error) {
	result := make(map[string][]PermissionWorkbenchAPIPolicyResp, len(roleKeys))
	if len(roleKeys) == 0 {
		return result, nil
	}

	var policies []database.CasbinRule
	if err := s.db.Model(&database.CasbinRule{}).
		Where("ptype = ? AND v0 IN ?", "p", roleKeys).
		Order("v0 asc, v1 asc, v2 asc, id asc").
		Find(&policies).Error; err != nil {
		return nil, err
	}

	for _, policy := range policies {
		result[policy.V0] = append(result[policy.V0], PermissionWorkbenchAPIPolicyResp{
			ID:     policy.ID,
			Path:   policy.V1,
			Method: policy.V2,
		})
	}
	return result, nil
}

func (s *PermissionService) loadLatestWorkbenchRemediationEvents(roleKeys []string) (map[string]*PermissionWorkbenchRemediationEvent, error) {
	result := make(map[string]*PermissionWorkbenchRemediationEvent, len(roleKeys))
	if len(roleKeys) == 0 {
		return result, nil
	}
	if !s.db.Migrator().HasTable(&PermissionWorkbenchRemediationEvent{}) {
		return result, nil
	}

	var events []PermissionWorkbenchRemediationEvent
	if err := s.db.Model(&PermissionWorkbenchRemediationEvent{}).
		Where("role_key IN ?", roleKeys).
		Order("created_at desc, id desc").
		Find(&events).Error; err != nil {
		return nil, err
	}
	for index := range events {
		if _, ok := result[events[index].RoleKey]; ok {
			continue
		}
		result[events[index].RoleKey] = &events[index]
	}
	return result, nil
}

func (s *PermissionService) countRecentWorkbenchRemediationEvents(roleKeys []string, limit int) (int, error) {
	if len(roleKeys) == 0 || limit <= 0 {
		return 0, nil
	}
	if !s.db.Migrator().HasTable(&PermissionWorkbenchRemediationEvent{}) {
		return 0, nil
	}

	var events []PermissionWorkbenchRemediationEvent
	if err := s.db.Model(&PermissionWorkbenchRemediationEvent{}).
		Where("role_key IN ?", roleKeys).
		Order("created_at desc, id desc").
		Limit(limit).
		Find(&events).Error; err != nil {
		return 0, err
	}
	return len(events), nil
}

func collectRequiredAPIPolicies(pagePermissions []PermissionWorkbenchPermissionResp, actionPermissions []PermissionWorkbenchPermissionResp) []permissionRequiredAPIPolicy {
	seen := make(map[string]struct{})
	result := make([]permissionRequiredAPIPolicy, 0)
	appendPolicy := func(path string, method string) {
		path = strings.TrimSpace(path)
		method = strings.ToUpper(strings.TrimSpace(method))
		if path == "" || method == "" {
			return
		}
		key := method + " " + path
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		result = append(result, permissionRequiredAPIPolicy{Path: path, Method: method})
	}

	for _, item := range pagePermissions {
		for _, policy := range requiredAPIPoliciesByPermissionKey(item.Key) {
			appendPolicy(policy.Path, policy.Method)
		}
	}
	for _, item := range actionPermissions {
		for _, policy := range requiredAPIPoliciesByPermissionKey(item.Key) {
			appendPolicy(policy.Path, policy.Method)
		}
	}
	return result
}

func diffMissingAPIPolicies(required []permissionRequiredAPIPolicy, actual []PermissionWorkbenchAPIPolicyResp) []PermissionWorkbenchAPIPolicyResp {
	if len(required) == 0 {
		return []PermissionWorkbenchAPIPolicyResp{}
	}
	actualSet := make(map[string]struct{}, len(actual))
	for _, item := range actual {
		key := strings.ToUpper(strings.TrimSpace(item.Method)) + " " + strings.TrimSpace(item.Path)
		actualSet[key] = struct{}{}
	}

	missing := make([]PermissionWorkbenchAPIPolicyResp, 0)
	for _, item := range required {
		key := item.Method + " " + item.Path
		if _, ok := actualSet[key]; ok {
			continue
		}
		missing = append(missing, PermissionWorkbenchAPIPolicyResp{
			Path:   item.Path,
			Method: item.Method,
		})
	}
	return missing
}

func requiredAPIPoliciesByPermissionKey(permissionKey string) []permissionRequiredAPIPolicy {
	switch strings.TrimSpace(permissionKey) {
	case "system:security-event:list":
		return []permissionRequiredAPIPolicy{
			{Path: "/api/v1/system/security-event/list", Method: "GET"},
		}
	case "system:module:list":
		return []permissionRequiredAPIPolicy{
			{Path: "/api/v1/system/dynamic-modules", Method: "GET"},
		}
	case "system:module:register":
		return []permissionRequiredAPIPolicy{
			{Path: "/api/v1/system/dynamic-modules", Method: "POST"},
		}
	case "system:module:unregister":
		return []permissionRequiredAPIPolicy{
			{Path: "/api/v1/system/dynamic-modules/:name", Method: "DELETE"},
		}
	case "system:module:delete_record":
		return []permissionRequiredAPIPolicy{
			{Path: "/api/v1/system/dynamic-modules/:name/record", Method: "DELETE"},
		}
	case "system:module:purge":
		return []permissionRequiredAPIPolicy{
			{Path: "/api/v1/system/dynamic-modules/:name/purge", Method: "DELETE"},
		}
	case "system:module:generate":
		return []permissionRequiredAPIPolicy{
			{Path: "/api/v1/system/dynamic-modules/generate", Method: "POST"},
		}
	case "system:generator:datasource:manage":
		return []permissionRequiredAPIPolicy{
			{Path: "/api/v1/system/generator/datasources", Method: "POST"},
			{Path: "/api/v1/system/generator/datasources/:id", Method: "PUT"},
			{Path: "/api/v1/system/generator/datasources/:id", Method: "DELETE"},
			{Path: "/api/v1/system/generator/datasources/:id/test", Method: "POST"},
		}
	default:
		return nil
	}
}
