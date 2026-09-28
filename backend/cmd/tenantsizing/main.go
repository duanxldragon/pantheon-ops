// Command tenantsizing computes the tenant-migration maintenance-window
// estimate (G3 input) from live production table sizes, using the
// rehearsal→production conversion formula in TENANT_MIGRATION_RUNBOOK.md §4.2:
//
//	T(生产) ≈ T(副本) × (rows_prod / rows_rehearsal) × 3   (safety factor 3)
//	窗口预留   = T(生产) × 2
//
// Tables with rows > 10,000,000 are flagged: runbook §4.2 requires them to be
// rehearsed in staging at production scale before their window counts.
//
// Read-only against the target database. Three subcommands:
//
//	tenantsizing snapshot -dsn <prod_dsn> [-out report.json]   capture sizes + fixture state
//	tenantsizing estimate -report report.json [-rehearsal-rows N] [-output text|json]
//	tenantsizing plan -report report.json                      full G3 worksheet (both, human-readable)
//
// Usage for the G3 gate:
//  1. Run `snapshot` against production (read-only) during the G2 restore
//     drill or any low-traffic moment; keep the JSON as evidence.
//  2. Run `plan` to print the per-table window worksheet; paste it into the
//     gate state record (.harness/state/2026-09-10-tenant-verification-and-gray-gates/).
//  3. Any table flagged OVER-10M must get a production-scale staging
//     rehearsal (runbook §4.2) before its per-table estimate may be used.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Rehearsal baselines from TENANT_MIGRATION_RUNBOOK.md §4.2 (2026-09-11,
// rehearse-20260911_065959): fresh full migration ~3s on 31 tables with
// empty-to-micro data; per-table ADD/DROP < 100ms on a 1-row table.
const (
	rehearsalFullMigrationSeconds = 3.0
	rehearsalPerTableSeconds      = 0.1
	rehearsalPerRowBackfillRate   = 100_000 // rows/sec assumed for batched backfill (B2), conservative
)

const (
	safetyFactor   = 3.0 // runbook §4.2: covers IO differences rehearsal→production
	windowReserve  = 2.0 // runbook §1/§4.2: reserve = estimate × 2
	over10mRows    = 10_000_000
	over10mAction  = "must rehearse at production scale in staging before this estimate counts (runbook §4.2)"
	batchSize      = 5000 // runbook §3.2 B2 backfill batch size
	perBatchCommit = 0.05 // 50ms commit overhead per batch, conservative
)

// migrateScope mirrors runbook §2.1 Phase-1 decisions: which tables get the
// tenant_id column (stages B1–B4), which get unique-key conversion (stage C),
// and which stay global. Keys are table names.
var migrateScope = map[string]struct {
	addColumn     bool // stages B1–B4 apply
	uniqueKeySwap bool // stage C applies (per §2.2)
}{
	"system_user":                  {addColumn: true, uniqueKeySwap: true},
	"system_user_profile_ext":      {addColumn: true},
	"system_user_role":             {addColumn: true},
	"system_user_password_history": {addColumn: true},
	"system_user_session":          {addColumn: true}, "system_role": {addColumn: true, uniqueKeySwap: true},
	"system_role_menu":                       {addColumn: true},
	"system_role_permission":                 {addColumn: true},
	"system_role_data_scope":                 {addColumn: true},
	"system_dept":                            {addColumn: true},
	"system_post":                            {addColumn: true},
	"system_dict_type":                       {addColumn: true},
	"system_dict_item":                       {addColumn: true},
	"system_log_login":                       {addColumn: true},
	"system_log_oper":                        {addColumn: true},
	"system_auth_security_event":             {addColumn: true},
	"system_auth_factor":                     {addColumn: true},
	"system_auth_mfa_challenge":              {addColumn: true},
	"system_generator_datasource":            {addColumn: true},
	"system_module_registration":             {addColumn: true},
	"permission_workbench_remediation_event": {addColumn: true},
	// Kept global per runbook §2.1: system_menu, system_setting,
	// system_setting_audit_log, system_i18n, system_login_throttle,
	// system_refresh_version, module_registration,
	// permission_role_data_scope_policy, casbin_rule, schema_migrations.
}

type tableStat struct {
	Name       string  `json:"name"`
	Rows       uint64  `json:"rows"`
	DataMB     float64 `json:"dataMb"`
	IndexMB    float64 `json:"indexMb"`
	InScope    bool    `json:"inMigrationScope"`
	UniqueSwap bool    `json:"uniqueKeySwap"`
}

