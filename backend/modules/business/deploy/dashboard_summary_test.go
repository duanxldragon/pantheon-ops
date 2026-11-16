package deploy

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/duanxldragon/pantheon-base/backend/modules/business/cmdb"
	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
	"gorm.io/datatypes"
)

func TestDeployDashboardSummaryBoundScopeAndCancellation(t *testing.T) {
	db := setupDeployTestDB(t)
	svc := NewDeployService(db, cmdb.NewDeployCMDBCapability(db))
	hosts := []deployTestHost{{Hostname: "visible-dashboard", IP: "10.71.0.91", DeptID: 71}, {Hostname: "hidden-dashboard", IP: "10.72.0.92", DeptID: 72}}
	if err := db.Create(&hosts).Error; err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 12; i++ {
		row := DeployTask{Name: "dashboard-task", Status: TaskStatusFailed, BusinessScopeID: uint64(i%2 + 1), ExecutionSnapshot: datatypes.JSON(`{"password":"secret-marker"}`)}
		if err := db.Create(&row).Error; err != nil {
			t.Fatal(err)
		}
		if err := db.Create(&DeployTaskHost{TaskID: row.ID, HostID: hosts[i%2].ID}).Error; err != nil {
			t.Fatal(err)
		}
	}
	all, err := svc.DashboardSummary(context.Background(), 99, TaskStatusFailed, &common.DataScopeReq{Mode: "all"})
	if err != nil || len(all.Items) != 8 || !all.Partial {
		t.Fatalf("global bound: %+v %v", all, err)
	}
	scoped, err := svc.DashboardSummary(context.Background(), 3, TaskStatusFailed, &common.DataScopeReq{Mode: "dept", DeptID: 71})
	if err != nil || len(scoped.Items) != 2 || !scoped.Partial {
		t.Fatalf("restricted recent sample: %+v %v", scoped, err)
	}
	for _, row := range scoped.Items {
		if row.BusinessScopeID != 1 {
			t.Fatalf("hidden task leaked: %+v", row)
		}
	}
	bytes, _ := json.Marshal(all)
	if strings.Contains(string(bytes), "secret-marker") || strings.Contains(string(bytes), "executionSnapshot") {
		t.Fatal("task secret entered summary")
	}
	if _, err := svc.DashboardSummary(context.Background(), 8, "", nil); err == nil {
		t.Fatal("missing scope admitted")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := svc.DashboardSummary(ctx, 8, "", &common.DataScopeReq{Mode: "all"}); !errors.Is(err, context.Canceled) {
		t.Fatalf("pre-canceled query: %v", err)
	}
	list, err := svc.ListTasks(TaskQuery{Page: 1, PageSize: 10, BusinessScopeID: 1}, &common.DataScopeReq{Mode: "all"})
	if err != nil || len(list.Items) != 6 || list.Total != 6 {
		t.Fatalf("scope deep-link list filter: %+v %v", list, err)
	}
}

func TestDeployDashboardTimeWindowAndDrilldown(t *testing.T) {
	db := setupDeployTestDB(t)
	svc := NewDeployService(db, cmdb.NewDeployCMDBCapability(db))
	now := time.Now().UTC()
	for _, offset := range []time.Duration{-48 * time.Hour, -time.Hour, time.Hour} {
		row := DeployTask{Name: "time-window", Status: TaskStatusFailed, CreatedAt: now.Add(offset)}
		if err := db.Create(&row).Error; err != nil {
			t.Fatal(err)
		}
	}
	resp, err := svc.DashboardSummary(context.Background(), 8, TaskStatusFailed, &common.DataScopeReq{Mode: "all"})
	if err != nil || len(resp.Items) != 1 || resp.RangeStart == nil || resp.RangeEnd == nil {
		t.Fatalf("24h selection: %+v %v", resp, err)
	}
	if resp.RangeEnd.Sub(*resp.RangeStart) != 24*time.Hour {
		t.Fatal("range is not exactly 24h")
	}
	list, err := svc.ListTasks(TaskQuery{Page: 1, PageSize: 10, CreatedFrom: resp.RangeStart.Format(time.RFC3339Nano), CreatedTo: resp.RangeEnd.Format(time.RFC3339Nano)}, &common.DataScopeReq{Mode: "all"})
	if err != nil || len(list.Items) != 1 || list.Items[0].ID != resp.Items[0].ID {
		t.Fatalf("drilldown drift: %+v %v", list, err)
	}
	start := now.Truncate(time.Second).Add(-4 * time.Hour)
	end := start.Add(time.Hour)
	for _, stamp := range []time.Time{start, end} {
		if err := db.Create(&DeployTask{Name: "boundary", Status: TaskStatusFailed, CreatedAt: stamp}).Error; err != nil {
			t.Fatal(err)
		}
	}
	list, err = svc.ListTasks(TaskQuery{Page: 1, PageSize: 10, CreatedFrom: start.Format(time.RFC3339Nano), CreatedTo: end.Format(time.RFC3339Nano)}, &common.DataScopeReq{Mode: "all"})
	if err != nil || list.Total != 1 {
		t.Fatalf("half-open boundary: %+v %v", list, err)
	}
	for _, q := range []TaskQuery{{CreatedFrom: "invalid", CreatedTo: end.Format(time.RFC3339Nano)}, {CreatedFrom: end.Format(time.RFC3339Nano)}, {CreatedFrom: end.Format(time.RFC3339Nano), CreatedTo: start.Format(time.RFC3339Nano)}} {
		if _, err := svc.ListTasks(q, &common.DataScopeReq{Mode: "all"}); err == nil {
			t.Fatalf("invalid range admitted: %+v", q)
		}
	}
}
