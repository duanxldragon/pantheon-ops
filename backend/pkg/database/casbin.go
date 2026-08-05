package database

import (
	"log/slog"
	"os"

	"github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/model"
	"gorm.io/gorm"
)

var Enforcer *casbin.SyncedEnforcer

// InitCasbin 初始化权限引擎
func InitCasbin(db *gorm.DB) {
	if db == nil {
		exitCasbinInit("casbin init error: database not initialized", nil)
	}
	setCasbinWatcher(nil)

	m := loadCasbinModel()
	adapter, err := NewGormCasbinAdapter(db)
	if err != nil {
		exitCasbinInit("casbin adapter error", err)
	}

	Enforcer, err = casbin.NewSyncedEnforcer(m, adapter)
	if err != nil {
		exitCasbinInit("casbin enforcer error", err)
	}
	loadCasbinPolicyOrExit("casbin load policy error")
	seedAdminCasbinPolicies()
	configureCasbinWatcher()
}

func loadCasbinModel() model.Model {
	m, err := model.NewModelFromString(`
		[request_definition]
		r = sub, obj, act
		[policy_definition]
		p = sub, obj, act
		[role_definition]
		g = _, _
		[policy_effect]
		e = some(where (p.eft == allow))
		[matchers]
		m = (r.sub == p.sub || g(r.sub, p.sub)) && keyMatch2(r.obj, p.obj) && r.act == p.act
	`)
	if err != nil {
		exitCasbinInit("casbin model error", err)
	}
	return m
}

func seedAdminCasbinPolicies() {
	changed := false
	for _, method := range []string{"GET", "POST", "PUT", "PATCH", "DELETE"} {
		added, err := Enforcer.AddPolicy("admin", "/api/v1/*", method)
		if err != nil {
			exitCasbinInit("casbin seed policy error", err)
		}
		changed = changed || added
	}
	if changed {
		loadCasbinPolicyOrExit("casbin reload policy error")
	}
}

func loadCasbinPolicyOrExit(message string) {
	if err := Enforcer.LoadPolicy(); err != nil {
		exitCasbinInit(message, err)
	}
}

func configureCasbinWatcher() {
	watcher := initCasbinWatcher(RDB)
	if watcher == nil {
		return
	}
	if err := Enforcer.SetWatcher(watcher); err != nil {
		slog.Warn("casbin watcher init failed", "error", err)
		watcher.Close()
		return
	}
	setCasbinWatcher(watcher)
}

func exitCasbinInit(message string, err error) {
	if err == nil {
		slog.Error(message)
	} else {
		slog.Error(message, "error", err)
	}
	os.Exit(1)
}
