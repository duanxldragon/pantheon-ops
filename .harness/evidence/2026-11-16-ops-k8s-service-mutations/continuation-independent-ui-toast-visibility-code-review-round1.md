---
doc_type: independent-code-review
reviewer_lane: c03_code_review
round: 1
recorded_at_utc: "2026-11-17T17:10:05.065Z"
verdict: REQUEST CHANGES
status: narrow-ui-toast-visibility-red-checkpoint
files_reviewed: 4
critical: 0
high: 0
medium: 1
low: 0
whole_task: false
merge_ready: false
---

## Code Review Summary

**REQUEST CHANGES for this C04 UI message-layer candidate.** The fresh root UI run is 10 passed / 3 failed; the failed conflict cases observe two canonical notices, then zero after expiry. One MEDIUM finding remains. Completed lint/type-check/build checks do not supersede this dynamic red. This verdict does not expand into backend, whole C04 or production acceptance.

**Files reviewed:** ServiceEditor.tsx, ServiceList.tsx, the Service create/update/delete API wrappers in frontend/src/modules/business/k8s/api.ts, and the relevant k8s-pages.spec.ts assertions. AppModal and installed Arco Message/Notice source were read for contract context. All seventeen candidate files were hash-bound; broader locale/docs/seed behavior was not newly approved here.

### MEDIUM C04-UI-TOAST-01: owned hook produces duplicate conflict notices

**File:** frontend/src/modules/business/k8s/service/ServiceList.tsx:65; related call at lines 70 and 207.

The candidate moves write-error feedback into Message.useMessage under the existing modal title. The fresh root log/receipt confirms failures in stale edit and both 1440px/390px primary-render cases. Each call history records ten polls resolving to two canonical .arco-message elements, followed by zero after the notices expire. The same source hashes are unchanged before/after the run and match the seventeen-file manifest. Consequently the required one visible error notice is not satisfied.

Static dependency inspection rules out the proposed generic outer/inner-class explanation: installed Arco _class/notice.js:138 has a classless outer div and line 139 has the one inner arco-message/error alert; Message/index.js:129 uses arco-message-wrapper for its container. Two canonical matches therefore are not automatically one notice counted twice by those source wrappers.

A source-backed root-cause hypothesis remains: frontend/src/main.tsx:29 enables React.StrictMode. Installed Message/useMessage.js creates its first Message with a callback ref that calls instance.add(_noticeProps) on every non-null ref attachment. Notification.add at _class/notification.js:30-49 generates a random identity when absent and concatenates it. React 19 StrictMode ref reattachment can therefore add the initial notice twice, before the existing-instance/maxCount path is used. memoization prevents ordinary parent renders from recreating the hook owner but does not prove ref-attachment idempotency. This causal explanation is a static hypothesis, not an independently executed lifecycle trace by this reviewer; the duplicated DOM count itself is grounded in root's fresh run.

**Fix:** repair initial owner-scoped notice creation so it is idempotent under the existing React/StrictMode lifecycle, or reuse a proven owner-scoped feedback primitive. Preserve the real error, original draft/identity and all current single-message/readability/cleanup assertions. If the controllable defect belongs to a shared primitive, route the generic repair Base-first rather than patching shared code in Ops. Do not make the red disappear with first(), weaker counts, disabled StrictMode, a longer timeout or a broad z-index override. Prove single ownership before declaring visibility or close cleanup fixed.

### Spec, error and owner-lifecycle checks

createK8sService and updateK8sService at api.ts:430/440 accept an optional skipErrorMessage display option and pass it through at lines 436/452. Their defaults keep shared feedback enabled; only this editor's create/update calls opt out at ServiceList.tsx:179/190. DELETE at api.ts:456 retains default global feedback. Permission/data/identity contracts are not bypassed by the option.

