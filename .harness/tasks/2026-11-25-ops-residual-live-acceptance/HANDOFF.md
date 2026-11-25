# 真实环境残余验收（多租户、受限角色、SA 轮换、Ingress、Velero、race Redis）

先读[父计划](../2026-11-20-ops-review-remediation/plan.md)、[本包 task packet](../../../docs/harness/tasks/2026-11-25-ops-residual-live-acceptance.task.md)、[manifest](./manifest.json)与[状态](./STATUS.md)。

本包承接 T16 关闭后的环境类残余：V4 双租户、V7 受限 SSH 用户、V8 SA 轮换、Ingress provider 变更/快照、Velero schedule/restore、完整 race（测试 Redis 26316）、`GAP-ARGOCD-ENVKEY`、WebSocket server 侧资源释放。T16 已完成部分及其证据仍按其归档回执阅读，不重复执行。

复用 T16 的探针与 lane 脚本；每项记录候选 commit、二进制哈希、环境身份（DB 名、provider 版本）与真实退出状态。环境不可得的项写 gap 并继续其他项。

执行验证时创建 `.harness/evidence/2026-11-25-ops-residual-live-acceptance/commands.json`、`summary.md`、`review.md`，并同步 manifest 与 task packet 的 Review File。

缺环境/凭据记录实际 gap。安全可逆步骤自主完成，维护者负责环境提供、门禁策略与最终功能验收。
