# Pantheon Base v0.16.0 — Release Notes

**Release date**: 2026-10-12
**Tag**: `pantheon-base-v0.16.0`
**Previous release**: `pantheon-base-v0.15.0`

## Summary

v0.16.0 is an internal-engineering release on top of v0.15.0. It carries no breaking
changes, adds no system module, and changes no consumer-facing interface: no API route,
permission code, menu seed, i18n key or shared-contract shape moves. It packages the three
workstreams merged after v0.15.0 — frontend/backend naming correspondence, a tiered and
module-split smoke suite, and dependency/quality hygiene.

Because nothing in the consumer contract changed, `pantheon-ops` can upgrade on demand;
there is no migration beyond the v0.15.0 upgrade path.

## Changed

### Frontend/backend naming and directory correspondence (#371)

- Go package names now equal their directory leaf across `system/{iam,org,config,audit,i18n}`
  (for example `system/iam/user` is `package user`, not `package iam`). Import **paths** are
  unchanged; the 45 non-canonical aliases were dropped or renamed to the package name, so
  "directory name == package name" is now a reliable locator.
- `frontend/src/modules/system/*` now mirrors the backend's three-level shape
  (`system/iam/*`, `system/org/*`, `system/config/*`). `componentKey`, the logical `module`
  metadata, route paths, permission codes and the menu seed are all unchanged.
- `core/permissions → core/permission`, `core/settings → core/setting`,
  `core/layout/index.tsx → core/layout/AppLayout.tsx`, and five PascalCase non-component
  `.ts` files renamed to camelCase; a path alias was added for the shared layer.

### Tiered, module-split smoke suite (#372, #373, #374)

- The ~3.7k-line `system/system-pages.spec.ts` monolith was split into per-module spec files
  (`system-routes`, `system-iam`, `system-config`, `system-i18n`, `system-audit`,
  `system-org`, `system-platform`) over a shared `system-pages-helpers.ts` barrel. The split
  is byte-for-byte equivalent to the original (59/59 test blocks, 845/845 helper lines).
- Smoke now runs in tiers, selected from **changed paths** rather than a manual label:
  - **L0 static** (`test:smoke:l0`) — smoke wiring contracts, no browser/server; part of `prebuild`.
  - **L1 critical path** (`test:smoke:l1`) — shell/pagination contract plus every system route; **blocking on every PR**.
  - **L2 module** (`test:smoke:module:<module>`) — one module suite; **blocking, but only for touched modules**.
  - **L3 full** (`test:smoke:all`) — everything including the viewport matrix; report-only / pre-release.
- New `scripts/resolve-smoke-modules.mjs` maps changed paths to module suites (covered by 13
  unit tests). Any change not unambiguously inside one module resolves to **every** module, so
  a shared-core edit can never silently skip a suite.
- Quality Gates gained a `pr-smoke-modules` job that runs only the touched suites.

## Fixed

### Security / dependency hygiene (#370)

- Dependabot alert #43 (`sprintf-js`, GHSA-hp3w-g68c-fv3c / CVE-2026-97058, DoS) in the
  frontend **dev** dependency graph is cleared by pinning `@istanbuljs/load-nyc-config`'s
  `js-yaml` to the hoisted `^5.4.2` via npm `overrides`. No runtime dependency changed.

### Quality hygiene (#372, #374, #375)

- Cleared the 17 SonarCloud code smells that the helper-file split re-surfaced on `main`
  (4× S7763, 1× S7503, 1× S4043, 11× justified `// NOSONAR`), restoring the Release Gate.
- Removed the empty `openspec/` skeleton directory (three `.gitkeep` files; no OpenSpec change
  ever landed in it).

## Quality

- SonarCloud: **0 unresolved issues** on `main`; overall duplication **2.8%**.
- CodeQL: 0 open error/critical. Dependabot: 0 high/critical.
- Smoke: PR Smoke Modules runs all 11 suites (118 passed / 0 failed) when the change scope
  resolves to every module; L1 runs on every PR.

## Verification

- Backend: `go build ./...`, `go vet ./...`, `gofmt -l`, `check:generated` clean.
- Frontend: `tsc -b`, `eslint`, `vite build`, vitest 163/163.
- Smoke: L0–L2 and the module suites executed in CI; `resolve-smoke-modules` 13/13 unit tests pass.
- Hosted: required checks and Release Gate green on the release commit.

## Consumer impact (`pantheon-ops`)

No base-side interface changed in this release — no route, permission code, menu seed, i18n key
or contract shape moved. `pantheon-ops` can adopt `pantheon-base-v0.16.0` on demand; the consumer
lock, inheritance snapshot and business smoke remain a separate consumer-repository follow-up.
