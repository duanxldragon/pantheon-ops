# 业务指标语义与集群维度收敛

- Status: completed
- Updated: 2026-11-24
- Baseline: 11f9b004（候选 HEAD）+ 未提交工作树；原始实现基线 697c0771，697c0771..11f9b004 只改 k8s/cluster 请求取消（#214），未触及本包文件
- Owner: 业务指标维护者
- Priority: P2
- Activation: core
- Dependencies: none

## Implemented

- `pantheon_k8s_cluster_capacity` 增加 `cluster` 标签，`SetClusterCapacity` 由收 `uint64` 集群 ID 改为收集群 code 并按 `(cluster, metric)` 写入；`UpdateClusterStatusMetrics` 与 `RecordClusterMetrics` 传 `Code`，多集群不再互相覆盖。
- `SetHostStatusMetrics` 改为整快照刷新：记录上轮标签集合，本轮消失的生命周期/连通性状态归零；包级互斥保护快照记录。
- `pantheon_host_attributes` 注释改为如实描述按主机 IP 展开的基数增长，并指明资产明细以 CMDB 为准。
- 新增 `backend/modules/business/metrics/metrics_test.go`（4 个测试，含并发刷新）。

## Verification

- 修复前 RED：临时移除陈旧状态归零后 `TestSetHostStatusMetricsResetsStaleStates` FAIL（`stale lifecycle label = 1, want 0`）。
- 11f9b004 复核：`go test -race`（3 包）、`go build ./...`、增量 `golangci-lint`（0 issues）、`check-boundaries --strict`（0 findings）全部重跑通过。
- 修复后 GREEN：`go test -race -count=1 ./modules/business/metrics/ ./modules/business/k8s/cluster/ ./modules/business/cmdb/host/` → 3 包 ok。
- `go build ./...` OK；`golangci-lint run --new-from-rev=697c0771` 0 issues；`check-boundaries.mjs --strict --repo pantheon-ops` 0 findings。
- 证据：`../evidence/2026-11-24-ops-metrics-semantics/`。

## Known Gaps

1. 租户间集群 code 可重复，容量指标无 tenant 标签，跨租户同名集群会合并序列（与既有 `ClusterSyncErrors` 同一约定与同一限制）。
2. `pantheon_k8s_cluster_status` 仍为全局聚合，未按集群拆分（本包刻意控制 diff）。
3. 集群/主机删除后对应容量与属性序列不会主动清除。
4. Grafana dashboard 与告警资产未逐项对齐（报告 O-10 独立范围）。

## Next Atomic Action

无（本包闭合）。如需租户维度或资产标签改造，另立设计任务。
