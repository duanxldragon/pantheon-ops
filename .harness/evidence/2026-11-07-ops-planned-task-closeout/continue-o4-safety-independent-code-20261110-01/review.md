# Independent CODE safety review — O4

**Verdict: COMMENT.** No reproducible code/spec/security defect remains in this scoped repair (CRITICAL 0, HIGH 0, MEDIUM 0, LOW 0). The four previous findings pass independent reproductions. Mandatory `lsp_diagnostics` is not callable, so this lane cannot issue APPROVE under the supplied reviewer rule. This diagnostic gap is not a source-code finding and does not imply a failed typecheck. Root owns the fresh-attempt gate together with independent ARCH; this report grants no P0, official-consumption, release or final assembly approval.

Parent: `2026-11-07-ops-planned-task-closeout`, L2. Business-owned overlay lifecycle tooling; logical ownership boundaries, no platform/system-domain physical change. Five requested files reviewed, plus focused context from `foundation-candidate.test.mjs` and `tests/scripts/business-overlay/check-shared-drift.test.mjs` (seven pinned files). Spec: Generator safety repair summary/plan and original independent CODE review. Applicable repository/base instructions and `repo-verify` read; no implementation, children, remote/credentials, lock/waiver/release changes.

## Stage 1 and root-cause guard

Both pass before Stage 2. Errors remain failures; deferred cleanup throws with the primary cause and never reports successful cleanup. No new masking fallback identified. Existing official assembler/locale compatibility paths are unchanged by this safety repair.

- Prior HIGH1: `foundation-candidate.mjs:591`, `:610`, `:613` authenticate exclusive-created/known regular-file descriptors, recheck ancestors, and mutate via descriptor. `rebuild-from-base.mjs:668` and `rebuild-from-candidate.mjs:266` guard report/receipt boundaries. Observed replacements before copy, existing-file open and receipt refuse; foreign bytes remain. Existing descriptor-write attack remains green.
- Prior HIGH2: `foundation-candidate.mjs:561` explicitly treats post-mkdir identity as observation, with no directory deletion authority. Independent empty and nonempty mkdir-to-registration substitutions retain the replacement and original; cleanup attempts zero deletes.
- Prior HIGH3: `foundation-candidate.mjs:944`, `:1027`, `:1098` preflight the full relevant tree against recorded identities. Unknown ordinary siblings refuse entire cleanup and known-file exclusion before any unlink; independent before/after trees retain all bytes and instances. Missing/replaced recorded objects also refuse.
- Prior MEDIUM4: `rebuild-from-base.mjs:637` excludes exact files or directory descendants before copy. Independent `docs/base.md` versus `docs` cases pass; `docs/base.md.backup` survives the file exclusion. Shared drift still requires explicit allowances for excluded shared files.
- Go: `foundation-candidate.mjs:1037` uses retained alternate modfiles outside assembled. New independent success-result attacks on an ancestor and target file refuse import and preserve foreign mod/sum bytes. Thrown primary error objects and nested causes survive identity/cleanup refusal (`:1066`, `:1102`). Existing real offline normalization test passes.
- Canonical-derived/report/source/import/migration and shared allowlist contracts remain covered by focused green suites; `check-shared-drift.mjs` and its focused test retain the earlier repair hashes unchanged.

## Validation and integrity

Node v24.14.0. `node run-suites-v2.mjs`: exit 0; ownership 16/16, candidate-selected 48/48, focused overlay 47/47: **111 passed, 0 failed, 0 skipped**. This is selected coverage, not a rerun of the full 171 suite or volume/executable-receipt suites. Exact commands/exits: [suite-commands-v2.json](suite-commands-v2.json).

`node --import <file-URL to preserve-new-fixtures.mjs> independent-reproductions.mjs`: exit 0, **9/9** independent checks; [results](independent-reproductions.json). All NEW fixture roots retained and listed in [fixture-roots.jsonl](fixture-roots.jsonl). Test-only recursive teardown retains bounded subtrees by identity-checked rename; production candidate cleanup is not stubbed into success. Initial Windows absolute-path ESM-loader failures (three exit-1 starts, no test bodies) remain in original logs; v2 uses a valid file URL.

`node final-validation.mjs`: exit 0. Seven `node --check` exits 0; `rg` literal security/mutation scan exit 0, reviewed matches; `node scripts/business-overlay/check-business-overlay.mjs` exit 0 (56 paths). Full source SHA256 values, before/after matches and per-file diagnostic gaps: [validation-and-final-hashes.json](validation-and-final-hashes.json). All seven sources and 35 protected original/sealed assets match starting pins. No full retained 1141/4634 byte-inventory rehash claimed in this review. LSP, CodeGraph and ast-grep unavailable; native syntax/literal scans do not establish semantic type safety.

## ARCH/root boundary and stop condition

Unproven directories remain retained, explicitly deferred. Node/Windows offers no atomic mkdir authentication or ancestor-relative open/unlink; checks observe replacements but do not close the final-check/syscall race. Authenticated file unlink remains a path syscall; Go workspace outputs are stable-read inputs, not authenticated creation provenance. These documented native limits require ARCH/root threat-boundary judgment and are not speculative new code blockers. Scope ends at review evidence for a fresh staged candidate attempt. No actual fresh assembly, full frontend/Go/build, paired runtime, official consumption or release performed.

## Machine Readable

```json
{
  "taskId": "2026-11-07-ops-planned-task-closeout",
  "lane": "CODE",
  "tier": "L2",
  "verdict": "COMMENT",
  "filesReviewed": 7,
  "requestedFilesReviewed": 5,
  "severityCounts": { "CRITICAL": 0, "HIGH": 0, "MEDIUM": 0, "LOW": 0 },
  "priorFindingsIndependentlyVerified": 4,
  "stage1": "pass",
  "rootCauseGuard": "pass",
  "existingTestsPassed": 111,
  "independentChecksPassed": 9,
  "failed": 0,
  "skipped": 0,
  "sourceHashesStable": true,
  "protectedPinsStable": 35,
  "lspDiagnostics": "unavailable",
  "semanticTypeSafety": "not established",
  "productEdits": false,
  "actualCandidateAssembly": "not run",
  "freshAttemptApproval": "root/ARCH gate pending; no formal CODE APPROVE because required LSP unavailable",
  "releaseApproval": false,
  "evidenceDir": ".harness/evidence/2026-11-07-ops-planned-task-closeout/continue-o4-safety-independent-code-20261110-01"
}
```
