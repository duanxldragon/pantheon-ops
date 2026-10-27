package testmysql

import (
	"crypto/rand"
	"database/sql"
	"fmt"
	"math/big"
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

// requiredEnvVar switches resolution from "skip when unconfigured" to "fail".
// Set it in any job that provisions MySQL so a typo'd or renamed variable fails
// the build instead of reducing coverage silently.
const requiredEnvVar = "PANTHEON_TEST_DSN_REQUIRED"

// reporter is the subset of *testing.T that DSN resolution needs. Taking the
// narrow interface keeps the fail-vs-skip decision testable: Fatalf/Skipf end
// the calling goroutine, so a real *testing.T cannot be asserted on.
type reporter interface {
	Helper()
	Fatalf(format string, args ...any)
	Skipf(format string, args ...any)
	Cleanup(func())
	Name() string
}

func isTruthy(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func resolveDSN(t reporter) string {
	t.Helper()

	dsn := strings.TrimSpace(os.Getenv("PANTHEON_TEST_DSN"))
	if dsn == "" {
		dsn = strings.TrimSpace(os.Getenv("PANTHEON_DSN"))
	}
	if dsn == "" {
		if isTruthy(os.Getenv(requiredEnvVar)) {
			t.Fatalf(
				"mysql is required (%s is set) but neither PANTHEON_TEST_DSN nor PANTHEON_DSN is configured; a misconfigured variable would otherwise skip this test silently",
				requiredEnvVar,
			)
		} else {
			t.Skipf("mysql test dsn is not configured (PANTHEON_TEST_DSN and PANTHEON_DSN unset)")
		}
	}
	return dsn
}

// Open creates an isolated MySQL database for the current test.
func Open(t *testing.T) *gorm.DB {
	return open(t)
}

// OpenTB is the testing.TB form of Open so benchmarks (testing.B) can also
// create isolated per-run databases. It shares Open's DSN resolution, naming,
// and cleanup behavior.
func OpenTB(tb testing.TB) *gorm.DB {
	return open(tb)
}

func open(t reporter) *gorm.DB {
	t.Helper()

	dsn := resolveDSN(t)
	if dsn == "" {
		return nil
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
	t.Cleanup(func() {
		_ = adminDB.Close()
	})

	testDBName, err := buildTestDBName(cfg.DBName, t.Name())
	if err != nil {
		t.Fatalf("build test database name: %v", err)
	}
	createDatabaseStatement := buildCreateDatabaseStatement(testDBName)
	if _, err := adminDB.Exec(createDatabaseStatement); err != nil { // NOSONAR — test helper, controlled input
		t.Fatalf("create test database %s: %v", testDBName, err)
	}
	dropDatabaseStatement := buildDropDatabaseStatement(testDBName)
	t.Cleanup(func() {
		_, _ = adminDB.Exec(dropDatabaseStatement) // NOSONAR — test helper, controlled input
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
	t.Cleanup(func() {
		_ = sqlDB.Close()
	})
	return db
}

var testDBNameSanitizer = regexp.MustCompile(`[^a-z0-9]+`)
var validMySQLIdentifierPattern = regexp.MustCompile(`^[a-z0-9_]+$`)

const maxTestDBNameLength = 60

func buildTestDBName(base, testName string) (string, error) {
	normalizedBase := normalizeDBNameSegment(base, "pantheon")
	if normalizedBase == "" {
		normalizedBase = "pantheon"
	}
	normalizedName := normalizeDBNameSegment(strings.ReplaceAll(testName, "/", "_"), "test")
	if normalizedName == "" {
		normalizedName = "test"
	}

	suffix, err := randomSuffix()
	if err != nil {
		return "", err
	}

	prefix := strings.TrimRight(fmt.Sprintf("%s_%s", normalizedBase, normalizedName), "_")
	maxPrefixLength := maxTestDBNameLength - len(suffix) - 1
	if maxPrefixLength <= 0 {
		return "", fmt.Errorf("generated mysql suffix %q exceeds identifier budget", suffix)
	}
	if len(prefix) > maxPrefixLength {
		prefix = strings.TrimRight(prefix[:maxPrefixLength], "_")
	}
	if prefix == "" {
		prefix = "test"
	}

	name := prefix + "_" + suffix
	if !validMySQLIdentifierPattern.MatchString(name) {
		return "", fmt.Errorf("invalid generated database name %q", name)
	}
	return name, nil
}

func normalizeDBNameSegment(value, fallback string) string {
	normalized := strings.ToLower(strings.TrimSpace(value))
	normalized = testDBNameSanitizer.ReplaceAllString(normalized, "_")
	normalized = strings.Trim(normalized, "_")
	if normalized == "" {
		return fallback
	}
	return normalized
}

func randomSuffix() (string, error) {
	value, err := rand.Int(rand.Reader, big.NewInt(10000))
	if err != nil {
		return "", fmt.Errorf("generate random suffix: %w", err)
	}
	return fmt.Sprintf("%d_%04d", time.Now().UnixNano(), value.Int64()), nil
}

func buildCreateDatabaseStatement(name string) string {
	return "CREATE DATABASE " + quoteMySQLIdentifier(name) + " CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci"
}

func buildDropDatabaseStatement(name string) string {
	return "DROP DATABASE IF EXISTS " + quoteMySQLIdentifier(name)
}

func quoteMySQLIdentifier(name string) string {
	normalized := strings.TrimSpace(name)
	if !validMySQLIdentifierPattern.MatchString(normalized) {
		panic(fmt.Sprintf("unsafe mysql identifier %q", name))
	}
	return "`" + normalized + "`"
}
