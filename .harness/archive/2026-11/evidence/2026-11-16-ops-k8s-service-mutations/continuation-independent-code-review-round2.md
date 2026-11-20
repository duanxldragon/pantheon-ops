---
doc_type: independent-code-review
round: 2
reviewer_lane: c03_code_review
recorded_at_utc: "2026-11-17T16:53:56.148Z"
verdict: APPROVE
status: narrow-c04-r1-01-opaque-port-fix
files_reviewed: 2
critical: 0
high: 0
medium: 0
low: 0
whole_task: false
merge_ready: false
---

## Code Review Summary

**APPROVE for the narrow C04-R1-01 repair.** The update now preserves an existing logical port's appProtocol and prevents new or renamed ports from inheriting unrelated metadata. No new issue was found. This is a code-lane disposition for the prior finding, not whole-task or merge approval.

**Files reviewed:** backend/modules/business/k8s/service/service.go and mutation_contract_test.go. All nine manifest files were hash-bound, but the remaining seven—including k8s_seed.go—were not newly reviewed for approval in this round.

**Issues:** CRITICAL 0 / HIGH 0 / MEDIUM 0 / LOW 0. Prior MEDIUM C04-R1-01 is resolved at the bound source hashes.

### Spec and root-cause repair

At service.go:232 the existing Service is deep-copied. Lines 236-245 copy AppProtocol only from a matching existing port before replacing Spec.Ports at line 246. Named ports match the same unique, nonempty name; their port/protocol can change while retaining that declared logical identity. Unnamed ports require both sides unnamed and the same protocol and port number. Therefore rename, named-to-unnamed, unnamed-to-named, or a new unnamed protocol/port identity does not inherit unrelated appProtocol. API normalization at line 388 rejects duplicate named identities, so a valid existing Service does not introduce ambiguous named matching. The pointer is copied from the deep copy, rather than mutating the original fetched object.

The primary GET, expected UID/resourceVersion check at line 222, fetched-identity PUT at line 247, admission and provider error paths remain intact. This repairs the serialized field loss itself; it adds no retry, alternate provider path, silent success or validation/permission bypass.

### Regression and execution evidence

mutation_contract_test.go:270 sets an existing appProtocol, performs the real Service handler PUT through client-go HTTP serialization, and inspects the simulated provider object. Each case at line 299 checks two provider requests, one mutation, routing fields and changed selector before evaluating preservation at line 302 or absence at line 306.

The five cases cover named selector-only edits, named number/protocol changes, unchanged unnamed identity, renamed ports, and changed unnamed port numbers. The unnamed protocol-change branch is also unambiguously excluded by service.go:239; there is no separate protocol-only unnamed negative subtest. The successful provider controls and field assertions cover the actual prior failure rather than merely checking that the endpoint returns success.

Root's preserved opaque-port red has three preservation failures and two passing new-identity controls, Service package 0.780s. Root's final green receipt records exit 0, CGO_ENABLED=1, required disposable MySQL, GOFLAGS=-p=1 -mod=readonly and filtered go test -race -count=1 -timeout=10m ./modules/business/k8s/... -v -run ^TestK8sService(Mutation|PermissionRoutes). All ten mutation and one permission top-level tests pass; all five opaque-port subtests pass. Service package 7.226s; root K8s package 1.453s. These are bounded simulated-provider tests, not live Kubernetes/kubectl acceptance or an unfiltered full-race pass.

The reviewer ran gopls check on both narrowed modified Go files: exit 0, no diagnostics. Scoped git diff --check also exited 0. Named lsp_diagnostics and ast_grep_search tools are unavailable; manual source inspection and gopls are explicitly recorded substitutes. No long test, database change or browser/runtime lifecycle action was performed by this reviewer.

All nine current source hashes match continuation-backend-opaque-port-fixed-source.json. The manifest's awaiting-root status was left unchanged. Source binding includes seed and other owner files without expanding this narrow approval to their behavior.

