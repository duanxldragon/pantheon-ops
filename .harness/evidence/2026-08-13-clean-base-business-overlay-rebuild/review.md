# Review

Findings-first review of the clean-base + business-overlay rebuild.

## Findings and resolutions

1. **#1 (blocker) Lock↔rebuild-source disconnect — RESOLVED.**
   `rebuild-from-base.mjs` read the live `../pantheon-base` working tree and never verified the locked artifact. Now resolves Base from `foundation-release.lock.json` → `repoSnapshot` (`repo.tar`, sha256-verified), falling back to `manifest.base.source` only when no lock is present.

2. **#2 (blocker) Uncommitted business source — RESOLVED (Phase 0).**
   Business view refactors + smoke specs + manifest + tool + tests committed before the rebuild, so `git ls-files` is the reproducible source of truth.

3. **#3 (blocker) Verification never executed — RESOLVED (Phase 3).**
   Staged tree verified: backend `go build ./...` + `go vet` + `go test ./modules/business/...`; frontend `tsc -b` + `vite build` + `eslint`; contract `check-business-overlay.mjs` (both source and staged); unit tests 3/3.

4. **#4 (risk) Silent overwrite of Base hook files — RESOLVED.**
   Ops no longer declares `retired_modules.go`/`_test.go` in `businessPaths`; the cmdb retirement spec now lives in `retired_modules_overlay.go` (an `init()` append to Base's `retiredBusinessModules`). `assertBusinessPathsDoNotOverwriteBase` now hard-fails any business path that would overwrite a Base file of different content.

5. **#5 (risk) Old consume toolchain not retired — RESOLVED.**
   Deleted `consume/install/upgrade-from-local` + `shared-foundation-rules` and the byte-sync/drift checks (`check-base-backend-sync`, `check-clean-consumer`, `check-overlay-compile`, `check-inheritance-contract`, `sync-base-shared`, `sync-foundation-i18n`) plus their tests. Rewired `package.json`, `CLAUDE.md`, agent skills, `README.en.md`, and the CI-baseline test to the new `rebuild:from-base` / `check:business-overlay` discipline.

6. **#6 (structural) Go module identity switch — RECORDED, by design.**
   The rebuilt backend adopts Base's multi-module layout: `backend/go.mod` is `module pantheon-base`, and business Go imports are rewritten `pantheon-ops/backend/...` → `pantheon-base/...` by `rewriteBusinessGoImports`. This is intentional — the task packet mandates "preserve Base's repository layout and Go module identity". Ops is now a thin consumer; it no longer owns a full `pantheon-ops` module at the backend root.

## Verification evidence

- `node --test tests/scripts/business-overlay/*.test.mjs` → 3 pass (determinism, generic-path rejection, Base-overwrite rejection).
- Rebuilt backend: `go build ./...`, `go vet ./modules/business/...`, `go test ./modules/business/...` all exit 0.
- Rebuilt frontend: `tsc -b`, `vite build`, `eslint` all exit 0.
- `check-business-overlay.mjs --root .` and `--root .tmp/business-overlay-rebuild` both return `OK`.
- Manifest completeness audit: all 43 tracked `docs/` entries are covered (4 uncovered docs handled: PLATFORM_SRE_EVOLUTION_PLAN added, FOUNDATION_UPGRADE_PATH retired).

## Residual risk

- Business Playwright smoke is **green** — cmdb 9/9, deploy-api 4/4, deploy-pages 10/10 (Phase 4, isolated stack `:8081`; see `summary.md`).
- `go test -race` not run — requires cgo/gcc, unavailable on this machine; plain `go test` used instead.
- The `repo.tar` snapshot was generated locally via `git archive <lock.baseCommit>`; Workstream A (base publishing `repo.tar` in the release `.tgz`) is still pending for v0.10.21+.

## Machine Readable

```json
{
  "taskId": "2026-08-13-clean-base-business-overlay-rebuild",
  "status": "reviewed",
  "findings": [
    {"id": "#1", "severity": "blocker", "title": "Lock↔rebuild-source disconnect", "state": "resolved"},
    {"id": "#2", "severity": "blocker", "title": "Uncommitted business source", "state": "resolved"},
    {"id": "#3", "severity": "blocker", "title": "Verification never executed", "state": "resolved"},
    {"id": "#4", "severity": "risk", "title": "Silent overwrite of Base hook files", "state": "resolved"},
    {"id": "#5", "severity": "risk", "title": "Old consume toolchain not retired", "state": "resolved"},
    {"id": "#6", "severity": "structural", "title": "Go module identity switch", "state": "recorded"}
  ],
  "residualRisk": [
    "go test -race unavailable (no cgo)",
    "Base-side repo.tar packaging pending (Workstream A)"
  ]
}
```
