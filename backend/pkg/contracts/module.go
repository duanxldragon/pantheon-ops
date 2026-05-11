package contracts

import (
	"log"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type BackendModule interface {
	Name() string
	Migrate(db *gorm.DB) error
	RegisterRoutes(r *gin.RouterGroup)
	SeedMenus(db *gorm.DB) error
	SeedPerms(db *gorm.DB) error
	SeedI18n(db *gorm.DB) error
}

type FuncModule struct {
	ModuleName    string
	MigrateFunc   func(db *gorm.DB) error
	Register      func(r *gin.RouterGroup)
	SeedMenusFunc func(db *gorm.DB) error
	SeedPermsFunc func(db *gorm.DB) error
	SeedI18nFunc  func(db *gorm.DB) error
}

func (m FuncModule) Name() string {
	return m.ModuleName
}

func (m FuncModule) Migrate(db *gorm.DB) error {
	if m.MigrateFunc == nil {
		return nil
	}
	return m.MigrateFunc(db)
}

func (m FuncModule) RegisterRoutes(r *gin.RouterGroup) {
	if m.Register != nil {
		m.Register(r)
	}
}

func (m FuncModule) SeedMenus(db *gorm.DB) error {
	if m.SeedMenusFunc == nil {
		return nil
	}
	return m.SeedMenusFunc(db)
}

func (m FuncModule) SeedPerms(db *gorm.DB) error {
	if m.SeedPermsFunc == nil {
		return nil
	}
	return m.SeedPermsFunc(db)
}

func (m FuncModule) SeedI18n(db *gorm.DB) error {
	if m.SeedI18nFunc == nil {
		return nil
	}
	return m.SeedI18nFunc(db)
}

func RegisterBackendModules(r *gin.RouterGroup, db *gorm.DB, modules ...BackendModule) {
	for _, module := range modules {
		runModuleSeed(module.Name(), "migrate", db, module.Migrate)
	}
	for _, module := range modules {
		runModuleSeed(module.Name(), "menus", db, module.SeedMenus)
	}
	for _, module := range modules {
		runModuleSeed(module.Name(), "perms", db, module.SeedPerms)
	}
	for _, module := range modules {
		runModuleSeed(module.Name(), "i18n", db, module.SeedI18n)
	}
	for _, module := range modules {
		module.RegisterRoutes(r)
	}
}

func runModuleSeed(moduleName string, seedType string, db *gorm.DB, seedFunc func(*gorm.DB) error) {
	if seedFunc == nil {
		return
	}
	if err := seedFunc(db); err != nil {
		log.Printf("%s module seed %s error: %v", moduleName, seedType, err)
	}
}
