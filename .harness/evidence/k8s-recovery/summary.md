# Evidence Summary — k8s-recovery

## Linkage

- Recovery source: `.harness/tasks/k8s-recovery/recover.py` (replays the Claude
  Code transcript's Write/Edit ops against `business/k8s` files)
- Companion closeout: `.harness/evidence/cmdb-host-import-export/`

## Status

Claude Code left a k8s business module as an **orphan**: the backend files
existed but were unregistered (no `business_overlay_registry.go` entry) and
compiled against missing deps, which broke `go build ./...`; the frontend
pages were not mounted and had broken imports. The module is now fully
re-integrated and the repository gates are green.

## What was delivered

### Backend (`pantheon-ops/backend/modules/business/k8s/`)

- Added missing deps to `go.mod`/`go.sum`: `k8s.io/client-go`, `k8s.io/api`,
  `k8s.io/apimachinery` (v0.36.3) and `github.com/gorilla/websocket` (v1.5.3) —
  pinned to the versions already present in the module cache, which is the
  strongest evidence of the versions the recovered code was originally written
  against. `go mod tidy` backfilled transitive `go.sum` entries.
- Registered the module in `business_overlay_registry.go`.
- `go build -buildvcs=false ./...` and the full `go test` suite pass.

### Frontend (`pantheon-ops/frontend/src/modules/business/k8s/`)

- Fixed relative import depth (`../../../../` vs `../../../`) in
  `ClusterList`/`ClusterForm`/`ReleaseList`/`WorkloadList` and the 
  `list-page.css` reference.
- Replaced invalid icons (`IconRollback`, `IconMindMapping`) and `MenuIconKey`
  values with the legal set from `src/core/menu/icon.tsx`.
- Mounted the module: `businessOverlay.ts` declares the module, the component
  registry maps its four component keys, and `ReleaseList`'s create action was
  moved into `TableBatchActionBar` to satisfy the governance-bar contract.
- `tsc -b`, `eslint src`, `vite build`, and `check-menu-contract.mjs` all pass.

## Verification evidence

- `go build ./...` ✓ · `go test ./...` ✓ (backend)
- `tsc -b` ✓ · `eslint src` ✓ · `vite build` ✓ (frontend)
- `check-menu-contract.mjs` ✓ (26/34/151/34)

## Explicit gaps

- **No live-cluster verification** — compile + tests only. The recovered files
  have no git history of their own (recovered from a transcript) and no k8s
  runtime was available to exercise them.
- `go test -race` not run (no cgo/gcc).
