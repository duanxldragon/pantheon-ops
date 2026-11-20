---
title: 服务目录维护、分页与受限角色
doc_type: Remediation
layer: business
status: Archived
linked_contracts:
  - docs/PROJECT_INHERITANCE.md
updated_at: 2026-11-20
---

# Task Packet: 2026-11-16-ops-service-catalog-usability

## Goal

服务目录维护、分页与受限角色

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

- 复用已有 PUT 与共享表单补齐范围内应用/服务/实例编辑，服务端分页与目标选项按需查询。
- 服务目录读取与可选目标查询分离；缺 CMDB/K8s 操作权限不能使已授权目录整页失败。
- 按业务域过滤并结构化选择 K8s namespace/workload；权限失败和目标失效可解释。

### Out

- platform/system/* 通用实现与规则副本；共享问题 Base-first
- 未授权生产操作、规则豁免与风险接受
- 自动 exporter/file_sd/商业 License 等后续产品扩张

## Expected Files

### Create

- docs/harness/tasks/2026-11-16-ops-service-catalog-usability.task.md
- .harness/tasks/2026-11-16-ops-service-catalog-usability/manifest.json
- .harness/evidence/2026-11-16-ops-service-catalog-usability/summary.md
- .harness/evidence/2026-11-16-ops-service-catalog-usability/review.md
- .harness/evidence/2026-11-16-ops-service-catalog-usability/commands.json

### Modify

- frontend/src/modules/business/service/ServiceCatalog.tsx
- frontend/src/modules/business/service/api.ts
- frontend/tests/smoke/business/service/service-pages.spec.ts
- backend/modules/business/service/dto.go
- backend/modules/business/service/service.go
- backend/modules/business/service/handler.go
- backend/modules/business/service/seed.go

### Do Not Touch

- ../pantheon-base/**
- foundation-release.lock.json
- backend/modules/system/**
- frontend/src/modules/system/**

## Implementation Notes

- Owner: business/service owner + UI
- Minimum Complexity Rung: reuse
- 开始实现前重新读取当前候选与结构；本审查行号绑定旧基线，不能覆盖并发升级。
- Windows race 必须 PATH 前置 D:\msys64\mingw64\bin，CGO_ENABLED=1；普通 Go tests 不替代。
- 业务 mutation 的 API/权限/菜单/i18n/迁移/fixture/smoke 同步关闭；复用共享契约，禁止本地 override。
- 依赖未完成不执行后续；状态只按实际 evidence 更新。

## Structural Scope

- Affected Subgraph: business/service UI
- Boundary Crossings: 业务 owner capability / HTTP UI / 外部 provider；治理任务只关联任务与证据
- Risk Nodes: 目标身份、ownership、状态回写、引用、发布与证据真实性
- Graph Focus: frontend/src/modules/business/service/ServiceCatalog.tsx:170; frontend/src/modules/business/service/ServiceCatalog.tsx:322

## Execution Roles

- Implementer Posture: executor/Generator 在明确 owner 内实现；当前计划会话只改治理资料
- Reviewer Posture: 独立 code-reviewer/verifier 对照 In/Out 和运行证据；静态推断不得写成已复现

## Verification Plan

- npm run lint
- npm run type-check
- npm run test:smoke:business:service

## Acceptance Criteria

- 101 条以上对象可分页查询、编辑，汇总使用服务端总数。
- 服务只读角色无需 CMDB/K8s 列表权限也可读目录；操作按钮按授权呈现，后端仍拒绝越权。
- 版本/目标编辑走 owner 校验及冲突语义；未默认要求新增实时健康探测。

## Dependencies

- none

## Linkage

- Task ID: 2026-11-16-ops-service-catalog-usability
- OpenSpec Change: none
- Superpowers Plan: none
- Task Manifest: .harness/tasks/2026-11-16-ops-service-catalog-usability/manifest.json
- Evidence Directory: .harness/evidence/2026-11-16-ops-service-catalog-usability/
- Review File: .harness/evidence/2026-11-16-ops-service-catalog-usability/review.md
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

## C03 owner contract continuation

The original version/target edit acceptance requires owner-side validation and conflict semantics. Exact additional business owner files: backend/modules/business/service/dto.go, backend/modules/business/service/service.go, backend/modules/business/service/handler.go, backend/modules/business/service/seed.go; focused real-MySQL regression: backend/modules/business/service/instance_update_test.go. Five canonical v12locale sources and the business error appendix synchronize existing error keys. Plan: .harness/tasks/2026-11-16-ops-service-catalog-usability/continuation-plan.md.

HTTP edits require expectedLifecycleVersion (zero is valid), reject currentVersion publication, and accept an optional atomic target replacement. Internal compatibility keeps existing owner calls. Exact CAS, request context, target identity/scope/locator and active-reference refusal must be dynamically verified. Typed target lifetime coordination and full K8s NamespaceBinding/environment/action validation remain explicitly open; this contract checkpoint does not establish completion or waive them.

## 2026-11-17 owner continuation scope

Original In/Out and acceptance are retained. Additional modified owner files: backend/modules/business/service/module.go, backend/modules/business/k8s/capability.go. Additional created route/provider regression files: backend/modules/business/service/permission_routes.go, backend/modules/business/service/catalog_review_test.go, backend/modules/business/k8s/capability_error_test.go. Plan: .harness/tasks/2026-11-16-ops-service-catalog-usability/continuation-plan.md. Real red/green and remaining runtime/provider gaps are recorded in .harness/evidence/2026-11-17-ops-history-audit/; no archive or completion is implied.

## 2026-11-18T07:31:44.466Z Continuation checkpoint

Status: in-progress. 表单重新激活立即清空旧选项/错误并进入 loading；回归 RED 2失败/5通过，GREEN 7/7；scoped lint和隔离 strict typecheck通过。

Original acceptance and dependencies remain unchanged. Whole-package acceptance is incomplete; see [summary](../../../.harness/evidence/2026-11-16-ops-service-catalog-usability/summary.md) and [progress receipt](../../../.harness/evidence/2026-11-17-ops-history-audit/continuation-ten-packages-progress-20261118.json).

## 2026-11-18T07:58:38.569Z Integration continuation

Status: in-progress. 常规Vitest配置已发现新useCatalogOptions测试并通过7/7、0skip/1.71s；精确business-overlay路径与本地overlay检查通过。完整app/build/render、backend target lifetime及真实provider/角色仍未验收。

常规unit/overlay集成有实际receipt；独立完整产品验收仍PENDING，历史activation和owner窄评审保留。 Original acceptance, dependencies and human gates remain unmet. See [summary](../../../.harness/evidence/2026-11-16-ops-service-catalog-usability/summary.md) and [latest progress](../../../.harness/evidence/2026-11-17-ops-history-audit/continuation-ten-packages-progress-integration-20261118.json).

## 2026-11-19T00:11:15.272Z Verified continuation checkpoint

Status: in-progress. 当前C03与E01两个canonical suite集中Vitest共35PASS/8.68s，frontend lint/type/build均通过；精确overlay路径、正式重建与1242源等价通过。当前桌面/手机render2/2通过、root实际查看editor/stale4图，business data mock；额外live admin lifecycle单行中文期望修后1PASS/0skip/5.7s、finally cleanup全部通过，executor查看主机#533截图；原错误fixture失败和两个未改transitive strict TypeScript诊断保留。

原activation/owner窄评审保留；整包真实provider/角色和维护者验收未完成。 Original acceptance and human gates remain unmet. See [summary](../../../.harness/evidence/2026-11-16-ops-service-catalog-usability/summary.md) and [structured progress](../../../.harness/evidence/2026-11-17-ops-history-audit/continuation-ten-packages-progress-final-20261118.json).

## Result — History Closeout 2026-11-20

Manifest disposition: completed

服务目录可用性验收候选证据（62 文件）随 #195 落地；真实角色/UI 渲染与维护者终验保留为 residual。

Residual finding/owner/successor/proof: .harness/evidence/2026-11-20-ops-board-complete/dispositions.json. Canonical references remain archive-aware. Historical body/evidence retained. Candidate/production/maintainer gates remain separate.

## Result — History Closeout 2026-11-20

Manifest disposition: completed

服务目录可用性验收候选证据（62 文件）随 #195 落地；真实角色/UI 渲染与维护者终验保留为 residual。

Residual finding/owner/successor/proof: .harness/evidence/2026-11-20-ops-board-complete/dispositions.json. Canonical references remain archive-aware. Historical body/evidence retained. Candidate/production/maintainer gates remain separate.
