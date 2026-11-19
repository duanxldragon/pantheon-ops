---
title: 页面服务实例到部署结果闭环
doc_type: Remediation
layer: business
status: Active
linked_contracts:
  - docs/PROJECT_INHERITANCE.md
updated_at: 2026-11-16
---

# Task Packet: 2026-11-16-ops-service-deploy-user-flow

## Goal

页面服务实例到部署结果闭环

## Priority

P1；L2；首期核心/治理范围。

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

- 复用现有部署表单与预览，选择可见服务实例并展示匹配的目标快照；提交关联字段。
- 保留不关联实例的显式独立部署入口，避免现有部署兼容性退化。
- 通过浏览器完成创建目录、绑定实例、创建/预览/执行部署和结果回读，不用 API 预造任务替代关键 UI 动作。

### Out

- platform/system/* 通用实现与规则副本；共享问题 Base-first
- 未授权生产操作、规则豁免与风险接受
- 自动 exporter/file_sd/商业 License 等后续产品扩张

## Expected Files

### Create

- docs/harness/tasks/2026-11-16-ops-service-deploy-user-flow.task.md
- .harness/tasks/2026-11-16-ops-service-deploy-user-flow/manifest.json
- .harness/evidence/2026-11-16-ops-service-deploy-user-flow/summary.md
- .harness/evidence/2026-11-16-ops-service-deploy-user-flow/review.md
- .harness/evidence/2026-11-16-ops-service-deploy-user-flow/commands.json

### Modify

- frontend/src/modules/business/deploy/api.ts
- frontend/src/modules/business/deploy/task/DeployTaskList.tsx
- frontend/src/modules/business/service/ServiceCatalog.tsx
- frontend/tests/smoke/business/deploy/deploy-pages.spec.ts
- frontend/tests/smoke/business/service/service-pages.spec.ts

### Do Not Touch

- ../pantheon-base/**
- foundation-release.lock.json
- backend/modules/system/**
- frontend/src/modules/system/**

## Implementation Notes

- Owner: business/deploy + business/service UI
- Minimum Complexity Rung: reuse
- 开始实现前重新读取当前候选与结构；本审查行号绑定旧基线，不能覆盖并发升级。
- Windows race 必须 PATH 前置 D:\msys64\mingw64\bin，CGO_ENABLED=1；普通 Go tests 不替代。
- 业务 mutation 的 API/权限/菜单/i18n/迁移/fixture/smoke 同步关闭；复用共享契约，禁止本地 override。
- 依赖未完成不执行后续；状态只按实际 evidence 更新。

## Structural Scope

- Affected Subgraph: business/deploy + business/service UI
- Boundary Crossings: 业务 owner capability / HTTP UI / 外部 provider；治理任务只关联任务与证据
- Risk Nodes: 目标身份、ownership、状态回写、引用、发布与证据真实性
- Graph Focus: frontend/src/modules/business/deploy/api.ts:238; frontend/src/modules/business/deploy/task/DeployTaskList.tsx:478

## Execution Roles

- Implementer Posture: executor/Generator 在明确 owner 内实现；当前计划会话只改治理资料
- Reviewer Posture: 独立 code-reviewer/verifier 对照 In/Out 和运行证据；静态推断不得写成已复现

## Verification Plan

- npm run lint
- npm run type-check
- npm run test:smoke:business:deploy
- npm run test:smoke:business:service

## Acceptance Criteria

- UI 提交的关联字段、预览目标与 C01 后端校验一致。
- 成功/失败部署后服务目录展示对应实例终态，日志及审计可按任务追溯。
- 桌面/窄屏及 loading/empty/error/forbidden/submitting 状态有当前 revision 渲染证据。

## Dependencies

- 2026-11-16-ops-service-deploy-integrity

## Linkage

- Task ID: 2026-11-16-ops-service-deploy-user-flow
- OpenSpec Change: none
- Superpowers Plan: none
- Task Manifest: .harness/tasks/2026-11-16-ops-service-deploy-user-flow/manifest.json
- Evidence Directory: .harness/evidence/2026-11-16-ops-service-deploy-user-flow/
- Review File: .harness/evidence/2026-11-16-ops-service-deploy-user-flow/review.md
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

planned；只在依赖与实际验收完成后转 completed。

## 2026-11-18T07:31:44.466Z Continuation checkpoint

Status: planned. 等待 C01 合同及依赖验收；没有本包浏览器部署全程的当前候选证据。

Original acceptance and dependencies remain unchanged. Whole-package acceptance is incomplete; see [summary](../../../.harness/evidence/2026-11-16-ops-service-deploy-user-flow/summary.md) and [progress receipt](../../../.harness/evidence/2026-11-17-ops-history-audit/continuation-ten-packages-progress-20261118.json).

## 2026-11-19T00:11:15.272Z Verified continuation checkpoint

Status: planned. 保持planned，等待C01原合同及依赖验收；全局frontend构建通过，尚无本包完整浏览器部署全程验收。

独立整包产品评审PENDING；全局构建不改变C01依赖。 Original acceptance and human gates remain unmet. See [summary](../../../.harness/evidence/2026-11-16-ops-service-deploy-user-flow/summary.md) and [structured progress](../../../.harness/evidence/2026-11-17-ops-history-audit/continuation-ten-packages-progress-final-20261118.json).
