package platform

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/duanxldragon/pantheon-base/backend/pkg/testmysql"

	"github.com/gin-gonic/gin"
)

// Task 2026-09-22-production-redis-and-security-gates: readiness must
// reflect migration state, not only process liveness. With a migrated
// database the `migrations` dependency reports ok; on an empty/unmigrated
// database (schema_migrations absent or empty) the endpoint degrades with a
// stable public message.

func TestHealth_MigrationsOkAfterMigration(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := testmysql.Open(t)
	// testmysql.Open creates an empty per-test database; create the
	// migrations table the same way RunMigrations (golang-migrate) would.
	if err := db.Exec("CREATE TABLE IF NOT EXISTS schema_migrations (version bigint NOT NULL, dirty tinyint(1) NOT NULL DEFAULT 0, PRIMARY KEY (version))").Error; err != nil {
		t.Fatalf("create schema_migrations: %v", err)
	}
	if err := db.Exec("INSERT INTO schema_migrations (version, dirty) VALUES (1, 0)").Error; err != nil {
		t.Fatalf("seed schema_migrations: %v", err)
	}

	engine := gin.New()
	RegisterHealthRoutes(engine.Group("/api/v1"), db)
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/health", nil))

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected healthy status with migrations applied, got %d: %s", recorder.Code, recorder.Body.String())
	}
	resp := decodeHealthTestResponse(t, recorder)
	if resp.Data.Dependencies["migrations"].Status != "ok" {
		t.Fatalf("expected migrations dependency ok, got %+v", resp.Data.Dependencies["migrations"])
	}
}

func TestHealth_MigrationsIncompleteDegradesReadiness(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := testmysql.Open(t)
	// No schema_migrations table: the database is up but migrations never ran.

	engine := gin.New()
	RegisterHealthRoutes(engine.Group("/api/v1"), db)
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/health", nil))

	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 while migrations are incomplete, got %d", recorder.Code)
	}
	resp := decodeHealthTestResponse(t, recorder)
	dependency := resp.Data.Dependencies["migrations"]
	if dependency.Message != "migrations.incomplete" {
		t.Fatalf("expected stable migrations message, got %q", dependency.Message)
	}
	if dependency.Status != "down" {
		t.Fatalf("expected migrations dependency down, got %+v", dependency)
	}
}