The local catch at ServiceList.tsx:198-207 chooses the original RequestError messageKey and kind fallback or the translated non-RequestError create/update failure key. It retains draft and expected UID/resourceVersion, adds no refreshed-identity retry and clears submitting in finally. The reference remains owned by the modal; absence after unmount avoids an orphan global toast. This is a scoped ownership change, not blanket suppression of arbitrary RequestError elsewhere.

The holder at ServiceList.tsx:63 is memoized and rendered in the existing AppModal title at line 420, with maxCount=1. AppModal's default unmountOnExit=true removes the holder after closing. Cancel, Escape and close are disabled during submission at lines 425-430. DELETE remains a danger action behind Popconfirm at lines 269-306. ServiceEditor.tsx:151-162 removes only a draft port with a neutral IconMinus/text button and prevents deleting the final port; it is distinct from a persisted Service deletion.

Those source contracts are plausible and bounded. They do not overcome the duplicated first notice, nor establish runtime owner cleanup while the primary tests fail before reaching their cancel/reopen assertions.

### Tests, diagnostic status and visual gap

k8s-pages.spec.ts:159-225 requires both canonical notice count=1 and total error count=1, then checks full viewport bounds, settled opacity/animations, content fit and center/four-corner elementFromPoint hits. The assertions preserve useful failing evidence; no first() or alternate selector bypass was introduced. The primary-state cases at lines 499-503 also require the modal to close, the owned message to disappear, and reopening Create to show no stale notice.

The final fresh root receipt is exit 1, sourceUnchanged=true: 10 passed / 3 failed in about 1.6 minutes. The failed primary cases stop at canonical count, so five-point readability, cancel/reopen cleanup and their conflict screenshots are not completed evidence. Prior thirteen-case green screenshots from the pre-layer candidate cannot approve this replacement.

Root's completed full frontend lint, type-check and build receipts all have exit 0. The owner-scoped eslint/type-check/shell contract/discovery/diff checks in verification.json also completed with exit 0. Named lsp_diagnostics and ast_grep_search are unavailable; this read-only lane did not start diagnostic or test processes and reports the actual root language-check evidence. All seventeen current hashes match the supplied candidate manifest; the fresh UI receipt binds the same seventeen source files before and after its failed run.

Impeccable applies to the operational create/edit modal and recovery feedback. Both intended viewports are represented by the fresh suite, but conflict readability and close cleanup remain dynamically blocked. The run captured loading/editor/submitting states; no successful new conflict screenshot exists in the two failed primary-state directories. No visual polish/approval is claimed.

### Stop condition and boundaries

The independent code/spec lane has read all four scoped sources and the actual Message/modal contracts, preserved and assessed the fresh red, checked completed diagnostics and current hashes, and handed the root-cause hypothesis upward. Further implementation and dynamic DOM/lifecycle verification belong to the executor/root lanes. A new immutable report is created for this failed candidate; older backend/UI reports and all source/runtime/database state remain unchanged. Whole C04, backend acceptance, live provider/role admission, broader UI completion and final maintainer acceptance remain open.

### Machine-readable conclusion

