# C01 queued Retire rejection candidate

Parent: `2026-11-17-ops-history-audit`, C01 / G003. This snapshot is the minimal queued-rejection production candidate; database GREEN, full-package compatibility, complete race and independent review remain Root-owned gates.

## Change

`runAsyncTask` logs rejected execution intent, invokes the existing `failQueuedTask` fenced transaction, and logs any settlement error with its original rejection reason. `failQueuedTask` returns its transaction error instead of discarding it. Both existing callers handle settlement errors. The transaction's lock order, host/lease settlement, running-attempt protection and owner failure behavior are unchanged; no local-only/skip-owner mode was introduced.

Eight queued regressions cover real `ReconcileDeployQueue` dispatch of legacy Retire, unknown-kind Install and metadata with SSH; direct dispatch of Manual metadata; an existing running attempt; stale/missing owner snapshots; and a fault at the real transaction-bound Service query entry. Query gates and the actual worker slot's `done` channel join worker exit without sleeps. Unknown-kind Install retains its existing real failure owner transition. Retire failure with valid owner state has no Service Apply or CMDB write; failed owner binding/query rolls back and remains observable in logs.

## Verification and preservation

- Original queued RED: [log](../../continuation-c01-retire-queued-rejection-red-20261118.log), [receipt](../../continuation-c01-retire-queued-rejection-red-20261118.log.receipt.json); exit 1, all eight subcases failed, 9.408s, 730 backend source files stable. The RED's absent fault-query hit records the rejected branch's failure to attempt settlement.
- Original queued RED test source: [raw](../queued-red/retire_lifecycle_test.go.raw), SHA-256 `04e9bb5ea52d96a33eccdbda1c00133c53c8697ccd2259334905e9be86f75a24`.
- Candidate compilation: `go test -run '^$' ./modules/business/deploy` from `backend` returned exit 0 with `[no tests to run]` (0.033s). This compiles the candidate and tests without running database regressions.
- Static checks: `go vet ./modules/business/deploy` returned exit 0; `gofmt -d` on both changed files returned no diff.
- Seven candidate source files and their raw bytes are frozen in [source-manifest.json](./source-manifest.json). The initial `green-candidate` directory and prior review/RED records were preserved.

No aggregate goal, task status, archive count, release, current-binary acceptance or production acceptance was changed by this candidate.
