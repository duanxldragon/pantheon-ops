## Summary

**Architectural status: BLOCK.** The three round-2 defects are repaired in the inspected source: retry identity now has its own execution generation, resume checks the frozen owner binding transactionally, and superseded cancellation can settle Deploy while preserving newer Service state. One remaining failure-handling edge allows that cancellation shortcut to bypass an unavailable Service state transaction factory.

The backend response-to-frontend manual token path is complete by source inspection. Full race and formal-lock rebuild/source binding remain outstanding; this review does not approve C01 closeout.

## Analysis

### 1. BLOCK — Superseded cancellation bypasses an unavailable state transaction factory

Transaction composition deliberately substitutes `unboundServiceInstanceState{}` when the configured state factory is missing or returns nil/typed nil (`backend/modules/business/deploy/target_selection.go:192`, `:196`, `:198`). That substitute is nonnil and only rejects when its command is invoked (`:228`).

The new cancellation path can avoid that invocation:

1. `serviceInstanceForCallback` checks only whether the state capability is nil (`backend/modules/business/deploy/service_link.go:127`).
2. A valid snapshot with a newer current lifecycle/transition returns `errServiceExecutionSuperseded` (`:145`).
3. Canceled completion returns success immediately for that sentinel (`backend/modules/business/deploy/deploy_service.go:2498`).
4. Consequently, the known unbound state capability never produces its transaction refusal.

**Evidence boundary:** This is a statically grounded configuration-failure path, not a claimed dynamic reproduction. The inspected negative cancellation test covers missing snapshot, missing reader and an applier returning a separate `state_stale` error, but does not combine superseded owner state with an unavailable transaction factory (`backend/modules/business/deploy/service_link_control_test.go:363`).

**Smallest correct fix:** Reject the known unbound transaction-state capability before classifying an execution as superseded. Do not invoke a state mutation merely to probe availability, and do not reject lifecycle version zero: the Service model permits zero (`backend/modules/business/service/model.go:111`).

Add superseded-owner cancellation cases for missing factory, nil binding and typed-nil binding. Require refusal and unchanged task/hosts/leases/owner facts.

### 2. Round-2 retry identity defect is repaired

Start initializes `ExecutionGeneration` to one for linked execution (`backend/modules/business/deploy/deploy_service.go:1361`). Retry rejects zero/overflow generation, increments it and persists the replacement snapshot inside the common task-control transaction (`:1964`, `:1972`, `:1983`; task locking at `:1900`, `:1906`).

The manual execution digest includes this generation (`backend/modules/business/deploy/service_link.go:33`). It therefore rotates even when a retire action leaves Service lifecycle unchanged. The new test verifies unchanged owner lifecycle, old-token rejection, fresh completion and terminal replay (`backend/modules/business/deploy/service_link_control_test.go:170`, `:197`, `:201`, `:207`, `:212`).

### 3. Round-2 resume defect is repaired

Direct resume now uses `taskControlTransaction`, and dispatch occurs after that transaction returns successfully (`backend/modules/business/deploy/deploy_service.go:1825`, `:1832`). Its inner command validates current admission and the frozen owner binding before changing task status (`:1846`, `:1850`, `:1855`).

Versioned controls already run through the transaction composition seam (`backend/modules/business/deploy/task_command.go:211`, `:229`, `:261`). The new direct-resume regression verifies that a superseded owner rejects resume without changing persisted facts (`backend/modules/business/deploy/service_link_control_test.go:220`).

### 4. Superseded cancellation semantics are sound with valid capabilities

The sentinel is emitted after association, host snapshot, Service-state snapshot identity and execution-generation checks (`backend/modules/business/deploy/service_link.go:134`, `:139`, `:142`). Malformed/missing structural bindings return `serviceLinkInvalid`; ordinary success/failure completion continues to reject superseded state (`backend/modules/business/deploy/deploy_service.go:2498`, `:2501`).

Direct cancellation settles hosts, finishes the linked execution and releases leases inside the control transaction (`backend/modules/business/deploy/deploy_service.go:1742`, `:1754`, `:1763`). Owned completion and recovery use the same terminal recomputation (`backend/modules/business/deploy/attempt_ownership.go:903`, `:1009`), which excludes live attempts before settling and releasing this task’s leases (`:662`, `:686`, `:690`, `:717`).

The positive regression covers manual, unclaimed recovery and owned cancellation, checks owner preservation and checks recovery replay (`backend/modules/business/deploy/service_link_control_test.go:238`, `:264`, `:268`, `:275`, `:276`, `:280`).

### 5. Manual token propagation is complete; rendering remains unverified

Backend task responses populate host execution tokens (`backend/modules/business/deploy/deploy_service.go:4524`). Report and replay responses also carry the token (`:2332`, `:2360`). Frontend host and payload types expose it, and the API forwards the payload (`frontend/src/modules/business/deploy/api.ts:140`, `:276`, `:483`).

Both success and failure actions share `submitResult`, which submits the captured host token (`frontend/src/modules/business/deploy/task/DeployTaskDetail.tsx:295`, `:1074`, `:1093`). The failed-result modal captures the host row when opened (`:1055`), preserving the reviewed execution identity.

**Non-blocking context boundary:** The production manual-result handler still calls the root service without passing the HTTP request context (`backend/modules/business/deploy/deploy_handler.go:587`). Transactions retain their operation timeout, but HTTP disconnect cancellation is not established for this endpoint.

`impeccable` was applied to this interaction-contract inspection. No desktop/mobile rendering was produced: the available preview binary is explicitly historical, while current C01 source is verified from the worktree (`.harness/evidence/2026-11-17-ops-history-audit/continuation-c01-runtime-recovery-finalized.json:7`). No visual-polish verdict is given.

## Root Cause

The cancellation repair correctly separates result publication from cleanup. Its early return also skips the refusal mechanism of a known unusable transaction-state capability. Capability availability must be established before that shortcut.

## Recommendations

1. **High priority, low effort:** reject unbound transaction state before superseded classification; add the combined failure cases.
2. **Low effort:** retain the manual HTTP-context limitation explicitly, or propagate request context if disconnect cancellation belongs to C01 acceptance.
3. Refresh required verification and independent reviews after the final change.

## Architectural Status

`BLOCK`

## Trade-offs

| Approach                                                                | Benefit                                                 | Cost                                              |
| ----------------------------------------------------------------------- | ------------------------------------------------------- | ------------------------------------------------- |
| Skip state publication for a specifically superseded canceled execution | Preserves newer owner intent and releases old execution | Must verify required capabilities before shortcut |
| Reject all stale cancellation                                           | Simple refusal rule                                     | Reintroduces stranded references                  |
| Freeze execution generation independently                               | Handles actions without lifecycle mutation              | Adds one field to the existing snapshot           |

## Validation and Stop Condition

The fresh targeted run used disposable MySQL and CGO race and exited zero (`.harness/evidence/2026-11-17-ops-history-audit/continuation-c01-review-fixes-race.log.receipt.json:4`, `:8`, `:10`). The log records passing retire, resume and three-mode superseded cancellation regressions, with Deploy completing in 74.085 seconds (`continuation-c01-review-fixes-race.log:67`, `:72`, `:86`, `:229`).

That receipt does not demonstrate the combined superseded/unavailable-factory case. Full race and official-lock rebuild/source binding still require refresh (`docs/harness/tasks/2026-11-16-ops-service-deploy-integrity.task.md:174`).

**Stop condition:** close the remaining factory-refusal edge, verify it dynamically, then refresh the required final evidence. No C01 closure approval is given.
