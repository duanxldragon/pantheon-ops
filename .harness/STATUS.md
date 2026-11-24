# Pantheon Ops 当前状态

本页是当前重点与验收 gap 的入口；状态和计数以 [TASK_INDEX](./TASK_INDEX.md) 及各包 manifest 为准。进度追加到目标任务 summary，历史检查点单独保存。

## 当前工作

- 当前执行主线：[审查整改父计划](./tasks/2026-11-20-ops-review-remediation/plan.md) / [执行队列](./tasks/2026-11-20-ops-review-remediation/EXECUTION_QUEUE.md)。原十包未验收 residual 由本主线承接。
- Base 消费版本及 commit 以 [foundation lock](../foundation-release.lock.json) 为准；产品 VERSION、消费版本、实际运行候选分别记录。
- 11/24 检查点记录 PR #205 合并到 `5e4dac4f`，随后第八轮记录 T06 读授权/路由的代码级收口及其残余 gap；这些结论只证明原候选。当前存在未提交 K8s 等业务改动，须按当前候选重新验证受影响范围。
- 方法消费 policy pin 已提供 CI 来源；新增 Harness helper/checker 尚未发行。本次机制改进见 [Harness 任务包](../../pantheon-harness/docs/harness/tasks/2026-11-24-workflow-context-improvement.task.md)。

## 仍开放的验收边界

- T04 核心租户隔离：旧检查点记录 Base keyMatch2 详情路由阻塞；后续修改必须在当前候选核实，compat/multi live 不因迁移测试通过而关闭。
- T06/T07：NamespaceAll/读路由绑定、真实集群凭据轮换、断网/取消与 UI 草稿恢复仍按对应任务证据验收。
- T08：goroutine 释放、慢客户端背压、重连渲染证据。
- T09：旧完整 deploy 套件 1800 秒超时，不能记 PASS；相关定向测试结果只证明其范围，LiveVM 与完整 suite 由当前任务承接。
- T12：真实 ArgoCD provider、webhook 密钥和源仓库配置缺口保持显式；未配置不宣称验收。
- T16/T17：受限角色 SSH、双租户、浏览器、SA 轮换、Ingress、全新安装/回滚、WebSocket、Velero schedule/restore、完整 race 和最终维护者验收，以同候选 evidence 为准。
- 非必需外部 CI 红灯按原证据记录；不修改必过策略或使用 solo-override。

## 下一步与恢复

从执行队列选择前置已满足的任务，读 manifest、最新 summary/review，核对当前文件后继续。需要环境时先验证候选身份、DB/Redis隔离、内存及 provider 可用性；超时先定位阻塞，不重复运行同一无变化尝试。

## 历史

- [完整原状态快照](./archive/2026-11/STATUS-before-workflow-improvement-20261124.md)：所有原结论、候选和检查点按原字节保留。
- [开工时原状态](./archive/2026-11/STATUS-at-workflow-intake-20261124.md)：保留最初备份；上方快照还包含整理期间追加的第八轮检查点，两份原文均保留。
- [ARCHIVE](./ARCHIVE.md)：完成/归档不代表当前产品和视觉验收。
