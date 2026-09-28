// Package maintenance runs periodic infrastructure housekeeping — retention
// sweeps and inventory governance — outside of request paths.
//
// Background (task 2026-09-22-request-path-maintenance): list/export handlers
// used to run full-table DELETE sweeps inline, so an ordinary read could pay
// for an unbounded purge and hide its failure behind a slower response. Those
// sweeps now register here and run on their own schedule; request paths stay
// read-only.
//
// The registry owns throttle and overlap state, so each task implementation
// only has to describe "how to purge", never "how often".
package maintenance

import (
	"context"
	"sync"
	"time"

	"github.com/duanxldragon/pantheon-base/backend/pkg/logging"
	"github.com/duanxldragon/pantheon-base/backend/pkg/metrics"

	"go.uber.org/zap"
)

// Task is one periodic maintenance unit.
//
// Interval is the minimum gap between two runs of this task. Run must be safe
// to call repeatedly, must tolerate concurrent explicit invocations from the
// cleanup endpoints it also backs, and should return an error instead of
// logging-and-continuing so failures stay measurable.
type Task struct {
	Name     string
	Interval time.Duration
	Run      func(ctx context.Context) error
}

// Outcome classifies what one RunDue pass did with a task.
type Outcome string

const (
	// OutcomeSucceeded means Run ran and returned nil.
	OutcomeSucceeded Outcome = "succeeded"
	// OutcomeFailed means Run ran and returned an error (or panicked).
	OutcomeFailed Outcome = "failed"
	// OutcomeSkipped means the task's Interval had not elapsed yet.
	OutcomeSkipped Outcome = "skipped"
	// OutcomeOverlap means a previous run of the same task was still in
	// flight, so this pass refused to start a second one.
	OutcomeOverlap Outcome = "overlap"
	// OutcomeUnknown means the caller asked for a task that is not registered.
	OutcomeUnknown Outcome = "unknown"
)

// Result reports what happened to one task during one pass.
type Result struct {
	Name     string
	Outcome  Outcome
	Duration time.Duration
	Err      error
}

type taskState struct {
	task    Task
	running bool
	lastRun time.Time
}

// Registry owns the registered tasks plus their throttle/overlap state.
// It is safe for concurrent use.
type Registry struct {
	mu    sync.Mutex
	tasks map[string]*taskState
	order []string
	now   func() time.Time
}

// NewRegistry creates an empty registry.
func NewRegistry() *Registry {
	return &Registry{
		tasks: make(map[string]*taskState),
		now:   time.Now,
	}
}

var defaultRegistry = NewRegistry()

// Default returns the process-wide registry that module wiring registers into
// and cmd/server runs. Tests should prefer NewRegistry so runs do not leak
// across cases.
func Default() *Registry { return defaultRegistry }

// Register adds a task, replacing any task already registered under the same
// name. Registration is idempotent so wiring that runs more than once (module
// init in tests, repeated composition) cannot accumulate duplicates.
func (r *Registry) Register(task Task) {
	if r == nil || task.Name == "" || task.Run == nil {
		return
	}
	if task.Interval <= 0 {
		task.Interval = DefaultInterval
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.tasks[task.Name]; !exists {
		r.order = append(r.order, task.Name)
	}
	// Preserve existing throttle state on re-registration: a task that just
	// ran must not be reset into an immediate second sweep by re-wiring.
	if state, ok := r.tasks[task.Name]; ok {
		state.task = task
		return
	}
	r.tasks[task.Name] = &taskState{task: task}
}

// Tasks returns the registered task names in registration order.
func (r *Registry) Tasks() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	names := make([]string, len(r.order))
	copy(names, r.order)
	return names
}

// RunDue runs every task whose Interval has elapsed, in registration order.
//
// A task already in flight is skipped rather than started twice, and a failing
// task never aborts the pass: failures are logged, counted, and returned in
// the results so the caller (or /metrics) can see them. The returned error is
// always nil for context cancellation being the only stop signal.
func (r *Registry) RunDue(ctx context.Context) []Result {
	if r == nil {
		return nil
	}
	r.mu.Lock()
	names := make([]string, len(r.order))
	copy(names, r.order)
	r.mu.Unlock()

	results := make([]Result, 0, len(names))
	for _, name := range names {
		if ctx != nil {
			select {
			case <-ctx.Done():
				return results
			default:
			}
		}
		results = append(results, r.run(ctx, name, false))
	}
	return results
}

