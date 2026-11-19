---
doc_type: independent-code-review
reviewer_lane: c03_code_review
recorded_at_utc: "2026-11-17T17:54:46.347Z"
verdict: REQUEST CHANGES
status: narrow-retire-green-candidate-review
files_reviewed: 7
critical: 0
high: 0
medium: 1
low: 0
whole_task: false
merge_ready: false
archive_approved: false
---

## Code Review Summary

**REQUEST CHANGES: one MEDIUM newly introduced async error-suppression branch blocks the narrow Retire approval.** The four Retire regression groups are GREEN, and the primary metadata/owner/projection contracts are implemented. Passing those groups does not cover queued legacy rejection, where the new guard hides its error.

### MEDIUM C01-RETIRE-ASYNC-01 — preserve async rejection and settle the queued task explicitly

**File:** backend/modules/business/deploy/deploy_service.go:1576-1577. The new validateRetireRemoteIntent failure immediately returns from runAsyncTask without error logging, a rejection event or queued-task settlement. Existing ReconcileDeployQueue at :1844-1851 still selects running SSH tasks and reports the dispatch count. The existing slot cleanup merely retires the in-memory worker (:1541-1567).

Minimal counterexample: a persisted pre-upgrade running/SSH legacy executable Retire, one pending/running task host, an owned physical lease and no running attempt. Queue recovery selects it and returns one dispatch. The guard always rejects task.Action=Retire, then releases the worker slot and closes done; the task stays running, the host lacks the rejection error and its lease remains until separate expiry/recovery. A running non-Retire SSH record with unknown snapshot kind follows the same error-loss path. This is a direct source trace, not a DB reproduction executed by this reviewer. Ordinary empty legacy snapshots are not falsely rejected: decodeExecutionSnapshot(:4394-4396) returns an empty value without error and the new validator permits an ordinary unmarked task.

**Fix:** retain the remote guard and repair its failure contract: explicitly reject through the fenced queued-settlement path, preserve the validation reason, and expose any settlement error. Reuse failQueuedTask where appropriate, but its existing discarded fencedTransaction result at :1770 must not make the newly added rejection silently disappear on failure. Add real queue-recovery regression coverage for legacy Retire and unknown kind, with no runner/command, observable rejection, appropriate terminal or explicit blocked state, worker-slot release and owned-lease handling. The test must also retain owner/CMDB-failure evidence. Broader recovery work may remain separately scoped; this new masking branch cannot be approved by declaring recovery out of scope.

The helper directly marks hosts failed, releases this task's leases and recomputes task status (:1783-1791); it does not call commitHostResult or WriteDeployHostResult. Failed Retire maps to ok=false at :2845-2847, so it does not apply a Retire owner transition or write software versions. Linked settlement still binds CMDB and reads/validates Service ownership (attempt_ownership.go:702-718 and deploy_service.go:2699); owner staleness, unbinding or SQL failures can roll back settlement. Therefore a call followed by ignored errors is insufficient. Ordinary Install failure can invoke its existing failed-owner transition; its policy should be tested separately from Retire's no-owner-mutation expectation.

### Spec and root-cause review

Review is based on pre-production raw -> green raw diffs, isolating this change from earlier C01 edits. deploy_model.go:326 adds snapshot Kind. Create validates Manual/linked Retire and explicit executable params (:506-508), loads enabled package identity without install-parameter requirements, and freezes metadata_only with a non-nil empty steps array (:4134-4141). Template Steps are retained only as reference identity, while explicit command/probe params remain rejected (:4145-4152). resolveTaskExecutionPlan validates kind/action/executor/linkage/params and rejects unknown kind, nil/null steps, polluted metadata and legacy executable Retire (:4119-4123/:4155-4170).

Update checks the final action after template defaults (:858-870), rejects both action-direction conversions and legacy Retire laundering, and rebuilds draft metadata only after validating its existing marker. The added template-default laundering regression preserves all persisted facts on rejection. Start/Preview/admission, worker claim, transport, step, checks, rendering and component-recording entry guards prevent execution. These guards fix the software-execution contract without a fallback execution path. The runAsyncTask error-loss branch above is the outstanding root-cause-guard failure.

Success summary rejection occurs under lock before host/projection/owner writes (:2466-2474). Metadata writeback skips the entire WriteDeployHostResult call (:2633-2637), rather than only suppressing component deltas. DesiredVersion becomes empty for Retire (:2713-2715), and the Retire state result carries no CurrentVersion; the real-owner test proves desired/current/rollback values unchanged. Failed Retire/retry leave the owner untouched. Stopped desired/observed admission and completion remain a conservative supported subset, not universal proof for unknown/failed. Service PATCH and wider host leases remain leader/architecture scope.

