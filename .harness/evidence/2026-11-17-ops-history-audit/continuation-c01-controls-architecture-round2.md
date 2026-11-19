## Summary

**Architectural status: BLOCK.** The initial group lock inversion and deletion-reference handshake are repaired, but the inspected implementation still has three control-path defects: superseded owners prevent cancellation/recovery from converging; retire retries can reuse a manual execution token; and resume does not verify the frozen owner binding.

This report covers the source inspected before the coordinator’s proposed repairs. Those repairs require fresh inspection and verification. Foundation remains `pantheon-base-v0.18.9` (`foundation-release.lock.json:5`).

## Analysis

### 1. BLOCK — A superseded owner strands its deployment reference

A legal `UpdateInstance` increments `lifecycle_version`, including changes to desired version (`backend/modules/business/service/service.go:468`). Service reconciliation also advances lifecycle state (`backend/modules/business/service/state.go:149`). Deployment callbacks correctly reject a frozen lifecycle/transition binding that differs from the current instance (`backend/modules/business/deploy/service_link.go:137`).

However, cancellation uses that same strict completion guard:

- Direct cancellation calls `finishServiceInstanceState` and returns its error, rolling back task/host changes (`backend/modules/business/deploy/deploy_service.go:1751`).
- Terminal recomputation also returns the callback error, rolling back recovery and lease release (`backend/modules/business/deploy/attempt_ownership.go:717`).
- Queue reconciliation stops at the first requested task that fails recomputation (`backend/modules/business/deploy/deploy_service.go:1658`).
- The still-running/paused/cancel-requested task continues protecting the instance from deletion (`backend/modules/business/deploy/service_reference.go:37`).

**Consequence:** Update or reconcile → stale completion rejection → cancellation/recovery rejection leaves the old execution reference occupied. One such task can also interrupt startup recovery for subsequent tasks.

**Evidence boundary:** This is a statically grounded liveness defect. The existing test dynamically verifies rejection and unchanged persisted facts after owner update/reconciliation, but does not attempt cancellation or recovery afterward (`backend/modules/business/deploy/service_link_control_test.go:56`, `:74`).

**Smallest correct repair:** Distinguish a valid, readable owner whose frozen execution has been superseded from missing/malformed associations and capability/database failures. For cancellation only, allow Deploy task/hosts/leases to settle while preserving the newer Service state. Keep ordinary success/failure reports rejected. Do not blanket-ignore `state_stale` strings or owner failures.

### 2. BLOCK — Retire retry can accept a prior execution’s report

Manual execution identity currently hashes task IDs, frozen Service state and hosts (`backend/modules/business/deploy/service_link.go:30`).

Retire exposes a collision:

1. Retire has no begin transition (`backend/modules/business/deploy/deploy_service.go:2514`).
2. Failed retire leaves Service state unchanged (`backend/modules/business/deploy/deploy_service.go:2611`).
3. Retry replaces the frozen Service state with that same unchanged projection (`backend/modules/business/deploy/deploy_service.go:1946`, `:1950`).
4. The resulting execution token remains identical, so the prior report passes the current token comparison (`backend/modules/business/deploy/deploy_service.go:2297`).

The existing retry-token test exercises install, whose lifecycle changes; it does not establish correctness for actions without a begin transition (`backend/modules/business/deploy/service_link_control_test.go:117`).

**Smallest correct repair:** Add a per-task execution generation to the existing frozen snapshot. Initialize it on start, increment it under the task lock on every retry, and include it in the execution digest. This is execution identity, not a second Service state machine. Verify failed retire → retry → old failure/success rejection → fresh completion → terminal replay.

### 3. BLOCK — Resume admits a superseded frozen execution

Resume validates the current association and active execution scope, but never compares the admitted frozen lifecycle/transition binding (`backend/modules/business/deploy/deploy_service.go:1830`; `backend/modules/business/deploy/task_command.go:137`). A paused deployment whose instance has been updated can therefore resume its old execution and only discover the stale binding when finishing.

