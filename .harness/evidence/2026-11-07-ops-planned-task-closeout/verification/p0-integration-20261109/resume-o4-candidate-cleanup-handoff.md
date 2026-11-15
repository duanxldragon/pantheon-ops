# Resume O4 candidate cleanup handoff

Status: scoped Generator correction locally verified; independent CODE/ARCH approval pending. No product sealing or final P0 approval.

## Linkage and scope

- Parent task: `docs/harness/tasks/2026-11-07-ops-planned-task-closeout.task.md`.
- Plan: `resume-o4-candidate-cleanup-plan.md` in this directory.
- Evidence prefix: `resume-o4-candidate-cleanup-*` under `.harness/evidence/2026-11-07-ops-planned-task-closeout/verification/p0-integration-20261109/`.
- Authoritative detailed handoff: this file, `resume-o4-candidate-cleanup-handoff.md`. It is separate from adapter output and previous generator reports.
- Source scope: only `scripts/business-overlay/foundation-candidate.mjs` and `scripts/business-overlay/foundation-candidate.test.mjs`.
- L2 because this is a safety-sensitive candidate/consumer inheritance boundary. Landing side is Ops candidate tooling. No platform layer, system domain, business runtime or physical module layout change.
- No Base, formal CLI, candidate CLI, assembler, drift checker, lock, manifest, backend, frontend, SQL, publication, delegation or model changes.

## Findings to correction mapping

| Prerequisite finding                                                                                                                                      | Correction                                                                                                                                                                                                                                                                                                                    | Evidence                                                                                                                                                               | Disposition                                                      |
| --------------------------------------------------------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------- |
| CODE MEDIUM, former `foundation-candidate.mjs:682`: an owned path is not an owned directory instance; rename/recreate can delete a foreign real directory | Replace the path Set with a per-stage identity Map. Immediately after each exclusive source mkdir, capture real-directory bigint lstat dev/ino. Compare both before deletion; reject missing, unsupported, unknown or replaced identity.                                                                                      | Source diff; extraction and assembly failure hook tests for stage/snapshot/nested/ancestor replacement; retained-original and replacement-content assertions           | Locally verified; independent CODE rerun required                |
| ARCH BLOCK: symlink refusal does not guard real-directory replacement, including descendants under a recursively removed ancestor                         | Capture all directly created source directory identities and existing ancestor identities; preflight the complete subtree before any removal. Recheck the directory chain before each unlink/rmdir. Remove recursive rm entirely from candidate cleanup.                                                                      | Source diff; nested/ancestor and unknown-descendant tests; deletion-boundary test                                                                                      | Locally verified conservatively; independent ARCH rerun required |
| Assembled target has no trustworthy creation-identity seam in this lane                                                                                   | Do not enroll directories just because they are found under the owned stage. The unchanged assembler creates assembled directories outside this consumer's direct mkdir calls, so their identity ownership is unknown and failure cleanup retains the stage with `CandidateCleanupError` / `ERR_CANDIDATE_CLEANUP_OWNERSHIP`. | Assembly/assembled rename-recreate test; eight source mutation cases and drift/migration failure cases assert refusal, retained stage and absence of candidate receipt | Deliberate fail-closed behavior within two-file scope            |
| Windows 64-bit directory IDs cannot safely be compared as JS Numbers                                                                                      | Use bigint lstat values; refuse unavailable or nonpositive ino and invalid dev. No timestamp/marker fallback.                                                                                                                                                                                                                 | `resume-o4-candidate-cleanup-windows-identity.json`; missing/zero-identity tests                                                                                       | Local Windows semantics verified                                 |
| Parent replacement must not be hidden by moving the same snapshot into the new parent                                                                     | Source verification now checks recorded directory identities before capturing/checking file stamps.                                                                                                                                                                                                                           | Unchanged-snapshot-moved-into-replacement-stage regression                                                                                                             | Locally verified                                                 |

## Reproduction and verification

All runs used the current Windows host with Node v24.14.0. Fixture construction alone imports the existing Base producer; no Base source was changed.

- Unchanged initial candidate suite: `node --test scripts/business-overlay/foundation-candidate.test.mjs` -> 48/48 passed, 0 failed/skipped. Log: `resume-o4-candidate-cleanup-baseline.txt`.
- Red before implementation: the same command with the first 11 added tests -> 59 total, 49 passed, 10 failed, 0 skipped. Nine actual extraction-write/assembly-copy rename/recreate failure scenarios and unknown-descendant refusal failed against the path-only implementation; unchanged cleanup passed. Log: `resume-o4-candidate-cleanup-red.txt`.
- First green: 59/59 passed, 0 skipped. Log retained at `resume-o4-candidate-cleanup-green.txt`.
- Final candidate suite: `node --test scripts/business-overlay/foundation-candidate.test.mjs` -> 63/63 passed, 0 failed/cancelled/skipped/todo. Log: `resume-o4-candidate-cleanup-tests-final.txt`.
- Final existing overlay suite: `npm run test:business-overlay` -> 62/62 passed, 0 failed/cancelled/skipped/todo. Log: `resume-o4-candidate-cleanup-overlay-tests-final.txt`.
- `node --check scripts/business-overlay/foundation-candidate.mjs` -> exit 0.
- `node --check scripts/business-overlay/foundation-candidate.test.mjs` -> exit 0.
- `node node_modules/prettier/bin/prettier.cjs --check scripts/business-overlay/foundation-candidate.mjs scripts/business-overlay/foundation-candidate.test.mjs` -> exit 0.
- `npm run check:business-overlay` -> exit 0.
- Scoped `git diff --no-index --check` for both before/after source files emitted no whitespace diagnostics; exit 1 is the no-index content-difference status. Corresponding `*-whitespace.txt` logs are empty.
- Full workspace `git diff --check` -> exit 2 with EOF blank-line findings in unrelated existing task packet, business error appendix and k8s frontend edits. Those files are outside this lane and were not repaired. Exact diagnostics: `resume-o4-candidate-cleanup-diff-check.txt`.
- No backend/UI/manifest changes or commit/PR operation: Go race, frontend build and a formal release rebuild were not applicable and were not run. The overlay test suite exercised existing assembly contracts in isolated fixtures.

