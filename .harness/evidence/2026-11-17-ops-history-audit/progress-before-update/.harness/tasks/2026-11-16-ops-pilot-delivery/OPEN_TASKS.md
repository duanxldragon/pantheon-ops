# Pantheon Ops 全量开放任务与旧记录盘点

核对日期：2026-11-17。新计划只有10个业务子包，不是全部历史待办。以下分别列当前执行包、历史未关闭队列、证据/状态不完整记录；父包与子项、同一gap在新旧计划出现不重复相加。

19组历史处置已执行（17新归档＋2重复live保全），未关闭的新业务与外部门禁保持开放。详见本轮[处置清单](../../evidence/2026-11-17-ops-history-audit/DISPOSITIONS.md)。

## 当前业务执行包

| 编号 | 任务                                                                                                                 | 状态    | 范围                               |
| ---- | -------------------------------------------------------------------------------------------------------------------- | ------- | ---------------------------------- |
| C01  | [部署/服务实例关联与引用保护](../../../docs/harness/tasks/2026-11-16-ops-service-deploy-integrity.task.md)           | planned | 首期必交                           |
| C02  | [页面部署与实例结果回写](../../../docs/harness/tasks/2026-11-16-ops-service-deploy-user-flow.task.md)                | planned | 首期必交                           |
| C03  | [服务目录编辑、分页、受限角色](../../../docs/harness/tasks/2026-11-16-ops-service-catalog-usability.task.md)         | planned | 首期必交                           |
| C04  | [K8s Service 更新及并发保护](../../../docs/harness/tasks/2026-11-16-ops-k8s-service-mutations.task.md)               | planned | 首期必交                           |
| C05  | [当前候选真实环境、权限、业务UI验收](../../../docs/harness/tasks/2026-11-16-ops-business-runtime-acceptance.task.md) | planned | 首期必交                           |
| C06  | [交付包、安装、升级、回滚与最终验收](../../../docs/harness/tasks/2026-11-16-ops-pilot-release-closeout.task.md)      | planned | 首期必交                           |
| E01  | [文件分发创建入口与可靠控制](../../../docs/harness/tasks/2026-11-16-ops-filedist-product-closure.task.md)            | planned | 条件扩展；开放必须验收，裁剪须留证 |
| E02  | [ArgoCD 归属隔离及真实状态收敛](../../../docs/harness/tasks/2026-11-16-ops-argocd-ownership-convergence.task.md)     | planned | 条件扩展；开放必须验收，裁剪须留证 |
| E03  | [Helm 能力口径与真实Chart边界](../../../docs/harness/tasks/2026-11-16-ops-helm-capability-truth.task.md)             | planned | 条件扩展；开放必须验收，裁剪须留证 |
| E04  | [实际开放扩展的真实外部验收](../../../docs/harness/tasks/2026-11-16-ops-extensions-live-acceptance.task.md)          | planned | 条件扩展；开放必须验收，裁剪须留证 |

父包 [2026-11-16-ops-pilot-delivery](./manifest.json) 为planned，仅统计容器。菜单 [menu-m3c](../../../docs/harness/tasks/2026-11-16-ops-business-menu-m3c.task.md) 实现已合并但运行/视觉验收未完，状态in-progress，交C05关闭。本轮readiness-plan-closeout治理已完成。

## 历史工作台队列：O1–O7尚未关闭

来源：[旧STATUS](../2026-08-31-ops-operational-workbench-design/STATUS.md)和[队列](../2026-08-31-ops-operational-workbench-design/EXECUTION_QUEUE.md)。历史状态不代表当前没有代码；Base“未发布”阻断标签已过时，需对照9/1消费及当前v0.18.9核对实际适配/验收。

| 旧编号 | 工作项                  | 原记录          | 与新计划关系                                |
| ------ | ----------------------- | --------------- | ------------------------------------------- |
| O1     | 统一业务上下文/目标选择 | blocked，未关闭 | 部分与C02/C03重叠；范围对账                 |
| O2     | 部署执行工作台          | blocked，未关闭 | C02/C05承接部分闭环，完整工作台不自动完成   |
| O3     | 变更预览、差异与风险    | blocked，未关闭 | C02/C05部分重叠，完整change diff仍需核对    |
| O4     | 可观测查询工作台        | blocked，未关闭 | 当前已有查询实现；E04验收，旧工作台目标对账 |
| O5     | 部署任务/Pod日志体验    | blocked，未关闭 | 当前已有日志API/页面；完整体验需旧合同验收  |
| O6     | 业务Dashboard组件       | blocked，未关闭 | 未在首期10包显式包含，独立后续目标          |
| O7     | CMDB关系/影响发现       | deferred        | 明确延期，非当前试点退出项                  |

