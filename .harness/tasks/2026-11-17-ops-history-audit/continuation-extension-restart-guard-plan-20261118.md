# Local preview extension restart guard

Implementer: `/root/extension_startup_integration`. Parent: `2026-11-17-ops-history-audit`; business lifecycle continuation references `2026-11-16-ops-filedist-product-closure` and `2026-11-16-ops-argocd-ownership-convergence`. L1 local operational helper under the existing L2 parent scope.

In: change only `.local/manual-preview-20261116/check-preview-restart-safety.go`, preserve its exact prior bytes, and record static review/evidence. Reuse the existing read-only Repeatable Read transaction, overall ten-second context, local `tcp` endpoint allowlist `127.0.0.1:33316`, sanitized errors, and all-zero restart decision. Preserve all three existing Deploy checks unchanged; add three independent SELECT COUNT checks.

Out: frozen backend, locales and designs; Go execution, service restart, schema/data mutation, credentials, production actions and work-package completion decisions. Root owns actual guard execution through `restart-ops.ps1` after the full backend race run completes. No new dependencies or abstractions.

## Queries and source basis

- `filedist_task` and `filedist_target`: `status IS NULL OR status NOT IN ('completed','failed','cancelled')`. Model table names and terminal constants come from `backend/modules/business/filedist/distribution_model.go:18` and `:50` / `:79`. This includes current pending/running/transferring/retrying, historical paused/distributing, and unknown unfinished states. Independent target counts also catch unfinished rows under terminal or missing parents. Include soft-deleted rows conservatively: deletion alone does not prove prior execution has stopped. They can therefore block the preview until separately reconciled; this helper does not mutate them.
- Include future scheduled pending tasks: `backend/modules/business/filedist/distribution_lifecycle.go:230` schedules due pending work, so a restarted scheduler can later activate them. Do not filter by due date. Startup recovery's narrower running/cancelled parent and pending/transferring/retrying target selection (`:132`, `:153`, `:16`) is insufficient as a restart-safety whitelist.
- `biz_argocd_sync_history`: `status = 'Running' AND finished_at IS NULL`. Table name and nullable finish field come from `backend/modules/business/cicd/argocd/argocd_model.go:93` and `:105`; exact status comes from `argocd_observation.go:10`, and unfinished history detection from `argocd_reconciler.go:232`. Count histories independently of application/server joins; missing/deleted parents do not silently exempt unfinished history.
- A missing or unreadable new table follows the same sanitized query-error/exit-1 refusal as the existing checks; there is no schema exemption.

## Verification and handoff

Static verification will compare the three original checks with the preserved bytes, assert exactly six read-only count queries and the unchanged endpoint/transaction/timeout/zero-count policy, and check formatting without running Go. Record old/new SHA256 and the real runtime status (`not run in this lane`). Root must obtain an execution receipt before using this guard for restart.

This check is a read-only point-in-time snapshot. It does not prevent new requests, scheduler work or Start races between the check and restart, prove process-level graceful joins, or establish remote rollback/fencing. An all-zero snapshot must retain these limitations.

Original bytes: `.harness/evidence/2026-11-17-ops-history-audit/continuation-extension-restart-guard-before-20261118.go.raw`, SHA256 `194512a7c8cf2a2c8f86868da0535cd3d91be4dcaba959f9cf3cd7126114be1a`; preserve without overwrite. Review/handoff artifacts use the same evidence directory with prefix `continuation-extension-restart-guard-20261118`.
