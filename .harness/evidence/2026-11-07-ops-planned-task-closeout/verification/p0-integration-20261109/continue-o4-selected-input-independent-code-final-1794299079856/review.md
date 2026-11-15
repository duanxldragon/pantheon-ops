# CODE REQUEST CHANGES

Parent: docs/harness/tasks/2026-11-07-ops-planned-task-closeout.task.md. L2, Ops consumer inheritance tooling; no platform/system implementation or architectural verdict.

8 files reviewed; 3 issues: CRITICAL 0, HIGH 2, MEDIUM 1, LOW 0. Stage 1 is not satisfied; no normal quality approval follows.

1. **[HIGH] scripts/business-overlay/rebuild-from-base.mjs:27 — Valid Git directory records abort selection.** The actual root Git query returned 7,142 records, including 175 valid embedded-repository directory records ending in `/`; parsing throws `Unsafe declared Ops input path` before stage creation. A fresh small fixture containing `.harness/nested-evidence/.git` and ordinary report.md reproduces this exactly, with report preserved and no stage. The two exact new ignores do not fix this remaining primary input-shape contract. **Fix:** explicitly handle Git directory records in the shared selection contract and derive an exact supported leaf plan for capture/copy/verification. Do not silently strip the slash, recursively capture unselected data, or widen the ignores. Add an embedded-repository regression and verify actual root selection. [Proof](D:/workspace/go/pantheon-platform/pantheon-ops/.local/ops-acceptance/continue-o4-selected-input-independent-code-20261110-review-1794299079856/nested-git-evidence-result.json); [actual root](D:/workspace/go/pantheon-platform/pantheon-ops/.local/ops-acceptance/continue-o4-selected-input-independent-code-20261110-review-1794299079856/actual-selector-check.json).

2. **[HIGH] scripts/business-overlay/rebuild-from-base.mjs:713 — Package replay is not bound to frozen input bytes.** Assembly reads Ops package.json at line 392; the new canonical replay at line 730 reads it again through this raw reader, whose hash check covers only business locale paths. Injecting a cached raw-read buffer for that exact package path produced a successful fixture receipt with `sourceStable:true`, while its recorded source hash remained the unchanged on-disk package and its generated script was `echo unpinned cached recipe`. This is explicit read-cache fault injection, not proof of an OS race exploit. **Fix:** use a source-bound reader that rejects unselected reads and checks returned bytes against the frozen source hashes; pass that same frozen contract into canonical derivation. Add cached-read and copied-byte mismatch regressions and preserve primary rejection evidence. [Proof](D:/workspace/go/pantheon-platform/pantheon-ops/.local/ops-acceptance/continue-o4-selected-input-independent-code-20261110-review-1794299079856/adversarial-results.json).

3. **[MEDIUM] scripts/business-overlay/rebuild-from-base.mjs:354 — Canonical locale ordering differs from assembly.** Replay sorts whole selected paths; assembly at line 651 still recursively visits directories. `cmdb-extra/locales/en-US.json` sorts before `cmdb/locales/en-US.json` in replay, while directory traversal visits cmdb first. Even disjoint keys produce different JSON property order and fail exact canonical byte comparison; overlapping keys also change precedence. **Fix:** use the same explicitly ordered actual input plan for both generation and replay, including retained Base inputs, and cover prefix-related directories with disjoint and overlapping keys. [Proof](D:/workspace/go/pantheon-platform/pantheon-ops/.local/ops-acceptance/continue-o4-selected-input-independent-code-20261110-review-1794299079856/adversarial-results.json).

Validation: request SHA and all 8 source hashes matched before/after; native TypeScript 6.0.3 LanguageService on 7 .mjs files: 0 errors, checkJs=false (no full type-safety claim). lsp_diagnostics/ast_grep/CodeGraph tools were unavailable; native AST review found no empty catch or credential literals. Focused tests: 28/28, 0 skip. Root overlay: 63/63, 0 skip with process-local core.longpaths=true; initial 57 pass/6 MAX_PATH failures retained. Root-suite Go fixture subprocesses inherited default Go caches/GOPATH; writes were not audited, so that run is not cache-isolation evidence.

Original descriptor SHA 2f105d76e38c82e8c301d816269ea26669a7a0291a5860be74d6ca71281a9ce6 and artifact SHA matched; 1,141-file descriptor retained. Original hardlink inode/nlink/57 bytes/content hash and cache sizes match; device ID differs from historical full representation but matches its low 32 bits. No full device-ID equality claim. Historical sealed tree/report and prior failures were not changed or cleaned. All new fixtures/logs are under .local/ops-acceptance/continue-o4-selected-input-independent-code-20261110-review-1794299079856; only this new review report is written under EV.

