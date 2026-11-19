# C01 queue recovery and canceled claim continuation — 2026-11-18

- Parent task: `docs/harness/tasks/2026-11-16-ops-service-deploy-integrity.task.md`
- Evidence/review: `.harness/evidence/2026-11-16-ops-service-deploy-integrity/`
- Tier: L2; repository/layer: pantheon-ops / business/deploy. Implementation remains in the business owner; no platform/system or schema changes.
- Implementer: C01 queue/host executor. Root owns integration, runtime evidence and status; independent review and final acceptance remain required.

## Confirmed facts and smallest approach

`runAsyncTaskHost` checks cancellation before calling `runOwnedHost`, but the latter's initial claim uses the service DB context. A task-row lock can hold that claim across parent cancellation. The existing request-bound service helper can pass the parent context into the claim transaction without copying worker maps or changing durable ownership contracts.

Startup reconcile/read and final recompute errors retire worker reservations. Process-local pending dispatch can be lost on those exits, while Running SSH tasks remain discoverable by `ReconcileDeployQueue`. Verify recovery deterministically before proposing a new dispatcher or durable generation.

## In / Out

In:

1. Add real MySQL task-row-lock regression and pre-canceled-parent regression. Cancellation must return while the blocker still holds its lock, with no new attempt, lease, execution or projection facts.
2. Reuse `targetRequestService(parent)` for the initial attempt claim only; preserve independently bounded ownership renewal, completion and Retire/RC repairs.
3. Inject startup reconcile/task/host reads and final recompute failures with a real pending reservation. Demonstrate retirement plus later durable queue recovery without duplicate attempts or writes.
4. Run parent-scheduled targeted CGO/race tests and inspect the scoped diff. Register exact filters and test readiness before production edits.

Out: frontend/locales/config/Base/schema changes; full race/rebuild/runtime/browser work, commits and publishing (root owns scheduling); redesigning cross-owner policies; blanket closure of host/provider/UI or whole C01 acceptance.

## File plan and checks

- Create `backend/modules/business/deploy/worker_claim_context_test.go` for blocked and pre-canceled claim barriers.
- Create `backend/modules/business/deploy/worker_queue_recovery_test.go` for injected-error and queue replay regressions.
- Modify only initial claim in `backend/modules/business/deploy/attempt_ownership.go` after RED is recorded.
- Target RED filter: `^TestWorkerClaimParent(CanceledBeforeClaim|CancellationWhileTaskLocked)$`.
- Target GREEN filter: `^TestWorker(ClaimParent|QueueRecovery)` plus existing ownership/worker-handoff/cancel/host-scope regressions as scheduled by root.
- No SQL-lock-wait telemetry is claimed: the barrier holds an actual SQL row lock, observes query entry and bounded non-completion, then requires cancellation before lock release. All failure paths release locks and join workers before fixture cleanup.
- Stop condition for this subtask: fresh targeted non-SKIP validation plus reviewed minimal diff; report residual acceptance gaps without changing root-owned task status.
