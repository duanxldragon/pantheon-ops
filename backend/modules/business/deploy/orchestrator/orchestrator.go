// Package orchestrator runs multi-step execution plans against registered
// step providers, evaluating step conditions, dependencies and failure
// policies in a deterministic order.
package orchestrator

import (
	"context"
	"sort"
	"strings"

	"github.com/duanxldragon/pantheon-base/backend/modules/business/deploy/executor"
)

// Step condition types. Values mirror the template condition payloads used by
// the deploy module (condition["type"]).
const (
	ConditionAlways      = "always"
	ConditionOnSuccess   = "on_success"
	ConditionOnFailure   = "on_failure"
	ConditionOnChanged   = "on_changed"
	ConditionParamEquals = "param_equals"
)

// Host execution terminal statuses. Values are shared with the deploy step
// status vocabulary so callers can compare directly.
const (
	StatusSuccess  = "success"
	StatusFailed   = "failed"
	StatusCanceled = "canceled"
)

// DAGNode is one node of the execution graph used for validation and ordering.
type DAGNode struct {
	StepKey       string
	Dependencies  []string
	Condition     string
	FailurePolicy string
	StepOrder     int
}

// DAGValidation reports whether a set of nodes forms a valid DAG and lists
// the offending errors otherwise.
type DAGValidation struct {
	Valid  bool
	Errors []string
}

// StepResult is the orchestrator's view of one executed step.
type StepResult struct {
	StepKey string
	Status  string
	Error   string
	Changed bool
}

// HostExecution aggregates the outcome of a whole host run.
type HostExecution struct {
	Status string
	Steps  []StepResult
}

// Orchestrator dispatches plan steps to providers from a shared registry.
type Orchestrator struct {
	registry *executor.Registry
}

// New creates an orchestrator backed by the given provider registry.
func New(registry *executor.Registry) *Orchestrator {
	return &Orchestrator{registry: registry}
}

// ValidateDAG checks node references, cycles and duplicate keys. A missing
// dependency, a self-referencing step or a cycle is reported as invalid.
func ValidateDAG(nodes []DAGNode) DAGValidation {
	byKey, known, errs := indexDAGNodes(nodes)
	errs = append(errs, validateDAGReferences(nodes, known)...)
	if hasDAGCycle(byKey) {
		errs = append(errs, "cycle detected in step dependencies")
	}
	if len(errs) == 0 {
		return DAGValidation{Valid: true}
	}
	return DAGValidation{Valid: false, Errors: errs}
}

// indexDAGNodes maps the nodes by step key and reports empty or duplicate keys.
func indexDAGNodes(nodes []DAGNode) (map[string]DAGNode, map[string]bool, []string) {
	byKey := make(map[string]DAGNode, len(nodes))
	seen := make(map[string]bool, len(nodes))
	var errs []string
	for _, n := range nodes {
		if strings.TrimSpace(n.StepKey) == "" {
			errs = append(errs, "step key must not be empty")
			continue
		}
		if seen[n.StepKey] {
			errs = append(errs, "duplicate step key: "+n.StepKey)
			continue
		}
		seen[n.StepKey] = true
		byKey[n.StepKey] = n
	}
	return byKey, seen, errs
}

// validateDAGReferences reports self references and dependencies on unknown steps.
func validateDAGReferences(nodes []DAGNode, known map[string]bool) []string {
	var errs []string
	for _, n := range nodes {
		for _, dep := range n.Dependencies {
			if dep == n.StepKey {
				errs = append(errs, "step "+n.StepKey+" depends on itself")
				continue
			}
			if !known[dep] {
				errs = append(errs, "step "+n.StepKey+" depends on unknown step "+dep)
			}
		}
	}
	return errs
}

// hasDAGCycle walks the dependency edges depth first and reports a back edge.
func hasDAGCycle(byKey map[string]DAGNode) bool {
	const (
		visiting = 1
		done     = 2
	)
	state := make(map[string]int, len(byKey)) // 0=unvisited 1=visiting 2=done
	var visit func(key string) bool
	visit = func(key string) bool {
		switch state[key] {
		case visiting:
			return true // back edge -> cycle
		case done:
			return false
		}
		state[key] = visiting
		node, ok := byKey[key]
		if ok {
			for _, dep := range node.Dependencies {
				if visit(dep) {
					return true
				}
			}
		}
		state[key] = done
		return false
	}
	for key := range byKey {
		if visit(key) {
			return true
		}
	}
	return false
}

