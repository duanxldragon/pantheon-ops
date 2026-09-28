package system

import (
	"testing"
	"time"

	"github.com/duanxldragon/pantheon-base/backend/internal/middleware"
)

// ---- normalizeRetentionOptions ----

func TestNormalizeRetentionOptions_DeduplicatesAndSorts(t *testing.T) {
	got := normalizeRetentionOptions([]int{30, 1, 7, 30, 1}, []int{1, 7, 30})
	if len(got) != 3 || got[0] != 1 || got[1] != 7 || got[2] != 30 {
		t.Fatalf("expected sorted unique [1 7 30], got %v", got)
	}
}

func TestNormalizeRetentionOptions_DropsNonPositiveValues(t *testing.T) {
	got := normalizeRetentionOptions([]int{0, -5, 7}, []int{1, 7, 30})
	if len(got) != 1 || got[0] != 7 {
		t.Fatalf("expected [7], got %v", got)
	}
}

func TestNormalizeRetentionOptions_EmptyResultFallsBack(t *testing.T) {
	fallback := []int{1, 7, 30}
	got := normalizeRetentionOptions([]int{0, -1}, fallback)
	if len(got) != 3 || got[0] != 1 || got[1] != 7 || got[2] != 30 {
		t.Fatalf("expected fallback [1 7 30], got %v", got)
	}
}

func TestNormalizeRetentionOptions_NilInputFallsBack(t *testing.T) {
	fallback := []int{7, 14}
	got := normalizeRetentionOptions(nil, fallback)
	if len(got) != 2 || got[0] != 7 || got[1] != 14 {
		t.Fatalf("expected fallback [7 14], got %v", got)
	}
}

// ---- normalizeOperationLogPageQuery ----

func TestNormalizeOperationLogPageQuery_Nil(t *testing.T) {
	page, pageSize := normalizeOperationLogPageQuery(nil)
	if page != 1 || pageSize != 10 {
		t.Fatalf("expected (1, 10), got (%d, %d)", page, pageSize)
	}
}

func TestNormalizeOperationLogPageQuery_ZeroValues(t *testing.T) {
	page, pageSize := normalizeOperationLogPageQuery(&OperationLogQuery{})
	if page != 1 || pageSize != 10 {
		t.Fatalf("expected (1, 10) for zero values, got (%d, %d)", page, pageSize)
	}
}

func TestNormalizeOperationLogPageQuery_CustomValues(t *testing.T) {
	page, pageSize := normalizeOperationLogPageQuery(&OperationLogQuery{Page: 2, PageSize: 50})
	if page != 2 || pageSize != 50 {
		t.Fatalf("expected (2, 50), got (%d, %d)", page, pageSize)
	}
}

func TestNormalizeOperationLogPageQuery_CapsPageSizeAt100(t *testing.T) {
	_, pageSize := normalizeOperationLogPageQuery(&OperationLogQuery{Page: 1, PageSize: 250})
	if pageSize != 100 {
		t.Fatalf("expected pageSize capped at 100, got %d", pageSize)
	}
}

// ---- normalizeOperationLogSort ----

func TestNormalizeOperationLogSort_Nil(t *testing.T) {
	column, desc := normalizeOperationLogSort(nil)
	if column != "id" || desc != true {
		t.Fatalf("expected (id, true), got (%s, %v)", column, desc)
	}
}

func TestNormalizeOperationLogSort_DefaultSort(t *testing.T) {
	column, desc := normalizeOperationLogSort(&OperationLogQuery{})
	if column != "id" || desc != true {
		t.Fatalf("expected (id, true) for empty query, got (%s, %v)", column, desc)
	}
}

func TestNormalizeOperationLogSort_WhitelistedFields(t *testing.T) {
	cases := []struct {
		sortField string
		want      string
	}{
		{sortField: "id", want: "id"},
		{sortField: "operTime", want: "oper_time"},
		{sortField: "oper_time", want: "oper_time"},
		{sortField: "status", want: "status"},
		{sortField: "businessType", want: "business_type"},
		{sortField: "business_type", want: "business_type"},
		{sortField: "operName", want: "oper_name"},
		{sortField: "oper_name", want: "oper_name"},
		{sortField: "title", want: "title"},
		{sortField: "sourceDomain", want: "source_domain"},
		{sortField: "source_domain", want: "source_domain"},
		{sortField: "failureCategory", want: "failure_category"},
		{sortField: "failure_category", want: "failure_category"},
	}
	for _, tc := range cases {
		column, _ := normalizeOperationLogSort(&OperationLogQuery{SortField: tc.sortField})
		if column != tc.want {
			t.Fatalf("expected %s to map to %s, got %s", tc.sortField, tc.want, column)
		}
	}
}

func TestNormalizeOperationLogSort_InvalidFieldFallsBackToIDDesc(t *testing.T) {
	column, desc := normalizeOperationLogSort(&OperationLogQuery{SortField: "oper_ip; DROP TABLE system_log_oper", SortOrder: "asc"})
	if column != "id" || desc != true {
		t.Fatalf("expected (id, true) fallback for unknown field, got (%s, %v)", column, desc)
	}
}

func TestNormalizeOperationLogSort_OrderOnlyDescWhenExplicit(t *testing.T) {
	column, desc := normalizeOperationLogSort(&OperationLogQuery{SortField: "operTime", SortOrder: "asc"})
	if column != "oper_time" || desc != false {
		t.Fatalf("expected (oper_time, false), got (%s, %v)", column, desc)
	}

	column, desc = normalizeOperationLogSort(&OperationLogQuery{SortField: "operTime", SortOrder: "DESC"})
	if column != "oper_time" || desc != true {
		t.Fatalf("expected case-insensitive desc, got (%s, %v)", column, desc)
	}
}

