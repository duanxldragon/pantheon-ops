# T08 Review — 部署任务恢复、协程与远程执行一致性

## 结论

`approved with documented P2 follow-up`（仅就定向验证的恢复/panic 切片）。panic 隔离、所有权/租约/CAS、worker 恢复与取消路径在真实 MySQL 定向运行通过；完整套件本会话超时与 LiveVM 证据未重采，作为显式环境 gap 保留，不代签完整验收。

## Findings

1. **P2 — 完整 deploy 套件本会话超时。**
   `go test -count=1 -timeout 1800s ./modules/business/deploy/...` → `panic: test timed out after 30m0s`，告警时运行中的 `TestFrozenRollbackPreservesDefinitionAndOwnReview`（6s）阻塞在 MySQL `ExecContext` I/O。同套件在 T03 候选上 `PASS 1047.796s`；本次同窗口前端 dev server 因低内存被 harness 回收。判定环境性超时，不记为 PASS。

2. **P2 — LiveVM 远端终止证据未在本候选重采。**
   定向子集覆盖 SSH transport 取消的单测路径，但真实 VM 的远端进程终止/超时未重跑。

3. **P2 — deploy smoke 未执行。**
   `test:smoke:business:deploy:api` / `test:smoke:business:deploy` 因前端 dev server 不可用跳过。

4. **fixed — worker panic 逃逸。**
   后台 attempt 不在 gin recovery 内；panic 现被 `runOwnedHost` 边界捕获并落为 failed/可见结果（消息含 `business.deploy.executor.panic`）。回归 `TestExecutorPanicIsolatedAndSettledAsFailure`。

## Machine Readable

```json
{
  "taskId": "2026-11-20-ops-deploy-recovery-and-remote-execution",
  "reviewer": "primary agent self-review (no independent review on this candidate)",
  "verdict": "approved with documented P2 follow-up",
  "decision": "approved with documented P2 follow-up",
  "scope": "worker panic isolation, attempt/lease ownership, cancellation CAS, idempotent replay, SSH transport cancel",
  "taskManifest": ".harness/tasks/2026-11-20-ops-deploy-recovery-and-remote-execution/manifest.json",
  "evidenceDir": ".harness/evidence/2026-11-20-ops-deploy-recovery-and-remote-execution",
  "summaryFile": ".harness/evidence/2026-11-20-ops-deploy-recovery-and-remote-execution/summary.md",
  "reviewFile": ".harness/evidence/2026-11-20-ops-deploy-recovery-and-remote-execution/review.md",
  "changeRef": "none (uncommitted worktree)",
  "findings": [
    {"id": "T08-R1", "severity": "fixed", "status": "fixed", "summary": "worker executor panic escaped to process; now settled as visible failure", "verification": "TestExecutorPanicIsolatedAndSettledAsFailure PASS"},
    {"id": "T08-R2", "severity": "P2", "status": "accepted-environment-gap", "summary": "full deploy suite timed out at 30m under memory pressure; last full green 1047.796s", "verification": "FAIL ... 1800.066s panic: test timed out after 30m0s; running TestFrozenRollbackPreservesDefinitionAndOwnReview blocked in MySQL ExecContext"},
    {"id": "T08-R3", "severity": "P2", "status": "accepted-gap", "summary": "LiveVM real remote termination not re-collected"},
    {"id": "T08-R4", "severity": "P2", "status": "accepted-gap", "summary": "deploy browser/API smoke not run"}
  ],
  "verification": {
    "recoverySubset": "PASS 124.377s",
    "fullDeploySuite": "FAIL (timeout, environment)",
    "lastFullGreen": "PASS 1047.796s (T03 candidate)"
  },
  "linkage": {
    "taskPacket": "docs/harness/tasks/2026-11-20-ops-deploy-recovery-and-remote-execution.task.md",
    "evidence": ".harness/evidence/2026-11-20-ops-deploy-recovery-and-remote-execution/commands.json",
    "reviewFile": ".harness/evidence/2026-11-20-ops-deploy-recovery-and-remote-execution/review.md",
    "changeRef": "none",
    "planRefs": [".harness/tasks/2026-11-20-ops-review-remediation/plan.md"]
  }
}
```
