---
doc_type: independent-code-review
round: 1
reviewer_lane: c03_code_review
recorded_at_utc: "2026-11-17T15:28:39.855Z"
verdict: REQUEST CHANGES
status: bounded-c04-backend-contract-review
files_reviewed: 8
critical: 0
high: 0
medium: 1
low: 0
full_c04_acceptance: false
merge_ready: false
---

## Code Review Summary

**Recommendation: REQUEST CHANGES for the bounded C04 backend candidate.** One MEDIUM correctness finding remains: a normal Service edit silently clears an existing port's appProtocol. The selected identity, cancellation, owner-admission and exact-route permission tests pass. This report does not approve the whole C04 task, concurrent UI/seed work, live Kubernetes behavior or merge readiness.

**Files reviewed:** eight exact files bound by continuation-backend-provider-cause-source.json: service/service.go, service/handler.go, service/mutation_contract_test.go, namespace/namespace_service.go, cluster/cluster_service.go, k8s.go, permission_routes.go and permission_routes_test.go, all beneath backend/modules/business/k8s/.

**Issue counts:** CRITICAL 0 / HIGH 0 / MEDIUM 1 / LOW 0.

### Finding C04-R1-01 — MEDIUM: existing port appProtocol is cleared

**File:** backend/modules/business/k8s/service/service.go:233. Related source: ServicePort DTO at line 32, toItem at line 281, and toServicePorts at line 309.

A valid existing Service can have spec.ports[0].appProtocol set to, for example, kubernetes.io/h2c. The read DTO omits that field. When the caller changes only selector and round-trips the supported port fields with matching UID/resourceVersion, UpdateWithContext deep-copies the existing Service but then replaces its whole Spec.Ports slice with newly constructed ports. Those ports omit AppProtocol, so the submitted update clears the existing value. Protocol-aware controllers can consequently change their handling although the user only edited a selector. DeepCopy preserves object-level fields but does not preserve fields inside a slice that is subsequently replaced.

**Fix:** retain non-editable fields from the matching existing port under a defined identity rule, or expose appProtocol as an explicit editable field throughout the API/UI round-trip. Extend the existing provider fixture with an appProtocol value and assert it survives a selector edit and the serialized PUT. The minimal repair should keep the current UID/resourceVersion checks and avoid a retry or alternate update path.

**Evidence status:** deterministic static source trace, not independently reproduced against a running provider. The current preservation test at mutation_contract_test.go:255 checks ClusterIP, labels, selector, port and named targetPort, but its fixture at line 163 does not contain AppProtocol. Its passing result therefore does not refute this finding. This is an unintended field-preservation defect, not a speculative style concern.

### Stage 1: spec and scope compliance

The original task packet and continuation plan retain L2 business/k8s ownership, existing client/owner contracts and exact permissions. All eight reviewed files remain in the authorized business boundary; no shared-system or foundation implementation is part of this review. Concurrent k8s_seed.go, frontend locales/forms/smoke and governance changes are excluded.

- Lists/detail publish resourceVersion and UID at service.go:301-302, and retain numeric or named targetPort at line 284.
- PUT checks required expected resourceVersion at line 205 and optional update UID at line 222. It submits the fetched object's identity to Kubernetes at line 234; a provider-side change after GET is still rejected by the API's optimistic concurrency contract. Existing ClusterIP, ClusterIPs, metadata and other object-level fields survive the deep copy, subject to finding C04-R1-01 for port-level fields.
- Public DELETE requires expected resourceVersion and UID at lines 257-260 and submits both Kubernetes Preconditions at line 271. Legacy unversioned Delete returns an explicit validation error; no HTTP identity-fetch fallback was added.
- Namespace writes call the owner capability at line 332. namespace_service.go:87 obtains an authorized Cluster projection, line 91 reads the explicit binding, line 95 compares nonzero business scope and environment, and line 98 enforces the action before provider requests. The Cluster seam at cluster_service.go:133 carries request context rather than adding direct cross-owner table reads in Service.
- The permission registry maps list/view to GET only, create to POST, update to PUT and delete to DELETE at permission_routes.go:8-23. k8s.go:94 registers those bindings through the existing registry. It does not assign roles or grant cluster/node routes.

