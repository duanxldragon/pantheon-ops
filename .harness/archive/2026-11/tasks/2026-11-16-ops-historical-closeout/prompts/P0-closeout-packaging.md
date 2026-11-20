# Prompt P0 — 历史收尾物料补齐（closeout packaging）

> 派发模型：按 DISPATCH 契约交由执行 Claude 通过 `codex exec ... -C pantheon-ops` 实施；本仓库 Claude 为 planner/reviewer。
> 本提示词只允许写 `.harness/` 与 `docs/harness/tasks/` 下**明确列示**的文件；不触碰业务源码、不改 lock、不发布、不伪造历史命令、不覆盖他人改动。

## 范围（In / Out）

- In：把 4 处历史收尾的"缺物料/悬空引用/状态滞后"补齐，使 governance 检查一致。
- Out：任何业务实现、任何 `foundation-release.lock.json`/`VERSION`/业务模块改动。

## 精确文件所有权

1. `2026-11-13-base-v0187-consumption` live manifest `.harness/tasks/2026-11-13-base-v0187-consumption/manifest.json`：其 `status` 字段仍为 `in-progress`，而 packet/archive/evidence 已记录 PR #183 合并 + required CI 通过。仅将其 `status` 归一为 `completed`，在 `implementationNotes`/`closeout` 补一条指向 archive 的证据链接。不改任何代码、不确定状态为"重开"。
2. `.harness/evidence/2026-11-07-ops-planned-task-closeout/`：补缺失的 `summary.md`、`commands.json`、`review.md`（其验证日志已全部绿色，`review-final.log` 的 8 条 FAIL 是陈旧 legacy archive allowlist 条目，非本内容失败）。逐项据现有日志如实写，不得虚构运行。若 `review-final.log` 挂靠的 8 条陈旧 allowlist 属于停工历史扫描，在本批把 allowlist 里已归档、其 artifact 已不存在的条目标注为 superseded（在允许的治理清单文件内）。
3. 若 `docs/harness/tasks/2026-11-07-ops-planned-task-closeout.task.md` 与 `2026-11-14-ops-latest-foundation-and-plans.task.md` 缺失，按模板补最小父卡并 link 到对应证据；**不得**把缺失卡当作"工作已完成"或纳入 D5 缺陷计数之外的新缺陷。
4. `.harness/evidence/2026-11-14-ops-latest-foundation-and-plans/`：补缺失的 `summary.md`/`review.md`/`commands.json`，或在其现有文本中显式声明所引用 artifacts 悬空、归属为 superseded（其验证已被后续 base-v0-18-x 消费目录取代），并 link 到 `2026-11-13-base-v0187-consumption`/`2026-11-14` 相关证据。
5. 对陈旧 review-allowlist：仅限治理清单文件（若由 `check-*` 脚本读取的 allowlist）内将该条目判为 superseded，不改 allowlist 逻辑、不删除其他在途条目。

## 约束

- Base-first：仅收尾/物质，不改平台/系统域语义。
- 不触碰业务源码：本批所有权仅 `.harness/` 与上述两缺失 task 卡。
- 不提交、不 push、不发布、不改 `foundation-release.lock.json`。产出仅本地工作树 + evidence。

## 真实验收（Codex 会核）

- `npm run check:docs-governance`（CI 同款）通过；
- `.harness/tasks/*/manifest.json` 状态与各自 evidence 一致（无 `in-progress`-但我们有 merged 证据的记录）；
- 无悬空引用：本批列出的缺失物料均已就位或显式 link 声明。
- 未覆盖任何他人未提交改动、未回滚历史。
