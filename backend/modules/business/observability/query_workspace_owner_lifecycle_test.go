package observability

import (
	"context"
	"crypto/rand"
	"database/sql"
	"errors"
	"fmt"
	"math/big"
	"os"
	"regexp"
	"testing"
	"time"

	"github.com/duanxldragon/pantheon-base/backend/modules/auth/login"
	"github.com/duanxldragon/pantheon-base/backend/pkg/contracts"
	"github.com/duanxldragon/pantheon-base/backend/pkg/contracts/businessauth"
	mysqlDriver "github.com/go-sql-driver/mysql"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Keep the actual acceptance fixtures for readback; never reset a shared DB or
// remove prior failures. Only synthetic authoritative lifecycle columns exist.
func lifecycleMySQL(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("PANTHEON_TEST_DSN")
	if dsn == "" {
		if os.Getenv("PANTHEON_TEST_DSN_REQUIRED") != "" {
			t.Fatal("required lifecycle MySQL reference missing")
		}
		t.Skip("lifecycle MySQL reference missing")
	}
	cfg, err := mysqlDriver.ParseDSN(dsn)
	if err != nil || !regexp.MustCompile(`^[a-z0-9_]+$`).MatchString(cfg.DBName) {
		t.Fatal("invalid lifecycle fixture reference")
	}
	suffix, err := rand.Int(rand.Reader, big.NewInt(10000))
	if err != nil {
		t.Fatal("fixture identity unavailable")
	}
	prefix := cfg.DBName
	if len(prefix) > 27 {
		prefix = prefix[:27]
	}
	name := fmt.Sprintf("%s_testo4_life_%d_%04d", prefix, time.Now().UnixNano(), suffix.Int64())
	if len(name) > 64 || !regexp.MustCompile(`^[a-z0-9_]+$`).MatchString(name) {
		t.Fatal("invalid owned fixture name")
	}
	adminConfig := *cfg
	adminConfig.DBName = ""
	admin, err := sql.Open("mysql", adminConfig.FormatDSN())
	if err != nil {
		t.Fatal("fixture admin unavailable")
	}
	defer admin.Close()
	if _, err := admin.Exec("CREATE DATABASE `" + name + "` CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci"); err != nil {
		t.Fatal("owned lifecycle fixture creation failed")
	}
	fixtureConfig := *cfg
	fixtureConfig.DBName, fixtureConfig.ParseTime = name, true
	db, err := gorm.Open(mysql.Open(fixtureConfig.FormatDSN()), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal("owned fixture unavailable")
	}
	pool, err := db.DB()
	if err != nil {
		t.Fatal("owned fixture pool unavailable")
	}
	pool.SetMaxOpenConns(20)
	t.Cleanup(func() { _ = pool.Close() })
	t.Logf("retained lifecycle fixture: %s", name)
	for _, statement := range []string{
		"CREATE TABLE system_user (id BIGINT UNSIGNED PRIMARY KEY, tenant_id BIGINT UNSIGNED NOT NULL, status INT NOT NULL, deleted_at DATETIME(3) NULL)",
		"CREATE TABLE tenants (id BIGINT UNSIGNED PRIMARY KEY, status VARCHAR(16) NOT NULL)",
		"CREATE TABLE system_setting (id BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT, tenant_id BIGINT UNSIGNED NOT NULL DEFAULT 0, setting_key VARCHAR(128), setting_value TEXT)",
		"INSERT INTO system_user (id,tenant_id,status) VALUES (7,0,1),(8,0,2),(9,11,1)",
		"INSERT INTO tenants (id,status) VALUES (11,'active'),(12,'suspended'),(13,'archived'),(14,'deleted')",
	} {
		if err := db.Exec(statement).Error; err != nil {
			t.Fatal("synthetic lifecycle schema failed")
		}
	}
	if err := db.AutoMigrate(&PrometheusDatasource{}, &QueryHistory{}, &SavedView{}, &QueryOwnerBudget{}); err != nil {
		t.Fatal(err)
	}
	return db
}

func actualOwnerLifecycle(t *testing.T, db *gorm.DB) (businessauth.Reader, businessauth.OwnerLifecycleReader) {
	t.Helper()
	// Use the real public auth composition. No credential repository is supplied
	// or accessed; this optional port counts only actor/tenant lifecycle metadata.
	provider := login.NewBusinessAuthorizationReader(login.NewRuntime(db, nil), nil, nil)
	reader, ok := provider.(businessauth.OwnerLifecycleReader)
	if !ok {
		t.Fatal("reviewed public lifecycle capability missing")
	}
	return provider, reader
}

func lifecycleSource(t *testing.T, db *gorm.DB, tenantID uint64, name string) PrometheusDatasource {
	t.Helper()
	source := PrometheusDatasource{TenantID: tenantID, Name: name, Type: "prometheus", URL: "http://127.0.0.1:1"}
	if err := db.Create(&source).Error; err != nil {
		t.Fatal(err)
	}
	return source
}

func lifecycleHistory(t *testing.T, db *gorm.DB, source PrometheusDatasource, actorID uint64) QueryHistory {
	t.Helper()
	row := QueryHistory{TenantID: source.TenantID, ActorID: actorID, DatasourceID: source.ID, SourceKind: source.Type, Time: []byte(`{"mode":"relative","duration":"15m"}`), Limit: 100, ResultMode: "table", Outcome: "success", ExpiresAt: time.Now().Add(time.Hour)}
	if err := db.Create(&row).Error; err != nil {
		t.Fatal(err)
	}
	return row
}

func TestWorkspaceLifecycleMissingOwnerIsNotIgnored(t *testing.T) {
	db := lifecycleMySQL(t)
	source := lifecycleSource(t, db, 11, "missing-owner")
	row := lifecycleHistory(t, db, source, 999)
	provider, reader := actualOwnerLifecycle(t, db)
	facts, err := reader.ReadOwnerLifecycle(context.Background(), 11, 999)
	if err != nil || !facts.TenantExists || facts.ActorExists {
		t.Fatalf("actual absence control failed: %v", err)
	}
	if err := contracts.RegisterBusinessAuthorizationReader(provider); err != nil {
		t.Fatal(err)
	}
	if err := SweepQueryWorkspace(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	var count int64
	if err := db.Model(&QueryHistory{}).Where("id = ?", row.ID).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatal("reliably absent owner metadata was ignored")
	}
}

func TestWorkspaceLifecycleActualPoolOneStarvationControl(t *testing.T) {
	db := lifecycleMySQL(t)
	_, reader := actualOwnerLifecycle(t, db)
	pool, _ := db.DB()
	pool.SetMaxOpenConns(1)
	if _, err := reader.ReadOwnerLifecycle(context.Background(), 11, 7); err != nil {
		t.Fatal(err)
	}
	tx := db.Begin()
	defer tx.Rollback()
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	started := time.Now()
	facts, err := reader.ReadOwnerLifecycle(ctx, 11, 7)
	if !errors.Is(err, businessauth.ErrOwnerLifecycleUnavailable) || facts != (businessauth.OwnerLifecycle{}) || time.Since(started) > time.Second {
		t.Fatal("actual single-pool lock-time read did not become bounded unknown")
	}
	if err := tx.Rollback().Error; err != nil {
		t.Fatal(err)
	}
	if _, err := reader.ReadOwnerLifecycle(context.Background(), 11, 7); err != nil {
		t.Fatal("transaction connection was not released")
	}
}
