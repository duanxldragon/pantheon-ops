---
title: 业务指标语义与集群维度收敛
doc_type: Remediation
layer: business
status: Archived
updated_at: 2026-11-24
linked_contracts:
  - docs/designs/BUSINESS_FUNCTIONAL_DESIGN.md
---

# Task Packet: 2026-11-24-ops-metrics-semantics

业务指标语义与集群维度收敛

## Goal

修正 Ops 业务 Prometheus 指标的三处语义缺陷：集群容量指标忽略集群身份导致多集群互相覆盖、主机状态 Gauge 不重置已消失的状态、主机属性标签基数说明与实际不符，并以回归测试锁定

## Primary Layer

business/*

## Dependency Layers

- business/*

## Dependencies

- blockedBy: none
- Conditional Dependencies: none
- Activation: core

## Harness Profile

- Template: custom
- Overlay: pantheon-ops
- Quality Profile: none
- Delivery Tier: L1
- Priority: P2
- Coverage Dimensions:
  - behaviour
  - architecture-fitness

## Workspace Context

- Target Repository: pantheon-ops
- Repository Role: business-consumer
- Upstream Dependencies: pantheon-base
- Downstream Consumers: none
- Sync Expectation: none
- Release Requirement: none
- Downstream Sync Note: 指标实现与调用方均在 business overlay，源文件归属 Ops，不涉及 Base 回流和 foundation release。

## Contract Anchors

- `AGENTS.md`
- `docs/PROJECT_INHERITANCE.md`
- `docs/TASK_PACKET_OPS_TEMPLATE.md`
- `business-overlay.json`
- `docs/designs/BUSINESS_FUNCTIONAL_DESIGN.md`

## Source Findings

- O-04（2026-11-24 外部综合审查报告 §4.4：业务 Metrics 语义问题）

来源判定：指标名与标签缺陷可由源码直接核实；报告对“Grafana 资产与 Base 完全相同”的结论本轮未核实，不作为本包依据。

## Scope

### In

- `pantheon_k8s_cluster_capacity` 增加 `cluster` 标签并在刷新与单集群路径传入集群 code，使多集群互不覆盖
- `pantheon_hosts_total` / `pantheon_host_connectivity` 改为整快照刷新：本次集合中消失的状态归零
- 修正 `pantheon_host_attributes` 标签基数注释，避免把按 IP 展开的明细称作“避免高基数”
- 为上述语义补包内回归测试

### Out

- `platform/system/*`、共享 UI/权限/租户/请求拦截器实现由 pantheon-base 拥有，按正式 foundation release 同步
- Grafana dashboard / 告警规则资产对齐（报告 O-10，尚需逐项核对真实资产）
- 指标租户维度、主机资产标签改造与 CMDB 明细下移（需独立设计）
- 未授权生产操作、发布、凭证变更及门禁豁免

## Assumptions and Open Questions

- Confirmed Facts: `SetClusterCapacity` 收 `uint64` 集群 ID 但只按 `metric` 单标签写值；`SetHostStatusMetrics` 只对本次出现的标签 Set；调用方为 `cmdb/host/host_metrics.go` 与 `k8s/cluster/cluster_metrics.go`。
- Working Assumptions: 既有 `cluster` 标签约定取集群 code（`ClusterSyncErrors`/`ClusterSyncDuration`），容量指标沿用同一取值以免同名标签语义分裂。
- Open Questions: 租户间集群 code 可重复，容量指标暂无 tenant 标签，跨租户同名集群会合并序列；按显式 gap 登记，不在本包引入标签契约变更。
- Evidence State: 本包已执行并收口代码级验证，运行态 Grafana 渲染不在范围。

## Structural Scope

- Affected Subgraph: 业务指标刷新 -> cluster/host 采集调用方
- Boundary Crossings: business/* -> Prometheus 客户端库
- Risk Nodes: O-04
- Graph Focus: call-depth

## Expected Files

### Create

- `docs/harness/tasks/2026-11-24-ops-metrics-semantics.task.md`
- `.harness/tasks/2026-11-24-ops-metrics-semantics/manifest.json`
- `.harness/tasks/2026-11-24-ops-metrics-semantics/STATUS.md`
- `.harness/evidence/2026-11-24-ops-metrics-semantics/commands.json`
- `.harness/evidence/2026-11-24-ops-metrics-semantics/summary.md`
- `.harness/evidence/2026-11-24-ops-metrics-semantics/review.md`
- `backend/modules/business/metrics/metrics_test.go`

### Modify

- `backend/modules/business/metrics/metrics.go`
- `backend/modules/business/k8s/cluster/cluster_metrics.go`

### Do Not Touch

- `backend/modules/system/**`
- `backend/internal/**`
- `frontend/**`
- `foundation-release.lock.json`
- `docker-compose.yml`
- `既有归档原始证据与其他执行者未提交改动`

## Implementation Notes

- Minimal Complexity Rung: 最小新增代码
- 复用既有 `normalizeLabel` 与 `promauto` 注册，不引入新库；测试使用已依赖的 `prometheus/client_golang/prometheus/testutil`
- 集群容量沿用既有 `cluster=code` 标签约定；不改 `pantheon_k8s_cluster_status` 的聚合语义以控制 diff
- 主机状态刷新用包级互斥保护快照记录，保证并发刷新下陈旧状态归零一致
- 指标实现与调用方均在 `business-overlay.json` 的 `businessPaths`，源文件权威归 Ops

## Acceptance Criteria

- [x] 两个集群同时刷新时各自容量序列独立，互不覆盖
- [x] 主机集合中消失的生命周期/连通性状态在下一次刷新归零
- [x] 集群 code 为空时回落到 `unknown` 标签
- [x] 并发刷新在 `-race` 下无数据竞争
- [x] 受影响包单测与整后端编译通过，增量 lint 0 issues

## Verification Plan

- `go test -race -count=1 ./modules/business/metrics/ ./modules/business/k8s/cluster/ ./modules/business/cmdb/host/`，工作目录 `backend`
- `go build ./...`，工作目录 `backend`
- `golangci-lint run --new-from-rev=697c0771 ./modules/business/metrics/... ./modules/business/k8s/cluster/...`，工作目录 `backend`
- `node scripts/harness/check-boundaries.mjs --strict --repo pantheon-ops`，工作目录仓库根

- 修复前 RED 与修复后 GREEN 均已记录（`TestSetHostStatusMetricsResetsStaleStates`）
- 运行态 Grafana/告警渲染不在本包范围；指标契约变更为新增标签，无既有资产引用（仓库内检索为空）
- 命名空间/租户边界未变，无需权限、菜单、i18n、schema、seed、fixture 同步

## Execution Roles

- Implementer Posture: 业务指标维护者，只修改声明的业务文件，复用既有标签约定。
- Reviewer Posture: 对照多集群、陈旧状态与并发刷新副作用审查，确认无指标契约外的行为变化。

## Stop Points

- 需要改变既有 `cluster` 标签取值语义、引入 tenant 标签或调整指标名时停止确认
- 需要修改 Base 跟踪文件（如 `backend/.golangci.yml`）时不在本包内改，另立 Base 任务

## State Plan

- Checkpoint Expectation: 单包代码级闭合，含 RED/GREEN 与 lint/build evidence

## Linkage

- Task ID: 2026-11-24-ops-metrics-semantics
- Task Manifest: .harness/tasks/2026-11-24-ops-metrics-semantics/manifest.json
- Evidence Directory: .harness/evidence/2026-11-24-ops-metrics-semantics/
- Review File: .harness/evidence/2026-11-24-ops-metrics-semantics/review.md
- OpenSpec Change: none
- Change Ref: none
- Superpowers Plan: none
- Plan References: none

## Evidence Required

- commands.json 含真实执行状态、cwd、退出码与候选身份
- summary.md 记录 RED/GREEN、受影响包与显式 gap
- review.md 为 findings-first 自审并含机器可读 linkage

## Human Gates

- 需求/范围偏好、门禁策略与豁免、最终视觉/功能验收按维护者三触点契约处理
- 正式发布与仓库基线替换需已有授权

## Rollback Plan

回退本包三处业务文件改动即可恢复原指标标签与刷新语义；无数据库与配置变更

## Completion Checklist

- [x] Layer and boundary declared
- [x] Contract anchors read
- [x] Verification run or exception recorded
- [x] Evidence saved or summarized
- [x] Review completed
