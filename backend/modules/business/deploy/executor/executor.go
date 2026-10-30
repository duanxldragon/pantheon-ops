// Package executor defines the provider boundary for remote task execution.
//
// Providers receive a frozen task/host execution plan and must never persist
// secret material. The registry maps executor types to providers so that the
// orchestrator can dispatch steps without depending on concrete backends.
package executor

import "context"

// ExecutorTypeSSH is the canonical executor type for SSH-based providers.
// It is mirrored by the deploy package constant; kept here so the provider
// boundary is self-contained.
const ExecutorTypeSSH = "ssh"

// ExecutionPlan is the frozen execution context supplied to a provider for one
// target host. Every field is a snapshot taken at task start; providers must
// not reach back into the database for execution decisions.
type ExecutionPlan struct {
	TaskID           uint64
	TaskHostID       uint64
	TaskName         string
	Action           string
	ExecutorType     string
	TargetIP         string
	TargetPort       int
	TargetOS         string
	MaxParallelSteps int
	Actor            string
	Steps            []StepPlan
}

// StepPlan describes one step within an execution plan.
type StepPlan struct {
	StepKey               string
	StepName              string
	StepType              string
	Action                string
	ContinueOnHostFailure bool
	PackageName           string
	PackageVersion        string
	TemplateParams        map[string]any
	Dependencies          []string
	Condition             string
	ConditionParams       map[string]any
	FailurePolicy         string
	InitialStatus         string
}

// StepRequest carries the identity of the step a provider must execute.
type StepRequest struct {
	TaskID     uint64
	TaskHostID uint64
	StepKey    string
	StepType   string
	Actor      string
}

// StepResult is the outcome of a single step execution.
type StepResult struct {
	Status       string
	ErrorMessage string
	Changed      bool
}

// LogWriter receives appended log output for a step, keyed by stream
// (for example "stdout" or "stderr").
type LogWriter interface {
	WriteLog(taskID, taskHostID uint64, stepKey, stream, content string)
}

// ExecutionHandle allows a provider to be cancelled asynchronously.
type ExecutionHandle struct {
	CancelFunc func()
}

// StepProvider is the provider contract implemented by concrete backends
// (for example SSH). Providers are registered in a Registry by Type and
// invoked by the orchestrator.
type StepProvider interface {
	// Type returns the executor type this provider implements.
	Type() string
	// Validate checks a plan before execution starts.
	Validate(ctx context.Context, plan ExecutionPlan) error
	// ExecuteStep runs one step and reports its result.
	ExecuteStep(ctx context.Context, req StepRequest, logWriter LogWriter) StepResult
	// Cancel requests cancellation of an in-flight execution.
	Cancel(ctx context.Context, handle ExecutionHandle) error
}

// Registry maps executor types to registered providers.
type Registry struct {
	providers map[string]StepProvider
}

// NewRegistry creates an empty provider registry.
func NewRegistry() *Registry {
	return &Registry{providers: make(map[string]StepProvider)}
}

// Register adds a provider keyed by its Type.
func (r *Registry) Register(provider StepProvider) {
	if provider == nil {
		return
	}
	r.providers[provider.Type()] = provider
}

// Get returns the provider registered for the given executor type.
func (r *Registry) Get(executorType string) (StepProvider, bool) {
	p, ok := r.providers[executorType]
	return p, ok
}

// Types returns the sorted executor types currently registered.
func (r *Registry) Types() []string {
	types := make([]string, 0, len(r.providers))
	for t := range r.providers {
		types = append(types, t)
	}
	return types
}
