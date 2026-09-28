package main

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fixtureSnapshot writes a snapshot JSON to disk and returns the path.
func fixtureSnapshot(t *testing.T, tables []tableStat) string {
	t.Helper()
	snap := &snapshotReport{
		RunID:    "g3-sizing-fixture",
		Database: "fixture_db",
		DSNHost:  "fixture-host",
		Tables:   tables,
		Notes:    []string{"fixture"},
	}
	for _, tab := range tables {
		snap.TotalRows += tab.Rows
		if tab.InScope {
			snap.InScopeRows += tab.Rows
		}
	}
	path := filepath.Join(t.TempDir(), "snapshot.json")
	raw, err := json.Marshal(snap)
	if err != nil {
		t.Fatalf("marshal fixture: %v", err)
	}
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	return path
}

func findTable(t *testing.T, est *estimateReport, name string) *tableEstimate {
	t.Helper()
	for i := range est.PerTable {
		if est.PerTable[i].Name == name {
			return &est.PerTable[i]
		}
	}
	t.Fatalf("table %q missing from estimate", name)
	return nil
}

func almostEqual(a, b float64) bool {
	return math.Abs(a-b) <= math.Max(1e-9, math.Max(math.Abs(a), math.Abs(b))*1e-9)
}

// Core §4.2 conversion: per-stage decomposition and the unique-swap weighting.
func TestBuildEstimate_PerStageMath(t *testing.T) {
	tables := []tableStat{
		{Name: "system_user", Rows: 1000, InScope: true, UniqueSwap: true},
		{Name: "system_dept", Rows: 1000, InScope: true},
		{Name: "system_menu", Rows: 1000}, // global: must be excluded
	}
	est := buildEstimate(&snapshotReport{Tables: tables, RunID: "x"}, 1)

	if got := len(est.PerTable); got != 2 {
		t.Fatalf("global tables must be excluded from per-table estimate, got %d entries", got)
	}

	// Rehearsal baseline 0.1s/table, ratio 1000/1, safety 3 → 0.1*1000*3 = 300s.
	user := findTable(t, est, "system_user")
	if !almostEqual(user.AddColumnSec, 300) || !almostEqual(user.TightenSec, 300) {
		t.Fatalf("B1/B4 DDL math: got add=%v tighten=%v, want 300 each", user.AddColumnSec, user.TightenSec)
	}
	// Unique swap: double the weighted DDL → 600s.
	if !almostEqual(user.UniqueSwapSec, 600) {
		t.Fatalf("unique swap must weight ×2: got %v, want 600", user.UniqueSwapSec)
	}
	// Backfill: 1000 rows / 100000 rps + ceil-free batch cost 1000/5000 batches.
	wantBackfill := 1000.0/rehearsalPerRowBackfillRate + (1000.0/batchSize)*perBatchCommit
	if !almostEqual(user.BackfillSec, wantBackfill) {
		t.Fatalf("backfill math: got %v, want %v", user.BackfillSec, wantBackfill)
	}
	// Table total = add + backfill + tighten + swap.
	wantTotal := 300 + wantBackfill + 300 + 600
	if !almostEqual(user.TableTotalSec, wantTotal) {
		t.Fatalf("table total: got %v, want %v", user.TableTotalSec, wantTotal)
	}
	// Window reserve = total × 2.
	if !almostEqual(user.WindowReserveSec, wantTotal*windowReserve) {
		t.Fatalf("window reserve: got %v, want %v", user.WindowReserveSec, wantTotal*windowReserve)
	}

	// Non-swap table has no unique component.
	dept := findTable(t, est, "system_dept")
	if dept.UniqueSwapSec != 0 {
		t.Fatalf("non-swap table must have zero unique-swap seconds, got %v", dept.UniqueSwapSec)
	}
}

// Small tables must never estimate below the rehearsal floor (0.1s per DDL step).
func TestBuildEstimate_RehearsalFloor(t *testing.T) {
	tables := []tableStat{{Name: "system_post", Rows: 3, InScope: true}}
	est := buildEstimate(&snapshotReport{Tables: tables}, 1)
	p := findTable(t, est, "system_post")
	if p.AddColumnSec < rehearsalPerTableSeconds || p.TightenSec < rehearsalPerTableSeconds {
		t.Fatalf("floor not enforced: add=%v tighten=%v, both must be >= %v", p.AddColumnSec, p.TightenSec, rehearsalPerTableSeconds)
	}
}

// rehearsalRows=0 must be normalized to 1 instead of dividing by zero.
func TestBuildEstimate_ZeroRehearsalRowsNormalized(t *testing.T) {
	tables := []tableStat{{Name: "system_user", Rows: 100, InScope: true, UniqueSwap: true}}
	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("zero rehearsal rows must not panic: %v", r)
			}
		}()
		est := buildEstimate(&snapshotReport{Tables: tables}, 0)
		user := findTable(t, est, "system_user")
		if !almostEqual(user.AddColumnSec, 100*rehearsalPerTableSeconds*safetyFactor) {
			t.Fatalf("expected ratio-100 conversion, got %v", user.AddColumnSec)
		}
	}()
}

