---
title: 试点交付包、安装回滚与状态收口
doc_type: Remediation
layer: business
status: Active
linked_contracts:
  - docs/PROJECT_INHERITANCE.md
updated_at: 2026-11-16
---

# Task Packet: 2026-11-16-ops-pilot-release-closeout

## Goal

试点交付包、安装回滚与状态收口

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

- 从通过 C05 的 clean candidate 输出 Ops 交付包、SHA-256、业务版本和锁定 foundation/迁移信息。
- 对隔离环境执行离线/目标安装方式、备份、升级、失败恢复及回滚，复用 Base runner 与发布契约。
- 把真实已交付、裁剪扩展和已接受风险写进准确发布说明，最终功能/视觉验收后才声明试点完成。

### Out

- platform/system/* 通用实现与规则副本；共享问题 Base-first
- 未授权生产操作、规则豁免与风险接受
- 自动 exporter/file_sd/商业 License 等后续产品扩张

## Expected Files

### Create

- docs/harness/tasks/2026-11-16-ops-pilot-release-closeout.task.md
- .harness/tasks/2026-11-16-ops-pilot-release-closeout/manifest.json
- .harness/evidence/2026-11-16-ops-pilot-release-closeout/summary.md
- .harness/evidence/2026-11-16-ops-pilot-release-closeout/review.md
- .harness/evidence/2026-11-16-ops-pilot-release-closeout/commands.json

### Modify

- README.md
- README.en.md
- docs/README.md
- docs/README.en.md
- docs/V1_RELEASE_CHECKLIST.md
- docker-compose.yml
- releases

### Do Not Touch

- ../pantheon-base/**
- foundation-release.lock.json
- backend/modules/system/**
- frontend/src/modules/system/**

## Implementation Notes

- Owner: business-consumer release
- Minimum Complexity Rung: reuse
- 开始实现前重新读取当前候选与结构；本审查行号绑定旧基线，不能覆盖并发升级。
- Windows race 必须 PATH 前置 D:\msys64\mingw64\bin，CGO_ENABLED=1；普通 Go tests 不替代。
- 业务 mutation 的 API/权限/菜单/i18n/迁移/fixture/smoke 同步关闭；复用共享契约，禁止本地 override。
- 依赖未完成不执行后续；状态只按实际 evidence 更新。

## Structural Scope

- Affected Subgraph: business-consumer release
- Boundary Crossings: 业务 owner capability / HTTP UI / 外部 provider；治理任务只关联任务与证据
- Risk Nodes: 目标身份、ownership、状态回写、引用、发布与证据真实性
- Graph Focus: docs/V1_RELEASE_CHECKLIST.md:62; README.md:5

## Execution Roles

- Implementer Posture: executor/Generator 在明确 owner 内实现；当前计划会话只改治理资料
- Reviewer Posture: 独立 code-reviewer/verifier 对照 In/Out 和运行证据；静态推断不得写成已复现

## Verification Plan

- npm run check:business-overlay
- npm run check:shared-drift
- npm run rebuild:from-base -- --target .tmp/pilot-release-candidate
- node scripts/harness/check-evidence.mjs --strict
- node scripts/harness/check-review.mjs --strict

## Acceptance Criteria

- 独立执行者从 clean checkout/交付包按文档完成安装、升级及回滚，校验和与候选一致。
- C01-C05 完成且本包交付/回滚验收通过，P0 零遗留；P1 延期须有维护者决定、owner、截止日期及补偿控制。
- 业务包不能仅是 Base release；不以 GitHub merge、测试全绿或旧 GA 文案替代业务验收。
- 候选开放扩展通过 E04；不开放扩展的菜单、路由/权限、可见能力与说明已裁剪并验证。

## Dependencies

- 2026-11-16-ops-business-runtime-acceptance

## Linkage

- Task ID: 2026-11-16-ops-pilot-release-closeout
- OpenSpec Change: none
- Superpowers Plan: none
- Task Manifest: .harness/tasks/2026-11-16-ops-pilot-release-closeout/manifest.json
- Evidence Directory: .harness/evidence/2026-11-16-ops-pilot-release-closeout/
- Review File: .harness/evidence/2026-11-16-ops-pilot-release-closeout/review.md
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

条件依赖：试点开放任何扩展能力，则 E04 必须完成；全部裁剪时记录并验证候选菜单/路由/权限/可见能力与说明。

## 2026-11-18T07:31:44.466Z Continuation checkpoint

Status: in-progress. 本地交付计划/checklist已覆盖4条原标准；8项依赖交付操作均 not-run。准备不表示已经执行安装/升级/回滚。

Original acceptance and dependencies remain unchanged. Whole-package acceptance is incomplete; see [summary](../../../.harness/evidence/2026-11-16-ops-pilot-release-closeout/summary.md) and [progress receipt](../../../.harness/evidence/2026-11-17-ops-history-audit/continuation-ten-packages-progress-20261118.json).

## 2026-11-19T00:11:15.272Z Verified continuation checkpoint

Status: in-progress. 本地交付计划/checklist保留4条原标准和新增明确交付标准；8项实际交付操作not-run。正式暂存重建、source-bound编译与预览重启已记录，不等于独立安装/升级/回滚。

交付准备独立评审及独立执行者安装/升级/回滚和最终维护者验收PENDING。 Original acceptance and human gates remain unmet. See [summary](../../../.harness/evidence/2026-11-16-ops-pilot-release-closeout/summary.md) and [structured progress](../../../.harness/evidence/2026-11-17-ops-history-audit/continuation-ten-packages-progress-final-20261118.json).
