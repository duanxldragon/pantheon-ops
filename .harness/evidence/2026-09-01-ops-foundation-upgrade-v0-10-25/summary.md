# Evidence: Consume foundation release pantheon-base-v0.10.25

Task ID: `2026-09-01-ops-foundation-upgrade-v0-10-25`
Date: 2026-09-01
Mode: inheritance-sync (L2), staged-tree verification before human gate

## 1. Release asset verification

- Downloaded from GitHub release `duanxldragon/pantheon-base@pantheon-base-v0.10.25`:
  - `foundation-release-pantheon-base-v0.10.25.tgz` (16,763,771 bytes)
    sha256 `c8b8a3df8701795c1f1b4d7f97a13175182b1c6ed924515e615fb972ff3c50df`
  - `repo.tar` (26,163,200 bytes)
    sha256 `db4173819341e0b4c32c44a3345b5119b744da38ed18e227f1afc4a15b55a0b5`
- `tar -tf repo.tar` full listing: 1823 entries, no EOF/truncation errors
  (v0.10.24 cached snapshot was truncated at 66 entries — regenerated).
- Asset `manifest.json` pins baseCommit `e7bd96466eb006bd4b4fbddd6a1df2c4ec98263c`;
  consumer-impact: "B1-B5 operational workbench changes; business overlays
  must be rebuilt from the published repo snapshot."
- Snapshot spot-check: `backend/cmd/server/main.go`,
  `backend/pkg/common/re_export.go` byte-identical to local base main sync.

## 2. Lock update

`foundation-release.lock.json`: releaseVersion `pantheon-base-v0.10.25`,
baseCommit `e7bd9646...`, checksums as above, lockedAt 2026-09-01T04:35:43Z.

## 3. Overlay breakpoint repairs (ops source tree)

| Drift (base-owned file in ops) | Repair |
|---|---|
| `backend/pkg/common/re_export.go` shim (`ResponseSuccess`/`ResponseError`/`GetUsernameFromContext`) | filedist + observability handlers rewritten to `common.Success` / `common.FailWithCode` / `c.GetString("username")` (113 call sites) |
| `backend/pkg/metrics/business_metrics.go` + 11 metric vars in `prometheus.go` | new overlay package `backend/modules/business/metrics` |
| `backend/pkg/database/timeout.go` `WithCustomTimeout` | new overlay package `backend/modules/business/shared` |
| `backend/internal/middleware/idempotency_middleware.go` + main.go global registration | migrated to `backend/modules/business/opsinit`, registered on the business router group only (uses `database.RDB`) |
| main.go `r.Static("/uploads", "../uploads")` | dropped: no frontend/smoke consumer; filedist serves via DataScopedGroup routes |
| `body_limit_middleware.go` `PANTHEON_MAX_BODY_BYTES` env override | **explicit gap**: engine-level `DefaultMaxBodyBytes` (10 MB) persists; fix belongs in base (recorded, not blocking) |
| `frontend/scripts/check-menu-contract.mjs` observability v2 exemption | dropped (base HEAD converged; observability is canonical) |
| hand-edited `business_overlay_registry.go` (5→9 modules, cross-module wiring) | new composition root `backend/modules/business/opsinit/opsinit.go`; manifest `backendModules` now a single `opsinit.Init` entry |
| `businessOverlay.ts` env-gated observability module | observability canonical (8 modules, no env gate) |
| hand-maintained generated i18n fallbacks (1078 zh-CN keys, 476 not in module JSONs) | extracted to `frontend/src/modules/business/v12locales/{zh-CN,en-US}.json` (+111 legacy keys from resources TS = 1189) |
| untracked V1.2 docs (CHANGELOG, e2e-tests, audits, V1_RELEASE_CHECKLIST, 23 design docs, smoke suites) | added to `business-overlay.json` inventory (58 repositoryOverlayPaths, 53 businessPaths) |
| go.mod extras: websocket/k8s.io/*/sftp | declared in `goDependencies` (6 entries) |

Rebuilder fixes (ops-owned tooling):

- `mergePackages` now preserves ops-only governance scripts
  (`check:business-boundaries`, `test:business-boundaries`, `check:business-foundation`).
- `trackedFiles` decodes git `core.quotepath` octal escapes so non-ASCII
  overlay paths (docs/audits/v1.0/*.md) copy correctly (652→653 files).

Base-first fix applied in `pantheon-base` main (ships with next release):
`docs/harness/tasks/2026-08-11-foundation-release-v0-10-12.task.md` linked
contract pointed at a gitignored local-only `releases/` manifest; removed the
dead link (base frontmatter check still passes, 230 docs).

## 4. Staged-tree gate results (`.tmp/business-overlay-rebuild`)

- `check-business-overlay --root .tmp/business-overlay-rebuild`: OK (53 declared paths, rebuilt tree)
- backend `go mod tidy && go build ./...`: pass
- backend `go test -count=1 ./modules/business/...`: 18 ok / 0 fail
  (argocd, cmdb, cmdb/group, cmdb/host, cmdb/label, deploy, filedist,
  k8s/backup, k8s/cluster, k8s/namespace, k8s/pkg, k8s/release, k8s/workload,
  observability, secrets, service, + others)
- frontend `npm run type-check`: pass
- frontend `npm run build` (prebuild chain: patch-arco-react19, check:menu-contract
  [37 menus / 47 routes / 208 permissions / 49 component keys],
  check:i18n-hardcode, check:i18n-generated-scope,
  check:system-datetime-presentation, check:shell-visual-contract,
  check:ui-contract, check:search-toolbar-contract, check:important-budget,
  check:system-page-admission, check:smoke-web-base,
  check:smoke-coverage-contract): pass, `built in 933ms`
- determinism: two consecutive `npm run rebuild:from-base` runs produce
  byte-identical `.business-overlay-report.json` (624→653 declared files
  after quotepath fix)
- `check:docs-frontmatter`: pass (305 docs staged)
- `check:business-boundaries`: pass
- `check:business-foundation`: pass (8/8 overlay tests)
- `check:encoding` (staged): pass

## 5. Explicit gaps

1. `PANTHEON_MAX_BODY_BYTES` env override dropped — engine-level 10 MB body
   limit applies to large offline-package uploads until base ships the
   override (Base-first; suggested v0.10.26).
2. Base task-doc dead link fixed in base main only; the v0.10.25 asset still
   contains the stale link (cosmetic, patched in staged tree during rebuild
   verification; will self-heal at next base release consumption).
3. Windows CGO race gate + full business smoke deferred to post-replacement
   (require the final working tree; race gate documented in AGENTS.md).
4. External runtime evidence (Prometheus/Loki/Tempo/K8s/Argo CD real
   endpoints) not applicable to this inheritance-sync turn.

## 6. Human gate

Baseline replacement (staged tree → ops working tree, preserving `.git`,
`.foundation`, `node_modules`, local-only files) awaits maintainer
confirmation, per PROJECT_INHERITANCE.md §4.
