# Plan — Business Navigation Repair (2026-11-16)

Task: `.harness/tasks/2026-11-16-ops-navigation-repair/manifest.json`
Goal: repair the entire business navigation of pantheon-ops (duplicate 404 service menu,
in-page ServiceCatalog tabs that don't respond, ordering of the business block, ArgoCD
label collision). Sole executor: Claude. Codex dispatches + independently accepts.

## Root causes (confirmed via probe.json / before-all/all-business-clicks.json)

1. **Duplicate service directory** — upgraded legacy tree carries `id159 /business/service`
   (top-level C leaf, `pagePerm business:service:list`, no registered page route → 404)
   alongside current `id1818 /business/service-catalog` (+children 1819/1820/1821). The
   service seed's `findSeedMenuID` only matches by `/business/service-catalog`, so id159 is
   never merged and recurs on every start.
2. **Service in-page tab click no response** — `ServiceCatalog.tsx` uses
   `<Tabs defaultActiveTab={activeTab}>` (uncontrolled) and never calls `navigate()` on
   change; a mounted page only reads the initial URL once.
3. **Ordering** — parent sorts today: CMDB 1, BizScope 2, Service 2, Deploy 3, K8s 4,
   Observability 5, Filedist 50, ArgoCD 60. System domain starts at 20, so filedist and
   ArgoCD are pushed below system governance.
4. **ArgoCD label collision** — `operations.cicd.argocd.menu` renders "应用发布", identical
   to the K8s release child, so the sidebar shows two "应用发布" → strict-mode Playwright
   click failure on id134.

## Target business block (stable, continuous, before system settings)

| order | sort | menu                    | change |
| ----- | ---- | ----------------------- | ------ |
| 1     | 1    | CMDB (host/group/label) | —      |
| 2     | 2    | BizScope                | —      |
| 3     | 3    | Service                 | 2 → 3  |
| 4     | 4    | Deploy                  | 3 → 4  |
| 5     | 5    | Filedist                | 50 → 5 |
| 6     | 6    | K8s                     | 4 → 6  |
| 7     | 7    | ArgoCD                  | 60 → 7 |
| 8     | 8    | Observability           | 5 → 8  |

## Changes

### A. service seed — idempotent legacy merge (`backend/modules/business/service/seed.go`)

Add `mergeLegacyServiceMenu(tx)` called at the top of `seedMenus` before the seed loop.
Converge on the LEGACY id to preserve old id + role-menu bindings:

- (A) if canonical (`/business/service-catalog`, id1818) AND legacy (`/business/service`,
  id159) both exist and differ:
  - reparent canonical children onto legacy (`UPDATE system_menu SET parent_id=legacy
WHERE parent_id=canonical AND module='business.service'`)
  - absorb canonical role_menu bindings into legacy (`INSERT INTO system_role_menu
(tenant_id, role_id, menu_id) SELECT tenant_id, role_id, legacy FROM
system_role_menu WHERE menu_id=canonical AND NOT EXISTS (
  SELECT 1 FROM system_role_menu s WHERE s.tenant_id=system_role_menu.tenant_id
  AND s.role_id=system_role_menu.role_id AND s.menu_id=legacy)`)
  - delete canonical role_menu rows, then the canonical menu row
  - set legacy path → `/business/service-catalog`
- (B) legacy exists, no canonical → update legacy path in place (keeps id, becomes canonical)
- (C) otherwise no-op.
  All rows touched are gated `module='business.service'`. Never deletes a same-named custom
  menu. Idempotent over consecutive seeds/starts. Change parent seed `Sort: 2 → 3`.

### B. ServiceCatalog controlled tabs (`frontend/src/modules/business/service/ServiceCatalog.tsx`)

- import `useNavigate`
- `<Tabs activeTab={activeTab} onChange={(key) => void navigate(`/business/service-catalog/${key}`)}>`

### C. Reorder parent sorts (source-of-truth seed)

- `service/seed.go` parent `Sort: 2 → 3`
- `deploy/deploy_seed.go` parent `Sort: 3 → 4`
- `filedist/seed.go` `Sort: 50 → 5`
- `k8s/k8s_seed.go` parent `Sort: 4 → 6`
- `cicd/seed.go` parent `Sort: 60 → 7`
- `observability/seed.go` parent `Sort: 5 → 8`

### D. ArgoCD distinct label (`backend/modules/business/cicd/seed.go` + `frontend/src/i18n/resources/business/zh-CN.ts`)

Rename zh-CN `operations.cicd.argocd.menu` (and the hero eyebrow/title) from "应用发布" to
"ArgoCD 应用交付" so the sidebar no longer shows two "应用发布" and the GitOps entry is
unambiguously named. Keep en "ArgoCD App Delivery". The header/breadcrumb share the same
resource so the whole ArgoCD surface reads as ArgoCD/GitOps, not a K8s release action.

### E. Workload page bare-kebab keys (`frontend/src/i18n/resources/business/zh-CN.ts` + `en-US.ts`)

The K8s workload page renders raw `business.k8s.workload.*` keys when a key is missing.
Mirror the full workload key set already used by the WorkloadList component into the business
i18n resources so the navigation matrix (and any rendered page) shows translated text, not a
bare key.

### F. frontend module registration order (`frontend/src/modules/business/index.ts` + `business-overlay.json`)

Index registration order and the overlay frontendModules declaration must both match the
backend seed Sort, keeping the business block contiguous:
Cmdb, BizScope(业务域), Service(服务目录), Deploy, Filedist, K8s, ArgoCD(CICD), Observability.
Do NOT extend the shared `ModuleMenuMeta` sort field; the seed remains the sort source of
truth (see blocker 3 note).

## Regression tests

- `backend/modules/business/service/seed_migrate_test.go` — upgraded-old-menu fixture;
  asserts binding preservation (incl. tenant_id), duplicate convergence, idempotency over
  two consecutive seeds, and the `module='business.service'` ownership boundary.
- `frontend/src/modules/business/service/ServiceCatalog.nav.spec.ts` — deep-link tab render,
  in-page tab click syncs URL + content, refresh/forward/back consistency, all business
  leaves render, no duplicate 应用发布.

## Verification

- `go test -race ./backend/modules/business/service/...` (CGO_ENABLED=1, mingw64 in PATH)
- frontend `typecheck && lint && build`
- menu-contract + business-overlay/boundary checks + `check:docs-governance`
- staged overlay rebuild via `rebuild-from-base.mjs` (must not overwrite working tree)
- Codex: restart backend with fresh seed on real upgraded DB, Playwright regression,
  desktop 1440x900 + mobile 390x844 screenshots.
- Record NAVIGATION_MATRIX.md, summary.md, commands.json.

## Boundaries

- Do NOT touch `backend/modules/business/cmdb/host/**`.
- No lock upgrade, no commit/push/PR/release.
- No wholesale menu deletion; local-only SQL rewriting of existing DB business data is OUT.
- RBAC permission tree unchanged (pagePerm count per C-node stays 1).
- Nothing outside business menu seed/registration, business frontend menu/route/page/i18n,
  business nav smoke/regression, business-overlay declarations + generated registry.
