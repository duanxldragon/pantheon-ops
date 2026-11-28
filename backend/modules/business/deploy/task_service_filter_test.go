package deploy

import (
	"testing"

	"github.com/duanxldragon/pantheon-base/backend/modules/business/cmdb"
	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
)

// T20 S1: the instance detail page needs "recent deployments of this instance".
// The filter must narrow the list in SQL while leaving host data-scope
// visibility untouched, and it must not reveal that an instance has tasks the
// caller cannot see.
func TestListTasksFiltersByServiceAndInstanceWithinDataScope(t *testing.T) {
	db := setupDeployTestDB(t)
	svc := NewDeployService(db, cmdb.NewDeployCMDBCapability(db))
	deptScope := &common.DataScopeReq{Mode: common.DataScopeModeDept, DeptID: 20}

	pkg, err := svc.CreatePackage(CreatePackageRequest{
		Name: "nginx", Version: "1.26", InstallCommand: "apt-get install -y nginx", Status: "enabled",
	}, "1")
	if err != nil {
		t.Fatalf("create package: %v", err)
	}
	scope := deployTestBizScope{Code: "his-dev", Name: "HIS 开发", Status: "active", DeptID: 20}
	if err := db.Create(&scope).Error; err != nil {
		t.Fatalf("seed scope: %v", err)
	}
	host := deployTestHost{
		Hostname: "deploy-host-filter", IP: "10.40.0.71", OS: "linux", Status: "assigned",
		BusinessScopeID: scope.ID, BusinessScopeName: scope.Name, DeptID: 20,
	}
	if err := db.Create(&host).Error; err != nil {
		t.Fatalf("seed host: %v", err)
	}

	makeTask := func(name string, serviceID, instanceID uint64) uint64 {
		t.Helper()
		task, err := svc.CreateTask(CreateTaskRequest{
			Name: name, PackageID: pkg.ID, BusinessScopeID: scope.ID,
			TargetType: "host", TargetIDs: []uint64{host.ID}, ExecutorType: "manual",
		}, "1", deptScope)
		if err != nil {
			t.Fatalf("create task %s: %v", name, err)
		}
		// Link columns are written directly: admission of real service links is
		// covered by service_link_*_test.go, not by this list-filter test.
		if err := db.Model(&DeployTask{}).Where("id = ?", task.ID).
			Updates(map[string]any{"service_id": serviceID, "service_instance_id": instanceID}).Error; err != nil {
			t.Fatalf("link task %s: %v", name, err)
		}
		return task.ID
	}
	a1 := makeTask("a1", 11, 501)
	a2 := makeTask("a2", 11, 501)
	b1 := makeTask("b1", 11, 502)
	c1 := makeTask("c1", 12, 601)
	makeTask("unlinked", 0, 0)

	ids := func(resp *TaskListResponse) map[uint64]bool {
		out := map[uint64]bool{}
		for _, item := range resp.Items {
			out[item.ID] = true
		}
		return out
	}
	list := func(q TaskQuery, ds *common.DataScopeReq) *TaskListResponse {
		t.Helper()
		q.Page, q.PageSize = 1, 20
		resp, err := svc.ListTasks(q, ds)
		if err != nil {
			t.Fatalf("list %+v: %v", q, err)
		}
		return resp
	}

	cases := []struct {
		name  string
		query TaskQuery
		want  []uint64
	}{
		{"by instance", TaskQuery{ServiceInstanceID: 501}, []uint64{a1, a2}},
		{"other instance", TaskQuery{ServiceInstanceID: 502}, []uint64{b1}},
		{"by service", TaskQuery{ServiceID: 11}, []uint64{a1, a2, b1}},
		{"service and instance agree", TaskQuery{ServiceID: 11, ServiceInstanceID: 502}, []uint64{b1}},
		{"service and instance disagree", TaskQuery{ServiceID: 12, ServiceInstanceID: 501}, nil},
		{"unknown instance", TaskQuery{ServiceInstanceID: 999}, nil},
		{"other service only", TaskQuery{ServiceID: 12}, []uint64{c1}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resp := list(tc.query, deptScope)
			got := ids(resp)
			if int(resp.Total) != len(tc.want) || len(got) != len(tc.want) {
				t.Fatalf("total=%d items=%v, want %v", resp.Total, got, tc.want)
			}
			for _, id := range tc.want {
				if !got[id] {
					t.Fatalf("missing task %d in %v", id, got)
				}
			}
		})
	}

	t.Run("no filter keeps every visible task", func(t *testing.T) {
		if resp := list(TaskQuery{}, deptScope); resp.Total != 5 {
			t.Fatalf("expected 5 visible tasks without a filter, got %d", resp.Total)
		}
	})

	t.Run("unscoped caller filters in SQL with exact total", func(t *testing.T) {
		resp := list(TaskQuery{ServiceInstanceID: 501}, nil)
		if resp.Total != 2 || len(resp.Items) != 2 {
			t.Fatalf("expected 2 tasks for instance 501, got total=%d items=%d", resp.Total, len(resp.Items))
		}
	})

	t.Run("out-of-scope caller learns nothing about the instance", func(t *testing.T) {
		hidden := &common.DataScopeReq{Mode: common.DataScopeModeDept, DeptID: 10}
		for _, q := range []TaskQuery{{ServiceInstanceID: 501}, {ServiceID: 11}, {ServiceInstanceID: 999}} {
			resp := list(q, hidden)
			if resp.Total != 0 || len(resp.Items) != 0 {
				t.Fatalf("query %+v leaked %d tasks to an out-of-scope caller", q, resp.Total)
			}
		}
	})

	t.Run("paging respects the filter total", func(t *testing.T) {
		resp, err := svc.ListTasks(TaskQuery{ServiceID: 11, Page: 2, PageSize: 2}, deptScope)
		if err != nil {
			t.Fatalf("page 2: %v", err)
		}
		if resp.Total != 3 || len(resp.Items) != 1 {
			t.Fatalf("expected total 3 with 1 item on page 2, got total=%d items=%d", resp.Total, len(resp.Items))
		}
	})

	t.Run("export uses the same filter", func(t *testing.T) {
		file, err := svc.ExportTasks(TaskQuery{ServiceInstanceID: 502, Page: 1, PageSize: 20}, deptScope)
		if err != nil {
			t.Fatalf("export: %v", err)
		}
		if len(file.Rows) != 1 {
			t.Fatalf("expected 1 exported row for instance 502, got %d", len(file.Rows))
		}
	})
}

