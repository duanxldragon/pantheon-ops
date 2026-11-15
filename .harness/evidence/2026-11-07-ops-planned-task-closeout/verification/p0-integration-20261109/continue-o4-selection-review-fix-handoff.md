# O4 selected-input review repair handoff

Parent: [2026-11-07-ops-planned-task-closeout](../../../../../docs/harness/tasks/2026-11-07-ops-planned-task-closeout.task.md). L2; installed Generator/executor; Ops consumer tooling. Historical request SHA256 `8a866ac9c1d608558ee3d1e04805cd7f8a5e0a444b8fab2fc1dfe8ac7bd84a82` remains immutable. This handoff records scoped implementation and verification; independent rereview and actual complete assembly remain pending.

## Changes

- `.gitignore`: classify only the exact `o4-independent-code-review-20261110-01/fixtures/` run root as retained test assets. Independent ARCH’s original 175 diagnosed records all belong there, including 13 overlength records. Existing two exclusions remain. Reports stay selected; physical bytes and links remain in place. No blanket evidence/fixture exclusion or larger limits.
- `rebuild-from-base.mjs`: explicitly parse directory records, expand permitted nested metadata repositories through their own NUL-safe Git leaf inventory, and include directory ancestors of declared leaves. Preserve ignored tracked files; explicitly reject unsupported product/foreign directories, gitlinks, unresolved stages, unsafe paths and links.
- `foundation-candidate.mjs` / `rebuild-from-candidate.mjs`: bind replay reads to captured selected files, actual returned byte length/hash and existing descriptor/path/stamp/ancestor guards. Unselected or mismatched reads refuse publication. Base reads use the descriptor inventory; standalone replay also freezes its inputs. Original failure and deferred cleanup semantics remain intact.
- Assembly and replay reuse one codepoint-ordered locale leaf plan, including retained Base leaves. Ops replaces the same Base leaf; later paths override duplicate keys deterministically. Object insertion order agrees. Undeclared/ignored locales stay excluded.
- `foundation-candidate.test.mjs`: 11 additional regressions cover these contracts. Other source edits were preserved.

## Verification

[Machine evidence](continue-o4-selection-review-fix-evidence.json) records full commands, environment, counts, source hashes, logs and preservation paths.

Meaningful RED against the original implementation: **0/8 passed**, eight failures, zero skips. Cached package, frontend-package and locale bytes were accepted; directory selection and both prefix-directory locale cases failed. GREEN on final current code: **198/198**, zero failures/skips, covering complete candidate/ownership, business-overlay and drift suites, including all existing safety tests. Final focused regressions: **11/11**.

```powershell
node --import ./.local/ops-acceptance/continue-o4-selection-review-fix-preload.mjs --test --test-reporter=tap scripts/business-overlay/foundation-candidate.test.mjs scripts/business-overlay/candidate-ownership.test.mjs tests/scripts/business-overlay/*.test.mjs
```

Process-local `core.longpaths=true`; new Node fixture roots and explicit Go caches/GOPATH/module cache are under `.local`; Go proxy/sumdb disabled. No new `.harness` cache. `npm run check:business-overlay` passed (56 declarations). `node --check` passed on all seven reviewed scripts. Native TypeScript **6.0.3 LanguageService: seven files, zero errors**, no suppressions; `checkJs=false`, so this does not prove full type safety.

Historical CODE attack script copied to a new `.local` root: **6/6 groups** meet corrected expectations. Original script/results remain immutable. Cached raw package attack produces no receipt; locale counterexamples now succeed with canonical deterministic outputs. This tests the explicit read contract and makes no OS-attack claim.

Final real-root **read-only selector/capture succeeded: 5112/5112 leaves**, Git selection stable before/after. Earlier observations have different counts because concurrent work added evidence; each recorded capture agrees internally. No real complete assembly was run.

## Preservation and pending work

Before/after proof covers **30,598 original entries / 19,069 file hashes**, including independent counterexamples, reports, hardlinks and excluded caches: hash, inode, nlink, size, mode, mtime and ctime match. Historical descriptor/archive pins remain exact; original **1141/1141 snapshot** and **4634/4634 report** hashes and physical metadata match. Current device metadata matches between observations; no historical full-width device-ID equivalence is asserted.

Gate Outcomes: source-byte binding caught injected reads | locale plan caught precedence/insertion drift | selector classified the exact retained test assets while preserving boundaries.

Stop condition reached: scoped concrete fixes plus fresh tests/evidence complete. Dispatcher owns real complete assembly; independent CODE/ARCH rereview remains pending. No self-approval, P0 closure, runtime acceptance, official consumption, seal, release, lock, remote or credential action.
