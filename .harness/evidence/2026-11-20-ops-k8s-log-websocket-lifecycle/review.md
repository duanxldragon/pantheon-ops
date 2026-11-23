# T09 Review — Pod 日志 WebSocket 来源与连接生命周期

## 结论

`approved with documented P2 follow-up`。并发写 panic（日志流 + ping ticker）已修复并由回归锁定；Origin 校验与 context 取消链路成立。生命周期/背压/重连的运行时证据仍缺，不代签完整验收。

## Findings

1. **P1 fixed — 并发写同一 websocket 连接会 panic。**
   gorilla/websocket 要求单写者；ping ticker 与日志流并发 `WriteMessage`。修复为 `logWriter` 互斥 + 写超时；回归先红（无 mutex 时 panic）后绿。

2. **P2 — goroutine 基线未采集。**
   页面关闭后 provider goroutine 是否归零没有运行时对比证据。

3. **P2 — 慢客户端背压未演练。**
   当前逐帧写入 + 写超时，无有界队列；慢消费者会阻塞到写超时。

4. **P2 — provider 错误后重连的渲染证据缺失。**

## Machine Readable

```json
{
  "taskId": "2026-11-20-ops-k8s-log-websocket-lifecycle",
  "reviewer": "primary agent self-review (no independent review on this candidate)",
  "verdict": "approved with documented P2 follow-up",
  "decision": "approved with documented P2 follow-up",
  "scope": "websocket origin allowlist, request-context cancellation, single-writer log protocol",
  "taskManifest": ".harness/tasks/2026-11-20-ops-k8s-log-websocket-lifecycle/manifest.json",
  "evidenceDir": ".harness/evidence/2026-11-20-ops-k8s-log-websocket-lifecycle",
  "summaryFile": ".harness/evidence/2026-11-20-ops-k8s-log-websocket-lifecycle/summary.md",
  "reviewFile": ".harness/evidence/2026-11-20-ops-k8s-log-websocket-lifecycle/review.md",
  "changeRef": "none (uncommitted worktree)",
  "findings": [
    {"id": "T09-R1", "severity": "P1", "status": "fixed", "summary": "concurrent websocket writers panic", "verification": "TestLogWriterSerializesConcurrentFrames PASS; fails without mutex"},
    {"id": "T09-R2", "severity": "P2", "status": "accepted-gap", "summary": "no runtime goroutine baseline after tab close"},
    {"id": "T09-R3", "severity": "P2", "status": "accepted-gap", "summary": "slow-client backpressure not exercised"},
    {"id": "T09-R4", "severity": "P2", "status": "accepted-gap", "summary": "reconnect rendered evidence missing"}
  ],
  "verification": {
    "targetedLogTests": "4/4 PASS",
    "workloadPackage": "ok",
    "k8sModuleTests": "11/11 packages ok"
  },
  "linkage": {
    "taskPacket": "docs/harness/tasks/2026-11-20-ops-k8s-log-websocket-lifecycle.task.md",
    "evidence": ".harness/evidence/2026-11-20-ops-k8s-log-websocket-lifecycle/commands.json",
    "reviewFile": ".harness/evidence/2026-11-20-ops-k8s-log-websocket-lifecycle/review.md",
    "changeRef": "none",
    "planRefs": [".harness/tasks/2026-11-20-ops-review-remediation/plan.md"]
  }
}
```
