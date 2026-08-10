package login

import (
	"testing"
	"time"
)

func TestRuntime_ListLoginLogsKeywordMatchesMultipleFields(t *testing.T) {
	db := setupTestDB(t)
	s := NewRuntime(db)

	now := time.Now().UTC()
	if err := db.Create(&[]SystemLogLogin{
		{Username: "zhangsan", Ipaddr: "10.0.0.8", LoginLocation: "Beijing", Status: 1, LoginTime: now},
		{Username: "lisi", Ipaddr: "192.168.1.5", LoginLocation: "Shanghai", Status: 1, LoginTime: now},
	}).Error; err != nil {
		t.Fatalf("seed login logs: %v", err)
	}

	cases := []struct {
		name    string
		keyword string
		want    int
	}{
		{name: "matches username", keyword: "zhangsan", want: 1},
		{name: "matches ip", keyword: "192.168", want: 1},
		{name: "matches location", keyword: "Beijing", want: 1},
		{name: "no match", keyword: "missing", want: 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := s.ListLoginLogs(&LoginLogQuery{Keyword: tc.keyword, Page: 1, PageSize: 10})
			if err != nil {
				t.Fatalf("list login logs: %v", err)
			}
			if len(resp.Items) != tc.want {
				t.Fatalf("keyword %q: expected %d rows, got %d", tc.keyword, tc.want, len(resp.Items))
			}
		})
	}
}
