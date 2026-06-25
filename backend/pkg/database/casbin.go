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
		slog.Error("casbin init error: database not initialized")
		os.Exit(1)
	}

	// 模型 (定义在 system 模块内部或本地配置)
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
		slog.Error("casbin model error", "error", err)
		os.Exit(1)
	}

	adapter, err := NewGormCasbinAdapter(db)
	if err != nil {
		slog.Error("casbin adapter error", "error", err)
		os.Exit(1)
	}

	Enforcer, err = casbin.NewSyncedEnforcer(m, adapter)
	if err != nil {
		slog.Error("casbin enforcer error", "error", err)
		os.Exit(1)
	}
	if err := Enforcer.LoadPolicy(); err != nil {
		slog.Error("casbin load policy error", "error", err)
		os.Exit(1)
	}

	changed := false
	for _, method := range []string{"GET", "POST", "PUT", "PATCH", "DELETE"} {
		added, err := Enforcer.AddPolicy("admin", "/api/v1/*", method)
		if err != nil {
			slog.Error("casbin seed policy error", "error", err)
			os.Exit(1)
		}
		if added {
			changed = true
		}
	}
	if changed {
		if err := Enforcer.LoadPolicy(); err != nil {
			slog.Error("casbin reload policy error", "error", err)
			os.Exit(1)
		}
	}
}