The direct `ResumeTask` entry also performs validation and mutation outside the common control transaction (`backend/modules/business/deploy/deploy_service.go:1822`, `:1834`), unlike the new direct retry/cancel wrappers (`:1860`, `:1686`).

**Smallest correct repair:** Route direct resume through the existing control transaction, then verify the frozen Service binding before the status change. Preserve the binding when continuing the same execution; reject superseded state without publishing a resume event or dispatching work.

### 4. Repaired architecture to retain

The inspected implementation resolves CMDB targets before locking Service rows, including plain group association validation (`backend/modules/business/deploy/service_link.go:66`). Deployment locking remains task → hosts → attempts → physical leases (`backend/modules/business/deploy/attempt_ownership.go:271`), while Service locking reads use instance → service → application (`backend/modules/business/service/service.go:628`, `:603`, `:588`).

Deletion now authorizes/locks the owner and reads Deploy-owned references without locking task rows, using READ COMMITTED (`backend/modules/business/service/deployment_reference.go:99`, `:107`; `backend/modules/business/deploy/service_reference.go:36`). Reference queries are not filtered by caller DataScope, preventing another user’s executing task from being hidden after owner authorization (`backend/modules/business/deploy/service_reference.go:38`).

Reader/state factories propagate through transaction, request, discovery and worker views (`backend/modules/business/deploy/target_selection.go:180`, `:233`, `:263`; `backend/modules/business/deploy/attempt_ownership.go:411`). The Service transaction seam retains the caller transaction/context and rejects a nontransaction connection (`backend/modules/business/service/reader_transaction.go:23`). Production wiring connects both directions before launching recovery (`backend/modules/business/deploy/deploy.go:51`, `:56`, `:60`; `backend/modules/business/opsinit/opsinit.go:69`).

No additional lock-order or production factory-propagation blocker was found in these inspected paths.

## Root Cause

The new design correctly separates task control versions from Service lifecycle versions, but execution identity still depends on whether a Service action happens to mutate lifecycle state. It also applies the same strict callback eligibility rule to result publication and cancellation cleanup. Those operations need different behavior when a newer owner state supersedes an old execution.

## Recommendations

1. **High priority, medium effort:** implement narrowly classified superseded-owner cancellation/recovery and deterministic liveness tests.
2. **High priority, low effort:** persist execution generation independently of Service lifecycle mutation; test retire retries.
3. **High priority, low effort:** make direct resume transactional and validate its frozen owner binding.
4. Refresh targeted race, full CGO race, official-lock rebuild/source binding and both independent review lanes after these changes.

## Architectural Status

`BLOCK`

## Trade-offs

| Option                                                     | Benefit                                                | Cost                                                 |
| ---------------------------------------------------------- | ------------------------------------------------------ | ---------------------------------------------------- |
| Reject every stale callback, including cancellation        | Strict projection protection                           | Strands references and recovery                      |
| Permit cancellation of a specifically superseded execution | Preserves newer owner state and releases old ownership | Requires narrow classification and liveness evidence |
| Derive tokens solely from Service lifecycle                | Fewer snapshot fields                                  | Fails for actions that leave lifecycle unchanged     |
| Freeze per-task execution generation                       | Reliable retry identity for every action               | One additional snapshot field                        |

## Validation and Stop Condition

The newest targeted CGO race has completed successfully: its receipt records `cgoEnabled: "1"` and `exitCode: 0` (`.harness/evidence/2026-11-17-ops-history-audit/continuation-c01-report-version-race.log.receipt.json:17`, `:19`). This supersedes the earlier “running” checkpoint for that run only.

It does not establish the newly identified cases or full closeout. The task explicitly still requires refreshed full race, formal-lock rebuild/source binding and independent reviews (`docs/harness/tasks/2026-11-16-ops-service-deploy-integrity.task.md:174`).

**Stop condition:** repair all three control-path defects, inspect the resulting source, and obtain fresh required evidence. C01 remains open; no closure approval is given.
