package system

import (
	"database/sql"
	"fmt"
	"strings"
	"testing"
	"time"

	"pantheon-ops/backend/internal/middleware"
)

const auditBenchmarkRowCount = 20000

func BenchmarkAuditServiceListOperationLogs_Unfiltered(b *testing.B) {
	service := setupAuditBenchmarkService(b, auditBenchmarkRowCount)
	query := &OperationLogQuery{
		Page:     1,
		PageSize: 20,
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		page, err := service.ListOperationLogs(query)
		if err != nil {
			b.Fatalf("list operation logs: %v", err)
		}
		if len(page.Items) == 0 {
			b.Fatal("expected benchmark query to return rows")
		}
	}
}

func BenchmarkAuditServiceListOperationLogs_FilterBySourceDomainPage(b *testing.B) {
	service := setupAuditBenchmarkService(b, auditBenchmarkRowCount)
	query := &OperationLogQuery{
		SourceDomain: operationLogSourceDomainConfig,
		SourcePage:   operationLogSourcePageUpload,
		Page:         1,
		PageSize:     20,
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		page, err := service.ListOperationLogs(query)
		if err != nil {
			b.Fatalf("list operation logs with source filters: %v", err)
		}
		if len(page.Items) == 0 {
			b.Fatal("expected benchmark query to return rows")
		}
	}
}

func BenchmarkAuditServiceListOperationLogs_FilterByFailureCategory(b *testing.B) {
	service := setupAuditBenchmarkService(b, auditBenchmarkRowCount)
	query := &OperationLogQuery{
		FailureCategory: operationLogFailureValidation,
		Page:            1,
		PageSize:        20,
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		page, err := service.ListOperationLogs(query)
		if err != nil {
			b.Fatalf("list operation logs with failure filter: %v", err)
		}
		if len(page.Items) == 0 {
			b.Fatal("expected benchmark query to return rows")
		}
	}
}

func TestAuditService_QueryPlanUsesDerivedIndexes(t *testing.T) {
	service := setupAuditBenchmarkService(t, auditBenchmarkRowCount)

	sourcePlan := explainOperationLogQueryPlan(t, service,
		`SELECT id FROM system_log_oper WHERE source_domain = ? AND source_page = ? ORDER BY id DESC LIMIT 20`,
		operationLogSourceDomainConfig,
		operationLogSourcePageUpload,
	)
	if !containsAnyPlanToken(sourcePlan, "idx_system_log_oper_source_domain_page") {
		t.Fatalf("expected source query plan to use composite index, got %v", sourcePlan)
	}

	failurePlan := explainOperationLogQueryPlan(t, service,
		`SELECT id FROM system_log_oper WHERE failure_category = ? ORDER BY id DESC LIMIT 20`,
		operationLogFailureValidation,
	)
	if !containsAnyPlanToken(failurePlan, "idx_system_log_oper_failure_category") {
		t.Fatalf("expected failure query plan to use failure index, got %v", failurePlan)
	}
}

func setupAuditBenchmarkService(tb testing.TB, rowCount int) *AuditService {
	tb.Helper()

	db := setupAuditTestDBForName(tb, fmt.Sprintf("%s_bench_%d", sanitizeBenchmarkName(tb.Name()), rowCount))
	service := NewAuditService(db)
	if err := service.Migrate(); err != nil {
		tb.Fatalf("migrate audit benchmark db: %v", err)
	}

	rows := make([]middleware.SystemLogOper, 0, rowCount)
	baseTime := time.Date(2026, 4, 24, 12, 0, 0, 0, time.UTC)
	for index := 0; index < rowCount; index++ {
		domain, page, failureCategory, operURL, status, errorMsg, jsonResult := buildAuditBenchmarkRow(index)
		rows = append(rows, middleware.SystemLogOper{
			Title:           fmt.Sprintf("审计基准-%d", index%12),
			BusinessType:    (index % 4) + 1,
			Method:          "POST",
			OperName:        fmt.Sprintf("user-%d", index%25),
			OperURL:         operURL,
			OperIP:          fmt.Sprintf("10.0.%d.%d", (index/255)%255, index%255),
			SourceDomain:    domain,
			SourcePage:      page,
			OperParam:       `{"from":"benchmark"}`,
			JsonResult:      jsonResult,
			Status:          status,
			FailureCategory: failureCategory,
			ErrorMsg:        errorMsg,
			OperTime:        baseTime.Add(time.Duration(index) * time.Second),
			CostTime:        int64((index % 40) + 5),
		})
	}
	if err := db.CreateInBatches(rows, 1000).Error; err != nil {
		tb.Fatalf("seed benchmark rows: %v", err)
	}
	return service
}

