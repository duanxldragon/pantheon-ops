package opsinit

import (
	"context"
	"errors"
	"sync"
)

type extensionDistributor interface {
	Start(context.Context) error
	Stop()
}

type extensionObserver interface {
	Start(context.Context)
	Stop()
}

// startExtensionRuntime starts only after migration and provider wiring. Its
// cleanup joins the observer and stops the distributor, including partial
// startup failure. This is owned-component cleanup, not a Base shutdown hook.
func startExtensionRuntime(parent context.Context, distributor extensionDistributor, observer extensionObserver) (func(), error) {
	if parent == nil {
		return nil, errors.New("business.filedist.task.invalidContext")
	}
	if err := parent.Err(); err != nil {
		return nil, err
	}
	ctx, cancel := context.WithCancel(parent)
	if err := distributor.Start(ctx); err != nil {
		cancel()
		distributor.Stop()
		return nil, err
	}
	observerDone := make(chan struct{})
	go func() {
		defer close(observerDone)
		observer.Start(ctx)
	}()
	var once sync.Once
	return func() {
		once.Do(func() {
			cancel()
			observer.Stop()
			<-observerDone
			distributor.Stop()
		})
	}, nil
}
