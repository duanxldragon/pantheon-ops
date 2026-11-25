# 业务指标语义与集群维度收敛 — 实施摘要

- Task: `2026-11-24-ops-metrics-semantics`
- 日期: 2026-11-24
- 状态: **三处指标语义缺陷已修复并由回归锁定**
- 交付层级: L1（自审见 `review.md`）

## 来源与判定

来源为 2026-11-24 外部综合审查报告 §4.4（O-04）。三条结论均可由源码直接核实：

| 报告结论                                                                                             | 核实结果                                                                     |
| ---------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------- |
| `SetClusterCapacity(clusterID, …)` 收集群 ID 但 `ClusterCapacity` 只有 `metric` 标签，多集群互相覆盖 | 成立。原实现四行 `WithLabelValues("total_nodes"/… )`，`clusterID` 完全未使用 |
| `SetHostStatusMetrics` 只 Set 本次存在的状态，消失的状态保留旧值                                     | 成立。原实现仅遍历本次 map                                                   |
| `pantheon_host_attributes` 用 IP 作标签，注释写“避免高基数”不实                                      | 成立。注释与实现矛盾                                                         |

报告同时称“Ops 的 k8s/Grafana 清单与 Base 完全相同”，本轮未逐文件核实，不作为本包依据。

## 交付内容

| 验收项               | 实现                                                                                                                                                                | 验证                                                                        |
| -------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------- |
| 多集群容量互不覆盖   | `ClusterCapacity` 标签改为 `{cluster, metric}`；`SetClusterCapacity` 签名由 `uint64` 改为集群 code；`UpdateClusterStatusMetrics` / `RecordClusterMetrics` 传 `Code` | `TestSetClusterCapacityKeepsClustersIndependent`（两集群 8 个序列逐一断言） |
| 空集群 code 语义一致 | `normalizeLabel` 回落 `unknown`                                                                                                                                     | `TestSetClusterCapacityNormalizesEmptyClusterCode`                          |
| 消失状态归零         | `SetHostStatusMetrics` 改为整快照：维护上轮标签集合，本轮缺失项 `Set(0)`；包级 `sync.Mutex` 保护                                                                    | `TestSetHostStatusMetricsResetsStaleStates`（先红后绿）                     |
| 并发刷新安全         | 快照记录在互斥内读写                                                                                                                                                | `TestSetHostStatusMetricsConcurrentRefresh` 在 `-race` 下通过               |
| 注释与实现一致       | `HostAttributes` 注释如实描述基数随主机数增长、明细以 CMDB 为准                                                                                                     | 代码审阅                                                                    |

## 验证

- RED：临时移除陈旧状态归零循环后 `TestSetHostStatusMetricsResetsStaleStates` FAIL（`stale lifecycle label = 1, want 0`），随后立即恢复。
- GREEN：`go test -race -count=1 ./modules/business/metrics/ ./modules/business/k8s/cluster/ ./modules/business/cmdb/host/` → 3 包 ok。
- `go build ./...` OK；`golangci-lint run --new-from-rev=697c0771` 0 issues；`check-boundaries --strict` 0 findings。

## 显式 gap

1. 容量指标的 `cluster` 标签取集群 code，与 `ClusterSyncErrors`/`ClusterSyncDuration` 一致；但 code 仅在租户内唯一，指标无 tenant 标签，跨租户同名集群仍会合并序列。
2. `pantheon_k8s_cluster_status` 仍为全局聚合，未按集群拆分（刻意控制 diff）。
3. 集群/主机删除后对应容量与属性序列不主动清除。
4. Grafana dashboard / 告警规则未逐项对齐（O-10 独立范围）。