type snapshotReport struct {
	RunID       string      `json:"runId"`
	CapturedAt  time.Time   `json:"capturedAt"`
	DSNHost     string      `json:"dsnHost"`
	Database    string      `json:"database"`
	Tables      []tableStat `json:"tables"`
	TotalRows   uint64      `json:"totalRows"`
	InScopeRows uint64      `json:"inScopeRows"`
	Notes       []string    `json:"notes"`
}

type tableEstimate struct {
	Name             string  `json:"name"`
	Rows             uint64  `json:"rows"`
	AddColumnSec     float64 `json:"addColumnSec"`
	BackfillSec      float64 `json:"backfillSec"`
	TightenSec       float64 `json:"tightenSec"`
	UniqueSwapSec    float64 `json:"uniqueSwapSec"`
	TableTotalSec    float64 `json:"tableTotalSec"`
	WindowReserveSec float64 `json:"windowReserveSec"`
	Over10M          bool    `json:"over10m"`
	Over10MAction    string  `json:"over10mAction,omitempty"`
}

type estimateReport struct {
	RunID              string          `json:"runId"`
	GeneratedAt        time.Time       `json:"generatedAt"`
	Formula            string          `json:"formula"`
	PerTable           []tableEstimate `json:"perTable"`
	TotalEstimateSec   float64         `json:"totalEstimateSec"`
	WindowReserveSec   float64         `json:"windowReserveSec"`
	WindowReserveHuman string          `json:"windowReserveHuman"`
	Over10MTables      []string        `json:"over10mTables"`
	Verdict            string          `json:"verdict"`
}

func main() {
	if len(os.Args) < 2 {
		fatal("usage: tenantsizing snapshot|estimate|plan ...")
	}
	switch os.Args[1] {
	case "snapshot":
		cmdSnapshot(os.Args[2:])
	case "estimate":
		cmdEstimate(os.Args[2:], false)
	case "plan":
		cmdEstimate(os.Args[2:], true)
	default:
		fatal("unknown subcommand %q (want snapshot|estimate|plan)", os.Args[1])
	}
}

func cmdSnapshot(args []string) {
	fs := flag.NewFlagSet("snapshot", flag.ExitOnError)
	dsn := fs.String("dsn", "", "production DSN (read-only use), e.g. 'user:pass@tcp(host:3306)/pantheon_base'")
	out := fs.String("out", "", "write JSON report to this path (default stdout)")
	if err := fs.Parse(args); err != nil {
		fatal("parse flags: %v", err)
	}
	if *dsn == "" {
		fatal("-dsn is required (read-only account recommended)")
	}
	db := openDB(*dsn)
	defer closeDB(db)

	report := buildSnapshot(db)
	if *out != "" {
		writeJSON(*out, report)
		fmt.Printf("snapshot written to %s (%d tables, %d in-scope rows)\n", *out, len(report.Tables), report.InScopeRows)
		return
	}
	writeJSONWriter(os.Stdout, report)
}

func cmdEstimate(args []string, humanPlan bool) {
	fs := flag.NewFlagSet("estimate", flag.ExitOnError)
	reportPath := fs.String("report", "", "path to snapshot JSON from `snapshot`")
	rehearsalRows := fs.Uint64("rehearsal-rows", 1, "rehearsal row count used in §4.2 conversion (default 1, matching tenant_rehearsal)")
	if err := fs.Parse(args); err != nil {
		fatal("parse flags: %v", err)
	}
	if *reportPath == "" {
		fatal("-report is required (path to snapshot JSON)")
	}
	raw, err := os.ReadFile(*reportPath)
	if err != nil {
		fatal("read report: %v", err)
	}
	var snap snapshotReport
	if err := json.Unmarshal(raw, &snap); err != nil {
		fatal("parse snapshot JSON: %v", err)
	}
	est := buildEstimate(&snap, *rehearsalRows)
	if humanPlan {
		printPlan(&snap, est)
		return
	}
	writeJSONWriter(os.Stdout, est)
}

func openDB(dsn string) *gorm.DB {
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		fatal("open database: %v", err)
	}
	return db
}

func closeDB(db *gorm.DB) {
	if sqlDB, err := db.DB(); err == nil {
		_ = sqlDB.Close()
	}
}

