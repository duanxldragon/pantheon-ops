# 当前任务索引

由 `.harness/tasks/*/manifest.json` 生成；仅展示当前任务树。状态以 manifest 为准，completed 不代表产品或生产验收。

planned: 8 · in-progress: 6 · completed: 17 · abandoned: 0（当前树共 31 项；不统计月度归档）

| 当前任务 | 状态 | 优先级 | 最新证据 |
| --- | --- | --- | --- |
| [ArgoCD 归属、Webhook 与状态收敛](./tasks/2026-11-20-ops-argocd-ownership-and-webhook/manifest.json) | planned | P1 | 未提供 summary |
| [核心业务资源租户归属与兼容迁移](./tasks/2026-11-20-ops-core-business-tenant-isolation/manifest.json) | in-progress | P1 | [summary](./evidence/2026-11-20-ops-core-business-tenant-isolation/summary.md) |
| [部署凭据资源授权与管理页面](./tasks/2026-11-20-ops-deploy-credential-ownership-and-ui/manifest.json) | planned | P1 | 未提供 summary |
| [模板编辑与回滚追溯体验](./tasks/2026-11-20-ops-deploy-template-and-task-experience/manifest.json) | planned | P2 | 未提供 summary |
| [扩展业务与异步链路租户隔离](./tasks/2026-11-20-ops-extension-and-async-tenant-isolation/manifest.json) | planned | P1 | 未提供 summary |
| [文件分发队列、恢复与取消](./tasks/2026-11-20-ops-filedist-durable-lifecycle/manifest.json) | planned | P1 | 未提供 summary |
| [K8s 凭据导入、连接轮换与请求边界](./tasks/2026-11-20-ops-k8s-client-lifecycle-and-config-safety/manifest.json) | in-progress | P1 | [summary](./evidence/2026-11-20-ops-k8s-client-lifecycle-and-config-safety/summary.md) |
| [Pod 日志 WebSocket 来源与连接生命周期](./tasks/2026-11-20-ops-k8s-log-websocket-lifecycle/manifest.json) | in-progress | P1 | [summary](./evidence/2026-11-20-ops-k8s-log-websocket-lifecycle/summary.md) |
| [K8s 资源读写授权与变更准入](./tasks/2026-11-20-ops-k8s-resource-authorization-and-mutations/manifest.json) | in-progress | P1 | [summary](./evidence/2026-11-20-ops-k8s-resource-authorization-and-mutations/summary.md) |
| [试点交付包、安装回滚与审查收口](./tasks/2026-11-20-ops-review-delivery-and-closeout/manifest.json) | planned | P0 | 未提供 summary |
| [能力文档、测试资产与治理对账](./tasks/2026-11-20-ops-review-docs-and-test-governance/manifest.json) | planned | P2 | 未提供 summary |
| [同候选真实环境与受限角色验收](./tasks/2026-11-20-ops-review-live-runtime-acceptance/manifest.json) | in-progress | P0 | [summary](./evidence/2026-11-20-ops-review-live-runtime-acceptance/summary.md) |
| [11月20日审查整改与真实交付任务树](./tasks/2026-11-20-ops-review-remediation/manifest.json) | in-progress | P0 | 未提供 summary |
| [服务目录与实例部署关联闭环](./tasks/2026-11-20-ops-service-deploy-business-closure/manifest.json) | planned | P0 | 未提供 summary |

按目标任务读取 manifest、task packet 和最新 summary；历史入口为 [ARCHIVE](./ARCHIVE.md)。
