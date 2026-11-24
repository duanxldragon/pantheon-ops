# T09 Review — Pod 日志 WebSocket 来源与连接生命周期

## 结论

`approved with documented P2 follow-up`。并发写 panic（日志流 + ping ticker）已修复并由回归锁定；Origin 校验、context 取消链路与写超时背压上界成立；真 K3s 的 k12 Pod 日志 WebSocket 握手/首帧/关闭 PASS。浏览器探针补齐了 goroutine 基线与重连渲染，两个原 P2 gap 关闭；CI smoke 页面覆盖仍为显式 gap，不代签完整验收。

## Findings

1. **P1 fixed — 并发写同一 websocket 连接会 panic。**
   gorilla/websocket 要求单写者；ping ticker 与日志流并发 `WriteMessage`。修复为 `logWriter` 互斥 + 写超时；回归先红（无 mutex 时 panic）后绿。

2. **P2 fixed — goroutine 基线未采集。**
   `omx-t07-browser-20261124-143959`：35 基线 / 39 持流 / 关闭抽屉后 35 / 恢复后关闭 35 / 终值 34，关闭后回到基线。

3. **fixed — P2 慢客户端背压上界未验证。**
   设计无有界队列，依靠同步写 + `logWriteWait` 写超时把慢消费者限制在固定时间内；`TestLogWriterStalledConsumerBoundedByDeadline`（不读取对端，1 MB 帧循环，100 ms 测试超时）在第 3 MB 触发超时并使流返回。

4. **P2 fixed — provider 错误后重连的渲染证据缺失。**
   缩容到 0 后抽屉渲染 `已断开`；点 `重连` 渲染 `failed to open pod log stream: pods "…" not found`；扩回 1 后新 Pod 重新 `已连接`。

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
    {
      "id": "T09-R1",
      "severity": "P1",
      "status": "fixed",
      "summary": "concurrent websocket writers panic",
      "verification": "TestLogWriterSerializesConcurrentFrames PASS; fails without mutex"
    },
    {
      "id": "T09-R2",
      "severity": "P2",
      "status": "fixed",
      "summary": "no runtime goroutine baseline after tab close",
      "verification": "omx-t07-browser-20261124-143959: 35 baseline / 39 with stream / 35 after close / 34 final"
    },
    {
      "id": "T09-R3",
      "severity": "P2",
      "status": "fixed",
      "summary": "slow-client backpressure bounded by the write deadline",
      "verification": "TestLogWriterStalledConsumerBoundedByDeadline PASS (stream stops after 3 MB with a 100 ms test deadline)"
    },
    {
      "id": "T09-R4",
      "severity": "P2",
      "status": "fixed",
      "summary": "reconnect rendered evidence missing",
      "verification": "provider end rendered 已断开; reconnect rendered 'failed to open pod log stream'; replacement pod reached 已连接"
    },
    {
      "id": "T09-R5",
      "severity": "P2",
      "status": "accepted-gap",
      "summary": "CI k8s business smoke does not open the pod-log page"
    }
  ],
  "verification": {
    "targetedLogTests": "6/6 PASS",
    "workloadPackage": "ok",
    "k8sModuleTests": "11/11 packages ok",
    "livePodLogWebSocket": "k12 PASS on real K3s (omx-accept-20261121-k8s38, 17/17 PASS)",
    "browserProbe": "PASS on omx-t07-browser-20261124-143959 (goroutines back to baseline; provider end and reconnect rendered)"
  },
  "linkage": {
    "taskManifest": ".harness/tasks/2026-11-20-ops-k8s-log-websocket-lifecycle/manifest.json",
    "taskPacket": "docs/harness/tasks/2026-11-20-ops-k8s-log-websocket-lifecycle.task.md",
    "evidence": ".harness/evidence/2026-11-20-ops-k8s-log-websocket-lifecycle/commands.json",
    "reviewFile": ".harness/evidence/2026-11-20-ops-k8s-log-websocket-lifecycle/review.md",
    "changeRef": "none",
    "planRefs": [".harness/tasks/2026-11-20-ops-review-remediation/plan.md"]
  }
}
```
