# T08 部署任务恢复、协程与远程执行一致性 — 实施摘要

- Task: `2026-11-20-ops-deploy-recovery-and-remote-execution`
- 日期: 2026-11-24
- 状态: **恢复/所有权/panic 分支在真实 MySQL 上定向通过；完整套件本会话超时，未重跑 LiveVM**
- 交付层级: L2（自审见 `review.md`）

## 已交付

| 验收项 | 状态 | 证据 |
|---|---|---|
| worker panic 转为可见失败，不拖垮服务 | 已实现 | `runOwnedHost` 边界 `recover()` 落库 `business.deploy.executor.panic` + panic 文本；`TestExecutorPanicIsolatedAndSettledAsFailure` PASS |
| attempt/lease/CAS 与恢复/取消/重试 | 已实现 | 定向 124s 子集（`TestOwnership*`、`TestDeployHostLease*`、`TestWorker*`、`TestCancelTaskCAS`、幂等重放/陈旧冲突）全 PASS，真实 MySQL |
| SSH 取消/超时 | 单测级已覆盖 | `TestDeploySSHTransportCancelDuring{Handshake,Output,SessionOpen}`、`PreCanceledContextDoesNotOpenSession`、`CanceledConnectionPreservesCommand` PASS |
| 未知状态不盲目重试、Agent 任务不被 running 卡死 | 部分 | 所有权/取消测试覆盖；真实远程终止证据未在本候选重采 |

## 验证

- T08 定向子集：`ok ... 124.377s`（exit 0）
- **完整 deploy 套件**：`FAIL ... 1800.066s`，`panic: test timed out after 30m0s`；告警时运行中的用例为 `TestFrozenRollbackPreservesDefinitionAndOwnReview`（6s，阻塞在 MySQL `ExecContext` I/O）。同套件在 T03 候选上曾 **PASS 1047.796s**；本次同窗口前端 dev server 因低内存被 harness 回收，判定为环境/慢 fixture 超时，**不记为 PASS**。

## 显式 gap

1. 完整 deploy 套件本会话未跑完（内存压力下 30m 超时）。
2. `TestDeployLiveVM*` 未纳入定向运行；真实 SSH 远端终止证据沿用早期轮次，未在本候选重采。
3. deploy 浏览器/API smoke 未执行（前端 dev server 被内存回收）。
