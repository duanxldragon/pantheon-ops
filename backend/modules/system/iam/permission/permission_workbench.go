package iam

import (
	"pantheon-base/pkg/common"
	"sort"
	"strings"
	"time"

	"pantheon-base/pkg/database"

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
		return nil, common.ErrDatabaseNotInitialized
	}

	roles, err := s.fetchWorkbenchRoles(query)
	if err != nil {
		return nil, err
	}

	resp := &PermissionWorkbenchResp{
		Overview: PermissionWorkbenchOverviewResp{},
		Roles:    make([]PermissionWorkbenchRoleResp, 0, len(roles)),
	}
	if len(roles) == 0 {
		return resp, nil
	}

	roleIndex := make(map[uint64]int, len(roles))
	resp.Overview.EnabledRoleCount = s.buildWorkbenchRoleResponses(roles, &resp.Roles, roleIndex)

	permissionCatalog, err := s.attachWorkbenchMenus(resp, roleIndex)
	if err != nil {
		return nil, err
	}
	if err := s.attachWorkbenchPermissions(resp, roleIndex, permissionCatalog); err != nil {
		return nil, err
	}
	if err := s.finalizeWorkbenchRoles(resp, roleIndex, query); err != nil {
		return nil, err
	}

	resp.Overview = summarizeWorkbenchOverview(resp.Roles)
	resp.Overview.RecentRemediationCount, err = s.countRecentWorkbenchRemediationEvents(extractWorkbenchRoleKeys(resp.Roles), 20)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (s *PermissionService) fetchWorkbenchRoles(query *PermissionWorkbenchQuery) ([]permissionWorkbenchRoleRow, error) {
	db := s.db.Table("system_role").Where("deleted_at IS NULL")
	if query != nil {
		if strings.TrimSpace(query.RoleKey) != "" {
			db = db.Where("role_key LIKE ?", "%"+common.EscapeLikePattern(strings.TrimSpace(query.RoleKey))+"%")
		}
		if query.Status != nil && common.IsEnabledStatus(*query.Status) {
			db = db.Where("status = ?", *query.Status)
		}
	}
	var roles []permissionWorkbenchRoleRow
	if err := db.
		Order(clause.OrderByColumn{Column: clause.Column{Name: "sort"}, Desc: false}).
		Order(clause.OrderByColumn{Column: clause.Column{Name: "id"}, Desc: false}).
		Find(&roles).Error; err != nil {
		return nil, err
	}
	return roles, nil
}

