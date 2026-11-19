# 历史任务收尾执行计划

日期：2026-11-16
父任务：`2026-11-16-ops-historical-closeout`
配套：`INVENTORY.json`（逐项 disposition）、`.harness/evidence/2026-11-16-ops-historical-closeout/intake-summary.md`
执行模型：Claude intake 已完成；本文给 Codex 派发批次用。**本文件与 INVENTORY 只对账，不实施**；真正缺实现不断言已关闭。

## In / Out

**In**

- 全部历史任务对账（INVENTORY 覆盖 60 个历史记录 + O1-O7 + D1-D5 + 9 个 infra/orphan + 12 个已归档引用核查）。
- 真正缺实现的批次（S1 SSH 配置、G1 自动合并、W1 O1、W2 O3、P1 权限 PR3）：实施并验证。
- 缺验收证据批次（G2、N1、X1 等缺失-verification）：补证据闭环。
- 外部 gate 批次（X2 D1、X4 D4）：给前置清单与 ready-to-run 提示词，实际运行等端点/签署，不"转交后误报完成"。
- 可与证据归档的 verified-complete 记录：保持孤儿/未关闭链，待 Codex 验收后按原范围成对收口。

**Out**

- 无范围关联的新功能、生产写操作、代维护者/DBA 签署、未验证的状态关闭、发布/合并、改正式 Base 锁版本、改写共享源码。
- 新 C/E 计划只承接与历史真实重叠部分；已被历史 O2/O4/O5/O6/D2 实现的，禁止因"映射"自动认定完成或重复开发。
- O7 明确延期（成熟度门禁未过），不进入本次试点；列为 deferred-forward，不静默降为可延期。

## 关键前提（跨批次，最先核对）

1. 当前已消费 **pantheon-base-v0.18.9**（`foundation-release.lock.json`，只读）。所有 `v0.10.x–v0.18.x` 升级/消费为 obsolete-with-proof 或 upstream-owned，**不重做、不开实现包**。
2. **计划 C02/C05/E04 与已实现功能重叠**：O2 部署工作台、O4 可观测查询工作台、O5 日志、O6 Dashboard 组件、D2 SSRF 代码均已在 `b51559c5`（2026-11-16，clean main）+ 后续落地。任何新计划若假设这些未建，会重复实现；应改为承接其**验收/证据**（O1、O3-K8s/ArgoCD、D1、D2 证据、O7 才是这里真正缺的）。
3. 维护者触碰点仅三处：需求澄清（一次性汇总）、门禁策略、最终验收。批间可逆工作无需逐次确认；门禁与证据替代口头确认。
4. Windows 强制门禁：`go test -race ./...` 必须 `PATH='D:\msys64\mingw64\bin;'+$PATH; CGO_ENABLED=1`。UI 触碰先读 `../pantheon-base/DESIGN.md` 并应用 `impeccable`，给渲染证据。
5. 维护者已明确授权 **Claude 实现，Codex 只派发、独立验收、运行验证和协调文档**；此前角色分配已失效。业务源码通过本地 Claude CLI 派发，执行者不得自批准。Codex 不直接修业务源码。

## 批次顺序与所有权

