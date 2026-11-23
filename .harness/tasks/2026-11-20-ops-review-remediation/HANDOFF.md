# 11月20日审查整改与真实交付任务树

先读[父计划](./plan.md)、[本包task packet](../../../docs/harness/tasks/2026-11-20-ops-review-remediation.task.md)、[manifest](./manifest.json)与[状态](./STATUS.md)。

本包负责人为业务交付协调负责人。开工时记录实际commit、未提交改动与前置合同，核实来源发现是否仍存在，按task packet的实施步骤、验收与验证执行。

依赖为 无。激活条件为 `core`。共享文件冲突按父计划串行交接。

下一步先核实 SEC-001~008、DEP-001~007、ARC-001、NEW-01~05、R-01~16、V1~V8、权限/CI/文档补审 的当前实现与历史证据，记录已完成实现及剩余验收。

执行验证时创建 `.harness/evidence/2026-11-20-ops-review-remediation/commands.json`、`summary.md`、真实`review.md`，并同步manifest和task packet的Review File。

缺环境/provider/凭据记录实际gap。安全可逆步骤自主完成，维护者负责范围偏好、门禁策略与最终功能/视觉验收。