Stage 1 is a partial pass: the main mutation contract is implemented and evidenced, but the promised field preservation still has the concrete defect above. Final full-task acceptance remains open for the explicitly excluded work and live evidence.

### Root-cause guard and diagnostics

No newly introduced broad compatibility shim, swallowed provider error, silent success, retry-around-conflict path or authorization bypass was found in the scoped additions. classifyProviderError at service.go:335 retains the original provider error together with a semantic sentinel using multiple %w values; unclassified errors return unchanged. handler.go:153 handles exact legacy owner keys and sends unknown failures to FailWithError, preserving their diagnostic path. Its fallback argument selects an error message; it does not hide a successful/failed primary mutation. The narrow test proxy does not forward requests or weaken production SSRF checks.

The pre-existing Cluster credential alternate lookup behavior at cluster_service.go:473/477 is outside the new C04 projection seam and was not approved or claimed repaired here. Namespace admission/provider lifetime coordination is a separate architectural concern handed to the coordinator/architect, not absorbed into this code lane's design verdict.

The named lsp_diagnostics and ast_grep_search tools are unavailable in this lane. The reviewer ran gopls check on all eight modified Go files from backend/: exit 0, no diagnostics. Manual source and scoped text-pattern inspection covered empty catches, hardcoded API keys, console logging and fallback/best-effort markers. The scoped git diff --check exited 0. No new secret literal or injection/XSS path was found in the selected Go additions; parameterized owner queries and the existing client factory remain in use. Normal style approval is withheld because the correctness finding remains.

### Reviewed execution evidence and source binding

Root executed the tests; this reviewer inspected logs, receipts, source and hashes and did not execute tests or mutate the database/browser/runtime lifecycle.

- The effective provider-bound red has a passing allowed-create control and real provider mutation counters, then fails on missing identity fields/PUT, unguarded deletion and wrong binding scope/environment. The first two preserved red runs were blocked by production SSRF and are fixture-inadequate for mutation proof. They are not promoted into effective red evidence.
- continuation-c04-mutation-provider-cause-fixed.log and receipt record exit 0, CGO_ENABLED=1, required disposable real MySQL, GOFLAGS=-p=1 -mod=readonly and go test -race -count=1 -timeout=10m ./modules/business/k8s/... -v -run ^TestK8sService(Mutation|PermissionRoutes). Receipt interval: 2026-11-17T15:21:06.9492362Z through 2026-11-17T15:22:16.7779477Z.
- All nine mutation and one exact-permission top-level tests passed. Service package: 5.816s; root K8s package: 1.315s. Other packages marked no tests to run are compilation evidence for this filtered command, not an unfiltered compatibility/race pass.
- The deterministic GET→PUT race test at mutation_contract_test.go:282 asserts two provider requests, one conflict, no mutation, and retention of the external selector/revision. Delete cases at line 297 cover matching identity, stale revision, recreated-name UID and missing identity. Admission cases at line 331 cover all three mutations for cross-department, binding-scope, binding-environment and missing-action denial with zero provider calls. Cancellation cases at line 366 prove already-canceled requests stop before the provider; they do not claim an in-flight provider cancellation matrix.
- Provider error cases at line 222 check actual client-go 404/403/500 causes and expected envelopes. The read detail test verifies named targetPort and identity. List identity follows the same mapper but is not a separately selected positive HTTP list test.
- The permission test at permission_routes_test.go:15 registers the actual module, verifies concrete action routes and evaluates a controlled exact-method Casbin policy including node-route denial. Mutation fixtures intentionally use nil handler RBAC. These tests do not prove production token authentication, real user roles, policy reconciliation or environment-level authorization end to end.
- All eight current source hashes matched the supplied manifest at 2026-11-17T15:28:39.855Z. Its recorded validationStatus remains pending-provider-cause-race; this reviewer did not rewrite it. The subsequent successful receipt was independently inspected.

The provider fixture is a bounded HTTP simulation using real owner MySQL and real client-go serialization. It is not a live Kubernetes API server and cannot prove Kubernetes allocation/defaulting/validation, real kubectl interference or production authorization.

### Remaining boundaries and stop condition