```json
{
  "schema": "pantheon-independent-code-review/v1",
  "taskId": "2026-11-16-ops-k8s-service-mutations",
  "reviewerLane": "c03_code_review",
  "recordedAtUtc": "2026-11-17T17:10:05.065Z",
  "reviewRound": 1,
  "verdict": "REQUEST CHANGES",
  "verdictScope": "NARROW_C04_UI_TOAST_VISIBILITY_CANDIDATE",
  "filesReviewed": 4,
  "issueCounts": {
    "CRITICAL": 0,
    "HIGH": 0,
    "MEDIUM": 1,
    "LOW": 0
  },
  "newFindings": [
    {
      "id": "C04-UI-TOAST-01",
      "severity": "MEDIUM",
      "file": "frontend/src/modules/business/k8s/service/ServiceList.tsx",
      "line": 65,
      "relatedLines": [70, 207],
      "title": "The new owned Message hook renders duplicate conflict notices",
      "evidence": "Root fresh UI run: 10 PASS / 3 FAIL, each canonical message locator resolves to 2 before timing out at 0.",
      "reproductionStatus": "OBSERVED_IN_ROOT_FRESH_RUN_BOUND_TO_CURRENT_HASHES",
      "rootCauseStatus": "STRICTMODE_CALLBACK_REF_DOUBLE_ADD_IS_STATIC_HYPOTHESIS_NOT_INDEPENDENTLY_EXECUTED",
      "fix": "Make initial owned-notice creation idempotent under the existing React 19/StrictMode lifecycle or reuse a proven owner-scoped feedback primitive. Preserve canonical/total count=1, five-point hit tests, draft/CAS and cancel cleanup assertions. Do not hide with first(), weaken counts, disable StrictMode or broadly alter shared stacking."
    }
  ],
  "stage1": "FAIL_SINGLE_VISIBLE_ERROR_AND_FULL_PRIMARY_STATE_ACCEPTANCE",
  "rootCauseGuard": "HOLD_PRIMARY_LOCAL_MESSAGE_CONTRACT_NEEDS_REPAIR_NO_TEST_BYPASS_ALLOWED",
  "sourceBinding": {
    "verification": ".harness/evidence/2026-11-16-ops-k8s-service-mutations/continuation-ui/toast-visibility-20261118/verification.json",
    "matched": 17,
    "total": 17,
    "files": [
      {
        "path": "frontend/src/modules/business/k8s/api.ts",
        "sha256": "b5da34dbaf6741ebb5025527596746d804eb238cec45f232ae94c2fde15c6fe9",
        "actualSha256": "b5da34dbaf6741ebb5025527596746d804eb238cec45f232ae94c2fde15c6fe9",
        "matches": true
      },
      {
        "path": "frontend/src/modules/business/k8s/service/ServiceList.tsx",
        "sha256": "05b888983b870c006e197d1f8e70765b4f879f86a050e07a8a858df0ba2afc91",
        "actualSha256": "05b888983b870c006e197d1f8e70765b4f879f86a050e07a8a858df0ba2afc91",
        "matches": true
      },
      {
        "path": "frontend/src/modules/business/k8s/service/ServiceEditor.tsx",
        "sha256": "a805d7558bcf94a6a1a33e80f1b9078907fd3f597ec052cfb54c80a5d9efbc6c",
        "actualSha256": "a805d7558bcf94a6a1a33e80f1b9078907fd3f597ec052cfb54c80a5d9efbc6c",
        "matches": true
      },
      {
        "path": "frontend/src/modules/business/k8s/service/serviceDraft.ts",
        "sha256": "a537a941695469d3001a46ef39c3e01a81afc26bc2826bda289fc238bdbfb522",
        "actualSha256": "a537a941695469d3001a46ef39c3e01a81afc26bc2826bda289fc238bdbfb522",
        "matches": true
      },
      {
        "path": "frontend/src/modules/business/k8s/service/service-editor.css",
        "sha256": "e4365e2e6929efeb3dd7b160cb4a4faf2d69902a90e3768066b053ca5bca030a",
        "actualSha256": "e4365e2e6929efeb3dd7b160cb4a4faf2d69902a90e3768066b053ca5bca030a",
        "matches": true
      },
      {
        "path": "frontend/tests/smoke/business/k8s/k8s-pages.spec.ts",
        "sha256": "11d380dfc3a75adaf7ad6cf377caa5be0fdc3e722ba89a2a153b73b991014c8b",
        "actualSha256": "11d380dfc3a75adaf7ad6cf377caa5be0fdc3e722ba89a2a153b73b991014c8b",
        "matches": true
      },
      {
        "path": "docs/designs/BUSINESS_ERROR_SEMANTICS_APPENDIX.md",
        "sha256": "ab7ca8b39a71328bcbc265a58ceaff0bcbf7d38e2a42b9a4532435b853bc4d4d",
        "actualSha256": "ab7ca8b39a71328bcbc265a58ceaff0bcbf7d38e2a42b9a4532435b853bc4d4d",
        "matches": true
      },
      {
        "path": "frontend/src/modules/business/v12locales/zh-CN.json",
        "sha256": "227775201f7ef0600efa9bbeecaa59987755af373bd5f55aea69fb3568f9a497",
        "actualSha256": "227775201f7ef0600efa9bbeecaa59987755af373bd5f55aea69fb3568f9a497",
        "matches": true
      },
      {
        "path": "frontend/src/i18n/resources/business/zh-CN.ts",
        "sha256": "8afaf31a4802897c811033b3d455cc9dcf178a925a26249db542ff572005cb6b",
        "actualSha256": "8afaf31a4802897c811033b3d455cc9dcf178a925a26249db542ff572005cb6b",
        "matches": true
      },
      {
        "path": "frontend/src/modules/business/v12locales/en-US.json",
        "sha256": "d26f80a51a58ab04365a41303131ea3afb4eb582d4f224a82eeaed1c3097c7cb",
        "actualSha256": "d26f80a51a58ab04365a41303131ea3afb4eb582d4f224a82eeaed1c3097c7cb",
        "matches": true
      },
      {
        "path": "frontend/src/i18n/resources/business/en-US.ts",
        "sha256": "b23daa06a55d7456903f0f634e12571c4895daa101f319bcc307587bf836502d",
        "actualSha256": "b23daa06a55d7456903f0f634e12571c4895daa101f319bcc307587bf836502d",
        "matches": true
      },
      {
        "path": "frontend/src/modules/business/v12locales/ja-JP.json",
        "sha256": "ff1c96664e32767092497bbb65bbc953fceb804bcaf3db26c34391c98573ff66",
        "actualSha256": "ff1c96664e32767092497bbb65bbc953fceb804bcaf3db26c34391c98573ff66",
        "matches": true
      },
      {
        "path": "frontend/src/i18n/resources/business/ja-JP.ts",
        "sha256": "428a8832aa32a19d5309dd0656829386eb9ad64d657ffed1040d207ce707253d",
        "actualSha256": "428a8832aa32a19d5309dd0656829386eb9ad64d657ffed1040d207ce707253d",
        "matches": true
      },
      {
        "path": "frontend/src/modules/business/v12locales/ko-KR.json",
        "sha256": "ea55b0eb1e03caa62668b7cdbd19645c73b372c74454e68200d6cc3eb133ecf8",
        "actualSha256": "ea55b0eb1e03caa62668b7cdbd19645c73b372c74454e68200d6cc3eb133ecf8",
        "matches": true
      },
      {
        "path": "frontend/src/i18n/resources/business/ko-KR.ts",
        "sha256": "8805f78453d00c46f4b256c432966b406f953cab00fdab4fb52a8ab99cc228e5",
        "actualSha256": "8805f78453d00c46f4b256c432966b406f953cab00fdab4fb52a8ab99cc228e5",
        "matches": true
      },
      {
        "path": "frontend/src/modules/business/v12locales/fr-FR.json",
        "sha256": "94405937f248745cf252d5f7a4f46878f9b6402a6dfa76f6f89fdb2b6702fb9f",
        "actualSha256": "94405937f248745cf252d5f7a4f46878f9b6402a6dfa76f6f89fdb2b6702fb9f",
        "matches": true
      },
      {
        "path": "frontend/src/i18n/resources/business/fr-FR.ts",
        "sha256": "fd1d3089f4798928d070277d85dc8c64fdd2256249de92a2fae790aaf1168768",
        "actualSha256": "fd1d3089f4798928d070277d85dc8c64fdd2256249de92a2fae790aaf1168768",
        "matches": true
      }
    ]
  },
  "implementationChecks": {
    "apiGlobalFeedbackOptIn": "Only create/update accept options; defaults remain enabled",
    "localMessageOwner": "memoized forwardRef holder in existing AppModal title; maxCount=1 does not establish first-mount idempotency",
    "draftAndIdentity": "Original draft/UID/resourceVersion retained on catch; no token refetch/retry",
    "cancel": "Disabled while submitting; modal unmountOnExit destroys owned holder after close",
    "delete": "Global default feedback, danger status and Popconfirm remain",
    "draftRemoval": "Neutral IconMinus text button; no persisted-delete styling"
  },
  "diagnostics": {
    "lspDiagnosticsTool": "UNAVAILABLE",
    "astGrepSearchTool": "UNAVAILABLE",
    "reviewerDiagnosticsProcessesStarted": false,
    "rootScopedEslintExitCode": 0,
    "rootScopedTypeCheckExitCode": 0,
    "rootFrontendResults": [
      {
        "name": "frontend-lint-final-c04-toast-visibility-fixed",
        "exitCode": 0,
        "status": "passed",
        "log": ".harness/evidence/2026-11-17-ops-history-audit/continuation-frontend-lint-final-c04-toast-visibility-fixed.log"
      },
      {
        "name": "frontend-typecheck-final-c04-toast-visibility-fixed",
        "exitCode": 0,
        "status": "passed",
        "log": ".harness/evidence/2026-11-17-ops-history-audit/continuation-frontend-typecheck-final-c04-toast-visibility-fixed.log"
      },
      {
        "name": "frontend-build-final-c04-toast-visibility-fixed",
        "exitCode": 0,
        "status": "passed",
        "log": ".harness/evidence/2026-11-17-ops-history-audit/continuation-frontend-build-final-c04-toast-visibility-fixed.log"
      }
    ],
    "rootDiffCheckExitCode": 0
  },
  "evidence": {
    "freshUIReceipt": ".harness/evidence/2026-11-17-ops-history-audit/continuation-c04-ui-toast-visibility-fixed.receipt.json",
    "freshUILog": ".harness/evidence/2026-11-17-ops-history-audit/continuation-c04-ui-toast-visibility-fixed.log",
    "exitCode": 1,
    "passed": 10,
    "failed": 3,
    "failedCases": [
      "stale edit",
      "1440px rendered primary states",
      "390px rendered primary states"
    ],
    "sourcesBeforeAfterStable": true,
    "executionOwner": "ROOT_EXISTING_LOGS_AND_RECEIPTS_REVIEWED",
    "businessAPIAndRoles": "MOCKED",
    "realLocalFrontendAndLogin": true
  },
  "staticDependencyEvidence": {
    "noticeClasses": "Notice outer div has no class, inner alert has arco-message/error; Message container has arco-message-wrapper",
    "strictMode": "frontend/src/main.tsx:29",
    "firstMessageRef": "Arco es/Message/useMessage.js addNotice creates ref callback that invokes add on every non-null ref",
    "identity": "Arco es/_class/notification.js:30-49 generates a new random id if absent and concatenates",
    "memoizationLimit": "memo avoids parent re-render; does not suppress StrictMode ref reattachment"
  },
  "visualGate": {
    "skill": "impeccable",
    "surface": "Operational Kubernetes Service create/edit modal error recovery",
    "viewports": ["1440x900", "390x844"],
    "freshConflictReadability": "BLOCKED_BEFORE_FIVE_POINT_VISIBILITY_ASSERTIONS",
    "closeOwnedMessageCleanup": "NOT_REACHED_BY_FAILED_PRIMARY_STATE_TESTS",
    "freshConflictScreenshots": "NOT_CAPTURED_IN_FAILED_PRIMARY_STATE_CASES",
    "visualApproval": false
  },
  "excludedScope": [
    "Backend owners and previous appProtocol review",
    "Other manifest files hash-bound only; broader locale/seed validation belongs to root",
    "Production provider/role admission",
    "Whole C04 acceptance"
  ],
  "wholeTask": false,
  "fullC04Acceptance": false,
  "mergeReady": false,
  "productionCodeModified": false,
  "testCodeModified": false,
  "goalStateModified": false,
  "runtimeChanged": false,
  "reportCreationMode": "CREATE_NEW_NO_OVERWRITE"
}
```
