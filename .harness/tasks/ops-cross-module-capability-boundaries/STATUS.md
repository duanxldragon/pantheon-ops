# Status: Capability Boundaries

- Status: `complete`
- Updated: `2026-08-17T13:43:59.4905879+08:00`
- Executor: `Codex GPT-5`
- Baseline commit: `ee213508a32ebdf27685128e0c719e9036272de5`
- Workdir: `pantheon-ops/backend`

## Completed

- Task packet records the maintainer's mandatory no-cross-table rule.
- Start preflight recorded for the dirty worktree; no code changes made before this status update.
- Initial forbidden-access scan recorded the runtime cross-module access in:
  - `backend/modules/business/cmdb/host/host_service.go`: imports `bizscope` and returns `*bizscope.BizScope`.
  - `backend/modules/business/cmdb/host/host_import_export.go`: direct `biz_business_scope` table lookup.
  - `backend/modules/business/bizscope/bizscope_service.go`: direct `biz_cmdb_host` joins/table updates.
  - `backend/modules/business/deploy/deploy_service.go`: direct `biz_business_scope`, direct `biz_cmdb_host` join/update.
  - `backend/modules/business/k8s/cluster/cluster_service.go`: imports `bizscope` and returns `*bizscope.BizScope`.
- Revalidated and removed those runtime paths; the remaining provider-owned CMDB
  table access is behind `cmdb/capability.go` and the checker allowlist.
- Added `CMDBHostReader.GetByIDs` for batch visibility checks and completed
  overlay injection for BizScope, CMDB, Deploy, and K8s.
- Added and passed `scripts/check-business-module-boundaries.mjs`.
- Dependency graph baseline from `go list -f '{{.ImportPath}} => {{join .Imports ","}}' ./modules/business/...`:
  - `business/cmdb/host` currently imports `business/bizscope`.
  - `business/deploy` imports provider package `business/cmdb`.
  - `business/k8s/cluster` currently imports `business/bizscope`.
  - `business/bizscope` does not import CMDB package but directly accesses `biz_cmdb_host` by table name.

## Decisions

- Same-process typed Go APIs are the default; do not use loopback HTTP.
- Provider DTOs must not expose GORM models or `*gorm.DB`.
- Use provider-owned capability contracts where this avoids cycles:
  - BizScope owns `BizScopeReader` and stable `BizScopeRef` DTOs.
  - CMDB owns host reader/ownership command contracts and stable host DTOs.
  - Consumers depend only on provider contracts, not provider GORM models.
- Current dirty-file collisions to preserve:
  - Modified: `backend/modules/business/bizscope/module.go`, `backend/modules/business/cmdb/module.go`, `backend/modules/business/business_overlay_registry.go`.
  - Untracked and task-relevant: `backend/modules/business/cmdb/host/host_import_export.go`, `backend/modules/business/k8s/cluster/cluster_service.go`, `backend/modules/business/k8s/k8s.go`.
  - No reset/restore/revert is authorized; edits must preserve existing content.

## Evidence

Created `.harness/evidence/ops-cross-module-capability-boundaries/commands.json`,
`scan.txt`, `dependency-diagram.md`, `summary.md`, and `review.md`.

## Next Atomic Action

Hand off to the now-unblocked `ops-bizscope-datascope-ownership`,
`ops-deploy-immutability-idempotency`, or `ops-k8s-release-reliability` child;
each owner must record a fresh dirty-worktree collision preflight.

## Blockers

None. Provider DataScope enforcement and Deploy/K8s lifecycle reliability are
explicitly delegated to downstream child tasks.
