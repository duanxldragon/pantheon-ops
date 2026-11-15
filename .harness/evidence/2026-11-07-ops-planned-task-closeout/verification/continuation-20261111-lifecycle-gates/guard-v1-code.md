```json
{
  "subject": "Base-owner-lifecycle-candidate-guard-v1",
  "lane": "CODE",
  "independent": true,
  "decision": "REQUEST CHANGES",
  "decisionScope": "Three frozen candidate-preparation helpers only; output confinement defect blocks approval.",
  "sourceStable": true,
  "sourceHashes": {
    ".local/o4-owner-lifecycle-candidate-20261111/guard.mjs": "d30ae06ffcce1b2dfa653cc642339ee96ac178b601ba1e5c0bb4dcb0068405c0",
    ".local/o4-owner-lifecycle-candidate-20261111/prepare.mjs": "f32c34e31fee115dba11973e7b72f9a3879460fba1208f7be75b749346664b89",
    ".local/o4-owner-lifecycle-candidate-20261111/guard.test.mjs": "ccad580d8970d265c626477e536b48f070846f3d5019e585e4af46ce44d29c54"
  },
  "requestSHA256": "77c437e8c230a49e4bbe6e67f4cba196827134e5a50b3937332531ca4936aeaf",
  "requestBytesSHA256": "53b8470d2ead6b8c4da2dedeb438f9a2fc784223148d5222d5ef8c7d014e73bd",
  "findings": [
    {
      "id": "CODE-GUARD-001",
      "severity": "MEDIUM",
      "ref": ".local/o4-owner-lifecycle-candidate-20261111/prepare.mjs:126",
      "relatedRefs": [
        ".local/o4-owner-lifecycle-candidate-20261111/prepare.mjs:117",
        ".local/o4-owner-lifecycle-candidate-20261111/prepare.mjs:119",
        ".local/o4-owner-lifecycle-candidate-20261111/prepare.mjs:123"
      ],
      "issue": "Output-parent validation occurs before preparation, but is not repeated before the exclusive write. Prefix validation permits nested output directories. If an initially real nested parent is replaced by a junction/symlink during preparation, writeFileSync follows that parent and creates a new input outside the owned lane. The wx flag prevents leaf overwrite but does not enforce ancestor confinement.",
      "evidence": "Line119 checks the parent; line123 performs archive/proof/source preparation; line126 writes without another parent check. The nested output directory is not covered by selected source or helper pin checks.",
      "fix": "Restrict output to a direct regular filename under the fixed helper lane, reject nested output paths, and revalidate the fixed parent immediately before the wx write. Add regression coverage for parent substitution between initial validation and final creation.",
      "verificationStatus": "Grounded in complete control-flow reading; no filesystem mutation or escape reproduction performed."
    }
  ],
  "specCompliance": {
    "status": "FAIL_OUTPUT_CONFINEMENT",
    "otherwiseVerified": [
      "Pinned predecessor1141 plus four updates and seven additions produces inventory1148 without changing predecessor policy.",
      "Source11 and helper3 require distinct independent CODE/ARCH approvals; guard approvals bind the immutable request core and helper hashes.",
      "Completed proof references are excluded from the core hash to avoid circular review references.",
      "All six gates require successful receipts, before/after byte hashes, equal inventories and current full-source agreement.",
      "Full race commands reject focused, filtered, compile-only and list substitutes; require readonly modules and CGO1.",
      "Adapter/dependency pins are checked when loaded and again near preparation completion; disagreeing gate pins fail.",
      "No producer execution mode exists; optional output uses exclusive creation."
    ]
  },
  "validation": {
    "helperHashMatchesBefore": 3,
    "helperHashMatchesAfter": 3,
    "requestBytesAndCoreHashRecomputed": true,
    "predecessorDescriptorHashMatched": true,
    "independentSyntaxChecks": "3/3 PASS",
    "independentNodeTests": {
      "tests": 42,
      "pass": 42,
      "fail": 0,
      "skipped": 0
    },
    "testsLimitation": "Synthetic positives verify guard logic, not actual readiness; existing tests do not cover output-parent substitution.",
    "scopedDiffCheck": "PASS; helper files are untracked, so syntax/body/hash checks provide the substantive coverage.",
    "diagnosticGap": "No applicable JavaScript LSP diagnostics tool available; node syntax checks and full body review completed."
  },
  "authorEvidenceVerified": {
    "receiptSHA256": "96359ad562fc71cfa1daf3a797c6bd1e547220068ca607cf84476f9d5c478a97",
    "handoffSHA256": "0bfa3aebf56ad379faee70f26f64711495fcafdfe6cc7d81fae32d6c6cc9e979",
    "testsLogSHA256": "18ab2ebdcf1f7bd75ac252d07be0f01e3f6dabb0dc82da0e07f17ed18d50bca4",
    "reportedResult": "Syntax3/3 and tests42/42 PASS, zero skips."
  },
  "sourceScope": "Only guard.mjs, prepare.mjs and guard.test.mjs reviewed; previous exact11 product-source review not repeated.",
  "actualReadiness": "WAITING: request contains five gate references, no full-race reference and no guard-review references.",
  "fullGatesApproved": false,
  "newCandidateApproved": false,
  "OpsRuntimeApproved": false,
  "P0": false,
  "sourceFilesEdited": false,
  "successfulPreparationExecuted": false,
  "preparationOutputCreated": false,
  "producerExecuted": false,
  "remoteOperations": false,
  "stopCondition": "Documented output boundary defect; revised helpers require a new request core and fresh independent review."
}
```