// ---- parseOperationLogTime ----

func TestParseOperationLogTime_EmptyIsNotParseable(t *testing.T) {
	if _, ok := parseOperationLogTime(""); ok {
		t.Fatal("expected empty value to be unparseable")
	}
	if _, ok := parseOperationLogTime("   "); ok {
		t.Fatal("expected whitespace value to be unparseable")
	}
}

func TestParseOperationLogTime_SupportsRFC3339(t *testing.T) {
	parsed, ok := parseOperationLogTime("2026-03-04T05:06:07Z")
	if !ok {
		t.Fatal("expected RFC3339 to parse")
	}
	if parsed.Year() != 2026 || parsed.Minute() != 6 {
		t.Fatalf("unexpected parsed time: %v", parsed)
	}
}

func TestParseOperationLogTime_SupportsDateTimeLayouts(t *testing.T) {
	if _, ok := parseOperationLogTime("2026-03-04 05:06"); !ok {
		t.Fatal("expected '2006-01-02 15:04' layout to parse")
	}
	if _, ok := parseOperationLogTime("2026-03-04 05:06:07"); !ok {
		t.Fatal("expected '2006-01-02 15:04:05' layout to parse")
	}
}

func TestParseOperationLogTime_RejectsGarbage(t *testing.T) {
	if _, ok := parseOperationLogTime("last-tuesday"); ok {
		t.Fatal("expected garbage value to be unparseable")
	}
}

// ---- parseOperationCleanupWindow ----

func TestParseOperationCleanupWindow_BothEmptyReturnsNilWindow(t *testing.T) {
	window, err := parseOperationCleanupWindow("", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if window != nil {
		t.Fatalf("expected nil window, got %+v", window)
	}
}

func TestParseOperationCleanupWindow_OneBoundOnlyIsInvalid(t *testing.T) {
	if _, err := parseOperationCleanupWindow("2026-01-01T00:00:00Z", ""); err == nil {
		t.Fatal("expected error for missing end")
	}
	if _, err := parseOperationCleanupWindow("", "2026-01-01T00:00:00Z"); err == nil {
		t.Fatal("expected error for missing start")
	}
}

func TestParseOperationCleanupWindow_MalformedTimestampsAreInvalid(t *testing.T) {
	if _, err := parseOperationCleanupWindow("garbage", "2026-01-01T00:00:00Z"); err == nil {
		t.Fatal("expected error for malformed start")
	}
	if _, err := parseOperationCleanupWindow("2026-01-01T00:00:00Z", "garbage"); err == nil {
		t.Fatal("expected error for malformed end")
	}
}

func TestParseOperationCleanupWindow_EndBeforeStartIsInvalid(t *testing.T) {
	_, err := parseOperationCleanupWindow(
		"2026-01-02T00:00:00Z",
		"2026-01-01T00:00:00Z",
	)
	if err == nil {
		t.Fatal("expected error when end precedes start")
	}
}

func TestParseOperationCleanupWindow_ValidWindowIsReturned(t *testing.T) {
	window, err := parseOperationCleanupWindow(
		"2026-01-01T00:00:00Z",
		"2026-01-02T00:00:00Z",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if window == nil {
		t.Fatal("expected non-nil window")
	}
	if window.StartedAt.IsZero() || window.EndedAt.IsZero() {
		t.Fatalf("expected parsed bounds, got %+v", window)
	}
	if !window.EndedAt.After(window.StartedAt) {
		t.Fatalf("expected end after start, got %+v", window)
	}
}

// ---- normalizeAuditLogIDs ----

func TestNormalizeAuditLogIDs_NilReturnsNil(t *testing.T) {
	if got := normalizeAuditLogIDs(nil); got != nil {
		t.Fatalf("expected nil for nil input, got %v", got)
	}
}

func TestNormalizeAuditLogIDs_DropsZerosAndDuplicates(t *testing.T) {
	got := normalizeAuditLogIDs([]uint64{9, 0, 4, 9, 0})
	if len(got) != 2 || got[0] != 9 || got[1] != 4 {
		t.Fatalf("expected [9 4], got %v", got)
	}
}

func TestNormalizeAuditLogIDs_AllZeroInputYieldsEmpty(t *testing.T) {
	got := normalizeAuditLogIDs([]uint64{0})
	if len(got) != 0 {
		t.Fatalf("expected empty result, got %v", got)
	}
}

// ---- operationLogToResp ----

func TestOperationLogToResp_TrimsAndFormatsFields(t *testing.T) {
	operTime := time.Date(2026, 2, 3, 4, 5, 6, 0, time.UTC)
	resp := operationLogToResp(middleware.SystemLogOper{
		ID:              12,
		RequestID:       "  req-1  ",
		Title:           "导出用户",
		BusinessType:    5,
		Method:          "POST",
		OperName:        "admin",
		OperURL:         "/api/v1/system/user/export",
		OperIP:          "127.0.0.1",
		SourceDomain:    " iam ",
		SourcePage:      " user ",
		FailureCategory: " validation ",
		OperTime:        operTime,
		CostTime:        33,
	})

	if resp.ID != 12 || resp.RequestID != "req-1" {
		t.Fatalf("unexpected id/request id: %+v", resp)
	}
	if resp.SourceDomain != "iam" || resp.SourcePage != "user" || resp.FailureCategory != "validation" {
		t.Fatalf("expected trimmed derived fields, got %+v", resp)
	}
	if resp.OperTime != "2026-02-03T04:05:06Z" {
		t.Fatalf("expected RFC3339 oper time, got %s", resp.OperTime)
	}
	if resp.CostTime != 33 || resp.BusinessType != 5 {
		t.Fatalf("unexpected numeric fields: %+v", resp)
	}
}
