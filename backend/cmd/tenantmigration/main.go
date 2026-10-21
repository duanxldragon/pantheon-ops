package main

import (
	"database/sql"
	"fmt"
	"os"
	"strings"

	"github.com/duanxldragon/pantheon-base/backend/pkg/database"
	// Blank import registers the go-sql-driver MySQL driver with database/sql so
	// sql.Open above can open the DSN; the driver is used indirectly via golang-migrate.
	_ "github.com/go-sql-driver/mysql"
)

func main() {
	if len(os.Args) != 2 || (os.Args[1] != "up" && os.Args[1] != "status") {
		fail("usage: tenantmigration up|status")
	}
	dsn := strings.TrimSpace(os.Getenv("PANTHEON_DSN"))
	if dsn == "" {
		fail("PANTHEON_DSN is required")
	}
	if os.Args[1] == "up" {
		if err := database.RunMigrations(dsn); err != nil {
			fail("migration failed: %v", err)
		}
	}
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		fail("open database: %v", err)
	}
	defer func() { _ = db.Close() }()
	if err := db.Ping(); err != nil {
		fail("connect database: %v", err)
	}
	var version int
	var dirty bool
	var migrationTable string
	if err := db.QueryRow("SHOW TABLES LIKE 'schema_migrations'").Scan(&migrationTable); err == sql.ErrNoRows {
		fmt.Println("Connected; no migration table found")
		return
	} else if err != nil {
		fail("check migration table: %v", err)
	}
	err = db.QueryRow("SELECT version, dirty FROM schema_migrations LIMIT 1").Scan(&version, &dirty)
	if err == sql.ErrNoRows {
		fmt.Println("Connected; no migration version recorded")
		return
	}
	if err != nil {
		fail("read migration status: %v", err)
	}
	fmt.Printf("Connected; migration version=%d dirty=%t\n", version, dirty)
	if dirty {
		os.Exit(1)
	}
}

func fail(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
