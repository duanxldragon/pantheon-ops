package contracts

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func TestRequiredModulesFailBeforeAnyRoutes(t *testing.T) {
	for _, mode := range []string{"", "false", "true"} {
		for _, failedStage := range []string{"prepare", "menus", "perms", "i18n"} {
			t.Run(mode+"/"+failedStage, func(t *testing.T) {
				assertRequiredModulesFailBeforeAnyRoutes(t, mode, failedStage)
			})
		}
	}
}

func assertRequiredModulesFailBeforeAnyRoutes(t *testing.T, mode, failedStage string) {
	t.Helper()
	t.Setenv("PANTHEON_AUTO_MIGRATE", mode)
	r := gin.New()
	cause := errors.New("required stage failed")
	stage := func(name string) func(*gorm.DB) error {
		return func(*gorm.DB) error {
			if name == failedStage {
				return cause
			}
			return nil
		}
	}
	registered := 0
	first := FuncModule{ModuleName: "first", Register: func(r *gin.RouterGroup) {
		registered++
		r.GET("/ready", func(c *gin.Context) { c.Status(http.StatusOK) })
	}}
	second := FuncModule{ModuleName: "second", MigrateFunc: stage("prepare"), BootstrapFunc: stage("prepare"),
		SeedMenusFunc: stage("menus"), SeedPermsFunc: stage("perms"), SeedI18nFunc: stage("i18n"),
		Register: func(*gin.RouterGroup) { registered++ },
	}
	err := RegisterRequiredBackendModules(r.Group("/api"), &gorm.DB{}, first, second)
	if !errors.Is(err, cause) {
		t.Fatalf("lost cause: %v", err)
	}
	if registered != 0 || len(r.Routes()) != 0 {
		t.Fatal("routes registered on failed readiness")
	}
	response := httptest.NewRecorder()
	r.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/ready", nil))
	if response.Code != http.StatusNotFound {
		t.Fatal("failed module reported ready")
	}
}

func TestRequiredModulesPhasedExecution(t *testing.T) {
	for _, mode := range []string{"", "true"} {
		t.Run(mode, func(t *testing.T) {
			t.Setenv("PANTHEON_AUTO_MIGRATE", mode)
			steps := []string{}
			err := RegisterRequiredBackendModules(gin.New().Group("/api"), &gorm.DB{},
				orderedModule{"a", &steps}, orderedModule{"b", &steps})
			if err != nil {
				t.Fatal(err)
			}
			prepare := "bootstrap"
			if mode == "true" {
				prepare = "migrate"
			}
			want := []string{"a:" + prepare, "b:" + prepare, "a:menus", "b:menus", "a:perms", "b:perms", "a:i18n", "b:i18n", "a:register", "b:register"}
			if len(steps) != len(want) {
				t.Fatalf("steps: %v", steps)
			}
			for i := range want {
				if steps[i] != want[i] {
					t.Fatalf("steps: %v", steps)
				}
			}
		})
	}
}

func TestRequiredModulesRejectMissingDependencies(t *testing.T) {
	r := gin.New()
	if RegisterRequiredBackendModules(nil, &gorm.DB{}) == nil {
		t.Fatal("nil router accepted")
	}
	if RegisterRequiredBackendModules(r.Group("/api"), nil) == nil {
		t.Fatal("nil database accepted")
	}
	if RegisterRequiredBackendModules(r.Group("/api"), &gorm.DB{}, nil) == nil {
		t.Fatal("nil module accepted")
	}
	var missing *orderedModule
	if RegisterRequiredBackendModules(r.Group("/api"), &gorm.DB{}, missing) == nil {
		t.Fatal("typed nil module accepted")
	}
	if len(r.Routes()) != 0 {
		t.Fatal("registered routes")
	}
}

func TestLegacyModulesStillServeAfterBootstrapFailure(t *testing.T) {
	t.Setenv("PANTHEON_AUTO_MIGRATE", "")
	r := gin.New()
	registered := false
	RegisterBackendModules(r.Group("/api"), &gorm.DB{}, FuncModule{ModuleName: "legacy",
		BootstrapFunc: func(*gorm.DB) error { return errors.New("legacy failure") },
		Register:      func(*gin.RouterGroup) { registered = true },
	})
	if !registered {
		t.Fatal("legacy warning-and-serve behavior changed")
	}
}
