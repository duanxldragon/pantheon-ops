package pkg

import (
	"crypto/sha256"
	"sync"
	"sync/atomic"
	"time"

	"k8s.io/client-go/kubernetes"
)

// poolKey identifies a cached Clientset by the cluster that owns it and a
// digest of the kubeconfig it was built from. Keying on the credential digest
// means a rotated credential can never be matched to a client built from the
// previous one: lookups simply miss and build a fresh client from the new
// input, so an old warmed client is never reused after a config change.
type poolKey struct {
	clusterID uint64
	digest    [32]byte
}

// ClientPool caches kubernetes Clientsets keyed by cluster ID AND credential
// digest so concurrent request handlers reuse a single authenticated client per
// credential. Entries are evicted after the idle timeout.
type ClientPool struct {
	mu      sync.RWMutex
	clients map[poolKey]*pooledClient
	idleTTL time.Duration
	lastGC  atomic.Int64
}

type pooledClient struct {
	clientset *kubernetes.Clientset
	lastUsed  atomic.Int64
}

// NewClientPool creates a client pool with the given idle timeout.
func NewClientPool(idleTTL time.Duration) *ClientPool {
	if idleTTL <= 0 {
		idleTTL = 30 * time.Second
	}
	pool := &ClientPool{
		clients: make(map[poolKey]*pooledClient),
		idleTTL: idleTTL,
	}
	pool.lastGC.Store(time.Now().UnixNano())
	return pool
}

// keyFor derives the cache identity for a cluster and its supplied kubeconfig.
func keyFor(clusterID uint64, kubeconfig string) poolKey {
	return poolKey{clusterID: clusterID, digest: sha256.Sum256([]byte(kubeconfig))}
}

// GetOrCreate returns the cached clientset for a cluster and its exact
// supplied credential, building a new one when no live entry exists. A changed
// kubeconfig for the same cluster hashes to a different key, so callers that
// present rotated credentials never receive a client built from the previous
// ones.
func (p *ClientPool) GetOrCreate(clusterID uint64, kubeconfig string) (*kubernetes.Clientset, error) {
	key := keyFor(clusterID, kubeconfig)
	now := time.Now()

	p.mu.RLock()
	entry, ok := p.clients[key]
	p.mu.RUnlock()
	if ok && entry.clientset != nil {
		entry.lastUsed.Store(now.UnixNano())
		return entry.clientset, nil
	}

	clientset, err := NewClientFromKubeconfig(kubeconfig)
	if err != nil {
		return nil, err
	}

	p.mu.Lock()
	// Re-check after acquiring the lock to avoid duplicate builds.
	if entry, ok := p.clients[key]; ok && entry.clientset != nil {
		p.mu.Unlock()
		entry.lastUsed.Store(now.UnixNano())
		return entry.clientset, nil
	}
	entry = &pooledClient{clientset: clientset}
	entry.lastUsed.Store(now.UnixNano())
	p.clients[key] = entry
	p.mu.Unlock()

	p.gc(now)
	return clientset, nil
}

// gcInterval is the minimum interval between garbage-collection sweeps.
const gcInterval = time.Minute

// gc evicts entries idle longer than the TTL. It runs at most once per minute
// to avoid lock contention on hot paths; the atomic lastGC lets the cheap
// read-side check skip the sweep lock entirely when no sweep is due.
func (p *ClientPool) gc(now time.Time) {
	if now.Sub(time.Unix(0, p.lastGC.Load())) < gcInterval {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if now.Sub(time.Unix(0, p.lastGC.Load())) < gcInterval {
		return
	}
	for key, entry := range p.clients {
		if now.Sub(time.Unix(0, entry.lastUsed.Load())) > p.idleTTL {
			delete(p.clients, key)
		}
	}
	p.lastGC.Store(now.UnixNano())
}

// Evict removes every cached client for a cluster. It is called after a
// successful credential change to drop stale-warmed clients promptly;
// digest-keyed lookups already guarantee rotated input can never hit an entry
// built from the previous credential.
func (p *ClientPool) Evict(clusterID uint64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	for key := range p.clients {
		if key.clusterID == clusterID {
			delete(p.clients, key)
		}
	}
}
