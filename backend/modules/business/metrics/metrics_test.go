package metrics

import (
	"sync"
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
)

// TestSetClusterCapacityKeepsClustersIndependent locks the multi-cluster
// regression: the capacity gauge used to ignore the cluster identifier, so every
// cluster overwrote the same series and the last synced cluster won.
func TestSetClusterCapacityKeepsClustersIndependent(t *testing.T) {
	SetClusterCapacity("metrics-test-cluster-a", 3, 2, 12, 48)
	SetClusterCapacity("metrics-test-cluster-b", 7, 7, 28, 112)

	cases := []struct {
		cluster string
		metric  string
		want    float64
	}{
		{"metrics-test-cluster-a", "total_nodes", 3},
		{"metrics-test-cluster-a", "ready_nodes", 2},
		{"metrics-test-cluster-a", "cpu_capacity", 12},
		{"metrics-test-cluster-a", "memory_capacity_gb", 48},
		{"metrics-test-cluster-b", "total_nodes", 7},
		{"metrics-test-cluster-b", "ready_nodes", 7},
		{"metrics-test-cluster-b", "cpu_capacity", 28},
		{"metrics-test-cluster-b", "memory_capacity_gb", 112},
	}
	for _, tc := range cases {
		got := testutil.ToFloat64(ClusterCapacity.WithLabelValues(tc.cluster, tc.metric))
		if got != tc.want {
			t.Fatalf("cluster %q metric %q = %v, want %v", tc.cluster, tc.metric, got, tc.want)
		}
	}
}

// TestSetClusterCapacityNormalizesEmptyClusterCode keeps the empty-code case
// consistent with the other cluster-labelled metrics.
func TestSetClusterCapacityNormalizesEmptyClusterCode(t *testing.T) {
	SetClusterCapacity("", 1, 1, 1, 1)

	if got := testutil.ToFloat64(ClusterCapacity.WithLabelValues("unknown", "total_nodes")); got != 1 {
		t.Fatalf("empty cluster code should fall back to the unknown label, got %v", got)
	}
}

// TestSetHostStatusMetricsResetsStaleStates locks the snapshot semantics: a
// state that disappears from the current host set must drop back to 0 instead of
// keeping its last non-zero value forever.
func TestSetHostStatusMetricsResetsStaleStates(t *testing.T) {
	SetHostStatusMetrics([]HostStatusMetric{
		{LifecycleState: "metrics-test-running", ConnectivityState: "metrics-test-online"},
		{LifecycleState: "metrics-test-offline", ConnectivityState: "metrics-test-unreachable"},
	})

	if got := testutil.ToFloat64(HostsTotal.WithLabelValues("metrics-test-offline")); got != 1 {
		t.Fatalf("initial offline count = %v, want 1", got)
	}

	SetHostStatusMetrics([]HostStatusMetric{
		{LifecycleState: "metrics-test-running", ConnectivityState: "metrics-test-online"},
		{LifecycleState: "metrics-test-running", ConnectivityState: "metrics-test-online"},
	})

	if got := testutil.ToFloat64(HostsTotal.WithLabelValues("metrics-test-offline")); got != 0 {
		t.Fatalf("stale lifecycle label = %v, want 0", got)
	}
	if got := testutil.ToFloat64(HostConnectivity.WithLabelValues("metrics-test-unreachable")); got != 0 {
		t.Fatalf("stale connectivity label = %v, want 0", got)
	}
	if got := testutil.ToFloat64(HostsTotal.WithLabelValues("metrics-test-running")); got != 2 {
		t.Fatalf("current lifecycle count = %v, want 2", got)
	}
}

// TestSetHostStatusMetricsConcurrentRefresh exercises the snapshot bookkeeping
// under concurrent refreshes so `go test -race` covers the tracked label maps.
func TestSetHostStatusMetricsConcurrentRefresh(t *testing.T) {
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			states := []HostStatusMetric{
				{LifecycleState: "metrics-test-concurrent-running", ConnectivityState: "metrics-test-concurrent-online"},
				{LifecycleState: "metrics-test-concurrent-idle", ConnectivityState: "metrics-test-concurrent-online"},
			}
			if i%2 == 0 {
				states = states[:1]
			}
			SetHostStatusMetrics(states)
		}(i)
	}
	wg.Wait()

	// Every snapshot contains the running host, so it must always be 1; the idle
	// label depends on which snapshot landed last and must be a clean 0 or 1.
	running := testutil.ToFloat64(HostsTotal.WithLabelValues("metrics-test-concurrent-running"))
	idle := testutil.ToFloat64(HostsTotal.WithLabelValues("metrics-test-concurrent-idle"))
	if running != 1 {
		t.Fatalf("running lifecycle count = %v, want 1", running)
	}
	if idle != 0 && idle != 1 {
		t.Fatalf("idle lifecycle count = %v, want 0 or 1", idle)
	}
}
