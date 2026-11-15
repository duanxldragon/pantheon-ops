Independent ARCH decision: **APPROVE**. Architectural status: **CLEAR**. No unresolved architectural findings were identified for the exact sealed v10 five-helper scope.

This approves the helper design only. Root must still obtain the separate independent CODE approval and establish phased runtime evidence; P0 acceptance remains pending. This lane wrote no files.

## Analysis

1. **The repair changes bulk representation while preserving owned-object semantics.** The unfiltered native command retains the 4 MiB output limit, empty-stderr requirement, 20-second request timeout, and 30-second process timeout (`.local/ops-acceptance/continue-velero-alias-v10-remote.py:24`, `:39`). Full reads of all 22 original objects still enforce identity, labels, policy, API version, and CRD Established/non-Terminating conditions before the projected inventory check (`:200`, `:209`, `:221`, `:230`, `:237`). Independent AST comparison found all 25 other existing remote functions unchanged after version normalization.

2. **The strongest visibility counterargument is addressed.** Projection omits foreign CRD schemas and annotations, but retains every row’s name, UID, group, all labels, and label presence. Parsing rejects duplicate identities, invalid types, malformed JSON, and incomplete or reordered UID coverage (`.local/ops-acceptance/continue-velero-alias-v10-contract.py:458`, `:470`, `:495`). Any Velero-group, expected-name, or matching-owner sibling must belong to the exact original owned name→UID set (`:498`). The inspected read-only control compares complete typed metadata against the full native list, including resourceVersion and absence (`.local/ops-acceptance/post-config-velero-v10/remote-crd-projection.py:41`, `:67`, `:95`). Its sealed receipt records equality for 53 total rows and 13 owned CRDs.

3. **Failed-v9 adoption preserves history without granting replay.** Adoption requires the exact consumed failed diagnostic, COMPLETE inspection, zero unknown entries, and explicit denial of continuation/replay. Only the two fully recorded v9 uploads extend the unchanged 41-file baseline to 43 files (`.local/ops-acceptance/continue-velero-alias-v10-contract.py:591`). Root preserves the canonical 22-object inventory and inherited seals (`.local/ops-acceptance/continue-velero-alias-v10-root.py:43`, `:73`). Success predecessors require current request/source bindings, successful nested receipts, and freshness (`.local/ops-acceptance/continue-velero-alias-v10-contract.py:826`). Both actual v9 failure/inspection receipts were independently rejected as v10 success predecessors.

4. **Effects and approval boundaries remain explicit.** Repair adds only the two absent normalized image aliases and verifies the exact image-list delta (`.local/ops-acceptance/continue-velero-alias-v10-remote.py:569`). Existing durable journals, provider/archive receipts, the exact 128-byte S3 probe, finite fixtures, and validation/cleanup remain part of the recovery lane (`:620`, `:649`, `:520`; `.local/ops-acceptance/continue-velero-alias-v10-root.py:278`, `:372`). Root requires independent CODE=APPROVE and ARCH=APPROVE/CLEAR bound to the current request and complete source map before its first connection (`:308`, `:330`, `:351`). Missing, inherited, incomplete, or WATCH/BLOCK reports were independently denied.

## Root Cause

The consumed v9 bulk CRD representation exceeded the unchanged output gate: 6,540,046 bytes versus 4,194,304 allowed. The inspected native typed projection produced 12,765 bytes while preserving the complete relevant inventory. The successor binds that exact observed comparison and unchanged budgets (`.local/ops-acceptance/continue-velero-alias-v10-contract.py:534`).

## Validation and Trade-offs

**Performed:** final hash readback of all 443 protected assets and five helpers; normalized AST comparison; 29 isolated local checks covering visibility, ownership, types, absence, coverage, approval gating, and consumed predecessor denial. All passed. No files, remote operations, generator phases, or evidence-writer phases were invoked.

**Inspected:** sealed postseal evidence reporting 373 executions, 320 unique regressions, 16 new v10 tests, 52 forgery cases, and zero failures/errors/skips; matching full-run receipt/log; actual Pyright 1.1.414 LSP receipt for all five exact sources with zero errors/warnings. This lane did not rerun the full suite or LSP.

| Option                                         | Benefit                                                                    | Trade-off                                                                                    |
| ---------------------------------------------- | -------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------- |
| Current typed projection plus full owned reads | Preserves ownership visibility and owned semantics within existing budgets | Depends on the reviewed native printer format; malformed or oversized output stops execution |
| Retain full bulk JSON                          | Retains foreign schemas in bulk output                                     | Reproduces the observed gate failure; increasing the limit changes the approved budget       |

