# O5 Task And Pod Log Experience

- Priority: `P1`
- Layer: `business/deploy + business/k8s`
- Status: `blocked`
- Base Dependency: B3 `TaskLogViewer`
- Business Dependency: Deploy chunk stream + Kubernetes pod log API

## Outcome

让 Deploy 与 Pod 日志共用一致的查看体验，同时保持两个数据源的权限、参数和重连语义独立。

## In Scope

- Deploy adapter：task/host/attempt/step/stream/sequence/stale/redaction metadata。
- Pod adapter：cluster/namespace/workload/pod/container/previous/since/timestamps/follow。
- search、pause tail、copy/download、wrap、fullscreen、level/source filters。
- 下载权限、审计、最大范围/大小和脱敏策略。

## Out Of Scope

- 不把 Pod 日志和 Deploy 日志合并成同一后端 API。
- 不在浏览器持久化完整日志。
- 不允许下载绕过页面的 DataScope 和脱敏。

## Expected Files

- `frontend/src/modules/business/k8s/workload/PodLogViewer.tsx`。
- Deploy task log composition、API adapters、i18n 和 tests。
- provider-owned stream/download endpoints 的必要增量。

## Acceptance

1. 追尾暂停不会丢失已接收序号；恢复时明确补齐或声明截断。
2. 切换 pod/container/host/step 后上下文和日志来源始终可见。
3. 10k+ 行保持有界 DOM/内存；搜索和筛选不改变原始顺序。
4. disconnect、stale、truncated、forbidden 和 redaction 状态可辨。
5. mobile 全屏日志可退出、可搜索，固定控件不遮挡文本。

## Evidence

- unit：adapter mapping、sequence dedupe、buffer cap、filter/search、download policy。
- backend：DataScope、stream auth、download audit、redaction、range limit。
- Playwright：follow/pause/search/filter/fullscreen/reconnect/source switch。
- runtime：测试 Pod 和 Deploy task；缺 K8s/VM credential 分别记录 gap。
- screenshots：desktop/mobile x following/paused/stale/forbidden x light/dark。

## Gates

- Base lock、日志保留/下载政策、外部集群/主机凭据、新公共 stream API。
