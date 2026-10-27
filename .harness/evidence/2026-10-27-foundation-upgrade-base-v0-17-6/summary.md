# Ops Consumption of pantheon-base-v0.17.6

## What was consumed

- Locked release: `pantheon-base-v0.17.6`, release line `release/0.17`, base commit `c9bc6c0f5d351016afb4fffdedf12e72cc73e61d`.
- Materialized under `.foundation/releases/pantheon-base-v0.17.6/` (tgz + `repo.tar` + extracted `repo/`).
- Both asset digests were re-hashed locally after materialization and matched the published `.sha256` sidecars and `foundation-release.lock.json`:
  - tgz `dbf1edd8f4bd082baee817c8e7797d298cf70f20d41b38c60d7bd39a5f362e35`
  - repo.tar `7a44e9ce0ed642e6d2d1f1508ef488292c46148da4a9e42a3f04ad848fda645d`
- The rebuild was run through the lock path (`node scripts/business-overlay/rebuild-from-base.mjs` with no `--base`), so the consumer only ever sees the published release snapshot, never the local Base working tree (which is ahead of the tag).

## Switch shape

Applying the rebuilt tree produced `418 files changed, 32638 insertions(+), 2573 deletions(-)` staged as 136 added / 2 deleted / 275 modified / 5 renamed. Five renames are pure case changes (`AppModalActions.ts` → `appModalActions.ts` and four pattern siblings). Because Windows has a case-insensitive filesystem, a plain copy cannot express a case-only rename; the old name had to be removed from the index and disk first so Git records them as `R` rather than leaving both spellings.

## Consumer-side repairs the swap forced

These are ops-side defects that the new Base surface exposed; each was fixed at its root rather than suppressed.

1. **`list-page.css` import paths (31 files).** Base moved the stylesheet to `src/assets/list-page.css`; ops business pages still imported it through the old `modules/business/shared/...` path. Rewritten per file to the correct relative depth (`../../../../assets/list-page.css` or `../../../assets/list-page.css`).
2. **`Input.Password` → `PasswordInput` (6 sites in 4 files).** Base's password-toggle accessibility contract only accepts the `PasswordInput` pattern. Fixed in `CmdbHostDetail.tsx` (1), `DeployTemplateList.tsx` (2), `AlertCenter.tsx` (1), `MonitoringCenter.tsx` (2), adding the default import at each file's correct depth. `check:password-toggle` went from 6 findings to `0 finding(s) across 327 file(s)`.
3. **Missing `check:shared-drift` script.** `.github/workflows/inheritance-drift-detection.yml` invokes `npm run check:shared-drift`, but the script entry did not exist in `package.json` after the merge, so that job could never pass. Added the entry in both `package.json` and `mergePackages` in the rebuild script, so it survives future rebuilds.
4. **Ops-only archived DDL.** The 23 pre-overlay business DDL files under `docs/migrations/legacy-ops-business-v010x/` were registered in `business-overlay.json` → `repositoryOverlayPaths` so they survive rebuild instead of relying on "we never delete" luck. Declared/generated file count moved 723 → 748 (the last increment includes this task's own packet, manifest and evidence).
5. **`gofmt` drift.** 18 business Go files repopulated by the rebuild were unformatted; `gofmt -w` brought `gofmt -l .` to zero.
6. **Dead code removed.** `backend/modules/system/iam/tenant/tenant_switch.go` (`RegisterSwitchRoutes` has no caller; Base serves `/auth/tenants/switchable|current|switch` from `backend/modules/auth/module.go:129-131`) and `backend/pkg/database/migrations/add_oidc_fields.sql` (name does not match the `NNNNNN_*.up.sql` pattern, so golang-migrate ignores it; Base folded those columns into `000029`).

## Schema consequence

The consumed release moves the migration baseline to `000030` and seeds a new tenant-operations menu group with 5 permission codes. Ops inherits that schema; runtime verification of startup plus migrate is recorded as an explicit gap, not claimed.

## Verification summary

See `commands.json` for per-command readings. Everything executed passed: overlay contract (53 declared paths), shared drift (805 identical / 10 allowlisted / 160 skipped, on both the staged tree and the swapped worktree), 13/13 business-overlay tests, backend build/vet/gofmt/test (63 ok, RC 0) and the CI rebuilt-tree form (19 packages ok), frontend `tsc -b` / `vite build` / `eslint --max-warnings 0` plus all 14 prebuild contract checks, and the governance set (structure 2233 files, encoding 2060 files, frontmatter 404 docs, duplication PASS).

## Known gaps

- No runtime acceptance run (backend startup + `golang-migrate` to `000030` + tenant-operations seeding) in this evidence set; no database credentials are recorded here.
- Ops has no consumer-side versioned migration channel; four business modules still use `AutoMigrate`.
- Historical `.harness/evidence/**/commands.json` packages predate the object schema and fail strict validation; that debt predates this swap and was not rewritten here.
- `check:harness-docs`, `check:ui-quality-gate`, `check:harness-visual` and `check:generated` still report findings on files untouched by this swap.
