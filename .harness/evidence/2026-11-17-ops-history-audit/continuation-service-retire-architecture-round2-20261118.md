## Summary

**Architectural Status: CLEAR for the narrow Service Retire admission scope; whole C01 remains WATCH.** The initial HIGH isolation defect is resolved, and fresh real-MySQL CGO race verification passed against unchanged source.

## Analysis

- **Initial HIGH finding, now closed:** the previous candidate reused any caller `gorm.TxCommitter` without proving isolation (`service-retire-admission-candidates-20261118/first-green/backend/modules/business/service/state.go.raw:66`). The preserved RED demonstrates a separately committed executing task was missed and the caller’s instance became retired (`continuation-retire-caller-snapshot-red-20261118-01.log:2`).

- **Isolation now has a trustworthy origin:** the helper begins SQL READ COMMITTED, wraps that actual transaction in a private pool, and records its exact pointer (`backend/modules/business/capability/transaction.go:135`). Verification requires that private wrapper and matching context proof (`transaction.go:74`). Arbitrary caller transactions, copied contexts on another pool, and lost proof fail closed. The unsafe decorator and structural-equality approaches were removed.

- **Replay and caller ownership are preserved:** replay returns before the actual-mutation proof requirement (`backend/modules/business/service/state.go:89`); actual Retire checks proof before references and updates (`state.go:110`). Context rebinding carries only verified source proof (`backend/modules/business/service/reader_transaction.go:34`). Nested helper calls use savepoints without independent commit or new hooks (`backend/modules/business/capability/transaction.go:118`).

- **Reference semantics and lock order remain intact:** the bound reader requires proof, then performs a nonlocking executing-status query without caller DataScope filtering or task exclusion (`backend/modules/business/deploy/service_reference.go:31`). Completion writes its own terminal task status before invoking Service, allowing the same transaction to observe that write (`backend/modules/business/deploy/attempt_ownership.go:660`, `:678`). No Deploy row lock was added after Service locks.

- **Adjacent compatibility paths are covered:** deletion and instance target updates use the same helper (`backend/modules/business/service/deployment_reference.go:66`, `:93`; `backend/modules/business/service/service.go:517`). Deploy ownership checks and SQL error observation use helper hooks (`backend/modules/business/deploy/attempt_ownership.go:190`; `backend/modules/business/capability/transaction.go:36`, `:53`).

## Root Cause

The earlier boundary proved transaction existence but did not prove snapshot freshness. Current Service row locks cannot refresh an established RR snapshot used by a subsequent nonlocking Deploy reference read.

Session isolation inspection would not establish the required guarantee: the pinned MySQL driver executes one-shot `SET TRANSACTION ISOLATION LEVEL` before BEGIN (`D:/cache/go/pkg/mod/github.com/go-sql-driver/mysql@v1.10.0/connection.go:620`).

## Validation

Fresh core and compatibility verification passed:

- Receipt: `continuation-retire-isolation-core-compat-green-20261118-02.log.receipt.json:5883` records exit 0; `:2943` records unchanged source; `:5897` and `:5900` record disposable MySQL and CGO.
- The log records RC freshness PASS (`:4`), caller RR rejection PASS (`:782`), self/foreign completion PASS (`:807`), and deterministic Start/Retire competition PASS (`:814`).
- Capability, Service, and Deploy passed (`:25`, `:328`, `:953`). All ten reviewed candidate files still match the receipt hashes.

Evidence filenames above are relative to `.harness/evidence/2026-11-17-ops-history-audit/`.

## Recommendations

1. **Retain the helper-only mutation contract — low effort, high impact.** Caller-bound actual Retire must use the verified RC helper; plain default/RR/RC transactions remain intentionally unsupported.
2. **Retain the narrow verdict — no additional implementation required.** Full-package verification, broader C01 recovery/runtime acceptance, and aggregate closure remain separate gates.

## Architectural Status

**CLEAR — narrow Retire admission.**  
**WATCH — whole C01/package acceptance.**

No unresolved architectural blocker remains within this review’s scope.

## Trade-offs

| Option                             | Benefit                                                              | Cost                                                                           |
| ---------------------------------- | -------------------------------------------------------------------- | ------------------------------------------------------------------------------ |
| Verified RC boundary, selected     | Fresh references, own terminal-write visibility, existing lock order | Callers must use the helper                                                    |
| Deploy locking reads/NOWAIT        | Current-row reads                                                    | Adds task-lock ordering and contention concerns; changes the accepted contract |
| Arbitrary caller transaction reuse | Broad caller compatibility                                           | Cannot guarantee nonlocking reference freshness                                |

## Machine Readable

```json
{
  "lane": "architect",
  "task": "continuation-service-retire-admission",
  "architecturalStatus": "CLEAR",
  "scope": "Narrow Service Retire admission and verified RC transaction integration",
  "wholeC01Status": "WATCH",
  "initialFindings": [
    {
      "severity": "HIGH",
      "issue": "Caller RR snapshot could miss a separately committed executing deployment reference",
      "status": "RESOLVED"
    }
  ],
  "unresolvedArchitecturalBlockers": [],
  "validation": {
    "receipt": ".harness/evidence/2026-11-17-ops-history-audit/continuation-retire-isolation-core-compat-green-20261118-02.log.receipt.json",
    "exitCode": 0,
    "realDisposableMySQL": true,
    "cgoRace": true,
    "sourceUnchanged": true,
    "reviewedCandidateHashMatches": 10
  },
  "compatibilityDecision": "Actual Retire requires helper-proven RC; arbitrary caller transactions fail closed; no-mutation replay remains compatible",
  "acceptanceLimit": "Does not establish full-package, whole-C01, runtime, visual, or aggregate completion",
  "reviewMode": "Independent read-only architecture lane",
  "filesModifiedByReviewer": []
}
```
