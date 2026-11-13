package business

import (
	"errors"
	"testing"

	"github.com/duanxldragon/pantheon-base/backend/pkg/contracts"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func newBusinessPreflightRetirementFixture(t *testing.T) *gorm.DB {
	t.Helper()
	db := openRetiredModuleTestDB(t)
	mustCreateRetiredModuleGovernanceTables(t, db)
	menu := mustInsertRetiredMenuRow(t, db, "/test/preflight", "test.preflight", "", "", "")
	mustInsertRetiredRoleMenu(t, db, menu)
	mustInsertRetiredPermission(t, db, "test:preflight:view")
	mustInsertRetiredI18n(t, db, "test.preflight")
	mustInsertManagedRegistration(t, db, "test.preflight", "", 1)
	mustCreateRetiredModuleBusinessTable(t, db, "biz_preflight_control")
	if err := db.Exec("INSERT INTO biz_preflight_control (name) VALUES ('synthetic retained control')").Error; err != nil {
		t.Fatal(err)
	}
	previous := retiredBusinessModules
	retiredBusinessModules = []retiredModuleSpec{{
		ModuleNames: []string{"test.preflight"}, PermissionPrefixes: []string{"test:preflight:"},
		MenuPaths: []string{"/test/preflight"},
	}}
	t.Cleanup(func() { retiredBusinessModules = previous })
	return db
}

func checkBusinessPreflightRetirementRows(t *testing.T, db *gorm.DB, want int64) {
	t.Helper()
	for _, table := range []string{"system_menu", "system_role_menu", "system_role_permission", "system_i18n", "system_module_registration"} {
		var count int64
		if err := db.Table(table).Count(&count).Error; err != nil {
			t.Fatal(err)
		}
		if count != want {
			t.Errorf("retirement effects in %s: wanted %d retained rows, got %d", table, want, count)
		}
	}
	assertRecordCount(t, db, "biz_preflight_control", "1 = 1", 1)
}

func TestBusinessInitializationPreflightFailsBeforeCleanup(t *testing.T) {
	scopedBusinessPreflight(t)
	db := newBusinessPreflightRetirementFixture(t)
	pool, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	pool.SetMaxOpenConns(1)
	cause := errors.New("consumer fixed readiness refusal")
	calls := 0
	if err := RegisterBusinessInitializationPreflight(func(input *gorm.DB) error {
		calls++
		if input != db || pool.Stats().InUse != 0 {
			return errors.New("preflight ran on a cloned DB or inside a transaction")
		}
		return cause
	}); err != nil {
		t.Fatal(err)
	}
	router := gin.New()
	got := capturedBusinessPreflightPanic(func() { InitBusinessModules(router.Group("/api"), db) })
	if got != cause || calls != 1 {
		t.Error("standard business entry must run readiness and preserve its original error")
	}
	checkBusinessPreflightRetirementRows(t, db, 1)
	if len(router.Routes()) != 0 {
		t.Fatal("failed preflight exposed business routes")
	}
}

func TestBusinessInitializationPreflightRejectsBeforeInvalidDBCleanup(t *testing.T) {
	for _, name := range []string{"nil", "invalid", "closed"} {
		t.Run(name, func(t *testing.T) {
			scopedBusinessPreflight(t)
			var db *gorm.DB
			switch name {
			case "invalid":
				db = &gorm.DB{}
			case "closed":
				db = openRetiredModuleTestDB(t)
				pool, err := db.DB()
				if err != nil {
					t.Fatal(err)
				}
				if err := pool.Close(); err != nil {
					t.Fatal(err)
				}
			}
			cause := errors.New("consumer safe dependency error")
			if err := RegisterBusinessInitializationPreflight(func(input *gorm.DB) error {
				if input != db {
					t.Fatal("preflight changed the consumer's DB handle")
				}
				return cause
			}); err != nil {
				t.Fatal(err)
			}
			got := capturedBusinessPreflightPanic(func() { InitBusinessModules(gin.New().Group("/api"), db) })
			if got != cause {
				t.Fatal("cleanup replaced the preflight's safe error with an earlier DB failure")
			}
		})
	}
}

func TestBusinessInitializationPreflightStandaloneCompatibility(t *testing.T) {
	for _, name := range []string{"nil_db", "healthy_cleanup"} {
		t.Run(name, func(t *testing.T) {
			scopedBusinessPreflight(t)
			var db *gorm.DB
			if name == "healthy_cleanup" {
				db = newBusinessPreflightRetirementFixture(t)
			}
			if capturedBusinessPreflightPanic(func() { initializeBusinessLifecycle(db) }) != nil {
				t.Fatal("standalone Base lifecycle behavior changed")
			}
			if db != nil {
				checkBusinessPreflightRetirementRows(t, db, 0)
			}
			if RegisterBusinessInitializationPreflight(func(*gorm.DB) error { return nil }) != contracts.ErrBusinessProviders {
				t.Fatal("standalone lifecycle did not seal registration")
			}
		})
	}
}

func TestBusinessInitializationPreflightRechecksBeforeRepeatCleanup(t *testing.T) {
	scopedBusinessPreflight(t)
	db := newBusinessPreflightRetirementFixture(t)
	cause := errors.New("consumer readiness changed after first initialization")
	calls := 0
	if err := RegisterBusinessInitializationPreflight(func(*gorm.DB) error {
		calls++
		if calls == 2 {
			return cause
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	initializeBusinessLifecycle(db)
	checkBusinessPreflightRetirementRows(t, db, 0)
	mustInsertRetiredPermission(t, db, "test:preflight:later")
	if got := capturedBusinessPreflightPanic(func() { initializeBusinessLifecycle(db) }); got != cause || calls != 2 {
		t.Fatal("Base lifecycle cached the first readiness result")
	}
	assertRecordCount(t, db, "system_role_permission", "1 = 1", 1)
	assertRecordCount(t, db, "biz_preflight_control", "1 = 1", 1)
}