// T21 S1: the CMDB host detail lists the deployments that targeted the host.
// The filter narrows in SQL but never widens host data-scope visibility.
func TestListTasksFiltersByHostWithinDataScope(t *testing.T) {
	db := setupDeployTestDB(t)
	svc := NewDeployService(db, cmdb.NewDeployCMDBCapability(db))
	deptScope := &common.DataScopeReq{Mode: common.DataScopeModeDept, DeptID: 20}

	pkg, err := svc.CreatePackage(CreatePackageRequest{
		Name: "nginx", Version: "1.26", InstallCommand: "apt-get install -y nginx", Status: "enabled",
	}, "1")
	if err != nil {
		t.Fatalf("create package: %v", err)
	}
	scope := deployTestBizScope{Code: "host-filter", Name: "Host filter", Status: "active", DeptID: 20}
	if err := db.Create(&scope).Error; err != nil {
		t.Fatalf("seed scope: %v", err)
	}
	foreignScope := deployTestBizScope{Code: "host-filter-other", Name: "Other", Status: "active", DeptID: 30}
	if err := db.Create(&foreignScope).Error; err != nil {
		t.Fatalf("seed foreign scope: %v", err)
	}
	hostA := deployTestHost{Hostname: "host-a", IP: "10.41.0.1", OS: "linux", Status: "assigned", BusinessScopeID: scope.ID, BusinessScopeName: scope.Name, DeptID: 20}
	hostB := deployTestHost{Hostname: "host-b", IP: "10.41.0.2", OS: "linux", Status: "assigned", BusinessScopeID: scope.ID, BusinessScopeName: scope.Name, DeptID: 20}
	foreignHost := deployTestHost{Hostname: "host-c", IP: "10.41.1.1", OS: "linux", Status: "assigned", BusinessScopeID: foreignScope.ID, BusinessScopeName: foreignScope.Name, DeptID: 30}
	for _, host := range []*deployTestHost{&hostA, &hostB, &foreignHost} {
		if err := db.Create(host).Error; err != nil {
			t.Fatalf("seed host: %v", err)
		}
	}

	makeTask := func(name string, scopeID uint64, targetIDs []uint64) uint64 {
		t.Helper()
		task, err := svc.CreateTask(CreateTaskRequest{
			Name: name, PackageID: pkg.ID, BusinessScopeID: scopeID,
			TargetType: "host", TargetIDs: targetIDs, ExecutorType: "manual",
		}, "1", nil)
		if err != nil {
			t.Fatalf("create task %s: %v", name, err)
		}
		return task.ID
	}
	onlyA := makeTask("only-a", scope.ID, []uint64{hostA.ID})
	both := makeTask("a-and-b", scope.ID, []uint64{hostA.ID, hostB.ID})
	onlyB := makeTask("only-b", scope.ID, []uint64{hostB.ID})
	foreign := makeTask("foreign", foreignScope.ID, []uint64{foreignHost.ID})

	ids := func(resp *TaskListResponse) map[uint64]bool {
		out := map[uint64]bool{}
		for _, item := range resp.Items {
			out[item.ID] = true
		}
		return out
	}
	list := func(hostID uint64, ds *common.DataScopeReq) map[uint64]bool {
		t.Helper()
		resp, err := svc.ListTasks(TaskQuery{HostID: hostID, Page: 1, PageSize: 20}, ds)
		if err != nil {
			t.Fatalf("list host %d: %v", hostID, err)
		}
		return ids(resp)
	}

	gotA := list(hostA.ID, deptScope)
	if len(gotA) != 2 || !gotA[onlyA] || !gotA[both] {
		t.Fatalf("host A filter returned %v", gotA)
	}
	gotB := list(hostB.ID, deptScope)
	if len(gotB) != 2 || !gotB[onlyB] || !gotB[both] {
		t.Fatalf("host B filter returned %v", gotB)
	}
	if got := list(foreignHost.ID, deptScope); len(got) != 0 {
		t.Fatalf("out-of-scope host leaked tasks: %v", got)
	}
	if got := list(foreignHost.ID, nil); len(got) != 1 || !got[foreign] {
		t.Fatalf("admin host filter returned %v", got)
	}
}
