package dynamicmodule

import (
	"path/filepath"
	"reflect"
	"sort"
	"testing"

	"pantheon-ops/backend/internal/scaffold"
)

func TestBuildFeatureLedgerSnapshot_ProjectsGeneratedModuleMetadata(t *testing.T) {
	service, workspaceRoot := newFeatureLedgerTestService(t)
	registerFeatureLedgerTicketModuleWithMetadata(t, service)

	snapshot, err := service.buildFeatureLedgerSnapshot()
	if err != nil {
		t.Fatalf("build feature ledger snapshot: %v", err)
	}
	if snapshot.Version != featureLedgerVersion {
		t.Fatalf("unexpected ledger version: %d", snapshot.Version)
	}
	if len(snapshot.Entries) != 1 {
		t.Fatalf("unexpected ledger entry count: %d", len(snapshot.Entries))
	}
	if len(snapshot.Issues) != 0 {
		t.Fatalf("expected clean snapshot, got issues: %#v", snapshot.Issues)
	}

	entry := snapshot.Entries[0]
	if entry.ModuleKey != "business.ticket" {
		t.Fatalf("unexpected module key: %s", entry.ModuleKey)
	}
	if entry.Owner != "platform" {
		t.Fatalf("unexpected owner: %s", entry.Owner)
	}
	if entry.BoundedContext != "ticketing" {
		t.Fatalf("unexpected bounded context: %s", entry.BoundedContext)
	}
	if entry.SourceMode != "manual" {
		t.Fatalf("unexpected source mode: %s", entry.SourceMode)
	}
	if entry.Source != "manual" {
		t.Fatalf("unexpected source: %s", entry.Source)
	}
	if entry.Maturity != "experimental" {
		t.Fatalf("unexpected maturity: %s", entry.Maturity)
	}

	assertFileContains(t, filepath.Join(workspaceRoot, "schema", "generated", "feature-ledger.json"), `"moduleKey": "business.ticket"`)
	assertFileContains(t, filepath.Join(workspaceRoot, "schema", "generated", "feature-ledger.json"), `"sourceMode": "manual"`)
}

func TestBuildFeatureLedgerSnapshot_ReportsMissingGeneratedMetadata(t *testing.T) {
	service, workspaceRoot := newFeatureLedgerTestService(t)
	registerFeatureLedgerTicketModule(t, service)

	snapshot, err := service.buildFeatureLedgerSnapshot()
	if err != nil {
		t.Fatalf("build feature ledger snapshot: %v", err)
	}
	if len(snapshot.Entries) != 1 {
		t.Fatalf("unexpected ledger entry count: %d", len(snapshot.Entries))
	}
	if len(snapshot.Issues) == 0 {
		t.Fatal("expected ledger drift issues")
	}
	assertFeatureLedgerIssue(t, snapshot.Issues, "business.ticket", "owner_missing")
	assertFeatureLedgerIssue(t, snapshot.Issues, "business.ticket", "bounded_context_missing")
	assertFeatureLedgerIssue(t, snapshot.Issues, "business.ticket", "source_mode_missing")

	assertFileContains(t, filepath.Join(workspaceRoot, "schema", "generated", "feature-ledger.json"), `"issues": [`)
	assertFileContains(t, filepath.Join(workspaceRoot, "schema", "generated", "feature-ledger.json"), `"owner_missing"`)
}

func TestBuildFeatureLedgerSnapshot_SortsIssuesDeterministically(t *testing.T) {
	service, workspaceRoot := newFeatureLedgerTestService(t)

	mustWriteFile(t, filepath.Join(workspaceRoot, "schema", "generated", "business", "zeta.json"), `{
  "name": "zeta",
  "scope": "business",
  "displayName": "Zeta",
  "metadata": {},
  "model": {
    "tableName": "biz_zeta"
  }
}`)

	seedFeatureLedgerAlphaRegistration(t, service)

	snapshot, err := service.buildFeatureLedgerSnapshot()
	if err != nil {
		t.Fatalf("build feature ledger snapshot: %v", err)
	}

	sortedIssues := sortedFeatureLedgerIssues(snapshot.Issues)

	if !reflect.DeepEqual(snapshot.Issues, sortedIssues) {
		t.Fatalf("expected issues to be sorted deterministically, got %#v want %#v", snapshot.Issues, sortedIssues)
	}
}

func newFeatureLedgerTestService(t *testing.T) (*DynamicModuleService, string) {
	t.Helper()
	db := openDynamicModuleTestDB(t)
	workspaceRoot := prepareDynamicModuleWorkspace(t)
	return &DynamicModuleService{
		db:            db,
		workspaceRoot: workspaceRoot,
	}, workspaceRoot
}

func registerFeatureLedgerTicketModuleWithMetadata(t *testing.T, service *DynamicModuleService) {
	t.Helper()
	req := newGeneratedModuleRequest("business", "ticket", "工单管理", "biz_ticket")
	req.Schema.Metadata.Owner = "platform"
	req.Schema.Metadata.BoundedContext = "ticketing"
	req.Schema.Metadata.SourceMode = "manual"
	req.Schema.Metadata.SourceTable = "biz_ticket_source"
	registerFeatureLedgerModule(t, service, req)
}

func registerFeatureLedgerTicketModule(t *testing.T, service *DynamicModuleService) {
	t.Helper()
	req := newGeneratedModuleRequest("business", "ticket", "工单管理", "biz_ticket")
	registerFeatureLedgerModule(t, service, req)
}

func registerFeatureLedgerModule(t *testing.T, service *DynamicModuleService, req *scaffold.RegisterGeneratedModuleRequest) {
	t.Helper()
	if _, _, _, err := service.RegisterGeneratedModule(req); err != nil {
		t.Fatalf("register generated module: %v", err)
	}
}

func seedFeatureLedgerAlphaRegistration(t *testing.T, service *DynamicModuleService) {
	t.Helper()
	if err := service.db.Create(&ModuleRegistration{
		Name:           "business.alpha",
		DisplayName:    "Alpha",
		Scope:          "business",
		Source:         "generated",
		ModelTableName: "biz_alpha",
		Status:         ModuleStatusActive,
		InstalledAt:    "2026-06-12T00:00:00Z",
	}).Error; err != nil {
		t.Fatalf("seed registration: %v", err)
	}
}

func sortedFeatureLedgerIssues(issues []FeatureLedgerIssue) []FeatureLedgerIssue {
	sortedIssues := append([]FeatureLedgerIssue(nil), issues...)
	sort.Slice(sortedIssues, func(i, j int) bool {
		return lessFeatureLedgerIssueForTest(sortedIssues[i], sortedIssues[j])
	})
	return sortedIssues
}

func lessFeatureLedgerIssueForTest(left, right FeatureLedgerIssue) bool {
	if left.ModuleKey != right.ModuleKey {
		return left.ModuleKey < right.ModuleKey
	}
	if left.Code != right.Code {
		return left.Code < right.Code
	}
	if left.Field != right.Field {
		return left.Field < right.Field
	}
	if left.Severity != right.Severity {
		return left.Severity < right.Severity
	}
	return left.Detail < right.Detail
}

func assertFeatureLedgerIssue(t *testing.T, issues []FeatureLedgerIssue, moduleKey, code string) {
	t.Helper()
	for _, issue := range issues {
		if issue.ModuleKey == moduleKey && issue.Code == code {
			return
		}
	}
	t.Fatalf("expected ledger issue %s for %s, got %#v", code, moduleKey, issues)
}
