# Findings-First Self-Review

Task: `ops-deploy-immutability-idempotency`.

## Findings

1. **Idempotent Start claim and lease acquisition are in one transaction.**
   The conditional `WHERE status IN (draft,pending)` update and per-host lease
   acquisition commit together, so a loser can never observe a running task
   without its leases. Resolved: correct by construction.

2. **Empty idempotency key still claims atomically but cannot replay.**
   A legacy caller that omits `idempotencyKey` gets the single-winner guarantee;
   a second start returns `alreadyRunning` (no replay). This is the documented
   fallback and is covered by `TestDeployTaskStartDifferentKeyWhileRunningConflict`.

3. **Execution snapshot is rebuilt on draft/pending update.**
   `persistTaskUpdate` re-runs `buildDeployExecutionSnapshot`, so a pre-Start edit
   never leaves a stale snapshot. Once running/terminal, UpdateTask is already
   rejected by state checks.

4. **Start no longer reads live package/template rows.**
   `resolveTaskExecutionPlan` now consumes only `execution_snapshot`. Verified by
   `TestDeployTaskExecutionSnapshotFrozenAgainstLiveMutation` (live mutation does
   not leak into the rendered SSH script).

5. **Callback CAS writes CMDB only on the winning transition.**
   The `WHERE status IN (pending,running)` update gates the CMDB writeback, so a
   duplicate/stale report cannot double-write host state. Verified by
   `TestDeployTaskHostReportIdempotentAndStaleConflict`.

6. **Lease release uses owner match.**
   `releaseHostLease(db, hostID, owner)` deletes only rows matching the task owner,
   so a takeover cannot release a newer owner's lease.

7. **Deploy test suite latent failures were unmasked.**
   Running against real MySQL exposed four stale assertions and four `dept_id`
   fixture gaps (the latter from the in-progress BizScope child). All are fixed
   here so the module is green; the bizscope/cmdb suite failures remain owned by
   `ops-bizscope-datascope-ownership`.

## Resolution

- Findings 1–6: implemented and covered by focused tests in
  `deploy_reliability_test.go`.
- Finding 7: deploy test file corrected; sibling-module failures documented as
  residual risk, not this task's scope.

## Remaining gap

- `go test -race` unproven in this environment (Cygwin GCC vs MinGW CGO). The
  conditional-update and unique-key semantics are exercised, but the race
  detector gate remains open until a MinGW toolchain is available.
