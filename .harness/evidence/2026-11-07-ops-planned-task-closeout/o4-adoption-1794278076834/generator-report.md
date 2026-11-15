# O4 backend Generator handoff — blocked at real candidate pairing

Date: 2026-11-10. Parent: `docs/harness/tasks/2026-11-07-ops-planned-task-closeout.task.md`. Layer: L2 business/observability adoption of Base platform/system ports. Implementation plan revision4 SHA256: `9aaa0e53b43c040a4d04949693d3ef6ab4b086336fe953c3e04105400e4caaf7`.

Disposition: **prerequisite product compatibility blocker; backend C–F not advanced**. This Generator does not approve source, runtime, P0 closure, official consumption or release. Maintainer business-tenancy approval is acknowledged; no repeated scope approval is requested.

## Findings and exact stop cause

1. **Actual Ops assembly failed before snapshot/assembly/drift execution.** `rebuildFromCandidate` freezes all `repositoryOverlayPaths` via `captureInputs`, including the entire declared `.foundation` directory. `readStableFile` refuses files over 33,554,432 bytes. Nine retained official release files exceed this bound; the first sorted failure is `.foundation/releases/pantheon-base-v0.17.6/repo.tar` (45,168,640 bytes, one link). Current official v0.18.6 `repo.tar` is 59,422,720 bytes, and its release TGZ is 36,214,646 bytes. Directly invoking the exact reviewed `readStableFile` on the current official `repo.tar` reproduces `Candidate input must be a bounded regular file without hard links`. See `failure-receipt.json` and `input-shape-diagnostics.json`. No candidate Ops stage or pairing receipt exists; this failure does not establish shared drift success. The official assets are preserved, not deleted or relabeled.

2. **A second real packaging incompatibility would remain after fixing the input bound.** The sealed archive contains the actual unchanged `config/foundation-consumer-artifacts.json` and `config/review-loop-legacy-artifacts.json`. The latter has eight `.harness/archive/.../review.md` references. All eight exist in actual Base, none can be present in the candidate archive because BOTH reviewed candidate path validators reject `.harness`. The existing exported `projectConsumerReviewRegistry` requires source existence before projecting excluded artifacts, and would report `Stale Base legacy entry: <reference>` in this snapshot. This later assembler error was not reached in the failed product run; the missing-file/path-policy contradiction itself was verified against the actual sealed archive in memory. No configuration was omitted or rewritten to hide it, and no shared sources/history were hand-copied.

These fixes require the prerequisite owning lane to change and independently review shared candidate tooling/packaging. This lane may change only necessary business assembly entries in the manifest; altering `.foundation` ownership, limits or archive validation would exceed that authority. The requested explicit stop on an actual prerequisite mismatch therefore applies. No repair of Base/shared Ops source, release lock, credentials, ledgers, existing failures or other product edits was attempted.

## Concrete work and candidate receipts

- Added `.local/ops-acceptance/o4-adoption-candidate.mjs`: thin stage entry importing the **actual** Base `buildCandidateArtifact` with explicit Base root and Ops-owned inventory/output, then the **actual** Ops `rebuildFromCandidate`, which owns assembler/drift invocation. No recursive CLI or children. The imported-producer alternative avoids writing Base or requesting sandbox escalation.
- Added `.local/ops-acceptance/o4-adoption-prerequisite.test.mjs`: meaningful replay of sealed byte integrity, real retained-release input rejection, real registry/path-policy mismatch, approved source/lock preservation and absence of a failed product stage.
- Wrote only new evidence in this task subdirectory. No backend, frontend or `business-overlay.json` changes by this Generator.
- Source inventory: 1,133 explicit product files selected from actual current Base backend/frontend/scripts/tests assets and explicit public package/config files. It includes all four capability families and required API exports. Private configuration, credentials, `.harness`, caches, dependencies and uploads were excluded by the reviewed producer policy. `source-inventory.json` records the complete selection and provenance commit. This is an execution inventory of the approved prerequisite source; no new independent product-inventory approval is asserted.
- Plan and all 69 receipt paths match review3 plus the two exact review4 changed hashes, before production and again after verification. The diagnostic CODE supplement APPROVE and ARCH CLEAR remain upstream dispositions; this report grants none.

