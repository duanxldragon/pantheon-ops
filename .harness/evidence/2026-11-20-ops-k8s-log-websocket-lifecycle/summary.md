# T09 Pod 日志 WebSocket 来源与连接生命周期 — 实施摘要

- Task: `2026-11-20-ops-k8s-log-websocket-lifecycle`
- 日期: 2026-11-24
- 状态: **单写者协议、Origin 校验与写超时背压上界已锁定；真实 Pod 日志 WebSocket（k12）PASS；浏览器 goroutine 基线与重连渲染证据已补齐**
- 交付层级: L2（自审见 `review.md`）

## 已交付

| 验收项                                            | 状态             | 证据                                                                                                                                               |
| ------------------------------------------------- | ---------------- | -------------------------------------------------------------------------------------------------------------------------------------------------- |
| Origin 允许清单 / 同源回退 / 越界拒绝             | 已实现           | `websocketOriginAllowed` + `TestWebsocketOriginAllowed`（含 trailing slash、foreign、null、缺失 Origin）与 `TestWebsocketOriginSameOriginFallback` |
| provider goroutine 随 context 取消释放            | 已实现           | `streamPodLogs(ctx, writer, ...)`；客户端断开触发 request context 取消；scan buffer 1MB 上界                                                       |
| 单写者 writer 协议（日志流与 ping ticker 并发写） | 已修复           | `logWriter{mu, conn}` 串行化 `WriteMessage` 并设置写超时；`TestLogWriterSerializesConcurrentFrames` 100 并发帧                                     |
| 测试红→绿证明                                     | 已成立           | 去掉 mutex 时 gorilla/websocket 并发写 panic；加 mutex 后 PASS                                                                                     |
| 慢客户端背压上界                                  | 已实现（代码级） | `logWriteWait` 写超时约束同步写；`TestLogWriterStalledConsumerBoundedByDeadline`：不读取的连接在第 3 MB 触发写超时，流停止，无无界缓冲             |
| 浏览器 goroutine 基线 / 关闭后泄漏对比            | 已实测           | `browser-20261124-143959/probe.json`：35 基线 / 39 持流 / 35 关闭 / 35 恢复后关闭 / 34 终值（≤ 基线 + 3）                                          |
| provider 结束后重连渲染                           | 已实测           | `provider-closed.png`（已断开）→ `reconnect-error.png`（`failed to open pod log stream: pods "…" not found`）→ `reconnect-success.png`（新 Pod 已连接） |

## 验证

- 定向 6 个测试全 PASS（新增写超时背压回归）；workload 包 ok；K8s 全模块 11/11 包 ok；golangci `--new-from-rev=32f7fa5a` 0 issues。
- **真实 Pod 日志 WebSocket（k12）PASS**：main `697c0771` 候选后端 + 真 K3s，`k8s-live.py --run-id omx-accept-20261121-k8s38` 17/17；握手、首帧含运行 marker、帧长 ≤4096、客户端关闭均成立。
- **浏览器全链路 PASS**：候选后端由 `origin/main` `c6a0dd67` 构建（sha256 `f33af1bf…`）+ 真 K3s，本地 Vite（5176，`ws:true`）+ Playwright；`omx-t07-browser-20261124-143959` PASS。工作负载列表 → Pod → 日志 `已连接` → 关闭后 goroutine 回基线 → 重开持流 → 缩容到 0 后 `已断开` → 重连渲染 provider 错误 → 扩容回 1 后新 Pod `已连接` → 关闭后 goroutine 回基线。证据目录 `browser-20261124-143959/`。

## 显式 gap

1. CI k8s 业务 smoke（PR #207 Smoke Sanity）为绿，但仍不打开 pod 日志 WebSocket 页面；本次浏览器证据来自本地 Playwright 探针，仓库 CI 尚无该页面覆盖。
2. 探针用本地 Vite 5176 并在 `PANTHEON_ALLOWED_ORIGINS` 显式登记该来源；生产部署来源白名单仍只由服务端 Origin 单测覆盖。
