# E01 backend lifecycle review amendment — 2026-11-18

Parent: `2026-11-16-ops-filedist-product-closure`. Implementer: `/root/e01_transfer_lifecycle_executor`.
Plan: `../../tasks/2026-11-16-ops-filedist-product-closure/continuation-backend-claim-failure-plan-20261118.md`.
Historical receipt preserved: `backend-lifecycle-handoff-20261118.md` and `.json`. Their old candidate hashes and pending-review status remain historical evidence; this amendment supplies the revised candidate. No independent approval or E01 package closure is claimed.

## Grounded failures before repair

`go test -mod=readonly -run '^TestDistributionClaim' -count=1 ./modules/business/filedist` exited 1 before production repair. Real GORM transactions through a standard-library SQL test connector injected parent-read, target-update, commit-applied and commit-rolled-back failures for Pending and Retrying targets. All eight subcases reported `terminal_attempts=0` with targets stranded in Pending/Retrying/Transferring. Failed terminal-persistence retry was also RED. The terminal-cancel preservation baseline passed.

`go test -mod=readonly -run '^TestDistributionDispatcherReports' -count=1 ./modules/business/filedist` exited 1 before scheduler logging repair: `discovery failure/recovery was silent` after two seconds. The test injects a real dispatcher SELECT failure and requires structured ERROR followed by INFO recovery with a subsystem and sanitized failure type.

## Revised behavior

- `claimTarget` returns `(claimed, error)`. `(false, nil)` remains a benign stale/terminal/duplicate result. SQL failures are distinguished, logged with type and IDs, and settled before local file lookup or remote IO.
- Claim-error settlement holds the consumed item active until a guarded unfinished-target failure write succeeds or the lifecycle is cancelled. Failed writes are observable and retry through five-second context-bound database calls and one-second cancellable waits. A cancelled/finished target cannot be resurrected. Commit ambiguity is settled as failure because no remote upload has begun.
- The dispatcher keeps its existing one-second retry ticker. A discovery failure emits a structured ERROR transition; the next successful discovery emits an INFO recovery transition. Raw SQL errors, query contents and credentials are omitted.
- Only the six newly introduced keys were remapped: `business.filedist.task.executionInterrupted`, `business.filedist.task.transferFailed`, `business.filedist.task.executionBusy`, `business.filedist.task.notDue`, `business.filedist.task.notCancellable`, `business.filedist.task.invalidContext`. Root owns canonical docs/translations.

## Fresh GREEN evidence

All commands below ran in `backend/`, except the repository-root diff/key checks. No MySQL server, live SSH provider or full race suite was invoked.

| Command                                                                                                                                                                                                       | Result                                                                                                                                                                                                                       |
| ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `go test -mod=readonly -run '^TestDistributionClaim' -count=1 ./modules/business/filedist`                                                                                                                    | Exit 0; `ok ... 1.040s` immediately after claim repair.                                                                                                                                                                      |
| `go test -mod=readonly -run '^TestDistribution(Claim\|DispatcherReports\|ActiveCancellation\|ContextReader\|SFTPHandshake\|LeaseLoss)' -count=1 ./modules/business/filedist`                                  | Exit 0; revised combined pure-test candidate passed in `2.041s`. Includes transaction outcomes, terminal cancel, transient terminal write, shutdown interruption, stale terminal queue items and discovery failure/recovery. |
| `go vet -mod=readonly ./modules/business/filedist`                                                                                                                                                            | Exit 0; no diagnostics.                                                                                                                                                                                                      |
| `git diff --check -- backend/modules/business/filedist`                                                                                                                                                       | Exit 0; no whitespace diagnostics.                                                                                                                                                                                           |
| `rg -n 'filedist.execution_interrupted\|filedist.transfer_failed\|filedist.execution_busy\|filedist.task_not_due\|filedist.task_not_cancellable\|filedist.invalid_context' backend/modules/business/filedist` | Exit 1 with no output: none of these six forbidden newly introduced short keys remain in the owned package. Historical receipts retain their original text.                                                                  |

The adjacent JSON records nine revised source SHA256 values. Startup integration, actual MySQL/Windows CGO race, independent rereview and provider/network-partition/product acceptance remain coordinator-owned. The original cancellation/remote rollback and advisory-lease evidence boundaries still apply.
