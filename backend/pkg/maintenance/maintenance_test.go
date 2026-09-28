package maintenance

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestRegistryRunDueThrottlesByInterval(t *testing.T) {
	reg := NewRegistry()
	var runs int32
	reg.Register(Task{
		Name:     "throttled",
		Interval: time.Hour,
		Run: func(context.Context) error {
			atomic.AddInt32(&runs, 1)
			return nil
		},
	})

	results := reg.RunDue(context.Background())
	if len(results) != 1 || results[0].Outcome != OutcomeSucceeded {
		t.Fatalf("first pass should run the task, got %+v", results)
	}

	results = reg.RunDue(context.Background())
	if len(results) != 1 || results[0].Outcome != OutcomeSkipped {
		t.Fatalf("second pass within the interval should be skipped, got %+v", results)
	}
	if got := atomic.LoadInt32(&runs); got != 1 {
		t.Fatalf("task ran %d times, want 1", got)
	}
}

func TestRegistryRunDueRefusesOverlap(t *testing.T) {
	reg := NewRegistry()
	started := make(chan struct{})
	release := make(chan struct{})
	var once sync.Once

	reg.Register(Task{
		Name:     "slow",
		Interval: time.Nanosecond,
		Run: func(context.Context) error {
			once.Do(func() { close(started) })
			<-release
			return nil
		},
	})

	done := make(chan []Result, 1)
	go func() { done <- reg.RunDue(context.Background()) }()
	<-started

	results := reg.RunDue(context.Background())
	if len(results) != 1 || results[0].Outcome != OutcomeOverlap {
		t.Fatalf("concurrent pass should report overlap, got %+v", results)
	}

	close(release)
	first := <-done
	if len(first) != 1 || first[0].Outcome != OutcomeSucceeded {
		t.Fatalf("first pass should succeed, got %+v", first)
	}
}

func TestRegistryRunDueIsolatesFailures(t *testing.T) {
	reg := NewRegistry()
	sentinel := errors.New("boom")
	reg.Register(Task{
		Name:     "fails",
		Interval: time.Nanosecond,
		Run:      func(context.Context) error { return sentinel },
	})
	var healthyRuns int32
	reg.Register(Task{
		Name:     "healthy",
		Interval: time.Nanosecond,
		Run: func(context.Context) error {
			atomic.AddInt32(&healthyRuns, 1)
			return nil
		},
	})

	byName := map[string]Result{}
	for _, result := range reg.RunDue(context.Background()) {
		byName[result.Name] = result
	}

	failed, ok := byName["fails"]
	if !ok || failed.Outcome != OutcomeFailed || !errors.Is(failed.Err, sentinel) {
		t.Fatalf("failing task should be reported without aborting the pass, got %+v", failed)
	}
	if healthy, ok := byName["healthy"]; !ok || healthy.Outcome != OutcomeSucceeded {
		t.Fatalf("healthy task should still run after a failure, got %+v", healthy)
	}
	if atomic.LoadInt32(&healthyRuns) != 1 {
		t.Fatalf("healthy task ran %d times, want 1", healthyRuns)
	}
}

func TestRegistryRunDueRecoversFromPanic(t *testing.T) {
	reg := NewRegistry()
	reg.Register(Task{
		Name:     "panics",
		Interval: time.Nanosecond,
		Run:      func(context.Context) error { panic("kaboom") },
	})
	var afterPanic int32
	reg.Register(Task{
		Name:     "after",
		Interval: time.Nanosecond,
		Run: func(context.Context) error {
			atomic.AddInt32(&afterPanic, 1)
			return nil
		},
	})

	byName := map[string]Result{}
	for _, result := range reg.RunDue(context.Background()) {
		byName[result.Name] = result
	}

	if got := byName["panics"]; got.Outcome != OutcomeFailed || !errors.Is(got.Err, errTaskPanic) {
		t.Fatalf("panic should surface as a failed result, got %+v", got)
	}
	if atomic.LoadInt32(&afterPanic) != 1 {
		t.Fatal("runner must continue after a panicking task")
	}
}

func TestRegistryRunNowIgnoresInterval(t *testing.T) {
	reg := NewRegistry()
	var runs int32
	reg.Register(Task{
		Name:     "manual",
		Interval: time.Hour,
		Run: func(context.Context) error {
			atomic.AddInt32(&runs, 1)
			return nil
		},
	})

	if _, ok := reg.RunNow(context.Background(), "manual"); !ok {
		t.Fatal("registered task should be runnable on demand")
	}
	if _, ok := reg.RunNow(context.Background(), "missing"); ok {
		t.Fatal("unknown task must not report success")
	}
	if got := atomic.LoadInt32(&runs); got != 1 {
		t.Fatalf("task ran %d times, want 1", got)
	}
}

func TestRegistryRegisterIsIdempotent(t *testing.T) {
	reg := NewRegistry()
	task := Task{Name: "same", Interval: time.Hour, Run: func(context.Context) error { return nil }}
	reg.Register(task)
	reg.Register(task)
	if names := reg.Tasks(); len(names) != 1 {
		t.Fatalf("re-registering a task must not duplicate it, got %v", names)
	}
}

func TestRegistryZeroValueIntervalDefaults(t *testing.T) {
	reg := NewRegistry()
	reg.Register(Task{Name: "no-interval", Run: func(context.Context) error { return nil }})

	results := reg.RunDue(context.Background())
	if len(results) != 1 || results[0].Outcome != OutcomeSucceeded {
		t.Fatalf("task without an interval should still run, got %+v", results)
	}
	if results := reg.RunDue(context.Background()); results[0].Outcome != OutcomeSkipped {
		t.Fatalf("the default interval must throttle the next pass, got %+v", results)
	}
}

func TestRunStopsWithContext(t *testing.T) {
	reg := NewRegistry()
	var runs int32
	reg.Register(Task{
		Name:     "counts",
		Interval: time.Nanosecond,
		Run: func(context.Context) error {
			atomic.AddInt32(&runs, 1)
			return nil
		},
	})

	ctx, cancel := context.WithCancel(context.Background())
	stopped := make(chan struct{})
	go func() {
		Run(ctx, reg, time.Millisecond)
		close(stopped)
	}()

	deadline := time.After(2 * time.Second)
	for atomic.LoadInt32(&runs) == 0 {
		select {
		case <-deadline:
			t.Fatal("runner did not execute its first pass")
		case <-time.After(time.Millisecond):
		}
	}
	cancel()
	select {
	case <-stopped:
	case <-time.After(2 * time.Second):
		t.Fatal("runner did not stop after context cancellation")
	}
}
