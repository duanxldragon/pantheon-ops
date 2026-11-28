package service

import (
	"context"
	"testing"

	bizcap "github.com/duanxldragon/pantheon-base/backend/modules/business/capability"
)

// T20 S1: the instance list page filters by environment in SQL. Total must be
// the exact filtered count so server-side paging stays correct past 100 rows.
func TestListInstancesFiltersByEnvironmentWithExactTotal(t *testing.T) {
	m, prod, scope := serviceInstanceUpdateFixture(t)
	m.cmdbReader = fakeHostReader{ref: bizcap.HostRef{ID: 56, BusinessScopeID: 10}}
	staging, err := m.CreateInstance(context.Background(), CreateInstanceRequest{
		ServiceID: prod.ServiceID, Environment: "staging", TargetType: TargetTypeVM, HostID: 56,
	}, "1", scope)
	if err != nil {
		t.Fatalf("create staging instance: %v", err)
	}

	list := func(q InstanceQuery) *InstanceListResponse {
		t.Helper()
		q.Page, q.PageSize = 1, 20
		resp, err := m.ListInstances(q, scope)
		if err != nil {
			t.Fatalf("list %+v: %v", q, err)
		}
		return resp
	}

	if resp := list(InstanceQuery{}); resp.Total != 2 {
		t.Fatalf("no filter should return both instances, got %d", resp.Total)
	}
	if resp := list(InstanceQuery{Environment: "staging"}); resp.Total != 1 || resp.Items[0].ID != staging.ID {
		t.Fatalf("staging filter returned %+v", resp)
	}
	if resp := list(InstanceQuery{Environment: "  prod  "}); resp.Total != 1 || resp.Items[0].ID != prod.ID {
		t.Fatalf("environment filter must trim whitespace, got %+v", resp)
	}
	if resp := list(InstanceQuery{Environment: "prod", TargetType: TargetTypeVM, ServiceID: prod.ServiceID}); resp.Total != 1 {
		t.Fatalf("combined filters should AND together, got %d", resp.Total)
	}
	if resp := list(InstanceQuery{Environment: "missing"}); resp.Total != 0 || len(resp.Items) != 0 {
		t.Fatalf("unknown environment must be empty, got %+v", resp)
	}
}

// T21 S1: the CMDB host detail lists the instances that target that host, and
// the business-scope summary counts only rows inside the caller's data scope.
func TestListInstancesFiltersByHostAndScopeSummary(t *testing.T) {
	m, prod, scope := serviceInstanceUpdateFixture(t)
	m.cmdbReader = fakeHostReader{ref: bizcap.HostRef{ID: 56, BusinessScopeID: 10}}
	other, err := m.CreateInstance(context.Background(), CreateInstanceRequest{
		ServiceID: prod.ServiceID, Environment: "prod", TargetType: TargetTypeVM, HostID: 56,
	}, "1", scope)
	if err != nil {
		t.Fatalf("create second instance: %v", err)
	}

	byHost := func(hostID uint64) []uint64 {
		t.Helper()
		resp, err := m.ListInstances(InstanceQuery{HostID: hostID, Page: 1, PageSize: 20}, scope)
		if err != nil {
			t.Fatalf("list host %d: %v", hostID, err)
		}
		ids := make([]uint64, 0, len(resp.Items))
		for _, item := range resp.Items {
			ids = append(ids, item.ID)
		}
		return ids
	}
	if ids := byHost(55); len(ids) != 1 || ids[0] != prod.ID {
		t.Fatalf("host 55 must return only the fixture instance, got %v", ids)
	}
	if ids := byHost(56); len(ids) != 1 || ids[0] != other.ID {
		t.Fatalf("host 56 must return only the second instance, got %v", ids)
	}
	if ids := byHost(999); len(ids) != 0 {
		t.Fatalf("unknown host must be empty, got %v", ids)
	}

	summary, err := m.ScopeSummary(context.Background(), 10, scope)
	if err != nil {
		t.Fatalf("scope summary: %v", err)
	}
	if summary.Applications != 1 || summary.Services != 1 || summary.Instances != 2 {
		t.Fatalf("scope 10 summary mismatch: %+v", summary)
	}
	empty, err := m.ScopeSummary(context.Background(), 11, scope)
	if err != nil {
		t.Fatalf("empty scope summary: %v", err)
	}
	if empty.Applications != 0 || empty.Services != 0 || empty.Instances != 0 {
		t.Fatalf("scope 11 must be empty, got %+v", empty)
	}
}