// RunHostSteps executes the plan against the provider registered for its
// executor type, honoring conditions, dependencies and failure policies.
// The sink receives per-step log output. Execution stops on the first
// unrecoverable step failure unless the step policy allows continuation.
func (o *Orchestrator) RunHostSteps(ctx context.Context, plan executor.ExecutionPlan, sink executor.LogWriter) HostExecution {
	provider, ok := o.registry.Get(plan.ExecutorType)
	if !ok {
		return singleStepFailure(planFirstStepKey(plan), "no executor provider registered for type: "+plan.ExecutorType)
	}
	if err := provider.Validate(ctx, plan); err != nil {
		return singleStepFailure(planFirstStepKey(plan), err.Error())
	}
	ordered := orderSteps(plan)
	run := hostStepRun{
		provider: provider,
		plan:     plan,
		sink:     sink,
		results:  make([]StepResult, 0, len(ordered)),
		executed: make(map[string]StepResult, len(ordered)),
	}
	for _, step := range ordered {
		if ctx.Err() != nil {
			run.canceled = true
			break
		}
		result, outcome := run.runStep(ctx, step)
		if outcome == stepTerminatesRun {
			return result
		}
		if outcome == stepStopsRun {
			break
		}
	}
	return run.finalize()
}

// stepOutcome tells the host loop whether to keep going, end with the
// accumulated status, or return a terminal failure immediately.
type stepOutcome int

const (
	stepContinues stepOutcome = iota
	stepStopsRun
	stepTerminatesRun
)

// hostStepRun carries the accumulating outcome of one host run.
type hostStepRun struct {
	provider executor.StepProvider
	plan     executor.ExecutionPlan
	sink     executor.LogWriter
	results  []StepResult
	executed map[string]StepResult
	failed   bool
	canceled bool
}

// runStep executes one ordered step. A condition error ends the run through the
// accumulated status, while a failed step returns its terminal execution right
// away unless the step policy allows continuation.
func (r *hostStepRun) runStep(ctx context.Context, step executor.StepPlan) (HostExecution, stepOutcome) {
	skip, err := evaluateCondition(step, r.executed, r.plan)
	if err != nil {
		// A condition error is reported for the step but never counts as an
		// executed outcome for its dependents.
		r.results = append(r.results, StepResult{StepKey: step.StepKey, Status: StatusFailed, Error: err.Error()})
		r.failed = true
		if stepContinueOnFailure(step) {
			return HostExecution{}, stepContinues
		}
		return HostExecution{}, stepStopsRun
	}
	if skip {
		r.remember(StepResult{StepKey: step.StepKey, Status: "skipped"})
		return HostExecution{}, stepContinues
	}
	// Checkpoint resume: skip steps already in a terminal successful state.
	if step.InitialStatus == "success" || step.InitialStatus == "skipped" {
		r.remember(StepResult{StepKey: step.StepKey, Status: step.InitialStatus, Changed: true})
		return HostExecution{}, stepContinues
	}
	res := r.provider.ExecuteStep(ctx, executor.StepRequest{
		TaskID:     r.plan.TaskID,
		TaskHostID: r.plan.TaskHostID,
		StepKey:    step.StepKey,
		StepType:   step.StepType,
		Actor:      r.plan.Actor,
	}, r.sink)
	r.remember(StepResult{StepKey: step.StepKey, Status: res.Status, Changed: res.Changed, Error: res.ErrorMessage})
	switch res.Status {
	case StatusFailed:
		r.failed = true
		if stepContinueOnFailure(step) {
			return HostExecution{}, stepContinues
		}
		return HostExecution{Status: StatusFailed, Steps: r.results}, stepTerminatesRun
	case StatusCanceled:
		r.canceled = true
	}
	return HostExecution{}, stepContinues
}

// remember appends a step outcome and exposes it to the conditions of the
// following steps.
func (r *hostStepRun) remember(res StepResult) {
	r.results = append(r.results, res)
	r.executed[res.StepKey] = res
}

