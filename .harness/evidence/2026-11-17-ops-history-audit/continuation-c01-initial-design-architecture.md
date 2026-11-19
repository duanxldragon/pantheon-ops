## Summary

**Architectural status: BLOCK for C01 closeout.** The initial transaction seams are a sound direction, but plain group creation still has a concrete lock-order inversion. The deletion guard and fresh regression evidence remain outstanding; this review does not accept the implementation.

Keep the fix inside `business/deploy`, `business/service`, and their composition root. The pinned foundation remains `pantheon-base-v0.18.9` (`foundation-release.lock.json:5`).

## Analysis

### 1. Preserve the new owner transaction seam

`transactionManager` rejects a nontransaction DB, retains the caller’s connection pool and context, disables independent default write transactions, and enables current locking reads (`backend/modules/business/service/reader_transaction.go:22`).

The owner reads now lock **instance → service → scoped application**, avoiding authorization through a stale nested subquery (`backend/modules/business/service/service.go:622`, `:600`, `:585`). The state-command factory reuses that same seam (`backend/modules/business/service/state.go:31`).

`validateServiceLink` reads the instance first and rejects a mismatched requested service before fetching that service (`backend/modules/business/deploy/service_link.go:48`, `:53`). This avoids acquiring two unrelated service hierarchies for a mismatched pair.

**Recommendation:** retain this order rather than introducing a second hierarchy-lock abstraction. DeleteInstance must follow the same order. DeleteService may lock service → application, but must not subsequently lock child instances.

### 2. Fix the plain-group lock inversion

The existing execution order is Deploy task → hosts → attempts → leases, followed by CMDB and then Service (`backend/modules/business/deploy/attempt_ownership.go:271`, `:844`, `:891`).

Plain group creation currently takes the opposite final order:

1. `resolveCreateTaskScopeName` returns without resolving CMDB targets for a group (`backend/modules/business/deploy/deploy_service.go:580`).
2. `validateServiceLink` locks the Service hierarchy (`backend/modules/business/deploy/service_link.go:48`).
3. With `resolved == nil`, it then resolves group targets (`:73`).
4. The transaction-bound CMDB capability locks host rows (`backend/modules/business/cmdb/capability.go:507`; `backend/modules/business/cmdb/target_selection.go:74`).

A group create can therefore hold Service while waiting for CMDB, while a host execution transaction holds CMDB while waiting for Service. **This is a statically grounded deadlock path; it has not yet been dynamically reproduced.**

**Minimal repair:** resolve linked targets through the transaction-bound CMDB owner before calling the locking Service reader, then pass those resolved hosts into `validateServiceLink`. Preserve exact single-host matching and Kubernetes rejection (`backend/modules/business/deploy/service_link.go:67`, `:77`). Check group-create/update scope behavior together; current update scope normalization treats non-host targets differently.

### 3. Keep begin-state inside the claim transaction

The new claim invokes final preview validation and then the transaction-bound begin-state command before changing the task to running (`backend/modules/business/deploy/deploy_service.go:1322`, `:1329`, `:1333`). This closes the earlier split between claim commit and Service begin-state.

The outer mutation transaction also covers ordinary and selection-based create/update paths and locks the task before update (`backend/modules/business/deploy/target_selection.go:37`, `:104`, `:271`).

The start claim still reserves tentative task-host rows and leases before final locked validation (`backend/modules/business/deploy/deploy_service.go:1307`, `:1317`). Moving Service locks ahead of those operations would conflict with the established execution order.

**Recommendation:** retain side-effect-free discovery before claim, followed by authoritative validation and atomic rollback inside claim. Update `continuation-plan.md:7` to distinguish tentative transaction writes from committed effects. The original packet requires rejection without partial effects (`docs/harness/tasks/2026-11-16-ops-service-deploy-integrity.task.md:61`); do not claim that no SQL writes occurred before final validation.

### 4. Implement deletion protection without reversing task locks