### Regression and compatibility evidence

All seven current source hashes match their raw snapshots, candidate manifest and fresh GREEN receipt before/after entries. The receipt records 730 backend Go/module files unchanged, CGO=1, real disposable MySQL, race/count=1 and exit 0. The ^TestRetire log records four top-level groups passed in 39.079s, including full Create -> Preview -> Start -> Mark -> replay. That case asserts no credentials/runner/steps/attempts/CMDB write, no software-version change, exactly one real owner retirement and immutable terminal replay.

The initial and corrective REDs remain preserved. MySQL-stored baselines, not pre-Save bytes, underlie snapshot assertions. Successful Uninstall still removes software exactly once and delegates real STOP then Retire with the expected revision/correlation ordering. Historical executable Retire remains readable/cancellable. service_link_control_test.go changes only the old RetireRetry setup from Update laundering to Create of new metadata; execution-generation, no-begin lifecycle, stale-report rejection and replay assertions are unchanged. That differently named test is not covered by ^TestRetire; root's corrected full-package compatibility run remains pending and is not claimed here.

### Diagnostics and stop condition

Named lsp_diagnostics/ast_grep_search tools are unavailable. Independent gopls check on all seven narrowed files exited 0 with no type errors and one existing analyzer diagnostic: unnecessary fmt.Sprintf at deploy_service.go:5075, unchanged in the Retire diff. Independent scoped git diff --check exited 0. Manual diff/context review found no new hardcoded secret, SQL injection or alternate software-execution path. No DB test, source edit or runtime/goal operation was performed by this reviewer.

The review scope is covered; approval waits for explicit async rejection repair and source-bound regression evidence. Whole C01, complete backend compatibility/race, live role/provider/runtime equivalence, UI and maintainer/archive acceptance remain open.

