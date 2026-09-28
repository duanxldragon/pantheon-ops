package system

import (
	"testing"
	"time"

	"github.com/duanxldragon/pantheon-base/backend/internal/middleware"
	"github.com/duanxldragon/pantheon-base/backend/pkg/tenant"

	"gorm.io/gorm"
)

// Queue-5 audit-slice isolation tests: a tenant subject must only ever read,
// export or delete the audit rows of its own tenant; platform-global subjects
// keep full visibility; compat behavior is unchanged (no filter, no stamp).
func TestAuditTenantIsolation_ListPinnedToRequestTenant(t *testing.T) {
	db := setupAuditTestDB(t)
	if err := db.AutoMigrate(&middleware.SystemLogOper{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	seed := []middleware.SystemLogOper{
		{TenantID: 101, Title: "tenant-101-op", OperURL: "/api/v1/system/dict/type", OperTime: time.Now()},
		{TenantID: 202, Title: "tenant-202-op", OperURL: "/api/v1/system/dict/type", OperTime: time.Now()},
		{TenantID: 0, Title: "platform-op", OperURL: "/api/v1/system/setting", OperTime: time.Now()},
	}
	if err := db.Create(&seed).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}
	service := NewAuditService(db)

	// Tenant 101 subject sees only its own rows — never 202, never platform rows.
	page, err := service.ListOperationLogs(&OperationLogQuery{}, &tenant.Context{TenantID: 101, Mode: tenant.ModeMulti})
	if err != nil {
		t.Fatalf("list as tenant 101: %v", err)
	}
	if page.Total != 1 || page.Items[0].Title != "tenant-101-op" || page.Items[0].TenantID != 101 {
		t.Fatalf("tenant 101 should see exactly its own row, got total=%d", page.Total)
	}

	// Platform-global subject (nil ctx = no filter) keeps full visibility.
	pageGlobal, err := service.ListOperationLogs(&OperationLogQuery{}, nil)
	if err != nil {
		t.Fatalf("list as platform: %v", err)
	}
	if pageGlobal.Total != 3 {
		t.Fatalf("platform subject should see all 3 rows, got %d", pageGlobal.Total)
	}

	// The success-count aggregate must honor the same tenant boundary.
	if page.SuccessCount+page.FailedCount != 1 {
		t.Fatalf("aggregate must be tenant-scoped, got success=%d failed=%d", page.SuccessCount, page.FailedCount)
	}
}

func TestAuditTenantIsolation_GetRejectsForeignRow(t *testing.T) {
	db := setupAuditTestDB(t)
	if err := db.AutoMigrate(&middleware.SystemLogOper{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	row := middleware.SystemLogOper{TenantID: 202, Title: "secret-202", OperTime: time.Now()}
	if err := db.Create(&row).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}
	service := NewAuditService(db)

	if _, err := service.GetOperationLog(row.ID, &tenant.Context{TenantID: 101, Mode: tenant.ModeMulti}); err == nil {
		t.Fatalf("tenant 101 must not read tenant 202 audit detail")
	}
	if _, err := service.GetOperationLog(row.ID, &tenant.Context{TenantID: 202, Mode: tenant.ModeMulti}); err != nil {
		t.Fatalf("tenant 202 must read its own row: %v", err)
	}
	if _, err := service.GetOperationLog(row.ID, nil); err != nil {
		t.Fatalf("platform subject must read any row: %v", err)
	}
}

func TestAuditTenantIsolation_DeleteAndBatchPinnedToTenant(t *testing.T) {
	db := setupAuditTestDB(t)
	if err := db.AutoMigrate(&middleware.SystemLogOper{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	rows := []middleware.SystemLogOper{
		{TenantID: 101, Title: "own", OperTime: time.Now()},
		{TenantID: 202, Title: "foreign", OperTime: time.Now()},
	}
	if err := db.Create(&rows).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}
	service := NewAuditService(db)
	ctx101 := &tenant.Context{TenantID: 101, Mode: tenant.ModeMulti}

	// Single delete of a foreign row must be a no-op (scoped delete, not an error leak).
	if err := service.DeleteOperationLog(rows[1].ID, ctx101); err != nil {
		t.Fatalf("foreign delete should fail silently via scope: %v", err)
	}
	var count202 int64
	db.Model(&middleware.SystemLogOper{}).Where("tenant_id = ?", 202).Count(&count202)
	if count202 != 1 {
		t.Fatalf("tenant 202 row must survive foreign delete")
	}

	// Batch delete skips ids outside the tenant scope.
	deleted, err := service.BatchDeleteOperationLogs([]uint64{rows[0].ID, rows[1].ID}, ctx101)
	if err != nil {
		t.Fatalf("batch delete: %v", err)
	}
	if deleted != 1 {
		t.Fatalf("tenant 101 may only delete its own row, deleted %d", deleted)
	}
}

func TestAuditTenantIsolation_ExportPinnedToTenant(t *testing.T) {
	db := setupAuditTestDB(t)
	if err := db.AutoMigrate(&middleware.SystemLogOper{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	seed := []middleware.SystemLogOper{
		{TenantID: 101, Title: "tenant-101-export", OperTime: time.Now()},
		{TenantID: 202, Title: "tenant-202-export", OperTime: time.Now()},
	}
	if err := db.Create(&seed).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}
	service := NewAuditService(db)

	file, err := service.ExportOperationLogs(&OperationLogQuery{}, &tenant.Context{TenantID: 101, Mode: tenant.ModeMulti})
	if err != nil {
		t.Fatalf("export as tenant 101: %v", err)
	}
	for _, row := range file.Rows {
		// operUrl column is index 7; the title (index 1) identifies the row.
		if row[1] == "tenant-202-export" {
			t.Fatalf("tenant 101 export must not contain tenant 202 rows")
		}
	}
	if len(file.Rows) != 1 {
		t.Fatalf("tenant 101 export should contain exactly 1 row, got %d", len(file.Rows))
	}
}

func TestAuditTenantIsolation_CompatUnchanged(t *testing.T) {
	db := setupAuditTestDB(t)
	if err := db.AutoMigrate(&middleware.SystemLogOper{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	seed := []middleware.SystemLogOper{
		{TenantID: 0, Title: "compat-row", OperTime: time.Now()},
	}
	if err := db.Create(&seed).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}
	service := NewAuditService(db)

	// Compat/legacy context (nil): no filter applied — single-tenant regression guarantee.
	page, err := service.ListOperationLogs(&OperationLogQuery{}, nil)
	if err != nil {
		t.Fatalf("compat list: %v", err)
	}
	if page.Total != 1 {
		t.Fatalf("compat mode must keep unfiltered behavior, got %d", page.Total)
	}
}

func TestAuditTenantIsolation_TenantIDFilterGuard(t *testing.T) {
	db := setupAuditTestDB(t)
	if err := db.AutoMigrate(&middleware.SystemLogOper{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	seed := []middleware.SystemLogOper{
		{TenantID: 101, Title: "own", OperTime: time.Now()},
		{TenantID: 202, Title: "foreign", OperTime: time.Now()},
	}
	if err := db.Create(&seed).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}
	service := NewAuditService(db)

	// Even if a tenant subject somehow passes a tenantIdFilter, the hard
	// tenant scope composes with it (AND), so it can only ever narrow.
	filter := uint64(202)
	page, err := service.ListOperationLogs(&OperationLogQuery{TenantIDFilter: filter}, &tenant.Context{TenantID: 101, Mode: tenant.ModeMulti})
	if err != nil {
		t.Fatalf("list with filter: %v", err)
	}
	if page.Total != 0 {
		t.Fatalf("filter 202 under tenant-101 scope must return nothing, got %d", page.Total)
	}

	// A platform subject can use the filter for cross-tenant queries.
	pageGlobal, err := service.ListOperationLogs(&OperationLogQuery{TenantIDFilter: filter}, nil)
	if err != nil {
		t.Fatalf("platform filtered list: %v", err)
	}
	if pageGlobal.Total != 1 || pageGlobal.Items[0].Title != "foreign" {
		t.Fatalf("platform subject filter should find the 202 row, got %d", pageGlobal.Total)
	}
}

// compile-time guard: keep the service on *gorm.DB plumbing
var _ = gorm.ErrRecordNotFound