DeleteInstance currently has no Deploy reference guard (`backend/modules/business/service/service.go:496`). DeleteService currently checks only Service instances (`backend/modules/business/service/service.go:356`).

Use a narrow Deploy-owned reference capability with these properties:

- Service authorizes and locks its owner row first.
- Deploy reads its own executing references on the **same READ COMMITTED transaction**, without `FOR UPDATE` on Deploy tasks.
- After owner authorization, the reference query must include all references to that owner ID; caller DataScope must not hide another user’s task.
- Include running, pause-requested, paused, and cancel-requested references. Keep terminal history and existing Service child-reference protection.
- Missing reference capability or query errors must reject deletion.

Do not use a locking task-reference query under Service locks: that would reverse the task-first execution order (`backend/modules/business/deploy/attempt_ownership.go:271`).

Explicit READ COMMITTED matters here. Deployment transactions already select it (`backend/modules/business/deploy/attempt_ownership.go:252`); the mutation wrapper currently uses GORM’s default transaction options (`backend/modules/business/deploy/target_selection.go:274`). A deletion transaction must not reuse an earlier InnoDB snapshot after waiting for an admission transaction.

The concurrency argument to test is:

- **Admission wins:** deletion waits on the owner lock, then sees the committed executing reference and rejects.
- **Deletion wins:** admission subsequently finds the owner deleted and rolls back.
- **Completion overlaps deletion:** the reference read does not wait on completion’s task lock, avoiding a task/Service cycle.

### 5. Preserve callback verification as an acceptance requirement

Current begin/finish callbacks omit `ExpectedLifecycleVersion` (`backend/modules/business/deploy/deploy_service.go:2387`). Service’s lifecycle-version predicate is conditional on that field (`backend/modules/business/service/state.go:100`).

Current row locks serialize operations, but do not by themselves prove replay or out-of-order behavior. Keep the packet’s paired completion/failure, replay, out-of-order, and concurrent-deletion tests before closing C01 (`.harness/tasks/2026-11-16-ops-service-deploy-integrity/continuation-plan.md:9`). Reuse existing fencing, correlation IDs, and lifecycle version if those tests expose an ordering gap.

## Root Cause

Deploy previously persisted association IDs without owner validation, and Service deletion lacked an executing-reference handshake. Atomic admission needs both owner-row serialization and a reference read with fresh visibility; adding locks independently can replace the association defect with deadlocks.

The preserved real-MySQL race run proves invalid create/update admission and mutation, not concurrency correctness (`backend/modules/business/deploy/service_link_admission_test.go:29`, `:36`, `:57`, `:64`; `.harness/evidence/2026-11-17-ops-history-audit/continuation-c01-service-link-red.log.receipt.json:21`).

## Recommendations

1. **High priority, low effort:** resolve linked group targets before Service locks; retain exact host matching.
2. **High priority, medium effort:** add same-transaction, nonlocking READ COMMITTED deletion references and deterministic concurrent tests.
3. **Medium effort:** verify rollback of task, secrets, task-host rows, leases, and Service state; then verify callbacks and production wiring before queue reconciliation starts (`backend/modules/business/deploy/deploy.go:44`).

## Trade-offs

| Approach                                                   | Benefit                                | Cost                                                                      |
| ---------------------------------------------------------- | -------------------------------------- | ------------------------------------------------------------------------- |
| Preserve current lock order and resolve targets first      | Small change; retains existing fencing | Tentative claim writes require rollback evidence                          |
| Move all Service locks ahead of Deploy/CMDB                | Earlier owner validation               | Requires coordinated changes across execution and completion paths        |
| Nonlocking READ COMMITTED reference check under owner lock | Avoids task/Service lock cycle         | Depends on every admission path participating in the owner-lock handshake |

**Stop condition:** fix the group inversion, implement deletion protection, and obtain fresh real-MySQL admission, rollback, concurrency, and callback evidence. C01 remains open until then.