## 历史交付门禁：D1、D2、D4、D5

来源：[D队列STATUS](../2026-09-28-ops-delivery-readiness-gap-closure/STATUS.md)。D3已完成；旧版本、估时与阻断原因不直接套用当前。

| 编号  | 剩余                                                        | 当前口径                                                                                                                            |
| ----- | ----------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------- |
| D1    | Prometheus/VM、Loki、Tempo、K8s、ArgoCD、Velero真实外部验收 | C05/E04覆盖，仍缺同候选证据                                                                                                         |
| D2    | 出站SSRF部署策略与完整负向验收                              | 9/29实现done；不是未写保护代码，尚缺策略/验收/关闭证据                                                                              |
| D4/G2 | 生产备份恢复、RPO/RTO签署                                   | [runbook仍OPEN](../../../docs/runbooks/TENANT_MIGRATION_RUNBOOK.md)，独立生产轨道                                                   |
| D4/G3 | 生产量级、迁移窗口与必要的大表演练                          | runbook仍OPEN，独立生产轨道                                                                                                         |
| D5    | Harness/生成资产/文档历史债务                               | 新治理范围通过；上轮全仓扫描仍有567项task错误、6项坏链接、50项review及149项evidence格式问题。此为上轮扫描基线，本轮未重跑全仓门禁。 |

另外，CMDB导入fixture/数据库smoke与K8s真实集群验收由C05承接；旧V1/V1.2真实验收、安装/发布未完成部分由C05/C06承接。真实S3 provider、维护冻结签署和Base FR028仍在上游/生产轨道，本次未关闭。

## 单独发现的旧业务遗留

- [2026-07-17-cmdb-ssh-config-externalize](../../../docs/harness/tasks/2026-07-17-cmdb-ssh-config-externalize.task.md)：SSH采集超时与默认端口通过Base config扩展点配置化，当前设计仍标未接入；未见业务源码使用两项key。当前10包没有显式包含它，应单独保留或明确延期。
- O6业务Dashboard、O7CMDB关系发现、D2出站策略、D4生产迁移、D5历史治理不能被10包全部替代；旧任务和新计划需要逐项映射后才能关闭。

## 未归档旧任务记录：逐项口径

