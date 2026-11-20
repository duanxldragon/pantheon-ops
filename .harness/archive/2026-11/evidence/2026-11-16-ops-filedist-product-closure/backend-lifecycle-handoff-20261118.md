# E01 backend lifecycle handoff — 2026-11-18

Parent task: `2026-11-16-ops-filedist-product-closure`.
Implementation plan: `../../tasks/2026-11-16-ops-filedist-product-closure/continuation-backend-lifecycle-plan-20261118.md`.
Implementer: `/root/e01_transfer_lifecycle_executor`; independent review and package closure remain with the coordinator. Source is frozen for coordinated verification, not approved or shipped.

## Delivered behavior

- Cancellation locks the persistent parent and terminates its unfinished targets transactionally, then interrupts only that parent's active transport/retry contexts. Cancel replay is idempotent; completed/failed parents cannot be overwritten by cancel. Target updates and monitor callbacks preserve terminal cancellation.
- Stable target/file/parent identity accompanies every queue entry. The parent must still be running before a pending/retrying target can be claimed. Existing live stable-ID authorization remains ahead of local file access.
- SFTP dialing, SSH/SFTP setup, upload and retry waits observe cancellation. The owned transport is closed to interrupt blocked remote requests. File-path database lookup inherits the operation context; terminal target writes use a five-second database timeout.
- Scheduled tasks are discovered in batches of at most 100 every second; both manual start and persistent execution claim reject future schedules. Duplicate dispatch loses the existing pending-to-running compare-and-swap.
- Startup acquires a database-scoped MySQL advisory lock on a dedicated pooled connection before recovery. Interrupted running work becomes failed, completed target history and aggregate counts are preserved, and ambiguous remote writes are never automatically replayed. An independent one-second ownership monitor cancels execution after a failed lease check; each check has a five-second timeout. Stop cancels the dispatcher/lease monitor and owned transfer engine before releasing the lock.

## Fresh local verification

Commands ran in `backend/` unless otherwise stated. All listed commands exited 0.

| Command                                                                                                                                                                                                                                     | Observed result                                                                                                                   |
| ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------- |
| `gofmt -w modules/business/filedist/{distribution_lifecycle.go,distribution_lifecycle_test.go,distribution_service.go,transfer_engine.go,sftp_client.go,storage_service.go,distribution_model.go,module.go}` (explicit paths in PowerShell) | No output; formatter completed.                                                                                                   |
| `go test -mod=readonly -run '^$' ./modules/business/filedist`                                                                                                                                                                               | Package compiles; `ok ... 0.042s [no tests to run]`.                                                                              |
| `go test -mod=readonly -run '^TestDistribution(ActiveCancellation\|ContextReader\|SFTPHandshake\|LeaseLoss)' -count=1 ./modules/business/filedist`                                                                                          | Four non-database cases passed; latest run `ok ... 0.044s`. The SSH fixture is a local loopback server that never sends a banner. |
| `go vet -mod=readonly ./modules/business/filedist`                                                                                                                                                                                          | No diagnostics.                                                                                                                   |
| `git diff --check -- backend/modules/business/filedist .harness/tasks/2026-11-16-ops-filedist-product-closure/continuation-backend-lifecycle-plan-20261118.md` (repository root)                                                            | No whitespace diagnostics.                                                                                                        |

Exact SHA256 values are recorded in `backend-lifecycle-handoff-20261118.json`. Unrelated pre-existing/shared-worktree changes, including `seed.go`, are outside this implementer snapshot.

## Required coordinator integration

1. Call `FileDistModule.Start(ctx)` after migrations and provider/authorizer wiring, before exposing HTTP; propagate errors and keep module Stop in shutdown cleanup. `InitFileDistModule` no longer starts workers. The database pool needs capacity for the held lifecycle connection plus ordinary queries. Concurrent lifecycle owners for the same database fail with `filedist.execution_busy`.
2. Add canonical error/i18n integration for `filedist.execution_interrupted`, `filedist.transfer_failed`, `filedist.execution_busy`, `filedist.task_not_due`, `filedist.task_not_cancellable` and `filedist.invalid_context`.
3. Run actual MySQL tests: `TestDistributionCancelStopsInFlightAndPreservesTerminalState`, `TestDistributionDueDispatchDoesNotStartFutureOrDuplicateTasks`, `TestDistributionRecoveryFailsInterruptedWorkWithoutReplaying`, `TestDistributionLifecycleLeasePreventsConcurrentRecovery`, `TestDistributionLifecycleLeaseHealthDetectsReleasedOwnership`.
4. Preserve compatibility coverage, especially `TestTransferErrorWithoutRetriesFinishesTarget`, `TestQueuedDistributionRevocationPrecedesFileAccess` and the existing authorized distribution start/revocation cases. The authorization check precedes the manual due-time check.
5. Run the coordinated CGO race gate, integration/runtime checks and independent review. This implementer did not run MySQL, full race, live provider or product smoke.

## Explicit evidence boundaries

- Cancellation interrupts future IO but cannot undo bytes already accepted by a remote host. Neither cancel nor startup recovery claims remote atomic rollback.
- The advisory lock and independent health monitor provide exclusive ownership while the dedicated database connection remains held, with bounded detection when it is lost. They are not a fencing token accepted by remote SSH hosts; a database/network partition can leave a short overlap before detection.
- Stop joins dispatcher/lease-monitor/engine workers. Progress-monitor contexts are cancelled and their database writes are context-bound; this change does not introduce a separate join barrier for every progress goroutine. Driver cancellation and provider behavior still need actual runtime/race evidence.
- No shared locale, startup composition, DTO/schema, Base, CMDB or frontend changes were made by this implementer. This handoff does not close E01 or accept its human gate.
