---
title: Kubernetes Service 编辑与并发保护
doc_type: Remediation
layer: business
status: Archived
linked_contracts:
  - docs/PROJECT_INHERITANCE.md
updated_at: 2026-11-20
---

# Task Packet: 2026-11-16-ops-k8s-service-mutations

## Goal

Kubernetes Service 编辑与并发保护

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

- 复用 K8s 客户端与 ownership/namespace guard，补 selector/ports 更新及 API/UI 权限。
- 携带并校验 resourceVersion；删除使用适用的 UID/resourceVersion precondition，拒绝外部更新后的陈旧操作。
- 同步 API、菜单权限/i18n、错误语义和 smoke；不把镜像 Release 当 Helm Chart 安装。

### Out

- platform/system/* 通用实现与规则副本；共享问题 Base-first
- 未授权生产操作、规则豁免与风险接受
- 自动 exporter/file_sd/商业 License 等后续产品扩张

## Expected Files

### Create

- docs/harness/tasks/2026-11-16-ops-k8s-service-mutations.task.md
- .harness/tasks/2026-11-16-ops-k8s-service-mutations/manifest.json
- .harness/archive/2026-11/evidence/2026-11-16-ops-k8s-service-mutations/summary.md
- .harness/archive/2026-11/evidence/2026-11-16-ops-k8s-service-mutations/review.md
- .harness/archive/2026-11/evidence/2026-11-16-ops-k8s-service-mutations/commands.json

### Modify

- backend/modules/business/k8s/service/handler.go
- backend/modules/business/k8s/service/service.go
- frontend/src/modules/business/k8s/service
- frontend/tests/smoke/business/k8s/k8s-pages.spec.ts

### Do Not Touch

- ../pantheon-base/**
- foundation-release.lock.json
- backend/modules/system/**
- frontend/src/modules/system/**

## Implementation Notes

- Owner: business/k8s/service
- Minimum Complexity Rung: reuse
- 开始实现前重新读取当前候选与结构；本审查行号绑定旧基线，不能覆盖并发升级。
- Windows race 必须 PATH 前置 D:\msys64\mingw64\bin，CGO_ENABLED=1；普通 Go tests 不替代。
- 业务 mutation 的 API/权限/菜单/i18n/迁移/fixture/smoke 同步关闭；复用共享契约，禁止本地 override。
- 依赖未完成不执行后续；状态只按实际 evidence 更新。

## Structural Scope

- Affected Subgraph: business/k8s/service
- Boundary Crossings: 业务 owner capability / HTTP UI / 外部 provider；治理任务只关联任务与证据
- Risk Nodes: 目标身份、ownership、状态回写、引用、发布与证据真实性
- Graph Focus: docs/designs/BUSINESS_K8S_MODULE_DESIGN.md:42; backend/modules/business/k8s/service/service.go:177

## Execution Roles

- Implementer Posture: executor/Generator 在明确 owner 内实现；当前计划会话只改治理资料
- Reviewer Posture: 独立 code-reviewer/verifier 对照 In/Out 和运行证据；静态推断不得写成已复现

## Verification Plan

- go test -race -count=1 ./modules/business/k8s/...
- npm run test:smoke:business:k8s

## Acceptance Criteria

- 已绑定 namespace 中有权限用户能创建、读取、编辑、删除 Service。
- 跨域/未绑定/缺操作权限在 Kubernetes mutation 前失败。
- 平台读取后 kubectl 修改，平台陈旧更新/删除返回冲突且对象未被覆盖或删错。

## Dependencies

- none

## Linkage

- Task ID: 2026-11-16-ops-k8s-service-mutations
- OpenSpec Change: none
- Superpowers Plan: none
- Task Manifest: .harness/tasks/2026-11-16-ops-k8s-service-mutations/manifest.json
- Evidence Directory: .harness/archive/2026-11/evidence/2026-11-16-ops-k8s-service-mutations/
- Review File: .harness/archive/2026-11/evidence/2026-11-16-ops-k8s-service-mutations/review.md
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

## 2026-11-17 owner continuation scope

Original In/Out and acceptance are retained. Additional modified owner files: backend/modules/business/k8s/namespace/namespace_service.go, backend/modules/business/k8s/cluster/cluster_service.go, backend/modules/business/k8s/k8s.go, backend/modules/business/k8s/k8s_seed.go. Additional created route/provider regression files: backend/modules/business/k8s/permission_routes.go, backend/modules/business/k8s/permission_routes_test.go, backend/modules/business/k8s/service/mutation_contract_test.go. Plan: .harness/tasks/2026-11-16-ops-k8s-service-mutations/continuation-plan.md. Real red/green and remaining runtime/provider gaps are recorded in .harness/evidence/2026-11-17-ops-history-audit/; no archive or completion is implied.

## 2026-11-18T07:31:44.466Z Continuation checkpoint

Status: in-progress. Cluster row→同事务 credential read→Binding row→bounded provider 锁协议实际 MySQL/race通过；取消 GREEN02 1.533s、扩展准入 GREEN02 4.837s，均744文件运行内稳定。原取消 fixture GREEN01 失败保留。

Original acceptance and dependencies remain unchanged. Whole-package acceptance is incomplete; see [summary](../../../.harness/archive/2026-11/evidence/2026-11-16-ops-k8s-service-mutations/summary.md) and [progress receipt](../../../.harness/evidence/2026-11-17-ops-history-audit/continuation-ten-packages-progress-20261118.json).

## 2026-11-19T00:11:15.272Z Verified continuation checkpoint

Status: in-progress. Cluster→同事务credential→Binding→bounded provider锁与取消窄回归已通过，当前完整748源race亦通过；原取消GREEN01失败保留。当前UI contract13/13通过、root实际查看桌面/手机ports/error4图，business/role mock。

架构round2窄CLEAR/整包WATCH保留；全部业务角色/真实CRUD与维护者验收仍OPEN。 Original acceptance and human gates remain unmet. See [summary](../../../.harness/archive/2026-11/evidence/2026-11-16-ops-k8s-service-mutations/summary.md) and [structured progress](../../../.harness/evidence/2026-11-17-ops-history-audit/continuation-ten-packages-progress-final-20261118.json).

## Result — History Closeout 2026-11-20

Manifest disposition: completed

K8s 服务变更候选实施 + 28 证据文件随 #195 落地；真实集群/受限角色验收保留为 residual。

Residual finding/owner/successor/proof: .harness/evidence/2026-11-20-ops-board-complete/dispositions.json. Canonical references remain archive-aware. Historical body/evidence retained. Candidate/production/maintainer gates remain separate.

## Result — History Closeout 2026-11-20

Manifest disposition: completed

K8s 服务变更候选实施 + 28 证据文件随 #195 落地；真实集群/受限角色验收保留为 residual。

Residual finding/owner/successor/proof: .harness/evidence/2026-11-20-ops-board-complete/dispositions.json. Canonical references remain archive-aware. Historical body/evidence retained. Candidate/production/maintainer gates remain separate.
