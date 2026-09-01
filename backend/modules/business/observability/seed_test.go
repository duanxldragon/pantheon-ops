package observability

import (
	"testing"
	"time"

	"pantheon-base/pkg/testmysql"

	"gorm.io/gorm"
)

type observabilitySeedTestMenu struct {
	ID        uint64 `gorm:"primaryKey;autoIncrement"`
	ParentID  uint64
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
	IsVisible bool
	IsCache   bool
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt
}

func (observabilitySeedTestMenu) TableName() string { return "system_menu" }

type observabilitySeedTestRole struct {
	ID      uint64 `gorm:"primaryKey;autoIncrement"`
	RoleKey string
}

func (observabilitySeedTestRole) TableName() string { return "system_role" }

type observabilitySeedTestRoleMenu struct {
	RoleID uint64
	MenuID uint64
}

func (observabilitySeedTestRoleMenu) TableName() string { return "system_role_menu" }

type observabilitySeedTestRolePermission struct {
	RoleID        uint64
	PermissionKey string
}

func (observabilitySeedTestRolePermission) TableName() string { return "system_role_permission" }

type observabilitySeedTestI18n struct {
	ID              uint64 `gorm:"primaryKey;autoIncrement"`
	Module          string
	Locale          string
	GroupName       string
	Key             string
	Value           string
	LifecycleStatus string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

func (observabilitySeedTestI18n) TableName() string { return "system_i18n" }

func TestSeedMenusAndI18nIsIdempotent(t *testing.T) {
	db := testmysql.Open(t)
	if err := db.AutoMigrate(
		&observabilitySeedTestMenu{},
		&observabilitySeedTestRole{},
		&observabilitySeedTestRoleMenu{},
		&observabilitySeedTestRolePermission{},
		&observabilitySeedTestI18n{},
	); err != nil {
		t.Fatalf("migrate seed fixtures: %v", err)
	}
	if err := db.Create(&observabilitySeedTestRole{ID: 1, RoleKey: "admin"}).Error; err != nil {
		t.Fatalf("create admin role: %v", err)
	}
	if err := SeedMenus(db); err != nil {
		t.Fatalf("first seed: %v", err)
	}
	if err := SeedMenus(db); err != nil {
		t.Fatalf("second seed: %v", err)
	}

	assertSeedCount(t, db, "system_menu", "module = ? AND type = ?", []any{observabilityModuleKey, "C"}, 4)
	assertSeedCount(t, db, "system_role_menu", "role_id = ?", []any{uint64(1)}, 4)
	assertSeedCount(t, db, "system_role_permission", "role_id = ?", []any{uint64(1)}, 11)
	assertSeedCount(t, db, "system_i18n", "module = ?", []any{observabilityModuleKey}, 24)
	assertSeedCount(t, db, "system_i18n", "module = ? AND `key` = ?", []any{observabilityModuleKey, "operations.observability.logs.menu"}, 2)
}

func assertSeedCount(t *testing.T, db *gorm.DB, table, where string, args []any, want int64) {
	t.Helper()
	var count int64
	if err := db.Table(table).Where(where, args...).Count(&count).Error; err != nil {
		t.Fatalf("count %s: %v", table, err)
	}
	if count != want {
		t.Fatalf("expected %d rows in %s, got %d", want, table, count)
	}
}
