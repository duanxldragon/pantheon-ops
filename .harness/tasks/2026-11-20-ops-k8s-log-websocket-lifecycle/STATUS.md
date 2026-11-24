# Pod 日志 WebSocket 来源与连接生命周期

- Status: completed（浏览器证据与 review 完成；维护者最终视觉/功能验收 Human Gate 仍开放）
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
- **浏览器全链路 PASS**：`origin/main` `c6a0dd67` 候选后端 + 真 K3s，Vite 5176（`ws:true`）+ Playwright，`omx-t07-browser-20261124-143959`。goroutine 35 基线 / 39 持流 / 35 关闭 / 34 终值；provider 结束渲染 `已断开`，重连渲染 `failed to open pod log stream`，新 Pod 恢复 `已连接`。
- 证据：`../evidence/2026-11-20-ops-k8s-log-websocket-lifecycle/`（浏览器产物在 `browser-20261124-143959/`）。

## Known Gaps

1. CI k8s 业务 smoke 仍不打开 pod 日志页面；该页面只有本地 Playwright 探针覆盖。
2. 探针来源 `http://127.0.0.1:5176` 是显式登记进 `PANTHEON_ALLOWED_ORIGINS` 的本地值；生产部署来源白名单仅由服务端单测覆盖。

## Next Atomic Action

维护者最终视觉/功能验收；如需 CI 覆盖，后续把 pod 日志页面加入 k8s 业务 smoke。
