# Code Review Summary

**Recommendation: APPROVE — narrow C01 Service Retire admission and verified READ COMMITTED repair only.**

**Files reviewed:** 10 scoped Go files.  
**Open issues:** 0 — CRITICAL 0, HIGH 0, MEDIUM 0, LOW 0.  
**Initial findings closed:** 5.

This supersedes the initial REQUEST CHANGES for this scope. Whole C01 remains **WATCH**; this review does not authorize aggregate completion or archival.

## Spec compliance and root-cause assessment

Stage 1 passed against `continuation-service-retire-admission-plan.md` and its `continuation-retire-isolation-plan.md` refinement before final quality approval.

- Actual Retire mutations require helper-created RC proof after no-mutation replay and lifecycle validation (`service/state.go:111`). Unknown RR/default/plain-RC bound transactions reject explicitly.
- The helper begins an actual `sql.LevelReadCommitted` transaction and verifies exact private wrapper identity (`capability/transaction.go:74`, `:108`). Copied contexts, different pools, lost proof and canceled scopes cannot certify a transaction.
- Owner reads retain instance → service → application locking and current DataScope checks. Retire CAS uses the locked lifecycle version even when the caller omits an expected version (`service/state.go:139`).
- Bound reference checks use the same proven transaction, include all executing reference statuses and apply no caller DataScope filter (`deploy/service_reference.go:35`, `:47`). They acquire no Deploy row locks after Service locks.
- Completion writes the task terminal state before the owner callback (`deploy/attempt_ownership.go:660`, `:678`). Its own terminal write permits completion; another running or paused reference rejects and rolls back the transaction.
- Legal unknown/stopped/failed lifecycle, replay, software-version preservation, stale CAS, caller rollback and identifiable failures remain covered.

The repair fixes the controlling isolation contract. It introduces no task exclusion, alternate out-of-transaction query, swallowed failure, or permissive reader fallback. The private helper wrapper preserves existing Deploy commit/error hooks without allowing public pool substitution.

Review of `service.go` is limited to the UpdateInstance transaction migration and relevant owner-read context; review of `attempt_ownership.go` is limited to transaction/hooks and completion seams.

## Initial findings and closure

References below identify the reviewed revision where each finding arose.

| Severity | Initial finding                                                                                                                                    | Closure                                                                                                                         |
| -------- | -------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------- |
| HIGH     | `deploy/service_reference.go:39`: nonlocking Count could reuse an established caller RR snapshot and miss a separately committed Start.            | Explicit unknown-transaction rejection plus actual RC begin; preserved RR failure evidence and passing current-read regression. |
| HIGH     | `capability/transaction.go:31`: preliminary `reflect.DeepEqual` accepted structurally equal distinct pools as identity.                            | Exact private wrapper pointer comparison; copied-wrapper and different-transaction tests pass.                                  |
| MEDIUM   | `capability/transaction.go:87`, `:98`: preliminary fresh/nested transactions did not bind supplied context; nested entry skipped pre-cancellation. | Context binds before begin/savepoint; fresh/nested cancellation, cause and rollback tests pass.                                 |
| MEDIUM   | `capability/transaction.go:105`: preliminary public decorator could admit a typed-nil or substitute pool.                                          | Decorator removed; helper owns the wrapper. Typed-nil BEGIN rejection passes without callback or panic.                         |
| MEDIUM   | `deploy/attempt_ownership.go:5`: unused `database/sql` import blocked compilation after wrapper removal.                                           | Import removed; final diagnostics and core/compatibility race run pass.                                                         |

The typed-nil test’s initial run **passed**. Its filename containing `red` is not evidence of a failing run; no production nil patch was needed.

## Validation evidence

- Final `gopls check` covered all 10 scoped Go files: **exit 0, no diagnostics**. The requested `lsp_diagnostics` and `ast_grep_search` tools were unavailable; actual `gopls` diagnostics, source inspection and literal pattern scans were used.
- Scoped `git diff --check` passed. No hardcoded secrets, injection construction, suppressed failures or masking fallback was found in the reviewed changes.
- `continuation-retire-isolation-core-compat-green-20261118-02.log` and its receipt confirm **exit 0**, CGO enabled, `go test -race -count=1`, and disposable MySQL across capability/service/deploy. All **734 bound sources remained unchanged**; Deploy completed in 190.274 seconds.
- Tests cover RC freshness after a separate commit, proof identity, unknown transactions, nested/outer rollback, cancellation, commit fence and SQL-error hooks; Retire lifecycle, PATCH admission, reference failures, competing Start, self-terminal completion and foreign-reference rollback; and selected existing transaction/reference/ownership compatibility.
- Official rebuild, staged overlay and official-cache drift checks each passed in `continuation-rebuild-cache-retire-isolation-final-20261118-verification.json`.
- `continuation-retire-isolation-final-20261118-source-equivalence.json` reports **1,221 equivalent sources**. All 10 reviewed file hashes match that manifest and the current frozen files.