```json
{
  "schema": "pantheon-independent-code-review/v1",
  "taskId": "2026-11-17-ops-history-audit",
  "workItem": "C01/G003 Retire metadata continuation",
  "reviewerLane": "c03_code_review",
  "recordedAtUtc": "2026-11-17T17:54:46.347Z",
  "verdict": "REQUEST CHANGES",
  "verdictScope": "SEVEN_FILE_RETIRE_PRE_PRODUCTION_TO_GREEN_DELTA",
  "filesReviewed": 7,
  "issueCounts": {
    "CRITICAL": 0,
    "HIGH": 0,
    "MEDIUM": 1,
    "LOW": 0
  },
  "newFindings": [
    {
      "id": "C01-RETIRE-ASYNC-01",
      "severity": "MEDIUM",
      "file": "backend/modules/business/deploy/deploy_service.go",
      "line": 1576,
      "relatedLines": [1577, 1769, 1770, 1783, 1787, 1844, 1849, 1851],
      "title": "Async Retire/unknown-kind rejection silently abandons a durable running task",
      "trigger": "A pre-existing running SSH legacy Retire or unknown-kind task with pending host, its own lease, and no running attempt is redispatched by ReconcileDeployQueue",
      "issue": "The new validation error is discarded; worker slot exits with task/host still running or pending, no rejection record, and no lease settlement",
      "fix": "Keep the remote guard, route its error through explicit fenced queued rejection; expose both the validation error and any settlement failure instead of discarding errors. Add real queue regression proving no script, observable rejection, appropriate terminal/blocked state and own-lease settlement without software/Retire owner mutation",
      "evidenceLevel": "DETERMINISTIC_SOURCE_TRACE; NEW_DB_REPRO_NOT_RUN_BY_REVIEWER",
      "rootCauseGuard": "FAIL_NEW_ERROR_SUPPRESSION_BRANCH"
    }
  ],
  "specCompliance": {
    "metadataMarker": "PASS_EXPLICIT_KIND_NON_NIL_EMPTY_STEPS",
    "createReferences": "PASS_IDENTITY_ONLY_NO_EXECUTABLE_TEMPLATE_COPY",
    "updateLaundering": "PASS_DIRECT_AND_TEMPLATE_DEFAULT_CONVERSION_REJECTED",
    "controlsAndWorkerClaims": "PASS_SCOPED_GUARDS",
    "completion": "PASS_NO_CMDB_WRITE_OR_SOFTWARE_VERSION_TRANSITION_AND_ATOMIC_SUMMARY_REJECTION",
    "historicalReadsAndUninstall": "PASS_RECORDED_CONTROLS",
    "asyncErrorVisibility": "FAIL"
  },
  "sourceBinding": {
    "filesBound": 7,
    "rawCurrentManifestAndGreenBeforeAfterMatch": true,
    "files": [
      {
        "source": "backend/modules/business/deploy/deploy_model.go",
        "sha256": "1a774c3587f12bb36044e9c8c28536ea52165b261785d1ee8717355476b0318f",
        "rawMatch": true,
        "currentMatch": true,
        "greenReceiptMatch": true
      },
      {
        "source": "backend/modules/business/deploy/deploy_service.go",
        "sha256": "e380400b2add09dddddc73e962a6bd70baa71c13aeab2f257f8ee16f4ba83b5a",
        "rawMatch": true,
        "currentMatch": true,
        "greenReceiptMatch": true
      },
      {
        "source": "backend/modules/business/deploy/task_command.go",
        "sha256": "05b8876df451612ca7f37f4803bc942be4412faa685c9cc283e10d17933d4825",
        "rawMatch": true,
        "currentMatch": true,
        "greenReceiptMatch": true
      },
      {
        "source": "backend/modules/business/deploy/attempt_ownership.go",
        "sha256": "09bc0f89dbf539905bd50781471c00f7c6596df5c69bdb2e756b60d60dd89410",
        "rawMatch": true,
        "currentMatch": true,
        "greenReceiptMatch": true
      },
      {
        "source": "backend/modules/business/deploy/retire_lifecycle_test.go",
        "sha256": "2c37bdeafb6e2b28fa7cab2abe979049614a92c437f6a5475d3d8089e386a252",
        "rawMatch": true,
        "currentMatch": true,
        "greenReceiptMatch": true
      },
      {
        "source": "backend/modules/business/deploy/service_link_control_test.go",
        "sha256": "13b99592f61c1a5110a161936252058b90d9cfbb11b580870b1120f120acce78",
        "rawMatch": true,
        "currentMatch": true,
        "greenReceiptMatch": true
      },
      {
        "source": "backend/modules/business/deploy/task_preview.go",
        "sha256": "9f415460dc8be26b80d976cf3dfd0b930ffd9787fda2e6889c077601c285a203",
        "rawMatch": true,
        "currentMatch": true,
        "greenReceiptMatch": true
      }
    ],
    "baseline": "retire-metadata-candidates-20261118/pre-production/*.raw",
    "candidate": "retire-metadata-candidates-20261118/green-candidate/*.raw"
  },
  "evidence": {
    "receipt": ".harness/evidence/2026-11-17-ops-history-audit/continuation-c01-retire-metadata-green-20261118-resume.log.receipt.json",
    "log": ".harness/evidence/2026-11-17-ops-history-audit/continuation-c01-retire-metadata-green-20261118-resume.log",
    "run": "go test -race -count=1 -timeout=10m ./modules/business/deploy -v -run ^TestRetire",
    "topLevelTestsPassed": 4,
    "seconds": 39.079,
    "exitCode": 0,
    "cgoEnabled": 1,
    "disposableMySQL": true,
    "backendBefore": 730,
    "backendAfter": 730,
    "sourceUnchanged": true,
    "fullCreatePreviewStartMarkReplay": true,
    "existingServiceLinkRetireRetryAssertions": "PRESERVED_SETUP_CHANGED_TO_CREATING_NEW_METADATA_TASK; EXECUTION_NOT_CLAIMED_BY_TESTRETIRE_REGEX",
    "fullDeployServiceCompatibility": "PENDING_ROOT_CORRECT_ARGS_VERIFICATION"
  },
  "diagnostics": {
    "lspDiagnosticsTool": "UNAVAILABLE",
    "astGrepSearchTool": "UNAVAILABLE",
    "substitute": "Independent gopls check on all 7 frozen files; manual scoped error-path/security review",
    "goplsExitCode": 0,
    "typeErrors": 0,
    "analyzerDiagnostics": 1,
    "analyzerDiagnostic": "Unchanged fmt.Sprintf without format args at deploy_service.go:5075; not introduced by Retire delta",
    "scopedDiffCheckExitCode": 0
  },
  "architectureEscalations": [
    "Conservative stopped-only support retained; not a proof for unknown/failed states",
    "Service PATCH retirement admission and wider host-wide foreign lease policy remain open",
    "Fenced queued rejection must preserve owner/CMDB failure evidence; broader recovery redesign remains leader-owned"
  ],
  "wholeTask": false,
  "mergeReady": false,
  "archiveApproved": false,
  "sourceModified": false,
  "testsExecutedByReviewer": false,
  "runtimeModified": false
}
```
