# Review — 业务指标语义与集群维度收敛

## 结论

`approved with documented P2 follow-up`。三处指标语义缺陷均成立、均已最小修复，并有回归测试锁定；残留在租户维度、状态聚合与资产对齐，属显式 gap，不影响本包范围闭合。

## Findings

1. **P2 fixed — 多集群容量互相覆盖。**
   `ClusterCapacity` 只有 `metric` 一个标签，`SetClusterCapacity` 收了 `clusterID` 却从不使用；`UpdateClusterStatusMetrics` 在循环里对每个集群写同一条序列，最后一个同步的集群覆盖前面所有集群。修复为 `{cluster, metric}` 双标签，标签值取集群 code（与 `ClusterSyncErrors`/`ClusterSyncDuration` 同名标签取值一致），调用方改传 `Code`。

2. **P2 fixed — 消失状态永不归零。**
   `SetHostStatusMetrics` 只对本次集合中出现的生命周期/连通性标签 `Set`，某状态消失后 Gauge 保留旧的非零值，状态分布长期失真。修复为整快照：记录上轮标签集合，本轮缺失项 `Set(0)`；包级互斥避免并发刷新交叉污染快照记录。

3. **P2 fixed — 注释与实现矛盾。**
   `pantheon_host_attributes` 注释称“按 IP 区分，避免高基数标签”，实际 IP 标签基数随资产数线性增长。改为如实描述，并指明资产明细以 CMDB 为准。

4. **P2 accepted-gap — 集群 code 无租户隔离。**
   标签取 code 保持了与既有集群指标的一致，但 code 仅租户内唯一，容量指标没有 tenant 标签，跨租户同名集群仍会合并时间序列。这是既有指标契约的既有局限，不在本包引入标签语义分裂；如需租户维度应独立设计。

5. **P2 accepted-gap — 删除对象遗留序列。**
   集群/主机删除后，其容量与属性序列不会被主动清除。

6. **P2 accepted-gap — 资产对齐未做。**
   Grafana dashboard 与告警规则未逐项对齐（报告 O-10）。仓库内检索确认没有已入库资产引用被改动的指标选择器，因此新增 `cluster` 标签不会打破已提交面板。

## 验证要点

- RED→GREEN 均记录在 `commands.json`：RED 为临时移除归零循环后的 `stale lifecycle label = 1, want 0`。
- 受影响包与两个调用方在 `-race` 下全绿；整后端 `go build ./...` 通过；增量 lint 0 issues；跨层边界门禁 0 findings。
- 未触碰 `backend/modules/system/**`、`frontend/**`、`foundation-release.lock.json` 与他人未提交改动。

## Machine Readable

```json
{
  "taskId": "2026-11-24-ops-metrics-semantics",
  "reviewer": "primary agent self-review (no independent review on this candidate)",
  "verdict": "approved with documented P2 follow-up",
  "decision": "approved with documented P2 follow-up",
  "scope": "business Prometheus metric semantics: cluster dimension on capacity, full-snapshot host status refresh, attribute label documentation",
  "taskManifest": ".harness/tasks/2026-11-24-ops-metrics-semantics/manifest.json",
  "evidenceDir": ".harness/evidence/2026-11-24-ops-metrics-semantics",
  "summaryFile": ".harness/evidence/2026-11-24-ops-metrics-semantics/summary.md",
  "reviewFile": ".harness/evidence/2026-11-24-ops-metrics-semantics/review.md",
  "changeRef": "none (uncommitted worktree)",
  "findings": [
    {
      "id": "OMS-R1",
      "severity": "P2",
      "status": "fixed",
      "summary": "multi-cluster capacity gauge overwrote a single shared series",
      "verification": "TestSetClusterCapacityKeepsClustersIndependent PASS"
    },
    {
      "id": "OMS-R2",
      "severity": "P2",
      "status": "fixed",
      "summary": "disappeared host lifecycle/connectivity states kept stale non-zero gauges",
      "verification": "TestSetHostStatusMetricsResetsStaleStates PASS; failed before the reset loop was restored"
    },
    {
      "id": "OMS-R3",
      "severity": "P2",
      "status": "fixed",
      "summary": "host attribute comment claimed low cardinality while labelling by IP",
      "verification": "code review"
    },
    {
      "id": "OMS-R4",
      "severity": "P2",
      "status": "accepted-gap",
      "summary": "cluster code label has no tenant dimension, same-named clusters in two tenants merge series"
    },
    {
      "id": "OMS-R5",
      "severity": "P2",
      "status": "accepted-gap",
      "summary": "capacity/attribute series of deleted clusters or hosts are not removed"
    },
    {
      "id": "OMS-R6",
      "severity": "P2",
      "status": "accepted-gap",
      "summary": "Grafana dashboards and alert rules not reconciled with the new label"
    }
  ],
  "verification": {
    "race": "go test -race -count=1 ./modules/business/metrics/ ./modules/business/k8s/cluster/ ./modules/business/cmdb/host/ -> ok",
    "build": "go build ./... -> ok",
    "lint": "golangci-lint run --new-from-rev=697c0771 ./modules/business/metrics/... ./modules/business/k8s/cluster/... -> 0 issues",
    "boundaries": "node scripts/harness/check-boundaries.mjs --strict --repo pantheon-ops -> 0 findings"
  },
  "linkage": {
    "taskManifest": ".harness/tasks/2026-11-24-ops-metrics-semantics/manifest.json",
    "taskPacket": "docs/harness/tasks/2026-11-24-ops-metrics-semantics.task.md",
    "evidence": ".harness/evidence/2026-11-24-ops-metrics-semantics/commands.json",
    "reviewFile": ".harness/evidence/2026-11-24-ops-metrics-semantics/review.md",
    "changeRef": "none",
    "planRefs": []
  },
  "humanGate": "final functional/visual acceptance belongs to the maintainer"
}
```
