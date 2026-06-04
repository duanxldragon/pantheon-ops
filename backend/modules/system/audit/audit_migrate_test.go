package system

import (
	"testing"

	"pantheon-ops/backend/internal/middleware"
)

func TestAuditService_MigrateBackfillsDerivedFields(t *testing.T) {
	db := setupAuditTestDB(t)
	if err := db.Exec(`CREATE TABLE system_log_oper (
id INTEGER PRIMARY KEY AUTOINCREMENT,
title TEXT NULL,
business_type INTEGER DEFAULT 0,
method TEXT NULL,
oper_name TEXT NULL,
oper_url TEXT NULL,
oper_ip TEXT NULL,
oper_param TEXT NULL,
json_result TEXT NULL,
status INTEGER DEFAULT 1,
error_msg TEXT NULL,
oper_time DATETIME NULL,
cost_time INTEGER DEFAULT 0
)`).Error; err != nil {
		t.Fatalf("create legacy operation log table: %v", err)
	}
	if err := db.Exec(`INSERT INTO system_log_oper (
title, business_type, method, oper_name, oper_url, oper_ip, oper_param, json_result, status, error_msg, oper_time, cost_time
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, datetime('now'), ?)`,
		"上传文件", 2, "POST", "admin", "/api/v1/system/upload", "127.0.0.1", `{"scope":"profile/avatar"}`, `{"code":400,"message":"upload.file.type_not_allowed"}`, 2, "upload.file.type_not_allowed", 18,
	).Error; err != nil {
		t.Fatalf("seed legacy operation log row: %v", err)
	}

	service := NewAuditService(db)
	if err := service.Migrate(); err != nil {
		t.Fatalf("migrate audit: %v", err)
	}

	var log middleware.SystemLogOper
	if err := db.First(&log).Error; err != nil {
		t.Fatalf("reload operation log: %v", err)
	}
	if log.SourceDomain != operationLogSourceDomainConfig {
		t.Fatalf("expected source domain %s, got %s", operationLogSourceDomainConfig, log.SourceDomain)
	}
	if log.SourcePage != operationLogSourcePageUpload {
		t.Fatalf("expected source page %s, got %s", operationLogSourcePageUpload, log.SourcePage)
	}
	if log.FailureCategory != operationLogFailureValidation {
		t.Fatalf("expected failure category %s, got %s", operationLogFailureValidation, log.FailureCategory)
	}
	if !db.Migrator().HasIndex(&middleware.SystemLogOper{}, "idx_system_log_oper_source_domain_page") {
		t.Fatalf("expected index idx_system_log_oper_source_domain_page to exist")
	}
	if !db.Migrator().HasIndex(&middleware.SystemLogOper{}, "idx_system_log_oper_source_page") {
		t.Fatalf("expected index idx_system_log_oper_source_page to exist")
	}
	if !db.Migrator().HasIndex(&middleware.SystemLogOper{}, "idx_system_log_oper_failure_category") {
		t.Fatalf("expected index idx_system_log_oper_failure_category to exist")
	}
}
