package cluster

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	bizcap "pantheon-base/modules/business/capability"
	"pantheon-base/pkg/common"
	"pantheon-base/pkg/testmysql"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

type blockingBizScopeReader struct {
	refs      map[uint64]bizcap.BizScopeRef
	blockID   uint64
	entered   chan struct{}
	release   chan struct{}
	enterOnce sync.Once
}

func (r *blockingBizScopeReader) GetActive(_ context.Context, id uint64, scope *common.DataScopeReq) (bizcap.BizScopeRef, error) {
	ref, ok := r.refs[id]
	if !ok {
		return bizcap.BizScopeRef{}, errors.New("business.bizscope.notFound")
	}
	if id == r.blockID {
		r.enterOnce.Do(func() { close(r.entered) })
		<-r.release
	}
	if scope != nil && !scope.IsAdmin && scope.Mode == common.DataScopeModeDept && scope.DeptID != ref.DeptID {
		return bizcap.BizScopeRef{}, errors.New("business.bizscope.notFound")
	}
	return ref, nil
}

func (r *blockingBizScopeReader) ResolveActiveByCodes(context.Context, []string, *common.DataScopeReq) (map[string]bizcap.BizScopeRef, error) {
	return nil, nil
}

func TestQuantityToCores(t *testing.T) {
	if got := quantityToCores(nil); got != 0 {
		t.Fatalf("nil quantity: got %v want 0", got)
	}
	q := resource.MustParse("2500m")
	if got := quantityToCores(&q); got != 2.5 {
		t.Fatalf("2500m: got %v want 2.5", got)
	}
}

func TestQuantityToGB(t *testing.T) {
	if got := quantityToGB(nil); got != 0 {
		t.Fatalf("nil quantity: got %v want 0", got)
	}
	q := resource.MustParse("4Gi")
	if got := quantityToGB(&q); got != 4 {
		t.Fatalf("4Gi: got %v want 4", got)
	}
}

func TestIsNodeReady(t *testing.T) {
	ready := &corev1.Node{
		Status: corev1.NodeStatus{
			Conditions: []corev1.NodeCondition{
				{Type: corev1.NodeReady, Status: corev1.ConditionTrue},
			},
		},
	}
	if !isNodeReady(ready) {
		t.Fatal("expected node with ready condition to be ready")
	}

	notReady := &corev1.Node{
		Status: corev1.NodeStatus{
			Conditions: []corev1.NodeCondition{
				{Type: corev1.NodeReady, Status: corev1.ConditionFalse},
			},
		},
	}
	if isNodeReady(notReady) {
		t.Fatal("expected node with not-ready condition to be not ready")
	}
}

func TestToResponse(t *testing.T) {
	now := time.Now()
	c := &Cluster{
		ID:                1,
		Code:              "prod-1",
		Name:              "Production",
		Environment:       "prod",
		BusinessScopeID:   10,
		BusinessScopeName: "Order System",
		DeptID:            5,
		APIServer:         "https://10.0.0.1:6443",
		Version:           "v1.28.0",
		Status:            "healthy",
		TotalNodes:        3,
		ReadyNodes:        3,
		TotalPods:         42,
		RunningPods:       40,
		CPUCapacity:       12,
		MemoryCapacity:    64,
		LastSyncedAt:      &now,
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	resp := toResponse(c)
	if resp.Code != "prod-1" || resp.BusinessScopeID != 10 || resp.DeptID != 5 {
		t.Fatalf("unexpected response fields: %+v", resp)
	}
	if resp.LastSyncedAt == "" {
		t.Fatal("expected lastSyncedAt to be formatted")
	}

	c.LastSyncedAt = nil
	resp = toResponse(c)
	if resp.LastSyncedAt != "" {
		t.Fatalf("expected empty lastSyncedAt for nil, got %q", resp.LastSyncedAt)
	}
}

func TestToNodeSnapshot(t *testing.T) {
	node := &corev1.Node{
		Status: corev1.NodeStatus{
			Addresses: []corev1.NodeAddress{
				{Type: corev1.NodeHostName, Address: "node-1"},
				{Type: corev1.NodeInternalIP, Address: "10.0.0.10"},
			},
			NodeInfo: corev1.NodeSystemInfo{
				OSImage:        "Ubuntu 22.04",
				KubeletVersion: "v1.28.0",
			},
			Capacity: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("4"),
				corev1.ResourceMemory: resource.MustParse("8Gi"),
				corev1.ResourcePods:   resource.MustParse("110"),
			},
			Allocatable: corev1.ResourceList{
				corev1.ResourcePods: resource.MustParse("100"),
			},
			Conditions: []corev1.NodeCondition{
				{Type: corev1.NodeReady, Status: corev1.ConditionTrue},
			},
		},
	}

	snap := toNodeSnapshot(node)
	if snap.InternalIP != "10.0.0.10" || snap.Status != "ready" || snap.PodCapacity != 110 || snap.AllocatablePods != 100 {
		t.Fatalf("unexpected node snapshot: %+v", snap)
	}
}

func TestClusterUpdateRejectsConcurrentStaleScopeMutation(t *testing.T) {
	db := testmysql.Open(t)
	if err := db.AutoMigrate(&Cluster{}); err != nil {
		t.Fatalf("migrate cluster: %v", err)
	}

	cluster := Cluster{Code: "concurrent-owner", Name: "Owned by A", Environment: "test", BusinessScopeID: 1, BusinessScopeName: "Scope A", DeptID: 10, Status: "unknown"}
	if err := db.Create(&cluster).Error; err != nil {
		t.Fatalf("create cluster: %v", err)
	}

	reader := &blockingBizScopeReader{
		refs: map[uint64]bizcap.BizScopeRef{
			2: {ID: 2, Code: "scope-b", Name: "Scope B", Status: "active", DeptID: 20},
		},
		blockID: 2,
		entered: make(chan struct{}),
		release: make(chan struct{}),
	}
	service := NewClusterService(db, reader)
	moveScope := &common.DataScopeReq{Mode: common.DataScopeModeCustom, DeptIDs: []uint64{10, 20}}
	staleScope := &common.DataScopeReq{Mode: common.DataScopeModeDept, DeptID: 10}
	targetScopeID := uint64(2)
	staleName := "stale overwrite"

	moveResult := make(chan error, 1)
	go func() {
		_, err := service.Update(cluster.ID, UpdateClusterRequest{BusinessScopeID: &targetScopeID}, "owner-move", moveScope)
		moveResult <- err
	}()
	<-reader.entered

	staleResult := make(chan error, 1)
	go func() {
		_, err := service.Update(cluster.ID, UpdateClusterRequest{Name: &staleName}, "stale-writer", staleScope)
		staleResult <- err
	}()
	close(reader.release)

	if err := <-moveResult; err != nil {
		t.Fatalf("move cluster ownership: %v", err)
	}
	if err := <-staleResult; err == nil || err.Error() != "k8s.cluster.not_found" {
		t.Fatalf("expected stale scoped update to be rejected, got %v", err)
	}

	var reloaded Cluster
	if err := db.First(&reloaded, cluster.ID).Error; err != nil {
		t.Fatalf("reload cluster: %v", err)
	}
	if reloaded.BusinessScopeID != 2 || reloaded.BusinessScopeName != "Scope B" || reloaded.DeptID != 20 || reloaded.Name != "Owned by A" {
		t.Fatalf("unexpected concurrent update result: %+v", reloaded)
	}
}
