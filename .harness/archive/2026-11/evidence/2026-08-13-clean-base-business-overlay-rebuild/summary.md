# Evidence Summary

## Linkage

- Task: `docs/harness/tasks/2026-08-13-clean-base-business-overlay-rebuild.task.md`
- Manifest: `business-overlay.json`
- Review: `.harness/evidence/2026-08-13-clean-base-business-overlay-rebuild/review.md`

## Status

Phases 0–6 are complete. The clean rebuild is wired to the locked Base
snapshot, verified end-to-end (backend build/vet/test, frontend tsc/vite/eslint,
contract checks, unit tests, business Playwright smoke), and all six gaps from
the plan are closed. The Ops product tree has been atomically replaced with the
rebuilt tree (Phase 6).

## Commits on `feat/clean-base-business-overlay-rebuild`

- `65ca628` feat: introduce clean-base business-overlay rebuild (Codex's original work)
- `9236bf9` fix(overlay): wire lock→snapshot rebuild + correct frontend import paths
- `7611320` fix(overlay): stop overwriting base retired_modules hook (gap #4)
- `1799a11` refactor(overlay): retire file-by-file sync toolchain (gap #5)
- `7a1ecfe` docs(overlay): findings-first review + evidence summary (gap #6)
- `dd752d4` feat(overlay): atomically replace Ops tree with clean-base rebuild

## Phase 4 — business smoke (green)

Staged tree `.tmp/business-overlay-rebuild` run against a live isolated stack:
backend on `:8081` (fresh MySQL `pantheon_ops_smoke`, `PANTHEON_AUTO_MIGRATE=true`,
admin seed `admin`/`123456`), frontend via vite with `/api` proxied to `:8081`.

Results (all green):

- `tests/smoke/business/cmdb/cmdb-pages.spec.ts` — **9/9 passed**
- `tests/smoke/business/deploy/deploy-api.spec.ts` — **4/4 passed**
- `tests/smoke/business/deploy/deploy-pages.spec.ts` — **10/10 passed**

Runtime notes (environment, not code):

- API specs resolve their base URL from `PANTHEON_API_BASE_URL` (default
  `http://127.0.0.1:8080/api/v1`); the isolated backend lives on `:8081`, so the
  smoke was run with `PANTHEON_API_BASE_URL=http://127.0.0.1:8081/api/v1`. In the
  standard CI/dev layout the backend occupies `:8080` and the default holds.
- Staged tree pinned `@playwright/test` 1.62.1 (chromium-1234 not installed). The
  isolated run used the root `node_modules` playwright 1.61.1 (chromium-1228
  already cached) — version-only substitution, no test or app code changed.

## Phase 6 — atomic replace (done)

Commit `dd752d4` replaces the Ops product tree with the rebuilt tree in a single
atomic commit (481 files changed: 176 added, 240 modified, 20 deleted, plus
rename detection for the Go-module and system-module restructure). Key outcomes:

- Root `go.mod` → `backend/go.mod` (`module pantheon-base`); multi-module layout adopted.
- Base repository layout preserved (`config/`, `grafana/`, `openspec/`, `releases/`,
  `Dockerfile`, `VERSION`, `SHELL_VERSION.json`, `CHANGELOG.md`).
- Business overlay (`bizscope`/`cmdb`/`deploy`) + generated registries + i18n fallbacks.
- `foundation-release.lock.json`, `frontend/README.en.md`, and the
  `check-encoding`/`workflow-baseline` tests added to `repositoryOverlayPaths`
  so a clean rebuild reproduces the full consumer tree.
- ~85 historical shared-source drift files retired (old nested `system/*`, `iam/*`,
  `org/*`, `foundation/*.json` i18n, `list-page.css`, ops QA scripts).

## GitHub-only consumption

`rebuild-from-base.mjs` no longer consumes a sibling `../pantheon-base` working
tree. Base resolves only from `foundation-release.lock.json` → `repoSnapshot`:

- Cached `repo.tar` under `releaseArtifact.localPath` is sha256-verified and used.
- Missing snapshot → downloaded from the locked GitHub release
  (`gh release download <releaseVersion> -p repo.tar --repo <githubRepo>`).
- No lock → hard-fail with a "published GitHub release, not a local working tree" message.

`business-overlay.json` dropped `base.source`; the lock's `baseRepo` now points at
`duanxldragon/pantheon-base`.

## Remaining

- Workstream A: base-side `repo.tar` packaging for v0.10.21+ releases
