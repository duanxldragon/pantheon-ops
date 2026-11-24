# T08 Review — 部署任务恢复、协程与远程执行一致性

## 结论

`approved with documented P2 follow-up`。panic 隔离、所有权/租约/CAS、worker 恢复与取消路径在真实 MySQL 完整套件 PASS；deploy 浏览器/API smoke 由 CI 绿；LiveVM 真实远端终止因验收凭证不可用保留为显式 gap，不代签完整验收。

## Findings

1. **fixed — 完整 deploy 套件未通过。**
   先前 `timeout after 30m` 与定向 `staleReport` 均定位为 DSN 时区配置（MySQL SYSTEM +08 vs go-sql-driver 默认 loc=UTC，`databaseNow` 超前 8 小时）。DSN 加 `loc=Local` 后 `go test -count=1 -buildvcs=false -p 1 -timeout 30m ./modules/business/deploy/...` → `PASS 1342.649s`。

2. **P2 — LiveVM 远端终止证据未在本候选重采。**
   定向子集覆盖 SSH transport 取消的单测路径，但真实 VM 的远端进程终止/超时未重跑（验收主机与凭证未在本会话提供）。

3. **fixed — deploy smoke 未执行。**
   PR #207 CI Smoke Sanity 17m30s 绿，执行 `test:smoke:business:deploy:api` 与 `test:smoke:business:deploy`。

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
    {"id": "T08-R2", "severity": "P2", "status": "fixed", "summary": "full deploy suite now green after DSN loc=Local (MySQL SYSTEM +08 vs driver loc=UTC)", "verification": "PASS 1342.649s on main 3e83ae7a"},
    {"id": "T08-R3", "severity": "P2", "status": "accepted-gap", "summary": "LiveVM real remote termination not re-collected (acceptance host/credentials unavailable)"},
    {"id": "T08-R4", "severity": "P2", "status": "fixed", "summary": "deploy browser/API smoke green in CI", "verification": "PR #207 Smoke Sanity 17m30s PASS (run 45308078162)"}
  ],
  "verification": {
    "recoverySubset": "PASS 124.377s",
    "fullDeploySuite": "PASS 1342.649s",
    "deploySmoke": "CI Smoke Sanity PASS 17m30s (deploy api + deploy UI + cmdb/service/k8s)"
  },
  "linkage": {
    "taskManifest": ".harness/tasks/2026-11-20-ops-deploy-recovery-and-remote-execution/manifest.json",
    "taskPacket": "docs/harness/tasks/2026-11-20-ops-deploy-recovery-and-remote-execution.task.md",
    "evidence": ".harness/evidence/2026-11-20-ops-deploy-recovery-and-remote-execution/commands.json",
    "reviewFile": ".harness/evidence/2026-11-20-ops-deploy-recovery-and-remote-execution/review.md",
    "changeRef": "none",
    "planRefs": [".harness/tasks/2026-11-20-ops-review-remediation/plan.md"]
  }
}
```