// RunNow runs a single task, ignoring its interval throttle. It still refuses
// to overlap a task that is currently running, so an explicit cleanup request
// cannot double up with the background sweep. The bool reports whether the
// task name is registered at all.
func (r *Registry) RunNow(ctx context.Context, name string) (Result, bool) {
	if r == nil {
		return Result{Name: name, Outcome: OutcomeUnknown}, false
	}
	r.mu.Lock()
	_, ok := r.tasks[name]
	r.mu.Unlock()
	if !ok {
		return Result{Name: name, Outcome: OutcomeUnknown}, false
	}
	return r.run(ctx, name, true), true
}

func (r *Registry) run(ctx context.Context, name string, force bool) Result {
	r.mu.Lock()
	state, ok := r.tasks[name]
	if !ok {
		r.mu.Unlock()
		return Result{Name: name, Outcome: OutcomeUnknown}
	}
	now := r.now()
	// Overlap is reported before the interval throttle: a task that is still
	// running is the more actionable fact, and the caller may have forced a run.
	if state.running {
		r.mu.Unlock()
		result := Result{Name: name, Outcome: OutcomeOverlap}
		recordOutcome(result)
		logging.Warn("maintenance task skipped: previous run still in flight", zap.String("task", name))
		return result
	}
	if !force && !state.lastRun.IsZero() && now.Sub(state.lastRun) < state.task.Interval {
		r.mu.Unlock()
		return Result{Name: name, Outcome: OutcomeSkipped}
	}
	state.running = true
	state.lastRun = now
	task := state.task
	r.mu.Unlock()

	return r.finish(ctx, task)
}

// finish executes task.Run and releases the in-flight claim.
func (r *Registry) finish(ctx context.Context, task Task) Result {
	startedAt := r.now()
	result := Result{Name: task.Name}
	err := runGuarded(ctx, task)
	result.Duration = r.now().Sub(startedAt)
	if err != nil {
		result.Outcome = OutcomeFailed
		result.Err = err
		logging.Error("maintenance task failed",
			zap.String("task", task.Name), zap.Duration("duration", result.Duration), zap.Error(err))
	} else {
		result.Outcome = OutcomeSucceeded
		logging.Info("maintenance task finished",
			zap.String("task", task.Name), zap.Duration("duration", result.Duration))
	}
	recordOutcome(result)

	r.mu.Lock()
	if state, ok := r.tasks[task.Name]; ok {
		state.running = false
	}
	r.mu.Unlock()
	return result
}

// runGuarded isolates a task failure from the runner: a panic becomes an
// error so one broken sweep cannot take down the maintenance loop.
func runGuarded(ctx context.Context, task Task) (err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			logging.Error("maintenance task panicked",
				zap.String("task", task.Name), zap.Any("panic", recovered))
			err = errTaskPanic
		}
	}()
	return task.Run(ctx)
}

func recordOutcome(result Result) {
	metrics.MaintenanceRunsTotal.WithLabelValues(result.Name, string(result.Outcome)).Inc()
	if result.Outcome == OutcomeSucceeded || result.Outcome == OutcomeFailed {
		metrics.MaintenanceDuration.WithLabelValues(result.Name).Observe(result.Duration.Seconds())
	}
}

// Run starts the periodic runner and blocks until ctx is done. The first pass
// runs immediately so a long-idle deployment does not wait a full interval
// before the first sweep; subsequent passes run every interval.
func Run(ctx context.Context, registry *Registry, interval time.Duration) {
	if interval <= 0 {
		interval = DefaultInterval
	}
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	runPass(runCtx, registry)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-runCtx.Done():
			return
		case <-ticker.C:
			runPass(runCtx, registry)
		}
	}
}

func runPass(ctx context.Context, registry *Registry) {
	for _, result := range registry.RunDue(ctx) {
		if result.Outcome == OutcomeFailed {
			// Already logged per-task in finish(); this is the pass-level
			// roll-up so a failing sweep is visible even in quiet logs.
			logging.Warn("maintenance pass contained a failed task", zap.String("task", result.Name))
		}
	}
}
