# T09 Pod 日志 WebSocket 来源与连接生命周期 — 实施摘要

- Task: `2026-11-20-ops-k8s-log-websocket-lifecycle`
- 日期: 2026-11-24
- 状态: **单写者协议、Origin 校验与写超时背压上界已锁定；goroutine/重连运行时证据为显式 gap**
- 交付层级: L2（自审见 `review.md`）

## 已交付

| 验收项 | 状态 | 证据 |
|---|---|---|
| Origin 允许清单 / 同源回退 / 越界拒绝 | 已实现 | `websocketOriginAllowed` + `TestWebsocketOriginAllowed`（含 trailing slash、foreign、null、缺失 Origin）与 `TestWebsocketOriginSameOriginFallback` |
| provider goroutine 随 context 取消释放 | 已实现 | `streamPodLogs(ctx, writer, ...)`；客户端断开触发 request context 取消；scan buffer 1MB 上界 |
| 单写者 writer 协议（日志流与 ping ticker 并发写） | 已修复 | `logWriter{mu, conn}` 串行化 `WriteMessage` 并设置写超时；`TestLogWriterSerializesConcurrentFrames` 100 并发帧 |
| 测试红→绿证明 | 已成立 | 去掉 mutex 时 gorilla/websocket 并发写 panic；加 mutex 后 PASS |
| 慢客户端背压上界 | 已实现（代码级） | `logWriteWait` 写超时约束同步写；`TestLogWriterStalledConsumerBoundedByDeadline`：不读取的连接在第 3 MB 触发写超时，流停止，无无界缓冲 |

## 验证

- 定向 6 个测试全 PASS（新增写超时背压回归）；workload 包 ok；K8s 全模块 11/11 包 ok；golangci `--new-from-rev=32f7fa5a` 0 issues。

## 显式 gap

1. 关闭/重开页面后的 goroutine 基线（泄漏对比）未采集。
2. provider 错误后重连的渲染证据缺失。CI k8s 业务 smoke（PR #207 Smoke Sanity）为绿，但该套件不打开 pod 日志 WebSocket 页面，故 pod 日志浏览器证据仍为显式 gap。
