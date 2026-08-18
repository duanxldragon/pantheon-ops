package cluster

import (
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

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
