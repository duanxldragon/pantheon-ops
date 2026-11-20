# 执行队列

唯一计划：[plan.md](./plan.md)。父包未开始执行；10 子包均 planned。

| 编号 | 工作包                                                                                                               | 优先级/档位 | 依赖                           | 状态    | 退出口径 |
| ---- | -------------------------------------------------------------------------------------------------------------------- | ----------- | ------------------------------ | ------- | -------- |
| C01  | [部署与服务实例关联及引用保护](../../../docs/harness/tasks/2026-11-16-ops-service-deploy-integrity.task.md)          | P0 / L2     | 无                             | planned | 核心必交 |
| C02  | [页面服务实例到部署结果闭环](../../../docs/harness/tasks/2026-11-16-ops-service-deploy-user-flow.task.md)            | P1 / L2     | C01                            | planned | 核心必交 |
| C03  | [服务目录维护、分页与受限角色](../../../docs/harness/tasks/2026-11-16-ops-service-catalog-usability.task.md)         | P1 / L2     | 无                             | planned | 核心必交 |
| C04  | [Kubernetes Service 编辑与并发保护](../../../docs/harness/tasks/2026-11-16-ops-k8s-service-mutations.task.md)        | P1 / L2     | 无                             | planned | 核心必交 |
| C05  | [当前版本数据库、执行与业务视觉验收](../../../docs/harness/tasks/2026-11-16-ops-business-runtime-acceptance.task.md) | P0 / L2     | C01, C02, C03, C04             | planned | 核心必交 |
| C06  | [试点交付包、安装回滚与状态收口](../../../docs/harness/tasks/2026-11-16-ops-pilot-release-closeout.task.md)          | P0 / L2     | C05                            | planned | 核心必交 |
| E01  | [文件分发创建入口与可靠控制](../../../docs/harness/tasks/2026-11-16-ops-filedist-product-closure.task.md)            | P1 / L2     | 无                             | planned | 条件扩展 |
| E02  | [ArgoCD 归属隔离与真实状态收敛](../../../docs/harness/tasks/2026-11-16-ops-argocd-ownership-convergence.task.md)     | P0 / L2     | 无                             | planned | 条件扩展 |
| E03  | [Helm 能力口径与真实 Chart 安装边界](../../../docs/harness/tasks/2026-11-16-ops-helm-capability-truth.task.md)       | P1 / L2     | 无                             | planned | 条件扩展 |
| E04  | [已启用扩展真实外部验收](../../../docs/harness/tasks/2026-11-16-ops-extensions-live-acceptance.task.md)              | P1 / L2     | 按开放能力条件依赖 E01/E02/E03 | planned | 条件扩展 |

C01/C03/C04 无包间依赖；C01→C02；C01-C04→C05→C06。ServiceCatalog 写入 lane 必须串行。扩展默认不计核心退出；实际已装配开放，交付前必须验证所选能力或实施可验证裁剪。

E04 无无条件依赖；按 manifest.conditionalDependencies 与实际开放能力清单逐项判定。C06 必须验收所有开放扩展，裁剪也要留证。