The original 48 baseline test cases are retained alongside 15 new cases, with no skip or count masking. The existing eight assembly mutation cases and two drift/migration failure cases now explicitly expect typed cleanup refusal and retained unknown assembled directories instead of successful recursive cleanup. Their failure/no-candidate-receipt requirement remains enforced. One enclosing test description was updated to describe retention; test cases were not removed.

## Windows directory identity semantics

Node/libuv Windows stat exposes a volume/device identity in dev and the filesystem file ID in ino. These are filesystem instance identifiers, not path names, markers or content hashes. Bigint avoids rounding 64-bit IDs (the observed IDs exceed Number.MAX_SAFE_INTEGER). On this local filesystem the experiment observed:

- Created directory: dev `59148087`, ino `75716768735343292`.
- Renamed retained original: identical dev/ino.
- Real replacement at the former path: dev `59148087`, ino `22236523160417184`.

Unavailable identity fails closed. These observations establish the tested local Windows behavior, not universal behavior for every remote filesystem/reparse implementation. No directory file-descriptor/handle locking guarantee is claimed.

## Honest race and error boundaries

- mkdir and identity lstat are separate pathname syscalls. Enrollment captures identity immediately after a successful exclusive mkdir; there is no atomic created-handle receipt in this Node interface. A hostile namespace replacement inside that syscall interval is outside the proved guarantee.
- Identity checks and unlink/rmdir are also separate syscalls. Rechecks narrow the boundaries; they do not create an atomic namespace lock. A race after the final check is outside the proved guarantee. Recursive removal is never delegated to rm after a check.
- A replacement observed before cleanup starts preserves the stage intact. A replacement observed after one owned unlink may leave the original stage partially cleaned; the deletion-boundary test explicitly proves preservation of the replacement and remaining original content without claiming rollback or atomicity.
- An unknown assembled directory always retains the stage on failure, including otherwise legitimate assembly failures. This trades automatic cleanup for a safe refusal until the coordinating root has a reviewed creation seam.
- Extraction cleanup receives the original failure as Error.cause. The unchanged `rebuild-from-candidate.mjs` catch passes only the root to cleanup, so an assembly cleanup refusal can mask the triggering assembly error; its cause records the ownership failure. Preserving the original assembly cause requires a root-owned call-site repair outside this lane. No claim that every original assembly message is preserved is made.
- No independent reviewer was spawned or impersonated. The user explicitly made this a no-delegation Generator leaf.

## Preservation and exact final source

`resume-o4-candidate-cleanup-source-receipt.json` compares 715 initial files: the two owned sources changed as expected; the other 713 protected files are byte-identical. Protected files include current official lock, formal CLIs, candidate CLI, assembler, manifests, drift checker/reports, Base producer/tooling and all prior files in this parent evidence directory. No previous evidence was overwritten. Before-source snapshots and scoped diffs are retained as new evidence.

Final SHA256:

- `scripts/business-overlay/foundation-candidate.mjs`: `da8a8eeeb0a9f52514b157353475a75d43675711f264731cd8564398d7610161`
- `scripts/business-overlay/foundation-candidate.test.mjs`: `aeeabfebbdd19bb9c09d18552601beb6ace20697e4ae268256c56a45912df151`

## Review and next owner

Findings-first local self-review: no remaining observed real-directory replacement deletion in the tested paths; unknown directory ownership is retained; the ancestor checks and the removal of recursive rm constrain the deletion boundary. No general lock/marker framework or inherited behavior override was introduced.

Coordinating root must rerun both independent CODE and ARCH reviewers after its scoped repairs, against these hashes or freshly recorded hashes if later changed. Review must explicitly evaluate the fail-closed assembled-directory policy, error-cause call-site gap and non-atomic pathname race boundaries. Actual product candidate sealing remains prohibited until those reviewers pass. Final P0 approval is not issued here.

Gate Outcomes: directory replacement regression caught foreign directory deletion under path-only ownership | full-workspace whitespace check caught unrelated existing EOF blank lines | scoped syntax/format/overlay checks passed

## Machine readable

```json
{
  "taskId": "2026-11-07-ops-planned-task-closeout",
  "tier": "L2",
  "role": "Generator",
  "localVerification": "passed",
  "baselineCandidateTests": 48,
  "addedCandidateTests": 15,
  "finalCandidateTests": 63,
  "existingOverlayTests": 62,
  "skippedTests": 0,
  "protectedUnchangedFiles": 713,
  "reviewMode": "scoped-self-review-only",
  "independentCodeReview": "pending-root-rerun",
  "independentArchitectureReview": "pending-root-rerun",
  "atomicNamespaceGuarantee": false,
  "assembledFailurePolicy": "retain-unknown-directory-instances",
  "productSealed": false,
  "finalP0Approval": false,
  "sourceReceipt": "resume-o4-candidate-cleanup-source-receipt.json"
}
```
