package iam

import (
	"errors"
	"fmt"
	"net/url"
	"pantheon-ops/backend/pkg/capability"
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
		return errors.New("database.not_initialized")
	}
	return s.db.AutoMigrate(&SystemMenu{})
}

// GetMenuTree 获取全量菜单树。
func (s *MenuService) GetMenuTree(query *MenuListQuery, roleKeys []string) ([]*MenuTreeResp, error) {
	if s.db == nil {
		return nil, errors.New("database.not_initialized")
	}

	if normalizeMenuScope(query) == "nav" {
		return s.getScopedNavigationMenuTree(roleKeys)
	}

	var menus []SystemMenu
	db := s.db.Model(&SystemMenu{})
	if query != nil {
		if strings.TrimSpace(query.TitleKey) != "" {
			db = db.Where("title_key LIKE ?", fmt.Sprintf("%%%s%%", strings.TrimSpace(query.TitleKey)))
		}
		if strings.TrimSpace(query.Path) != "" {
			db = db.Where("path LIKE ?", fmt.Sprintf("%%%s%%", strings.TrimSpace(query.Path)))
		}
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

	return buildMenuTree(menus, 0), nil
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

func (s *MenuService) getScopedNavigationMenuTree(roleKeys []string) ([]*MenuTreeResp, error) {
	allMenus, err := s.loadNavigationMenus()
	if err != nil {
		return nil, err
	}
	if len(allMenus) == 0 {
		return []*MenuTreeResp{}, nil
	}
	if hasRoleKey(roleKeys, "admin") {
		return buildMenuTree(allMenus, 0), nil
	}

	allowedIDs, err := s.loadAllowedNavigationMenuIDs(roleKeys)
	if err != nil {
		return nil, err
	}
	if len(allowedIDs) == 0 {
		return []*MenuTreeResp{}, nil
	}

	menuMap := make(map[uint64]SystemMenu, len(allMenus))
	selectedMap := make(map[uint64]struct{}, len(allowedIDs))
	for _, menu := range allMenus {
		menuMap[menu.ID] = menu
	}
	for _, menuID := range allowedIDs {
		currentID := menuID
		for currentID > 0 {
			menu, ok := menuMap[currentID]
			if !ok {
				break
			}
			if _, exists := selectedMap[menu.ID]; exists {
				currentID = menu.ParentID
				continue
			}
			selectedMap[menu.ID] = struct{}{}
			currentID = menu.ParentID
		}
	}

	selectedMenus := make([]SystemMenu, 0, len(selectedMap))
	for _, menu := range allMenus {
		if _, ok := selectedMap[menu.ID]; ok {
			selectedMenus = append(selectedMenus, menu)
		}
	}
	return buildMenuTree(selectedMenus, 0), nil
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
		return nil, errors.New("database.not_initialized")
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
		return nil, errors.New("database.not_initialized")
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

	if err := s.db.Save(&menu).Error; err != nil {
		return nil, err
	}
	return toMenuTreeResp(menu), nil
}

// DeleteMenu 删除菜单。
func (s *MenuService) DeleteMenu(menuID uint64) error {
	if s.db == nil {
		return errors.New("database.not_initialized")
	}

	var childCount int64
	if err := s.db.Model(&SystemMenu{}).Where("parent_id = ?", menuID).Count(&childCount).Error; err != nil {
		return err
	}
	if childCount > 0 {
		return errors.New("menu.delete.error.has_children")
	}

	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("DELETE FROM system_role_menu WHERE menu_id = ?", menuID).Error; err != nil {
			return err
		}
		return tx.Delete(&SystemMenu{}, menuID).Error
	})
}

func buildMenuTree(menus []SystemMenu, parentID uint64) []*MenuTreeResp {
	var tree []*MenuTreeResp
	for _, menu := range menus {
		if menu.ParentID == parentID {
			node := toMenuTreeResp(menu)
			node.Children = buildMenuTree(menus, menu.ID)
			tree = append(tree, node)
		}
	}
	return tree
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
		return nil
	}
	return tx.Exec("INSERT INTO system_role_menu (role_id, menu_id) VALUES (?, ?)", roleID, menuID).Error
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
		return errors.New("menu.update.error.parent_self")
	}
	if err := s.ensureParentExists(req.ParentID); err != nil {
		return err
	}
	return s.ensurePathUnique(menuID, req.Path)
}

func (s *MenuService) validateMenuMeta(menuID uint64, req *MenuCreateReq) error {
	routeName := normalizeMenuRouteName(req.RouteName)
	menuType := normalizeMenuType(req.Type)
	isExternal := normalizeMenuFlag(req.IsExternal)

	if menuType == "C" && routeName == "" {
		return errors.New("menu.route_name.required")
	}
	if menuType == "C" && isExternal != 1 && normalizeMenuPerm(req.PagePerm) == "" {
		return errors.New("menu.page_perm.required")
	}
	if menuType == "F" && normalizeMenuPerm(req.Perms) == "" {
		return errors.New("menu.perms.required")
	}
	if routeName != "" {
		if err := s.ensureRouteNameUnique(menuID, routeName); err != nil {
			return err
		}
	}
	if isExternal == 1 {
		if !isValidExternalMenuPath(req.Path) {
			return errors.New("menu.path.invalid_external")
		}
		return nil
	}
	componentKey := strings.TrimSpace(req.Component)
	if menuType == "C" && componentKey == "" {
		return errors.New("menu.component.required")
	}
	if menuType == "C" && requiresRegisteredMenuComponent(normalizeMenuModule(req.Module)) && !isRegisteredMenuComponentKey(componentKey) {
		return errors.New("menu.component.invalid")
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
		return errors.New("menu.parent.not_found")
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
		return errors.New("menu.path.exists")
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
		return errors.New("menu.route_name.exists")
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
