---
title: ArgoCD 归属隔离与真实状态收敛
doc_type: Remediation
layer: business
status: Archived
linked_contracts:
  - docs/PROJECT_INHERITANCE.md
updated_at: 2026-11-20
---

# Task Packet: 2026-11-16-ops-argocd-ownership-convergence

## Goal

ArgoCD 归属隔离与真实状态收敛

## Priority

P0；L2；扩展条件队列，不计首期核心退出。

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

- 以专项业务归属合同实现列表/Get/Sync/Delete 的 scope 校验，复用 base DataScope，不改共享中间件。
- Sync 接受不等于成功，状态来自实际 operation/health；在装配中启动有界 reconciler 并处理重启/乱序。
- 旧无归属记录设计显式迁移/隔离策略；不得以默认全量可见完成兼容。

### Out

- platform/system/* 通用实现与规则副本；共享问题 Base-first
- 未授权生产操作、规则豁免与风险接受
- 自动 exporter/file_sd/商业 License 等后续产品扩张

## Expected Files

### Create

- docs/harness/tasks/2026-11-16-ops-argocd-ownership-convergence.task.md
- .harness/tasks/2026-11-16-ops-argocd-ownership-convergence/manifest.json
- .harness/archive/2026-11/evidence/2026-11-16-ops-argocd-ownership-convergence/summary.md
- .harness/archive/2026-11/evidence/2026-11-16-ops-argocd-ownership-convergence/review.md
- .harness/archive/2026-11/evidence/2026-11-16-ops-argocd-ownership-convergence/commands.json

### Modify

- backend/modules/business/cicd/argocd
- backend/modules/business/opsinit/opsinit.go
- frontend/src/modules/business/cicd/argocd

### Do Not Touch

- ../pantheon-base/**
- foundation-release.lock.json
- backend/modules/system/**
- frontend/src/modules/system/**

## Implementation Notes

- Owner: business/cicd/argocd
- Minimum Complexity Rung: reuse
- 开始实现前重新读取当前候选与结构；本审查行号绑定旧基线，不能覆盖并发升级。
- Windows race 必须 PATH 前置 D:\msys64\mingw64\bin，CGO_ENABLED=1；普通 Go tests 不替代。
- 业务 mutation 的 API/权限/菜单/i18n/迁移/fixture/smoke 同步关闭；复用共享契约，禁止本地 override。
- 默认 planned，不开放本扩展时不阻塞首期；选择开放时相关 P0 成为必过。

## Structural Scope

- Affected Subgraph: business/cicd/argocd
- Boundary Crossings: 业务 owner capability / HTTP UI / 外部 provider；治理任务只关联任务与证据
- Risk Nodes: 目标身份、ownership、状态回写、引用、发布与证据真实性
- Graph Focus: backend/modules/business/cicd/argocd/argocd_service.go:360; backend/modules/business/cicd/argocd/argocd_service.go:482

## Execution Roles

- Implementer Posture: executor/Generator 在明确 owner 内实现；当前计划会话只改治理资料
- Reviewer Posture: 独立 code-reviewer/verifier 对照 In/Out 和运行证据；静态推断不得写成已复现

## Verification Plan

- go test -race -count=1 ./modules/business/cicd/...
- npm run lint
- npm run type-check

## Acceptance Criteria

- 受限部门角色不能读取/操作其他业务域应用；请求在 Argo 副作用前拒绝。
- 同步请求已接受但镜像拉取失败时，平台不提前显示 Succeeded/Synced。
- 成功、失败、Progressing、超时、外部删除及恢复自动收敛，凭据不回显。

## Dependencies

- none

## Linkage

- Task ID: 2026-11-16-ops-argocd-ownership-convergence
- OpenSpec Change: none
- Superpowers Plan: none
- Task Manifest: .harness/tasks/2026-11-16-ops-argocd-ownership-convergence/manifest.json
- Evidence Directory: .harness/archive/2026-11/evidence/2026-11-16-ops-argocd-ownership-convergence/
- Review File: .harness/archive/2026-11/evidence/2026-11-16-ops-argocd-ownership-convergence/review.md
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

in-progress；首期默认裁剪，启用时转为必验。

## 2026-11-18T07:31:44.466Z Continuation checkpoint

Status: in-progress. accepted sync保持 Running，实际operation+Synced+Healthy才成功；失联后仍有界本地过期收敛Unknown、近期任务保持pending、Stop取消真实探测。中央真实MySQL CGO/race：21顶层PASS/5.796s，含新3项。

Original acceptance and dependencies remain unchanged. Whole-package acceptance is incomplete; see [summary](../../../.harness/archive/2026-11/evidence/2026-11-16-ops-argocd-ownership-convergence/summary.md) and [progress receipt](../../../.harness/evidence/2026-11-17-ops-history-audit/continuation-ten-packages-progress-20261118.json).

## 2026-11-18T07:58:38.569Z Integration continuation

Status: in-progress. startup生命周期、3个E02 canonical key/handler和5locale已装配；17文件handoff保留Base shutdown/public异步授权边界。中央Argo3.720s、748源稳定窄MySQL CGO/race通过，ownership/legacy/UID/token/webhook/dual-write/timestamp及真实提供方验收仍OPEN。

Backend round3 COMMENT保持；startup/Helm独立窄CLEAR、整包WATCH（17＋4绑定hash核验），Base await/bind failure/public authority和原整包边界仍OPEN。 Original acceptance, dependencies and human gates remain unmet. See [summary](../../../.harness/archive/2026-11/evidence/2026-11-16-ops-argocd-ownership-convergence/summary.md) and [latest progress](../../../.harness/evidence/2026-11-17-ops-history-audit/continuation-ten-packages-progress-integration-20261118.json).

## 2026-11-19T00:11:15.272Z Verified continuation checkpoint

Status: in-progress. startup生命周期及3canonical key/handler/5locale已装配，窄Argo3.720s和完整当前748源race通过；现有backend round3 COMMENT保留。

Backend round3 COMMENT、startup窄CLEAR/整包WATCH不关闭原ownership/provider/人验收。 Original acceptance and human gates remain unmet. See [summary](../../../.harness/archive/2026-11/evidence/2026-11-16-ops-argocd-ownership-convergence/summary.md) and [structured progress](../../../.harness/evidence/2026-11-17-ops-history-audit/continuation-ten-packages-progress-final-20261118.json).

## Result — History Closeout 2026-11-20

Manifest disposition: completed

ArgoCD ownership/legacy 收敛实施 + candidate 证据随 #195 落地；真实 provider/受限角色验收保留为 residual。

Residual finding/owner/successor/proof: .harness/evidence/2026-11-20-ops-board-complete/dispositions.json. Canonical references remain archive-aware. Historical body/evidence retained. Candidate/production/maintainer gates remain separate.

## Result — History Closeout 2026-11-20

Manifest disposition: completed

ArgoCD ownership/legacy 收敛实施 + candidate 证据随 #195 落地；真实 provider/受限角色验收保留为 residual。

Residual finding/owner/successor/proof: .harness/evidence/2026-11-20-ops-board-complete/dispositions.json. Canonical references remain archive-aware. Historical body/evidence retained. Candidate/production/maintainer gates remain separate.
