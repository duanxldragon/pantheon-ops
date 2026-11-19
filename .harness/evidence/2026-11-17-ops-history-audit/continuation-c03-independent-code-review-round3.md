---
doc_type: independent-code-review
round: 3
reviewer_lane: c03_code_review
recorded_at_utc: "2026-11-17T14:59:58.909Z"
verdict: COMMENT
status: narrow-provider-cause-and-stale-fixture-checkpoint
files_reviewed: 3
critical: 0
high: 0
medium: 0
low: 0
full_c03_acceptance: false
merge_ready: false
---

## Code Review Summary

**Recommendation: COMMENT.** The concrete K8s reader now preserves non-NotFound provider failures, and the stale UI fixture matches the actual shared business envelope without weakening the user-facing conflict assertions. No new issue was found in this narrow review. This is not full C03 acceptance or a merge-ready verdict.

**Files reviewed:** three changed files: backend/modules/business/k8s/capability.go; backend/modules/business/k8s/capability_error_test.go; frontend/tests/smoke/business/service/service-pages.spec.ts. Existing shared HTTP envelope source was read for contract context. ServiceCatalog and service API were checked against the successful UI receipt hashes. Original four findings retain round-two dispositions and were not rerun. Concurrent C04 K8s service/namespace/cluster implementation is excluded.

**New issues:** CRITICAL 0 / HIGH 0 / MEDIUM 0 / LOW 0.

### Provider cause preservation

At capability.go:38 the existing Deployment, StatefulSet and DaemonSet reads retain their primary client-go GET calls. The shared error branch at capability.go:48 converts only apierrors.IsNotFound(err) into the established business.service.k8s_target_not_found key. Every other error returns unchanged at line 52. Target metadata lookup at line 54 is reached only after a successful workload read. The previous suppression of 403, 500, transport and cancellation causes is repaired in this bounded reader; no alternate execution path, silent default, broad catch, or bypass was introduced.

capability_error_test.go:85 uses real client-go REST clients with HTTP 404/403/500 server responses and an errors.Is transport sentinel for all three workload kinds. It asserts both correct cause/classification and that failed reads never proceed into metadata. The in-flight cancellation test at line 107 uses a transport-entry gate, cancels after entry, and checks errors.Is(context.Canceled) for all three kinds, without sleeps.

The integration test at capability_error_test.go:157 injects the concrete reader into the real Service owner with disposable MySQL, then exercises direct owner update and the actual Service instance HTTP handler. For a Deployment target, it verifies canonical not-found refusal, original forbidden/server/network cause classifications at the owner, expected business-envelope codes, and unchanged complete instance facts after both refusal paths. This does not claim HTTP cancellation or all-kind HTTP coverage: those three-kind and cancellation checks belong to the reader matrix.

The transport adapter is test-only, through the existing cluster client seam. Live cluster factory endpoint/SSRF checks, JWT/tenant authorization, credential access, live provider admission and ownership lifetime are not exercised by this test matrix.

### Stale envelope fixture

service-pages.spec.ts:302-303 now uses HTTP 200 with code 409, data null and message business.service.state_stale. This matches the shared HTTP FailWithData contract at backend/pkg/common/http/response.go:58-64 and FailWithError's resolved machine-key message at lines 104-121.

The assertion at service-pages.spec.ts:312 targets visible Chinese error feedback in .arco-message-error, rather than accidentally matching identical text elsewhere. The form-visible check, retained 3.0.0 input, exact expectedLifecycleVersion 7 payload and raw-i18n-key prohibition remain at lines 313-316. The first() selector narrows duplicate error feedback; it does not bypass the message text, visibility, form or CAS checks. The fixture remains a browser contract test and does not validate production authorization/provider behavior.

Impeccable's table/form interaction gate is retained. This review checked conflict feedback and preserved input/CAS assertions; no final desktop/mobile rendered visual QA was performed under the read-only/no-browser review boundary, and no polish claim is made.

### Reviewed evidence

- continuation-c03-k8s-reader-cause-red.log and receipt: three top-level provider tests failed, package 0.862s, exit 1, CGO=1, race enabled and disposable MySQL required. Real 404 controls passed while non-NotFound provider causes failed. The evidence is retained.
- continuation-c03-k8s-reader-cause-fixed.log and receipt: all three top-level tests passed, package 1.973s, exit 0, same CGO/race/required-MySQL conditions. The full subtest matrix was inspected.
- continuation-c03-ui-envelope-fixed.log and receipt: all eight selected mocked UI cases passed, test output 59.2s, receipt wall time about 60s, exit 0. Coverage includes both cross-tab parent-filter regressions, 125-object pagination/edit, read-only/no optional calls, optional-target denial isolation, structured K8s choices/reset, explicit versioned rebind and stale-input preservation.
- The two provider source hashes match continuation-c03-k8s-reader-cause.hashes.json. Both entries in ui-stale-envelope-source-manifest.json match current source. All three before/after source hashes in the successful UI receipt match current ServiceCatalog, service API and smoke source: five unique bound source files, no mismatch.
- Tests were executed by root; this reviewer only read source, logs, receipts and hashes through the existing Node runtime. No shell, test, database, browser or helper process was launched.
- lsp_diagnostics and ast_grep_search are unavailable in this lane. No gopls process was run under root's no-process boundary. The scoped root race test provides successful package compilation evidence; final language diagnostics remain explicit, not inferred as completed.

