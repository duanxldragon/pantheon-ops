# 扩展业务租户归属 Schema（T14 第一阶段）

先读[父计划](../2026-11-20-ops-review-remediation/plan.md)、[本包task packet](../../../docs/harness/tasks/2026-11-20-ops-extension-and-async-tenant-isolation.task.md)、[manifest](./manifest.json)与[状态](./STATUS.md)。

本包负责人为扩展业务租户迁移负责人。开工时记录实际commit、未提交改动与前置合同，核实来源发现是否仍存在，按task packet的实施步骤、验收与验证执行。

依赖为 T03、T04。激活条件为 `core`。共享文件冲突按父计划串行交接。

第一阶段已完成并归档：18 张扩展表 tenant schema 与五个租户优先唯一键已合并、真实 MySQL/托管 CI 已验证、独立审查通过。运行时隔离不属于本阶段；后续按 [T14 第二阶段计划](../2026-11-25-ops-tenant-runtime-enforcement/PLAN.md) 的模块顺序执行，不得据本包状态宣称租户运行时隔离完成。

证据见 `.harness/evidence/2026-11-20-ops-extension-and-async-tenant-isolation/commands.json`、`summary.md` 与 `review.md`。

缺环境/provider/凭据记录实际gap。安全可逆步骤自主完成，维护者负责范围偏好、门禁策略与最终功能/视觉验收。