func buildSnapshot(db *gorm.DB) *snapshotReport {
	var dbName string
	if err := db.Raw("SELECT DATABASE()").Scan(&dbName).Error; err != nil {
		fatal("resolve database name: %v", err)
	}

	var rawStats []struct {
		Name    string  `gorm:"column:table_name"`
		Rows    uint64  `gorm:"column:table_rows"`
		DataMB  float64 `gorm:"column:data_mb"`
		IndexMB float64 `gorm:"column:index_mb"`
	}
	err := db.Raw(`
		SELECT table_name, COALESCE(table_rows,0) AS table_rows,
		       COALESCE(data_length,0)/1048576 AS data_mb,
		       COALESCE(index_length,0)/1048576 AS index_mb
		FROM information_schema.tables
		WHERE table_schema = ? AND table_type = 'BASE TABLE'
		ORDER BY table_name`, dbName).Scan(&rawStats).Error
	if err != nil {
		fatal("query information_schema: %v", err)
	}
	stats := make([]tableStat, 0, len(rawStats))
	for _, r := range rawStats {
		stats = append(stats, tableStat{Name: r.Name, Rows: r.Rows, DataMB: r.DataMB, IndexMB: r.IndexMB})
	}

	notes := []string{
		"table_rows is InnoDB's estimate; for >10M tables confirm with exact COUNT(*) during the G2 restore-drill window",
		"scope per TENANT_MIGRATION_RUNBOOK.md §2.1 Phase-1 decisions",
	}
	total := uint64(0)
	inScope := uint64(0)
	for i := range stats {
		t := &stats[i]
		if scope, ok := migrateScope[t.Name]; ok {
			t.InScope = scope.addColumn
			t.UniqueSwap = scope.uniqueKeySwap
		}
		total += t.Rows
		if t.InScope {
			inScope += t.Rows
		}
	}

	host := "unknown"
	var hostName string
	if err := db.Raw("SELECT @@hostname").Scan(&hostName).Error; err == nil && strings.TrimSpace(hostName) != "" {
		host = hostName
	}

	return &snapshotReport{
		RunID:       fmt.Sprintf("g3-sizing-%s", time.Now().UTC().Format("20060102_150405")),
		CapturedAt:  time.Now().UTC(),
		DSNHost:     host,
		Database:    dbName,
		Tables:      stats,
		TotalRows:   total,
		InScopeRows: inScope,
		Notes:       notes,
	}
}

// estimateTable converts one in-scope table's rehearsal timing into a
// production-window estimate (§4.2 per-stage decomposition).
func estimateTable(t tableStat, rehearsalRows uint64) tableEstimate {
	e := tableEstimate{Name: t.Name, Rows: t.Rows}
	rows := float64(t.Rows)
	conv := rows / float64(rehearsalRows) * safetyFactor

	// B1 ADD COLUMN: INSTANT in MySQL 8.0 → metadata-only; rehearsal floor.
	e.AddColumnSec = rehearsalPerTableSeconds * conv
	if e.AddColumnSec < rehearsalPerTableSeconds {
		e.AddColumnSec = rehearsalPerTableSeconds
	}
	// B2 backfill: batched UPDATE at the assumed rate + per-batch commit cost.
	batches := rows / batchSize
	e.BackfillSec = rows/rehearsalPerRowBackfillRate + batches*perBatchCommit
	// B4 MODIFY NOT NULL DEFAULT 0: in-place after NULL elimination; rehearsal floor.
	e.TightenSec = rehearsalPerTableSeconds * conv
	if e.TightenSec < rehearsalPerTableSeconds {
		e.TightenSec = rehearsalPerTableSeconds
	}
	// Stage C unique-index swap (INPLACE build) — heaviest per-row op; double it.
	if t.UniqueSwap {
		e.UniqueSwapSec = rehearsalPerTableSeconds * conv * 2
	}
	e.TableTotalSec = e.AddColumnSec + e.BackfillSec + e.TightenSec + e.UniqueSwapSec
	e.WindowReserveSec = e.TableTotalSec * windowReserve
	e.Over10M = t.Rows > over10mRows
	if e.Over10M {
		e.Over10MAction = over10mAction
	}
	return e
}

