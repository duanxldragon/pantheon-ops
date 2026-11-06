package contracts

import (
	"testing"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type orderedModule struct {
	name  string
	steps *[]string
}

func (m orderedModule) Name() string { return m.name }
func (m orderedModule) Bootstrap(_ *gorm.DB) error {
	*m.steps = append(*m.steps, m.name+":bootstrap")
	return nil
}
func (m orderedModule) Migrate(_ *gorm.DB) error {
	*m.steps = append(*m.steps, m.name+":migrate")
	return nil
}
func (m orderedModule) RegisterRoutes(_ *gin.RouterGroup) {
	*m.steps = append(*m.steps, m.name+":register")
}
func (m orderedModule) SeedMenus(_ *gorm.DB) error {
	*m.steps = append(*m.steps, m.name+":menus")
	return nil
}
func (m orderedModule) SeedPerms(_ *gorm.DB) error {
	*m.steps = append(*m.steps, m.name+":perms")
	return nil
}
func (m orderedModule) SeedI18n(_ *gorm.DB) error {
	*m.steps = append(*m.steps, m.name+":i18n")
	return nil
}

func TestRegisterBackendModules_PhasedExecution(t *testing.T) {
	t.Setenv("PANTHEON_AUTO_MIGRATE", "")
	steps := make([]string, 0, 8)
	r := gin.New()
	api := r.Group("/api/v1")

	RegisterBackendModules(api, nil,
		orderedModule{name: "a", steps: &steps},
		orderedModule{name: "b", steps: &steps},
	)

	expected := []string{
		"a:bootstrap", "b:bootstrap",
		"a:menus", "b:menus",
		"a:perms", "b:perms",
		"a:i18n", "b:i18n",
		"a:register", "b:register",
	}

	if len(steps) != len(expected) {
		t.Fatalf("unexpected steps len: got %d want %d (%v)", len(steps), len(expected), steps)
	}
	for index, item := range expected {
		if steps[index] != item {
			t.Fatalf("unexpected step at %d: got %s want %s", index, steps[index], item)
		}
	}
}

func TestRegisterBackendModules_RunsMigrateWhenAutoMigrateEnabled(t *testing.T) {
	t.Setenv("PANTHEON_AUTO_MIGRATE", "true")
	steps := make([]string, 0, 10)
	r := gin.New()
	api := r.Group("/api/v1")

	RegisterBackendModules(api, nil,
		orderedModule{name: "a", steps: &steps},
		orderedModule{name: "b", steps: &steps},
	)

	expected := []string{
		"a:migrate", "b:migrate",
		"a:menus", "b:menus",
		"a:perms", "b:perms",
		"a:i18n", "b:i18n",
		"a:register", "b:register",
	}

	if len(steps) != len(expected) {
		t.Fatalf("unexpected steps len: got %d want %d (%v)", len(steps), len(expected), steps)
	}
	for index, item := range expected {
		if steps[index] != item {
			t.Fatalf("unexpected step at %d: got %s want %s", index, steps[index], item)
		}
	}
}

// AUD-001c lock: in versioned-migrations mode a module that only sets
// BootstrapFunc must still bootstrap, and a module that only sets MigrateFunc
// must not migrate. The existing tests use orderedModule, which implements both
// Migrate and Bootstrap, so they cannot distinguish "a complete module
// bootstrapped" from "a FuncModule carrying only BootstrapFunc bootstrapped" —
// and the latter is the actual shape of generated low-code modules
// (BootstrapFunc == service.Migrate()). Negative A (narrowing FuncModule.Bootstrap
// to require a non-nil MigrateFunc) fails ONLY this test; deleting the whole
// bootstrap phase also fails TestRegisterBackendModules_PhasedExecution
// (negative B), so this test is not the sole gatekeeper of that phase.
func TestRegisterBackendModules_DefaultModeRunsBootstrapForGeneratedModules(t *testing.T) {
	t.Setenv("PANTHEON_AUTO_MIGRATE", "")
	steps := make([]string, 0, 4)
	r := gin.New()
	api := r.Group("/api/v1")

	RegisterBackendModules(api, nil,
		FuncModule{
			ModuleName: "bootstrap-only",
			BootstrapFunc: func(_ *gorm.DB) error {
				steps = append(steps, "bootstrap-only:bootstrap")
				return nil
			},
		},
		FuncModule{
			ModuleName: "migrate-only",
			MigrateFunc: func(_ *gorm.DB) error {
				steps = append(steps, "migrate-only:migrate")
				return nil
			},
		},
	)

	for _, step := range steps {
		if step == "migrate-only:migrate" {
			t.Fatalf("module Migrate() must be skipped in versioned mode; steps=%v", steps)
		}
	}
	if len(steps) != 1 || steps[0] != "bootstrap-only:bootstrap" {
		t.Fatalf("a module that only sets BootstrapFunc must still bootstrap in versioned mode; otherwise low-code generated tables are never created (DEPLOYMENT_GUIDE §3): steps=%v", steps)
	}
}
