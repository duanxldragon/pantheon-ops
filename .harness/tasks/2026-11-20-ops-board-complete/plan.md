# Plan — 完成全部规划任务：归档收口 2026-11-20

## Goal

驱动每一条计划未完成任务到终态。全部 26 条 + auto-merge 补卡共 27 条入档；`history-audit` 原位置 completed。

## 方法与约定

- 继承 `2026-11-17-ops-history-audit` 的 close-history 约定：`historyCloseout` manifest 块 + 文档 `status: Archived` + `## Result — History Closeout` + dated `history-closeout.md` 证据。
- 程序：`close-board.mjs`（数据驱动 `dispositions.json`），dry-run 前检 + `--apply` 幂等 resume + journal + verify。
- 归档根：`.harness/archive/2026-11/{tasks,evidence}/<id>/`；工具链 `resolveArchiveAwarePath` 已 archive-aware。

## Steps

1. `dispositions.json`（evidence 权威副本）：26 任务 pending 列表 + title/status/reason/residuals。
2. `close-board.mjs` dry-run → 26 candidates。
3. `close-board.mjs --apply`：移动 + flip + doc + dated evidence，journal 全 155+ 条，resume 容忍中断。
4. `close-auto-merge.mjs`：补 auto-merge-reconcile 卡（task-manifest 布局）。
5. `history-audit` 原位置 completed（其 evidence 为 archive 索引 nexus，不搬迁）。
6. `ARCHIVE.md` 增补 2026-11-20 段；`STATUS.md` 新 checkpoint；`board-complete` manifest 置 completed。
7. evidence for board-complete：commands.json + summary.md + review.md + archive-dry-run/result。
8. governed PR（Evidence → commands.json），squash 合并。

## In / Out

见 manifest `in`/`out`。核心边界：不代签维护者终验与生产门禁；residual 全部显式保留在 dispositions.json + ARCHIVE.md。
