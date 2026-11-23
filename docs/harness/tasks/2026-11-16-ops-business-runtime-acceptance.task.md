---
title: 当前版本数据库、执行与业务视觉验收
doc_type: Remediation
layer: business
status: Archived
linked_contracts:
  - docs/PROJECT_INHERITANCE.md
updated_at: 2026-11-20
---

# Task Packet: 2026-11-16-ops-business-runtime-acceptance

## Goal

当前版本数据库、执行与业务视觉验收

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

- 固定实际候选 revision、已装配 Base release、业务迁移及 fixture；空库/升级/重复/故障/回滚隔离演练。
- 使用 task-owned MySQL/Redis/SSH/K8s，执行身份/归属、部署失败续跑、取消、实例回写及发布回滚；提供测试环境而不是访问生产。
- 在干净候选树运行 ops 核心业务 UI/API smoke；新环境管理、菜单和受限角色由浏览器驱动验收。
- 补业务 smoke 的 CI 接入或独立必过 release gate；现有 Smoke Sanity 只证明平台/系统域。

### Out

- platform/system/* 通用实现与规则副本；共享问题 Base-first
- 未授权生产操作、规则豁免与风险接受
- 自动 exporter/file_sd/商业 License 等后续产品扩张

## Expected Files

### Create

- docs/harness/tasks/2026-11-16-ops-business-runtime-acceptance.task.md
- .harness/tasks/2026-11-16-ops-business-runtime-acceptance/manifest.json
- .harness/archive/2026-11/evidence/2026-11-16-ops-business-runtime-acceptance/summary.md
- .harness/archive/2026-11/evidence/2026-11-16-ops-business-runtime-acceptance/review.md
- .harness/archive/2026-11/evidence/2026-11-16-ops-business-runtime-acceptance/commands.json

### Modify

- frontend/tests/smoke/business
- .github/workflows/quality.yml
- backend/modules/business/deploy/deploy_live_acceptance_test.go
- backend/modules/business/observability/query_workspace_owner_lifecycle_test.go

### Do Not Touch

- ../pantheon-base/**
- foundation-release.lock.json
- backend/modules/system/**
- frontend/src/modules/system/**

## Implementation Notes

- Owner: business/* acceptance
- Minimum Complexity Rung: reuse
- 开始实现前重新读取当前候选与结构；本审查行号绑定旧基线，不能覆盖并发升级。
- Windows race 必须 PATH 前置 D:\msys64\mingw64\bin，CGO_ENABLED=1；普通 Go tests 不替代。
- 业务 mutation 的 API/权限/菜单/i18n/迁移/fixture/smoke 同步关闭；复用共享契约，禁止本地 override。
- 依赖未完成不执行后续；状态只按实际 evidence 更新。

## Structural Scope

- Affected Subgraph: business/* acceptance
- Boundary Crossings: 业务 owner capability / HTTP UI / 外部 provider；治理任务只关联任务与证据
- Risk Nodes: 目标身份、ownership、状态回写、引用、发布与证据真实性
- Graph Focus: .github/workflows/quality.yml:394; .harness/archive/2026-11/evidence/2026-11-16-ops-business-menu-m3c/summary.md

## Execution Roles

- Implementer Posture: executor/Generator 在明确 owner 内实现；当前计划会话只改治理资料
- Reviewer Posture: 独立 code-reviewer/verifier 对照 In/Out 和运行证据；静态推断不得写成已复现

## Verification Plan

- go test -race -count=1 -json ./...
- npm run check:business-foundation
- npm run lint
- npm run type-check
- npm run build
- npm run test:smoke:business:cmdb
- npm run test:smoke:business:deploy:api
- npm run test:smoke:business:deploy
- npm run test:smoke:business:service
- npm run test:smoke:business:k8s

## Acceptance Criteria

- 关键 MySQL、SSH、K8s、目标错配及引用保护用例零未解释 skip，所有 P0 通过。
- 两部门/业务域及最小权限角色均验证；正向、越权、重复、并发、失败恢复、回滚留证。
- 当前 revision 桌面/窄屏和主要交互状态截图，source/迁移/hash/命令/退出码对应同一候选。
- 旧菜单任务运行 gap 得到当前证据关闭；旧历史截图不替代。

## Dependencies

- 2026-11-16-ops-service-deploy-integrity
- 2026-11-16-ops-service-deploy-user-flow
- 2026-11-16-ops-service-catalog-usability
- 2026-11-16-ops-k8s-service-mutations

## Linkage

- Task ID: 2026-11-16-ops-business-runtime-acceptance
- OpenSpec Change: none
- Superpowers Plan: none
- Task Manifest: .harness/tasks/2026-11-16-ops-business-runtime-acceptance/manifest.json
- Evidence Directory: .harness/archive/2026-11/evidence/2026-11-16-ops-business-runtime-acceptance/
- Review File: .harness/archive/2026-11/evidence/2026-11-16-ops-business-runtime-acceptance/review.md
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

in-progress；只在依赖与实际验收完成后转 completed。

## 2026-11-18T07:31:44.466Z Continuation checkpoint

Status: in-progress. CI 五项核心业务 smoke 已设为必过；workflow 实际 shell 失败传播回归 RED10/12→GREEN12/12、0skip。当前工作属于准备，未运行原完整验收。

Original acceptance and dependencies remain unchanged. Whole-package acceptance is incomplete; see [summary](../../../.harness/archive/2026-11/evidence/2026-11-16-ops-business-runtime-acceptance/summary.md) and [progress receipt](../../../.harness/evidence/2026-11-17-ops-history-audit/continuation-ten-packages-progress-20261118.json).

## 2026-11-19T00:11:15.272Z Verified continuation checkpoint

Status: in-progress. 五项业务CI smoke已必过，真实shell传播回归RED10/12→GREEN12/12；完整CGO/race、frontend lint/type/build与正式重建/漂移已通过，新source-bound binary/dependencies ready。仍处本地准备，未完成原运行态矩阵。

技术集成证据通过，独立完整运行态产品评审及原依赖验收仍未完成。 Original acceptance and human gates remain unmet. See [summary](../../../.harness/archive/2026-11/evidence/2026-11-16-ops-business-runtime-acceptance/summary.md) and [structured progress](../../../.harness/evidence/2026-11-17-ops-history-audit/continuation-ten-packages-progress-final-20261118.json).

## Result — History Closeout 2026-11-20

Manifest disposition: completed

业务运行时验收证据随 #195 落地；真实 provider/受限角色与维护者终验保留为 residual。

Residual finding/owner/successor/proof: .harness/evidence/2026-11-20-ops-board-complete/dispositions.json. Canonical references remain archive-aware. Historical body/evidence retained. Candidate/production/maintainer gates remain separate.

## Result — History Closeout 2026-11-20

Manifest disposition: completed

业务运行时验收证据随 #195 落地；真实 provider/受限角色与维护者终验保留为 residual。

Residual finding/owner/successor/proof: .harness/evidence/2026-11-20-ops-board-complete/dispositions.json. Canonical references remain archive-aware. Historical body/evidence retained. Candidate/production/maintainer gates remain separate.
