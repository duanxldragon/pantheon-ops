# ArgoCD 归属、Webhook 与状态收敛

先读[父计划](../2026-11-20-ops-review-remediation/plan.md)、[本包task packet](../../../docs/harness/tasks/2026-11-20-ops-argocd-ownership-and-webhook.task.md)、[manifest](./manifest.json)与[状态](./STATUS.md)。

本包负责人为ArgoCD 集成负责人。开工时记录实际commit、未提交改动与前置合同，核实来源发现是否仍存在，按task packet的实施步骤、验收与验证执行。

依赖为 T03、T14。激活条件为 `argocd-enabled`。共享文件冲突按父计划串行交接。

下一步先核实 R-16 的当前实现与历史证据，记录已完成实现及剩余验收。

执行验证时创建 `.harness/evidence/2026-11-20-ops-argocd-ownership-and-webhook/commands.json`、`summary.md`、真实`review.md`，并同步manifest和task packet的Review File。

缺环境/provider/凭据记录实际gap。安全可逆步骤自主完成，维护者负责范围偏好、门禁策略与最终功能/视觉验收。
