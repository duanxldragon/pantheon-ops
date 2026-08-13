package iam

import (
	"fmt"
	"log/slog"
	"net/url"
	"pantheon-base/pkg/capability"
	"pantheon-base/pkg/common"
	"pantheon-base/pkg/rbacbind"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type MenuService struct {
	db *gorm.DB
}

func NewMenuService(db *gorm.DB) *MenuService {
	return &MenuService{db: db}
}

func (s *MenuService) Migrate() error {
	if s.db == nil {
		return common.ErrDatabaseNotInitialized
	}
	return s.db.AutoMigrate(&SystemMenu{})
}

// GetMenuTree 获取全量菜单树。
func (s *MenuService) GetMenuTree(query *MenuListQuery, roleKeys []string) ([]*MenuTreeResp, error) {
	if s.db == nil {
		return nil, common.ErrDatabaseNotInitialized
	}

	if normalizeMenuScope(query) == "nav" {
		return s.getScopedNavigationMenuTree(roleKeys)
	}

	var menus []SystemMenu
	db := s.db.Model(&SystemMenu{})
	textFiltered := false
	if query != nil {
		// 文本过滤（keyword/titleKey/path）在内存中做：命中子节点时需要补全祖先链，
		// 直接在 SQL 里过滤会把"父不匹配、子匹配"的整条链丢掉，树装配后结果为空。
		textFiltered = strings.TrimSpace(query.Keyword) != "" ||
			strings.TrimSpace(query.TitleKey) != "" ||
			strings.TrimSpace(query.Path) != ""
		if query.IsVisible != nil && (*query.IsVisible == 0 || *query.IsVisible == 1) {
			db = db.Where("is_visible = ?", *query.IsVisible)
		}
	}

	sortColumn, sortDesc := normalizeMenuSort(query)
	if err := db.
		Order(clause.OrderByColumn{
			Column: clause.Column{Name: sortColumn},
			Desc:   sortDesc,
		}).
		Order(clause.OrderByColumn{
			Column: clause.Column{Name: "id"},
			Desc:   false,
		}).
		Find(&menus).Error; err != nil {
		return nil, err
	}

	if textFiltered {
		menus = filterMenusWithAncestors(menus, query)
	}

	return normalizeManageMenuTree(buildMenuTree(menus, 0), 0), nil
}

// menuTextFilter 保存文本过滤条件（已统一小写）。
type menuTextFilter struct {
	keyword  string
	titleKey string
	path     string
}

// buildMenuTextFilter 从查询构造文本过滤条件（统一小写，便于 Contains 比较）。
func buildMenuTextFilter(query *MenuListQuery) menuTextFilter {
	return menuTextFilter{
		keyword:  strings.ToLower(strings.TrimSpace(query.Keyword)),
		titleKey: strings.ToLower(strings.TrimSpace(query.TitleKey)),
		path:     strings.ToLower(strings.TrimSpace(query.Path)),
	}
}

// menuMatchesTextFilter 判断单个菜单是否满足文本过滤条件（与原内联匹配逻辑等价）。
func menuMatchesTextFilter(menu SystemMenu, f menuTextFilter) bool {
	menuTitle := strings.ToLower(menu.TitleKey)
	menuPath := strings.ToLower(menu.Path)
	if f.keyword != "" && !strings.Contains(menuTitle, f.keyword) && !strings.Contains(menuPath, f.keyword) {
		return false
	}
	if f.titleKey != "" && !strings.Contains(menuTitle, f.titleKey) {
		return false
	}
	if f.path != "" && !strings.Contains(menuPath, f.path) {
		return false
	}
	return true
}

// menuFilterPredicate 判断单个菜单是否满足过滤条件。
type menuFilterPredicate func(menu SystemMenu) bool

// collectMenusWithAncestors 在 menus 中选出满足 predicate 的节点，并补全其祖先链，
// 返回按 menus 原始顺序排列的结果（保持排序）。与 filterMenusWithAncestors /
// getScopedNavigationMenuTree 中的内联祖先补全逻辑等价。
func collectMenusWithAncestors(menus []SystemMenu, predicate menuFilterPredicate) []SystemMenu {
	menuMap := make(map[uint64]SystemMenu, len(menus))
	for _, menu := range menus {
		menuMap[menu.ID] = menu
	}

	selected := make(map[uint64]struct{}, len(menus))
	for _, menu := range menus {
		if !predicate(menu) {
			continue
		}
		walkAncestors(selected, menuMap, menu.ID)
	}

	result := make([]SystemMenu, 0, len(selected))
	for _, menu := range menus {
		if _, ok := selected[menu.ID]; ok {
			result = append(result, menu)
		}
	}
	return result
}

// walkAncestors 将 id 及其所有祖先加入 selected 集合（命中已存在节点即停止，
// 因为该节点的祖先此前已被加入，等价于继续向上遍历）。
func walkAncestors(selected map[uint64]struct{}, menuMap map[uint64]SystemMenu, id uint64) {
	for id > 0 {
		node, ok := menuMap[id]
		if !ok {
			return
		}
		if _, exists := selected[node.ID]; exists {
			return
		}
		selected[node.ID] = struct{}{}
		id = node.ParentID
	}
}

// filterMenusWithAncestors 按文本条件过滤菜单并保留命中节点的祖先链，
// 保证树装配后命中的子节点仍然可见（保持传入的排序）。
func filterMenusWithAncestors(menus []SystemMenu, query *MenuListQuery) []SystemMenu {
	f := buildMenuTextFilter(query)
	return collectMenusWithAncestors(menus, func(menu SystemMenu) bool {
		return menuMatchesTextFilter(menu, f)
	})
}

func (s *MenuService) HasManageAccess(roleKeys []string) (bool, error) {
	if hasRoleKey(roleKeys, "admin") {
		return true, nil
	}
	if len(roleKeys) == 0 {
		return false, nil
	}

	var count int64
	err := s.db.Table("system_role_permission").
		Joins("JOIN system_role ON system_role.id = system_role_permission.role_id").
		Where("system_role.role_key IN ? AND system_role.status = ? AND system_role_permission.permission_key = ?", roleKeys, 1, "system:menu:list").
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// toIDSet 将 id 切片转为查询集合，便于 O(1) 命中判断。
func toIDSet(ids []uint64) map[uint64]struct{} {
	set := make(map[uint64]struct{}, len(ids))
	for _, id := range ids {
		set[id] = struct{}{}
	}
	return set
}

func (s *MenuService) getScopedNavigationMenuTree(roleKeys []string) ([]*MenuTreeResp, error) {
	allMenus, err := s.loadNavigationMenus()
	if err != nil {
		return nil, err
	}
	if len(allMenus) == 0 {
		return []*MenuTreeResp{}, nil
	}
	if hasRoleKey(roleKeys, "admin") {
		return normalizeScopedNavigationMenuTree(buildMenuTree(allMenus, 0), 0), nil
	}

	allowedIDs, err := s.loadAllowedNavigationMenuIDs(roleKeys)
	if err != nil {
		return nil, err
	}
	if len(allowedIDs) == 0 {
		return []*MenuTreeResp{}, nil
	}

	allowedSet := toIDSet(allowedIDs)
	selectedMenus := collectMenusWithAncestors(allMenus, func(menu SystemMenu) bool {
		_, ok := allowedSet[menu.ID]
		return ok
	})
	return normalizeScopedNavigationMenuTree(buildMenuTree(selectedMenus, 0), 0), nil
}

func (s *MenuService) loadNavigationMenus() ([]SystemMenu, error) {
	var menus []SystemMenu
	db := s.db.Model(&SystemMenu{}).
		Where("is_visible = ? AND type <> ?", 1, "F").
		Order("sort asc, id asc")
	if !capability.Load(s.db).OrgEnabled {
		db = db.Where("module <> ?", "system.org")
	}
	err := db.Find(&menus).Error
	return menus, err
}

func (s *MenuService) loadAllowedNavigationMenuIDs(roleKeys []string) ([]uint64, error) {
	if len(roleKeys) == 0 {
		return []uint64{}, nil
	}
	var menuIDs []uint64
	db := s.db.Table("system_menu").
		Distinct("system_menu.id").
		Joins("JOIN system_role_menu ON system_role_menu.menu_id = system_menu.id").
		Joins("JOIN system_role ON system_role.id = system_role_menu.role_id").
		Where("system_role.role_key IN ? AND system_role.status = ? AND system_menu.is_visible = ? AND system_menu.type <> ?", roleKeys, 1, 1, "F")
	if !capability.Load(s.db).OrgEnabled {
		db = db.Where("system_menu.module <> ?", "system.org")
	}
	err := db.
		Order("system_menu.sort asc, system_menu.id asc").
		Pluck("system_menu.id", &menuIDs).Error
	return menuIDs, err
}

// CreateMenu 创建菜单。
func (s *MenuService) CreateMenu(req *MenuCreateReq) (*MenuTreeResp, error) {
	if s.db == nil {
		return nil, common.ErrDatabaseNotInitialized
	}
	if err := s.validateMenuCreate(req); err != nil {
		return nil, err
	}

	menu := SystemMenu{
		ParentID:   req.ParentID,
		TitleKey:   req.TitleKey,
		Path:       req.Path,
		Component:  req.Component,
		PagePerm:   normalizeMenuPerm(req.PagePerm),
		Perms:      normalizeMenuPerm(req.Perms),
		Type:       normalizeMenuType(req.Type),
		Icon:       req.Icon,
		RouteName:  normalizeMenuRouteName(req.RouteName),
		Module:     normalizeMenuModule(req.Module),
		Sort:       req.Sort,
		IsVisible:  normalizeVisible(req.IsVisible),
		IsCache:    normalizeMenuFlag(req.IsCache),
		IsExternal: normalizeMenuFlag(req.IsExternal),
		ActiveMenu: normalizeMenuActiveMenu(req.ActiveMenu),
		HideInNav:  normalizeMenuFlag(req.HideInNav),
	}

	err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&menu).Error; err != nil {
			return err
		}
		return bindMenuToAdmin(tx, menu.ID)
	})
	if err != nil {
		return nil, err
	}

	return toMenuTreeResp(menu), nil
}

