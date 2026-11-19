# C01 independent architecture review — handoff round 7

Provenance: coordinator transcription, condensed from the independent read-only `/root/c01_handoff_architecture` report on 2026-11-18 Asia/Shanghai. No reviewer tests, source edits or runtime operations.

**Architecture WATCH; narrow successful handoff/final-settlement fix CLEAR.** No deterministic blocker was found within the narrow fix. Whole C01, merge-ready and archive approval remain false.

## Evidence and boundaries

Slot reservation and retirement linearize under one mutex (`deploy_service.go:1459`, `:1469`, `:1484`); both orderings and stale-pointer rejection are covered (`worker_handoff_test.go:420`). Once-only cleanup and pointer-identity deletion prevent old defers from removing the real successor (`deploy_service.go:1507`, `:1553`, `:1556`; tests `:226`, `:373`, `:379`, `:382`, `:385`). Old synthetic successor-marker evidence remains a narrower historical fixture, not another real worker.

Pending admission uses the existing fenced transaction, locking task, hosts, attempts and physical leases before checking frozen Service identity and lifecycle (`deploy_service.go:1572`, `:1589`, `:1594`, `:1600`, `:1615`; `attempt_ownership.go:271`, `:357`, `:378`; `target_selection.go:184`, `:192`; `service_link.go:152`, `:160`). Negative tests protect intent already present before the read; they do not prove all owner changes after commit.

The old-context pre-loop guard (`deploy_service.go:1621`) blocks retries after the paused commit reply when cancellation is already visible. It is not atomic with `claimOwnedAttempt`, whose API does not receive parent context (`attempt_ownership.go:582`, `:588`, `:731`, `:741`, `:768`). All Pause/Resume versus blocked-claim interleavings remain unproven.

`done` closes after final settlement, pending admission and deferred cleanup (`deploy_service.go:1504`, `:1510`, `:1556`, `:1559`, `:1564`). The real commit/reply barrier proves deleted slot, still-open done and blocked fixture cleanup until reply release (`worker_handoff_test.go:64`, `:123`, `:129`, `:133`, `:138`, `:145`). Done signals this worker's return, not settlement success or transitive completion of all successors; tests separately wait for the captured real successor (`:373`, `:389`, `:504`).

## WATCH items

1. Startup/reconcile/read errors or final recompute errors retire the slot without consuming pending (`deploy_service.go:1512`, `:1516`, `:1525`, `:1559`, `:1563`). Existing queue recovery scans running SSH tasks (`:1767`, `:1785`, `:1789`). Error recovery needs its own deterministic regression before claiming error-path handoff preservation.
2. Pre-loop cancellation versus blocked claim requires an actual transaction barrier test. Do not introduce durable generations without a demonstrated contract need.
3. Retire remains independent: a successful empty summary reaches CMDB installed-component upsert while Service completion becomes retired (`deploy_service.go:2569`, `:2571`, `:2755`; `service/state.go:265`). Host components are a compatibility projection (`BUSINESS_SERVICE_MODULE_DESIGN.md:151`). Do not equate Retire with Uninstall to close this gap.
4. Full backend race, official rebuild, final source binding and current functional runtime remain open.

Independently read GREEN: 18 top-level PASS, zero skips, 43.248s, exit 0, CGO/race/disposable MySQL; no 1213/ERROR/DATA RACE/FAIL matches. Current hashes were stable across review, but the original receipt lacks before/after build hashes. A later source-bound rerun is a new evidence checkpoint.

## Machine Readable

```json
{
  "taskId": "2026-11-16-ops-service-deploy-integrity",
  "parentEvidenceId": "2026-11-17-ops-history-audit",
  "reviewRound": 7,
  "agentRole": "architect",
  "reviewer": "/root/c01_handoff_architecture",
  "reviewMode": "independent-read-only",
  "architecturalStatus": "WATCH",
  "narrowFixArchitecturalStatus": "CLEAR",
  "mergeReady": false,
  "wholeTaskComplete": false,
  "archiveApproved": false,
  "testsRunByReviewer": false,
  "runtimeTouched": false,
  "validation": {
    "topLevelPassed": 18,
    "skipped": 0,
    "packageSeconds": 43.248,
    "exitCode": 0,
    "race": true,
    "cgoEnabled": "1",
    "disposableMySQL": true,
    "forbiddenLogMatches": 0,
    "sourceHashesStableAcrossReview": true,
    "buildSourceBindingIndependentlyEstablished": false
  },
  "openGaps": [
    "Failure-path pending relies on queue recovery",
    "Cancellation versus blocked claim not atomically proven",
    "Owner change windows after current-intent read",
    "Retire lifecycle and compatibility projection",
    "Full race, rebuild, source binding, current runtime"
  ]
}
```