| 批                                           | 目标                                                                                                                                                                                                                         | 所有权（文件）                                                                                                                                                 | 依赖                                                                       | 验证                                                                                    | 退出条件                                                                                             |
| -------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------- | --------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------- |
| **P0 收尾物料**                              | v0.18.7 状态字段滞后归一；`2026-11-07-ops-planned-task-closeout` 补 summary/commands/review + 挂失于 missing task 卡；`2026-11-14-ops-latest-foundation-and-plans` 补物料；清 8 条陈旧 legacy review-allowlist；重跑 D5 基线 | `.harness/evidence/*`、`.harness/tasks/*/manifest.json`、`docs/**`、`docs/harness/tasks/*.task.md`（仅补缺的父卡）                                             | 无                                                                         | `npm run check:docs-governance`（CI 同款）、`docs:harness`、task-manifest 扫描          | 无悬空引用；manifest 状态一致；基线重跑完成；不覆盖他人改动                                          |
| **G1 自动合并矛盾**                          | 采用 `RELEASE_GATE_TOKEN` fail-closed；复用真实质量工作流的指定 SHA 恢复入口；保留 tests/scripts catch-all；统一测试与证据                                                                                                   | `.github/workflows/pr-automation.yml`、必要 Ops 质量工作流消费入口、`tests/scripts/{pr-automation-workflow,ci-workflow-dispatch}.test.mjs`                     | 不创建空壳 `ci.yml`；遵守共享工作流边界                                    | 自动合并、质量工作流、dispatch 与 catch-all 测试                                        | token 不回落；所有相关步骤验证目标 SHA；hosted 未运行则保留 gap                                      |
| **G2 缺陷-验证补证**                         | `dependabot-pr-governance` 补 manifest+commands/review+hosted 记录；`closure-permission-scope` 做全端点 DataScope 审计+测试+证据                                                                                             | `backend/modules/business/deploy/*`、`backend/modules/business/bizscope/*`、`.harness/{tasks,evidence}/2026-08-03-*`、`.harness/{tasks,evidence}/2026-06-20-*` | 无                                                                         | `go test ./backend/modules/business/...`                                                | 两条记录达 verified-complete（有证据）；不新增业务代码除非审计发现真缺口                             |
| **S1 SSH 配置外部化**（真正缺实现）          | `cmdb.ssh.collect_timeout`(默认10) + `cmdb.ssh.default_port`(默认22) 种子；host collect 经 Base 扩展点读取并回退 10s/22；删 §11.2 待办                                                                                       | `backend/modules/business/cmdb/host/host_service.go`、`backend/modules/business/cmdb/module.go`、`docs/designs/BUSINESS_CMDB_MODULE_DESIGN.md`                 | **阻塞：Base 是否暴露 business 可消费 config 扩展点**；若无则 STOP 回 Base | `go test ./backend/modules/business/cmdb/host` + 改配置值并验证生效                     | 两条配置 key 生效且改动配置改变采集行为；§11.2 待办删除；Base-first 不直接 import system/config 内部 |
| **P1 权限缺口收口**（insufficient-evidence） | 确认/关闭 `GET /packages/:id` 在 permissionroutes+keyMatch2 下授权（显式绑定或通过性测试）；同步 deploy 设计文档缺口注释                                                                                                     | `backend/modules/business/deploy/*`、`backend/modules/business/cmdb/permission_routes.go`                                                                      | 无                                                                         | `go test ./backend/modules/business/deploy ./backend/modules/business/cmdb/...`         | PR3 有明确结论（authorized 或显式补绑定）；`permission-gaps` 达 verified 或明确 external             |
| **W1 O1 上下文/目标选择器**（真正缺实现）    | 共享 BizScope 环境上下文头 + 通用 context/target 选择器适配 CMDB capability                                                                                                                                                  | `frontend/src/modules/business/bizscope/*`、`frontend/src/modules/business/shared/*`、`backend/modules/business/cmdb/target_selection.go`                      | **基础 B3 ContextSelector release**（先验证，若未发布则阻塞）              | `npm run type-check`、`go test ./modules/business/bizscope ./modules/business/cmdb`     | 统一选择器完成（10k bounded、全键盘、移动端、审计）；无 secret                                       |
| **W2 O3 K8s/ArgoCD 预览**（真正缺实现）      | K8s release 预览 + ArgoCD sync/rollback 版本树 diff + 跨域预览 envelope + 409 冲突；与 C02/E02 重叠协调                                                                                                                      | `frontend/src/modules/business/k8s/release/*`、`frontend/src/modules/business/cicd/*`、`backend/modules/business/deploy/task_preview.go`                       | **依赖基础 B3 ChangeDiff + ArgoCD 端点**                                   | `go test ./modules/business/deploy -run Preview`、Playwright preview->confirm->conflict | K8s/ArgoCD diff 预览闭环；secrets 脱敏；与现存 Deploy 预览同契约                                     |
| **X1 D2 SSRF 验收证据**（缺失-verification） | 已部署 allowlist/private-network/redirect/DNS 规则负向测试（127.0.0.1/169.254.169.254 拒绝）+ 审计；出 D2 验收证据包                                                                                                         | `backend/modules/business/observability/{datasource_service,notification_service}.go`、`backend/modules/business/cicd/argocd`、新 `.harness/evidence/...`      | 维护者 SSRF 策略决定                                                       | `go test ./pkg/security/ssrf/`、负向创建 datasource 用例                                | D2 达 verified-complete；策略号记录；`backend/pkg/security/ssrf` 归属 base-shared 确认无误           |
| **X2 D1 外部集成验收**（外部 gate）          | 对真实 Prometheus/VM、Loki、Tempo、K8s、ArgoCD、Velero 跑矩阵 + redacted 证据                                                                                                                                                | `backend/modules/business/observability/datasource_*`、`k8s/cluster`、`cicd/argocd`、velero 面                                                                 | **外部端点/凭证 gated + 维护者窗口**                                       | env-gated live 测试                                                                     | 六系统真实验收证据 + maintainer review；**未取得端点不宣称完成**                                     |
| **X4 D4-G2/G3**（生产 gate）                 | 生产恢复演练测 RPO/RTO + 签署；生产量级 sizing + 迁移窗口 workbook                                                                                                                                                           | `docs/runbooks/*`、`backend/cmd/tenantsizing`                                                                                                                  | **DBA 窗口/sign-off gated**                                                | 生产只读 snapshot/plan + 演练                                                           | §8.1 G2/G3 记录更新 + DBA/maintainer 签署；无签署则保持 OPEN                                         |
| **O7**                                       | 关系成熟度门禁判断                                                                                                                                                                                                           | 无                                                                                                                                                             | 成熟关系模型                                                               | fixture 用例                                                                            | 记录门禁结果；本次不实现                                                                             |