// Over-10M rule: verdict flips to provisional and the table is flagged.
func TestBuildEstimate_Over10MFlipsVerdict(t *testing.T) {
	tables := []tableStat{
		{Name: "system_log_oper", Rows: over10mRows + 1, InScope: true},
		{Name: "system_post", Rows: 5, InScope: true},
	}
	est := buildEstimate(&snapshotReport{Tables: tables}, 1)
	if !strings.Contains(est.Verdict, "ESTIMATE-PROVISIONAL") {
		t.Fatalf("over-10M table must flip verdict to ESTIMATE-PROVISIONAL, got %q", est.Verdict)
	}
	if len(est.Over10MTables) != 1 || est.Over10MTables[0] != "system_log_oper" {
		t.Fatalf("over10m list: got %v, want [system_log_oper]", est.Over10MTables)
	}
	flagged := findTable(t, est, "system_log_oper")
	if !flagged.Over10M || flagged.Over10MAction == "" {
		t.Fatalf("over-10M table must carry flag + action text")
	}
}

// Under the threshold the estimate is ready and nothing is flagged.
func TestBuildEstimate_AtThresholdReady(t *testing.T) {
	tables := []tableStat{{Name: "system_log_oper", Rows: over10mRows, InScope: true}}
	est := buildEstimate(&snapshotReport{Tables: tables}, 1)
	if !strings.Contains(est.Verdict, "ESTIMATE-READY") {
		t.Fatalf("exactly 10M rows is not over the threshold; verdict %q", est.Verdict)
	}
	if len(est.Over10MTables) != 0 {
		t.Fatalf("no table should be flagged at exactly 10M rows, got %v", est.Over10MTables)
	}
}

// Stage D: casbin_rule rows feed the global rewrite cost, outside per-table scope.
func TestBuildEstimate_StageDCasbinCost(t *testing.T) {
	with := []tableStat{
		{Name: "casbin_rule", Rows: 100000},
		{Name: "system_post", Rows: 1, InScope: true},
	}
	without := []tableStat{
		{Name: "casbin_rule", Rows: 0},
		{Name: "system_post", Rows: 1, InScope: true},
	}
	estWith := buildEstimate(&snapshotReport{Tables: with}, 1)
	estWithout := buildEstimate(&snapshotReport{Tables: without}, 1)
	delta := estWith.TotalEstimateSec - estWithout.TotalEstimateSec
	want := 100000.0/rehearsalPerRowBackfillRate + (100000.0/batchSize)*perBatchCommit
	if !almostEqual(delta, want) {
		t.Fatalf("stage D cost: got delta %v, want %v", delta, want)
	}
}

// End-to-end: JSON file → estimate via the same path cmdEstimate uses.
func TestEstimate_FromFixtureFile(t *testing.T) {
	path := fixtureSnapshot(t, []tableStat{
		{Name: "system_user", Rows: 1000, InScope: true, UniqueSwap: true},
		{Name: "system_menu", Rows: 500},
	})
	raw, err := os.ReadFile(path) //nolint:gosec // G304: t.TempDir fixture path
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	var snap snapshotReport
	if err := json.Unmarshal(raw, &snap); err != nil {
		t.Fatalf("parse fixture: %v", err)
	}
	est := buildEstimate(&snap, 1)
	if got := len(est.PerTable); got != 1 {
		t.Fatalf("only in-scope tables estimated, got %d", got)
	}
	user := findTable(t, est, "system_user")
	if user.Rows != 1000 {
		t.Fatalf("fixture rows lost: %d", user.Rows)
	}
	// Window reserve on the whole estimate = total × 2.
	if !almostEqual(est.WindowReserveSec, est.TotalEstimateSec*windowReserve) {
		t.Fatalf("estimate-level window reserve inconsistent: %v vs total %v", est.WindowReserveSec, est.TotalEstimateSec)
	}
	if est.Formula == "" || !strings.Contains(est.Formula, "× 3.0") {
		t.Fatalf("formula string must cite the §4.2 safety factor, got %q", est.Formula)
	}
}

// Per-table rows sort descending in the output (largest migration risk first).
func TestBuildEstimate_SortedByRowsDesc(t *testing.T) {
	tables := []tableStat{
		{Name: "system_user", Rows: 10, InScope: true},
		{Name: "system_dept", Rows: 900, InScope: true},
		{Name: "system_post", Rows: 100, InScope: true},
	}
	est := buildEstimate(&snapshotReport{Tables: tables}, 1)
	for i := 1; i < len(est.PerTable); i++ {
		if est.PerTable[i-1].Rows < est.PerTable[i].Rows {
			t.Fatalf("per-table list not sorted desc at %d: %d < %d", i, est.PerTable[i-1].Rows, est.PerTable[i].Rows)
		}
	}
}