func buildAuditBenchmarkRow(index int) (domain string, page string, failureCategory string, operURL string, status int, errorMsg string, jsonResult string) {
	switch index % 5 {
	case 0:
		return operationLogSourceDomainConfig, operationLogSourcePageUpload, operationLogFailureValidation,
			fmt.Sprintf("/api/v1/system/upload/%d", index), 2, "upload.file.type_not_allowed", `{"code":400,"message":"upload.file.type_not_allowed"}`
	case 1:
		return operationLogSourceDomainIAM, operationLogSourcePageUser, operationLogFailurePermission,
			fmt.Sprintf("/api/v1/system/user/%d", index), 2, "permission.denied", `{"code":403,"message":"permission.denied"}`
	case 2:
		return operationLogSourceDomainAuth, operationLogSourcePageSession, operationLogFailureAuth,
			fmt.Sprintf("/api/v1/system/session/%d", index), 2, "auth.token.invalid", `{"code":401,"message":"auth.token.invalid"}`
	case 3:
		return operationLogSourceDomainAudit, operationLogSourcePageOperationLog, "",
			fmt.Sprintf("/api/v1/system/operation-log/%d", index), 1, "", `{"code":200,"message":"success"}`
	default:
		return operationLogSourceDomainPlatform, operationLogSourcePageDashboard, operationLogFailureServer,
			fmt.Sprintf("/api/v1/platform/dashboard/%d", index), 2, "database.not_initialized", `{"code":500,"message":"database.not_initialized"}`
	}
}

func explainOperationLogQueryPlan(t *testing.T, service *AuditService, query string, args ...any) []string {
	t.Helper()

	rows, err := service.db.Raw("EXPLAIN "+query, args...).Rows()
	if err != nil {
		t.Fatalf("explain query plan: %v", err)
	}
	defer rows.Close()

	plan, err := collectOperationLogQueryPlan(rows)
	if err != nil {
		t.Fatalf("collect query plan: %v", err)
	}
	if len(plan) == 0 {
		t.Fatal("expected query plan rows")
	}
	return plan
}

func collectOperationLogQueryPlan(rows *sql.Rows) ([]string, error) {
	cols, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("get explain columns: %w", err)
	}
	plan := make([]string, 0, 4)
	for rows.Next() {
		row, err := scanOperationLogQueryPlanRow(rows, cols)
		if err != nil {
			return nil, err
		}
		plan = append(plan, row)
	}
	return plan, nil
}

func scanOperationLogQueryPlanRow(rows *sql.Rows, cols []string) (string, error) {
	values := make([]any, len(cols))
	valuePtrs := make([]any, len(cols))
	for i := range cols {
		valuePtrs[i] = &values[i]
	}
	if err := rows.Scan(valuePtrs...); err != nil {
		return "", fmt.Errorf("scan query plan row: %w", err)
	}

	parts := make([]string, 0, len(cols))
	for _, value := range values {
		part := formatOperationLogQueryPlanValue(value)
		if part != "" {
			parts = append(parts, part)
		}
	}
	return strings.Join(parts, " "), nil
}

func formatOperationLogQueryPlanValue(value any) string {
	switch v := value.(type) {
	case nil:
		return ""
	case []byte:
		return string(v)
	default:
		return fmt.Sprintf("%v", v)
	}
}

func containsAnyPlanToken(plan []string, token string) bool {
	for _, detail := range plan {
		if strings.Contains(detail, token) {
			return true
		}
	}
	return false
}

func sanitizeBenchmarkName(name string) string {
	replacer := strings.NewReplacer("/", "_", "\\", "_", " ", "_", ":", "_", "(", "_", ")", "_")
	return replacer.Replace(name)
}
