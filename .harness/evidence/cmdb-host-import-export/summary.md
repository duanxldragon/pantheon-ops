# Evidence Summary — cmdb-host-import-export

## Linkage

- Task: `.harness/tasks/cmdb-host-import-export/manifest.json`
- Plan: `.harness/tasks/cmdb-host-import-export/plan.md`
- Review: `.harness/evidence/cmdb-host-import-export/review.md`

## Status

Handed off from Claude Code with the host slice implemented but **unverified and
half-finished**: the rest of the import/export batch (group / label / bizscope)
was still broken (compile errors, wrong model fields, missing permission/i18n
seeds), the frontend list components used wrong shared-component props, and the
k8s module was an orphan that broke `go build ./...`.

Both work streams are now complete and verified:

1. **Import/export batch (host + group + label + bizscope)** — implemented,
   seeded, and green across backend/frontend/contracts.
2. **k8s module recovery** — re-integrated into the backend registry and
   frontend overlay (see `k8s-recovery` evidence).

## What was delivered

### Backend

- `cmdb/host/host_import_export.go` + `host_import_export_test.go` (Claude Code's
  original work, verified) — CSV export / template / transactional import reusing
  `pantheon-base/pkg/impexp`.
- `cmdb/group/group_import_export.go` (+ test) — rewired `Export` handler to the
  real service signature; import now hard-fails on unknown `parentName` instead
  of silently attaching to the root.
- `cmdb/label/label_import_export.go` (+ test) — **rewritten**: previous
  implementation referenced non-existent model fields (`valueType`,
  `defaultValue`, `sort`); real model is `category/valueMode/dictCode/status`.
  Handler rewritten to the host-style flow.
- `bizscope/bizscope_import_export.go` (+ test) — verified and covered.
- `cmdb/module.go` + `bizscope/module.go` — added the missing permission keys
  (`business:cmdb:group:import/export`, `business:cmdb:label:import/export`,
  `business:bizscope:import/export`), menu F-seeds, and runtime i18n seeds
  (previously only host was fully seeded).
- `cmdb/seeds_test.go` — asserts the new group/label permission seeds.

### Frontend

- `bizscope/BizScopeList.tsx`, `cmdb/group/CmdbGroupList.tsx`,
  `cmdb/label/CmdbLabelSchemaList.tsx` — fixed to the host reference pattern
  (`ListHeaderActions` `utility` prop + `ImportCsvButton`), wired
  `api.ts` export/template/import calls, and gated the buttons on the seeded
  permissions.
- `i18n/resources/zh-CN.ts` + `en-US.ts` — added the missing business permission
  / import-export keys so the menu contract passes.
- `modules/business/bizscope/index.ts` + `system/iam/menu/business_overlay_component_registry.go`
  — closed the bizscope component whitelist gap found by the menu contract check.

### k8s (separate stream, closed with this batch)

- Backend: `go.mod`/`go.sum` deps (`k8s.io/*` v0.36.3, `gorilla/websocket`
  v1.5.3) + registered the module in `business_overlay_registry.go`.
- Frontend: fixed import depth/icon/`MenuIconKey` in the four k8s pages, mounted
  the module in `businessOverlay.ts` + the component registry, moved the
  ReleaseList create action to `TableBatchActionBar` to satisfy the
  governance-bar contract.

## Verification evidence

- Backend: `go build -buildvcs=false ./...` ✓ ; `go test -buildvcs=false ./...` ✓
  (full suite, incl. cmdb/group/host/label + bizscope).
- Frontend: `tsc -b` ✓ ; `eslint src` ✓ ; `vite build` ✓.
- Contracts: `check-business-overlay.mjs` ✓ (16 declared paths) ;
  `check-menu-contract.mjs` ✓ (26 menus / 34 routes / 151 perms / 34 component
  keys) ; full 13-check prebuild contract chain ✓.

## Explicit gaps

- **Runtime smoke not run** — the DB-backed `test:smoke:business:cmdb` requires
  MySQL root credentials unavailable in this environment. The import/export
  flows are covered by Go unit tests + the contract chain; visual
  desktop/mobile evidence remains a gap. (Previous isolated-stack run of the
  base cmdb pages: 9/9 green, recorded in the Aug 13 rebuild evidence.)
- `go test -race` not run (no cgo/gcc).