The original caller-snapshot failure and the first core run’s genuine unused-import build failure remain preserved.

## Explicit gaps and stop condition

PATCH handler tests register directly on Gin without real authentication/Casbin middleware. They prove handler-to-owner admission; source inspection verifies context/DataScope forwarding. Real limited-role authorization and current runtime/provider acceptance remain open.

Full backend `./...` CGO race was still running when this narrow verdict was issued. Broader queue recovery, host policy, other C/E packages, final maintainer acceptance and aggregate cleanup remain open. Architecture’s final verdict belongs to the separate architect lane.

This code/spec/security lane stops with no remaining findings in the frozen narrow scope. Any scoped source change invalidates this source-bound recommendation until re-reviewed.

## MachineReadable

```json
{
  "recommendation": "APPROVE",
  "scope": "C01 Service Retire admission and verified READ COMMITTED repair only",
  "wholeC01": "WATCH",
  "aggregateCompletionApproved": false,
  "filesReviewed": 10,
  "openIssues": { "CRITICAL": 0, "HIGH": 0, "MEDIUM": 0, "LOW": 0 },
  "initialFindingsClosed": 5,
  "specCompliance": "PASS",
  "rootCauseGuard": "PASS",
  "diagnostics": {
    "tool": "gopls check",
    "exitCode": 0,
    "errors": 0,
    "lspDiagnosticsToolAvailable": false,
    "astGrepToolAvailable": false
  },
  "coreCompatibilityRace": {
    "exitCode": 0,
    "cgoEnabled": true,
    "disposableMySQL": true,
    "sourceUnchanged": true,
    "boundSources": 734,
    "receipt": "continuation-retire-isolation-core-compat-green-20261118-02.log.receipt.json"
  },
  "officialRebuildOverlayDrift": "PASS",
  "sourceEquivalence": {
    "passed": true,
    "sources": 1221,
    "sourceSetSHA256": "1578bb543691c97bd4a605fc30978dc98d5971b830f1936d3d2c9e8c599206c7",
    "manifest": "continuation-retire-isolation-final-20261118-source-equivalence.json"
  },
  "fullBackendRace": "RUNNING_AT_VERDICT",
  "runtimeRoleProviderAcceptance": "OPEN",
  "architectureVerdictOwner": "separate architect lane",
  "reviewedSHA256": {
    "backend/modules/business/capability/transaction.go": "208adcf662c1c3b25046f675d9b7bd29c5acfc067c90f4d5b45a2b0f7b631fb7",
    "backend/modules/business/capability/transaction_test.go": "30871a5a1061f17b82fc1315e0851c0a47372ef1c530e72b0d9974184f38dc14",
    "backend/modules/business/service/state.go": "8ea4aaf71dd9140546b05d90175ab40d72dd7396409c5a6bf22342dafab88610",
    "backend/modules/business/service/reader_transaction.go": "90b202dd369148c8227b3bb1a8791d211557e67b9a073dcf0f0d564ff6a4b3b6",
    "backend/modules/business/service/deployment_reference.go": "ff0da116b1b750cb7eabf6afe90cbac920c63bbba00b48f16f8c3ed2ab3a1fb0",
    "backend/modules/business/service/service.go": "20fb15768099b7774016e81e6da53b15418c61db82938e6f633bd8fa053962b6",
    "backend/modules/business/service/retire_admission_test.go": "fe3fc9e9f9a99f486f0c90ccb91b4122a988804e926fe9ff0c70ea43194a2fc8",
    "backend/modules/business/deploy/attempt_ownership.go": "f92249ffa2bcf177a91131098b3786c05c51d60fc3e148bc3f255c00d65ee490",
    "backend/modules/business/deploy/service_reference.go": "6e64f2c42d09bf5293f85ac089e41cae35c9fcce37a2681e10c97024221face1",
    "backend/modules/business/deploy/service_retire_admission_test.go": "1f2db8a3374b3e9bcd5675510b302523440000cbc8ec9af5a21a83796f37d13c"
  }
}
```