// UpdateMenu 更新菜单。
func (s *MenuService) UpdateMenu(menuID uint64, req *MenuUpdateReq) (*MenuTreeResp, error) {
	if s.db == nil {
		return nil, common.ErrDatabaseNotInitialized
	}

	var menu SystemMenu
	if err := s.db.First(&menu, menuID).Error; err != nil {
		return nil, err
	}
	if err := s.validateMenuUpdate(menuID, req); err != nil {
		return nil, err
	}

	menu.ParentID = req.ParentID
	menu.TitleKey = req.TitleKey
	menu.Path = req.Path
	menu.Component = req.Component
	menu.PagePerm = normalizeMenuPerm(req.PagePerm)
	menu.Perms = normalizeMenuPerm(req.Perms)
	menu.Type = normalizeMenuType(req.Type)
	menu.Icon = req.Icon
	menu.RouteName = normalizeMenuRouteName(req.RouteName)
	menu.Module = normalizeMenuModule(req.Module)
	menu.Sort = req.Sort
	menu.IsVisible = normalizeVisible(req.IsVisible)
	menu.IsCache = normalizeMenuFlag(req.IsCache)
	menu.IsExternal = normalizeMenuFlag(req.IsExternal)
	menu.ActiveMenu = normalizeMenuActiveMenu(req.ActiveMenu)
	menu.HideInNav = normalizeMenuFlag(req.HideInNav)

	if err := s.db.Save(&menu).Error; err != nil {
		return nil, err
	}
	return toMenuTreeResp(menu), nil
}