Fix and regress C04-R1-01 before code-lane approval. Architecture retains the namespace admission-to-provider lifetime concern: there is no cross-owner lock; the plan explicitly records that limit. Full unfiltered CGO race, official-lock rebuilt-product equivalence, real principal/role/environment acceptance, live Kubernetes/kubectl interference, frontend/menu/i18n/seed/smoke synchronization, rendered desktop/mobile acceptance, final candidate/source binding and the maintainer final gate remain open and unwaived.

Stop condition for this bounded round is satisfied: all eight files and relevant contract tests inspected, genuine red/green distinguished, language diagnostics and diff check completed, source binding checked, one concrete finding documented and handed to the coordinator. Only this new report artifact is written with exclusive creation; previous evidence, production/test source, goals, commits, database and browser/runtime lifecycle remain unchanged.

### Machine-readable conclusion

```json
{
  "schema": "pantheon-independent-code-review/v1",
  "taskId": "2026-11-16-ops-k8s-service-mutations",
  "reviewRound": 1,
  "reviewerLane": "c03_code_review",
  "recordedAtUtc": "2026-11-17T15:28:39.855Z",
  "verdict": "REQUEST CHANGES",
  "verdictScope": "BOUNDED_C04_BACKEND_CONTRACT",
  "filesReviewed": 8,
  "issueCounts": {
    "CRITICAL": 0,
    "HIGH": 0,
    "MEDIUM": 1,
    "LOW": 0
  },
  "newFindings": [
    {
      "id": "C04-R1-01",
      "severity": "MEDIUM",
      "file": "backend/modules/business/k8s/service/service.go",
      "line": 233,
      "title": "Existing port appProtocol is silently cleared by normal edits",
      "proof": "STATIC_SOURCE_TRACE_NOT_DYNAMICAL_REPRODUCTION",
      "trigger": "Read existing Service containing spec.ports[].appProtocol, change selector while round-tripping supported DTO port fields, PUT with matching UID/resourceVersion.",
      "cause": "ServicePort DTO and toItem/toServicePorts omit AppProtocol; Update replaces the complete Spec.Ports slice.",
      "fix": "Preserve non-editable existing port fields under a defined port-identity match or include appProtocol explicitly in the editable round-trip; extend the provider fixture to prove existing appProtocol survives a selector edit."
    }
  ],
  "stage1": "PARTIAL_PASS_FIELD_PRESERVATION_FINDING_REMAINS",
  "rootCauseGuard": "NO_NEW_MASKING_FALLBACK_FOUND",
  "sourceBinding": {
    "manifest": ".harness/evidence/2026-11-16-ops-k8s-service-mutations/continuation-backend-provider-cause-source.json",
    "manifestValidationStatus": "pending-provider-cause-race",
    "filesMatched": 8,
    "hashes": [
      {
        "path": "backend/modules/business/k8s/service/service.go",
        "expectedSha256": "c570f777cc02e6fa29a4229d8762b0661020ac6df9aa26cbf3abda0e7f9555d5",
        "actualSha256": "c570f777cc02e6fa29a4229d8762b0661020ac6df9aa26cbf3abda0e7f9555d5",
        "matches": true
      },
      {
        "path": "backend/modules/business/k8s/service/handler.go",
        "expectedSha256": "a6240fdbcef3287bb5a8087c083bb90ee49fc0542917f8db7ab34e16a7ac1765",
        "actualSha256": "a6240fdbcef3287bb5a8087c083bb90ee49fc0542917f8db7ab34e16a7ac1765",
        "matches": true
      },
      {
        "path": "backend/modules/business/k8s/service/mutation_contract_test.go",
        "expectedSha256": "946771a3d29ced8fc357e59c3e5bb8444e4b0dfa7bde67f0c1651f10b885c98b",
        "actualSha256": "946771a3d29ced8fc357e59c3e5bb8444e4b0dfa7bde67f0c1651f10b885c98b",
        "matches": true
      },
      {
        "path": "backend/modules/business/k8s/namespace/namespace_service.go",
        "expectedSha256": "2af0d803d4ab5b477b19222645ab3a8c48174fa183413ce7d449ce3342cfa9d3",
        "actualSha256": "2af0d803d4ab5b477b19222645ab3a8c48174fa183413ce7d449ce3342cfa9d3",
        "matches": true
      },
      {
        "path": "backend/modules/business/k8s/cluster/cluster_service.go",
        "expectedSha256": "f50af80184bdee9b8e50555c9a33dcef6581c3b35f84c68eaa293cdbb2f27bd6",
        "actualSha256": "f50af80184bdee9b8e50555c9a33dcef6581c3b35f84c68eaa293cdbb2f27bd6",
        "matches": true
      },
      {
        "path": "backend/modules/business/k8s/k8s.go",
        "expectedSha256": "fccfbcf7afc471632a29f43d950998e74193147682f247221a77c9b0c475dbb2",
        "actualSha256": "fccfbcf7afc471632a29f43d950998e74193147682f247221a77c9b0c475dbb2",
        "matches": true
      },
      {
        "path": "backend/modules/business/k8s/permission_routes.go",
        "expectedSha256": "ef2798ea5f1a70790b3a58d1d00d840c2eff92fd6147f1d155b34348e951c132",
        "actualSha256": "ef2798ea5f1a70790b3a58d1d00d840c2eff92fd6147f1d155b34348e951c132",
        "matches": true
      },
      {
        "path": "backend/modules/business/k8s/permission_routes_test.go",
        "expectedSha256": "d0ec4e8228970b5a35384488127d5754c226dd147d81d36c854ac8eefec85c39",
        "actualSha256": "d0ec4e8228970b5a35384488127d5754c226dd147d81d36c854ac8eefec85c39",
        "matches": true
      }
    ]
  },
  "diagnostics": {
    "lspDiagnosticsTool": "UNAVAILABLE",
    "astGrepSearchTool": "UNAVAILABLE",
    "substitute": "gopls check on all eight files; scoped manual source/pattern inspection",
    "goplsExitCode": 0,
    "goplsDiagnostics": 0,
    "scopedDiffCheckExitCode": 0
  },
  "evidence": {
    "greenLog": ".harness/evidence/2026-11-17-ops-history-audit/continuation-c04-mutation-provider-cause-fixed.log",
    "greenReceipt": ".harness/evidence/2026-11-17-ops-history-audit/continuation-c04-mutation-provider-cause-fixed.log.receipt.json",
    "executionOwner": "ROOT_RECEIPT_AND_LOG_REVIEWED",
    "greenExitCode": 0,
    "mutationTopLevelPasses": 9,
    "permissionTopLevelPasses": 1,
    "servicePackageSeconds": 5.816,
    "k8sRootPackageSeconds": 1.315,
    "raceEnabled": true,
    "cgoEnabled": true,
    "requiredDisposableMySQL": true,
    "provider": "BOUNDED_HTTP_SIMULATION_REAL_CLIENT_GO_SERIALIZATION",
    "effectiveRed": ".harness/evidence/2026-11-17-ops-history-audit/continuation-c04-mutation-provider-bound-red.log",
    "effectiveRedAllowedControl": "PASSED",
    "initialTwoReds": "SSRF_BLOCKED_FIXTURE_INADEQUATE_NOT_MUTATION_PROOF"
  },
  "architectureHandoff": [
    "Namespace admission and provider mutation have no cross-owner lifetime lock; plan explicitly defers additional locking. Coordinator/architect own the design verdict."
  ],
  "excludedScope": [
    "Concurrent C04 frontend and k8s_seed.go work",
    "C03 owner changes previously reviewed",
    "Full runtime JWT/role reconciliation and live provider acceptance"
  ],
  "openAcceptanceEvidence": [
    "appProtocol preservation regression",
    "full unfiltered CGO race",
    "official-lock rebuilt-product equivalence",
    "live Kubernetes/kubectl interference acceptance",
    "real principal/role/environment acceptance",
    "frontend i18n/menu/seed/smoke synchronization and rendered desktop/mobile evidence",
    "final candidate/source binding and maintainer final gate"
  ],
  "fullC04Acceptance": false,
  "mergeReady": false,
  "productionCodeModified": false,
  "testCodeModified": false,
  "goalStateModified": false,
  "reportCreationMode": "CREATE_NEW_NO_OVERWRITE"
}
```