| Task                                                                                                                                         | 机器状态             | 仍需处理/判定                                                                                                                                                                                                                       |
| -------------------------------------------------------------------------------------------------------------------------------------------- | -------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| [2026-06-17-github-governance-followup](../../../docs/harness/tasks/2026-06-17-github-governance-followup.task.md)                           | Archived (completed) | 2026-11-17 已按原范围审查归档；原治理、模板和工作流姿态已获批准；补记摘要，不扩大为产品完成。 原证据与残余见 `.harness/evidence/2026-11-17-ops-history-audit/archive-result.json`。                                                 |
| [2026-06-18-branch-hygiene-fallback](../../../docs/harness/tasks/2026-06-18-branch-hygiene-fallback.task.md)                                 | Archived (completed) | 2026-11-17 已按原范围审查归档；原测试和独立批准保留；找回 hosted run27737850759/27738932286，证明真实 closed-PR residue 删除。 原证据与残余见 `.harness/evidence/2026-11-17-ops-history-audit/archive-result.json`。                |
| [2026-06-18-branch-hygiene-slash-path-fix](../../../docs/harness/tasks/2026-06-18-branch-hygiene-slash-path-fix.task.md)                     | Archived (completed) | 2026-11-17 已按原范围审查归档；run27738932286 明确删除含斜杠的 PR58/59 分支，关闭原 hosted 缺证据；未重新执行删除。 原证据与残余见 `.harness/evidence/2026-11-17-ops-history-audit/archive-result.json`。                           |
| [2026-06-18-pr-branch-cleanup-automation](../../../docs/harness/tasks/2026-06-18-pr-branch-cleanup-automation.task.md)                       | abandoned / Archived | 旧 pull_request.closed 耦合已由两项 branch-hygiene 后继替代；独立批准并成对归档，原 evidence 保全。本地 divergent/未发布分支仍由 workspace 维护者负责。                                                                             |
| [2026-06-21-business-deploy-task-main-flow-closeout](../../../docs/harness/tasks/2026-06-21-business-deploy-task-main-flow-closeout.task.md) | Archived (completed) | 2026-11-17 已按原范围审查归档；原 canonical 状态/重试/删除语义与定向验证获批准，当前候选运行与渲染单独验收。 原证据与残余见 `.harness/evidence/2026-11-17-ops-history-audit/archive-result.json`。                                  |
| [2026-08-05-foundation-upgrade-base-v0-10-1](../../../docs/harness/tasks/2026-08-05-foundation-upgrade-base-v0-10-1.task.md)                 | 未规范记录           | 无当前summary/review；旧版本已被后续正式消费替代，待历史收口核对。                                                                                                                                                                  |
| [2026-08-05-foundation-upgrade-base-v0-10-2](../../../docs/harness/tasks/2026-08-05-foundation-upgrade-base-v0-10-2.task.md)                 | 未规范记录           | 无当前summary/review；旧版本已被后续正式消费替代，待历史收口核对。                                                                                                                                                                  |
| [2026-08-07-foundation-upgrade-base-v0-10-3](../../../docs/harness/tasks/2026-08-07-foundation-upgrade-base-v0-10-3.task.md)                 | 未规范记录           | 本地实现完成，旧摘要仍写hosted/PR75待处理；不能把旧分支状态当作当前待办，需历史对账。                                                                                                                                               |
| [2026-08-08-foundation-upgrade-base-v0-10-5](../../../docs/harness/tasks/2026-08-08-foundation-upgrade-base-v0-10-5.task.md)                 | 未规范记录           | 本地消费完成；旧PR75/CSRF门禁记录需对账，后续版本已替代。                                                                                                                                                                           |
| [2026-08-10-security-csrf-closeout](../../../.harness/tasks/2026-08-10-security-csrf-closeout/manifest.json)                                 | Archived (completed) | 2026-11-17 已按原范围审查归档；原 CSRF 修复、不可变消费和 hosted Security/Quality/Smoke 证据成立，不推断当前无漏洞。 原证据与残余见 `.harness/evidence/2026-11-17-ops-history-audit/archive-result.json`。                          |
| [2026-08-11-foundation-consumer-certification](../../../.harness/tasks/2026-08-11-foundation-consumer-certification/manifest.json)           | Archived (abandoned) | 2026-11-17 已按原范围审查归档；旧 marker/tree-checksum 工具链由 8/13 clean rebuild 明确废弃，当前使用正式 lock/snapshot assembler。 原证据与残余见 `.harness/evidence/2026-11-17-ops-history-audit/archive-result.json`。           |
| [2026-08-11-sonarcloud-ops-remediation](../../../.harness/tasks/2026-08-11-sonarcloud-ops-remediation/manifest.json)                         | Archived (abandoned) | 2026-11-17 已按原范围审查归档；原零问题目标未在八月完成，当时仍156项；10/30 main430a9f0 分析已零问题/gate OK/dup2.0，旧目标由其替代。 原证据与残余见 `.harness/evidence/2026-11-17-ops-history-audit/archive-result.json`。         |
| [2026-08-12-foundation-v0-10-14-clean-consumer](../../../.harness/tasks/2026-08-12-foundation-v0-10-14-clean-consumer/manifest.json)         | 未规范记录           | 实际目标滚动到v0.10.21，旧summary/review Pending；后续清洁重建及v0.18.x已完成，历史范围需要对账。                                                                                                                                   |
| [2026-08-13-clean-base-business-overlay-rebuild](../../../.harness/tasks/2026-08-13-clean-base-business-overlay-rebuild/manifest.json)       | Archived (completed) | 2026-11-17 已按原范围审查归档；原锁定源码、原子替换和消费机制已落地；August本地repo.tar/未跑race保持历史事实，后续正式release消费提供新证据。 原证据与残余见 `.harness/evidence/2026-11-17-ops-history-audit/archive-result.json`。 |
| [2026-08-19-ops-v1-backlog-closeout](../../../docs/harness/tasks/2026-08-19-ops-v1-backlog-closeout.task.md)                                 | Archived (completed) | 2026-11-17 已按原范围审查归档；规范化 complete-with-explicit-gates；仅授权隔离F阶段完成，保留生产与人工门禁。 原证据与残余见 `.harness/evidence/2026-11-17-ops-history-audit/archive-result.json`。                                 |
| [2026-08-31-ops-operational-workbench-design](../../../docs/harness/tasks/2026-08-31-ops-operational-workbench-design.task.md)               | planned              | 设计完成；O1–O7旧实施队列未关闭。O1–O6的“Base未发布”标签已过时：9/1已有B1–B5消费证据，需对当前实现/验收对账。                                                                                                                       |
| [2026-09-28-ops-delivery-readiness-gap-closure](../../../docs/harness/tasks/2026-09-28-ops-delivery-readiness-gap-closure.task.md)           | planned              | D1外部验收、D2出站策略完整验收、D4租户G2/G3、D5治理仍未完整关闭；D3已完成不再计入。                                                                                                                                                 |
| [2026-09-29-ops-ssrf-protection-integration](../../../.harness/tasks/2026-09-29-ops-ssrf-protection-integration/manifest.json)               | done                 | 实现manifest为done但无summary/review；不算SSRF从零开发。D2还要证明实际部署allowlist/private-network/redirect/DNS规则、审计和完整负向验收。                                                                                          |
| [2026-10-12-foundation-upgrade-base-v0-16-1](../../../docs/harness/tasks/2026-10-12-foundation-upgrade-base-v0-16-1.task.md)                 | 未规范记录           | 旧summary仍有CGO/发布/hostedgap；后续v0.17.6–v0.18.9已消费，旧版本不重做，需历史收口。                                                                                                                                              |
| [cmdb-host-import-export](../../../.harness/tasks/cmdb-host-import-export/plan.md)                                                           | 未规范记录           | STATUS=complete，CSV实现完成；数据库业务smoke与桌面/窄屏验收未在该任务重跑，host-import fixture残余由C05承接。                                                                                                                      |
| [k8s-recovery](../../../.harness/evidence/k8s-recovery/summary.md)                                                                           | 缺manifest           | 模块恢复/装配完成；原任务缺live-cluster/race证据，当前live-cluster验收由C05承接，不重复恢复模块。                                                                                                                                   |

