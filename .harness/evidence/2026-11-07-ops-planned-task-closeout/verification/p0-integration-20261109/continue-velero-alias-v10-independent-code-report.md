# Independent CODE review: sealed Velero alias v10

**APPROVE for the exact sealed CODE scope.** No findings requiring changes were identified. Five helpers reviewed; CRITICAL 0, HIGH 0, MEDIUM 0, LOW 0. This is a fresh independent code/spec/security decision; predecessor or author approvals were not adopted. The ARCH lane and runtime/P0 acceptance remain separate.

## Spec compliance and root-cause guard

The schema 11 request and handoff match the externally supplied SHA256 values, and the complete five-source map matches actual file bytes. The scope is the existing local Ops acceptance recovery lane; no platform/system/product implementation or permission policy was changed by this review.

The repair addresses the demonstrated bulk-output size contract directly. The native CRD projection is unfiltered and retains typed list kind/resourceVersion, ordered complete UID coverage and every CRD name/UID/group/label map, including absent versus empty optional fields. The strict parser rejects malformed UTF8/JSON, duplicate JSON keys, nonfinite constants, wrong types/arity, duplicate global names/UIDs, invalid identities, missing/reordered coverage and extra documents. Owner/group/name sibling detection requires the exact original owned CRD set. See `.local/ops-acceptance/continue-velero-alias-v10-contract.py:470` and `:498`, and `continue-velero-alias-v10-remote.py:39`.

The production stdout cap remains 4 MiB, stderr must be empty, request timeout remains 20 seconds and process timeout 30 seconds. The full individual object validation loop still checks 22 objects, including complete CRD spec/API version/annotations/labels/UID and Established/non-Terminating conditions. The full cluster RBAC gate remains. Independent AST comparison after version-token normalization confirms 25 preexisting remote functions unchanged, only the inventory portion of `objects()` changed, and both the individual-object and RBAC loops identical; only `crd_inventory()` was added. See `continue-velero-alias-v10-remote.py:200`.

The root-owned native control receipt is pinned atSHA256`51c357a0024718f09e95411b4da2de9fde768ad145c053a68851894fbad10714`. Its actual 53 rows/13 owned comparison reports full typed metadata, labels/absence, ordered UID coverage and resourceVersion/absence equality: 12,765 projected bytes versus 6,540,046 full bytes. Its associated comparison helper and upstream Kubernetes printer sources were inspected. The 8 MiB cap exists only in that read-only full-response memory control. The production change introduces no failure-masking alternate path or swallowed-error success; the root-cause guard passes.

## Security, approval and failure boundaries

Current request readback preserves 443 protected assets, exact 43 remote baseline files and canonical 22 object inventory. Consumed v9 diagnostic failure and COMPLETE inspection adoption allow only the two exact uploaded helpers; neither failed receipt nor inspection becomes success/replay authorization. See `continue-velero-alias-v10-contract.py:591` and `:826`.

Both final independent report decisions, exact request/hash map, and ARCH CLEAR are required before connection. Successful predecessor request/source/owner/phase/freshness checks remain at effect boundaries; failed or attempted phases require inspection. Exclusive helper creation and durable effect intents/outcomes preserve uncertainty after upload/effect failure. The independent additional checks exercised current approval acceptance, rejection of missing/foreign/malformed/partial/non-APPROVE reports, duplicate fences and unsupported `json report` fences, an exclusive upload creation race, and rejection of failed v9 under both predecessor bindings. See `continue-velero-alias-v10-root.py:93`, `:308`, `:326` and `:429`.

Broad exception handlers retain failure exit state, safe error keys and inspection/uncertainty evidence. The sole empty handler recognizes `FileNotFoundError` for exclusive creation. Dynamic loading/adaptation is restricted to pinned captured source. No literal credential assignment or `shell=True` call was found. Raw projected label/schema/annotation/status bodies are not emitted to telemetry; individual Secret bodies are neither hashed nor retained there, and raw stderr is not published. This preserves the primary failure evidence without exposing its private body.

## Performed verification

All execution was local. The isolated review runner relocated only fixture storage/source lookup and added write/network/process guards; immutable helpers and assertions remained unchanged. Existing preparation assertions captured output in BytesIO; no generator entrypoint or disk preparation was run. Root/remote dispatcher functions were exercised only through local fixtures/mocks.

