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
