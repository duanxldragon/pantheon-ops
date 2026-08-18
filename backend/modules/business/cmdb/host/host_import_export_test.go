package host

import (
	"context"
	"encoding/json"
	"testing"

	bizcap "pantheon-base/modules/business/capability"
	"pantheon-base/pkg/common"
	"pantheon-base/pkg/impexp"

	"gorm.io/gorm"
)

type testBizScopeReader struct{}

func (testBizScopeReader) GetActive(_ context.Context, id uint64, _ *common.DataScopeReq) (bizcap.BizScopeRef, error) {
	return bizcap.BizScopeRef{ID: id, Code: "scope", Name: "Scope", Status: "active"}, nil
}

func (testBizScopeReader) ResolveActiveByCodes(_ context.Context, codes []string, _ *common.DataScopeReq) (map[string]bizcap.BizScopeRef, error) {
	result := make(map[string]bizcap.BizScopeRef, len(codes))
	for _, code := range codes {
		if code == "scope-prod" {
			result[code] = bizcap.BizScopeRef{ID: 7, Code: "scope-prod", Name: "Production", Status: "active"}
		}
	}
	return result, nil
}

func setupHostImportExportTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db := setupTestDB(t)
	if err := db.Exec(`
CREATE TABLE biz_business_scope (
	id BIGINT PRIMARY KEY AUTO_INCREMENT,
	code VARCHAR(64) NOT NULL,
	name VARCHAR(128) NOT NULL,
	deleted_at DATETIME NULL
)`).Error; err != nil {
		t.Fatalf("create business scope table: %v", err)
	}
	return db
}

func TestHostServiceExportHonorsRowCap(t *testing.T) {
	db := setupTestDB(t)
	svc := NewHostService(db)

	oldCap := maxHostExportRows
	maxHostExportRows = 2
	t.Cleanup(func() {
		maxHostExportRows = oldCap
	})

	for i, item := range []Host{
		{Hostname: "host-a", IP: "10.0.0.1", OS: "linux", SSHPort: 22},
		{Hostname: "host-b", IP: "10.0.0.2", OS: "linux", SSHPort: 22},
		{Hostname: "host-c", IP: "10.0.0.3", OS: "windows", SSHPort: 22},
	} {
		if err := db.Create(&item).Error; err != nil {
			t.Fatalf("seed host %d: %v", i, err)
		}
	}

	exported, err := svc.Export(HostListQuery{}, nil)
	if err != nil {
		t.Fatalf("export hosts: %v", err)
	}
	if len(exported.Rows) != 2 {
		t.Fatalf("expected export capped at 2 rows, got %d", len(exported.Rows))
	}
	if exported.Filename != "cmdb-host-export.csv" {
		t.Fatalf("unexpected export filename %q", exported.Filename)
	}
}

func TestHostServiceImportEmptyFile(t *testing.T) {
	svc := NewHostService(setupTestDB(t))

	result, err := svc.Import([][]string{}, nil, "1")
	if err != nil {
		t.Fatalf("import empty file: %v", err)
	}
	if result.Applied {
		t.Fatal("expected applied=false for empty import")
	}
	if !hasHostImportError(result.Errors, "import.file.empty") {
		t.Fatalf("expected import.file.empty, got %+v", result.Errors)
	}
}

func TestHostServiceImportCreatesAndUpdatesByIP(t *testing.T) {
	db := setupHostImportExportTestDB(t)
	svc := NewHostService(db, testBizScopeReader{})

	existingLabels, _ := json.Marshal([]LabelEntry{{Key: "env", Val: "test"}})
	existing := Host{
		Hostname:    "existing",
		IP:          "10.0.1.1",
		SSHPort:     22,
		OS:          "linux",
		LabelValues: existingLabels,
		Status:      "online",
		CreatedBy:   "seed",
		UpdatedBy:   "seed",
	}
	if err := db.Create(&existing).Error; err != nil {
		t.Fatalf("seed existing host: %v", err)
	}
	if err := db.Exec(
		"INSERT INTO biz_business_scope (id, code, name) VALUES (?, ?, ?)",
		7,
		"scope-prod",
		"Production",
	).Error; err != nil {
		t.Fatalf("seed business scope: %v", err)
	}

	records := [][]string{
		hostImportExportHeaders(),
		{"existing-updated", "10.0.1.1", "", "linux", "Ubuntu 24.04", "8", "16", "200", "env=prod, region=cn-east-1", "", "12", "ops", "", "updated"},
		{"new-host", "10.0.1.2", "2222", "linux", "Ubuntu 22.04", "4", "8", "100", "env=prod", "scope-prod", "13", "owner", "", "created"},
	}

	result, err := svc.Import(records, nil, "99")
	if err != nil {
		t.Fatalf("import hosts: %v", err)
	}
	if !result.Applied || result.Created != 1 || result.Updated != 1 || result.Failed != 0 {
		t.Fatalf("unexpected import result: %+v", result)
	}

	var updated Host
	if err := db.Where("ip = ?", "10.0.1.1").First(&updated).Error; err != nil {
		t.Fatalf("load updated host: %v", err)
	}
	if updated.Hostname != "existing-updated" || updated.Status != "online" || updated.UpdatedBy != "99" {
		t.Fatalf("unexpected updated host: %+v", updated)
	}

	var created Host
	if err := db.Where("ip = ?", "10.0.1.2").First(&created).Error; err != nil {
		t.Fatalf("load created host: %v", err)
	}
	if created.Status != "assigned" || created.BusinessScopeID != 7 || created.BusinessScopeCode != "scope-prod" {
		t.Fatalf("unexpected created host scope/status: %+v", created)
	}
	if created.SSHPort != 2222 || created.CreatedBy != "99" || created.UpdatedBy != "99" {
		t.Fatalf("unexpected created host metadata: %+v", created)
	}
}

func TestHostServiceImportRejectsDuplicateIP(t *testing.T) {
	db := setupHostImportExportTestDB(t)
	svc := NewHostService(db)
	records := [][]string{
		hostImportExportHeaders(),
		{"host-a", "10.0.2.1", "22", "linux", "", "", "", "", "", "", "", "", "", ""},
		{"host-b", "10.0.2.1", "22", "linux", "", "", "", "", "", "", "", "", "", ""},
	}

	result, err := svc.Import(records, nil, "1")
	if err != nil {
		t.Fatalf("import duplicate IP: %v", err)
	}
	if result.Applied || !hasHostImportError(result.Errors, "import.duplicate.row.2") {
		t.Fatalf("expected duplicate row error, got %+v", result)
	}
	var count int64
	if err := db.Model(&Host{}).Count(&count).Error; err != nil {
		t.Fatalf("count hosts: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected no hosts written, got %d", count)
	}
}

func TestHostServiceImportRejectsInvalidStatus(t *testing.T) {
	db := setupHostImportExportTestDB(t)
	svc := NewHostService(db)
	records := [][]string{
		hostImportExportHeaders(),
		{"host-a", "10.0.3.1", "22", "linux", "", "", "", "", "", "", "", "", "unknown", ""},
	}

	result, err := svc.Import(records, nil, "1")
	if err != nil {
		t.Fatalf("import invalid status: %v", err)
	}
	if result.Applied || !hasHostImportError(result.Errors, "cmdbhost.status.invalid") {
		t.Fatalf("expected invalid status error, got %+v", result)
	}
}

func hasHostImportError(errors []impexp.ImportError, message string) bool {
	for _, item := range errors {
		if item.Message == message {
			return true
		}
	}
	return false
}
