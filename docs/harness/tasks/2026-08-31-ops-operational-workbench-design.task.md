---
title: Define Ops business operational workbenches
doc_type: Design
layer: business/*
status: Active
updated_at: 2026-08-31
linked_contracts:
  - docs/designs/BUSINESS_OPERATIONAL_WORKBENCH_DESIGN.md
---

# Task Packet: 2026-08-31-ops-operational-workbench-design

## Goal

把 Pantheon Ops 的目标选择、变更预览、执行、查询、日志和 Dashboard 体验拆成可独立实现且遵守 Base-first 继承边界的业务工作包。

## Priority

`high`

## Estimated Complexity

`epic`

## Primary Layer

business/*

## Workspace Context

- Target Repository: `pantheon-ops`
- Repository Role: `business-consumer`
- Upstream Dependencies: `pantheon-harness`, `pantheon-base`
- Downstream Consumers: `none`
- Sync Expectation: `required`
- Release Requirement: `consumer-lock-update`

## Dependency Layers

- foundation-source: `pantheon-base`
- business domains: `business/bizscope`, `business/cmdb`, `business/deploy`, `business/k8s`, `business/observability`

## Dependencies

- blockedBy: Base B1-B4 按 packet 映射的 immutable foundation release
- blocks: Ops O1-O7 implementation

## Harness Profile

- Template: ui-heavy-product
- Overlay: pantheon-ops
- Quality Profile: ui-runtime
- Portable Failure Class: architecture-drift
- Owner Layer: consumer-repository
- Coverage Dimensions:
  - behaviour
  - maintainability
  - architecture-fitness
  - runtime-quality
  - method-health

## Contract Anchors

- `AGENTS.md`
- `docs/PROJECT_INHERITANCE.md`
- `../pantheon-base/DESIGN.md`
- `../pantheon-base/docs/designs/OPERATIONAL_WORKBENCH_COMPONENTS_DESIGN.md`
- `docs/designs/BUSINESS_OPERATIONAL_WORKBENCH_DESIGN.md`
- `docs/designs/BUSINESS_DEPLOY_MODULE_DESIGN.md`
- `docs/designs/BUSINESS_K8S_MODULE_DESIGN.md`
- `docs/designs/BUSINESS_OBSERVABILITY_MODULE_DESIGN.md`

## Scope

### In

- 盘点现有 Ops 业务 UI 与成熟运维交互模式的差距。
- 定义统一目标选择、部署执行、变更预览、查询、日志和 Dashboard 组合。
- 将实现拆为 O1-O7，并声明 Base 依赖、业务权限、DataScope、审计和证据矩阵。
- 将 CMDB 关系视图设为成熟度 gate 后的 P2 deferred 包。

### Out

- 不修改生产代码、数据库或外部系统。
- 不覆盖 `platform`、`system/*` 或 Base 共享组件。
- 不引入 BK Design 依赖。
- 不声称 Prometheus/Loki/Tempo/K8s/Argo CD/真实主机已完成外部验收。

## Assumptions and Open Questions

- Confirmed Facts: Ops 已有业务页面与部分运行链路，但共享工作台能力需由 Base 发布。
- Working Assumptions: 业务 API 将保留现有权限/DataScope/审计边界并增量补充查询或日志能力。
- Open Questions: none；外部环境与凭据缺失直接作为 runtime gap，不阻塞设计。

## Structural Scope

- Affected Subgraph: `Base shared contracts -> Ops business adapters -> permission/DataScope/audit -> external integrations`
- Boundary Crossings: `base -> ops`, `business UI -> business API -> external system`
- Risk Nodes: `target resolution, immutable intent, log streaming, query compiler, dashboard aggregation`
- Graph Focus: `sensitive-input-flow`

## Expected Files

### Create

- `docs/designs/BUSINESS_OPERATIONAL_WORKBENCH_DESIGN.md`
- `docs/harness/tasks/2026-08-31-ops-operational-workbench-design.task.md`
- `.harness/tasks/2026-08-31-ops-operational-workbench-design/**`

### Modify

- none

### Do Not Touch

- `backend/**`
- `frontend/**`
- `database/**`
- `../pantheon-base/**`
- existing dirty worktree files

## Implementation Notes

- Ops 只实现 `business/*` adapter 和业务组合；generic drift 回 Base。
- UI 显示动作必须以后端状态机、权限和 DataScope 为准。
- 外部运行证据不可用时记录 gap，不用 mock 截图冒充真机验收。

## Minimum Viable Approach

- Selected Rung: reuse
- Why This Is Enough: 在现有页面、API、SSE 和 Base 共享合同上增量组合，不建设第二套设计系统。
- Upgrade Trigger: 只有稳定业务模型和两个以上真实消费场景才新增跨业务抽象。

## Success Criteria

- Behaviour Outcome: O1-O7 可按依赖独立领取，且共享问题不会落成 Ops 本地副本。
- Verification Signal: task/manifest/link/docs checks 通过，每个子卡有业务和视觉可证伪验收。
- Regression Watch: 当前 Base inheritance、业务权限/DataScope、现有外部集成边界不变。
- Economics Watch: handoff 从父包定位到单个 packet，不回放完整审计聊天。

## Context Strategy

- Entry Sources: `AGENTS.md`, `docs/PROJECT_INHERITANCE.md`, 本 task packet, 父包 `HANDOFF.md`
- Retrieval Order: `entry -> design -> selected packet -> current code -> evidence`
- Retrieval Helpers: codegraph
- Promotion Target: `docs/designs/BUSINESS_OPERATIONAL_WORKBENCH_DESIGN.md`
- Response Budget: terse
- Sensitive Context: 不在 `.harness` 保存凭据、真实秘密或未脱敏日志

## Method Readiness

- Consumer-Specific Controls: business overlay、permission/DataScope contracts、smoke 和 external gap policy
- Required Sensors: command | review | runtime evidence
- Required Evidence: backend/frontend checks | screenshots | smoke | external gap | review
- Minimal Complexity Rung: reuse
- Ratchet Decision: guide-updated
- Deferred Code Issues: O1-O7 后续实施；O7 还受关系模型成熟度 gate 阻塞

## Delivery Governance

- Design Gate: canonical business design + parent packet
- Development Gate: Base dependency green and selected packet ownership
- QA Acceptance Gate: browser + runtime + human visual review
- GitHub Governance Gate: repo-quality-gate

## Execution Roles

- Implementer Posture: reviewer-assisted
- Reviewer Posture: UX-QA

## Stop Points

- 需要在 Ops 复制或 override Base 共享实现。
- 新外部生产动作、凭据或不可逆操作。
- 公共 API/schema/权限模型变化。
- foundation lock 更新与最终视觉/功能验收。

## State Plan

- Checkpoint Expectation: `.harness/tasks/2026-08-31-ops-operational-workbench-design/STATUS.md`
- Resume Artifacts: manifest、HANDOFF、STATUS、EXECUTION_QUEUE、选中 packet 与 evidence

## Verification Plan

### Backend

- none

### Frontend

- `node scripts/harness/check-task-packet.mjs --root . docs/harness/tasks/2026-08-31-ops-operational-workbench-design.task.md`
- `npm run check:harness-docs`
- `npm run check:harness-inventory`

### Browser / Smoke

- 本轮为设计文档，无新 UI 渲染；实现子卡执行 desktop/mobile 与业务 smoke。

### Runtime Evidence

- explicit runtime gap: 本轮无运行时代码，外部系统未触发。

## Linkage

- Task ID: `2026-08-31-ops-operational-workbench-design`
- Task Manifest: `.harness/tasks/2026-08-31-ops-operational-workbench-design/manifest.json`
- OpenSpec Change: `none`
- Superpowers Plan: `none`
- Plan References: `.harness/tasks/2026-08-31-ops-operational-workbench-design/EXECUTION_QUEUE.md`
- Evidence Directory: `.harness/evidence/2026-08-31-ops-operational-workbench-design/`
- Review File: `.harness/evidence/2026-08-31-ops-operational-workbench-design/review.md`

## Evidence Required

- 文档门禁结果与 impeccable 设计 review。
- 本轮没有目标 UI 渲染和外部运行证据的明确 gap。
- 后续每个子卡的 permission/DataScope/audit、desktop/mobile、状态与真机/外部 gap 证据。

## Human Gates

- Base foundation lock 更新。
- 新 schema、公共 API、权限模型或外部生产动作。
- 最终视觉/功能验收。

## Foundation Consumption

- Shared change owner: `pantheon-base`
- Required Base packets: O1/O2/O3/O4/O5/O6 分别声明
- Consumer sync status: `not-started`
- Downstream validation command: `npm run check:foundation-consumer`
- Stop point: immutable Base release -> Ops lock update -> overlay/build/smoke -> visual/function acceptance

## Completion Checklist

- [x] Layer and boundary declared
- [x] Quality profile declared
- [x] Ratchet decision declared
- [x] Delivery governance gates declared
- [x] Contract anchors read
- [x] Verification run or exception recorded
- [x] Evidence saved or summarized
- [x] Docs updated if contracts changed
- [x] Review completed