**不派发批次**（仅证据收口/维持）：`obsolete-with-proof`(23)、`upstream-owned`(12)、已 verified 的 June 治理/部署、8-13 rebuild、8-10 csrf、10-29/10-30 sonar、O2/O4/O5/O6、cmdb-host-import-export、k8s-recovery、12 已归档——无实现工作，Codex 验收后按原范围归档或维持。

## 共享工作区纪律

所有执行者共享同一 worktree；**不得覆盖/撤销/回滚他人未提交改动**。批次间文件无重叠；G1 与 G2 都触碰 `.github/workflows/pr-automation.yml` —— **串行**执行（先 G1 后 G2，或 G2 仅补 2026-08-03 的 manifest/evidence，不碰该 workflow）。任何批不改 `foundation-release.lock.json`、不发布、不新建业务模块、不手抄 Base 源码。

## 验收（Codex）

1. 每历史 ID/子项一个结论 + 原始依据 + 当前事实，以 INVENTORY.netJSON 核对。
2. 缺实现项实施并验证（S1/G1/W1/W2/P1）；缺真实环境项无法获取则**保持开放**，不得转交后报全完成。
3. O1-O7 逐项对原 packet；O7 若成熟度缺失说明真实前置。
4. 有闭环证据才归档，task/evidence 成对、引用有效；不得回写虚构历史命令。
5. Harness 检查/文档链接/business overlay/代码+运行检查通过；UI 按 impeccable 给渲染证据。
6. 逐项记录 accepted/rejected/external-gate；仅无剩余必做项时报告全完成。

## 已知外部/前置（不闭合不入账）

- D1 六系统端点/凭证、D4 G2/G3 DBA+生产窗口+签署、D2 维护者策略决定、SONAR_TOKEN 轮换、生产 DDL/维护冻结签署、Base FR028（上游）、维护者验收。

## 最终本批次返回物

- `INVENTORY.json`（逐项 disposition）、`PLAN.md`（本文）、`prompts/<批>.md`（每批派发提示词，见 `prompts/`）、`intake-summary.md`、`ask-claude-historical-intake-20261116.result.json`。仅这些目录写入；其余只读。
