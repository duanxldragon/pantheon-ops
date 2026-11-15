# ARCH lane — independent architecture review of the v0.18.7 consumer delta

Rounds executed by the independent installed architect CLI adapter (ephemeral, default model). Scope: origin/main 0a1ff023 -> consume/base-v0.18.7 (round 1 at 4304970, round 2 at 9b51415).

## Round 1 (commit 4304970) — ARCHITECTURAL STATUS: BLOCKED / VERDICT: REQUEST_CHANGES

**P1 — Inheritance anchors not repinned; packet/evidence incoherence**
- `docs/PROJECT_INHERITANCE.md` (lines 36–37, 80) and `docs/PROJECT_INHERITANCE.en.md` (lines 36–37) still pin `pantheon-base-v0.18.6` / `5553bc0e…` while `foundation-release.lock.json` pins v0.18.7 / `3203624e…`. Both files are declared **Modify** in the task packet and in the manifest, yet neither appears in `git diff --name-only HEAD~1 HEAD`. This violates the sync-close rule and breaks the contract-anchor prerequisite; it must be corrected in this PR.

**P2 — Residual / disclosure gaps**
- `frontend/src/modules/generated/system.ts`: HEAD drops the `// Code generated … DO NOT EDIT` header present in the snapshot. Outside sharedPaths (derived), but the only declared gap concerned untouched business files — this file is touched. Add explicit provenance or restore the marker.
- `docs/release-notes/RELEASE_NOTES_v0.18.7.md`: consumer rewrites base-side preparation wording with Ops evidence. Not a sharedPath, but provenance of the base-authored text is altered; should be flagged as an overlay doc rather than silently edited.
- `.foundation/foundation-release.lock.json` (tracked) still says `lockedRelease: pantheon-base-v0.18.2`; pre-existing staleness, root lock is authoritative — ratchet candidate for cleanup.

**Clear (verified)**
- Chain fidelity: lock anchors (v0.18.7, `3203624e…`, tgz `1daa3e7f…`, repo.tar sha) coherent vs HEAD~1; acquisition/staging/validation/materialization commands logged; review.md honestly records pending panel.
- Snapshot fidelity: full sweep of all 156 changed files vs the immutable snapshot — deviations only go.mod, the allowlisted `business.go` hook, i18n overlay spreads (53-path overlay contract) and the two P2 files. No hand-edit of base-owned shared source.
- Boundaries: `check:shared-drift` passes unmodified (866/856/10, exactly the 10 documented hooks); `check:business-overlay` OK; overlay-hook reference set identical to HEAD~1 — no new hook.
- Module graph: go.mod only promotes `otel/trace`, `proto/otlp`, `protobuf` indirect→direct, all already declared by base v0.18.7 at the same versions; go.sum untouched — no new dependency.
- No gate weakening: `.foundation/allowed-drift.json` unchanged; checker scripts replaced only by stronger snapshot copies; seal ordering verbatim from base. Rollback target (v0.18.6 lock, checksum `ceb95f7b…`) fully recoverable from HEAD~1.

## Round 2 (fix commit 9b51415) — ARCHITECTURAL STATUS: CLEAR / VERDICT: APPROVE

Round-1 fixes all verified in 9b51415; one residual P2 on evidence wording.

**Findings**

- P2 — `.harness/evidence/2026-11-13-base-v0187-consumption/commands.json`: the sweep entry claimed the nine exit-1 gates contain findings "unchanged from origin/main". For `check:harness-docs` this was imprecise: the 6 base task-local doc-link findings are new with this branch's inherited docs. Mitigation verified: the six flagged docs are byte-identical to the immutable v0.18.7 snapshot (assembler-faithful, no hand-edit), summary.md names the class explicitly and routes it Base-first per the inheritance boundary, and the links never existed in Ops history so no consumer-side fix is legitimate without unallowlisted drift. The commands.json sentence and knownGaps were corrected accordingly in the evidence commit.
- No P0/P1.

**Verification of round-1 items**
1. Scope of 9b51415 = exactly the 6 claimed paths.
2. Anchors: zero `v0.18.6`/`5553bc0e` hits in both PROJECT_INHERITANCE docs; `v0.18.7`/`3203624e…` present; root lock (v0.18.7, checksum 1daa3e7f…) and VERSION 0.18.7 mutually consistent. `.foundation/foundation-release.lock.json` v0.18.2 narrative unchanged (pre-existing, accepted round 1).
3. Governance: every branch-diff file maps to manifest create/modify classes; `system.ts` and `scripts/frontmatter-check.mjs` declared with rationale; no CHANGELOG in diff.
4. Inheritance: `git diff origin/main..HEAD -- .foundation/allowed-drift.json` empty; 10 allowlisted entries identical to origin/main; drift gate 866/856/10 exit 0; overlay contract 53 declared paths exit 0; hook reference set identical to origin/main.
5. Module graph: go.mod vs origin/main shows only the three `// indirect` removals (otel/trace, proto/otlp, protobuf); residual snapshot deltas are pre-existing ops business deps; go.sum untouched.
6. Gates: inheritance-contract 0; harness-adoption 0; task-packet 1 with the 2026-11-13 packet [PASS] (all FAIL packets are legacy, none in branch diff); harness-docs 1 (see P2 above).
7. Rollback: `origin/main:foundation-release.lock.json` pins v0.18.6/5553bc0e; v0.18.6 snapshot and all gate/allowlist/workflow configs intact.

## Checks performed (round 2)
- `git diff 4304970..9b51415 --name-only`; `git diff origin/main..HEAD --name-only`; `git rev-parse HEAD`
- version/commit-hash greps on both inheritance docs; lock/VERSION reads; manifest.json expectedFiles dump
- `git diff origin/main..HEAD -- .foundation/allowed-drift.json` (empty); allowed-drift entry count (10)
- `npm run check:shared-drift` (0), `check:business-overlay` (0), `check:inheritance-contract` (0), `check:harness-adoption` (0), `check:task-packet` (1; this packet PASS), `check:harness-docs` (1; 6 inherited findings)
- `diff backend/go.mod` vs origin/main and vs snapshot; `cmp frontend/src/modules/generated/system.ts` vs snapshot (identical); hook greps vs `git grep origin/main`
- `git show origin/main:foundation-release.lock.json`; `git ls-tree origin/main .harness/tasks/`; link-origin greps on origin/main doc versions
