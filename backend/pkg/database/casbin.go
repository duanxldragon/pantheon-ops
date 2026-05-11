package database

import (
	"log"

	"github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/model"
	"gorm.io/gorm"
)

var Enforcer *casbin.SyncedEnforcer

// InitCasbin 初始化权限引擎
func InitCasbin(db *gorm.DB) {
	if db == nil {
		log.Fatalf("casbin init error: database not initialized")
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
		log.Fatalf("casbin model error: %v", err)
	}

	adapter, err := NewGormCasbinAdapter(db)
	if err != nil {
		log.Fatalf("casbin adapter error: %v", err)
	}

	Enforcer, err = casbin.NewSyncedEnforcer(m, adapter)
	if err != nil {
		log.Fatalf("casbin enforcer error: %v", err)
	}
	if err := Enforcer.LoadPolicy(); err != nil {
		log.Fatalf("casbin load policy error: %v", err)
	}

	changed := false
	for _, method := range []string{"GET", "POST", "PUT", "PATCH", "DELETE"} {
		added, err := Enforcer.AddPolicy("admin", "/api/v1/*", method)
		if err != nil {
			log.Fatalf("casbin seed policy error: %v", err)
		}
		if added {
			changed = true
		}
	}
	if changed {
		if err := Enforcer.LoadPolicy(); err != nil {
			log.Fatalf("casbin reload policy error: %v", err)
		}
	}
}
