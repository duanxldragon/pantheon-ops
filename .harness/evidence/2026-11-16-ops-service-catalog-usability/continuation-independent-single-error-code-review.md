---
doc_type: independent-code-review
reviewer_lane: c03_code_review
recorded_at_utc: "2026-11-17T16:54:40.058Z"
verdict: COMMENT
status: narrow-single-error-feedback-review
files_reviewed: 3
critical: 0
high: 0
medium: 0
low: 1
whole_task: false
merge_ready: false
---

## Code Review Summary

**COMMENT: the scoped single-error implementation is correct; one LOW regression-hardening suggestion remains.** The shared request layer retains localized feedback and the catalog no longer adds a second mutation error toast. This review covers ServiceCatalog.tsx, service/api.ts and the relevant service-pages.spec.ts assertions, not whole C03 acceptance.

### LOW C03-SINGLE-ERROR-01: count all error toasts

**File:** frontend/tests/smoke/business/service/service-pages.spec.ts:323; the same pattern appears at line 378. staleError.toHaveCount(1) counts only the exact canonical conflict message. A second generic error toast with different text would still satisfy that assertion. Add a total visible .arco-message-error count of one after the mutation settles, retaining the exact Chinese message, input, form and CAS assertions. The inspected current screenshots show one toast, so this is optional regression hardening rather than a newly observed production defect.

### Scope, contract and root-cause check

ServiceCatalog.tsx:445-450 scopes duplicate-feedback suppression to these mutation wrappers. service/api.ts:29/33/133/137/173/189 use shared apiRequest without skipErrorMessage. The package self-export resolves to frontend/src/index.ts, which exports the same request module from which isRequestError is imported. Shared frontend/src/api/request.ts:441-458 and 483-500 display translated business/transport feedback under their existing default policy and then throw RequestError. The catalog's local catch consequently avoids adding generic feedback for that already-owned request path, while non-RequestError failures still receive local fallback feedback. This does not turn RequestError into a global handled-feedback marker or change other request call sites, authentication suppression, transport classification, retries or shared implementation.

Local form validation remains at ServiceCatalog.tsx:511/528/545 before the request try/catch. The explicit unavailable-target validation at line 554 keeps its localized Message.error and exits before submitting. Failed mutations still run finally to clear submitting; they do not reset or close the form or issue success feedback. The RequestError remains rejected and visible through the primary shared feedback path. No masking fallback or alternate mutation was introduced.

### Evidence and visual gate

Root's single-error-fixed receipt/log records eight mocked catalog cases passing in 56.9s, exit 0. The separate single-error-rendered receipt/log records both viewport cases passing in 27.0s, exit 0. All three current source hashes match both receipts' before/after hashes. Business/provider data is mocked; local login and the frontend are real. These runs do not establish production owner routes, roles or providers.

Using the impeccable gate, the reviewer directly inspected the final 1440x900 and 390x844 stale-error-input-retained screenshots. Both have one readable localized conflict toast, the open instance editor retains 3.0.0, and confirm/cancel controls are visible without modal/error clipping. The source assertions at service-pages.spec.ts:320-327/375-383 retain completed single request, cleared loading, localized visible feedback, open form, retained input and exact lifecycleVersion 7 mutation payload. The reviewer did not inspect every other screenshot or perform independent keyboard/contrast QA; no whole-page polish or full accessibility claim is made.

Named lsp_diagnostics and ast_grep_search tools are unavailable. Read-only TypeScript 6.0.3 compiler-API diagnostics found zero errors in the three reviewed root files and zero application-source diagnostics. The first custom run omitted project cwd/ambient declarations and was corrected. Combining Playwright with tsconfig.app.json leaves five existing helper-context diagnostics in tests/smoke/helpers/auth.ts (Node process globals and a .mjs declaration); smoke files are outside that app configuration. This is explicitly not a full smoke type-gate green. No diagnostic was suppressed and no source/config was changed. Scoped git diff --check exited 0.

### Stop condition

Default-feedback ownership, local validation, single-error assertions, current source hashes, eight mock/two rendered results and two final error-state screenshots are assessed. The LOW suggestion does not invalidate the observed implementation. Full C03 lifecycle/authorization/provider acceptance, broader rendered/accessibility review, final full race/rebuild and maintainer acceptance remain open. Actual task/evidence identity is 2026-11-16-ops-service-catalog-usability; the report is saved there, not under a new 2026-11-17 task. Existing reports/evidence and all production/test source remain unchanged.

### Machine-readable conclusion

