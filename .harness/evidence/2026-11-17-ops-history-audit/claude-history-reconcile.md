# Ops 历史记录对账摘要（2026-11-17 执行者：本地 Claude）

父任务：`2026-11-17-ops-history-audit`；对照 `.harness/tasks/2026-11-16-ops-historical-closeout/{INVENTORY.json,PLAN.md}`。
逐项决策见同目录 `claude-history-reconcile.json`。本摘要只列四类关键项，不虚构测试结果；运行态/代码佐证由协调者 Codex 负责，本文件只核查证据文件存在性。

## 1. 可立即归档候选（27 项，均有成对 packet+evidence）

- 已验证且自含证据：`06-21-business-deploy-task-main-flow-closeout`、`08-11-sonarcloud-ops-remediation`、`10-29-sonar-ci-analysis-restore`、`10-30-sonar-findings-remediation`、`08-10-security-csrf-closeout`、`08-13-clean-base-business-overlay-rebuild`。
- 被 v0.18.9 正式消费替代的旧升级/发布范围（仅旧升级范围 supersede，不吞残余）：`08-04..08-13` 各 v0.10.x 升级/消费/认证/分发、`09-01-ops-foundation-upgrade-v0-10-25`、`10-12-v0-16-1`、`10-27-v0-17-6/-17-7`、`10-29-v0-18-1`、`10-31-v0-18-2`、`11-05-base-v0183`、`foundation-upgrade-base-v0.9.0`。
  - 注意：`10-27/10-29/10-31/11-05` 的开放 bounded residual（runtime acceptance、migration channel、SONAR_TOKEN、check:task-packet、QA002/host-import）须转移跟踪，不得被 supersede 吞掉。

## 2. 缺证据项（保持 open，不推荐 completed）

- 缺 completion summary（`.harness/evidence/<id>/` 无 `summary.md`）：`06-17-github-governance-followup`、`06-18-branch-hygiene-fallback`、`06-18-branch-hygiene-slash-path-fix`。
- 缺独立 review：`cmdb-host-import-export`、`k8s-recovery`、`2026-11-07-ops-planned-task-closeout`（顶层 review.md 仍缺，仅深层 `.tmp` 快照副本）、`2026-11-14-ops-latest-foundation-and-plans`。
- 缺整份证据/归档复盘：`08-03-dependabot-pr-governance`、`06-20-business-closure-permission-scope`、`08-12-pr-auto-merge-push-trigger`、`08-12-pr-auto-merge-token`、`2026-09-10-tenant-verification-and-gray`（仅两张图片）。

## 3. 生效项 —— 自 11-16 INVENTORY 以来的证据新增（不改旧结论，仅补记）

- `2026-07-17-cmdb-ssh-config-externalize`：manifest + summary + commands 已落盘（untracked，2026-11-17 新增），`host_service.go`/设计文档已改。INVENTORY 曾记 NOT IMPLEMENTED。**配置读取的行为佐证交 Codex/S1**，本记录保持 open 直至验证。
- `2026-11-07-ops-planned-task-closeout`、`2026-11-14-ops-latest-foundation-and-plans`：summary.md + commands.json 已补（原为悬空引用）；顶层 review 与父 task 卡仍在 P0 待补。
- `2026-11-17-ops-auto-merge-reconcile`：G1 自动合并对账（pr-automation/pr-auto-merge 两记录）今日进行中；其结果决定 `pr-auto-merge-push-trigger`/`pr-auto-merge-token`/`dependabot-pr-governance` 的收口。

## 4. 失效项（旧记录，不再作为当前缺陷证据）

- over 计数/平台层旧卡：`07-29-sonar-ts-final`、`08-02-sonarcloud-open-issues`、`08-21-sonarcloud-remediation`、`09-01-release-gate-sonar-cleanup`、`09-01-sonar-duplication-reduction` —— 归属 base / 已被 10-29/10-30 门禁与 Sonar 绿态替代。
- 描述性证据目录名不得当完成/缺陷读：`argo-adoption-{20261110,v2,v3}`（sealed-exec 自测，非 ArgoCD 优化执行）。

## 5. 上游项（12+ 项，归属 pantheon-base）

base 发布、库安全依赖、交付认证、基础工作台设计 B1-B5、生成器/共享 smoke 契约、repo.snapshot 打包、跨仓库交接、本地合并等一切 platform/system/上游 gate：按 `upstream-owned` 处理，ops 不重复验证/不改共享源码。

## 已归档引用核查（不移动）

`2026-11-07-base-v0186`、`2026-11-13-base-v0187` 及八月 8 子包：归档包 manifest/evidence 引用完好。`2026-11-13-base-v0187` 的 live manifest 状态字段仍读 `in-progress`（记录滞后），建议 P0 归一为 completed——仅状态回填，不构成取消归档。

## 责任/边界

- 本对账只写入 `claude-history-reconcile.json` / `claude-history-reconcile.md`；未改任何 status/archive/code/config/credential，未提交/发布。
- 无任何测试命令结果被虚构；全部运行态验证保留给 Codex 与最终维护者验收。
