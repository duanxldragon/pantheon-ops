---
doc_type: independent-code-review
round: 1
reviewer_lane: c03_code_review
recorded_at_utc: "2026-11-17T14:33:29Z"
verdict: REQUEST CHANGES
status: preserved-original-round1
files_reviewed: 13
critical: 0
high: 2
medium: 2
low: 0
repository: pantheon-ops
working_tree_head_at_recording: 71c86cfe664ef9fde310cc68efbf2e8992e648f2
candidate_binding: working-tree-review-before-remediation; recording HEAD is not an immutable reviewed source binding
production_code_modified: false
test_code_modified: false
re_review_performed: false
---

This artifact preserves the complete original round-one conclusion delivered to root. Recording time is the artifact persistence time; no new review of concurrent fixes was performed. The original report delivery timestamp was not captured in this lane. Other agents are applying backend/UI fixes; this report does not assess those changes.

Review scope: frontend/src/modules/business/service/ServiceCatalog.tsx; frontend/src/modules/business/service/api.ts; frontend/tests/smoke/business/service/service-pages.spec.ts; frontend/src/modules/business/v12locales/{en-US,zh-CN,ja-JP,ko-KR,fr-FR}.json; backend/modules/business/service/{dto.go,service.go,handler.go,seed.go,instance_update_test.go}. Backend review covers current C03 UpdateInstance, target validation and seed additions. Existing C01 state/delete/reader changes were not attributed to C03. Related owner/route files were read for contract context.

## Code Review Summary

**Recommendation: REQUEST CHANGES**

**Files reviewed:** 13 C03 scope files, plus related owner/route context.  
**Issues:** CRITICAL 0 · HIGH 2 · MEDIUM 2 · LOW 0.

### Issues

1. **[HIGH] Every target rebind requires an unregistered HTTP endpoint.**  
   [api.ts:129](D:/workspace/go/pantheon-platform/pantheon-ops/frontend/src/modules/business/service/api.ts:129) adds `GET /business/service/services/:id`; [ServiceCatalog.tsx:428](D:/workspace/go/pantheon-platform/pantheon-ops/frontend/src/modules/business/service/ServiceCatalog.tsx:428) requires its response before rebinding. [handler.go:49](D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/service/handler.go:49) registers no service-detail GET route. Consequently, the real backend returns 404 and the target form remains blocked. The mock fixture accepts numeric detail URLs at [service-pages.spec.ts:68](D:/workspace/go/pantheon-platform/pantheon-ops/frontend/tests/smoke/business/service/service-pages.spec.ts:68), hiding this mismatch.  
   **Fix:** Register a scoped, authorized owner detail endpoint returning the camelCase HTTP projection, and test the real route. Do not replace the failing lookup with unverified cached data.

2. **[HIGH] Runtime editing can race with target rebinding and commit an incompatible service/instance pair.**  
   The new rebind path reads and locks its Service at [service.go:529](D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/service/service.go:529). Existing `UpdateService` checks incompatible instances at [service.go:331](D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/service/service.go:331) before its update obtains the Service lock. A valid interleaving is:

   - Rebind holds the Service lock after reading runtime `hybrid`, with the stored target still VM.
   - Concurrent runtime edit to `vm` counts zero incompatible instances, then waits for that lock.
   - Rebind commits a K8s target.
   - Runtime edit resumes and commits `vm`, leaving a K8s instance under a VM-only service.

   **Fix:** Put runtime validation and update inside the same owner transaction and acquire the Service lock before counting incompatible instances. Add a controlled real-MySQL concurrency regression. This finding concerns the new mutable-target interaction with an existing writer; it does not attribute C01 state/delete/reader changes to C03. **Evidence:** static interleaving, not dynamically reproduced by this reviewer.

3. **[MEDIUM] Empty filters disable unrelated creation flows.**  
   [ServiceCatalog.tsx:678](D:/workspace/go/pantheon-platform/pantheon-ops/frontend/src/modules/business/service/ServiceCatalog.tsx:678) disables service creation using the filtered application total; [ServiceCatalog.tsx:695](D:/workspace/go/pantheon-platform/pantheon-ops/frontend/src/modules/business/service/ServiceCatalog.tsx:695) similarly disables instance creation using the filtered service total. Searching for a nonexistent application or service and changing tabs blocks creation even when valid parents exist.  
   **Fix:** Determine parent availability independently of table filters, or permit opening the modal and let its lazy owner query establish availability. Cover both cross-tab cases.

4. **[MEDIUM] Production K8s missing-target errors miss the new classification and translations.**  
   [handler.go:280](D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/service/handler.go:280) recognizes `business.service.target_not_found`, while the production reader returns `business.service.k8s_target_not_found` at [k8s/capability.go:40](D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/k8s/capability.go:40). The service propagates that error unchanged. A removed workload therefore gets generic `CodeError`; its key is absent from the five canonical locale sources inspected.  
   **Fix:** Align the narrow missing-target contract, HTTP classification, and translations; add an HTTP regression. Preserve unexpected provider failures. The reader's existing conflation of provider errors requires separate owner review.