// DeleteMenu 删除菜单。
func (s *MenuService) DeleteMenu(menuID uint64) error {
	if s.db == nil {
		return common.ErrDatabaseNotInitialized
	}

	return s.db.Transaction(func(tx *gorm.DB) error {
		menu, err := loadDeletableMenu(tx, menuID)
		if err != nil {
			return err
		}
		if err := deleteMenuBindingsAndRow(tx, menuID); err != nil {
			return err
		}
		return cleanupOrphanedMenuPermissions(tx, menu.PagePerm, menu.Perms)
	})
}

func loadDeletableMenu(tx *gorm.DB, menuID uint64) (SystemMenu, error) {
	var menu SystemMenu
	if err := tx.First(&menu, menuID).Error; err != nil {
		return menu, err
	}
	// 子菜单检查放在事务内，收窄检查与删除之间的并发窗口。
	var childCount int64
	if err := tx.Model(&SystemMenu{}).Where("parent_id = ?", menuID).Count(&childCount).Error; err != nil {
		return menu, err
	}
	if childCount > 0 {
		return menu, common.NewInternal("menu.delete.error.has_children")
	}
	return menu, nil
}

func deleteMenuBindingsAndRow(tx *gorm.DB, menuID uint64) error {
	if err := tx.Table("system_role_menu").Where("menu_id = ?", menuID).Delete(nil).Error; err != nil {
		return err
	}
	return tx.Delete(&SystemMenu{}, menuID).Error
}

