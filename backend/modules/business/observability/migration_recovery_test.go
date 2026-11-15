package observability

import (
	"context"
	"strings"
	"testing"

	"github.com/duanxldragon/pantheon-base/backend/pkg/testmysql"
)

func TestO4_LegacyCiphertextAndIndexUpgrade(t *testing.T) {
	db := testmysql.Open(t)
	ctx := context.Background()
	sqlDB, _ := db.DB()
	conn, err := sqlDB.Conn(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	up, _ := o4MigrationFiles.ReadFile("migrations/000037_o4_business_tenant_workspace.up.sql")
	normalized := strings.ReplaceAll(string(up), "\r\n", "\n")
	parts := strings.SplitN(normalized, "-- O4 STEP datasource\n", 2)
	if len(parts) != 2 {
		t.Fatal("reviewed datasource step missing")
	}
	after := parts[1]
	create := strings.SplitN(after, ";\n", 2)[0]
	var kept []string
	for _, line := range strings.Split(create, "\n") {
		if strings.Contains(line, "`tenant_id`") || strings.Contains(line, "`revision`") || strings.Contains(line, "idx_datasource_tenant_") {
			continue
		}
		kept = append(kept, line)
	}
	legacy := strings.Join(kept, "\n")
	legacy = strings.Replace(legacy, "  PRIMARY KEY", "  UNIQUE KEY idx_obs_prometheus_datasource_name (name),\n  PRIMARY KEY", 1)
	if _, err := conn.ExecContext(ctx, legacy); err != nil {
		t.Fatal(err)
	}
	if err := validateQueryTable(db, "obs_prometheus_datasource", legacyDatasourceShape(), false); err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("INSERT INTO obs_prometheus_datasource (id,name,type,url,password,bearer_token) VALUES (42,'legacy-source','prometheus','http://127.0.0.1:1','retained-encrypted-password','retained-encrypted-token')").Error; err != nil {
		t.Fatal(err)
	}
	before, err := DatasourceRetentionSHA256(db)
	if err != nil {
		t.Fatal(err)
	}
	if err := executeO4SQL(ctx, conn, up); err != nil {
		t.Fatal(err)
	}
	afterHash, err := DatasourceRetentionSHA256(db)
	if err != nil || afterHash != before {
		t.Fatal("legacy data/ciphertext changed")
	}
	var row PrometheusDatasource
	if err := db.First(&row, 42).Error; err != nil || row.TenantID != 0 || row.Revision != 1 {
		t.Fatal("legacy ownership inferred")
	}
	if err := QueryWorkspaceReady(db); err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&PrometheusDatasource{TenantID: 11, Name: row.Name, Type: "prometheus", URL: "http://127.0.0.1:1"}).Error; err != nil {
		t.Fatal("known global index survived")
	}
}
func TestO4_FreshAndRetainedMigrationRecipe(t *testing.T) {
	db := testmysql.Open(t)
	ctx := context.Background()
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	conn, err := sqlDB.Conn(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	up, err := o4MigrationFiles.ReadFile("migrations/000037_o4_business_tenant_workspace.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	if err := executeO4SQL(ctx, conn, up); err != nil {
		t.Fatal(err)
	}
	if err := QueryWorkspaceReady(db); err != nil {
		t.Fatal(err)
	}
	for _, tenantID := range []uint64{0, 11, 22} {
		if err := db.Create(&PrometheusDatasource{TenantID: tenantID, Name: "same-name", Type: "prometheus", URL: "http://127.0.0.1:1", Password: "retained-ciphertext", BearerToken: "retained-token-ciphertext"}).Error; err != nil {
			t.Fatal(err)
		}
	}
	before, err := DatasourceRetentionSHA256(db)
	if err != nil {
		t.Fatal(err)
	}
	if err := executeO4SQL(ctx, conn, up); err != nil {
		t.Fatal(err)
	}
	down, _ := o4MigrationFiles.ReadFile("migrations/000037_o4_business_tenant_workspace.down.sql")
	if err := executeO4SQL(ctx, conn, down); err != nil {
		t.Fatal(err)
	}
	if err := QueryWorkspaceReady(db); err != nil {
		t.Fatal("retained-down lost capability/shape")
	}
	after, err := DatasourceRetentionSHA256(db)
	if err != nil || after != before {
		t.Fatal("legacy data/ciphertext changed")
	}
}
func TestO4_KnownPartialRecoveryAndUnknownRefusal(t *testing.T) {
	for _, step := range []string{"-- O4 STEP revision", "-- O4 STEP index idx_datasource_tenant_default", "-- O4 STEP obs_saved_view"} {
		t.Run(strings.TrimPrefix(step, "-- O4 STEP "), func(t *testing.T) {
			db := testmysql.Open(t)
			ctx := context.Background()
			sqlDB, _ := db.DB()
			conn, err := sqlDB.Conn(ctx)
			if err != nil {
				t.Fatal(err)
			}
			defer conn.Close()
			up, _ := o4MigrationFiles.ReadFile("migrations/000037_o4_business_tenant_workspace.up.sql")
			// Deliberately fail after a real subset of known DDL. Official migrator
			// fault/restart is additionally tested in the assembled paired runner.
			prefix := strings.SplitN(string(up), step, 2)[0]
			if err := executeO4SQL(ctx, conn, []byte(prefix)); err != nil {
				t.Fatal(err)
			}
			if err := db.Exec("CREATE TABLE schema_migrations (version BIGINT NOT NULL PRIMARY KEY,dirty BOOLEAN NOT NULL)").Error; err != nil {
				t.Fatal(err)
			}
			if err := db.Exec("INSERT INTO schema_migrations VALUES (37,true)").Error; err != nil {
				t.Fatal(err)
			}
			var name string
			db.Raw("SELECT DATABASE()").Scan(&name)
			before, err := DatasourceRetentionSHA256(db)
			if err != nil {
				t.Fatal(err)
			}
			ownership := O4RecoveryOwnership{Database: name, BeforeDatasourceSHA256: before, ExclusiveWritersStopped: true, QuiescenceReceiptSHA256: strings.Repeat("a", 64)}
			if _, err := RecoverO4Migration37(ctx, db, ownership); err != nil {
				t.Fatal(err)
			}
			if err := QueryWorkspaceReady(db); err != nil {
				t.Fatal(err)
			}
			if err := db.Exec("UPDATE schema_migrations SET dirty=true").Error; err != nil {
				t.Fatal(err)
			}
			if err := db.Exec("ALTER TABLE obs_saved_view ADD COLUMN unexpected VARCHAR(8)").Error; err != nil {
				t.Fatal(err)
			}
			if _, err := RecoverO4Migration37(ctx, db, ownership); err == nil {
				t.Fatal("unknown partial shape cleared dirty")
			}
			var dirty bool
			db.Raw("SELECT dirty FROM schema_migrations WHERE version=37").Scan(&dirty)
			if !dirty {
				t.Fatal("dirty evidence overwritten")
			}
		})
	}
}
