package testmysql

import (
	"database/sql"
	"fmt"
	"math/rand"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	mysqlDriver "github.com/go-sql-driver/mysql"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

func Open(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := strings.TrimSpace(os.Getenv("PANTHEON_TEST_DSN"))
	if dsn == "" {
		dsn = strings.TrimSpace(os.Getenv("PANTHEON_DSN"))
	}
	if dsn == "" {
		t.Skip("mysql test dsn is not configured")
	}

	cfg, err := mysqlDriver.ParseDSN(dsn)
	if err != nil {
		t.Fatalf("parse mysql dsn: %v", err)
	}
	if strings.TrimSpace(cfg.DBName) == "" {
		t.Fatalf("mysql test dsn must include database name")
	}

	adminCfg := *cfg
	adminCfg.DBName = ""
	adminDB, err := sql.Open("mysql", adminCfg.FormatDSN())
	if err != nil {
		t.Fatalf("open mysql admin connection: %v", err)
	}
	t.Cleanup(func() { _ = adminDB.Close() })

	testDBName := buildTestDBName(cfg.DBName, t.Name())
	if _, err := adminDB.Exec(fmt.Sprintf("CREATE DATABASE `%s` CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci", testDBName)); err != nil {
		t.Fatalf("create test database %s: %v", testDBName, err)
	}
	t.Cleanup(func() {
		_, _ = adminDB.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS `%s`", testDBName))
	})

	testCfg := *cfg
	testCfg.DBName = testDBName
	testCfg.ParseTime = true
	if testCfg.Loc == nil {
		testCfg.Loc = time.Local
	}
	if testCfg.Params == nil {
		testCfg.Params = map[string]string{}
	}
	if _, ok := testCfg.Params["charset"]; !ok {
		testCfg.Params["charset"] = "utf8mb4"
	}

	db, err := gorm.Open(mysql.Open(testCfg.FormatDSN()), &gorm.Config{
		NamingStrategy: schema.NamingStrategy{SingularTable: true},
	})
	if err != nil {
		t.Fatalf("open gorm mysql connection: %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("resolve sql db: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	return db
}

var testDBNameSanitizer = regexp.MustCompile(`[^a-z0-9]+`)

func buildTestDBName(base string, testName string) string {
	normalizedBase := strings.ToLower(strings.TrimSpace(base))
	if normalizedBase == "" {
		normalizedBase = "pantheon"
	}
	normalizedName := strings.ToLower(strings.TrimSpace(testName))
	normalizedName = strings.ReplaceAll(normalizedName, "/", "_")
	normalizedName = testDBNameSanitizer.ReplaceAllString(normalizedName, "_")
	normalizedName = strings.Trim(normalizedName, "_")
	if normalizedName == "" {
		normalizedName = "test"
	}

	suffix := fmt.Sprintf("%d_%04d", time.Now().UnixNano(), rand.Intn(10000))
	name := fmt.Sprintf("%s_%s_%s", normalizedBase, normalizedName, suffix)
	if len(name) > 60 {
		name = name[:60]
	}
	return strings.TrimRight(name, "_")
}