func cleanupOrphanedMenuPermissions(tx *gorm.DB, permissionKeys ...string) error {
	// 若被删菜单是某 permission_key 的最后一个持有者，同步清理授权行；
	// 多个菜单共享同一 key 时保留。
	for _, key := range uniqueNonEmpty(permissionKeys...) {
		var owners int64
		if err := tx.Model(&SystemMenu{}).
			Where("page_perm = ? OR perms = ?", key, key).
			Count(&owners).Error; err != nil {
			return err
		}
		if owners > 0 {
			continue
		}
		if err := tx.Table("system_role_permission").
			Where("permission_key = ?", key).Delete(nil).Error; err != nil {
			return err
		}
	}
	return nil
}

func uniqueNonEmpty(values ...string) []string {
	result := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func buildMenuTree(menus []SystemMenu, parentID uint64) []*MenuTreeResp {
	// Build index: parentID -> menus with that parent
	// Deduplicate by path (for menus with path) or perms (for menus without path)
	index := make(map[uint64][]*MenuTreeResp)
	seen := make(map[string]struct{}) // track unique key per parent
	for i := range menus {
		m := menus[i]
		// Determine deduplication key: prefer path, fallback to perms
		var dedupKey string
		if m.Path != "" {
			dedupKey = m.Path
		} else if m.Perms != "" {
			dedupKey = m.Perms
		} else {
			// No path or perms — use id as fallback key
			dedupKey = fmt.Sprintf("id:%d", m.ID)
		}
		seenKey := fmt.Sprintf("%d:%s", m.ParentID, dedupKey)
		if _, exists := seen[seenKey]; exists {
			continue // skip duplicate
		}
		seen[seenKey] = struct{}{}
		node := toMenuTreeResp(m)
		index[m.ParentID] = append(index[m.ParentID], node)
	}
	// Recursively attach children using the index — O(n) total
	var attachChildren func(parentID uint64)
	attachChildren = func(parentID uint64) {
		children := index[parentID]
		for _, child := range children {
			child.Children = index[child.ID]
			if len(child.Children) > 0 {
				attachChildren(child.ID)
			}
		}
	}
	attachChildren(parentID)
	return index[parentID]
}

func normalizeManageMenuTree(nodes []*MenuTreeResp, parentID uint64) []*MenuTreeResp {
	normalized := make([]*MenuTreeResp, 0, len(nodes))
	for _, node := range nodes {
		normalized = append(normalized, normalizeManageMenuNode(node, parentID)...)
	}
	return normalized
}

func normalizeManageMenuNode(node *MenuTreeResp, parentID uint64) []*MenuTreeResp {
	if shouldHideManageMenuNode(node) {
		return normalizeManageMenuTree(node.Children, parentID)
	}

	cloned := *node
	cloned.ParentID = parentID
	cloned.Children = normalizeManageMenuTree(node.Children, cloned.ID)
	return []*MenuTreeResp{&cloned}
}

func shouldHideManageMenuNode(node *MenuTreeResp) bool {
	return node.HideInNav == 1 || isLegacyHiddenContainer(node.Path)
}

func normalizeScopedNavigationMenuTree(nodes []*MenuTreeResp, parentID uint64) []*MenuTreeResp {
	normalized := make([]*MenuTreeResp, 0, len(nodes))
	dedupSeen := make(map[string]struct{}) // global dedup by path at this level
	for _, node := range nodes {
		for _, n := range normalizeScopedNavigationMenuNode(node, parentID) {
			if n.Path != "" {
				if _, exists := dedupSeen[n.Path]; exists {
					continue // skip duplicate at this level (e.g. children promoted from multiple hidden containers)
				}
				dedupSeen[n.Path] = struct{}{}
			}
			normalized = append(normalized, n)
		}
	}
	return normalized
}

func normalizeScopedNavigationMenuNode(node *MenuTreeResp, parentID uint64) []*MenuTreeResp {
	if shouldHideScopedNavigationMenuNode(node) {
		return normalizeScopedNavigationMenuTree(node.Children, parentID)
	}

	cloned := *node
	cloned.ParentID = parentID
	cloned.Children = normalizeScopedNavigationMenuTree(node.Children, cloned.ID)
	return []*MenuTreeResp{&cloned}
}

func shouldHideScopedNavigationMenuNode(node *MenuTreeResp) bool {
	return node.HideInNav == 1 || isLegacyHiddenContainer(node.Path)
}

func isLegacyHiddenContainer(path string) bool {
	switch strings.TrimSpace(path) {
	case "/workspace", "/operations":
		return true
	default:
		return false
	}
}

func toMenuTreeResp(menu SystemMenu) *MenuTreeResp {
	return &MenuTreeResp{
		ID:         menu.ID,
		ParentID:   menu.ParentID,
		TitleKey:   menu.TitleKey,
		Path:       menu.Path,
		Component:  menu.Component,
		PagePerm:   menu.PagePerm,
		Perms:      menu.Perms,
		Type:       menu.Type,
		Icon:       menu.Icon,
		RouteName:  menu.RouteName,
		Module:     menu.Module,
		Sort:       menu.Sort,
		IsVisible:  menu.IsVisible,
		IsCache:    menu.IsCache,
		IsExternal: menu.IsExternal,
		ActiveMenu: menu.ActiveMenu,
		HideInNav:  menu.HideInNav,
	}
}

func normalizeVisible(value int) int {
	if value == 0 {
		return 0
	}
	return 1
}

func normalizeMenuType(value string) string {
	switch value {
	case "M", "C", "F":
		return value
	default:
		return "C"
	}
}

func normalizeMenuFlag(value int) int {
	if value == 1 {
		return 1
	}
	return 0
}

func normalizeMenuRouteName(value string) string {
	return strings.TrimSpace(value)
}

func normalizeMenuPerm(value string) string {
	return strings.TrimSpace(value)
}

func normalizeMenuModule(value string) string {
	if strings.TrimSpace(value) == "" {
		return "system"
	}
	return strings.TrimSpace(value)
}

func normalizeMenuActiveMenu(value string) string {
	return strings.TrimSpace(value)
}

func normalizeMenuSort(query *MenuListQuery) (string, bool) {
	if query == nil {
		return "sort", false
	}

	sortWhitelist := map[string]string{
		"id":          "id",
		"titleKey":    "title_key",
		"title_key":   "title_key",
		"path":        "path",
		"routeName":   "route_name",
		"route_name":  "route_name",
		"pagePerm":    "page_perm",
		"page_perm":   "page_perm",
		"perms":       "perms",
		"type":        "type",
		"module":      "module",
		"sort":        "sort",
		"isCache":     "is_cache",
		"is_cache":    "is_cache",
		"isExternal":  "is_external",
		"is_external": "is_external",
		"isVisible":   "is_visible",
		"is_visible":  "is_visible",
	}

	column, ok := sortWhitelist[strings.TrimSpace(query.SortField)]
	if !ok {
		column = "sort"
	}

	order := strings.ToLower(strings.TrimSpace(query.SortOrder))
	return column, order == "desc"
}

func normalizeMenuScope(query *MenuListQuery) string {
	if query == nil {
		return "nav"
	}
	if strings.ToLower(strings.TrimSpace(query.Scope)) == "manage" {
		return "manage"
	}
	return "nav"
}

func hasRoleKey(roleKeys []string, target string) bool {
	for _, item := range roleKeys {
		if item == target {
			return true
		}
	}
	return false
}

func bindMenuToAdmin(tx *gorm.DB, menuID uint64) error {
	var roleID uint64
	if err := tx.Table("system_role").Select("id").Where("role_key = ?", "admin").Limit(1).Pluck("id", &roleID).Error; err != nil {
		return err
	}
	if roleID == 0 {
		slog.Warn("bindMenuToAdmin: admin role not found, skipping menu binding", "menuID", menuID)
		return nil
	}
	return rbacbind.EnsureRoleMenu(tx, roleID, menuID)
}

func (s *MenuService) validateMenuCreate(req *MenuCreateReq) error {
	if err := s.validateMenuMeta(0, req); err != nil {
		return err
	}
	if err := s.ensureParentExists(req.ParentID); err != nil {
		return err
	}
	return s.ensurePathUnique(0, req.Path)
}

func (s *MenuService) validateMenuUpdate(menuID uint64, req *MenuUpdateReq) error {
	if err := s.validateMenuMeta(menuID, &MenuCreateReq{
		ParentID:   req.ParentID,
		TitleKey:   req.TitleKey,
		Path:       req.Path,
		Component:  req.Component,
		PagePerm:   req.PagePerm,
		Perms:      req.Perms,
		Type:       req.Type,
		Icon:       req.Icon,
		RouteName:  req.RouteName,
		Module:     req.Module,
		Sort:       req.Sort,
		IsVisible:  req.IsVisible,
		IsCache:    req.IsCache,
		IsExternal: req.IsExternal,
		ActiveMenu: req.ActiveMenu,
	}); err != nil {
		return err
	}
	if req.ParentID == menuID {
		return common.NewInternal("menu.update.error.parent_self")
	}
	if err := s.ensureParentExists(req.ParentID); err != nil {
		return err
	}
	return s.ensurePathUnique(menuID, req.Path)
}

// isRouteNameRequired 目录/菜单类型且未提供路由名时必填（与原内联守卫等价）。
func isRouteNameRequired(menuType, routeName string) bool {
	return menuType == "C" && routeName == ""
}

// isPagePermRequired 菜单类型、非外链且未配置页面权限时必填（与原内联守卫等价）。
func isPagePermRequired(menuType string, isExternal int, pagePerm string) bool {
	return menuType == "C" && isExternal != 1 && pagePerm == ""
}

// isPermsRequired 按钮类型且未配置权限标识时必填（与原内联守卫等价）。
func isPermsRequired(menuType, perms string) bool {
	return menuType == "F" && perms == ""
}

// ensureRouteNameUniqueIfPresent 仅当路由名非空时校验唯一性（与原嵌套守卫等价）。
func ensureRouteNameUniqueIfPresent(s *MenuService, menuID uint64, routeName string) error {
	if routeName == "" {
		return nil
	}
	return s.ensureRouteNameUnique(menuID, routeName)
}

// validateExternalMenuPath 校验外链菜单的 path 必须是合法 http/https 链接。
func validateExternalMenuPath(path string) error {
	if !isValidExternalMenuPath(path) {
		return common.NewBadRequest("menu.path.invalid_external")
	}
	return nil
}

// isComponentRequired 菜单类型且未提供组件键时必填（与原内联守卫等价）。
func isComponentRequired(menuType, componentKey string) bool {
	return menuType == "C" && componentKey == ""
}

// isComponentInvalid 菜单类型、所属模块要求注册组件且组件键未在注册表中时非法（与原内联守卫等价）。
func isComponentInvalid(menuType, module, componentKey string) bool {
	return menuType == "C" && requiresRegisteredMenuComponent(module) && !isRegisteredMenuComponentKey(componentKey)
}

func (s *MenuService) validateMenuMeta(menuID uint64, req *MenuCreateReq) error {
	routeName := normalizeMenuRouteName(req.RouteName)
	menuType := normalizeMenuType(req.Type)
	isExternal := normalizeMenuFlag(req.IsExternal)

	if isRouteNameRequired(menuType, routeName) {
		return common.NewBadRequest("menu.route_name.required")
	}
	if isPagePermRequired(menuType, isExternal, normalizeMenuPerm(req.PagePerm)) {
		return common.NewBadRequest("menu.page_perm.required")
	}
	if isPermsRequired(menuType, normalizeMenuPerm(req.Perms)) {
		return common.NewBadRequest("menu.perms.required")
	}
	if err := ensureRouteNameUniqueIfPresent(s, menuID, routeName); err != nil {
		return err
	}
	if isExternal == 1 {
		return validateExternalMenuPath(req.Path)
	}
	componentKey := strings.TrimSpace(req.Component)
	if isComponentRequired(menuType, componentKey) {
		return common.NewBadRequest("menu.component.required")
	}
	if isComponentInvalid(menuType, normalizeMenuModule(req.Module), componentKey) {
		return common.NewBadRequest("menu.component.invalid")
	}
	return nil
}

func (s *MenuService) ensureParentExists(parentID uint64) error {
	if parentID == 0 {
		return nil
	}

	var count int64
	if err := s.db.Model(&SystemMenu{}).Where("id = ?", parentID).Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return common.NewNotFound("menu.parent.not_found")
	}
	return nil
}

func (s *MenuService) ensurePathUnique(menuID uint64, path string) error {
	if strings.TrimSpace(path) == "" {
		return nil
	}

	var count int64
	db := s.db.Model(&SystemMenu{}).Where("path = ?", path)
	if menuID > 0 {
		db = db.Where("id <> ?", menuID)
	}
	if err := db.Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return common.NewConflict("menu.path.exists")
	}
	return nil
}

func (s *MenuService) ensureRouteNameUnique(menuID uint64, routeName string) error {
	var count int64
	db := s.db.Model(&SystemMenu{}).Where("route_name = ?", routeName)
	if menuID > 0 {
		db = db.Where("id <> ?", menuID)
	}
	if err := db.Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return common.NewConflict("menu.route_name.exists")
	}
	return nil
}

func isValidExternalMenuPath(path string) bool {
	parsed, err := url.Parse(strings.TrimSpace(path))
	if err != nil {
		return false
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return false
	}
	return parsed.Host != ""
}
