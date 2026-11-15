## Findings

No blocking implementation contradiction found in the pinned six-file repair.

1. **Replay authenticates fixed derived bytes independently of report claims.** It validates Base commit, manifest/report hashes and declared business import ownership, then reruns existing registry, hook, locale, cleanup and smoke transformations in memory. Outputs must match canonical bytes and report hashes. Arbitrary `generatedFiles` entries receive ordinary shared comparison ([replay:668](D:/workspace/go/pantheon-platform/pantheon-ops/scripts/business-overlay/rebuild-from-base.mjs:668), [comparison:110](D:/workspace/go/pantheon-platform/pantheon-ops/scripts/business-overlay/check-shared-drift.mjs:110)). Forged output/report and arbitrary shared-file cases have focused regressions ([tests:484](D:/workspace/go/pantheon-platform/pantheon-ops/scripts/business-overlay/foundation-candidate.test.mjs:484)).

2. **The shared i18n seam remains exact and bounded.** Five business fallback resources are generated; only `zh-CN` and `en-US` shared resources receive the existing import/spread transformation. Registry wiring requires exactly one Base anchor occurrence ([fallbacks:359](D:/workspace/go/pantheon-platform/pantheon-ops/scripts/business-overlay/rebuild-from-base.mjs:359), [anchors:348](D:/workspace/go/pantheon-platform/pantheon-ops/scripts/business-overlay/rebuild-from-base.mjs:348)). Candidate package replay uses current, stability-checked original package inputs; the standalone official checker retains its legacy package path ([packages:724](D:/workspace/go/pantheon-platform/pantheon-ops/scripts/business-overlay/rebuild-from-base.mjs:724), [source checks:152](D:/workspace/go/pantheon-platform/pantheon-ops/scripts/business-overlay/rebuild-from-candidate.mjs:152)).

3. **Coverage claims are appropriately limited.** Identical in-coverage allowances still fail; canonical changed outputs supersede allowances without granting exemptions. Out-of-coverage entries emit `ALLOWLIST_NOT_COMPARED` ([coverage:142](D:/workspace/go/pantheon-platform/pantheon-ops/scripts/business-overlay/check-shared-drift.mjs:142)). Package, cleanup and smoke paths remain outside this candidate’s frontend shared coverage, although candidate replay separately checks their derived bytes. The stale-entry regression exercises the failure ([test:474](D:/workspace/go/pantheon-platform/pantheon-ops/scripts/business-overlay/foundation-candidate.test.mjs:474)).

4. **Directory ownership fixes the cleanup seam, with a syscall limitation.** Exclusive creation immediately captures BigInt device/inode identity. Cleanup preflights the subtree and rechecks ancestors before unlink/rmdir; unknown/replaced directories refuse cleanup and preserve the primary error through `cause` ([creation:553](D:/workspace/go/pantheon-platform/pantheon-ops/scripts/business-overlay/foundation-candidate.mjs:553), [cleanup:876](D:/workspace/go/pantheon-platform/pantheon-ops/scripts/business-overlay/foundation-candidate.mjs:876), [cause:505](D:/workspace/go/pantheon-platform/pantheon-ops/scripts/business-overlay/foundation-candidate.mjs:505)). Checks and mutations remain separate syscalls; atomic protection is not established.

5. **`--attempt` preserves the protected candidate identity.** It selects a fresh sibling attempt stage, revalidates descriptor/archive and retained snapshot inventory, and separates snapshot read identities from deletion ownership. Retained mode skips extraction; receipts are written only into the new assembled stage after verification ([routing:125](D:/workspace/go/pantheon-platform/pantheon-ops/scripts/business-overlay/rebuild-from-candidate.mjs:125), [retained handling:757](D:/workspace/go/pantheon-platform/pantheon-ops/scripts/business-overlay/foundation-candidate.mjs:757), [receipt:257](D:/workspace/go/pantheon-platform/pantheon-ops/scripts/business-overlay/rebuild-from-candidate.mjs:257)). It cannot certify the old failed consumer through this path.

## Architectural Status

**`CLEAR` — limited to this implementation and the next source-quiescent `--attempt` integration.** Independent CODE review remains separate; this grants no P0 closure, release approval or waiver.

The added call paths reuse existing transformations and standard filesystem primitives: checker → replay → existing helpers; assembler → directory adapter → identity checks. No new orchestration layer is necessary.

## Root Cause

Consumer verification treated legitimate deterministic business derivatives as shared drift, applied stale-allowlist reasoning beyond selected coverage, and lacked creation-time ownership receipts for assembler directories ([repair summary:5](D:/workspace/go/pantheon-platform/pantheon-ops/.harness/evidence/2026-11-07-ops-planned-task-closeout/o4-generator-integration-repair/summary.md:5)).

## Recommendations and Trade-offs

1. **Required next step — low effort:** after independent CODE review and source quiescence, run the prepared fresh attempt. Require its own current-source receipt; preserve the original failed stage.
2. **No mandatory source repair identified.** Keep build/runtime acceptance separate from fixture and retained-tree evidence.

| Choice                          | Benefit                                       | Limit                                          |
| ------------------------------- | --------------------------------------------- | ---------------------------------------------- |
| Reuse canonical helpers         | Small change; exact assembler equivalence     | Does not independently prove product semantics |
| Identity preflight and rechecks | Refuses observed foreign/replaced directories | Separate syscall race windows remain           |

## Validation and Stop Condition

Fresh read-only checks confirmed all six current/baseline pins and exact in-memory patch reconstruction against [source-hashes.json](D:/workspace/go/pantheon-platform/pantheon-ops/.harness/evidence/2026-11-07-ops-planned-task-closeout/o4-generator-integration-repair/source-hashes.json:1). Descriptor pin `2f105d76…9ce6`, Base commit `135730fd6f63ad73cf40f6e4b321fdd9f69ea932`, archive and 1,141 snapshot files matched; all 4,634 report-owned hashes matched. Fresh retained checking verified 801 shared files, 17 canonical outputs and 10 migrations. Both consumer receipts remain absent.

Six syntax checks passed. The recorded 155/155 suite was inspected, not rerun. Full assembly, semantic build, race, runtime and rendered UI evidence remain open ([evidence:63](D:/workspace/go/pantheon-platform/pantheon-ops/.harness/evidence/2026-11-07-ops-planned-task-closeout/o4-generator-integration-repair/commands.json:63)). No files were changed; O2 remains outside this review.