## 只有旧文档/缺关联链的任务卡

下表保留全部未被manifest/已归档任务映射的旧卡。多数是Base历史任务副本或已实现但未更新状态；它们是待核实记录，不等于当前业务缺陷清单。

| 旧卡                                                                                                                                           | 文档状态 | 核对结论                                                                                                               |
| ---------------------------------------------------------------------------------------------------------------------------------------------- | -------- | ---------------------------------------------------------------------------------------------------------------------- |
| [2026-06-20-business-closure-permission-scope](../../../docs/harness/tasks/2026-06-20-business-closure-permission-scope.task.md)               | 无状态   | 旧权限/数据范围任务，缺关闭链，当前角色验收交C05；需核对是否已被六月闭环完成。                                         |
| [2026-06-21-business-deploy-state-guard](../../../docs/harness/tasks/2026-06-21-business-deploy-state-guard.task.md)                           | 无状态   | 旧状态限制任务，后续Deploy状态机已有实现，待记录对账，不直接认定缺代码。                                               |
| [2026-06-21-business-deploy-task-delete](../../../docs/harness/tasks/2026-06-21-business-deploy-task-delete.task.md)                           | 无状态   | 旧任务删除语义，后续Deploy主流程已覆盖，待关闭证据对账。                                                               |
| [2026-07-17-bizscope-error-key-canonicalization](../../../docs/harness/tasks/2026-07-17-bizscope-error-key-canonicalization.task.md)           | 无状态   | 当前业务错误常量和附录已迁移canonical；旧key仅兼容locale清单，待旧任务关闭记录。                                       |
| [2026-07-17-business-design-docs-truthing](../../../docs/harness/tasks/2026-07-17-business-design-docs-truthing.task.md)                       | 无状态   | 多项已完成/部分文档仍保留旧缺口；需要与新功能设计及源码对账，不整包认定尚未实施。                                      |
| [2026-07-17-business-permission-gaps](../../../docs/harness/tasks/2026-07-17-business-permission-gaps.task.md)                                 | 无状态   | 模板按钮seed与CMDB label-options映射当前源码已存在；旧卡和旧设计缺口叙述过时。待完整权限验收/状态收口。                |
| [2026-07-17-cmdb-ssh-config-externalize](../../../docs/harness/tasks/2026-07-17-cmdb-ssh-config-externalize.task.md)                           | 无状态   | 独立业务遗留：SSH采集timeout/default_port配置外部化。设计仍标未接入，business源码未见两配置key；未被当前10包显式覆盖。 |
| [2026-07-29-sonar-ts-final](../../../docs/harness/tasks/2026-07-29-sonar-ts-final.task.md)                                                     | 无状态   | 只有旧任务文档或缺关联证据，待历史核对；未证实当前实现未完成。                                                         |
| [2026-07-30-local-consolidation](../../../docs/harness/tasks/2026-07-30-local-consolidation.task.md)                                           | 无状态   | 上游/跨仓库历史卡，非独立Ops业务任务；在Base核实当前完成状态，不按Active算Ops剩余。                                    |
| [2026-08-02-sonarcloud-open-issues](../../../docs/harness/tasks/2026-08-02-sonarcloud-open-issues.task.md)                                     | Active   | 只有旧任务文档或缺关联证据，待历史核对；未证实当前实现未完成。                                                         |
| [2026-08-03-dependabot-pr-governance](../../../docs/harness/tasks/2026-08-03-dependabot-pr-governance.task.md)                                 | Active   | 只有旧任务文档或缺关联证据，待历史核对；未证实当前实现未完成。                                                         |
| [2026-08-03-sonarcloud-encoding-warning](../../../docs/harness/tasks/2026-08-03-sonarcloud-encoding-warning.task.md)                           | Active   | 只有旧任务文档或缺关联证据，待历史核对；未证实当前实现未完成。                                                         |
| [2026-08-04-base-v0-10-0-release-candidate](../../../docs/harness/tasks/2026-08-04-base-v0-10-0-release-candidate.task.md)                     | Archived | 只有旧任务文档或缺关联证据，待历史核对；未证实当前实现未完成。                                                         |
| [2026-08-05-base-v0-10-1-release](../../../docs/harness/tasks/2026-08-05-base-v0-10-1-release.task.md)                                         | Active   | 上游/跨仓库历史卡，非独立Ops业务任务；在Base核实当前完成状态，不按Active算Ops剩余。                                    |
| [2026-08-05-base-v0-10-2-visual-checker](../../../docs/harness/tasks/2026-08-05-base-v0-10-2-visual-checker.task.md)                           | Active   | 上游/跨仓库历史卡，非独立Ops业务任务；在Base核实当前完成状态，不按Active算Ops剩余。                                    |
| [2026-08-06-generator-business-page-route](../../../docs/harness/tasks/2026-08-06-generator-business-page-route.task.md)                       | Active   | 只有旧任务文档或缺关联证据，待历史核对；未证实当前实现未完成。                                                         |
| [2026-08-10-shared-csrf-release](../../../docs/harness/tasks/2026-08-10-shared-csrf-release.task.md)                                           | Active   | 只有旧任务文档或缺关联证据，待历史核对；未证实当前实现未完成。                                                         |
| [2026-08-11-base-security-dependencies](../../../docs/harness/tasks/2026-08-11-base-security-dependencies.task.md)                             | Archived | 只有旧任务文档或缺关联证据，待历史核对；未证实当前实现未完成。                                                         |
| [2026-08-11-delivery-certification](../../../docs/harness/tasks/2026-08-11-delivery-certification.task.md)                                     | Active   | 只有旧任务文档或缺关联证据，待历史核对；未证实当前实现未完成。                                                         |
| [2026-08-11-foundation-release-v0-10-12](../../../docs/harness/tasks/2026-08-11-foundation-release-v0-10-12.task.md)                           | Active   | 上游/跨仓库历史卡，非独立Ops业务任务；在Base核实当前完成状态，不按Active算Ops剩余。                                    |
| [2026-08-11-foundation-release-v0-10-13](../../../docs/harness/tasks/2026-08-11-foundation-release-v0-10-13.task.md)                           | Active   | 上游/跨仓库历史卡，非独立Ops业务任务；在Base核实当前完成状态，不按Active算Ops剩余。                                    |
| [2026-08-12-foundation-consumer-cleanup-overlay](../../../docs/harness/tasks/2026-08-12-foundation-consumer-cleanup-overlay.task.md)           | Active   | 上游/跨仓库历史卡，非独立Ops业务任务；在Base核实当前完成状态，不按Active算Ops剩余。                                    |
| [2026-08-12-foundation-consumer-runtime-portability](../../../docs/harness/tasks/2026-08-12-foundation-consumer-runtime-portability.task.md)   | Active   | 上游/跨仓库历史卡，非独立Ops业务任务；在Base核实当前完成状态，不按Active算Ops剩余。                                    |
| [2026-08-12-foundation-smoke-closure](../../../docs/harness/tasks/2026-08-12-foundation-smoke-closure.task.md)                                 | Active   | 上游/跨仓库历史卡，非独立Ops业务任务；在Base核实当前完成状态，不按Active算Ops剩余。                                    |
| [2026-08-12-foundation-smoke-contract-closure](../../../docs/harness/tasks/2026-08-12-foundation-smoke-contract-closure.task.md)               | Active   | 上游/跨仓库历史卡，非独立Ops业务任务；在Base核实当前完成状态，不按Active算Ops剩余。                                    |
| [2026-08-12-pr-auto-merge-push-trigger](../../../docs/harness/tasks/2026-08-12-pr-auto-merge-push-trigger.task.md)                             | Active   | 只有旧任务文档或缺关联证据，待历史核对；未证实当前实现未完成。                                                         |
| [2026-08-12-pr-auto-merge-token](../../../docs/harness/tasks/2026-08-12-pr-auto-merge-token.task.md)                                           | Active   | 只有旧任务文档或缺关联证据，待历史核对；未证实当前实现未完成。                                                         |
| [2026-08-13-foundation-release-repo-snapshot](../../../docs/harness/tasks/2026-08-13-foundation-release-repo-snapshot.task.md)                 | Active   | 上游/跨仓库历史卡，非独立Ops业务任务；在Base核实当前完成状态，不按Active算Ops剩余。                                    |
| [2026-08-17-foundation-cross-repo-handoff-adoption](../../../docs/harness/tasks/2026-08-17-foundation-cross-repo-handoff-adoption.task.md)     | Active   | 上游/跨仓库历史卡，非独立Ops业务任务；在Base核实当前完成状态，不按Active算Ops剩余。                                    |
| [2026-08-21-sonarcloud-remediation](../../../docs/harness/tasks/2026-08-21-sonarcloud-remediation.task.md)                                     | 无状态   | 只有旧任务文档或缺关联证据，待历史核对；未证实当前实现未完成。                                                         |
| [2026-08-31-base-operational-workbench-design](../../../docs/harness/tasks/2026-08-31-base-operational-workbench-design.task.md)               | Active   | 上游/跨仓库历史卡，非独立Ops业务任务；在Base核实当前完成状态，不按Active算Ops剩余。                                    |
| [2026-08-31-ops-v1-2-production-readiness-round2](../../../docs/harness/tasks/2026-08-31-ops-v1-2-production-readiness-round2.task.md)         | 无状态   | 历史实现/截图完成，但外部验收/出站策略/全部CRUD点击验收未通过；由D1/D2/C05/E04承接。                                   |
| [2026-09-01-ops-foundation-upgrade-v0-10-25](../../../docs/harness/tasks/2026-09-01-ops-foundation-upgrade-v0-10-25.task.md)                   | Active   | 消费证据已证明B1–B5正式资产/装配验证；旧replacement/human gate段待历史对账，不应再把O1–O6说成Base合同从未发布。        |
| [2026-09-01-release-docs-reconciliation](../../../docs/harness/tasks/2026-09-01-release-docs-reconciliation.task.md)                           | 无状态   | 上游/跨仓库历史卡，非独立Ops业务任务；在Base核实当前完成状态，不按Active算Ops剩余。                                    |
| [2026-09-01-release-gate-sonar-cleanup](../../../docs/harness/tasks/2026-09-01-release-gate-sonar-cleanup.task.md)                             | 无状态   | 上游/跨仓库历史卡，非独立Ops业务任务；在Base核实当前完成状态，不按Active算Ops剩余。                                    |
| [2026-09-01-sonar-duplication-reduction](../../../docs/harness/tasks/2026-09-01-sonar-duplication-reduction.task.md)                           | Active   | 上游/跨仓库历史卡，非独立Ops业务任务；在Base核实当前完成状态，不按Active算Ops剩余。                                    |
| [2026-09-02-foundation-release-v0-10-26-and-ops-sync](../../../docs/harness/tasks/2026-09-02-foundation-release-v0-10-26-and-ops-sync.task.md) | 无状态   | 上游/跨仓库历史卡，非独立Ops业务任务；在Base核实当前完成状态，不按Active算Ops剩余。                                    |

