package system

import (
	"testing"
	"time"

	"pantheon-base/internal/middleware"
)

func TestAuditService_ListOperationLogsKeywordMatchesMultipleFields(t *testing.T) {
	db := setupAuditTestDB(t)
	service := NewAuditService(db)
	if err := service.Migrate(); err != nil {
		t.Fatalf("migrate audit: %v", err)
	}

	seed := []middleware.SystemLogOper{
		{
			Title:     "上传文件",
			RequestID: "req-alpha-001",
			Method:    "POST",
			OperName:  "admin",
			OperURL:   "/api/v1/system/upload",
			Status:    1,
			OperTime:  time.Date(2026, 7, 15, 12, 0, 0, 0, time.UTC),
		},
		{
			Title:     "删除用户",
			RequestID: "req-beta-002",
			Method:    "DELETE",
			OperName:  "auditor",
			OperURL:   "/api/v1/system/user/2",
			Status:    1,
			OperTime:  time.Date(2026, 7, 15, 12, 1, 0, 0, time.UTC),
		},
	}
	if err := db.Create(&seed).Error; err != nil {
		t.Fatalf("seed audit logs: %v", err)
	}

	cases := []struct {
		name    string
		keyword string
		want    int
	}{
		{name: "matches title", keyword: "上传", want: 1},
		{name: "matches oper name", keyword: "auditor", want: 1},
		{name: "matches request id", keyword: "req-alpha", want: 1},
		{name: "no match", keyword: "missing", want: 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := service.ListOperationLogs(&OperationLogQuery{Keyword: tc.keyword, Page: 1, PageSize: 20})
			if err != nil {
				t.Fatalf("list operation logs: %v", err)
			}
			if len(resp.Items) != tc.want {
				t.Fatalf("keyword %q: expected %d rows, got %d", tc.keyword, tc.want, len(resp.Items))
			}
		})
	}
}
