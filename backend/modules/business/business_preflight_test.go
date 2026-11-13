package business

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/duanxldragon/pantheon-base/backend/pkg/contracts"
	"gorm.io/gorm"
)

func scopedBusinessPreflight(t *testing.T) {
	t.Helper()
	previous := businessInitializationPreflight
	businessInitializationPreflight = &businessInitializationPreflightRegistry{}
	t.Cleanup(func() { businessInitializationPreflight = previous })
}

func capturedBusinessPreflightPanic(run func()) (value any) {
	defer func() { value = recover() }()
	run()
	return nil
}

func waitBusinessPreflight(t *testing.T, result <-chan error) error {
	t.Helper()
	select {
	case err := <-result:
		return err
	case <-time.After(2 * time.Second):
		t.Fatal("preflight blocked instead of refusing recursion/overlap")
		return nil
	}
}

func TestBusinessPreflightRegistration(t *testing.T) {
	scopedBusinessPreflight(t)
	var missing func(*gorm.DB) error
	for _, preflight := range []func(*gorm.DB) error{nil, missing} {
		if RegisterBusinessInitializationPreflight(preflight) != contracts.ErrBusinessProviders {
			t.Fatal("nil callback must return the fixed provider error")
		}
	}
	callback := func(*gorm.DB) error { return nil }
	if err := RegisterBusinessInitializationPreflight(callback); err != nil {
		t.Fatal(err)
	}
	if RegisterBusinessInitializationPreflight(callback) != contracts.ErrBusinessProviders {
		t.Fatal("duplicate callback accepted")
	}
	if err := businessInitializationPreflight.run(nil); err != nil {
		t.Fatal(err)
	}
	if RegisterBusinessInitializationPreflight(callback) != contracts.ErrBusinessProviders {
		t.Fatal("late callback accepted")
	}
}

func TestBusinessPreflightEmptyRunSealsRegistration(t *testing.T) {
	registry := &businessInitializationPreflightRegistry{}
	for range 2 {
		if err := registry.run(nil); err != nil {
			t.Fatal("standalone empty preflight must retain success")
		}
	}
	if registry.register(func(*gorm.DB) error { return nil }) != contracts.ErrBusinessProviders {
		t.Fatal("registration after empty initialization accepted")
	}
}

func TestBusinessPreflightRepeatsReadinessAndPreservesError(t *testing.T) {
	registry := &businessInitializationPreflightRegistry{}
	cause := errors.New("consumer readiness unavailable")
	input := &gorm.DB{}
	calls := 0
	if err := registry.register(func(db *gorm.DB) error {
		if db != input {
			t.Fatal("composition callback received a different DB handle")
		}
		calls++
		if calls == 2 {
			return cause
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	for _, want := range []error{nil, cause, nil} {
		if registry.run(input) != want {
			t.Fatal("readiness result was cached/wrapped or active guard was retained after failure")
		}
	}
	if calls != 3 {
		t.Fatal("readiness was not executed on every invocation")
	}
	if registry.register(func(*gorm.DB) error { return nil }) != contracts.ErrBusinessProviders {
		t.Fatal("failed readiness unsealed registration")
	}
}

func TestBusinessPreflightPanicReleasesActiveGuard(t *testing.T) {
	registry := &businessInitializationPreflightRegistry{}
	marker := &struct{ name string }{"consumer panic"}
	calls := 0
	if err := registry.register(func(*gorm.DB) error {
		calls++
		if calls == 1 {
			panic(marker)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if capturedBusinessPreflightPanic(func() { _ = registry.run(nil) }) != marker {
		t.Fatal("callback panic was changed or swallowed")
	}
	if err := registry.run(nil); err != nil || calls != 2 {
		t.Fatal("panic retained the active guard")
	}
	if registry.register(func(*gorm.DB) error { return nil }) != contracts.ErrBusinessProviders {
		t.Fatal("panic unsealed registration")
	}
}

func TestBusinessPreflightRunsOutsideMutexAndRejectsReentry(t *testing.T) {
	registry := &businessInitializationPreflightRegistry{}
	if err := registry.register(func(*gorm.DB) error {
		if registry.register(func(*gorm.DB) error { return nil }) != contracts.ErrBusinessProviders {
			return errors.New("active registration accepted")
		}
		if registry.run(nil) != contracts.ErrBusinessProviders {
			return errors.New("recursive preflight accepted")
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	result := make(chan error, 1)
	go func() { result <- registry.run(nil) }()
	if err := waitBusinessPreflight(t, result); err != nil {
		t.Fatal(err)
	}
}

func TestBusinessPreflightRejectsOverlap(t *testing.T) {
	registry := &businessInitializationPreflightRegistry{}
	started := make(chan struct{})
	release := make(chan struct{})
	var releaseOnce sync.Once
	unblock := func() { releaseOnce.Do(func() { close(release) }) }
	defer unblock()
	var calls atomic.Int32
	if err := registry.register(func(*gorm.DB) error {
		if calls.Add(1) == 1 {
			close(started)
		}
		<-release
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	first, second := make(chan error, 1), make(chan error, 1)
	go func() { first <- registry.run(nil) }()
	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("first callback did not start")
	}
	go func() { second <- registry.run(nil) }()
	if waitBusinessPreflight(t, second) != contracts.ErrBusinessProviders {
		t.Fatal("overlapping preflight did not return the fixed provider error")
	}
	if calls.Load() != 1 {
		t.Fatal("overlap executed the composition callback")
	}
	// Release before waiting while retaining cleanup for any fatal path.
	unblock()
	if err := waitBusinessPreflight(t, first); err != nil {
		t.Fatal(err)
	}
}

func TestBusinessPreflightConcurrentRegistration(t *testing.T) {
	registry := &businessInitializationPreflightRegistry{}
	start := make(chan struct{})
	results := make(chan error, 16)
	var workers sync.WaitGroup
	for range 16 {
		workers.Go(func() {
			<-start
			results <- registry.register(func(*gorm.DB) error { return nil })
		})
	}
	close(start)
	workers.Wait()
	close(results)
	accepted := 0
	for err := range results {
		if err == nil {
			accepted++
		} else if err != contracts.ErrBusinessProviders {
			t.Fatal("registration returned an unexpected error")
		}
	}
	if accepted != 1 {
		t.Fatal("registration did not choose exactly one composition callback")
	}
}

func TestBusinessPreflightRegistrationSealRace(t *testing.T) {
	for range 100 {
		registry := &businessInitializationPreflightRegistry{}
		start := make(chan struct{})
		registered, initialized := make(chan error, 1), make(chan error, 1)
		var calls atomic.Int32
		go func() {
			<-start
			registered <- registry.register(func(*gorm.DB) error { calls.Add(1); return nil })
		}()
		go func() { <-start; initialized <- registry.run(nil) }()
		close(start)
		registration := waitBusinessPreflight(t, registered)
		if err := waitBusinessPreflight(t, initialized); err != nil {
			t.Fatal(err)
		}
		want := int32(0)
		if registration == nil {
			want = 1
		} else if registration != contracts.ErrBusinessProviders {
			t.Fatal("registration/seal race returned an unexpected error")
		}
		if calls.Load() != want {
			t.Fatal("atomic snapshot did not execute the accepted callback")
		}
		if err := registry.run(nil); err != nil || calls.Load() != 2*want {
			t.Fatal("sealed callback was changed or readiness success was cached")
		}
		if registry.register(func(*gorm.DB) error { return nil }) != contracts.ErrBusinessProviders {
			t.Fatal("late registration accepted after the seal race")
		}
	}
}
