package login

import (
	"strings"

	"gorm.io/gorm"
)

type menuSeed struct {
	Key        string
	ParentKey  string
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
	IsCache    int
	IsExternal int
	ActiveMenu string
}

func SeedAuthModuleMenus(db *gorm.DB) error {
	return ensureMenuSeeds(db, authMenuSeeds())
}

func authMenuSeeds() []menuSeed {
	return []menuSeed{
		{
			Key:       "login-log",
			ParentKey: "security",
			TitleKey:  "system.menu.loginLog",
			Path:      "/system/login-log",
			Component: "auth/LoginLogList",
			PagePerm:  "system:login-log:list",
			Type:      "C",
			Icon:      "clock",
			RouteName: "system-login-log",
			Module:    "system.auth",
			Sort:      10,
		},
		{
			Key:       "session",
			ParentKey: "security",
			TitleKey:  "system.menu.session",
			Path:      "/system/session",
			Component: "auth/SessionList",
			PagePerm:  "system:session:list",
			Type:      "C",
			Icon:      "desktop",
			RouteName: "system-session",
			Module:    "system.auth",
			Sort:      20,
		},
		{
			Key:       "security-event",
			ParentKey: "security",
			TitleKey:  "system.menu.securityEvent",
			Path:      "/system/security-event",
			Component: "auth/SecurityEventList",
			PagePerm:  "system:security-event:list",
			Type:      "C",
			Icon:      "safe",
			RouteName: "system-security-event",
			Module:    "system.auth",
			Sort:      30,
		},
		{Key: "login-log-export", ParentKey: "login-log", TitleKey: "system.permission.login_log.export", Perms: "system:login-log:export", Type: "F", Sort: 1},
		{Key: "login-log-clear", ParentKey: "login-log", TitleKey: "system.permission.login_log.clear", Perms: "system:login-log:clear", Type: "F", Sort: 2},
		{Key: "login-log-delete", ParentKey: "login-log", TitleKey: "system.permission.login_log.delete", Perms: "system:login-log:delete", Type: "F", Sort: 3},
		{Key: "session-delete", ParentKey: "session", TitleKey: "system.permission.session.delete", Perms: "system:session:delete", Type: "F", Sort: 1},
		{Key: "session-clear", ParentKey: "session", TitleKey: "system.permission.session.clear", Perms: "system:session:clear", Type: "F", Sort: 2},
		{Key: "security-event-acknowledge", ParentKey: "security-event", TitleKey: "system.permission.security_event.acknowledge", Perms: "system:security-event:acknowledge", Type: "F", Sort: 1},
	}
}

func ensureMenuSeeds(db *gorm.DB, seeds []menuSeed) error {
	if db == nil {
		return nil
	}
	for _, seed := range seeds {
		if err := ensureSingleMenuSeed(db, seed); err != nil {
			return err
		}
	}
	return nil
}

func ensureSingleMenuSeed(db *gorm.DB, seed menuSeed) error {
	parentID, err := resolveMenuParentID(db, seed.ParentKey)
	if err != nil {
		return err
	}
	menuID, err := upsertSeedMenu(db, seed, parentID)
	if err != nil {
		return err
	}
	if menuID == 0 {
		return nil
	}
	return ensureAdminRoleMenuBinding(db, menuID)
}

func upsertSeedMenu(db *gorm.DB, seed menuSeed, parentID uint64) (uint64, error) {
	menuID, err := lookupSeedMenuID(db, seed)
	if err != nil || menuID == 0 {
		if err != nil {
			return 0, err
		}
		if err := db.Table("system_menu").Create(buildSeedMenuPayload(seed, parentID)).Error; err != nil {
			return 0, err
		}
		return lookupSeedMenuID(db, seed)
	}

	if err := db.Table("system_menu").Where("id = ?", menuID).Updates(buildSeedMenuPayload(seed, parentID)).Error; err != nil {
		return 0, err
	}
	return menuID, nil
}

func lookupSeedMenuID(db *gorm.DB, seed menuSeed) (uint64, error) {
	var menuID uint64
	switch {
	case seed.Path != "":
		err := db.Table("system_menu").Select("id").Where("path = ?", seed.Path).Limit(1).Pluck("id", &menuID).Error
		return menuID, err
	case seed.Perms != "":
		err := db.Table("system_menu").Select("id").Where("perms = ?", seed.Perms).Limit(1).Pluck("id", &menuID).Error
		return menuID, err
	default:
		return 0, nil
	}
}

func buildSeedMenuPayload(seed menuSeed, parentID uint64) map[string]interface{} {
	return map[string]interface{}{
		"parent_id":   parentID,
		"title_key":   seed.TitleKey,
		"path":        seed.Path,
		"component":   seed.Component,
		"page_perm":   seed.PagePerm,
		"perms":       seed.Perms,
		"type":        normalizeSeedMenuType(seed.Type),
		"icon":        seed.Icon,
		"route_name":  strings.TrimSpace(seed.RouteName),
		"module":      normalizeSeedMenuModule(seed.Module),
		"sort":        seed.Sort,
		"is_visible":  1,
		"is_cache":    normalizeSeedMenuFlag(seed.IsCache),
		"is_external": normalizeSeedMenuFlag(seed.IsExternal),
		"active_menu": strings.TrimSpace(seed.ActiveMenu),
	}
}

func ensureAdminRoleMenuBinding(db *gorm.DB, menuID uint64) error {
	adminRoleID, err := lookupAdminRoleID(db)
	if err != nil || adminRoleID == 0 {
		return err
	}

	var count int64
	if err := db.Table("system_role_menu").Where("role_id = ? AND menu_id = ?", adminRoleID, menuID).Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return db.Exec("INSERT INTO system_role_menu (role_id, menu_id) VALUES (?, ?)", adminRoleID, menuID).Error
	}
	return nil
}

func lookupAdminRoleID(db *gorm.DB) (uint64, error) {
	var adminRoleID uint64
	if err := db.Table("system_role").Select("id").Where("role_key = ?", "admin").Limit(1).Pluck("id", &adminRoleID).Error; err != nil {
		return 0, err
	}
	return adminRoleID, nil
}

func resolveMenuParentID(db *gorm.DB, parentKey string) (uint64, error) {
	if parentKey == "" {
		return 0, nil
	}
	parentPaths := map[string]string{
		"security":       "/system/security",
		"login-log":      "/system/login-log",
		"session":        "/system/session",
		"security-event": "/system/security-event",
	}
	parentPath, ok := parentPaths[parentKey]
	if !ok {
		return 0, nil
	}
	return lookupMenuIDByPath(db, parentPath)
}

func lookupMenuIDByPath(db *gorm.DB, path string) (uint64, error) {
	var menuID uint64
	if err := db.Table("system_menu").Select("id").Where("path = ?", path).Limit(1).Pluck("id", &menuID).Error; err != nil {
		return 0, err
	}
	return menuID, nil
}

func normalizeSeedMenuType(value string) string {
	switch value {
	case "M", "C", "F":
		return value
	default:
		return "C"
	}
}

func normalizeSeedMenuModule(value string) string {
	if strings.TrimSpace(value) == "" {
		return "system"
	}
	return strings.TrimSpace(value)
}

func normalizeSeedMenuFlag(value int) int {
	if value == 1 {
		return 1
	}
	return 0
}
