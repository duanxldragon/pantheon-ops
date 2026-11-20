# 历史任务收尾：Claude Intake 对账摘要

日期：2026-11-16
父任务：`2026-11-16-ops-historical-closeout`
输入基线：`.harness/tasks/2026-11-16-ops-pilot-delivery/OPEN_TASKS.md`
角色：Claude = Planner/intake；Codex = 派发与验收。本文件**只做对账，不实现**。

## 读取与检查依据

- 治理：`CLAUDE.md`、`AGENTS.md`、`docs/PROJECT_INHERITANCE.md`、`../pantheon-base/DESIGN.md`、`../pantheon-base/AGENTS.md`、`.agents/adapters/claude-code.md`、`.agents/skills/README.md`、`DISPATCH.md`。
- 基建基线：`foundation-release.lock.json`（锁定 `pantheon-base-v0.18.9`，只读消费，未改 lock）、`business-overlay.json`（`business/` 目录结构、`backend/pkg/security/ssrf` 为 base-shared 路径，非 business overlay）、`.harness/STATUS.md`、`.harness/ARCHIVE.md`（12 个已归档索引）。
- 逐项原文与证据：7 个子代理并行读取全部历史卡/manifest/evidence/summary/review/commands，并按当前源码核验 `codePresent`。已归档 12 包逐项核验 `archive/2026-*/{tasks,evidence}` manifest+summary 引用完整。
- 关键复核（人工二次校验）：`cmdb/ssh` 配置 key 全仓 grep 为零命中（`host_service.go` 仍硬编码 22/10s）；`pr-automation.yml` 自动合并用 `github.token` 且无 `RELEASE_GATE_TOKEN`/`fail-closed`/`ci.yml`；`backend/pkg/security/ssrf` 归属 base-shared（grep 到 `pantheon-base/backend/pkg/security/ssrf` import）。

## 结论（仅对账口径，未实施）

- 当前已消费 v0.18.9 后，历史 `v0.10.x–v0.18.x` 升级/消费记录绝大多数为 **obsolete-with-proof**（被后续正式 release 替代）或 **upstream-owned**（属于 pantheon-base），不足为当前业务缺陷。
- 现存**真正缺实现**记录（其中 `cmdb-ssh-config-externalize` 为最干净、低复杂度、Base-first 明确的独立实现）：
  - `2026-07-17-cmdb-ssh-config-externalize`（SSH 采集 timeout/default_port 配置外部化，源码确认缺失）
  - `2026-08-12-pr-auto-merge-push-trigger`（`RELEASE_GATE_TOKEN` fail-closed + `ci.yml` 触发器缺失；与 `pr-auto-merge-token` 存在互相矛盾的树内状态，需先定夺）
  - 工作台 `O1`（统一 BizScope 上下文/目标选择器，与 C02/C03 部分重叠）
  - 工作台 `O3`（仅 Deploy 预览存在，K8s/ArgoCD 版本树 diff 预览缺失）
  - `O7` 明确**延期**（关系成熟度门禁未过，非本次试点退出项）
- 现存**缺验收证据**：`D1`（六类外部集成真实验收缺失）、`D2`（SSRF 代码已在 base-shared，缺 allowlist/private-network/redirect/DNS 负向验收与部署策略）、`D5`（治理债务基线 567/6/50/149 需重跑）、`pr-auto-merge-token`、`dependabot-pr-governance`、`closure-permission-scope`、`2026-11-14-ops-latest-foundation-and-plans`（悬空引用）。
- **生产/外部前置**：`D4-G2`（RPO/RTO + 恢复演练，§8.1 仍 OPEN）、`D4-G3`（生产量级 sizing，OPEN）、`D1` 需端点/凭证、`SONAR_TOKEN` 轮换、生产 DDL/维护冻结签署、Base FR028 上游。均是外部 gate/签署，不"转交"后误报全完成。
- **可有证据归档**：一批 `verified-complete` 记录（六月治理/部署、8-13 clean-rebuild、8-10 csrf、10-29/10-30 sonar、12 已归档引用完整、workbench O2/O4/O5/O6、cmdb-host-import-export、k8s-recovery、v0.18.6/v0.18.7）已具备成对证据，可在 Codex 验收后按原范围收口。`2026-11-07-ops-planned-task-closeout` 实质验证完成但缺 summary/卡，需补物料；`v0.18.7` live manifest 状态字段仍 `in-progress`（记录滞后，非重开）。

## 边界

未修改业务源码；未自动改任何任务状态；未进行额外归档（保持 `STATUS.md`、`OPEN_TASKS.md`、untracked 日志原样）；未提交/发布/合并；未改 lock。详见同目录 `INVENTORY.json` 与 `../<closeout>/PLAN.md`。
