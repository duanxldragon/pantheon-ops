package system

import (
	"testing"
	"time"

	"pantheon-platform/backend/internal/middleware"
	"pantheon-platform/backend/pkg/testmysql"

	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupAuditTestDBForName(tb testing.TB, _ string) *gorm.DB {
	tb.Helper()

	t, ok := tb.(*testing.T)
	if !ok {
		tb.Fatalf("audit mysql test helper requires *testing.T")
	}
	db := testmysql.Open(t)
	sessionDB := db.Session(&gorm.Session{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	return sessionDB
}

func setupAuditTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	return setupAuditTestDBForName(t, t.Name())
}

func TestAuditService_ExportOperationLogsIncludesDerivedColumns(t *testing.T) {
	db := setupAuditTestDB(t)
	service := NewAuditService(db)
	if err := service.Migrate(); err != nil {
		t.Fatalf("migrate audit: %v", err)
	}

	seed := []middleware.SystemLogOper{
		{
			Title:      "上传文件",
			Method:     "POST",
			OperName:   "admin",
			OperURL:    "/api/v1/system/upload",
			OperIP:     "127.0.0.1",
			Status:     2,
			ErrorMsg:   "upload.file.type_not_allowed",
			JsonResult: `{"code":400,"message":"upload.file.type_not_allowed"}`,
			OperTime:   time.Date(2026, 4, 24, 12, 0, 0, 0, time.UTC),
			CostTime:   18,
		},
		{
			Title:      "删除用户",
			Method:     "DELETE",
			OperName:   "admin",
			OperURL:    "/api/v1/system/user/2",
			OperIP:     "127.0.0.1",
			Status:     1,
			ErrorMsg:   "",
			JsonResult: `{"code":200,"message":"success"}`,
			OperTime:   time.Date(2026, 4, 24, 12, 1, 0, 0, time.UTC),
			CostTime:   21,
		},
	}
	if err := db.Create(&seed).Error; err != nil {
		t.Fatalf("seed audit logs: %v", err)
	}

	file, err := service.ExportOperationLogs(&OperationLogQuery{})
	if err != nil {
		t.Fatalf("export audit logs: %v", err)
	}

	expectedHeaders := []string{"title", "businessType", "sourceDomain", "sourcePage", "method", "operName", "operUrl", "operIp", "status", "failureCategory", "errorMsg", "operTime", "costTime"}
	if len(file.Headers) != len(expectedHeaders) {
		t.Fatalf("expected %d headers, got %d", len(expectedHeaders), len(file.Headers))
	}
	for index, header := range expectedHeaders {
		if file.Headers[index] != header {
			t.Fatalf("expected header %d to be %s, got %s", index, header, file.Headers[index])
		}
	}

	if len(file.Rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(file.Rows))
	}

	if file.Rows[1][0] != "上传文件" {
		t.Fatalf("expected latest row first, got %s", file.Rows[1][0])
	}

	if file.Rows[0][2] != operationLogSourceDomainIAM {
		t.Fatalf("expected success row source domain %s, got %s", operationLogSourceDomainIAM, file.Rows[0][2])
	}
	if file.Rows[0][3] != operationLogSourcePageUser {
		t.Fatalf("expected success row source page %s, got %s", operationLogSourcePageUser, file.Rows[0][3])
	}
	if file.Rows[0][9] != "" {
		t.Fatalf("expected success row failure category empty, got %s", file.Rows[0][9])
	}

	if file.Rows[1][2] != operationLogSourceDomainConfig {
		t.Fatalf("expected upload row source domain %s, got %s", operationLogSourceDomainConfig, file.Rows[1][2])
	}
	if file.Rows[1][3] != operationLogSourcePageUpload {
		t.Fatalf("expected upload row source page %s, got %s", operationLogSourcePageUpload, file.Rows[1][3])
	}
	if file.Rows[1][9] != operationLogFailureValidation {
		t.Fatalf("expected upload row failure category %s, got %s", operationLogFailureValidation, file.Rows[1][9])
	}
}

func TestAuditService_ExportOperationLogsRespectsDerivedFilters(t *testing.T) {
	db := setupAuditTestDB(t)
	service := NewAuditService(db)
	if err := service.Migrate(); err != nil {
		t.Fatalf("migrate audit: %v", err)
	}

	seed := []middleware.SystemLogOper{
		{
			Title:           "上传文件",
			Method:          "POST",
			OperName:        "admin",
			OperURL:         "/api/v1/system/upload",
			OperIP:          "127.0.0.1",
			SourceDomain:    operationLogSourceDomainConfig,
			SourcePage:      operationLogSourcePageUpload,
			Status:          2,
			FailureCategory: operationLogFailureValidation,
			ErrorMsg:        "upload.file.type_not_allowed",
			JsonResult:      `{"code":400,"message":"upload.file.type_not_allowed"}`,
			OperTime:        time.Now().UTC(),
			CostTime:        18,
		},
		{
			Title:           "删除用户",
			Method:          "DELETE",
			OperName:        "admin",
			OperURL:         "/api/v1/system/user/2",
			OperIP:          "127.0.0.1",
			SourceDomain:    operationLogSourceDomainIAM,
			SourcePage:      operationLogSourcePageUser,
			Status:          2,
			FailureCategory: operationLogFailurePermission,
			ErrorMsg:        "permission.denied",
			JsonResult:      `{"code":403,"message":"permission.denied"}`,
			OperTime:        time.Now().UTC(),
			CostTime:        21,
		},
	}
	if err := db.Create(&seed).Error; err != nil {
		t.Fatalf("seed audit logs: %v", err)
	}

	file, err := service.ExportOperationLogs(&OperationLogQuery{
		SourceDomain:    operationLogSourceDomainConfig,
		SourcePage:      operationLogSourcePageUpload,
		FailureCategory: operationLogFailureValidation,
	})
	if err != nil {
		t.Fatalf("export filtered audit logs: %v", err)
	}

	if len(file.Rows) != 1 {
		t.Fatalf("expected 1 filtered row, got %d", len(file.Rows))
	}
	if file.Rows[0][0] != "上传文件" {
		t.Fatalf("expected filtered row title 上传文件, got %s", file.Rows[0][0])
	}
}

func TestAuditService_ListOperationLogsRespectsDerivedFilters(t *testing.T) {
	db := setupAuditTestDB(t)
	service := NewAuditService(db)
	if err := service.Migrate(); err != nil {
		t.Fatalf("migrate audit: %v", err)
	}

	seed := []middleware.SystemLogOper{
		{
			Title:           "上传文件",
			Method:          "POST",
			OperName:        "admin",
			OperURL:         "/api/v1/system/upload",
			OperIP:          "127.0.0.1",
			SourceDomain:    operationLogSourceDomainConfig,
			SourcePage:      operationLogSourcePageUpload,
			Status:          2,
			FailureCategory: operationLogFailureValidation,
			ErrorMsg:        "upload.file.type_not_allowed",
			JsonResult:      `{"code":400,"message":"upload.file.type_not_allowed"}`,
			OperTime:        time.Now().UTC(),
			CostTime:        18,
		},
		{
			Title:           "删除用户",
			Method:          "DELETE",
			OperName:        "admin",
			OperURL:         "/api/v1/system/user/2",
			OperIP:          "127.0.0.1",
			SourceDomain:    operationLogSourceDomainIAM,
			SourcePage:      operationLogSourcePageUser,
			Status:          2,
			FailureCategory: operationLogFailurePermission,
			ErrorMsg:        "permission.denied",
			JsonResult:      `{"code":403,"message":"permission.denied"}`,
			OperTime:        time.Now().UTC(),
			CostTime:        21,
		},
	}
	if err := db.Create(&seed).Error; err != nil {
		t.Fatalf("seed audit logs: %v", err)
	}

	page, err := service.ListOperationLogs(&OperationLogQuery{
		SourceDomain:    operationLogSourceDomainConfig,
		SourcePage:      operationLogSourcePageUpload,
		FailureCategory: operationLogFailureValidation,
		Page:            1,
		PageSize:        10,
	})
	if err != nil {
		t.Fatalf("list filtered audit logs: %v", err)
	}

	if page.Total != 1 {
		t.Fatalf("expected total 1 filtered row, got %d", page.Total)
	}
	if len(page.Items) != 1 {
		t.Fatalf("expected 1 filtered item, got %d", len(page.Items))
	}
	if page.Items[0].Title != "上传文件" {
		t.Fatalf("expected filtered row title 上传文件, got %s", page.Items[0].Title)
	}
	if page.Items[0].SourceDomain != operationLogSourceDomainConfig {
		t.Fatalf("expected source domain %s, got %s", operationLogSourceDomainConfig, page.Items[0].SourceDomain)
	}
	if page.Items[0].SourcePage != operationLogSourcePageUpload {
		t.Fatalf("expected source page %s, got %s", operationLogSourcePageUpload, page.Items[0].SourcePage)
	}
	if page.Items[0].FailureCategory != operationLogFailureValidation {
		t.Fatalf("expected failure category %s, got %s", operationLogFailureValidation, page.Items[0].FailureCategory)
	}
}

func TestAuditService_ListOperationLogsDoesNotBackfillDuringQuery(t *testing.T) {
	db := setupAuditTestDB(t)
	service := NewAuditService(db)
	if err := service.Migrate(); err != nil {
		t.Fatalf("migrate audit: %v", err)
	}

	row := middleware.SystemLogOper{
		Title:           "历史脏数据",
		Method:          "POST",
		OperName:        "admin",
		OperURL:         "/api/v1/system/upload",
		OperIP:          "127.0.0.1",
		Status:          2,
		ErrorMsg:        "upload.file.type_not_allowed",
		JsonResult:      `{"code":400,"message":"upload.file.type_not_allowed"}`,
		SourceDomain:    "",
		SourcePage:      "",
		FailureCategory: "",
		OperTime:        time.Now().UTC(),
		CostTime:        18,
	}
	if err := db.Create(&row).Error; err != nil {
		t.Fatalf("seed legacy-like audit row: %v", err)
	}

	page, err := service.ListOperationLogs(&OperationLogQuery{
		SourceDomain:    operationLogSourceDomainConfig,
		SourcePage:      operationLogSourcePageUpload,
		FailureCategory: operationLogFailureValidation,
		Page:            1,
		PageSize:        10,
	})
	if err != nil {
		t.Fatalf("list filtered audit logs: %v", err)
	}
	if page.Total != 0 {
		t.Fatalf("expected no rows because query path no longer backfills, got %d", page.Total)
	}

	var reloaded middleware.SystemLogOper
	if err := db.First(&reloaded, row.ID).Error; err != nil {
		t.Fatalf("reload audit row: %v", err)
	}
	if reloaded.SourceDomain != "" || reloaded.SourcePage != "" || reloaded.FailureCategory != "" {
		t.Fatalf("expected query path to avoid backfill, got domain=%s page=%s failure=%s", reloaded.SourceDomain, reloaded.SourcePage, reloaded.FailureCategory)
	}
}
