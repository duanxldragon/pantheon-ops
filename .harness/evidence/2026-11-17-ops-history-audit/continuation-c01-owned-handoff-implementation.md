# C01 owned pause / Resume worker handoff

Task: `2026-11-16-ops-service-deploy-integrity`.
Parent evidence: `2026-11-17-ops-history-audit`.
Status: implementation checkpoint; fresh source-bound green and independent review pending. No completion or archive approval.

## Scope

Only the Deploy business worker reservation and drain handoff changed. No persisted state, attempt ownership contract, lifecycle begin-state, retry budget, physical lease mutation or C03 contract was added.

- `backend/modules/business/deploy/deploy_service.go`: a task-local mutex protects active/retired and pending dispatch; stale retired pointers retry the map. One-shot cleanup cancels the old context and removes its cancellation before identity-deleting the worker slot. Hosts drain before retirement. Only a captured dispatch request triggers a fenced current-intent read, followed by dispatch after commit.
- The read requires running SSH with unfinished hosts, preserves live attempts and foreign physical leases, and uses existing CMDB then transaction-bound Service capability reads to reject a superseded frozen snapshot. Database errors remain observable.
- The canceled old host retry loop returns before any further claim, leaving resumed execution to the successor.
- `cancel_unclaimed_test.go`: the successor marker is a valid worker-slot object; its identity must survive old deferred cleanup.
- `worker_handoff_test.go`: actual SQL commit/reply separation selects the exact paused attempt UPDATE by status and WHERE id arguments, rather than transaction count.

## Verification evidence

Root executes tests against disposable real MySQL with CGO enabled, `-race -count=1`, and `GOFLAGS=-p=1 -mod=readonly`. This executor does not run tests or own runtime processes.

- `continuation-c01-owned-resume-handoff-red.log` and receipt: exit 1, 7.158 s package duration. Resume committed but the replacement never entered without recovery.
- `continuation-c01-owned-resume-handoff-first-fixed.log`: retained exit 1. Source changed during the initial build window, so it cannot bind the current source. Positive tests show the old worker making canceled attempts 2 and 3 before replacement attempt 4. A current-source retry guard prevents those extra claims.
- The primitive retirement fixture initially stored an int map key and deleted a uint64 key. The fixture now uses an explicitly typed uint64 task ID, preserving the cancellation deletion assertion.
- The failed positive test cleanup logged MySQL 1213. `continuation-c01-handoff-innodb-deadlock.log` has no usable latest-deadlock cycle. This does not establish a new lock-order defect; no database retry was introduced.
- `continuation-c01-owned-resume-handoff-guard-fixed.log` and receipt: fresh narrow run exits 0, 6.603 s. Both positive owned cases and both retirement orders pass, confirming the canceled retry guard and typed fixture key. The asynchronous positive case still logs MySQL 1213 during final worker settlement, so this is not a clean closure gate.
- gofmt and scoped `git diff --check` pass. A clean scoped gate is pending.

The test currently waits for map-slot deletion. Production deletes that slot before its final `recomputeTaskStatus` returns. Test cleanup can consequently reach `testmysql` connection close / database DROP while the successor is still settling. A metadata-lock interaction is a source-supported hypothesis, not a captured wait-cycle proof; no InnoDB latest-deadlock cycle is available. Root is reviewing a narrow worker-completion signal / deterministic settlement barrier before edits.

## Required regression set

`^TestWorkerHandoff|^TestCancellationUnclaimed|^TestControlTransitionsRecordEvents$|^TestDeployTaskRoutesServiceInstanceStateThroughTypedCommand$|^TestServiceLinkControlSupersededCancellationRemainsLive$`

Worker handoff includes synchronous duplicate dispatch and the actual asynchronous Resume API, cancel, Service supersession, foreign physical lease, foreign live attempt, retirement orders, Manual Resume, and database-read failure.

Positive owned tests hold the actual replacement remote runner until old worker drain/defer completes, then assert the paused attempt, incremented attempt number, and live successor worker/cancellation identities.

## Evidence boundaries

- Retirement-order tests exercise the real reservation/release primitives in both orders. They preserve a stale pointer across retirement and verify rejection followed by fresh reservation. They do not freeze the LoadOrStore-to-mutex instruction interval with a production test hook.
- The map preserves process-local dispatch requests. Persisted lease fencing and existing queue recovery continue to provide cross-process execution protection.
- This checkpoint does not close the full C01 task. Complete race, rebuild/source binding, runtime acceptance and independent review remain with root.
