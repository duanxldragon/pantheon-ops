---
doc_type: independent-code-review-appendix
reviewer_lane: c03_code_review
recorded_at_utc: "2026-11-17T18:15:04.857Z"
verdict: APPROVE
status: narrow-retire-queued-rejection-closure
files_reviewed: 2
files_bound: 7
critical: 0
high: 0
medium: 0
low: 0
whole_task: false
merge_ready: false
archive_approved: false
---

## Code Review Summary

**APPROVE the queued-green Retire candidate within this narrow scope. C01-RETIRE-ASYNC-01 (MEDIUM) is CLOSED; no new findings.** This exclusive round2 appendix preserves round1 REQUEST CHANGES and the raw/RED/GREEN evidence. It closes the new silent rejection branch rather than granting whole C01, merge or archive approval.

### Primary failure contract and scope

The green-candidate -> queued-green raw diff changes only deploy_service.go and retire_lifecycle_test.go. The production repair retains the existing fenced helper and ownership policy. failQueuedTask at deploy_service.go:1777-1800 now returns the original fencedTransaction result. runAsyncTask at :1578-1583 logs the intent rejection, invokes that helper and logs both the original reason and settlement error when it cannot settle. It adds no localOnly transaction, alternate execution path, retry, bypass or lock-order change.

Both earlier callers also handle the returned error: failStartedTask at :1491-1492 and credential-failure handling at :1587-1588. Successful settlement retains the rejection reason in the failed host row; failed settlement is reported rather than ignored. These old caller paths are source-reviewed here, without a claim that the new eight-case matrix separately executes both old callers.

The helper still refuses running attempts, directly fails queued hosts, releases only the task-owned leases and recomputes status in the same transaction. It never invokes host-result software projection. Failed Retire still has no owner transition or three-version write. Unknown-kind Install follows its existing real failure owner transition. If running ownership or Service snapshot/staleness/query validation prevents settlement, the transaction rolls back and the logs retain both errors; the change does not pretend that every failure can become terminal. Broader queued recovery policy remains leader-owned.

### Genuine regression evidence

The frozen eight-case test SHA is identical in queued RED and GREEN. Root's real CGO/MySQL race RED exited 1 in 9.408s with all eight cases failed. The fresh all-green receipt invokes go test -race -count=1 -timeout=10m ./modules/business/deploy -v -run ^TestRetire, exits 0 and records all five top-level Retire groups passed in 51.208s. The queue group passes all eight subcases: legacy executable Retire, unknown-kind Install, metadata SSH, Manual metadata misdispatch, running attempt, stale owner, missing owner snapshot and actual transaction-bound owner-query failure.

retire_lifecycle_test.go:701-742 captures the real async worker at query entry and waits for slot.done after deferred cleanup; it does not substitute a sleep or timing sample for worker exit. Lines :745-828 require visible task/host failure and exact rejection reason, one completion event, lease release, zero attempts/credentials/runners/CMDB writes, frozen software/snapshot facts and immutable replay. The Install case preserves its real owner failed transition; Retire cases preserve all owner facts. Lines :831-895 require running-attempt protection or owner-fault rollback, both logged causes, zero owner/CMDB mutations, and actual entry into the injected transaction-bound owner query. The owner spies still delegate original capabilities/factory(tx). No production failure is faked into success.

All seven current candidate sources match their raw snapshots, source manifest and fresh execution before/after hashes. The receipt records all 730 backend Go/module sources unchanged. The same fresh five-group run reruns marker/admission/laundering, remote bypass isolation, full Create -> Preview -> Start -> Mark/replay, version-neutral atomic completion, Uninstall STOP-to-Retire and historical cancellation controls. Five source files remain byte-identical to round1; earlier valid contract conclusions remain applicable. The separately named ServiceLink RetireRetry assertions remain preserved, with no execution claim under ^TestRetire.

### Diagnostics and limits

