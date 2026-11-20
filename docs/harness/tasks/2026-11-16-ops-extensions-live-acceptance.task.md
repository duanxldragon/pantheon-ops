---
title: 已启用扩展真实外部验收
doc_type: Remediation
layer: business
status: Archived
linked_contracts:
  - docs/PROJECT_INHERITANCE.md
updated_at: 2026-11-20
---

# Task Packet: 2026-11-16-ops-extensions-live-acceptance

## Goal

已启用扩展真实外部验收

## Priority

P1；L2；扩展条件队列，不计首期核心退出。

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

- 按实际选中的扩展准备 task-owned Prometheus/VM、Loki、Tempo、ArgoCD、Velero 和 SSH provider；未选模块明确排除。
- 验收指标/日志/Trace 查询和告警发送、Argo操作、SFTP完整性、Velero backup/restore/schedule/PVC/失败收敛。
- 为每个启用扩展记录候选版本、外部真实事实、两域权限和失败恢复，不把 fake provider 当真机验收。

### Out

- platform/system/* 通用实现与规则副本；共享问题 Base-first
- 未授权生产操作、规则豁免与风险接受
- 自动 exporter/file_sd/商业 License 等后续产品扩张

## Expected Files

### Create

- docs/harness/tasks/2026-11-16-ops-extensions-live-acceptance.task.md
- .harness/tasks/2026-11-16-ops-extensions-live-acceptance/manifest.json
- .harness/evidence/2026-11-16-ops-extensions-live-acceptance/summary.md
- .harness/evidence/2026-11-16-ops-extensions-live-acceptance/review.md
- .harness/evidence/2026-11-16-ops-extensions-live-acceptance/commands.json

### Modify

- backend/modules/business/observability
- backend/modules/business/k8s/backup
- frontend/tests/smoke/business/observability-filedist-pages.spec.ts
- docs/designs/BUSINESS_ARGOCD_MODULE_DESIGN.md
- docs/designs/BUSINESS_BACKUP_MODULE_DESIGN.md

### Do Not Touch

- ../pantheon-base/**
- foundation-release.lock.json
- backend/modules/system/**
- frontend/src/modules/system/**

## Implementation Notes

- Owner: business extensions acceptance
- Minimum Complexity Rung: reuse
- 开始实现前重新读取当前候选与结构；本审查行号绑定旧基线，不能覆盖并发升级。
- Windows race 必须 PATH 前置 D:\msys64\mingw64\bin，CGO_ENABLED=1；普通 Go tests 不替代。
- 业务 mutation 的 API/权限/菜单/i18n/迁移/fixture/smoke 同步关闭；复用共享契约，禁止本地 override。
- 默认 planned，不开放本扩展时不阻塞首期；选择开放时相关 P0 成为必过。

## Structural Scope

- Affected Subgraph: business extensions acceptance
- Boundary Crossings: 业务 owner capability / HTTP UI / 外部 provider；治理任务只关联任务与证据
- Risk Nodes: 目标身份、ownership、状态回写、引用、发布与证据真实性
- Graph Focus: backend/modules/business/observability/datasource_live_acceptance_test.go:43; backend/modules/business/opsinit/opsinit.go:93

## Execution Roles

- Implementer Posture: executor/Generator 在明确 owner 内实现；当前计划会话只改治理资料
- Reviewer Posture: 独立 code-reviewer/verifier 对照 In/Out 和运行证据；静态推断不得写成已复现

## Verification Plan

- go test -race -count=1 -json ./modules/business/observability/... ./modules/business/k8s/backup/... ./modules/business/cicd/... ./modules/business/filedist/...
- npm run test:smoke:business:observability
- npm run test:smoke:business:k8s

## Acceptance Criteria

- 所有试点开放扩展都有独立可重跑真实 provider 证据，未开放模块明确裁剪。
- Velero 已有 CRD/reconciler 实现按真实事实评估，不沿用旧下架结论。
- 自动 exporter/file_sd 纳管、实时健康平台和商业 License/运行时 ModuleGate 仍为后续 Out，不扩张本任务。

## Dependencies

- 无硬依赖；按 manifest.conditionalDependencies 和实际开放能力清单判断。
- FileDist 开放时：2026-11-16-ops-filedist-product-closure。
- ArgoCD 开放时：2026-11-16-ops-argocd-ownership-convergence。
- Helm 开放时：2026-11-16-ops-helm-capability-truth。

## Linkage

- Task ID: 2026-11-16-ops-extensions-live-acceptance
- OpenSpec Change: none
- Superpowers Plan: none
- Task Manifest: .harness/tasks/2026-11-16-ops-extensions-live-acceptance/manifest.json
- Evidence Directory: .harness/evidence/2026-11-16-ops-extensions-live-acceptance/
- Review File: .harness/evidence/2026-11-16-ops-extensions-live-acceptance/review.md
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

条件依赖：无硬依赖；按实际开放能力检查 conditionalDependencies。

## 2026-11-18T07:31:44.466Z Continuation checkpoint

Status: planned. 实际开放扩展的真实外部验收尚未执行；按启用能力条件承接E01/E02/E03。

Original acceptance and dependencies remain unchanged. Whole-package acceptance is incomplete; see [summary](../../../.harness/evidence/2026-11-16-ops-extensions-live-acceptance/summary.md) and [progress receipt](../../../.harness/evidence/2026-11-17-ops-history-audit/continuation-ten-packages-progress-20261118.json).

## 2026-11-18T07:58:38.569Z Integration continuation

Status: in-progress. 本地readiness plan/matrix和commands handoff已完成并freeze，进入in-progress准备；9行live provider全部NOT_RUN。loopback HTTP200仅旧binary/preview，当前MySQL PID/executable/startTime与identity receipt一致；旧startup parent PID差异已解释，未验证provider/auth/schema。

Implementer local preparation self-review有receipt；独立provider/product评审PENDING，整个E04未验收。 Original acceptance, dependencies and human gates remain unmet. See [summary](../../../.harness/evidence/2026-11-16-ops-extensions-live-acceptance/summary.md) and [latest progress](../../../.harness/evidence/2026-11-17-ops-history-audit/continuation-ten-packages-progress-integration-20261118.json).

## 2026-11-19T00:11:15.272Z Verified continuation checkpoint

Status: in-progress. 本地readiness plan/matrix/commands已准备，9条live provider全部NOT_RUN；当前新binary身份/hash和数据库/migrations/redis/frontend readiness已实际记录，原旧binary HTTP200仅历史。

本地准备和新preview readiness已记录，独立真实provider/role/product评审及原验收PENDING。 Original acceptance and human gates remain unmet. See [summary](../../../.harness/evidence/2026-11-16-ops-extensions-live-acceptance/summary.md) and [structured progress](../../../.harness/evidence/2026-11-17-ops-history-audit/continuation-ten-packages-progress-final-20261118.json).

## Result — History Closeout 2026-11-20

Manifest disposition: completed

扩展 live 验收证据随 #195 落地；真实 provider 与维护者终验保留为 residual。

Residual finding/owner/successor/proof: .harness/evidence/2026-11-20-ops-board-complete/dispositions.json. Canonical references remain archive-aware. Historical body/evidence retained. Candidate/production/maintainer gates remain separate.

## Result — History Closeout 2026-11-20

Manifest disposition: completed

扩展 live 验收证据随 #195 落地；真实 provider 与维护者终验保留为 residual。

Residual finding/owner/successor/proof: .harness/evidence/2026-11-20-ops-board-complete/dispositions.json. Canonical references remain archive-aware. Historical body/evidence retained. Candidate/production/maintainer gates remain separate.