func buildEstimate(snap *snapshotReport, rehearsalRows uint64) *estimateReport {
	if rehearsalRows == 0 {
		rehearsalRows = 1
	}
	est := &estimateReport{
		RunID:       snap.RunID,
		GeneratedAt: time.Now().UTC(),
		Formula:     fmt.Sprintf("T(prod) = T(rehearsal) × (rows_prod / %d) × %.1f; window = T(prod) × %.1f (runbook §4.2)", rehearsalRows, safetyFactor, windowReserve),
	}

	for _, t := range snap.Tables {
		if !t.InScope {
			continue
		}
		e := estimateTable(t, rehearsalRows)
		if e.Over10M {
			est.Over10MTables = append(est.Over10MTables, t.Name)
		}
		est.PerTable = append(est.PerTable, e)
		est.TotalEstimateSec += e.TableTotalSec
	}

	// Global stages not per-table: stage A (new tables, tiny), stage D (casbin
	// domain rewrite, batched UPDATE over casbin_rule), stage E deferred (real
	// tenant backfill is a separate window per runbook §3.5).
	var casbinRows uint64
	for _, t := range snap.Tables {
		if t.Name == "casbin_rule" {
			casbinRows = t.Rows
		}
	}
	if casbinRows > 0 {
		stageD := float64(casbinRows)/rehearsalPerRowBackfillRate + (float64(casbinRows)/batchSize)*perBatchCommit
		est.TotalEstimateSec += stageD
	}

	est.WindowReserveSec = est.TotalEstimateSec * windowReserve
	est.WindowReserveHuman = humanDuration(est.WindowReserveSec)

	sort.Slice(est.PerTable, func(i, j int) bool { return est.PerTable[i].Rows > est.PerTable[j].Rows })

	// Verdict: honest about limits — this is an estimate, gated by the 10M rule.
	if len(est.Over10MTables) > 0 {
		est.Verdict = fmt.Sprintf("ESTIMATE-PROVISIONAL: %d table(s) over 10M rows (%s) require production-scale staging rehearsal before the window may be used for gate G3",
			len(est.Over10MTables), strings.Join(est.Over10MTables, ", "))
	} else {
		est.Verdict = "ESTIMATE-READY: no table exceeds 10M rows; window estimate may back gate G3 once captured from production"
	}
	return est
}

func printPlan(snap *snapshotReport, est *estimateReport) {
	fmt.Printf("G3 Window Worksheet — %s (%s/%s @ %s)\n", est.RunID, snap.DSNHost, snap.Database, snap.CapturedAt.Format(time.RFC3339))
	fmt.Printf("Formula: %s\n\n", est.Formula)

	w := tabwriter.NewWriter(os.Stdout, 2, 4, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "TABLE\tROWS\tB1+B4 DDL(s)\tBACKFILL(s)\tUNIQUE C(s)\tTABLE TOTAL(s)\tWINDOW ×2\tFLAGS")
	for _, e := range est.PerTable {
		flags := []string{}
		if e.UniqueSwapSec > 0 {
			flags = append(flags, "uk-swap")
		}
		if e.Over10M {
			flags = append(flags, "OVER-10M")
		}
		_, _ = fmt.Fprintf(w, "%s\t%d\t%.1f\t%.1f\t%.1f\t%.1f\t%.1f\t%s\n",
			e.Name, e.Rows, e.AddColumnSec+e.TightenSec, e.BackfillSec, e.UniqueSwapSec, e.TableTotalSec, e.WindowReserveSec, strings.Join(flags, ","))
	}
	_ = w.Flush()

	fmt.Printf("\nTotal estimate: %.1f s   Window reserve (×2): %.1f s (%s)\n",
		est.TotalEstimateSec, est.WindowReserveSec, est.WindowReserveHuman)
	if len(est.Over10MTables) > 0 {
		fmt.Printf("\n⚠ OVER-10M TABLES (staging rehearsal at production scale REQUIRED before G3):\n")
		for _, name := range est.Over10MTables {
			fmt.Printf("  - %s\n", name)
		}
	}
	fmt.Printf("\nVerdict: %s\n", est.Verdict)
	fmt.Printf("\nNext steps:\n  1. Paste this worksheet into .harness/state/2026-09-10-tenant-verification-and-gray-gates/\n")
	fmt.Printf("  2. If OVER-10M flags exist: rehearse those tables in staging with production-scale data, then re-run with updated rehearsal rows\n")
	fmt.Printf("  3. Maintainer flips G3 in runbook §8.1 only when the window is approved against this estimate\n")
}

func humanDuration(seconds float64) string {
	d := time.Duration(seconds * float64(time.Second))
	if d < time.Minute {
		return d.Round(time.Second).String()
	}
	if d < time.Hour {
		return d.Round(time.Minute).String()
	}
	return d.Round(time.Minute).String()
}

func writeJSON(path string, v any) {
	f, err := os.Create(path) //nolint:gosec // G304: path comes from the CLI flag, by design
	if err != nil {
		fatal("create %s: %v", path, err)
	}
	defer func() { _ = f.Close() }()
	writeJSONWriter(f, v)
}

func writeJSONWriter(f *os.File, v any) {
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		fatal("encode JSON: %v", err)
	}
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "tenantsizing: "+format+"\n", args...)
	os.Exit(1)
}