func (s *PermissionService) buildWorkbenchRoleResponses(roles []permissionWorkbenchRoleRow, out *[]PermissionWorkbenchRoleResp, roleIndex map[uint64]int) int {
	enabledCount := 0
	for index, item := range roles {
		roleIndex[item.ID] = index
		if item.Status == common.StatusEnabled {
			enabledCount++
		}
		*out = append(*out, PermissionWorkbenchRoleResp{
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
	return enabledCount
}

func (s *PermissionService) attachWorkbenchMenus(resp *PermissionWorkbenchResp, roleIndex map[uint64]int) (map[string]PermissionWorkbenchPermissionResp, error) {
	_, permissionCatalog, err := s.loadPermissionCatalog()
	if err != nil {
		return nil, err
	}
	roleIDs := make([]uint64, 0, len(resp.Roles))
	for _, r := range resp.Roles {
		roleIDs = append(roleIDs, r.ID)
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
	return permissionCatalog, nil
}

func (s *PermissionService) attachWorkbenchPermissions(resp *PermissionWorkbenchResp, roleIndex map[uint64]int, permissionCatalog map[string]PermissionWorkbenchPermissionResp) error {
	roleIDs := make([]uint64, 0, len(resp.Roles))
	for _, r := range resp.Roles {
		roleIDs = append(roleIDs, r.ID)
	}
	rolePermissions, err := s.loadWorkbenchPermissions(roleIDs)
	if err != nil {
		return err
	}
	for roleID, permissionKeys := range rolePermissions {
		index := roleIndex[roleID]
		s.classifyWorkbenchPermissions(resp, index, permissionKeys, permissionCatalog)
	}
	return nil
}

func (s *PermissionService) classifyWorkbenchPermissions(resp *PermissionWorkbenchResp, index int, permissionKeys []string, permissionCatalog map[string]PermissionWorkbenchPermissionResp) {
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

func (s *PermissionService) finalizeWorkbenchRoles(resp *PermissionWorkbenchResp, roleIndex map[uint64]int, query *PermissionWorkbenchQuery) error {
	roleKeys := make([]string, 0, len(resp.Roles))
	for _, r := range resp.Roles {
		roleKeys = append(roleKeys, r.RoleKey)
	}
	rolePolicies, err := s.loadWorkbenchPolicies(roleKeys)
	if err != nil {
		return err
	}
	latestRemediationEvents, err := s.loadLatestWorkbenchRemediationEvents(roleKeys)
	if err != nil {
		return err
	}
	for index := range resp.Roles {
		s.finalizeSingleWorkbenchRole(resp, index, rolePolicies, latestRemediationEvents)
	}
	if query != nil {
		s.applyWorkbenchFilters(resp, query)
	}
	return nil
}

func (s *PermissionService) finalizeSingleWorkbenchRole(resp *PermissionWorkbenchResp, index int, rolePolicies map[string][]PermissionWorkbenchAPIPolicyResp, latestRemediationEvents map[string]*PermissionWorkbenchRemediationEvent) {
	role := &resp.Roles[index]
	policies := rolePolicies[role.RoleKey]
	role.APIPolicies = policies
	role.APIPolicyCount = len(policies)
	role.HasPageGap = role.MenuCount > 0 && role.PagePermissionCount == 0
	requiredPolicies := collectRequiredAPIPolicies(role.PagePermissions, role.ActionPermissions)
	role.RequiredAPIPolicyCount = len(requiredPolicies)
	role.MissingAPIPolicies = diffMissingAPIPolicies(requiredPolicies, policies)
	role.MissingAPIPolicyCount = len(role.MissingAPIPolicies)
	role.HasAPIGap = role.MissingAPIPolicyCount > 0
	latestEvent := latestRemediationEvents[role.RoleKey]
	role.GovernanceStatus = resolveWorkbenchGovernanceStatus(*role, latestEvent)
	if latestEvent != nil {
		role.LastRemediationAction = latestEvent.Action
		role.LastRemediationAt = latestEvent.CreatedAt.Format(time.RFC3339)
	}
	resp.Overview.APIActionCount += len(policies)
	sortWorkbenchRoleSlices(role)
}

func sortWorkbenchRoleSlices(role *PermissionWorkbenchRoleResp) {
	sort.Slice(role.Menus, func(i, j int) bool {
		if role.Menus[i].Module == role.Menus[j].Module {
			if role.Menus[i].Path == role.Menus[j].Path {
				return role.Menus[i].ID < role.Menus[j].ID
			}
			return role.Menus[i].Path < role.Menus[j].Path
		}
		return role.Menus[i].Module < role.Menus[j].Module
	})
	sort.Slice(role.PagePermissions, func(i, j int) bool { return role.PagePermissions[i].Key < role.PagePermissions[j].Key })
	sort.Slice(role.ActionPermissions, func(i, j int) bool { return role.ActionPermissions[i].Key < role.ActionPermissions[j].Key })
	sort.Slice(role.UnknownPermissions, func(i, j int) bool { return role.UnknownPermissions[i].Key < role.UnknownPermissions[j].Key })
	sort.Slice(role.APIPolicies, func(i, j int) bool {
		if role.APIPolicies[i].Path == role.APIPolicies[j].Path {
			return role.APIPolicies[i].Method < role.APIPolicies[j].Method
		}
		return role.APIPolicies[i].Path < role.APIPolicies[j].Path
	})
	sort.Slice(role.MissingAPIPolicies, func(i, j int) bool {
		if role.MissingAPIPolicies[i].Path == role.MissingAPIPolicies[j].Path {
			return role.MissingAPIPolicies[i].Method < role.MissingAPIPolicies[j].Method
		}
		return role.MissingAPIPolicies[i].Path < role.MissingAPIPolicies[j].Path
	})
}

func (s *PermissionService) applyWorkbenchFilters(resp *PermissionWorkbenchResp, query *PermissionWorkbenchQuery) {
	switch strings.TrimSpace(query.Integrity) {
	case workbenchIntegrityUnknown:
		resp.Roles = filterWorkbenchRoles(resp.Roles, func(r PermissionWorkbenchRoleResp) bool { return r.UnknownPermissionCount > 0 })
	case workbenchIntegrityClean:
		resp.Roles = filterWorkbenchRoles(resp.Roles, func(r PermissionWorkbenchRoleResp) bool { return r.UnknownPermissionCount == 0 })
	}
	switch strings.TrimSpace(query.Coverage) {
	case workbenchCoveragePageGap:
		resp.Roles = filterWorkbenchRoles(resp.Roles, func(r PermissionWorkbenchRoleResp) bool { return r.HasPageGap })
	case workbenchCoverageAPIGap:
		resp.Roles = filterWorkbenchRoles(resp.Roles, func(r PermissionWorkbenchRoleResp) bool { return r.HasAPIGap })
	case workbenchCoverageComplete:
		resp.Roles = filterWorkbenchRoles(resp.Roles, func(r PermissionWorkbenchRoleResp) bool { return !r.HasPageGap && !r.HasAPIGap })
	}
}

func filterWorkbenchRoles(roles []PermissionWorkbenchRoleResp, pred func(PermissionWorkbenchRoleResp) bool) []PermissionWorkbenchRoleResp {
	filtered := make([]PermissionWorkbenchRoleResp, 0, len(roles))
	for _, role := range roles {
		if pred(role) {
			filtered = append(filtered, role)
		}
	}
	return filtered
}

func summarizeWorkbenchOverview(roles []PermissionWorkbenchRoleResp) PermissionWorkbenchOverviewResp {
	overview := PermissionWorkbenchOverviewResp{
		RoleCount: len(roles),
	}
	for _, role := range roles {
		if role.Status == common.StatusEnabled {
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
	return workbenchIntegrityClean
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

func collectRequiredAPIPolicies(pagePermissions, actionPermissions []PermissionWorkbenchPermissionResp) []permissionRequiredAPIPolicy {
	seen := make(map[string]struct{})
	result := make([]permissionRequiredAPIPolicy, 0)
	appendPolicy := func(path, method string) {
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

// Entries stay one-line "METHOD path" strings so this data table remains
// below copy-paste-detection token thresholds.
var requiredAPIRoutesByPermission = map[string][]string{
	"system:user:list":                   {"GET /api/v1/system/user/list"},
	"system:user:create":                 {"POST /api/v1/system/user/create"},
	"system:security-event:list":         {"GET /api/v1/system/security-event/list"},
	"system:security-event:acknowledge":  {"POST /api/v1/system/security-event/:id/acknowledge", "POST /api/v1/system/security-event/batch-acknowledge"},
	"system:security-event:clear":        {"POST /api/v1/system/security-event/cleanup"},
	"system:module:list":                 {"GET /api/v1/lowcode/dynamic-modules"},
	"system:module:register":             {"POST /api/v1/lowcode/dynamic-modules"},
	"system:module:unregister":           {"DELETE /api/v1/lowcode/dynamic-modules/:name"},
	"system:module:delete_record":        {"DELETE /api/v1/lowcode/dynamic-modules/:name/record"},
	"system:module:purge":                {"DELETE /api/v1/lowcode/dynamic-modules/:name/purge"},
	"system:module:generate":             {"POST /api/v1/lowcode/dynamic-modules/generate"},
	"system:generator:datasource:manage": {"POST /api/v1/lowcode/generator/datasources", "PUT /api/v1/lowcode/generator/datasources/:id", "DELETE /api/v1/lowcode/generator/datasources/:id", "POST /api/v1/lowcode/generator/datasources/:id/test"},
}

func requiredAPIPoliciesByPermissionKey(permissionKey string) []permissionRequiredAPIPolicy {
	routes, ok := requiredAPIRoutesByPermission[strings.TrimSpace(permissionKey)]
	if !ok {
		return nil
	}
	policies := make([]permissionRequiredAPIPolicy, 0, len(routes))
	for _, route := range routes {
		method, path, _ := strings.Cut(route, " ")
		policies = append(policies, permissionRequiredAPIPolicy{Path: path, Method: method})
	}
	return policies
}

// getRoleMissingAPIPolicies fetches only the data needed to determine missing API policies
// for a single role, without computing the full workbench. This avoids loading overview,
// remediation events, integrity/coverage filters, and unrelated roles.
func (s *PermissionService) getRoleMissingAPIPolicies(roleKey string) (*PermissionWorkbenchRoleResp, error) {
	var role permissionWorkbenchRoleRow
	if err := s.db.Table("system_role").
		Where("role_key = ? AND deleted_at IS NULL", roleKey).
		First(&role).Error; err != nil {
		return nil, err
	}

	result := &PermissionWorkbenchRoleResp{
		ID:                 role.ID,
		RoleName:           role.RoleName,
		RoleKey:            role.RoleKey,
		Status:             role.Status,
		Menus:              []PermissionWorkbenchMenuResp{},
		PagePermissions:    []PermissionWorkbenchPermissionResp{},
		ActionPermissions:  []PermissionWorkbenchPermissionResp{},
		UnknownPermissions: []PermissionWorkbenchPermissionResp{},
		APIPolicies:        []PermissionWorkbenchAPIPolicyResp{},
		MissingAPIPolicies: []PermissionWorkbenchAPIPolicyResp{},
	}

	_, permissionCatalog, err := s.loadPermissionCatalog()
	if err != nil {
		return nil, err
	}

	roleMenus, err := s.loadWorkbenchMenus([]uint64{role.ID})
	if err != nil {
		return nil, err
	}
	result.Menus = roleMenus[role.ID]
	result.MenuCount = len(result.Menus)

	rolePermissions, err := s.loadWorkbenchPermissions([]uint64{role.ID})
	if err != nil {
		return nil, err
	}
	for _, key := range rolePermissions[role.ID] {
		meta, ok := permissionCatalog[key]
		if !ok {
			result.UnknownPermissions = append(result.UnknownPermissions, PermissionWorkbenchPermissionResp{
				Key:  key,
				Kind: "unknown",
			})
			continue
		}
		if meta.Kind == "page" {
			result.PagePermissions = append(result.PagePermissions, meta)
		} else {
			result.ActionPermissions = append(result.ActionPermissions, meta)
		}
	}
	result.PagePermissionCount = len(result.PagePermissions)
	result.ActionPermissionCount = len(result.ActionPermissions)
	result.UnknownPermissionCount = len(result.UnknownPermissions)

	policies, err := s.loadWorkbenchPolicies([]string{roleKey})
	if err != nil {
		return nil, err
	}
	result.APIPolicies = policies[roleKey]
	result.APIPolicyCount = len(result.APIPolicies)

	requiredPolicies := collectRequiredAPIPolicies(result.PagePermissions, result.ActionPermissions)
	result.RequiredAPIPolicyCount = len(requiredPolicies)
	result.MissingAPIPolicies = diffMissingAPIPolicies(requiredPolicies, result.APIPolicies)
	result.MissingAPIPolicyCount = len(result.MissingAPIPolicies)
	result.HasPageGap = result.MenuCount > 0 && result.PagePermissionCount == 0
	result.HasAPIGap = result.MissingAPIPolicyCount > 0

	return result, nil
}
