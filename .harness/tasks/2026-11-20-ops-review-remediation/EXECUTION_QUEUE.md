# 审查任务执行队列

所有任务初始状态为planned。先读[父计划](./plan.md)与[FINDINGS映射](./FINDINGS.json)，实际执行状态以每包manifest为准。

| 包  | 任务                                                                                                      | 前置                                                  | 激活             |
| --- | --------------------------------------------------------------------------------------------------------- | ----------------------------------------------------- | ---------------- |
| T01 | [当前基线核实与业务边界决策](../2026-11-20-ops-review-baseline-and-decisions/HANDOFF.md)                  | 无                                                    | core             |
| T02 | [固定部署模板参数与卸载路径安全](../2026-11-20-ops-deploy-fixed-template-security/HANDOFF.md)             | 无                                                    | core             |
| T03 | [业务权限绑定与部署任务数据可见性](../2026-11-20-ops-business-permissions-and-task-visibility/HANDOFF.md) | T01                                                   | core             |
| T04 | [核心业务资源租户归属与兼容迁移](../2026-11-20-ops-core-business-tenant-isolation/HANDOFF.md)             | T01                                                   | core             |
| T05 | [部署凭据资源授权与管理页面](../2026-11-20-ops-deploy-credential-ownership-and-ui/HANDOFF.md)             | T03、T04                                              | core             |
| T06 | [K8s 资源读写授权与变更准入](../2026-11-20-ops-k8s-resource-authorization-and-mutations/HANDOFF.md)       | T03、T04                                              | core             |
| T07 | [K8s 凭据导入、连接轮换与请求边界](../2026-11-20-ops-k8s-client-lifecycle-and-config-safety/HANDOFF.md)   | T01、T04                                              | core             |
| T08 | [Pod 日志 WebSocket 来源与连接生命周期](../2026-11-20-ops-k8s-log-websocket-lifecycle/HANDOFF.md)         | T06、T07                                              | core             |
| T09 | [部署任务恢复、协程与远程执行一致性](../2026-11-20-ops-deploy-recovery-and-remote-execution/HANDOFF.md)   | T02、T03、T04                                         | core             |
| T10 | [文件分发队列、恢复与取消](../2026-11-20-ops-filedist-durable-lifecycle/HANDOFF.md)                       | T03、T14                                              | filedist-enabled |
| T11 | [服务目录与实例部署关联闭环](../2026-11-20-ops-service-deploy-business-closure/HANDOFF.md)                | T03、T04、T05、T06、T09                               | core             |
| T12 | [ArgoCD 归属、Webhook 与状态收敛](../2026-11-20-ops-argocd-ownership-and-webhook/HANDOFF.md)              | T03、T14                                              | argocd-enabled   |
| T13 | [模板编辑与回滚追溯体验](../2026-11-20-ops-deploy-template-and-task-experience/HANDOFF.md)                | T05、T08、T09、T11                                    | after-core-p1    |
| T14 | [扩展业务与异步链路租户隔离](../2026-11-20-ops-extension-and-async-tenant-isolation/HANDOFF.md)           | T03、T04                                              | core             |
| T15 | [能力文档、测试资产与治理对账](../2026-11-20-ops-review-docs-and-test-governance/HANDOFF.md)              | T01、T03                                              | core             |
| T16 | [同候选真实环境与受限角色验收](../2026-11-20-ops-review-live-runtime-acceptance/HANDOFF.md)               | T02、T03、T04、T05、T06、T07、T08、T09、T11、T14、T15 | core             |
| T17 | [试点交付包、安装回滚与审查收口](../2026-11-20-ops-review-delivery-and-closeout/HANDOFF.md)               | T16                                                   | core             |
| T18 | [Base 发版后恢复业务详情路由租户绑定](../2026-11-25-ops-tenant-detail-route-binding/HANDOFF.md)           | Base 正式 release（#525）；T04 已收口                 | core             |
| T19 | [真实环境残余验收](../2026-11-25-ops-residual-live-acceptance/HANDOFF.md)                                 | T05、T11、T12、T13、T14                               | core             |

T01和T02无前置包。其余按依赖推进，共同文件按父计划交接。T16条件依赖根据已确认开放清单和本候选范围计算，T17等待真实验收完成。

维护者三触点为需求/范围、门禁策略、最终功能/视觉验收。本批拆包完成后由任务执行阶段继续这些工作。
