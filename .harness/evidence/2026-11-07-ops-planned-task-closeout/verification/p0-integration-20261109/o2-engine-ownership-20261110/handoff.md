# O2 Generator handoff — implementation and scoped verification

L2 parent: `2026-11-07-ops-planned-task-closeout`. Implementation follows `resume-o2-engine-ownership-plan-v2.md` and `-v3.md`. Independent design approval is retained; implementation CODE/ARCH review and O2/P0 acceptance remain open.

## Changes Made

Implemented attempt-local ownership throughout executor/SSH callbacks, persisted task-first serialization, latest-attempt/worker/physical-owner/live-DB-expiry fences, and atomic five-minute attempt/thirty-minute physical renewal. Production cadence is one minute, DB deadline five seconds, conservative watchdog margin ten seconds. The watchdog runs independently of DB I/O. First errors cancel execution, void log writes retain the cancellation cause, and late replies cannot restore ownership.

Finish, host/component projection, retry disposition, release and final task recomputation share the fenced transaction. A local lifecycle guard prevents a queued renewal from misclassifying our own committed finish as lost ownership; admission and cross-service exclusion remain persisted. Pure renewal never later locks task. Initial start/preview discovery is nonlocking; final exact-footprint validation follows physical lease locks. Every retained projection begins outermost READ COMMITTED and rejects unmarked existing transactions, using the existing transaction-bound CMDB capability.

Preserved cumulative three-chargeable-attempt budget, free paused attempts, crash recovery, successful checkpoints, append-only history, frozen definitions, credential checks and DataScope admission. SSH cancellation joins session/output and handshake cancellation. No shared source, CMDB implementation, frontend, manifest/status, dependency or migration edits.

Exact deploy file changes and old→new SHA256 values are in [source-delta-final.json](source-delta-final.json):

- New: `attempt_ownership.go`, `attempt_ownership_test.go`.
- Updated: `deploy_service.go`, `deploy_rollback.go`, `target_selection.go`, `task_preview.go`, `task_command.go`, `deploy_reliability_test.go`, `deploy_ssh_transport_test.go`, `resume_scope_revocation_test.go`.

New local verification helpers and their hashes: [verification-helper-hashes-final.json](verification-helper-hashes-final.json). Approved CMDB/model/history bindings remain unchanged: [retained-source-bindings-v2.json](retained-source-bindings-v2.json).

## Verification

Windows Go 1.26.10, `D:/msys64/mingw64/bin`, `CGO_ENABLED=1`, task-local `.local/o2-engine-gocache`, offline dependency resolution. Required MySQL: supplied local listener `127.0.0.1:33316`, `PANTHEON_TEST_DSN_REQUIRED=1`. Independent DeployService instances share fresh isolated fixtures.

- `go test -mod=readonly -race -count=1 -timeout=25m ./modules/business/deploy/... -v` → PASS, 271.113s; **157 top-level passes, two live-VM skips, zero ownership-test skips**.
- `go vet -mod=readonly ./modules/business/deploy/...` → exit 0.
- `go build -mod=readonly ./modules/business/deploy/...` → exit 0.
- `gopls check` on all ten changed deploy files → exit 0; no compile/type errors. One pre-existing unnecessary `fmt.Sprintf` suggestion at `deploy_service.go:4612` is retained.
- `gofmt`, `git diff --check`, local fixture-helper vet/run → clean.
- Required local Redis read-only RESP PING → `+PONG`; Redis readiness is evidenced, not claimed as the ownership coordinator.

[Source-stable focused receipt](stable-v2-focused-receipt.json): all **46 deploy files** unchanged throughout final race/vet/build/native diagnostics, `06:14:08–06:18:48 UTC`. Current hashes still match that receipt.

Regressions cover both shortened lease horizons, same/cross-task physical contention, revoked runners, deliberately fresh stale callbacks against every fenced table, blocked DB/watchdog/late replies, commit failure before blocked rollback, renewal/reconcile and finish/reconcile winners, finish/queued-renewal commit barriers, exact DB equality/null recovery, third-chargeable budget, pause/cancel/history, projection rollback and all RC entry points.

Meaningful RED artifacts: `red-snapshot-actual.txt` (RR loses B’s committed component), `red-discovery-lock-order.txt`, `red-deadline-historical-projection-actual.txt`, `preflight-stable-v2.txt` (finish/renew false cancellation). Corresponding final regressions pass. Preliminary compile failures/timeouts remain historical evidence and are not counted as passes.

Two exact fixtures left by earlier timeouts were cleaned; two unrelated historical databases remained intact. See `fixture-exact-cleanup.txt` and `fixture-inventory-after-cleanup.json`.

## Assumptions / Notes

`TestDeployLiveVMExecution` and `TestDeployLiveVMCancellation` skipped because live acceptance was not requested/configured. No remote/credential access was performed. Root retains VM/cold-restart acceptance, full-tree/source-stable race, integration/rebuild/frontend gates and independent implementation reviews.

Existing `ServiceInstanceStateApplier` has no transaction-binding capability. Its calls are retained under serialized deploy fences, but its existing separate commit cannot be claimed atomic with deploy/CMDB rollback. This limitation remains for independent review; it was not hidden by changing another module or waiving a gate. No exactly-once remote-effects or arbitrary remote-child termination guarantee is made.

## Summary

Scoped engine implementation and local verification are ready for Dispatcher review. O2/P0 have not been self-approved or closed. Gate Outcomes: MySQL regressions caught stale projections, expiry/command gaps and finish/renew cancellation; RC barrier caught RR lost updates.
