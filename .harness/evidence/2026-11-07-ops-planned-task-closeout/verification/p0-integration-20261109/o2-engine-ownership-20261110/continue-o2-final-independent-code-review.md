# Independent O2 final implementation CODE review

**REQUEST CHANGES.** Files reviewed: the ten exact deploy files in `source-delta-final.json`, with narrow owner/capability context. Issues: CRITICAL 0, HIGH 2, MEDIUM 0, LOW 0. L2 parent: `2026-11-07-ops-planned-task-closeout`.

## Stage 1: specification compliance

V2 plus V3 and their independent design reviews are requirements, not implementation approval. Latest attempt/worker/physical ownership fences, DB equality/null rejection, atomic dual renewal, independent watchdog, token propagation into SSH and callbacks, void-writer cancellation cause, task-first serialization and outermost READ COMMITTED/bound CMDB projection are supported by source and focused regression evidence. Two completion/recovery requirements fail below. Stage 1 has not passed; no normal quality approval or style findings are issued.

## Findings

**[HIGH] Service-instance projection commits outside the owned finish transaction.** `backend/modules/business/deploy/attempt_ownership.go:875` calls `finishServiceInstanceState` before the fenced outer transaction commits. `target_selection.go:167` carries the unchanged applier into the transaction service; `deploy_service.go:2368` calls that unbound applier. The actual Service manager writes through its own DB (`service/state.go:75`). Manual reporting also calls it before commit (`deploy_service.go:2221`).

Reproduction: `TestIndependentO2ServiceProjectionRollsBackWithFinish` uses the actual Service manager and required isolated MySQL, then injects an outer commit failure. The original cause cancels ownership and all deploy/CMDB facts roll back, but the service instance commits `observed=stopped`, `currentVersion=1`, `lifecycleVersion=1`. This is a durable false completion, not an SSH exactly-once issue. The handoff's disclosure preserves evidence but does not satisfy the all-effects completion contract.

Fix: ask root/ARCH to authorize the smallest cross-domain change: an explicit transaction-binding capability implemented by the Service owner, modeled on existing CMDB/BizScope binding, and consumed by both finish entry points. Retain the same outer READ COMMITTED transaction and ownership fence, carry its context into owner reads/writes, reject an unbound configured applier, and keep this rollback regression. Do not implement a deploy-side Service-table override, move the write after commit, or swallow the error. This CODE finding is escalated for architecture authorization; it does not constitute an architecture verdict.

**[HIGH] Cancel before the first claim never settles and retains the physical lease.** `backend/modules/business/deploy/deploy_service.go:1456` ends the worker without the removed task-level cancellation/release finalization. Recovery at `deploy_service.go:1610` scans only pause requests, while `attempt_ownership.go:906` discovers only running attempts. A valid admitted SSH task canceled before its first claim has neither a completing owner nor an attempt for recovery to discover.

Reproduction: `TestIndependentO2CancelBeforeFirstClaimSettles` materializes the admission lease, invokes the real version-checked `ControlTask(cancel)`, and invokes another service's `ReconcileDeployQueue`. The task remains `cancel_requested` with one physical lease. It cannot reach its terminal cancellation event, and competing deployment is blocked until the thirty-minute lease expires.

Fix: add serialized task-first cancellation recovery for requested tasks even when no attempts exist. Under the full latest-attempt/physical ownership checks, settle only when no live owner remains; release this task's unused leases and recompute terminal state/event in the same transaction. Preserve live-runner exclusion and add the no-claim cancellation regression, rather than restoring unfenced post-worker writes.

## Root-cause and verification evidence

No masking fallback is accepted. The retained nil/void compatibility writers were inspected with their cause-latching ownership paths; swallowed primary failures are not offered as a remedy for either finding.

- Independent regressions: CGO=1, MinGW, REQUIRED MySQL `127.0.0.1:33316`, Go overlay only; **exit 1, two top-level invariant failures, zero skips**. Actual failure facts are in `continue-o2-final-independent-code-regressions.txt`.
- Concern-driven original race subset: **exit 0, 15 top-level passes, zero failures/skips**, 26.688s; covers both leases/contention, fresh stale callbacks, watchdog/blocked rollback, lifecycle, recovery/budgets/history, exact DB boundaries, RC entry points and SSH cancellation. See `continue-o2-final-independent-code-targeted-race.txt`.
- Native `gopls check` on all ten changed files: **exit 0**, no compile/type errors; retained unnecessary `fmt.Sprintf` suggestion at service line 4612. `git diff --check`: **exit 0**. The named LSP/AST tools are unavailable; native gopls and literal pattern/context review were used. No tool execution is fabricated.
- Final **46/46 source hashes and 10/10 delta hashes match before/after**, recorded in the new source manifests and `continue-o2-final-independent-code-receipt.json`. The earlier source-stable receipt's 157 passes, two explicit live-VM skips, and vet/build/native exit-zero evidence remain valid and were not mislabeled as independent reruns.

Only new prefixed evidence/test-overlay artifacts and task-owned caches were created. Tests use fresh generated fixture databases with exact-name cleanup. No source/other agent edits, remote access, author fix, full-tree race, release or waiver occurred. Review stops with two reproduced HIGH findings. Root retains VM/public HTTP/IAM/audit/browser/full-tree/release gates; O2/P0 remain open.