Named lsp_diagnostics/ast_grep_search tools are unavailable. Independent gopls check on both modified files exited 0 with no type errors; its one pre-existing analyzer diagnostic is the unchanged unnecessary fmt.Sprintf at deploy_service.go:5083 (round1 :5075 before eight inserted lines). Round1 diagnostics cover the five unchanged bound files. Scoped git diff --check exited 0. Source/pattern inspection finds no new masking fallback, secret, injection or alternate command path in the helper/error-log change.

The original round1 report SHA256 remains 740a70f6159548f8e7336fea5c63c581b955d820ad3d9ec333c2b63a243b2b5d. Full backend race and rebuilt-runtime verification are ongoing root-owned gates, not results claimed by this appendix. Service PATCH retirement admission, wider host lease policy, conservative stopped-only limits, live role/provider/runtime and maintainer acceptance remain open. No DB tests, source edits, browser/runtime or goal mutations were performed by the reviewer.

```json
{
  "schema": "pantheon-independent-code-review/v1",
  "taskId": "2026-11-17-ops-history-audit",
  "workItem": "C01/G003 Retire metadata queued-rejection fix",
  "reviewerLane": "c03_code_review",
  "recordedAtUtc": "2026-11-17T18:15:04.857Z",
  "verdict": "APPROVE",
  "verdictScope": "NARROW_QUEUED_REJECTION_REPAIR_AND_PRIOR_RETIRE_CANDIDATE",
  "filesReviewed": 2,
  "filesBound": 7,
  "issueCounts": {
    "CRITICAL": 0,
    "HIGH": 0,
    "MEDIUM": 0,
    "LOW": 0
  },
  "closedFindings": [
    {
      "id": "C01-RETIRE-ASYNC-01",
      "priorSeverity": "MEDIUM",
      "status": "CLOSED_FOR_QUEUED_GREEN_CANDIDATE",
      "file": "backend/modules/business/deploy/deploy_service.go",
      "line": 1578
    }
  ],
  "rootCauseGuard": "PASS_PRIMARY_FENCED_REJECTION_REUSED_WITH_VALIDATION_AND_SETTLEMENT_ERRORS_VISIBLE",
  "productionDelta": "failQueuedTask returns the original fencedTransaction error; all 3 callers handle settlement error; no localOnly path or lock-order change",
  "priorReport": {
    "file": ".harness/evidence/2026-11-17-ops-history-audit/retire-metadata-candidates-20261118/green-candidate/independent-code-review-round1.md",
    "sha256": "740a70f6159548f8e7336fea5c63c581b955d820ad3d9ec333c2b63a243b2b5d",
    "verdict": "REQUEST CHANGES",
    "medium": 1
  },
  "sourceBinding": {
    "rawCurrentCandidateAndRootBeforeAfterMatch": true,
    "files": [
      {
        "source": "backend/modules/business/deploy/deploy_model.go",
        "sha256": "1a774c3587f12bb36044e9c8c28536ea52165b261785d1ee8717355476b0318f",
        "currentAndRawMatch": true,
        "rootBeforeAfterMatch": true
      },
      {
        "source": "backend/modules/business/deploy/deploy_service.go",
        "sha256": "1cc0c8e17198fa268b0261809824d0f9cb2a4d73e2bcfe4d06da796ab58a4df9",
        "currentAndRawMatch": true,
        "rootBeforeAfterMatch": true
      },
      {
        "source": "backend/modules/business/deploy/task_command.go",
        "sha256": "05b8876df451612ca7f37f4803bc942be4412faa685c9cc283e10d17933d4825",
        "currentAndRawMatch": true,
        "rootBeforeAfterMatch": true
      },
      {
        "source": "backend/modules/business/deploy/attempt_ownership.go",
        "sha256": "09bc0f89dbf539905bd50781471c00f7c6596df5c69bdb2e756b60d60dd89410",
        "currentAndRawMatch": true,
        "rootBeforeAfterMatch": true
      },
      {
        "source": "backend/modules/business/deploy/retire_lifecycle_test.go",
        "sha256": "04e9bb5ea52d96a33eccdbda1c00133c53c8697ccd2259334905e9be86f75a24",
        "currentAndRawMatch": true,
        "rootBeforeAfterMatch": true
      },
      {
        "source": "backend/modules/business/deploy/service_link_control_test.go",
        "sha256": "13b99592f61c1a5110a161936252058b90d9cfbb11b580870b1120f120acce78",
        "currentAndRawMatch": true,
        "rootBeforeAfterMatch": true
      },
      {
        "source": "backend/modules/business/deploy/task_preview.go",
        "sha256": "9f415460dc8be26b80d976cf3dfd0b930ffd9787fda2e6889c077601c285a203",
        "currentAndRawMatch": true,
        "rootBeforeAfterMatch": true
      }
    ],
    "unchangedFilesFromR1": 5,
    "queuedRedTestHashMatchesGreen": true
  },
  "evidence": {
    "receipt": ".harness/evidence/2026-11-17-ops-history-audit/continuation-c01-retire-queued-all-green-20261118.log.receipt.json",
    "log": ".harness/evidence/2026-11-17-ops-history-audit/continuation-c01-retire-queued-all-green-20261118.log",
    "args": [
      "test",
      "-race",
      "-count=1",
      "-timeout=10m",
      "./modules/business/deploy",
      "-v",
      "-run",
      "^TestRetire"
    ],
    "startedAt": "2026-11-17T18:08:07.6183174Z",
    "finishedAt": "2026-11-17T18:09:11.5928350Z",
    "cgoEnabled": 1,
    "race": true,
    "disposableMySQL": true,
    "exitCode": 0,
    "topLevelTestsPassed": 5,
    "queuedSubtestsPassed": 8,
    "seconds": 51.208,
    "backendBefore": 730,
    "backendAfter": 730,
    "sourceUnchanged": true,
    "queuedRedExitCode": 1,
    "queuedRedAll8Failed": true,
    "queuedRedSeconds": 9.408,
    "fullCreatePreviewStartMarkReplay": "PASSED_IN_FRESH_5_GROUP_RUN",
    "fullBackendRace": "PENDING_ROOT",
    "rebuiltRuntimeEquivalence": "PENDING_ROOT"
  },
  "diagnostics": {
    "lspDiagnosticsTool": "UNAVAILABLE",
    "astGrepSearchTool": "UNAVAILABLE",
    "substitute": "Independent gopls check on both changed files; R1 seven-file checks remain applicable to five unchanged files; scoped manual error/security review",
    "goplsExitCode": 0,
    "typeErrors": 0,
    "analyzerDiagnostics": 1,
    "analyzerDiagnostic": "Existing unnecessary fmt.Sprintf at deploy_service.go:5083; original R1 line5075 shifted by 8 inserted lines",
    "scopedDiffCheckExitCode": 0
  },
  "failureBehavior": {
    "validUnclaimedRetire": "TASK_HOST_FAILED_REASON_PERSISTED_OWN_LEASE_RELEASED_NO_OWNER_OR_CMDB_WRITE",
    "invalidInstall": "EXISTING_REAL_FAILED_OWNER_TRANSITION_PRESERVED",
    "runningAttempt": "NOT_OVERWRITTEN_REJECTION_AND_OWNERSHIP_FAILURE_LOGGED",
    "staleMissingOrQueryFailedOwner": "FENCED_ROLLBACK_FACTS_UNCHANGED_BOTH_ERRORS_LOGGED",
    "workerReplay": "TERMINAL_FACTS_UNCHANGED",
    "oldStartAndCredentialCallers": "ALL_SETTLEMENT_ERRORS_HANDLED_BY_SOURCE_REVIEW_NO_SEPARATE_RUNTIME_COVERAGE_CLAIM"
  },
  "wholeTask": false,
  "mergeReady": false,
  "archiveApproved": false,
  "sourceModified": false,
  "testsExecutedByReviewer": false,
  "runtimeModified": false
}
```
