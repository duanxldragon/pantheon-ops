package business

import (
	"sync"

	"github.com/duanxldragon/pantheon-base/backend/pkg/contracts"
	"gorm.io/gorm"
)

type businessInitializationPreflightRegistry struct {
	mu        sync.Mutex
	preflight func(*gorm.DB) error
	sealed    bool
	active    bool
}

var businessInitializationPreflight = &businessInitializationPreflightRegistry{}

// RegisterBusinessInitializationPreflight registers the consumer's single
// read-only composition check before the first business initialization.
func RegisterBusinessInitializationPreflight(preflight func(*gorm.DB) error) error {
	return businessInitializationPreflight.register(preflight)
}

func (r *businessInitializationPreflightRegistry) register(preflight func(*gorm.DB) error) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if preflight == nil || r.preflight != nil || r.sealed || r.active {
		return contracts.ErrBusinessProviders
	}
	r.preflight = preflight
	return nil
}

func (r *businessInitializationPreflightRegistry) run(db *gorm.DB) error {
	r.mu.Lock()
	r.sealed = true
	if r.active {
		r.mu.Unlock()
		return contracts.ErrBusinessProviders
	}
	preflight := r.preflight
	if preflight == nil {
		r.mu.Unlock()
		return nil
	}
	r.active = true
	r.mu.Unlock()
	defer func() {
		r.mu.Lock()
		r.active = false
		r.mu.Unlock()
	}()
	return preflight(db)
}
