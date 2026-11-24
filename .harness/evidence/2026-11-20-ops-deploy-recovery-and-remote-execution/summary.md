# T08 部署任务恢复、协程与远程执行一致性 — 实施摘要

- Task: `2026-11-20-ops-deploy-recovery-and-remote-execution`
- 日期: 2026-11-24
- 状态: **完整 deploy 套件（含恢复/所有权/panic/取消）在真实 MySQL 上 PASS，CI deploy smoke 绿；LiveVM 远端终止为显式 gap**
- 交付层级: L2（自审见 `review.md`）

## 已交付

| 验收项 | 状态 | 证据 |
|---|---|---|
| worker panic 转为可见失败，不拖垮服务 | 已实现 | `runOwnedHost` 边界 `recover()` 落库 `business.deploy.executor.panic` + panic 文本；`TestExecutorPanicIsolatedAndSettledAsFailure` PASS |
| attempt/lease/CAS 与恢复/取消/重试 | 已实现 | 定向 124s 子集（`TestOwnership*`、`TestDeployHostLease*`、`TestWorker*`、`TestCancelTaskCAS`、幂等重放/陈旧冲突）全 PASS，真实 MySQL |
| SSH 取消/超时 | 单测级已覆盖 | `TestDeploySSHTransportCancelDuring{Handshake,Output,SessionOpen}`、`PreCanceledContextDoesNotOpenSession`、`CanceledConnectionPreservesCommand` PASS |
| 未知状态不盲目重试、Agent 任务不被 running 卡死 | 部分 | 所有权/取消测试覆盖；真实远程终止证据未在本候选重采 |

## 验证

- 定向子集：`ok ... 124.377s`（exit 0）。
- **完整 deploy 套件**：在 `main` `3e83ae7a` 上 `PASS 1342.649s`（22m22s，真实 MySQL）。此前 30m 超时与单测 `staleReport` 失败均为 DSN 配置：本地 MySQL 会话时区为 SYSTEM(+08)，而 go-sql-driver 默认 `loc=UTC`，`databaseNow()` 读到超前 8 小时、所有 Go 侧租约被判过期；DSN 加 `loc=Local` 后同套件全绿（历史 1047s PASS 同一约定）。
- **CI deploy smoke**：PR #207 Smoke Sanity（run 45308078162）17m30s 绿，执行 `test:smoke:business:deploy:api` 与 `test:smoke:business:deploy` 以及 cmdb/service/k8s smoke。

## 显式 gap

1. `TestDeployLiveVM*` 因 `PANTHEON_ACCEPTANCE_HOST` 未设置且本会话无可用的验收 SSH 凭证而 skip；真实远端终止证据沿用早期轮次，未在本候选重采。
2. 完整套件本地耗时 ~22 分钟，CI Backend Tests 不覆盖同等深度的 DB 重负载。