```json
{
  "schema": "pantheon-independent-code-review/v1",
  "taskId": "2026-11-16-ops-service-catalog-usability",
  "reviewerLane": "c03_code_review",
  "recordedAtUtc": "2026-11-17T16:54:40.058Z",
  "verdict": "COMMENT",
  "verdictScope": "NARROW_LOCAL_MUTATION_SINGLE_ERROR_FEEDBACK",
  "filesReviewed": 3,
  "issueCounts": {
    "CRITICAL": 0,
    "HIGH": 0,
    "MEDIUM": 0,
    "LOW": 1
  },
  "newFindings": [
    {
      "id": "C03-SINGLE-ERROR-01",
      "severity": "LOW",
      "file": "frontend/tests/smoke/business/service/service-pages.spec.ts",
      "line": 323,
      "relatedLine": 378,
      "title": "Single-toast assertion counts only the canonical message",
      "issue": "An additional generic error toast with different text would not fail the canonical staleError count.",
      "fix": "After the mutation finishes, also assert the visible .arco-message-error collection has count 1 while retaining the exact localized text and form/CAS assertions.",
      "productionBehavior": "CURRENT_SCREENSHOTS_SHOW_ONE_TOAST; OPTIONAL_REGRESSION_HARDENING"
    }
  ],
  "implementationDisposition": "CORRECT_IN_SCOPED_DEFAULT_FEEDBACK_WRAPPERS",
  "rootCauseGuard": "PASS_LOCAL_DUPLICATE_FEEDBACK_REMOVED_WITH_SHARED_REQUEST_ERROR_STILL_VISIBLE",
  "sourceBinding": {
    "receiptFilesMatched": 3,
    "mockAndRenderBeforeAfterMatched": true,
    "files": [
      {
        "file": "frontend/src/modules/business/service/ServiceCatalog.tsx",
        "sha256": "f748584a67685c25d219d715a904cd941cf27350561e8679d56028bab2282d97",
        "actualSha256": "f748584a67685c25d219d715a904cd941cf27350561e8679d56028bab2282d97",
        "matches": true
      },
      {
        "file": "frontend/src/modules/business/service/api.ts",
        "sha256": "6206773022d1ba4774e66cab246a2b568e42a7dd668995aac8682bfb638eee57",
        "actualSha256": "6206773022d1ba4774e66cab246a2b568e42a7dd668995aac8682bfb638eee57",
        "matches": true
      },
      {
        "file": "frontend/tests/smoke/business/service/service-pages.spec.ts",
        "sha256": "f1cd15bbe8c7934b9e34c78ac7380c708e2819f8fbc61cfc5b6c5bc3bb9da0db",
        "actualSha256": "f1cd15bbe8c7934b9e34c78ac7380c708e2819f8fbc61cfc5b6c5bc3bb9da0db",
        "matches": true
      }
    ]
  },
  "evidence": {
    "mockTests": 8,
    "mockExitCode": 0,
    "mockTestSeconds": 56.9,
    "renderTests": 2,
    "renderExitCode": 0,
    "renderTestSeconds": 27,
    "executionOwner": "ROOT_EXISTING_LOGS_AND_RECEIPTS_REVIEWED",
    "mockLog": ".harness/evidence/2026-11-17-ops-history-audit/continuation-c03-ui-single-error-fixed.log",
    "renderLog": ".harness/evidence/2026-11-17-ops-history-audit/continuation-c03-ui-single-error-rendered.log",
    "providerAndBusinessRoutes": "MOCKED",
    "realLocalLogin": true
  },
  "visualGate": {
    "skill": "impeccable",
    "surface": "Operational catalog instance-edit form and stale-error recovery",
    "feel": "Calm, compact error feedback that preserves editable context",
    "viewports": ["1440x900", "390x844"],
    "screenshotsInspected": [
      {
        "file": ".harness/evidence/2026-11-17-ops-history-audit/continuation-c03-ui-single-error-rendered-artifacts/smoke-business-service-ser-75875-ile-rendered-states-desktop/1440x900-stale-error-input-retained.png",
        "sha256": "33ca65bec6adc3a73fabfddca4ad1ecbdf8eaef208481c2a96a0dfaad2c33230"
      },
      {
        "file": ".harness/evidence/2026-11-17-ops-history-audit/continuation-c03-ui-single-error-rendered-artifacts/smoke-business-service-ser-985a9-bile-rendered-states-mobile/390x844-stale-error-input-retained.png",
        "sha256": "cb8c6c9e944d65f680051ae14c98de892bd90020ccb87317156aff059c4643e0"
      }
    ],
    "observed": "One readable localized toast; retained 3.0.0 input; visible confirm/cancel; no clipping in scoped modal/error state",
    "unverified": [
      "Other screenshots not visually inspected in this narrow round",
      "All focus/keyboard/contrast states",
      "Live principal/role/provider acceptance"
    ]
  },
  "diagnostics": {
    "lspDiagnosticsTool": "UNAVAILABLE",
    "astGrepSearchTool": "UNAVAILABLE",
    "substitute": "TypeScript 6.0.3 compiler API with noEmit and no file writes",
    "reviewedRootFileDiagnostics": 0,
    "appSourceDiagnostics": 0,
    "combinedAppAndPlaywrightHelperDiagnostics": 5,
    "helperDiagnosticLimit": "Smoke tests are outside tsconfig.app.json. Combined checking reports Node globals/mjs declaration context diagnostics in existing tests/smoke/helpers/auth.ts; not a complete smoke type-gate pass.",
    "initialIncorrectDiagnosticContext": "Missing project cwd/ambient Vite declarations; corrected rather than suppressing errors",
    "scopedDiffCheckExitCode": 0
  },
  "wholeTask": false,
  "fullC03Acceptance": false,
  "mergeReady": false,
  "productionCodeModified": false,
  "testCodeModified": false,
  "goalStateModified": false,
  "reportCreationMode": "CREATE_NEW_NO_OVERWRITE"
}
```
