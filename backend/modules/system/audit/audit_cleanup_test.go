package system

import (
	"testing"
	"time"

	"pantheon-ops/backend/internal/middleware"
)

func TestAuditService_CleanupOperationLogsUsesConfiguredRetentionOptions(t *testing.T) {
	db := setupAuditTestDB(t)
	service := NewAuditService(db)
	if err := service.Migrate(); err != nil {
		t.Fatalf("migrate audit: %v", err)
	}
	if err := db.Exec("CREATE TABLE IF NOT EXISTS system_setting (setting_key VARCHAR(191) PRIMARY KEY, setting_value TEXT)").Error; err != nil {
		t.Fatalf("create system_setting table: %v", err)
	}
	if err := db.Exec("INSERT INTO system_setting (setting_key, setting_value) VALUES ('audit.operation_log_retention_options', '[5,15]')").Error; err != nil {
		t.Fatalf("seed audit retention setting: %v", err)
	}
	if err := db.Create(&[]middleware.SystemLogOper{
		{Title: "old", OperURL: "/api/v1/system/user/1", OperTime: time.Now().UTC().AddDate(0, 0, -20)},
		{Title: "recent", OperURL: "/api/v1/system/user/2", OperTime: time.Now().UTC().AddDate(0, 0, -7)},
	}).Error; err != nil {
		t.Fatalf("seed operation logs: %v", err)
	}

	clearedCount, err := service.CleanupOperationLogs(15)
	if err != nil {
		t.Fatalf("cleanup operation logs with configured option: %v", err)
	}
	if clearedCount != 1 {
		t.Fatalf("expected to clean 1 operation log, got %d", clearedCount)
	}

	_, err = service.CleanupOperationLogs(7)
	if err == nil || err.Error() != "audit.operation_log.cleanup.days_invalid" {
		t.Fatalf("expected invalid retention days error, got %v", err)
	}
}

func TestAuditService_ListOperationLogsAppliesAutomaticRetention(t *testing.T) {
	db := setupAuditTestDB(t)
	service := NewAuditService(db)
	if err := service.Migrate(); err != nil {
		t.Fatalf("migrate audit: %v", err)
	}
	if err := db.Exec("CREATE TABLE IF NOT EXISTS system_setting (setting_key VARCHAR(191) PRIMARY KEY, setting_value TEXT)").Error; err != nil {
		t.Fatalf("create system_setting table: %v", err)
	}
	if err := db.Exec("INSERT INTO system_setting (setting_key, setting_value) VALUES ('audit.operation_log_retention_days', '5')").Error; err != nil {
		t.Fatalf("seed operation log retention days: %v", err)
	}
	if err := db.Create(&[]middleware.SystemLogOper{
		{Title: "expired", OperURL: "/api/v1/system/user/1", OperTime: time.Now().UTC().AddDate(0, 0, -20)},
		{Title: "retained", OperURL: "/api/v1/system/user/2", OperTime: time.Now().UTC().AddDate(0, 0, -2)},
	}).Error; err != nil {
		t.Fatalf("seed operation logs: %v", err)
	}

	resp, err := service.ListOperationLogs(&OperationLogQuery{Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("list operation logs: %v", err)
	}
	if resp.Total != 1 || len(resp.Items) != 1 || resp.Items[0].Title != "retained" {
		t.Fatalf("expected only retained operation log, got %+v", resp)
	}
}