### Remaining boundaries and stop condition

The round-one report remains unchanged. Its concrete field-preservation finding is resolved by this new report at the supplied hashes. Namespace admission-to-provider lifetime coordination stays with the coordinator/architect lane. Full unfiltered race, official rebuild equivalence, seed/i18n/UI permission synchronization, live role/environment/provider acceptance, final rendered C04 acceptance and maintainer final gate remain open and unwaived.

Stop condition: repair and five-case regression inspected; red/green and source binding checked; diagnostics complete for both narrowed files; no additional issue found. Only this report is exclusively created.

### Machine-readable conclusion

```json
{
  "schema": "pantheon-independent-code-review/v1",
  "taskId": "2026-11-16-ops-k8s-service-mutations",
  "reviewRound": 2,
  "reviewerLane": "c03_code_review",
  "recordedAtUtc": "2026-11-17T16:53:56.148Z",
  "verdict": "APPROVE",
  "verdictScope": "NARROW_C04_R1_01_OPAQUE_PORT_FIX",
  "filesReviewed": 2,
  "issueCounts": {
    "CRITICAL": 0,
    "HIGH": 0,
    "MEDIUM": 0,
    "LOW": 0
  },
  "newFindings": [],
  "priorFinding": {
    "id": "C04-R1-01",
    "status": "RESOLVED",
    "implementationFile": "backend/modules/business/k8s/service/service.go",
    "implementationLine": 236,
    "regressionFile": "backend/modules/business/k8s/service/mutation_contract_test.go",
    "regressionLine": 270
  },
  "portIdentityRule": {
    "named": "Unique nonempty name; routing number/protocol changes retain metadata for that logical port",
    "unnamed": "Both unnamed and equal protocol + port; changes form a new identity",
    "renamed": "No match; do not inherit",
    "new": "No match; do not inherit"
  },
  "sourceBinding": {
    "manifest": ".harness/evidence/2026-11-16-ops-k8s-service-mutations/continuation-backend-opaque-port-fixed-source.json",
    "filesMatched": 9,
    "files": [
      {
        "path": "backend/modules/business/k8s/service/service.go",
        "sha256": "38956b4db476fa365fa2daf49c42bab5c8e52184a19c3312c1918bfb25e35f56",
        "actualSha256": "38956b4db476fa365fa2daf49c42bab5c8e52184a19c3312c1918bfb25e35f56",
        "matches": true
      },
      {
        "path": "backend/modules/business/k8s/service/handler.go",
        "sha256": "a6240fdbcef3287bb5a8087c083bb90ee49fc0542917f8db7ab34e16a7ac1765",
        "actualSha256": "a6240fdbcef3287bb5a8087c083bb90ee49fc0542917f8db7ab34e16a7ac1765",
        "matches": true
      },
      {
        "path": "backend/modules/business/k8s/service/mutation_contract_test.go",
        "sha256": "8c5759893970cd112585491c4d38b0166d3f049a8b5f186562856921905c28d4",
        "actualSha256": "8c5759893970cd112585491c4d38b0166d3f049a8b5f186562856921905c28d4",
        "matches": true
      },
      {
        "path": "backend/modules/business/k8s/namespace/namespace_service.go",
        "sha256": "2af0d803d4ab5b477b19222645ab3a8c48174fa183413ce7d449ce3342cfa9d3",
        "actualSha256": "2af0d803d4ab5b477b19222645ab3a8c48174fa183413ce7d449ce3342cfa9d3",
        "matches": true
      },
      {
        "path": "backend/modules/business/k8s/cluster/cluster_service.go",
        "sha256": "f50af80184bdee9b8e50555c9a33dcef6581c3b35f84c68eaa293cdbb2f27bd6",
        "actualSha256": "f50af80184bdee9b8e50555c9a33dcef6581c3b35f84c68eaa293cdbb2f27bd6",
        "matches": true
      },
      {
        "path": "backend/modules/business/k8s/k8s.go",
        "sha256": "fccfbcf7afc471632a29f43d950998e74193147682f247221a77c9b0c475dbb2",
        "actualSha256": "fccfbcf7afc471632a29f43d950998e74193147682f247221a77c9b0c475dbb2",
        "matches": true
      },
      {
        "path": "backend/modules/business/k8s/permission_routes.go",
        "sha256": "ef2798ea5f1a70790b3a58d1d00d840c2eff92fd6147f1d155b34348e951c132",
        "actualSha256": "ef2798ea5f1a70790b3a58d1d00d840c2eff92fd6147f1d155b34348e951c132",
        "matches": true
      },
      {
        "path": "backend/modules/business/k8s/permission_routes_test.go",
        "sha256": "d0ec4e8228970b5a35384488127d5754c226dd147d81d36c854ac8eefec85c39",
        "actualSha256": "d0ec4e8228970b5a35384488127d5754c226dd147d81d36c854ac8eefec85c39",
        "matches": true
      },
      {
        "path": "backend/modules/business/k8s/k8s_seed.go",
        "sha256": "eaa84c72137075289a492412501c1c5276695a0fa7f2b11e89931cbda49c290a",
        "actualSha256": "eaa84c72137075289a492412501c1c5276695a0fa7f2b11e89931cbda49c290a",
        "matches": true
      }
    ]
  },
  "diagnostics": {
    "lspDiagnosticsTool": "UNAVAILABLE",
    "astGrepSearchTool": "UNAVAILABLE",
    "substitute": "gopls check on the two changed narrow-review files and manual scoped source inspection",
    "goplsExitCode": 0,
    "goplsDiagnostics": 0,
    "scopedDiffCheckExitCode": 0
  },
  "rootCauseGuard": "PASS_REAL_SERIALIZED_FIELD_PRESERVATION_NO_RETRY_OR_BYPASS",
  "evidence": {
    "executionOwner": "ROOT_LOG_AND_RECEIPT_REVIEWED",
    "redLog": ".harness/evidence/2026-11-17-ops-history-audit/continuation-c04-opaque-port-red.log",
    "redTopLevelFailures": 1,
    "redPreservationFailures": 3,
    "redNewIdentityControlsPassed": 2,
    "redServiceSeconds": 0.78,
    "greenLog": ".harness/evidence/2026-11-17-ops-history-audit/continuation-c04-opaque-port-fixed.log",
    "greenReceipt": ".harness/evidence/2026-11-17-ops-history-audit/continuation-c04-opaque-port-fixed.log.receipt.json",
    "greenExitCode": 0,
    "mutationTopLevelPasses": 10,
    "permissionTopLevelPasses": 1,
    "opaquePortSubtestPasses": 5,
    "servicePackageSeconds": 7.226,
    "k8sPackageSeconds": 1.453,
    "raceEnabled": true,
    "cgoEnabled": true,
    "requiredDisposableMySQL": true,
    "provider": "BOUNDED_HTTP_SIMULATION_WITH_REAL_CLIENT_GO_SERIALIZATION"
  },
  "coverageLimit": "Unnamed protocol-change refusal to inherit verified statically; existing negative subtest changes port number rather than protocol",
  "excludedScope": [
    "k8s_seed.go hash-bound only; seed/i18n/permission synthesis not re-reviewed here",
    "Other seven manifest files source-bound only; prior lane conclusion carried forward",
    "Whole C04 UI/render/live-provider/role acceptance"
  ],
  "openArchitectureContracts": [
    "Namespace admission-to-provider cross-owner lifetime coordination remains outside this narrow code verdict"
  ],
  "wholeTask": false,
  "fullC04Acceptance": false,
  "mergeReady": false,
  "productionCodeModified": false,
  "testCodeModified": false,
  "goalStateModified": false,
  "reportCreationMode": "CREATE_NEW_NO_OVERWRITE"
}
```
