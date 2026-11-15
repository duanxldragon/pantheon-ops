`continue-velero-alias-v3-independent-arch-report.md` — inline report only; no file written.

## Summary

**REQUEST CHANGES.** The original v2 expiry defect is fixed, but v3 does not durably record probe ownership before storage effects. Process termination can therefore leave an object without its receipted content hash; the root review gate also fails to enforce ARCH `CLEAR`.

Scope: business-domain acceptance tooling under the supplied L2 parent; logical separation, no product implementation changes.

## Analysis

1. **P1 — Probe metadata is durable too late.** [remote:542](/D:/workspace/go/pantheon-platform/pantheon-ops/.local/ops-acceptance/continue-velero-alias-v3-remote.py:542) stores path, content hash and outcomes in memory. PUT follows at line 549; the repair receipt is journaled only at [remote:664](/D:/workspace/go/pantheon-platform/pantheon-ops/.local/ops-acceptance/continue-velero-alias-v3-remote.py:664). The inherited command has an actual [900-second TERM boundary](/D:/workspace/go/pantheon-platform/pantheon-ops/.local/ops-acceptance/resume-velero-s3-v8-root.py:154).

   An isolated interruption after synthetic PUT committed reproduced one remaining object, four alias journals, and no durable probe metadata or repair receipt. This simulated termination; no actual S3 operation or process signal ran.

   [root:390](/D:/workspace/go/pantheon-platform/pantheon-ops/.local/ops-acceptance/continue-velero-alias-v3-root.py:390) inspects the probe only when the terminal repair receipt exists. With that receipt absent, the independently exercised function returned `exitCode: 0` without probe inspection. Replay remains forbidden, but inspection cannot establish the probe’s content identity.

2. **P2 — ARCH status is not enforced.** [approved_reviews:262](/D:/workspace/go/pantheon-platform/pantheon-ops/.local/ops-acceptance/continue-velero-alias-v3-root.py:262) validates lane, approval, independence, request hash and script hashes, but never checks `architecturalStatus`. Actual extracted function bodies accepted matching ARCH approvals with missing, `WATCH`, and `BLOCK` statuses. The [positive regression fixture:848](/D:/workspace/go/pantheon-platform/pantheon-ops/.local/ops-acceptance/continue-velero-alias-v3-test.py:848) also omits the status.

**Original defect status:** fixed. Freshness now precedes [PUT:547](/D:/workspace/go/pantheon-platform/pantheon-ops/.local/ops-acceptance/continue-velero-alias-v3-remote.py:547) and [DELETE:555](/D:/workspace/go/pantheon-platform/pantheon-ops/.local/ops-acceptance/continue-velero-alias-v3-remote.py:555). Independent expiry scenarios confirmed no PUT after expiry and no compensating DELETE after successful PUT followed by expiry.

The [immutable predecessor checks](/D:/workspace/go/pantheon-platform/pantheon-ops/.local/ops-acceptance/continue-velero-alias-v3-contract.py:147), [original 22-object authority](/D:/workspace/go/pantheon-platform/pantheon-ops/.local/ops-acceptance/continue-velero-alias-v3-root.py:55), captured loader and [attempted-phase replay refusal](/D:/workspace/go/pantheon-platform/pantheon-ops/.local/ops-acceptance/continue-velero-alias-v3-root.py:275) remain preserved.

## Root Cause

Exception-time journaling cannot guarantee ownership evidence across process termination. Separately, source-hash binding validates review identity without validating the architectural decision’s required status.

## Recommendations

1. **Small–medium effort, blocking:** durably journal exact probe intent before PUT, including content hash and uncertain outcome; record successful PUT and DELETE intent/outcome separately. Keep freshness immediately before each storage effect. Inspect exact finite journals when the terminal receipt is absent; unresolved evidence must return incomplete inspection. Preserve no replay and no compensation.
2. **Small effort:** require ARCH `architecturalStatus == "CLEAR"` alongside `APPROVE`; test missing/WATCH/BLOCK rejection. Deliver any repair as a newly reviewed seal, preserving existing failures and assets.

## Architectural Status

`BLOCK`

## Trade-offs

| Option                      | Benefit                                   | Cost                                |
| --------------------------- | ----------------------------------------- | ----------------------------------- |
| Exception-only receipt      | Minimal implementation                    | Loses probe identity on termination |
| Durable per-effect journals | Preserves inspect-only ownership evidence | Adds finite sealed outputs          |

**Strongest counterargument:** caught failures already produce durable receipts, and one-shot dispatch prevents replay. That protects ordinary exceptions, but does not cover the existing TERM boundary; pre-effect journaling preserves both protections.

## Verification and Stop Condition

Independently checked request SHA256, five actual script hashes, five in-memory compilations, and all **237 protected assets** before/after; all matched. Ran **10 isolated scenarios** using actual extracted function bodies and synthetic I/O.

Verified the [sealed records](/D:/workspace/go/pantheon-platform/pantheon-ops/.harness/evidence/2026-11-07-ops-planned-task-closeout/verification/p0-integration-20261109/continue-velero-alias-v3-verification.json:30): original230, preserved45+new8, **238 unique**, zero skips; hash-bound native Pyright1.1.414 reports five files, zero errors/warnings. Full suites and LSP were **not independently rerun**.

Stop: repair both findings and obtain independent review. Actual tag concurrency, Ready/BSL, S3, BackupRestore, persistence and cleanup remain [pending](/D:/workspace/go/pantheon-platform/pantheon-ops/.harness/evidence/2026-11-07-ops-planned-task-closeout/verification/p0-integration-20261109/continue-velero-alias-v3-handoff.md:78). Public identity/HTTP, IAM and audit were outside this five-file review and remain unverified here. No runtime or P0 success is asserted.

```json
{
  "lane": "ARCH",
  "decision": "REQUEST CHANGES",
  "independent": true,
  "architecturalStatus": "BLOCK",
  "requestSHA256": "96fef023020ad1f8e880e01f04612acfa0480bd3009c009063fa8756703e3b07",
  "scriptHashes": {
    ".local/ops-acceptance/continue-velero-alias-v3-contract.py": "bfac98f9c72ae3cbe3ff4bc3a9d97fe3d3c377f471e1ab851586c945772a4ca6",
    ".local/ops-acceptance/continue-velero-alias-v3-remote.py": "5e443950e04855771f6a36ad67041439a34902be9eb66dc0a960891e4dcb951b",
    ".local/ops-acceptance/continue-velero-alias-v3-root.py": "d3604678f345f258157a572add4fa8d8d367f30f27d66082e0e06589a1285897",
    ".local/ops-acceptance/continue-velero-alias-v3-generate.py": "a1c3738d2ebff357e18f20866f5ca9d84131d1f178332e321ef81ac2588ad88f",
    ".local/ops-acceptance/continue-velero-alias-v3-test.py": "da68651673b5a7281a152ecd0dce5d6138f1ccd6a9cd5efa70eb513585a8f176"
  }
}
```