Stop condition: executor must repair these contracts and submit fresh evidence/review. No real consumer receipt, runtime/P0/source-wide approval, official consumption, seal, release, or lock change is asserted.

```json
{
  "lane": "CODE",
  "decision": "REQUEST CHANGES",
  "independent": true,
  "subjectSHA256": "8a866ac9c1d608558ee3d1e04805cd7f8a5e0a444b8fab2fc1dfe8ac7bd84a82",
  "requestSHA256": "8a866ac9c1d608558ee3d1e04805cd7f8a5e0a444b8fab2fc1dfe8ac7bd84a82",
  "sourceHashes": {
    ".gitignore": "04b7bb31311f121d90dbd72fdf8cff3ce6127ee73b6b77469bbcf147a7e7df12",
    "scripts/business-overlay/foundation-candidate.mjs": "02deaf187133b267622359e5756100262f99c48b031f76db47716157c8c974f1",
    "scripts/business-overlay/rebuild-from-candidate.mjs": "b06fa5a04595f9a245920814f3ba0e31b5f3a9009061c249c62ef69fd8030f06",
    "scripts/business-overlay/rebuild-from-base.mjs": "24e7e834622d7e61d240a25aedfd5c914cee86ea9780dc7280a31ce55ee06c76",
    "scripts/business-overlay/foundation-candidate.test.mjs": "6f2cb9a366de1fdb71838ccc89ccfc76af39d68570aafde06cbe660dfd9b8894",
    "scripts/business-overlay/candidate-ownership.test.mjs": "30f4c185051b232d414b4867c7a9b35601d335405d8477b26dcb236be09f6062",
    "scripts/business-overlay/check-shared-drift.mjs": "a6ee2b769c1d27167c7f426ad913981c28b99f462054342e826dc194a1c4302b",
    "tests/scripts/business-overlay/check-shared-drift.test.mjs": "b79af64de465f3b1e2efda7e14d595fb2ba24ee46bdb36b04f2e79824338be36"
  },
  "sourceHashesVerifiedBeforeAndAfter": true,
  "scope": "O4 selected-input Ops consumer tool delta only",
  "issues": [
    {
      "id": "CODE-SEL-01",
      "severity": "HIGH",
      "file": "scripts/business-overlay/rebuild-from-base.mjs",
      "line": 27,
      "issue": "Git emits valid untracked embedded repository directory records; parser rejects them before selected-input assembly",
      "evidence": ".local/ops-acceptance/continue-o4-selected-input-independent-code-20261110-review-1794299079856/nested-git-evidence-result.json"
    },
    {
      "id": "CODE-SEL-02",
      "severity": "HIGH",
      "file": "scripts/business-overlay/rebuild-from-base.mjs",
      "line": 713,
      "issue": "Canonical package replay accepts unpinned cached raw-read bytes and can publish a sourceStable fixture receipt",
      "evidence": ".local/ops-acceptance/continue-o4-selected-input-independent-code-20261110-review-1794299079856/adversarial-results.json",
      "proofBoundary": "Injected cached fs.readFileSync response, not an OS race exploit"
    },
    {
      "id": "CODE-SEL-03",
      "severity": "MEDIUM",
      "file": "scripts/business-overlay/rebuild-from-base.mjs",
      "line": 354,
      "issue": "Sorted full-path locale replay differs from assembly directory traversal; disjoint keys already trigger byte inequality",
      "evidence": ".local/ops-acceptance/continue-o4-selected-input-independent-code-20261110-review-1794299079856/adversarial-results.json"
    }
  ],
  "officialConsumption": false,
  "actualCandidateAssemblySuccess": false,
  "P0Closed": false,
  "languageDiagnostics": {
    "tool": "TypeScript native LanguageService",
    "version": "6.0.3",
    "files": 7,
    "errors": 0,
    "checkJs": false,
    "lspToolAvailable": false,
    "fullTypeSafetyClaim": false
  },
  "verification": {
    "focusedTests": {
      "passed": 28,
      "failed": 0,
      "skipped": 0
    },
    "overlayTests": {
      "passed": 63,
      "failed": 0,
      "skipped": 0,
      "processLocalGitLongpaths": true
    },
    "initialOverlayRun": {
      "passed": 57,
      "failed": 6,
      "cause": "Git MAX_PATH with long independent fixture root; failed TAP retained"
    },
    "goCacheIsolation": "Go fixture subprocesses inherited default cache/GOPATH; cache writes not audited. Not cache-isolation evidence."
  }
}
```