### Validation and scope limits

- Spec compliance **fails** on the missing rebind route; normal quality approval is withheld.
- `gopls check` covered all five C03 Go files: **exit 0, no diagnostics**. `lsp_diagnostics` and `ast_grep_search` tools were unavailable; literal pattern inspection found no new hardcoded secrets, injection construction, unsafe HTML, or masking fallback.
- Reviewed the real-MySQL CGO race receipt and log: **13 top-level `TestServiceInstanceUpdate` tests passed**, including zero CAS, replay, concurrent single-winner CAS, atomic replacement, active-reference refusal, locator mismatch, and cancellation. External owner readers are fixtures, so this does not prove production K8s behavior.
- Reviewed frontend typecheck/lint receipts. The mock regressions cannot validate route registration or backend authorization.
- Rendered desktop/mobile verification, live authorization/provider acceptance, full race, rebuild, and final revision binding remain coordinator-owned gaps. No visual-polish claim.
- Target-lifetime coordination and K8s NamespaceBinding/environment/action concerns were surfaced upward for the architecture lane; they remain open and unwaived.

No files, tests, process lifecycle, commits, or goal state were changed during the original review. This persistence action creates only the requested report artifact.

### Machine-readable conclusion

```json
{
  "schema": "pantheon-independent-code-review/v1",
  "taskId": "2026-11-16-ops-service-catalog-usability",
  "reviewRound": 1,
  "reviewerLane": "c03_code_review",
  "recordedAtUtc": "2026-11-17T14:33:29Z",
  "verdict": "REQUEST CHANGES",
  "filesReviewed": 13,
  "issueCounts": { "CRITICAL": 0, "HIGH": 2, "MEDIUM": 2, "LOW": 0 },
  "specCompliance": "FAIL",
  "rootCauseFallbackGuard": "NO_NEW_MASKING_FALLBACK_FOUND",
  "diagnostics": {
    "goplsCheck": { "fileCount": 5, "exitCode": 0, "diagnostics": 0 },
    "lspDiagnosticsTool": "UNAVAILABLE",
    "astGrepSearchTool": "UNAVAILABLE"
  },
  "evidence": {
    "ownerReceipt": ".harness/evidence/2026-11-17-ops-history-audit/continuation-c03-owner-update-fixed.log.receipt.json",
    "ownerLog": ".harness/evidence/2026-11-17-ops-history-audit/continuation-c03-owner-update-fixed.log",
    "ownerTopLevelTestsPassed": 13,
    "ownerTestExecution": "ROOT_EXECUTED_RECEIPT_AND_LOG_REVIEWED",
    "frontendTypecheckReceipt": ".harness/evidence/2026-11-16-ops-service-catalog-usability/ui-typecheck.log.receipt.json",
    "frontendLintReceipt": ".harness/evidence/2026-11-16-ops-service-catalog-usability/ui-scoped-lint.log.receipt.json"
  },
  "findings": [
    {
      "id": "C03-R1-01",
      "severity": "HIGH",
      "file": "frontend/src/modules/business/service/api.ts",
      "line": 129,
      "title": "Target rebinding calls unregistered service-detail endpoint",
      "evidenceType": "STATIC_CONTRACT_MISMATCH"
    },
    {
      "id": "C03-R1-02",
      "severity": "HIGH",
      "file": "backend/modules/business/service/service.go",
      "line": 529,
      "title": "Runtime update and target rebind can commit incompatible final state",
      "evidenceType": "STATIC_CONCURRENCY_INTERLEAVING_NOT_DYNAMICALLY_REPRODUCED"
    },
    {
      "id": "C03-R1-03",
      "severity": "MEDIUM",
      "file": "frontend/src/modules/business/service/ServiceCatalog.tsx",
      "line": 678,
      "title": "Filtered totals disable creation in other tabs",
      "evidenceType": "STATIC_UI_STATE_FLOW"
    },
    {
      "id": "C03-R1-04",
      "severity": "MEDIUM",
      "file": "backend/modules/business/service/handler.go",
      "line": 280,
      "title": "Production K8s missing-target key misses classification and translations",
      "evidenceType": "STATIC_ERROR_CONTRACT_MISMATCH"
    }
  ],
  "architectureConcerns": "SURFACED_UPWARD_NOT_WAIVED",
  "openArchitectureContracts": [
    "cross-owner target lifetime coordination",
    "K8s NamespaceBinding/environment/action validation"
  ],
  "openAcceptanceEvidence": [
    "rendered desktop/mobile",
    "live authorization/provider acceptance",
    "full race",
    "official rebuild",
    "final source binding"
  ],
  "reReviewPerformed": false,
  "productionCodeModified": false,
  "testCodeModified": false,
  "goalStateModified": false
}
```
