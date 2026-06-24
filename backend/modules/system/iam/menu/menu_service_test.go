package iam

import (
	"testing"

	"gorm.io/gorm"
	"pantheon-ops/backend/pkg/testmysql"
)

func setupMenuTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db := testmysql.Open(t)
	if err := db.AutoMigrate(&SystemMenu{}); err != nil {
		t.Fatalf("migrate menu: %v", err)
	}
	_ = db.Exec("CREATE TABLE IF NOT EXISTS system_role (id BIGINT PRIMARY KEY AUTO_INCREMENT, role_key VARCHAR(64), status INT)")
	_ = db.Exec("CREATE TABLE IF NOT EXISTS system_role_menu (role_id BIGINT, menu_id BIGINT, PRIMARY KEY (role_id, menu_id))")
	_ = db.Exec("CREATE TABLE IF NOT EXISTS system_role_permission (role_id BIGINT, permission_key VARCHAR(128))")
	_ = db.Exec("CREATE TABLE IF NOT EXISTS system_setting (setting_key VARCHAR(128) PRIMARY KEY, setting_value TEXT)")
	return db
}

func TestMenuServiceCreateMenuBindsAdminRole(t *testing.T) {
	db := setupMenuTestDB(t)
	service := NewMenuService(db)

	if err := db.Exec("INSERT INTO system_role (id, role_key, status) VALUES (1, 'admin', 1)").Error; err != nil {
		t.Fatalf("seed admin role: %v", err)
	}

	menu, err := service.CreateMenu(&MenuCreateReq{
		TitleKey:   "system.menu.user",
		Path:       "/system/user-smoke",
		Component:  "system/user/UserList",
		PagePerm:   "system:user:list",
		Type:       "C",
		RouteName:  "system-user-smoke",
		Module:     "system.iam",
		IsVisible:  1,
		IsExternal: 0,
	})
	if err != nil {
		t.Fatalf("create menu: %v", err)
	}

	var count int64
	if err := db.Table("system_role_menu").Where("role_id = ? AND menu_id = ?", 1, menu.ID).Count(&count).Error; err != nil {
		t.Fatalf("count admin role menu bindings: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected admin role to bind created menu, got count=%d", count)
	}
}

func TestMenuServiceValidateMenuMetaRejectsUnknownRegisteredComponent(t *testing.T) {
	service := NewMenuService(setupMenuTestDB(t))

	err := service.validateMenuMeta(0, &MenuCreateReq{
		TitleKey:   "system.menu.example",
		Path:       "/system/example",
		Component:  "system/example/MissingPage",
		PagePerm:   "system:example:list",
		Type:       "C",
		RouteName:  "system-example",
		Module:     "system.iam",
		IsExternal: 0,
	})
	if err == nil || err.Error() != "menu.component.invalid" {
		t.Fatalf("expected menu.component.invalid, got %v", err)
	}
}

func TestMenuServiceValidateMenuMetaAcceptsRegisteredComponent(t *testing.T) {
	service := NewMenuService(setupMenuTestDB(t))

	err := service.validateMenuMeta(0, &MenuCreateReq{
		TitleKey:   "system.menu.user",
		Path:       "/system/user",
		Component:  "system/user/UserList",
		PagePerm:   "system:user:list",
		Type:       "C",
		RouteName:  "system-user",
		Module:     "system.iam",
		IsExternal: 0,
	})
	if err != nil {
		t.Fatalf("expected registered component to pass, got %v", err)
	}
}

func TestMenuServiceHasManageAccess(t *testing.T) {
	db := setupMenuTestDB(t)
	service := NewMenuService(db)

	if err := db.Exec("INSERT INTO system_role (id, role_key, status) VALUES (1, 'menu_manager', 1)").Error; err != nil {
		t.Fatalf("seed role: %v", err)
	}
	if err := db.Exec("INSERT INTO system_role_permission (role_id, permission_key) VALUES (1, 'system:menu:list')").Error; err != nil {
		t.Fatalf("seed role permission: %v", err)
	}

	allowed, err := service.HasManageAccess([]string{"menu_manager"})
	if err != nil {
		t.Fatalf("has manage access: %v", err)
	}
	if !allowed {
		t.Fatalf("expected menu_manager to have manage access")
	}

	allowed, err = service.HasManageAccess([]string{"guest"})
	if err != nil {
		t.Fatalf("has manage access for guest: %v", err)
	}
	if allowed {
		t.Fatalf("expected guest to have no manage access")
	}
}

func TestMenuServiceNavigationHidesOrgWhenCapabilityDisabled(t *testing.T) {
	db := setupMenuTestDB(t)
	service := NewMenuService(db)

	if err := db.Create(&SystemMenu{ID: 1, TitleKey: "system.menu.user", Path: "/system/user", Type: "C", Module: "system.iam", IsVisible: 1}).Error; err != nil {
		t.Fatalf("seed iam menu: %v", err)
	}
	if err := db.Create(&SystemMenu{ID: 2, TitleKey: "system.menu.org", Path: "/system/org", Type: "M", Module: "system.org", IsVisible: 1}).Error; err != nil {
		t.Fatalf("seed org menu: %v", err)
	}
	if err := db.Exec("INSERT INTO system_setting (setting_key, setting_value) VALUES ('org.enabled', 'false')").Error; err != nil {
		t.Fatalf("seed org capability: %v", err)
	}

	tree, err := service.GetMenuTree(&MenuListQuery{Scope: "nav"}, []string{"admin"})
	if err != nil {
		t.Fatalf("get nav tree: %v", err)
	}
	if len(tree) != 1 || tree[0].Path != "/system/user" {
		t.Fatalf("expected only iam menu when org disabled, got %+v", tree)
	}
}