func (r *hostStepRun) finalize() HostExecution {
	switch {
	case r.canceled:
		return HostExecution{Status: StatusCanceled, Steps: r.results}
	case r.failed:
		return HostExecution{Status: StatusFailed, Steps: r.results}
	default:
		return HostExecution{Status: StatusSuccess, Steps: r.results}
	}
}

func singleStepFailure(stepKey, message string) HostExecution {
	return HostExecution{Status: StatusFailed, Steps: []StepResult{{
		StepKey: stepKey,
		Status:  StatusFailed,
		Error:   message,
	}}}
}

func planFirstStepKey(plan executor.ExecutionPlan) string {
	if len(plan.Steps) > 0 {
		return plan.Steps[0].StepKey
	}
	return ""
}

func stepContinueOnFailure(step executor.StepPlan) bool {
	return step.ContinueOnHostFailure || strings.Contains(step.FailurePolicy, "continue")
}

// orderSteps returns plan steps ordered by StepOrder, honoring dependencies
// loosely (dependencies run before dependents where declared).
func orderSteps(plan executor.ExecutionPlan) []executor.StepPlan {
	steps := make([]executor.StepPlan, len(plan.Steps))
	copy(steps, plan.Steps)
	sort.SliceStable(steps, func(i, j int) bool {
		return steps[i].StepKey < steps[j].StepKey
	})
	_ = steps // replaced by deterministic dependency-aware ordering below
	steps = topoOrder(plan.Steps)
	return steps
}

func topoOrder(steps []executor.StepPlan) []executor.StepPlan {
	byKey := make(map[string]executor.StepPlan, len(steps))
	for _, s := range steps {
		byKey[s.StepKey] = s
	}
	var out []executor.StepPlan
	visited := make(map[string]int)
	var visit func(key string)
	visit = func(key string) {
		s, ok := byKey[key]
		if !ok || visited[key] == 2 {
			return
		}
		if visited[key] == 1 {
			return // cycle guarded by ValidateDAG
		}
		visited[key] = 1
		for _, dep := range s.Dependencies {
			visit(dep)
		}
		visited[key] = 2
		out = append(out, s)
	}
	for _, s := range steps {
		visit(s.StepKey)
	}
	return out
}

// evaluateCondition decides whether a step should run given the condition type
// and the outcomes of previously executed dependency steps.
func evaluateCondition(step executor.StepPlan, executed map[string]StepResult, plan executor.ExecutionPlan) (skip bool, err error) {
	cond := step.Condition
	if cond == "" || cond == ConditionAlways {
		return false, nil
	}
	switch cond {
	case ConditionOnSuccess:
		return dependenciesNotSatisfied(executed, step.Dependencies), nil
	case ConditionOnFailure:
		return !dependenciesAnyFailed(executed, step.Dependencies), nil
	case ConditionOnChanged:
		return !dependenciesAnyChanged(executed, step.Dependencies), nil
	case ConditionParamEquals:
		return paramEqualsSkips(step), nil
	default:
		return false, nil
	}
}

// dependenciesNotSatisfied reports whether an executed dependency ended in a
// state other than success or skipped.
func dependenciesNotSatisfied(executed map[string]StepResult, dependencies []string) bool {
	for _, dep := range dependencies {
		result, ok := executed[dep]
		if ok && result.Status != StatusSuccess && result.Status != "skipped" {
			return true
		}
	}
	return false
}

// dependenciesAnyFailed reports whether an executed dependency failed.
func dependenciesAnyFailed(executed map[string]StepResult, dependencies []string) bool {
	for _, dep := range dependencies {
		result, ok := executed[dep]
		if ok && result.Status == StatusFailed {
			return true
		}
	}
	return false
}

// dependenciesAnyChanged reports whether an executed dependency changed the
// host.
func dependenciesAnyChanged(executed map[string]StepResult, dependencies []string) bool {
	for _, dep := range dependencies {
		result, ok := executed[dep]
		if ok && result.Changed {
			return true
		}
	}
	return false
}

// paramEqualsSkips resolves the param_equals condition against the current and
// expected values carried by the step condition parameters.
func paramEqualsSkips(step executor.StepPlan) bool {
	expected, ok := step.ConditionParams["value"].(string)
	if !ok || expected == "" {
		return false
	}
	actual, ok := step.ConditionParams["current"].(string)
	return !ok || actual != expected
}