## 已完成但仍留在live目录的任务

以下不计未完成：

- [2026-10-27-foundation-upgrade-base-v0-17-6](../../../.harness/tasks/2026-10-27-foundation-upgrade-base-v0-17-6/manifest.json)：completed。
- [2026-10-27-foundation-upgrade-base-v0-17-7](../../../.harness/tasks/2026-10-27-foundation-upgrade-base-v0-17-7/manifest.json)：completed。
- [2026-10-29-foundation-upgrade-base-v0-18-1](../../../.harness/tasks/2026-10-29-foundation-upgrade-base-v0-18-1/manifest.json)：completed。
- [2026-10-29-sonar-ci-analysis-restore](../../../.harness/tasks/2026-10-29-sonar-ci-analysis-restore/manifest.json)：completed。
- [2026-10-30-sonar-findings-remediation](../../../.harness/tasks/2026-10-30-sonar-findings-remediation/manifest.json)：completed。
- [2026-10-31-foundation-upgrade-base-v0-18-2](../../../.harness/tasks/2026-10-31-foundation-upgrade-base-v0-18-2/manifest.json)：completed。
- [2026-11-05-base-v0183-consumption](../../../.harness/tasks/2026-11-05-base-v0183-consumption/manifest.json)：completed。

其中2026-11-05-base-v0183-consumption实际最终目标v0.18.5，manifest记录PR177已于11/7合并。后续v0.18.6证据中另有[合并raw](../../../.harness/archive/2026-11/evidence/2026-11-07-base-v0186-consumption/ops-pr177-prior-merged.json)及required checks通过记录。旧summary顶部Pending已过时；消费本身已完成，QA002/host-import/S3/freeze等残余须另跟踪。上轮未归档它是保守留档，不应据旧Pending把它继续算未完成。

