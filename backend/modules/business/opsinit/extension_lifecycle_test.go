package opsinit

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

type extensionDistributorFixture struct {
	startError error
	startCtx   context.Context
	starts     atomic.Int32
	stops      atomic.Int32
}

func (f *extensionDistributorFixture) Start(ctx context.Context) error {
	f.startCtx = ctx
	f.starts.Add(1)
	return f.startError
}

func (f *extensionDistributorFixture) Stop() { f.stops.Add(1) }

type extensionObserverFixture struct {
	started chan struct{}
	stopped chan struct{}
	release chan struct{}
	starts  atomic.Int32
	stops   atomic.Int32
}

func (f *extensionObserverFixture) Start(ctx context.Context) {
	f.starts.Add(1)
	close(f.started)
	<-ctx.Done()
	<-f.release
}

func (f *extensionObserverFixture) Stop() {
	f.stops.Add(1)
	close(f.stopped)
}

func TestExtensionLifecycleStartupFailureStopsOwnedDistributor(t *testing.T) {
	wantError := errors.New("lease unavailable")
	distributor := &extensionDistributorFixture{startError: wantError}
	observer := &extensionObserverFixture{}
	stop, err := startExtensionRuntime(context.Background(), distributor, observer)
	if !errors.Is(err, wantError) || stop != nil || distributor.stops.Load() != 1 || observer.starts.Load() != 0 || observer.stops.Load() != 0 {
		t.Fatalf("partial startup leaked resources or launched observer: err=%v distributorStops=%d observerStarts=%d", err, distributor.stops.Load(), observer.starts.Load())
	}
	if distributor.startCtx.Err() == nil {
		t.Fatal("failed startup context was not cancelled")
	}
}

func TestExtensionLifecycleCleanupCancelsAndJoinsObserver(t *testing.T) {
	parent, cancelParent := context.WithCancel(context.Background())
	defer cancelParent()
	distributor := &extensionDistributorFixture{}
	observer := &extensionObserverFixture{started: make(chan struct{}), stopped: make(chan struct{}), release: make(chan struct{})}
	stop, err := startExtensionRuntime(parent, distributor, observer)
	if err != nil {
		t.Fatal(err)
	}
	select {
	case <-observer.started:
	case <-time.After(2 * time.Second):
		t.Fatal("observer did not start")
	}
	cancelParent()
	cleanupDone := make(chan struct{})
	go func() { stop(); close(cleanupDone) }()
	select {
	case <-observer.stopped:
	case <-time.After(2 * time.Second):
		t.Fatal("cleanup did not stop observer")
	}
	select {
	case <-cleanupDone:
		t.Fatal("cleanup returned before observer joined")
	default:
	}
	if distributor.stops.Load() != 0 {
		t.Fatal("distributor stopped before observer joined")
	}
	close(observer.release)
	select {
	case <-cleanupDone:
	case <-time.After(2 * time.Second):
		t.Fatal("cleanup did not join observer")
	}
	stop()
	if distributor.stops.Load() != 1 || observer.stops.Load() != 1 || distributor.startCtx.Err() == nil {
		t.Fatal("cleanup must cancel and stop owned components exactly once")
	}
}

func TestExtensionLifecycleRejectsInvalidParentBeforeStarting(t *testing.T) {
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	for _, parent := range []context.Context{nil, cancelled} {
		distributor := &extensionDistributorFixture{}
		observer := &extensionObserverFixture{}
		if _, err := startExtensionRuntime(parent, distributor, observer); err == nil || distributor.starts.Load() != 0 || observer.starts.Load() != 0 {
			t.Fatal("invalid parent started an owned component")
		}
	}
}
