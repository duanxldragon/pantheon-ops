# Review — cmdb-host-import-export

Findings-first self-review of the import/export batch (host + group + label +
bizscope) and the k8s module recovery.

## Findings and resolutions

1. **#1 (blocker) group handler compiled against the wrong signature —
   RESOLVED.** `Export` called a non-existent handler/query shape and
   `GroupListQuery` was missing the import/export route plumbing. Rewired to the
   real service contract; package compiles.

2. **#2 (blocker) label import/export referenced non-existent model fields —
   RESOLVED.** The implementation used `valueType`/`defaultValue`/`sort`, but
   `LabelSchema` models `category`/`valueMode`/`dictCode`/`status`. Rewrote
   `label_import_export.go` against the real model (per
   `docs/import-export-cmdb-label-spec.md`) and rewrote the three handlers to the
   host-style flow.

3. **#3 (blocker) seeds were host-only — RESOLVED.** `cmdb/module.go` and
   `bizscope/module.go` carried routes but no permission keys, menu F-seeds, or
   runtime i18n for group/label/bizscope import/export. Added all three; the
   frontend buttons now gate on real permissions and the menu contract passes.

4. **#4 (risk) silent parent fallback on group import — RESOLVED.** Unknown
   `parentName` previously attached rows to the root. Import now fails with a
   row-level error, matching the spec's fail-fast intent.

5. **#5 (risk) frontend list components used wrong shared-component props —
   RESOLVED.** Three lists passed props that don't exist on `ListHeaderActions`
   / `ImportCsvButton` (type-check failures). Rewrote them to the host reference
   (`utility` prop + `ImportCsvButton` + permission-gated buttons).

6. **#6 (blocker) k8s module was an orphan that broke the build — RESOLVED.**
   The recovered files compiled against `k8s.io/client-go` v0.36.3 /
   `gorilla/websocket` v1.5.3 (present in the module cache), so those deps were
   pinned and `go mod tidy`'d. The module is registered in
   `business_overlay_registry.go`; the four frontend pages are fixed (import
   depth, `MenuIconKey`, icons) and mounted in `businessOverlay.ts` + the
   component registry. `go build ./...` is green again.

7. **#7 (risk) menu-contract gaps — RESOLVED.** The contract check surfaced
   missing frontend i18n keys (added to `zh-CN.ts`/`en-US.ts`), the missing
   bizscope component whitelist entry, and the ReleaseList governance-bar
   violation (create action moved to `TableBatchActionBar`).

## Verification evidence

- Backend `go build -buildvcs=false ./...` exit 0; `go test -buildvcs=false ./...`
  full suite green (cmdb, cmdb/group, cmdb/host, cmdb/label, bizscope all ok).
- Frontend `tsc -b`, `eslint src`, `vite build` all exit 0.
- `check-business-overlay.mjs` OK (16 declared paths);
  `check-menu-contract.mjs` ✓ (26/34/151/34); full prebuild contract chain ✓.
- New tests: `group_import_export_test.go`, `label_import_export_test.go`,
  `bizscope_import_export_test.go` — all pass alongside the existing
  `host_import_export_test.go`.

## Residual risk

- **Runtime smoke not executed** — MySQL root credentials unavailable in this
  environment (`PANTHEON_SMOKE_MYSQL_*` unset, no `.env`), so the DB-backed
  `test:smoke:business:cmdb` and visual desktop/mobile screenshots for the new
  buttons remain an explicit gap. Contract + unit coverage stands in for it.
- `go test -race` not run (no cgo/gcc on this machine).
- The k8s module files were recovered from a Claude Code transcript
  (`k8s-recovery/recover.py`); they have no git history of their own and were
  reviewed only via compile + tests, not a live cluster.

## Machine Readable

```json
{
  "taskId": "cmdb-host-import-export",
  "status": "reviewed",
  "findings": [
    {"id": "#1", "severity": "blocker", "title": "group handler compiled against wrong signature", "state": "resolved"},
    {"id": "#2", "severity": "blocker", "title": "label import/export referenced non-existent model fields", "state": "resolved"},
    {"id": "#3", "severity": "blocker", "title": "permission/i18n/menu seeds were host-only", "state": "resolved"},
    {"id": "#4", "severity": "risk", "title": "silent parent fallback on group import", "state": "resolved"},
    {"id": "#5", "severity": "risk", "title": "frontend lists used wrong shared-component props", "state": "resolved"},
    {"id": "#6", "severity": "blocker", "title": "k8s module orphaned, broke go build", "state": "resolved"},
    {"id": "#7", "severity": "risk", "title": "menu-contract gaps (i18n keys, whitelist, governance bar)", "state": "resolved"}
  ],
  "residualRisk": [
    "DB-backed runtime smoke not run (MySQL credentials unavailable)",
    "go test -race unavailable (no cgo)",
    "k8s files recovered from transcript, not live-cluster tested"
  ]
}
```
