# Prompt X4 — D4 生产门禁 G2/G3（gate-prepared，生产窗口授予后才执行）

> 派发模型：由执行 Claude 通过 `codex exec ... -C pantheon-ops` 实施；本仓库 Claude 为 planner/reviewer。
> **纯生产/外部 gate**：需 DBA 授权的只读访问 + 生产维护窗口 + DBA/maintainer 签署。未获授予则保持 OPEN，不"转交后误报全完成"。

## 参考事实

`docs/runbooks/TENANT_MIGRATION_RUNBOOK.md` §8.1（2026-09-15 记录）：

- **G2 = OPEN**：无生产备份 RPO/RTO 确认或生产恢复演练证据；`tenant_rehearsal` 是一次性本地副本，**不满足 G2**。生产 DDL/数据变更禁止仍在。
- **G3 = OPEN**：`backend/cmd/tenantsizing` 从未对生产 DB 运行；演练表仅 1 行无生产行量；窗口不可估。早期某并行会话声称 G1-G4 授予已被 audited 并 DISCREDITED（agent-signed、非生产）并已回滚。

## 范围

- G2：对真实生产备份执行恢复演练，实测 RPO/RTO，把结果与 DBA+maintainer 签署写入 G2 runbook（`PRODUCTION_BACKUP_RESTORE_DRILL_G2.md` + §8.1）。
- G3：`tenantsizing snapshot`（只读）于生产 DB 或 G2 restore 目标 → `tenantsizing plan` 产窗口 workbook（行量 + 窗口假设）；>10M 行的表需生产量级预发布演练后才能放 G3。

## 精确文件所有权

- `backend/cmd/tenantsizing`（运行工具，只读调用；不改逻辑除非发现真实缺陷）
- `docs/runbooks/TENANT_MIGRATION_RUNBOOK.md`（§8.1 状态）
- `docs/runbooks/PRODUCTION_BACKUP_RESTORE_DRILL_G2.md`
- `.harness/{tasks,evidence}/` 对应 D4 关闭链

## 硬约束

- 只读生产动作；除 DBA 已批准并记录外不做任何 DDL/数据变更。
- **不代签**：DBA/maintainer 签署必须由人实际完成并留证据；本批不得伪造签署。
- 不提交/push/发布影响到生产；不把本地演练冒充 G2/G3。
- 不回显任何生产凭证。

## 真实验收

- G2/G3 各有一条记录：OPEN→完成（有签署证据）或保持 OPEN（注明阻塞原因 + 真实前置）。绝不未获窗口而报完成。