12个已物理归档历史任务见[归档索引](../../ARCHIVE.md)，不重复统计。当前已提交Base v0.18.9消费不计未完成升级。

## 没有完成摘要的证据目录

这些只说明日志存在、关闭链缺失，不能凭目录名认定开发没做或完成：

- .harness/evidence/2026-06-17-github-governance-followup/
- .harness/evidence/2026-06-18-branch-hygiene-fallback/
- .harness/evidence/2026-06-18-branch-hygiene-slash-path-fix/
- .harness/evidence/2026-09-10-tenant-verification-and-gray/
- .harness/evidence/2026-11-07-ops-planned-task-closeout/
- .harness/evidence/2026-11-14-ops-latest-foundation-and-plans/
- .harness/evidence/argo-adoption-20261110/
- .harness/evidence/argo-adoption-v2-20261110/
- .harness/evidence/argo-adoption-v3-20261110/
- .harness/evidence/foundation-upgrade-base-v0.9.0/

## 使用此清单

执行当前C/E计划；旧O/D任务按映射关闭，不重复开实现包。只有缺关闭证据的旧版本/旧卡先做历史对账；过时Sonar计数、旧CGO失败、旧Base未发布标签不作为今天仍存在的事实。本文仅盘点，不修改业务源码、不自动改任务状态或进行额外归档。
