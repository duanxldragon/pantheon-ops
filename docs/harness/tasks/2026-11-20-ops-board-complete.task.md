---
title: 完成全部规划任务：归档收口 2026-11-20
doc_type: Remediation
layer: business
status: Active
linked_contracts:
  - docs/PROJECT_INHERITANCE.md
updated_at: 2026-11-20
---

# Task Packet: 2026-11-20-ops-board-complete

## Goal

把"已规划但未完成"的任务全部驱至终态：完成实施的卡以 completed 入档（保留显式 human-gate residuals），过期/被替代的卡以 superseded 直接归档。

## Implementation Notes

- Hard Stop：仅写 `.harness/` 与 `docs/harness/tasks/`（+ docs/README.md 链接修复）；不实现业务代码。
- 继承 close-history.mjs 约定（historyCloseout 块 / 文档 Archived / dated history-closeout.md）。
- 归档根 `.harness/archive/2026-11/{tasks,evidence}/`，canonical 引用 archive-aware。

## Verification Plan

- `close-board.mjs` dry-run → `--apply`（幂等 resume + journal + verify + resolveArchiveAwarePath）
- `close-auto-merge.mjs --apply`
- `npm run check:docs-frontmatter` / `check:encoding` / `check:structure` / `check:task-packet-template`
- `npm run test:scripts`
- `npm run check:pr-governance`（PR body）

## Linkage

- Evidence: `.harness/evidence/2026-11-20-ops-board-complete/`
- Review: 同目录 `review.md`（self-judgement；独立 codes review 为 human gate）
- Plan: 同任务目录 `plan.md`

## Evidence Required

见 manifest.evidenceRequired（dispositions / journal / result / commands+summary+review / ssrf 复验）。

## Human Gates

见 manifest.humanGates（十包最终验收、真实 provider、生产/发布、O7、独立 review）。

## Completion Checklist

manifest.completionChecklist 六项；PR governance 绿后由维护者验收收尾。
