# Pod 日志 WebSocket 来源与连接生命周期

- Status: in-progress
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

## Verification

- `TestLogWriterSerializesConcurrentFrames`（100 并发帧）PASS；去掉 mutex 时 gorilla/websocket 并发写 panic，加 mutex 后绿。
- `TestWebsocketOriginAllowed`（5 子用例）+ `TestWebsocketOriginSameOriginFallback` PASS。
- workload 包 ok；K8s 全模块 11/11 包 ok。
- 证据：`../evidence/2026-11-20-ops-k8s-log-websocket-lifecycle/`。

## Known Gaps

1. 页面关闭后的 goroutine 基线/泄漏对比未采集。
2. 慢客户端背压（有界队列/窄通道）未运行时演练。
3. provider 错误后重连的渲染证据缺失；k8s 浏览器 smoke 未执行。

## Next Atomic Action

采集关闭/重开页面 goroutine 基线，演练慢客户端背压与重连，再补齐 V6 渲染证据。
