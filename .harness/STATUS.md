# Pantheon Ops 当前状态

本页是当前重点与验收 gap 的入口；状态和计数以 [TASK_INDEX](./TASK_INDEX.md) 及各包 manifest 为准。进度追加到目标任务 summary，历史检查点单独保存。

## 当前工作

- 当前执行主线：[审查整改父计划](./tasks/2026-11-20-ops-review-remediation/plan.md) / [执行队列](./tasks/2026-11-20-ops-review-remediation/EXECUTION_QUEUE.md)。原十包未验收 residual 由本主线承接。
- Base 消费版本及 commit 以 [foundation lock](../foundation-release.lock.json) 为准；产品 VERSION、消费版本、实际运行候选分别记录。
- 11/24 检查点记录 PR #205 合并到 `5e4dac4f`，随后第八轮记录 T06 读授权/路由的代码级收口及其残余 gap；这些结论只证明原候选。#209-#219 已并入 main（`11f9b004`；#216 已合并为 `98536489`）；本轮 O-01/O-04 两包在 `11f9b004` 上复核后提交，其余历史结论仍按原候选阅读。
- 11/24 关闭[业务指标语义与集群维度收敛](./tasks/2026-11-24-ops-metrics-semantics/manifest.json)：多集群容量按 `cluster` 标签拆分、主机状态改整快照归零、属性标签注释更正；含 RED/GREEN、`-race`、build、增量 lint 与边界门禁证据（L1/P2，代码级闭合，Grafana 资产对齐仍开放）。
- 11/24 关闭[Compose 凭据外置与废弃 DDL 挂载移除](./tasks/2026-11-24-ops-compose-credentials/manifest.json)：口令改为 `${VAR:?}` 必填插值、移除 `system_init.sql` initdb 挂载，配置解析级双证据（缺变量失败/齐备成功），运行态未测（L2/P1）。
- 11/25 收口：T04（核心租户隔离）、T16（真实环境验收）与父计划三包按维护者指示关为 completed；残余拆为 [T18 业务详情路由租户绑定](./tasks/2026-11-25-ops-tenant-detail-route-binding/manifest.json)（P1，等 Base 正式 release）与 [T19 真实环境残余验收](./tasks/2026-11-25-ops-residual-live-acceptance/manifest.json)（P0，等多租户/受限用户/测试 Redis/Ingress/Velero 环境）。completed 不代表生产验收。
- 11/24 为 T04 补异步任务重启租户保持的回归测试（`deploy/tenant_async_recovery_test.go`）：隔离库写入 `tenant_id=4242`，新 `DeployService` 重载队列后执行器收到原租户，行租户不回退、无 tenant_id=0 副本（真实 MySQL `-race` PASS）。
- 方法消费 policy pin 已提供 CI 来源；新增 Harness helper/checker 尚未发行。本次机制改进见 [Harness 任务包](../../pantheon-harness/docs/harness/tasks/2026-11-24-workflow-context-improvement.task.md)。

## 仍开放的验收边界

- T04 核心租户隔离已收口（completed）；详情路由绑定转 T18，写入路径与 multi 端到端转 T14/T19。11/24 追加的 `TestAsyncRecoveryKeepsOriginatingTenant` 固定“重启后重载任务仍用原租户”的恢复路径。
- T06/T07：NamespaceAll/读路由绑定、真实集群凭据轮换、断网/取消与 UI 草稿恢复仍按对应任务证据验收。
- T08：goroutine 释放、慢客户端背压、重连渲染证据。
- T09：旧完整 deploy 套件 1800 秒超时，不能记 PASS；相关定向测试结果只证明其范围，LiveVM 与完整 suite 由当前任务承接。
- T12：真实 ArgoCD provider、webhook 密钥和源仓库配置缺口保持显式；未配置不宣称验收。
- T16 已收口（completed），环境类残余（V4/V7/V8、Ingress、Velero、race Redis、ArgoCD 密钥、WebSocket 资源释放）转 T19；T17 仍负责全新安装/回滚与交付收口；最终维护者验收仍开放。
- 非必需外部 CI 红灯按原证据记录；不修改必过策略或使用 solo-override。
- 外部审查 2026-11-24 新增发现：O-01 已由[Compose 凭据外置与废弃 DDL 挂载移除](./tasks/2026-11-24-ops-compose-credentials/manifest.json)关闭——`docker-compose.yml` 两处口令改为 `${VAR:?}` 必填插值、移除 `system_init.sql` initdb 挂载；容器启动与首建库尚未实测（本机 Docker daemon 未运行）。O-02 `backend/.golangci.yml` 的 depguard 空转已在 Base 源码修复（Base 任务 `2026-11-24-golangci-depguard-boundary-paths`：三条 deny 路径改为真实 module 前缀 + 回归测试，探针证明 0 issues→1 issue），但该文件属 Base 跟踪、不在 `business-overlay.json` overlay 清单内，Ops 仍持过期副本；须经正式 foundation release 与 overlay 重建才会生效，不在下游手改。真实跨层门禁 `scripts/harness/check-boundaries.mjs` 一直生效（strict 0 findings），因此空转的是纵深防御层而非门禁本身。

## 下一步与恢复

从执行队列选择前置已满足的任务，读 manifest、最新 summary/review，核对当前文件后继续。需要环境时先验证候选身份、DB/Redis隔离、内存及 provider 可用性；超时先定位阻塞，不重复运行同一无变化尝试。

## 历史

- [完整原状态快照](./archive/2026-11/STATUS-before-workflow-improvement-20261124.md)：所有原结论、候选和检查点按原字节保留。
- [开工时原状态](./archive/2026-11/STATUS-at-workflow-intake-20261124.md)：保留最初备份；上方快照还包含整理期间追加的第八轮检查点，两份原文均保留。
- [ARCHIVE](./ARCHIVE.md)：完成/归档不代表当前产品和视觉验收。