func TestMenuServiceNavigationFlattensPlatformWorkspaceMenus(t *testing.T) {
	db := setupMenuTestDB(t)
	service := NewMenuService(db)

	if err := db.Create(&SystemMenu{
		ID:        1,
		TitleKey:  "app.workspace",
		Path:      "/workspace",
		Type:      "M",
		Icon:      "dashboard",
		RouteName: "workspace",
		Module:    "platform",
		Sort:      10,
		IsVisible: 1,
	}).Error; err != nil {
		t.Fatalf("seed workspace menu: %v", err)
	}
	if err := db.Create(&SystemMenu{
		ID:        2,
		ParentID:   1,
		TitleKey:  "system.menu.dashboard",
		Path:      "/dashboard",
		Type:      "C",
		Icon:      "dashboard",
		RouteName: "dashboard",
		Module:    "platform",
		Sort:      1,
		IsVisible: 1,
	}).Error; err != nil {
		t.Fatalf("seed dashboard menu: %v", err)
	}
	if err := db.Create(&SystemMenu{
		ID:        3,
		ParentID:   1,
		TitleKey:  "operations.menu",
		Path:      "/operations",
		Type:      "M",
		Icon:      "desktop",
		RouteName: "operations",
		Module:    "platform",
		Sort:      20,
		IsVisible: 1,
	}).Error; err != nil {
		t.Fatalf("seed operations menu: %v", err)
	}
	if err := db.Create(&SystemMenu{
		ID:        4,
		ParentID:   3,
		TitleKey:  "business.ticket.menu",
		Path:      "/operations/ticket",
		Component: "business/ticket/TicketList",
		PagePerm:  "business:ticket:list",
		Type:      "C",
		Icon:      "file",
		RouteName: "business-ticket",
		Module:    "business.ticket",
		Sort:      1,
		IsVisible: 1,
	}).Error; err != nil {
		t.Fatalf("seed operations child menu: %v", err)
	}

	tree, err := service.GetMenuTree(&MenuListQuery{Scope: "nav"}, []string{"admin"})
	if err != nil {
		t.Fatalf("get nav tree: %v", err)
	}

	paths := collectMenuTreePaths(tree)
	if containsMenuPath(paths, "/workspace") {
		t.Fatalf("expected workspace root to be hidden, got %+v", paths)
	}
	if containsMenuPath(paths, "/operations") {
		t.Fatalf("expected operations root to be hidden, got %+v", paths)
	}
	if len(tree) != 2 {
		t.Fatalf("expected two visible roots after flattening, got %+v", paths)
	}
	if tree[0].Path != "/dashboard" || tree[0].ParentID != 0 {
		t.Fatalf("expected dashboard to become root, got %+v", tree[0])
	}
	if tree[1].Path != "/operations/ticket" || tree[1].ParentID != 0 {
		t.Fatalf("expected operations child to be promoted to root, got %+v", tree[1])
	}
}

func TestMenuServiceManageTreeHidesWorkspaceContainer(t *testing.T) {
	db := setupMenuTestDB(t)
	service := NewMenuService(db)

	if err := db.Create(&SystemMenu{
		ID:        1,
		TitleKey:  "app.workspace",
		Path:      "/workspace",
		Type:      "D",
		Icon:      "dashboard",
		RouteName: "workspace",
		Module:    "platform",
		Sort:      10,
		IsVisible: 1,
	}).Error; err != nil {
		t.Fatalf("seed workspace menu: %v", err)
	}
	if err := db.Create(&SystemMenu{
		ID:        2,
		ParentID:   1,
		TitleKey:  "system.menu.dashboard",
		Path:      "/dashboard",
		Type:      "C",
		Icon:      "dashboard",
		RouteName: "dashboard",
		Module:    "platform",
		Sort:      1,
		IsVisible: 1,
	}).Error; err != nil {
		t.Fatalf("seed dashboard menu: %v", err)
	}

	tree, err := service.GetMenuTree(&MenuListQuery{Scope: "manage"}, []string{"admin"})
	if err != nil {
		t.Fatalf("get manage tree: %v", err)
	}

	paths := collectMenuTreePaths(tree)
	if containsMenuPath(paths, "/workspace") {
		t.Fatalf("expected workspace container to be hidden in manage tree, got %+v", paths)
	}
	if len(tree) != 1 || tree[0].Path != "/dashboard" || tree[0].ParentID != 0 {
		t.Fatalf("expected dashboard to be promoted to root, got %+v", tree)
	}
}

func collectMenuTreePaths(nodes []*MenuTreeResp) []string {
	paths := make([]string, 0, len(nodes))
	for _, node := range nodes {
		paths = append(paths, node.Path)
		paths = append(paths, collectMenuTreePaths(node.Children)...)
	}
	return paths
}

func containsMenuPath(paths []string, target string) bool {
	for _, path := range paths {
		if path == target {
			return true
		}
	}
	return false
}
