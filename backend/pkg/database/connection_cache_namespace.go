package database

import (
	"fmt"

	mysqlDriver "github.com/go-sql-driver/mysql"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// ConnectionCacheNamespace returns a stable identity for the logical database
// behind db, suitable for keying process-local caches.
//
// The identity is derived from the connection DSN (host + database name) rather
// than from the *sql.DB pointer. Pointer identity is not safe as a cache key:
// once a pool is closed and becomes unreachable the Go runtime may hand the very
// same address to a brand-new pool, so an unrelated connection would inherit the
// previous one's cached verdicts. Per-test databases hit exactly that case — the
// data scope middleware cached "system_role_data_scope exists" against a pool
// that had already been dropped, and the next test's freshly created database
// inherited the verdict, which made MigrateDataScopePolicy skip its CREATE TABLE.
func ConnectionCacheNamespace(db *gorm.DB) string {
	if db == nil {
		return "nil"
	}
	if namespace := mysqlDSNCacheNamespace(db.Dialector); namespace != "" {
		return namespace
	}
	// Non-MySQL dialectors have no DSN contract here; fall back to pool identity.
	return fmt.Sprintf("pool:%p", db)
}

// mysqlDSNCacheNamespace derives "mysql:<addr>/<dbname>" from a MySQL dialector
// DSN. It returns "" when the dialector is not MySQL or the DSN carries no
// database name, so callers can fall back to another identity source.
func mysqlDSNCacheNamespace(dialector gorm.Dialector) string {
	config := mysqlDialectorConfig(dialector)
	if config == nil || config.DSN == "" {
		return ""
	}
	parsed, err := mysqlDriver.ParseDSN(config.DSN)
	if err != nil || parsed.DBName == "" {
		return ""
	}
	addr := parsed.Addr
	if addr == "" {
		addr = parsed.Net
	}
	return fmt.Sprintf("mysql:%s/%s", addr, parsed.DBName)
}

func mysqlDialectorConfig(dialector gorm.Dialector) *mysql.Config {
	switch typed := dialector.(type) {
	case *mysql.Dialector:
		return typed.Config
	case mysql.Dialector:
		return typed.Config
	default:
		return nil
	}
}
