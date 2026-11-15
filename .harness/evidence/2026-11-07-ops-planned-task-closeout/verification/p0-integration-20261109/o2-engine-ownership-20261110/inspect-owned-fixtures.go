// Local verification helper: list fixtures, or clean explicitly named owned
// fixtures after checking the generated test-name/creation-time suffix.
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"time"

	mysql "github.com/go-sql-driver/mysql"
)

func main() {
	cfg, err := mysql.ParseDSN(os.Getenv("PANTHEON_TEST_DSN"))
	if err != nil || cfg.Net != "tcp" || cfg.Addr != "127.0.0.1:33316" || cfg.DBName != "ops_p0_acceptance" {
		panic("expected the explicitly authorized local acceptance DSN")
	}
	cfg.DBName = ""
	db, err := sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		panic(err)
	}
	defer db.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	pattern := regexp.MustCompile(`^ops_p0_acceptance_(test[a-z0-9_]+)_([0-9]{19})_[0-9]{4}$`)
	ownedTimeoutFixtures := map[string]bool{
		"ops_p0_acceptance_testownershipexac_1794289745195290400_1738": true,
		"ops_p0_acceptance_testpreviewfrozen_1794289873661002300_2494": true,
	}
	for _, exact := range os.Args[1:] {
		if pattern.FindStringSubmatch(exact) == nil || !ownedTimeoutFixtures[exact] {
			panic("invalid owned fixture name")
		}
		if _, err := db.ExecContext(ctx, "DROP DATABASE IF EXISTS `"+exact+"`"); err != nil {
			panic(err)
		}
		fmt.Fprintln(os.Stderr, "cleaned exact fixture", exact)
	}
	rows, err := db.QueryContext(ctx, "SELECT SCHEMA_NAME FROM information_schema.SCHEMATA WHERE SCHEMA_NAME LIKE 'ops_p0_acceptance_%' ORDER BY SCHEMA_NAME")
	if err != nil {
		panic(err)
	}
	defer rows.Close()
	var fixtures []map[string]string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			panic(err)
		}
		parts := pattern.FindStringSubmatch(name)
		if parts == nil {
			continue
		}
		nanos, err := strconv.ParseInt(parts[2], 10, 64)
		if err != nil {
			panic(err)
		}
		fixtures = append(fixtures, map[string]string{"name": name, "createdUTC": time.Unix(0, nanos).UTC().Format(time.RFC3339Nano)})
	}
	if err := rows.Err(); err != nil {
		panic(err)
	}
	if err := json.NewEncoder(os.Stdout).Encode(fixtures); err != nil {
		panic(err)
	}
}
