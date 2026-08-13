# Pantheon Ops

Chinese operational rules: [AGENTS.md](./AGENTS.md)

pantheon-ops inherits pantheon-base as its foundation. Treat `pantheon-base` as the standard backoffice source of truth for platform, system domains, shared UI rules, backend contracts, permission, menu, i18n, audit, and acceptance standards.

pantheon-ops exists to add operations-domain business modules. It must not become a forked copy of the foundation.

## Maintainer Contract: three touchpoints only

The maintainer intervenes at exactly three points — (1) requirement clarification at intake (batch ALL questions once, produce In/Out/acceptance criteria, then stop asking), (2) gate-policy decisions (red gates, exemptions, rule changes), (3) final visual/functional acceptance. Between those, run autonomously: no mid-task confirmations for reversible in-scope work; gates and evidence replace verbal confirmation. See `../pantheon-harness/architecture/methodology/workflow-routing.md` → Human Touchpoints.

## Hard Stop: Claude does not implement

Same role boundary as pantheon-base: Claude Code in this repository is the **planner and reviewer**, not the executor.

After a plan is approved, Claude MUST NOT use Edit/Write to implement code changes. Delegate implementation to Codex via `codex exec ... -C pantheon-ops`. Everything under `backend/` and `frontend/src/` belongs to Codex.

Claude may directly modify only: `docs/harness/`, `.harness/`, `CLAUDE.md`, `AGENTS.md`, `DESIGN.md`, root config files (`.gitignore`, `package.json` scripts), and `.github/workflows/`.

Codex model tiers are shared with base: read `../pantheon-base/.codex/model-tiers.json` and apply the same autoSelect rules (see `../pantheon-base/CLAUDE.md`).

## Business work reading order

Before implementing, reviewing, debugging, or designing in this repository, read only the relevant files in this order:

1. `../docs/WORKSPACE_INHERITANCE.md`
2. `AGENTS.md`
3. `docs/PROJECT_INHERITANCE.md`
4. `../pantheon-base/DESIGN.md`
5. `../pantheon-base/AGENTS.md`
6. `../pantheon-base/docs/README.md`
7. matching `../pantheon-base/docs/contracts/*`
8. matching `../pantheon-base/docs/designs/*`
9. matching `../pantheon-base/docs/acceptances/*`
10. local business docs under `docs/designs/BUSINESS_*`

Repository-local workflow skills live under:

- `.agents/skills/README.md`

## Layer ownership

- `pantheon-base` owns `platform` and `system/*`.
- `pantheon-ops` owns `business/*` and repository-local business docs.
- Business modules may use base extension points, shared contracts, and public packages.
- Business modules must not directly rewrite base platform or system-domain behavior.

Do not fix platform or system-domain drift locally in pantheon-ops. If the behavior should apply to the standard backoffice, change `pantheon-base` first, then upgrade `pantheon-ops`.

## Sync discipline

pantheon-ops consumes pantheon-base as a **locked release snapshot**, not a file-by-file sync. The consumer tree is rebuilt deterministically from the locked Base snapshot (`foundation-release.lock.json`) plus the declared business overlay (`business-overlay.json`).

Before any non-trivial PR:

```powershell
# run from pantheon-ops repo root
node scripts/business-overlay/check-business-overlay.mjs
node scripts/business-overlay/rebuild-from-base.mjs --target .tmp/business-overlay-rebuild
```

- `check-business-overlay.mjs` verifies the overlay contract: every declared `businessPaths` exists and no business file still imports the old `pantheon-ops/backend` module path.
- `rebuild-from-base.mjs` deterministically produces the consumer tree from the locked snapshot; at the locked `baseCommit` it must be byte-for-byte reproducible (`.business-overlay-report.json`).
- The business overlay must not overwrite Base hook/source files (enforced by `assertBusinessPathsDoNotOverwriteBase`). To extend Base business semantics, inject via a dedicated overlay file — see `backend/modules/business/retired_modules_overlay.go`.
- To upgrade: update `foundation-release.lock.json` to the new release, re-lock the `repoSnapshot`, then re-run `rebuild-from-base.mjs`.

## Design system

Always read `../pantheon-base/DESIGN.md` before visual or UI decisions, then read local business design docs if the task touches `business/*` UI.

Key inherited constraints:

- Font: Source Sans 3 for body/UI, JetBrains Mono for code.
- No Inter, no radial-gradient, no large button shadows, no non-standard font weights.
- Radius: 4/6/8/12px. No pill-radius cards.
- All colors via Pantheon CSS tokens, never raw Arco variables.
