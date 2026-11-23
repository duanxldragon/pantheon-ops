package pkg

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"k8s.io/client-go/kubernetes"
)

// poolTestKubeconfig builds a parseable kubeconfig for a public API endpoint.
// The client is only constructed, never dialed, so no server exists.
func poolTestKubeconfig(token string) string {
	return fmt.Sprintf(`apiVersion: v1
kind: Config
clusters:
- name: test
  cluster:
    server: http://8.8.8.8:6443
contexts:
- name: test
  context:
    cluster: test
    user: test
current-context: test
users:
- name: test
  user:
    token: %s
`, token)
}

func TestPoolSameCredentialReturnsCachedClientset(t *testing.T) {
	pool := NewClientPool(time.Minute)
	kc := poolTestKubeconfig("shared-token")

	first, err := pool.GetOrCreate(1, kc)
	if err != nil {
		t.Fatalf("first GetOrCreate: %v", err)
	}
	second, err := pool.GetOrCreate(1, kc)
	if err != nil {
		t.Fatalf("second GetOrCreate: %v", err)
	}
	if first != second {
		t.Fatal("expected the same cached clientset for an identical credential")
	}
	if len(pool.clients) != 1 {
		t.Fatalf("expected one pooled entry, got %d", len(pool.clients))
	}
}

func TestPoolChangedCredentialNeverReusesWarmedClientset(t *testing.T) {
	pool := NewClientPool(time.Minute)
	oldKC := poolTestKubeconfig("privileged-token")
	warmed, err := pool.GetOrCreate(1, oldKC)
	if err != nil {
		t.Fatalf("warm GetOrCreate: %v", err)
	}

	newKC := poolTestKubeconfig("invalid-token")
	rotated, err := pool.GetOrCreate(1, newKC)
	if err != nil {
		t.Fatalf("GetOrCreate with changed credential: %v", err)
	}
	if rotated == warmed {
		t.Fatal("old warmed clientset reused for changed credential input")
	}

	oldKey := keyFor(1, oldKC)
	newKey := keyFor(1, newKC)
	if oldKey == newKey {
		t.Fatal("distinct credentials must not share a cache key")
	}
	if pool.clients[oldKey] == nil || pool.clients[newKey] == nil {
		t.Fatal("expected both credential entries to coexist in the pool")
	}

	// The supplied credential decides identity: restoring the original input
	// reuses its warm client rather than rebuilding it.
	restored, err := pool.GetOrCreate(1, oldKC)
	if err != nil {
		t.Fatalf("GetOrCreate with restored credential: %v", err)
	}
	if restored != warmed {
		t.Fatal("expected the original warm client for the restored credential")
	}
}

func TestPoolEvictClearsOnlyTargetCluster(t *testing.T) {
	pool := NewClientPool(time.Minute)
	clusterAClient, err := pool.GetOrCreate(1, poolTestKubeconfig("token-a"))
	if err != nil {
		t.Fatalf("cluster A: %v", err)
	}
	if _, err := pool.GetOrCreate(1, poolTestKubeconfig("token-b")); err != nil {
		t.Fatalf("cluster A second credential: %v", err)
	}
	clusterBClient, err := pool.GetOrCreate(2, poolTestKubeconfig("token-a"))
	if err != nil {
		t.Fatalf("cluster B: %v", err)
	}

	pool.Evict(1)
	if len(pool.clients) != 1 {
		t.Fatalf("expected only cluster B to remain after evicting cluster 1, got %d entries", len(pool.clients))
	}
	if recovered, err := pool.GetOrCreate(2, poolTestKubeconfig("token-a")); err != nil || recovered != clusterBClient {
		t.Fatalf("cluster B warm client should be untouched (err=%v)", err)
	}
	if rebuilt, err := pool.GetOrCreate(1, poolTestKubeconfig("token-a")); err != nil || rebuilt == clusterAClient {
		t.Fatalf("evicted cluster A should rebuild a fresh client (err=%v)", err)
	}
}

func TestPoolConcurrentSameCredentialDeduplicates(t *testing.T) {
	pool := NewClientPool(time.Minute)
	kc := poolTestKubeconfig("parallel-token")

	const workers = 16
	results := make([]*kubernetes.Clientset, workers)
	errs := make([]error, workers)
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			results[i], errs[i] = pool.GetOrCreate(7, kc)
		}(i)
	}
	wg.Wait()

	for i := 0; i < workers; i++ {
		if errs[i] != nil {
			t.Fatalf("worker %d: %v", i, errs[i])
		}
		if results[i] == nil || results[i] != results[0] {
			t.Fatalf("worker %d did not share the single pooled clientset", i)
		}
	}
	if len(pool.clients) != 1 {
		t.Fatalf("expected one pooled entry, got %d", len(pool.clients))
	}
}

func TestPoolConcurrentDistinctCredentialsStayIsolated(t *testing.T) {
	pool := NewClientPool(time.Minute)
	kubeconfigs := []string{
		poolTestKubeconfig("swap-token-1"),
		poolTestKubeconfig("swap-token-2"),
		poolTestKubeconfig("swap-token-3"),
	}
	warm := make([]*kubernetes.Clientset, len(kubeconfigs))
	for i, kc := range kubeconfigs {
		c, err := pool.GetOrCreate(9, kc)
		if err != nil {
			t.Fatalf("warm credential %d: %v", i, err)
		}
		warm[i] = c
	}

	var wg sync.WaitGroup
	for i := 0; i < 60; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			idx := i % len(kubeconfigs)
			client, err := pool.GetOrCreate(9, kubeconfigs[idx])
			if err != nil {
				t.Errorf("worker %d: %v", i, err)
				return
			}
			// A caller presenting credential idx must receive its own warm
			// client — never the one warmed for a different credential.
			if client != warm[idx] {
				t.Errorf("worker %d presented credential %d but got a foreign clientset", i, idx)
			}
		}(i)
	}
	wg.Wait()
}

func TestPoolGCEvictsIdleAndKeepsActive(t *testing.T) {
	pool := NewClientPool(30 * time.Second)
	kc := poolTestKubeconfig("gc-token")
	if _, err := pool.GetOrCreate(5, kc); err != nil {
		t.Fatalf("GetOrCreate: %v", err)
	}
	key := keyFor(5, kc)

	now := time.Now()
	// Force the next sweep to be due immediately.
	pool.lastGC.Store(now.Add(-2 * time.Minute).UnixNano())

	pool.mu.Lock()
	pool.clients[key].lastUsed.Store(now.Add(-time.Minute).UnixNano()) // idle beyond TTL
	pool.mu.Unlock()

	pool.gc(now)
	if _, ok := pool.clients[key]; ok {
		t.Fatal("idle entry was not evicted")
	}

	// A freshly used entry survives the sweep.
	if _, err := pool.GetOrCreate(5, kc); err != nil {
		t.Fatalf("rebuild after eviction: %v", err)
	}
	pool.lastGC.Store(now.Add(-2 * time.Minute).UnixNano())
	pool.gc(time.Now())
	if _, ok := pool.clients[key]; !ok {
		t.Fatal("active entry was evicted while in use")
	}
}