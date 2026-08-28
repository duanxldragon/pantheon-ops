package pkg

import (
	"sync"
	"time"

	"k8s.io/client-go/kubernetes"
)

// ClientPool caches kubernetes Clientsets keyed by cluster ID so concurrent
// request handlers reuse a single authenticated client per cluster. Entries
// are evicted after the idle timeout.
type ClientPool struct {
	mu       sync.RWMutex
	clients  map[uint64]*pooledClient
	idleTTL  time.Duration
	lastGC   time.Time
}

type pooledClient struct {
	clientset *kubernetes.Clientset
	createdAt time.Time
	lastUsed  time.Time
}

// NewClientPool creates a client pool with the given idle timeout.
func NewClientPool(idleTTL time.Duration) *ClientPool {
	if idleTTL <= 0 {
		idleTTL = 30 * time.Second
	}
	return &ClientPool{
		clients: make(map[uint64]*pooledClient),
		idleTTL: idleTTL,
		lastGC:  time.Now(),
	}
}

// GetOrCreate returns the cached clientset for a cluster, building a new one
// from the supplied kubeconfig when no live entry exists.
func (p *ClientPool) GetOrCreate(clusterID uint64, kubeconfig string) (*kubernetes.Clientset, error) {
	now := time.Now()
	p.mu.RLock()
	entry, ok := p.clients[clusterID]
	p.mu.RUnlock()
	if ok && entry.clientset != nil {
		entry.lastUsed = now
		return entry.clientset, nil
	}

	clientset, err := NewClientFromKubeconfig(kubeconfig)
	if err != nil {
		return nil, err
	}

	p.mu.Lock()
	// Re-check after acquiring the lock to avoid duplicate builds.
	if entry, ok := p.clients[clusterID]; ok && entry.clientset != nil {
		p.mu.Unlock()
		return entry.clientset, nil
	}
	p.clients[clusterID] = &pooledClient{clientset: clientset, createdAt: now, lastUsed: now}
	p.mu.Unlock()

	p.gc(now)
	return clientset, nil
}

// gc evicts entries idle longer than the TTL. It runs at most once per minute
// to avoid lock contention on hot paths.
func (p *ClientPool) gc(now time.Time) {
	if now.Sub(p.lastGC) < time.Minute {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	for id, entry := range p.clients {
		if now.Sub(entry.lastUsed) > p.idleTTL {
			delete(p.clients, id)
		}
	}
	p.lastGC = now
}

// Evict removes a cluster entry (used after credential rotation).
func (p *ClientPool) Evict(clusterID uint64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.clients, clusterID)
}