Recommendation: retain the sealed v10 design and existing phase gates. No architectural code change is required. Runtime drift, partial effects, or further output-format changes must stop at the existing evidence/review boundary.

## Machine Readable

```json
{
  "lane": "ARCH",
  "decision": "APPROVE",
  "architecturalStatus": "CLEAR",
  "independent": true,
  "requestSHA256": "935242362fc459942bcef49e955d6216eee95438a25121cf1eda443e390fcace",
  "scriptHashes": {
    ".local/ops-acceptance/continue-velero-alias-v10-contract.py": "56b4932709872f6ef9f34cdf1a88a5c0b952ddd9d2c04c1e15c08adb4efa1a33",
    ".local/ops-acceptance/continue-velero-alias-v10-remote.py": "0d95ec93fa7c292cb0a653d6e35dadb35196c90ed23d117a8f85b77d32450c59",
    ".local/ops-acceptance/continue-velero-alias-v10-root.py": "8283a467f24ef6bf81f44374a8a33342ab6cafb1afe5d627602c6b8c1ca729a4",
    ".local/ops-acceptance/continue-velero-alias-v10-generate.py": "4c9728711768d0111043c63160820d0f1993ce8e92842bb97594db92f45a36fe",
    ".local/ops-acceptance/continue-velero-alias-v10-test.py": "dcb4041a11bcab07ff55800f65ed08325fc477db3a15184b320dee9438a5a869"
  },
  "parentTaskPacket": "docs/harness/tasks/2026-11-07-ops-planned-task-closeout.task.md",
  "tier": "L2",
  "findings": [],
  "reportWrittenByLane": false,
  "performedChecks": {
    "protectedAssets": 443,
    "helperSources": 5,
    "finalHashesStable": true,
    "normalizedRemoteAST": {
      "unchangedExistingFunctions": 25,
      "changedExistingFunction": "objects",
      "newFunction": "crd_inventory",
      "objectsChange": "Only bulk CRD inventory verification; full owned validation prefix and RBAC suffix preserved"
    },
    "normalizedRootAST": {
      "onlyChangedExistingFunction": "reviewed_inputs"
    },
    "isolatedLocalCases": 29,
    "passed": 29,
    "failed": 0,
    "approvalCallLine": 330,
    "firstConnectionLine": 351,
    "actualConsumedV9FailureAndInspectionDeniedAsSuccessPredecessors": true,
    "plainJSONFenceAccepted": true,
    "extraFenceLanguageRejected": true,
    "fileWrites": 0,
    "remoteExecuted": false,
    "evidenceWriterInvoked": false,
    "generatorPhaseInvoked": false,
    "fullSuiteRerun": false,
    "LSPRerun": false
  },
  "inspectedEvidence": {
    "handoffSHA256": "665937e17696b90b12241fbd1557ddb29378b45e12579e36b8b2d834b46189d9",
    "postsealReadbackSHA256": "b454048216f6f6fa5da4981eb96e5e73791611a8dd07f19da5b7dc4d8895837e",
    "postsealFullReceiptSHA256": "0f0efece0e2ef0046aa8030dd6a7e451f14ef737d4683a711b047ce0f25ab01d",
    "postsealFullLogSHA256": "f218f30a543034e4f1ff9decd16337d2fa5ca960b054021793130fb468417e69",
    "actualFiveSourceLSPReceiptSHA256": "3bd8fa3a00105727c6848a1ece1a33651d7a2f16120fc000cc64816aa46dd820",
    "projectionControlSHA256": "51c357a0024718f09e95411b4da2de9fde768ad145c053a68851894fbad10714",
    "projectionEnvelopeSHA256": "ef0ec9afc9f32b585c5714dc077b5d2feedb704c403ad61a31661b98ad600420",
    "testsRun": 373,
    "uniqueRegressions": 320,
    "newV10Tests": 16,
    "forgeryCases": 52,
    "failures": 0,
    "errors": 0,
    "skips": 0,
    "pyrightVersion": "1.1.414",
    "pyrightErrors": 0,
    "pyrightWarnings": 0,
    "remoteBaselineFiles": 43,
    "ownedObjects": 22,
    "observedProjectionRows": 53,
    "observedOwnedCRDs": 13,
    "observedFullCRDBytes": 6540046,
    "observedProjectionBytes": 12765
  },
  "approvalScope": "Exact sealed five-helper architecture in the existing alias recovery lane",
  "predecessorApprovalInherited": false,
  "CODELaneApprovalClaimed": false,
  "runtimeApproval": false,
  "P0Approved": false,
  "stopCondition": "Independent ARCH review complete; Root must combine fresh CODE evidence and preserve phased runtime gates"
}
```