- Full inherited/current suite: 373 executions, 320 unique regressions, 16 new v10 tests, 52 inherited forgery cases; 0 failures/errors/skips. `full-second.log`, SHA256`fa17733e9958b1f28d1e938575be3490bc48489e67a123ea2f93ea3f8fd3319c`.
- Focused projection/preparation readback: 17 tests pass. `focused.log`, SHA256`f03acf11f31348bd0f6d346cdd90a1df6accd7e42c167c2dc89b1830a3b291ab`.
- Additional independent local boundary checks: 16 pass. `independent-checks.json`, SHA256`3997a88c12e28bc31810b826ac60a84cdb4a269bc70d30263c4ca14ae6c82b0c`.
- Actual Pyright 1.1.414 stdio LSP on all five exact sources: 0 errors/0 warnings, stable hashes. `artifacts/continue-velero-alias-v10-lsp-actual-20261111T054107692543.json`, SHA256`03e376dde68a5a9dd2712056169c507507cc8eadab1655b3f76851dad6e215f8`.
- Independent Python AST scan/delta and all 443 protected/five-source hashes passed before closeout. `static-check.json`, SHA256`264f61eac39a7154039abdb573a5f00d4cbfda693763d0dfca0dbd54baffd0d2`.

CodeGraph, `lsp_diagnostics` and `ast_grep_search` wrappers were not exposed in this session; real stdio LSP and Python AST supplied the diagnostic/structural checks. Git status/diff were inspected, but these ignored local helpers require the exact sealed hash map as the review baseline.

Initial reviewer harness evidence remains: `full.log`, `lsp.log`, `lsp-second.log`, `runner-initial-failures.json` and the first synthetic approval scratch directory. The initial full run was interrupted because inherited `tempfile(dir=LOCAL)` repeatedly hit the audit write guard on Windows; redirecting explicit temporary directories to this new review directory enabled the clean full run. The initial LSP whitelist needed the Windows command-string form. The first extra approval check patched the test contract rather than the distinct root contract; only that scratch patch target was corrected. No protected/source/candidate/request/evidence file was modified by these corrections.

## Inspected evidence and limits

The root native projection/consumed-v9/upstream receipts and author's postseal readback were inspected as pinned evidence, not independently rerun remotely. Author readback SHA256`b454048216f6f6fa5da4981eb96e5e73791611a8dd07f19da5b7dc4d8895837e` is supporting evidence only. The reviewer never executed a remote phase, uploader, root phase CLI, generator CLI or credentialed runtime acceptance. No P0 approval or runtime success is claimed. This CODE lane is complete for these exact bytes; any changed request/source requires a new binding and review.

```json
{
  "lane": "CODE",
  "decision": "APPROVE",
  "independent": true,
  "requestSHA256": "935242362fc459942bcef49e955d6216eee95438a25121cf1eda443e390fcace",
  "scriptHashes": {
    ".local/ops-acceptance/continue-velero-alias-v10-contract.py": "56b4932709872f6ef9f34cdf1a88a5c0b952ddd9d2c04c1e15c08adb4efa1a33",
    ".local/ops-acceptance/continue-velero-alias-v10-remote.py": "0d95ec93fa7c292cb0a653d6e35dadb35196c90ed23d117a8f85b77d32450c59",
    ".local/ops-acceptance/continue-velero-alias-v10-root.py": "8283a467f24ef6bf81f44374a8a33342ab6cafb1afe5d627602c6b8c1ca729a4",
    ".local/ops-acceptance/continue-velero-alias-v10-generate.py": "4c9728711768d0111043c63160820d0f1993ce8e92842bb97594db92f45a36fe",
    ".local/ops-acceptance/continue-velero-alias-v10-test.py": "dcb4041a11bcab07ff55800f65ed08325fc477db3a15184b320dee9438a5a869"
  },
  "filesReviewed": 5,
  "issues": [],
  "severityCounts": {
    "CRITICAL": 0,
    "HIGH": 0,
    "MEDIUM": 0,
    "LOW": 0
  },
  "specCompliance": "PASS",
  "rootCauseGuard": "PASS",
  "performedChecks": {
    "fullRegressionExecutions": 373,
    "fullUniqueRegressions": 320,
    "newV10UniqueTests": 16,
    "inheritedForgeryCases": 52,
    "focusedTests": 17,
    "additionalIndependentChecks": 16,
    "failures": 0,
    "errors": 0,
    "skips": 0,
    "actualLSP": {
      "server": "Pyright",
      "version": "1.1.414",
      "protocol": "stdio LSP",
      "files": 5,
      "errors": 0,
      "warnings": 0
    },
    "protectedAssetsStable": 443,
    "sourceHashesStable": true,
    "remoteBaselineFiles": 43
  },
  "remoteExecuted": false,
  "P0Approved": false,
  "reviewEvidence": "verification.json",
  "reviewEvidenceSHA256": "a60b41ea92b3abd6b2b845a9bde4d8099ce6a74b4f3a3f88fa12881a2b7e5693",
  "limitations": [
    "Actual native projection and consumed-v9 observations were inspected as hash-pinned root evidence; reviewer did not contact remote host.",
    "No root phase CLI, generator CLI, uploader, production mutation or credentialed runtime acceptance was executed.",
    "Independent ARCH decision and real P0/runtime acceptance are outside this CODE lane."
  ]
}
```