### Remaining boundaries and stop condition

Cross-owner target-lifetime coordination and K8s NamespaceBinding/environment/action admission remain open and unwaived. Concrete provider cause suppression is the specific prior gap repaired here; that does not establish those broader owner contracts. Architecture decisions remain with the independent architect/owner lanes.

Final full race, official rebuilt product equivalence, live role/provider acceptance, rendered desktop/mobile acceptance and final candidate/source binding remain open. The successful mocked UI run supersedes the prior failing UI checkpoint for the source hashes recorded in its receipt; it does not supersede the earlier evidence file or imply whole-task completion.

Stop condition for this narrow round: concrete error path and stale fixture inspected; root red-to-green and eight-case mock evidence assessed; exact source binding checked; no new issue found. Round-one and round-two artifacts remain unchanged.

Only this new report artifact was written. No production/test source, goal, commit, database, browser or process lifecycle was changed.

### Machine-readable conclusion

```json
{
  "schema": "pantheon-independent-code-review/v1",
  "taskId": "2026-11-16-ops-service-catalog-usability",
  "reviewRound": 3,
  "reviewerLane": "c03_code_review",
  "recordedAtUtc": "2026-11-17T14:59:58.909Z",
  "verdict": "COMMENT",
  "verdictScope": "NARROW_PROVIDER_CAUSE_AND_STALE_FIXTURE_CHECKPOINT",
  "filesReviewed": 3,
  "issueCounts": {
    "CRITICAL": 0,
    "HIGH": 0,
    "MEDIUM": 0,
    "LOW": 0
  },
  "newFindings": [],
  "priorRoundFourFindings": "ROUND_TWO_CONCLUSION_CARRIED_FORWARD_NOT_RERUN",
  "providerCauseSuppression": "REPAIRED_IN_BOUNDED_REVIEWED_READER",
  "staleFixtureEnvelope": "HTTP_200_CODE_409_MESSAGE_MACHINE_KEY",
  "staleAssertionsPreserved": [
    "Chinese error feedback",
    "form remains visible",
    "desiredVersion input retained",
    "exact lifecycle CAS payload",
    "no visible raw i18n keys"
  ],
  "sourceBinding": {
    "providerFilesMatched": 2,
    "uiReceiptFilesMatched": 3,
    "uniqueFilesMatched": 5
  },
  "evidence": {
    "providerRedTopLevelFailures": 3,
    "providerRedExitCode": 1,
    "providerRedPackageSeconds": 0.862,
    "providerGreenTopLevelPasses": 3,
    "providerGreenExitCode": 0,
    "providerGreenPackageSeconds": 1.973,
    "providerRealMySQL": true,
    "providerCGOEnabled": true,
    "providerRaceEnabled": true,
    "providerReaderMatrix": "404/403/500/network across Deployment/StatefulSet/DaemonSet; in-flight cancellation across all three",
    "serviceHTTPMatrix": "404/403/500/network with Deployment target and no persisted mutation",
    "mockUIPasses": 8,
    "mockUIExitCode": 0,
    "mockUITestSeconds": 59.2,
    "testExecutionOwner": "ROOT_RECEIPTS_AND_LOGS_REVIEWED"
  },
  "diagnostics": {
    "lspDiagnosticsTool": "UNAVAILABLE",
    "gopls": "NOT_RUN_THIS_ROUND_ROOT_NO_PROCESS_BOUNDARY",
    "providerCompilation": "ROOT_CGO_RACE_PACKAGE_TESTS_PASSED"
  },
  "excludedConcurrentScope": "C04 K8s service/namespace/cluster work",
  "openArchitectureContracts": [
    "cross-owner target lifetime coordination",
    "K8s NamespaceBinding/environment/action admission"
  ],
  "openAcceptanceEvidence": [
    "final language diagnostics",
    "full race",
    "official rebuild equivalence",
    "live role/provider acceptance",
    "desktop/mobile rendered acceptance",
    "final candidate/source binding"
  ],
  "fullC03Acceptance": false,
  "mergeReady": false,
  "productionCodeModified": false,
  "testCodeModified": false,
  "newProcessesStarted": false,
  "goalStateModified": false
}
```
