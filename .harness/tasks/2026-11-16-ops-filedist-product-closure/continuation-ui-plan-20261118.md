---
task_id: 2026-11-16-ops-filedist-product-closure
type: continuation-plan
author: claude
created_at: 2026-11-18
status: implemented-ui-only
---

# E01 UI Continuation Plan (2026-11-18)

## Contracts already read

- `AGENTS.md`, `CLAUDE.md`, `docs/PROJECT_INHERITANCE.md`
- `docs/harness/tasks/2026-11-16-ops-filedist-product-closure.task.md`
- `docs/designs/BUSINESS_FUNCTIONAL_DESIGN.md`, `docs/designs/BUSINESS_QA_ACCEPTANCE_PLAN.md`
- `../pantheon-base/DESIGN.md` and `C:/Users/xiaolong/.codex/skills/impeccable/SKILL.md` (visual checklist applied, no render run)
- Backend DTO/model/handler/source: `backend/modules/business/filedist/{distribution_model.go,filedist_handler.go,target_selection.go,module.go,opsinit.go}`
- CMDB selection contract: `backend/modules/business/cmdb/target_selection.go`
- Frontend target selector reuse: `frontend/src/modules/business/shared/targetSelector.ts`, `frontend/src/modules/business/deploy/task/DeployTargetSelector.tsx`, `frontend/src/modules/business/deploy/api.ts`
- Pagination/modal toolbar patterns: `frontend/src/components/{patterns/SearchToolbar.tsx,patterns/modals/AppModal.tsx,patterns/actions/SubmitBar.tsx,data-display/AppTable.tsx,table/standardPagination.ts}`

## Owner + boundary

- Claude owns edits ONLY under `frontend/src/modules/business/filedist/` plus new
  `.harness/tasks/2026-11-16-ops-filedist-product-closure/continuation-ui-plan-20261118.md` and
  `.harness/evidence/2026-11-16-ops-filedist-product-closure/claude-ui-handoff-20261118.md` / `.json`.
- Backend, other UI modules, shared locale resources, tests, manifest/status, goals, Base, package
  deps, README, and existing evidence are NOT touched by Claude. Codex integrates locale keys/tests
  and runs commands.

## Confirmed contract facts

1. `POST /business/filedist/tasks` accepts `DistributionTaskDTO`:
   `{ name, fileId, targets[TargetConfig], scheduledAt?, selection: cmdb.TargetSelection?, scopeVersion }`.
   `TargetConfig` binding requires `host`, `targetPath`, `username`, `hostFingerprint`; `hostId` +
   `stateVersion` must match the provider resolution; `host`/`port` are provider-canonicalized on
   create when consistent.
2. Authorized create only runs when `selection != nil`: `CreateAuthorizedTask` requires a wired
   `FileDistributionAuthorizer`, a wired `TargetSelectionCapability`, `scopeVersion == ""` rejected,
   `unavailable/ineligible == 0`, and `len(resolved.Hosts) == len(targets)`. Unauthorized/conflict
   errors surface as `permission.denied` / `business.cmdb.selection.conflict`, but the handler maps
   all create errors to HTTP 500 `create_failed`. UI must therefore not over-claim.
3. **Runtime prerequisite (exact)**: `backend/modules/business/opsinit/opsinit.go` wires
   `fileDistModule.SetTargetSelectionCapability(cmdbCapability)` but never calls
   `fileDistModule.SetFileDistributionAuthorizer(...)` → `s.authorizer == nil` → every authorized
   create fails fail-closed (`permission.denied` → 500 `create_failed`). The backend lifecycle
   executor must wire a `FileDistributionAuthorizer` before authorized creation can succeed. The UI
   keeps an honest error state and reports this prerequisite.
4. Route gate `requireFileDistUnrestrictedScope` is all/admin-only; the frontend must not derive
   authorization from menu — create button visibility is UX gating via `business:filedist:create`
   only; the backend remains the enforcement point.

## UI decisions (calm, compact operational workbench)

- **List page** (`FiledistList.tsx`): keep Tabs + upload + detail/delete behavior; add real server
  pagination for files and tasks tabs via `buildStandardPagination`; add one `SearchToolbar`
  (keyword shared, status inline filter only on tasks tab) with controlled keyword + `onClearAll`;
  summarize totals in the `GovernanceSummaryBar` metrics.
- **Create flow**: `AppModal size="xl"` with a vertical form. Choose an uploaded file (own paged
  load on open, independent of list pagination), pick business scope, reuse the existing
  `DeployTargetSelector` (same `useBusinessTargetSelector`/`resolveDeployTargets` contract) → the
  resolved `DeployTargetResolution` provides `selection + scopeVersion + hostIds/stateVersions`.
  Shared SSH username / target path / host fingerprint / max retries are collected once and applied
  to every resolved host (`TargetConfig` requires binding). No passwords/private keys are collected.
  Optional `scheduledAt` via DatePicker for future triggering (backend lifecycle executor owns
  scheduler semantics).
- **Submit guards**: require fresh resolution for the selected scope, `hosts.length > 0`,
  `unavailable == 0`, `ineligible == 0`. `createTask` uses `skipErrorMessage: true` and reports
  inline (forbidden vs failure) so a fail-closed backend state is explicit.
- **Detail page** (`FiledistDetail.tsx`): keep one-shot load but add a bounded 5s silent refresh poll
  while the task is `pending`/`running` (unmount + abort cleanup), per-action submitting state for
  start/cancel/retry, i18n for previously hardcoded button labels, status colors, AppTable instead
  of raw Table.
- The `DeployTargetSelector` is reused as-is (no generic selector invented); its `business.deploy.*`
  keys resolve from the bundled business resource fallback used by the filedist module.

## Acceptance linkage

- In-scope pieces delivered by this UI change: browser create entry, authorized target selection,
  server pagination, bounded progress refresh. Not delivered here (owner = backend lifecycle
  executor): runtime authorizer wiring, real scheduler/cancellation integration, E04 live transfer
  acceptance, rendered evidence.
- New i18n keys are listed in the separate locale delta inside
  `claude-ui-handoff-20261118.json`; Codex integrates them into shared resources.
