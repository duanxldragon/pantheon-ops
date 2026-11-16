---
title: Helm 能力口径与真实 Chart 安装边界
doc_type: Remediation
layer: business
status: Active
linked_contracts:
  - docs/PROJECT_INHERITANCE.md
updated_at: 2026-11-16
---

# Task Packet: 2026-11-16-ops-helm-capability-truth

## Goal

Helm 能力口径与真实 Chart 安装边界

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

- 先以最小复杂度判断现有登记能力是否应隐藏或明确标为登记，去除数据库插入即 deployed 的误导。
- 如扩展范围明确包含 Chart 安装，复用适当已有工具/依赖建立 install/upgrade/rollback/uninstall 真执行及观测；否则明确延期。

### Out

- platform/system/* 通用实现与规则副本；共享问题 Base-first
- 未授权生产操作、规则豁免与风险接受
- 自动 exporter/file_sd/商业 License 等后续产品扩张

## Expected Files

### Create

- docs/harness/tasks/2026-11-16-ops-helm-capability-truth.task.md
- .harness/tasks/2026-11-16-ops-helm-capability-truth/manifest.json
- .harness/evidence/2026-11-16-ops-helm-capability-truth/summary.md
- .harness/evidence/2026-11-16-ops-helm-capability-truth/review.md
- .harness/evidence/2026-11-16-ops-helm-capability-truth/commands.json

### Modify

- backend/modules/business/k8s/helm/helm.go
- frontend/src/modules/business/k8s
- README.md
- README.en.md

### Do Not Touch

- ../pantheon-base/**
- foundation-release.lock.json
- backend/modules/system/**
- frontend/src/modules/system/**

## Implementation Notes

- Owner: business/k8s/helm
- Minimum Complexity Rung: reuse
- 开始实现前重新读取当前候选与结构；本审查行号绑定旧基线，不能覆盖并发升级。
- Windows race 必须 PATH 前置 D:\msys64\mingw64\bin，CGO_ENABLED=1；普通 Go tests 不替代。
- 业务 mutation 的 API/权限/菜单/i18n/迁移/fixture/smoke 同步关闭；复用共享契约，禁止本地 override。
- 默认 planned，不开放本扩展时不阻塞首期；选择开放时相关 P0 成为必过。

## Structural Scope

- Affected Subgraph: business/k8s/helm
- Boundary Crossings: 业务 owner capability / HTTP UI / 外部 provider；治理任务只关联任务与证据
- Risk Nodes: 目标身份、ownership、状态回写、引用、发布与证据真实性
- Graph Focus: backend/modules/business/k8s/helm/helm.go:109

## Execution Roles

- Implementer Posture: executor/Generator 在明确 owner 内实现；当前计划会话只改治理资料
- Reviewer Posture: 独立 code-reviewer/verifier 对照 In/Out 和运行证据；静态推断不得写成已复现

## Verification Plan

- go test -race -count=1 ./modules/business/k8s/...
- npm run type-check
- npm run test:smoke:business:k8s

## Acceptance Criteria

- 未真正安装 Chart 的记录不显示已部署，不用 K8s 镜像 Release 为 Helm 能力背书。
- 若交付 Chart 执行，版本、values、权限、超时/冲突、历史和回滚具备真实集群证据；若延期，有可见能力裁剪和风险决定。

## Dependencies

- none

## Linkage

- Task ID: 2026-11-16-ops-helm-capability-truth
- OpenSpec Change: none
- Superpowers Plan: none
- Task Manifest: .harness/tasks/2026-11-16-ops-helm-capability-truth/manifest.json
- Evidence Directory: .harness/evidence/2026-11-16-ops-helm-capability-truth/
- Review File: .harness/evidence/2026-11-16-ops-helm-capability-truth/review.md
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

planned；首期默认裁剪，启用时转为必验。
