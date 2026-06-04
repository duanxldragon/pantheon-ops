package contracts

import (
	"testing"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"pantheon-platform/backend/pkg/testmysql"
)

type orderedModule struct {
	name  string
	steps *[]string
}

func (m orderedModule) Name() string { return m.name }
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
	db := testmysql.Open(t)

	steps := make([]string, 0, 10)
	r := gin.New()
	api := r.Group("/api/v1")

	RegisterBackendModules(api, db,
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
