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
- **完整 deploy 套件 PASS**：`go test -count=1 -buildvcs=false -p 1 -timeout 30m ./modules/business/deploy/...` → `ok ... 1342.649s`（22m22s，真实 MySQL，`main` `3e83ae7a`）。此前超时与 `staleReport` 失败根因是 DSN 需 `loc=Local`（MySQL SYSTEM +08 vs 驱动默认 UTC，`databaseNow` 超前 8 小时）。
- deploy 浏览器/API smoke 由 PR #207 CI Smoke Sanity（17m30s）覆盖并绿。
- 证据：`../evidence/2026-11-20-ops-deploy-recovery-and-remote-execution/`。

## Known Gaps

1. `TestDeployLiveVM*` 因验收主机/凭证不可用而 skip；真实远端终止证据未在本候选重采。

## Next Atomic Action

提供验收主机与 SSH 凭证后重跑 `TestDeployLiveVM*`，补齐真实远端终止证据后进入最终验收。
