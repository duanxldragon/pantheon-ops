---
title: Ops 首期试点业务交付计划
doc_type: Remediation
layer: business
status: Archived
linked_contracts:
  - docs/PROJECT_INHERITANCE.md
updated_at: 2026-11-20
---

# Task Packet: 2026-11-16-ops-pilot-delivery

## Goal

关闭首期核心业务闭环、当前版本验收与可复现交付物；扩展按条件单独验收。

## Priority

P0；L2；首期核心/治理范围。

## Primary Layer

business/*

## Dependency Layers

- business/*
- inheritance-sync

## Harness Profile

- Template: admin-platform
- Overlay: pantheon-ops
- Coverage Dimensions:
  - behaviour
  - architecture-fitness
  - runtime-quality

## Workspace Context

- Target Repository: pantheon-ops
- Repository Role: business-consumer
- Upstream Dependencies: pantheon-base
- Downstream Consumers: none
- Sync Expectation: not-required
- Release Requirement: none
- Downstream Sync Note: 仅业务消费者；共享变更回 Base 正式 release 后再消费

## Contract Anchors

- `AGENTS.md`
- `docs/PROJECT_INHERITANCE.md`
- `docs/designs/BUSINESS_FUNCTIONAL_DESIGN.md`
- `docs/designs/BUSINESS_QA_ACCEPTANCE_PLAN.md`

## Scope

### In

- C01-C06 首期核心
- E01-E04 条件扩展
- 保留跨仓库继承边界、历史证据与明确 gap

### Out

- platform/system/* 通用实现与规则副本；共享问题 Base-first
- 未授权生产操作、规则豁免与风险接受
- 自动 exporter/file_sd/商业 License 等后续产品扩张

## Expected Files

### Create

- docs/harness/tasks/2026-11-16-ops-pilot-delivery.task.md
- .harness/tasks/2026-11-16-ops-pilot-delivery/manifest.json
- .harness/evidence/2026-11-16-ops-pilot-delivery/summary.md
- .harness/evidence/2026-11-16-ops-pilot-delivery/review.md
- .harness/evidence/2026-11-16-ops-pilot-delivery/commands.json

### Modify

- none: program tracking only

### Do Not Touch

- ../pantheon-base/**
- foundation-release.lock.json
- backend/modules/system/**
- frontend/src/modules/system/**

## Implementation Notes

- Owner: business-consumer
- Minimum Complexity Rung: reuse
- 开始实现前重新读取当前候选与结构；本审查行号绑定旧基线，不能覆盖并发升级。
- Windows race 必须 PATH 前置 D:\msys64\mingw64\bin，CGO_ENABLED=1；普通 Go tests 不替代。
- 业务 mutation 的 API/权限/菜单/i18n/迁移/fixture/smoke 同步关闭；复用共享契约，禁止本地 override。
- 依赖未完成不执行后续；状态只按实际 evidence 更新。

## Structural Scope

- Affected Subgraph: business-consumer
- Boundary Crossings: 业务 owner capability / HTTP UI / 外部 provider；治理任务只关联任务与证据
- Risk Nodes: 目标身份、ownership、状态回写、引用、发布与证据真实性
- Graph Focus: manifest → task packet → evidence → archive-aware resolver

## Execution Roles

- Implementer Posture: executor/Generator 在明确 owner 内实现；当前计划会话只改治理资料
- Reviewer Posture: 独立 code-reviewer/verifier 对照 In/Out 和运行证据；静态推断不得写成已复现

## Verification Plan

- 逐项执行子任务 verificationPlan；所有 C 任务完成后最终验收

## Acceptance Criteria

- C01-C06 全部实际完成；可选扩展未启用或 E 队列按选中能力完成；维护者最终验收。

## Dependencies

- none

## Linkage

- Task ID: 2026-11-16-ops-pilot-delivery
- OpenSpec Change: none
- Superpowers Plan: none
- Task Manifest: .harness/tasks/2026-11-16-ops-pilot-delivery/manifest.json
- Evidence Directory: .harness/evidence/2026-11-16-ops-pilot-delivery/
- Review File: .harness/evidence/2026-11-16-ops-pilot-delivery/review.md
- Change Ref: none
- Plan References: .harness/tasks/2026-11-16-ops-pilot-delivery/plan.md

## Evidence Required

- commands.json 的真实命令/cwd/退出状态；summary.md 区分 passed/failed/not-run/skip；review.md 独立结论。
- 业务及外部验收还需候选 revision、foundation、迁移、fixture/hash、正反向结果、截图或明确 gap。
- planned 任务不预写通过证据，不以父计划文档验证冒充子任务完成。

## Human Gates

- 范围已由维护者确认：按业务设计评估试点；安全可逆步骤自主执行。
- 生产操作、门禁豁免、风险接受及最终视觉/功能验收由维护者决定；不得以等候时间作为批准。

## Stop Points

- 生产/凭据授权缺失、破坏性数据操作、门禁豁免、关键合同冲突时停止依赖动作；继续独立可逆工作。

## Completion Checklist

- [x] Layer and boundary declared
- [x] Contract anchors read
- [ ] Verification run or exception recorded
- [ ] Evidence saved or summarized
- [ ] Review completed

## Status

in-progress；C01/C03/C04 正在实施。只在依赖与实际验收完成后转 completed；当前证据见 .harness/evidence/2026-11-17-ops-history-audit/progress.md。

## Result — History Closeout 2026-11-20

Manifest disposition: completed

10 包 pilot 实施 + 验收证据随 PR #195 (2578eac90) 落地 main；候选验收证据已记录。原范围验收判据保留为显式 residual，不代签维护者验收。计划书随本 closeout 入档（.harness/archive/2026-11/tasks/2026-11-16-ops-pilot-delivery/plan.md）。

Residual finding/owner/successor/proof: .harness/evidence/2026-11-20-ops-board-complete/dispositions.json. Canonical references remain archive-aware. Historical body/evidence retained. Candidate/production/maintainer gates remain separate.

## Result — History Closeout 2026-11-20

Manifest disposition: completed

10 包 pilot 实施 + 验收证据随 PR #195 (2578eac90) 落地 main；候选验收证据已记录。原范围验收判据保留为显式 residual，不代签维护者验收。计划书随本 closeout 入档（.harness/archive/2026-11/tasks/2026-11-16-ops-pilot-delivery/plan.md）。

Residual finding/owner/successor/proof: .harness/evidence/2026-11-20-ops-board-complete/dispositions.json. Canonical references remain archive-aware. Historical body/evidence retained. Candidate/production/maintainer gates remain separate.
