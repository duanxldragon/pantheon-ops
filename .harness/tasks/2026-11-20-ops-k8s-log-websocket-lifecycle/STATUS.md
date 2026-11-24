# Pod 日志 WebSocket 来源与连接生命周期

- Status: in-progress (implementation complete; runtime browser evidence open)
- Updated: 2026-11-24
- Baseline: a2d69d4a759b5df2b26aaa9d26bdc5175babe9dc
- Owner: K8s 日志负责人
- Priority: P1
- Activation: core
- Dependencies: T06, T07

## Implemented

- Origin 允许清单（`PANTHEON_ALLOWED_ORIGINS`）+ 同源回退；越界/缺失 Origin 语义已锁定。
- 请求 context 取消传播到 provider `streamPodLogs`；客户端断开触发取消；scanner 1MB 上界。
- 单写者 websocket 协议：`logWriter` 互斥 + 写超时，串行化日志流与 ping ticker。
- 慢客户端背压上界：设计无有界队列，同步写 + `logWriteWait` 写超时把慢消费者限制在固定时间内（`logWriteWait` 改为包级变量以便回归缩短超时，生产默认 10s）。

## Verification

- `TestLogWriterSerializesConcurrentFrames`（100 并发帧）PASS；去掉 mutex 时 gorilla/websocket 并发写 panic，加 mutex 后绿。
- `TestLogWriterStalledConsumerBoundedByDeadline`（不读取对端、1 MB 帧循环、100 ms 测试超时）PASS：第 3 MB 触发写超时，流停止。
- `TestWebsocketOriginAllowed`（5 子用例）+ `TestWebsocketOriginSameOriginFallback` PASS。
- workload 包 ok；K8s 全模块 11/11 包 ok；golangci `--new-from-rev=32f7fa5a` 0 issues。
- **真实 Pod 日志 WebSocket（k12）PASS**：main `697c0771` 候选后端 + 真 K3s，`k8s-live.py --run-id omx-accept-20261121-k8s38` 17/17。
- 证据：`../evidence/2026-11-20-ops-k8s-log-websocket-lifecycle/`。

## Known Gaps

1. 页面关闭后的 goroutine 基线/泄漏对比未采集（浏览器侧）。
2. provider 错误后重连的浏览器渲染证据缺失；CI k8s 页面 smoke 不打开该页面。

## Next Atomic Action

采集关闭/重开页面 goroutine 基线并补齐重连渲染与 k8s 浏览器 smoke 证据。
