package database

import (
	"strings"
	"testing"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// openOfflineDB builds a MySQL-backed gorm handle without touching the network,
// so cache-namespace derivation can be unit tested without a live database.
func openOfflineDB(t *testing.T, dsn string) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(
		mysql.New(mysql.Config{DSN: dsn, SkipInitializeWithVersion: true}),
		&gorm.Config{DisableAutomaticPing: true},
	)
	if err != nil {
		t.Fatalf("open offline gorm db: %v", err)
	}
	return db
}

func TestConnectionCacheNamespaceDistinguishesDatabases(t *testing.T) {
	first := ConnectionCacheNamespace(openOfflineDB(t, "root:pw@tcp(127.0.0.1:3306)/pantheon_a?charset=utf8mb4"))
	second := ConnectionCacheNamespace(openOfflineDB(t, "root:pw@tcp(127.0.0.1:3306)/pantheon_b?charset=utf8mb4"))

	if first == "" || second == "" {
		t.Fatalf("expected mysql namespaces, got %q and %q", first, second)
	}
	if first == second {
		t.Fatalf("different databases must not share a cache namespace: %q", first)
	}
	if first != ConnectionCacheNamespace(openOfflineDB(t, "root:pw@tcp(127.0.0.1:3306)/pantheon_a?charset=utf8mb4")) {
		t.Fatalf("cache namespace must be stable for the same DSN: %q", first)
	}
}

func TestConnectionCacheNamespaceDistinguishesHosts(t *testing.T) {
	first := ConnectionCacheNamespace(openOfflineDB(t, "root:pw@tcp(10.0.0.1:3306)/pantheon?charset=utf8mb4"))
	second := ConnectionCacheNamespace(openOfflineDB(t, "root:pw@tcp(10.0.0.2:3306)/pantheon?charset=utf8mb4"))

	if first == second {
		t.Fatalf("different hosts must not share a cache namespace: %q", first)
	}
}

func TestConnectionCacheNamespaceNeverLeaksCredentials(t *testing.T) {
	namespace := ConnectionCacheNamespace(openOfflineDB(t, "root:super_secret@tcp(127.0.0.1:3306)/pantheon?charset=utf8mb4"))

	if namespace != "mysql:127.0.0.1:3306/pantheon" {
		t.Fatalf("unexpected namespace %q", namespace)
	}
}

func TestConnectionCacheNamespaceFallsBackWhenDSNIsUnusable(t *testing.T) {
	if got := ConnectionCacheNamespace(nil); got != "nil" {
		t.Fatalf("nil database must map to the nil namespace, got %q", got)
	}
	// Without a usable DSN there is no logical-database identity to key on, so
	// the caller falls back to pool identity rather than sharing one namespace.
	if got := ConnectionCacheNamespace(openOfflineDB(t, "root:pw@tcp(127.0.0.1:3306)/?charset=utf8mb4")); !strings.HasPrefix(got, "pool:") {
		t.Fatalf("expected a pool-identity fallback, got %q", got)
	}
	// A MySQL dialector without a database name cannot identify a logical database.
	if got := mysqlDSNCacheNamespace(&mysql.Dialector{Config: &mysql.Config{DSN: "root:pw@tcp(127.0.0.1:3306)/?charset=utf8mb4"}}); got != "" {
		t.Fatalf("expected no namespace without a database name, got %q", got)
	}
	if got := mysqlDSNCacheNamespace(&mysql.Dialector{Config: &mysql.Config{DSN: "not-a-mysql-dsn"}}); got != "" {
		t.Fatalf("expected no namespace for a malformed DSN, got %q", got)
	}
	if got := mysqlDSNCacheNamespace(&mysql.Dialector{}); got != "" {
		t.Fatalf("expected no namespace for a dialector without a config, got %q", got)
	}
	if got := mysqlDSNCacheNamespace(nil); got != "" {
		t.Fatalf("expected no namespace for a nil dialector, got %q", got)
	}
}
