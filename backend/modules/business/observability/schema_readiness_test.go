package observability

import (
	"testing"

	"github.com/duanxldragon/pantheon-base/backend/pkg/testmysql"
)

func TestQueryWorkspaceExactSchemaReadiness(t *testing.T) {
	db := testmysql.Open(t)
	t.Setenv("PANTHEON_AUTO_MIGRATE", "true")
	if err := QueryWorkspaceReady(db); err == nil {
		t.Fatal("missing schema accepted")
	}
	if err := BootstrapQueryWorkspaceDevelopment(db); err != nil {
		for table := range queryWorkspaceShapes() {
			columns, indexes, _ := inspectQueryTable(db, table)
			t.Logf("test-owned shape %s: %+v %+v", table, columns, indexes)
		}
		t.Fatal(err)
	}
	if err := QueryWorkspaceReady(db); err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("ALTER TABLE obs_saved_view DROP INDEX idx_view_owner").Error; err != nil {
		t.Fatal(err)
	}
	if err := QueryWorkspaceReady(db); err == nil {
		t.Fatal("marker masked missing index")
	}
	if err := db.Exec("ALTER TABLE obs_saved_view ADD INDEX idx_view_owner (tenant_id,actor_id,updated_at,id)").Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("CREATE TABLE schema_migrations (version BIGINT NOT NULL PRIMARY KEY,dirty BOOLEAN NOT NULL)").Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("INSERT INTO schema_migrations VALUES (37,true)").Error; err != nil {
		t.Fatal(err)
	}
	if err := QueryWorkspaceReady(db); err == nil {
		t.Fatal("dirty state accepted")
	}
	if err := BootstrapQueryWorkspaceDevelopment(db); err == nil {
		t.Fatal("dev bypassed dirty state")
	}
}
