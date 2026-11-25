# Base 发版后恢复业务详情路由租户绑定

先读[父计划](../2026-11-20-ops-review-remediation/plan.md)、[本包 task packet](../../../docs/harness/tasks/2026-11-25-ops-tenant-detail-route-binding.task.md)、[manifest](./manifest.json)与[状态](./STATUS.md)。

本包承接 T04 关闭后的详情路由残余。开工时先核实 Base 是否已发布携带 `63940437`（#525 静态/参数路由隔离）的正式 foundation release：`pantheon-base/VERSION`、最新 tag 与 `origin/main` 的关系。未发布时保持 planned，不手抄 Base 文件、不 override 匹配器。

已发布时按正式流程更新 `foundation-release.lock.json` 并重建 Ops，然后在声明的业务文件内恢复详情路由的租户作用域绑定，补双租户反例（跨租户详情/导出/回调被拒，正控 200）。

执行验证时创建 `.harness/evidence/2026-11-25-ops-tenant-detail-route-binding/commands.json`、`summary.md`、`review.md`，并同步 manifest 与 task packet 的 Review File。

缺 release/环境记录实际 gap。安全可逆步骤自主完成，维护者负责范围偏好、门禁策略与最终功能验收。