| Receipt                               | Value                                                                        |
| ------------------------------------- | ---------------------------------------------------------------------------- |
| Candidate identity / source aggregate | `candidate-64cbd8e211df852a17d9bc7a93ac09af300eb40fde4b08db5133a95c436a38de` |
| Descriptor SHA256                     | `77faf23ed88c721bfd6e14b2888194a92769de9c3402bce5ebe622093a283f51`           |
| Archive SHA256                        | `2d679d5f06d244e91e4c7c14b20118a30d9125f5b1bc01089d7a6eca3c49cb89`           |
| Official Ops lock before/after/final  | `95de1a87270e15c0d4c672418a5df906818a88652f8a3e6c480f1484ee9fa70c`           |
| Producer approved SHA256              | `646e622f2eef3ae30d29aed50272d3f4665dd54d04c2fcb1ffb41777e554d6e3`           |
| Producer test approved SHA256         | `17301133892fe649d7fdd2ec25ed7e0a570c45b47f7a0842ed896b0723dd9b11`           |
| Sealed output                         | `.local/ops-acceptance/o4-adoption-1794278076834-sealed/`                    |
| Product Ops stage                     | **None: input freeze failed before creation**                                |

The producer was really executed and sealed descriptor/archive bytes successfully, with `sourceStable=true`. It is a candidate, not an official Base release. The expected descriptor pin passed into consumer assembly came from the producer's original in-memory receipt; it was not recalculated from an altered file. Both approved validators subsequently verified the archive and descriptor again. Full descriptor and file hashes remain in the sealed output; `producer-receipt.json`, `prerequisite-receipt.json`, and `source-inventory.json` bind this run. `files-and-commands.json` binds the new helper/evidence files and exact commands.

## Fresh verification

All commands below ran from Ops root. No test suite skips, cancellations or TODOs occurred.

| Command                                                                                                             | Result / evidence                                                                       |
| ------------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------- |
| `node --check .local/ops-acceptance/o4-adoption-candidate.mjs`                                                      | exit0                                                                                   |
| `node .local/ops-acceptance/o4-adoption-candidate.mjs`                                                              | exit1: actual Ops input freeze failure; Base producer succeeded; `failure-receipt.json` |
| `node --check .local/ops-acceptance/o4-adoption-prerequisite.test.mjs`                                              | exit0                                                                                   |
| Set `O4_ADOPTION_EVIDENCE` to this directory; `node --test .local/ops-acceptance/o4-adoption-prerequisite.test.mjs` | exit0, 4/4; `real-input-regressions.log`                                                |
| `node --test ../pantheon-base/scripts/foundation-release/candidate-artifact.test.mjs`                               | exit0, 37/37; `base-candidate-tests.log`                                                |
| `node --test scripts/business-overlay/foundation-candidate.test.mjs`                                                | exit0, 64/64; `ops-candidate-tests.log`                                                 |
| `npm run check:business-overlay`                                                                                    | exit0; 56 declared paths, existing rebuilt tree; `overlay-check.log`                    |
| `npm run test:business-overlay`                                                                                     | exit0, 62/62; `overlay-tests.log`                                                       |
| `git diff --check -- backend/modules/business/observability backend/modules/business/opsinit business-overlay.json` | exit0; Git CRLF notices only; these preexisting diffs were not altered                  |

Syntax diagnostics cover the two new JS helpers; named LSP diagnostics are unavailable, and no static type-safety claim is made. Existing candidate/overlay suites use synthetic fixtures, including intentionally denied drift cases; their GREEN results do not establish real product assembly. No official `rebuild:from-base` or replacement stage was used to evade candidate pairing.

## Remaining backend and frontend handoff gaps

No new usable API/DTO/catalog contract is delivered to frontend. Revision4 D remains the approved implementation specification, not an implemented API. Planned canonical route families are `/api/v1/business/observability/query-history`, `/saved-views`, and `/datasources/:id/query-fields`; intended permission families use legal `business:observability:query-history:{view,delete,restore}` and `business:observability:saved-view:{view,create,update,delete,restore}`. Source-kind read permissions apply separately; restore never executes. These routes/permissions/catalogs must not be advertised as available from this candidate attempt.

All C–F product implementation remains: tenant datasource ownership/default/alias/legacy0 alert fences and late-result checks; actor-owned history/views, guard transactions, quota/retention and Redis dual-scope permits; enum/catalog recipes/compiler, transient POST query and metadata-only audit; migration37, strict readiness/dirty recovery and supported artifact startup gate. Targeted Go tests, build, vet and real MySQL/Redis acceptance **were not run against a stage**, because no stage exists and inherited official v0.18.6 does not contain the authoritative APIs required by this task. This is an explicit prerequisite/runtime gap, not a silent skipped test. Root final stable full-tree CGO race remains deferred as requested.

Next action belongs to the prerequisite owning lane: repair real input selection/bounds and consumer archive-registry packaging, obtain exact independent CODE/ARCH disposition for changes, then rerun the same product producer/assembler before adopting C–F. This Generator neither self-approves that repair nor closes O4/P0.

Gate Outcomes: exact source/plan/lock caught none | actual candidate assembly caught retained official asset size incompatibility | sealed real-input regression caught archive-registry packaging incompatibility.
