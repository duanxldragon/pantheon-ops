# 部署任务恢复、协程与远程执行一致性

- Status: in-progress
- Updated: 2026-11-24
- Baseline: a2d69d4a759b5df2b26aaa9d26bdc5175babe9dc
- Owner: 部署并发与恢复负责人
- Priority: P1
- Activation: core
- Dependencies: T02, T03, T04

## Implemented

- worker panic 边界：`runOwnedHost` 捕获 executor panic，落库 failed attempt（`business.deploy.executor.panic` + panic 文本），不拖垮进程。
- attempt/lease/CAS、worker claim/handoff/recovery、取消与幂等重放路径已实装（真实 MySQL 定向验证）。
- SSH transport 取消：握手/会话打开/输出阶段均有取消测试。

## Verification

- 定向子集（panic/ownership/lease/worker/SSH cancel/CAS/idempotent）：`ok ... 124.377s`（exit 0）。
- **完整 deploy 套件本会话超时**：`FAIL ... 1800.066s`，`panic: test timed out after 30m0s`；告警时运行中的 `TestFrozenRollbackPreservesDefinitionAndOwnReview`（6s）阻塞在 MySQL `ExecContext`。同套件曾在 T03 候选 `PASS 1047.796s`；本次同窗口前端 dev server 因低内存被 harness 回收 → 记为环境超时，不记 PASS。
- 证据：`../evidence/2026-11-20-ops-deploy-recovery-and-remote-execution/`。

## Known Gaps

1. 完整 deploy 套件未在本会话跑完（内存压力下 30m 超时）。
2. `TestDeployLiveVM*` 未纳入定向运行；真实远端终止证据未在本候选重采。
3. deploy 浏览器/API smoke 未执行（前端 dev server 被内存回收）。

## Next Atomic Action

在低负载时段重跑完整 deploy 套件与 LiveVM 用例，补齐远端终止与 smoke 证据后复核关闭。
