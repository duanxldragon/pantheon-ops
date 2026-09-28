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
	ConditionAlways       = "always"
	ConditionOnSuccess    = "on_success"
	ConditionOnFailure    = "on_failure"
	ConditionOnChanged    = "on_changed"
	ConditionParamEquals  = "param_equals"
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
	StepKey      string
	Dependencies []string
	Condition    string
	FailurePolicy string
	StepOrder    int
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
	byKey := make(map[string]DAGNode, len(nodes))
	seen := make(map[string]bool, len(nodes))
	var errors []string
	for _, n := range nodes {
		if strings.TrimSpace(n.StepKey) == "" {
			errors = append(errors, "step key must not be empty")
			continue
		}
		if seen[n.StepKey] {
			errors = append(errors, "duplicate step key: "+n.StepKey)
			continue
		}
		seen[n.StepKey] = true
		byKey[n.StepKey] = n
	}
	for _, n := range nodes {
		for _, dep := range n.Dependencies {
			if dep == n.StepKey {
				errors = append(errors, "step "+n.StepKey+" depends on itself")
				continue
			}
			if !seen[dep] {
				errors = append(errors, "step "+n.StepKey+" depends on unknown step "+dep)
			}
		}
	}
	// Cycle detection: DFS over dependency edges.
	state := make(map[string]int, len(nodes)) // 0=unvisited 1=visiting 2=done
	var visit func(key string) bool
	visit = func(key string) bool {
		switch state[key] {
		case 1:
			return true // back edge -> cycle
		case 2:
			return false
		}
		state[key] = 1
		node, ok := byKey[key]
		if ok {
			for _, dep := range node.Dependencies {
				if visit(dep) {
					return true
				}
			}
		}
		state[key] = 2
		return false
	}
	for key := range byKey {
		if visit(key) {
			errors = append(errors, "cycle detected in step dependencies")
			break
		}
	}
	if len(errors) == 0 {
		return DAGValidation{Valid: true}
	}
	return DAGValidation{Valid: false, Errors: errors}
}

// RunHostSteps executes the plan against the provider registered for its
// executor type, honoring conditions, dependencies and failure policies.
// The sink receives per-step log output. Execution stops on the first
// unrecoverable step failure unless the step policy allows continuation.
func (o *Orchestrator) RunHostSteps(ctx context.Context, plan executor.ExecutionPlan, sink executor.LogSink) HostExecution {
	provider, ok := o.registry.Get(plan.ExecutorType)
	if !ok {
		return HostExecution{Status: StatusFailed, Steps: []StepResult{{
			StepKey: planFirstStepKey(plan),
			Status:  StatusFailed,
			Error:   "no executor provider registered for type: " + plan.ExecutorType,
		}}}
	}
	if err := provider.Validate(ctx, plan); err != nil {
		return HostExecution{Status: StatusFailed, Steps: []StepResult{{
			StepKey: planFirstStepKey(plan),
			Status:  StatusFailed,
			Error:   err.Error(),
		}}}
	}

	ordered := orderSteps(plan)
	results := make([]StepResult, 0, len(ordered))
	executed := make(map[string]StepResult, len(ordered))
	failed := false
	canceled := false

	for _, step := range ordered {
		if ctx.Err() != nil {
			canceled = true
			break
		}
		// Condition evaluation.
		skip, err := evaluateCondition(step, executed, plan)
		if err != nil {
			results = append(results, StepResult{StepKey: step.StepKey, Status: StatusFailed, Error: err.Error()})
			failed = true
			if !stepContinueOnFailure(step) {
				break
			}
			continue
		}
		if skip {
			results = append(results, StepResult{StepKey: step.StepKey, Status: "skipped"})
			executed[step.StepKey] = StepResult{StepKey: step.StepKey, Status: "skipped"}
			continue
		}
		// Checkpoint resume: skip steps already in a terminal successful state.
		if step.InitialStatus == "success" || step.InitialStatus == "skipped" {
			results = append(results, StepResult{StepKey: step.StepKey, Status: step.InitialStatus, Changed: true})
			executed[step.StepKey] = StepResult{StepKey: step.StepKey, Status: step.InitialStatus, Changed: true}
			continue
		}

		res := provider.ExecuteStep(ctx, executor.StepRequest{
			TaskID:     plan.TaskID,
			TaskHostID: plan.TaskHostID,
			StepKey:    step.StepKey,
			StepType:   step.StepType,
			Actor:      plan.Actor,
		}, sink)
		sr := StepResult{StepKey: step.StepKey, Status: res.Status, Changed: res.Changed, Error: res.ErrorMessage}
		results = append(results, sr)
		executed[step.StepKey] = sr

		switch res.Status {
		case StatusFailed:
			failed = true
			if !stepContinueOnFailure(step) {
				return HostExecution{Status: StatusFailed, Steps: results}
			}
		case StatusCanceled:
			canceled = true
		}
	}

	switch {
	case canceled:
		return HostExecution{Status: StatusCanceled, Steps: results}
	case failed:
		return HostExecution{Status: StatusFailed, Steps: results}
	default:
		return HostExecution{Status: StatusSuccess, Steps: results}
	}
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
		for _, dep := range step.Dependencies {
			if r, ok := executed[dep]; ok && r.Status != StatusSuccess && r.Status != "skipped" {
				return true, nil
			}
		}
		return false, nil
	case ConditionOnFailure:
		for _, dep := range step.Dependencies {
			if r, ok := executed[dep]; ok && r.Status == StatusFailed {
				return false, nil
			}
		}
		return true, nil
	case ConditionOnChanged:
		for _, dep := range step.Dependencies {
			if r, ok := executed[dep]; ok && r.Changed {
				return false, nil
			}
		}
		return true, nil
	case ConditionParamEquals:
		expected, ok := step.ConditionParams["value"].(string)
		if !ok || expected == "" {
			return false, nil
		}
		actual, ok := step.ConditionParams["current"].(string)
		return !ok || actual != expected, nil
	default:
		return false, nil
	}
}
