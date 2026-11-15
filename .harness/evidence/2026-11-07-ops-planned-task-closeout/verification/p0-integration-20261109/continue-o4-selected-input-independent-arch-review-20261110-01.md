# Independent ARCH review — REQUEST CHANGES / BLOCK

L2 parent: docs/harness/tasks/2026-11-07-ops-planned-task-closeout.task.md. Subject/request SHA256: 8a866ac9c1d608558ee3d1e04805cd7f8a5e0a444b8fab2fc1dfe8ac7bd84a82. Review covers the selected-input consumer tool delta only. Historical CODE/ARCH conclusions do not approve this delta.

## Summary

Current-source adoption is blocked, and a separate deterministic locale replay defect is reproduced. Request changes; preserve retained assets and existing path/link/size/count gates.

## Findings

1. **P1 — Current-source adoption remains blocked.** Calling exported selectOverlayInputs on the actual Ops root throws Unsafe declared Ops input path, without running assembly. The exact two diagnosed ignore roots contribute zero Git records, but 175 remaining declaration-covered records fail validation: 162 trailing-slash directory records (first is an existing nested Git repository) and 13 paths longer than 240 characters. The NUL parser passes every Git record to assertRelativeInput before selection filtering (scripts/business-overlay/rebuild-from-base.mjs:22, :27, :44); that validator rejects empty path components and paths over 240 characters (scripts/business-overlay/foundation-candidate.mjs:675, :683). Candidate entry invokes selection before stage resolution (scripts/business-overlay/rebuild-from-candidate.mjs:67). This is an actual adoption blocker, not evidence of a safety bypass. Some retained over-limit inputs already violate the earlier gate; their presence is not attributed to a newly introduced length limit. Evidence: [actual selector summary](../../../../../.local/ops-acceptance/continue-o4-selected-input-independent-arch-20261110-01/actual-selector-summary.json) and full invalid-path inventory beside it.

2. **P2 — Canonical locale replay disagrees with assembly.** Assembly calls recursive generateBusinessFallbacks on the copied tree (scripts/business-overlay/rebuild-from-base.mjs:651), which merges Object.assign in native directory order (:358–361). Replay calls the selected-files branch (:724), taking sorted selection.files (:60) and merging in that order (:354–355). On Windows, legal cmdb/a/en-US.json and cmdb/Z/en-US.json with the same key produce assembly winner Z and replay winner a. The fixture fails canonical verification (:735–737) while the selected file hashes, stamps, ancestor identities and Git selection remain unchanged. No consumer receipt exists. No duplicate-key rejection currently makes this input invalid. Evidence: [independent adversarial results](../../../../../.local/ops-acceptance/continue-o4-selected-input-independent-arch-20261110-01/adversarial.json).

## Root cause and recommendations

Git enumeration is assumed to supply only regular leaf paths, while actual retained inputs include opaque nested repository records. Independently, locale derivation uses two ordering contracts.

1. Resolve the named remaining input-shape blockers before another real consumer attempt. Make non-leaf Git record handling explicit and diagnostic; any deterministic expansion must preserve selected/copy/hash equality and exclude Git internals. Do not silently discard records, add blanket evidence ignores, relax limits or clean foreign assets. Retained over-limit paths require an explicit source-input decision. Effort: medium; impact: restores reviewable current-source adoption.
2. Make assembly and replay consume the same deterministic effective locale leaf list. Add the a/Z collision regression and verify unselected locales cannot enter derivation. Effort: small; impact: removes false canonical drift failures.

## Trade-offs

| Option                                                        | Benefit                                       | Cost                                                                                      |
| ------------------------------------------------------------- | --------------------------------------------- | ----------------------------------------------------------------------------------------- |
| Explicitly reject non-leaf Git entries with named blockers    | Retains current safety boundary               | Actual input adoption stays blocked until ownership/input policy is resolved              |
| Expand permitted declared repository inputs deterministically | Can preserve legitimate evidence              | Requires independent review of enumeration, ignored files, links and frozen leaf equality |
| Deterministic locale merge order                              | Small fix preserving existing merge semantics | May change precedence where duplicate keys exist                                          |
| Reject duplicate locale keys                                  | Avoids ambiguous precedence                   | Changes the accepted input contract and needs explicit adoption                           |

## Validation and stop condition

Fresh focused selected-input/canonical suite: 29/29 pass, zero skips. Fresh drift suite: 27/27 pass, zero skips. Independent checks reproduce the locale defect and reject package raw-read/revert, selected-copy read/revert, go.sum changes and 13 invalid Git path forms; Unicode staged metadata is accepted. Fixtures/caches are under this review's new .local root; existing reports and source files were not edited.

Eight actual source hashes matched the request before and after review. Original descriptor pin 2f105d76e38c82e8c301d816269ea26669a7a0291a5860be74d6ca71281a9ce6 and retained archive digest match; descriptor still lists 1141 files. The original hardlink pair retains inode 281474979169911, nlink 2 and 57 bytes; the two original caches retain their inodes and 62530476/55594218 byte sizes. Large cache contents were not read. Historical 4634-file assembly was not rerun or cleaned. Author's 124/124 and root 63/63 receipts were inspected, not independently repeated in full; native TS 6.0.3 diagnostics have checkJs false and do not establish full type safety.

Stop: findings have reproducible evidence. Re-review after input-policy correction and deterministic locale alignment. No successful real consumer receipt, C-F implementation, runtime/P0 approval, seal, release, lock change or remote operation is asserted.
