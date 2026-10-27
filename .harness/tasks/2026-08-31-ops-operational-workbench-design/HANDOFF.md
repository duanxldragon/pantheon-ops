# Stateless Handoff

## Outcome

本包把 Ops 运维体验拆为 O1-O7。当前只完成设计 readiness；任何工作台、外部集成和视觉效果都没有因本文自动完成。

## Start Protocol

1. 阅读 `AGENTS.md`、`docs/PROJECT_INHERITANCE.md`、Base `DESIGN.md` 和父 task packet。
2. 阅读 `docs/designs/BUSINESS_OPERATIONAL_WORKBENCH_DESIGN.md`。
3. 检查 `git status --short` 并保留全部现有改动。
4. 从 `EXECUTION_QUEUE.md` 选择依赖已满足的一个 packet。
5. 确认所需 Base packet 已进入当前 immutable foundation lock；否则保持 blocked。
6. 用 CodeGraph 确认业务流、权限/DataScope/审计和外部边界，再记录 owner、基线与首个动作。

## Global Rules

- Ops 只改 `business/*` 和 business overlay；generic component drift 回 Base。
- 不从 Base/main 手抄共享文件，不覆盖 `platform` 或 `system/*`。
- UI 动作以后端状态机和权限为准；所有查询、日志和导出执行 DataScope。
- 外部系统不可用时记录 gap，不用 mock 结果宣称真机验收。
- UI 完成需要实际 desktop/mobile、状态、键盘和明暗主题证据。

## Evidence Minimum

- `commands.json`、`summary.md`、findings-first `review.md`。
- backend/frontend targeted tests、business overlay/permission/i18n checks。
- desktop/mobile x 关键状态 x light/dark。
- 可用外部环境的运行证据，或 endpoint/credential/production gate 的明确 gap。

## Stop Conditions

在 Base override、新 schema/public API、权限模型、生产外部动作、consumer lock 和最终视觉验收前停止并记录决策。

## Completion Boundary

子卡实现、Base sync、业务 smoke、外部 gap 和维护者视觉/功能 gate 缺一不可。设计文档与旧截图不能替代这些证据。
