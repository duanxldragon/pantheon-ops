# Extension restart guard: source-frozen handoff

Recorded at `2026-11-18T08:07:59.9044352Z`; implementer `/root/extension_startup_integration`. Plan: `../../tasks/2026-11-17-ops-history-audit/continuation-extension-restart-guard-plan-20261118.md`. Machine receipt: `continuation-extension-restart-guard-20261118.json`.

## Changed behavior

`.local/manual-preview-20261116/check-preview-restart-safety.go` now checks six counts in the original read-only Repeatable Read transaction. Original Deploy executing task, running attempt and host lease queries remain identical. Added:

1. `unfinishedFileDistTasks`: all `filedist_task` rows whose status is NULL or outside completed/failed/cancelled, including soft-deleted and future scheduled tasks.
2. `unfinishedFileDistTargets`: the same terminal whitelist against `filedist_target`, independently of parent status/existence.
3. `runningArgoSyncHistories`: every `biz_argocd_sync_history` row with exact application status literal `Running` and `finished_at IS NULL`, without app/server joins.

The local `tcp` / `127.0.0.1:33316` allowlist, ten-second context and all-zero decision are unchanged. Missing tables or any failed query return only a safe query label and exit 1. Output and rejection text now describe unfinished work and explicitly preserve the check-to-restart race limitation.

## Static evidence

- `gofmt -d .local/manual-preview-20261116/check-preview-restart-safety.go`: exit 0, empty output; tool chunk `2b2f08`. This is formatter inspection, not Go build/test execution.
- PowerShell static source inspection: exit 0, 13/13 assertions; tool chunk `664681`. It confirms original Deploy checks, exact new SELECT COUNT predicates, endpoint/timeout/read-only/zero-count/error handling and the point-in-time boundary. Reconstructing the candidate by removing only declared additions and reverting the two receipt/error strings equals the prior source, so no other helper behavior changed.
- Original byte preservation: SHA256 `194512a7c8cf2a2c8f86868da0535cd3d91be4dcaba959f9cf3cd7126114be1a`, unchanged at `continuation-extension-restart-guard-before-20261118.go.raw`.
- Frozen candidate SHA256: `5e951909403bb0090026891ad2118fcc3cc9636bba8a98683bd8268ffd661700`.
- Source references and their fresh SHA256 are included in the JSON receipt. Terminal statuses/table names are from FileDist's model; Argo's Running literal and unfinished-history condition are from observation/reconciler source, rather than older model comments.

## Root-owned remaining execution

No Go build/test/helper execution, real MySQL query, service restart, frozen backend/locale/design change or database mutation occurred in this lane. Root should verify the frozen candidate hash and obtain the real guard receipt through the existing `restart-ops.ps1` only after its full backend race run completes. This handoff is static verification; it does not claim runtime acceptance or work-package completion.

A zero count is a point-in-time snapshot and does not prevent new requests, scheduler work or Start races between check and restart. Soft-deleted unfinished rows deliberately refuse restart until separately reconciled. No missing-table exemption is permitted. Process-level graceful join, production FileDist authorization and remote rollback/fencing remain outside this guard's evidence.
