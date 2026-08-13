# Evidence Summary

## Linkage

- Task: `docs/harness/tasks/2026-08-13-clean-base-business-overlay-rebuild.task.md`
- Manifest: `business-overlay.json`
- Review: `.harness/evidence/2026-08-13-clean-base-business-overlay-rebuild/review.md`

## Status

Phases 0–5 are complete. The clean rebuild is wired to the locked Base
snapshot, verified end-to-end (backend build/vet/test, frontend tsc/vite/eslint,
contract checks, unit tests, and business Playwright smoke), and all six gaps
from the plan are closed. The current Ops product tree remains unchanged until
the atomic replace (Phase 6).

## Commits on `feat/clean-base-business-overlay-rebuild`

- `65ca628` feat: introduce clean-base business-overlay rebuild (Codex's original work)
- `9236bf9` fix(overlay): wire lock→snapshot rebuild + correct frontend import paths
- `7611320` fix(overlay): stop overwriting base retired_modules hook (gap #4)
- `1799a11` refactor(overlay): retire file-by-file sync toolchain (gap #5)
- `7a1ecfe` docs(overlay): findings-first review + evidence summary (gap #6)

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

## Remaining

- Phase 6: atomic replace of the Ops tree with the rebuilt tree
- Workstream A: base-side `repo.tar` packaging for v0.10.21+ releases
