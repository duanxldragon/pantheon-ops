---
title: 业务审查任务拆分、计划更新与完成任务归档
doc_type: Remediation
layer: business
status: Archived
linked_contracts:
  - docs/PROJECT_INHERITANCE.md
updated_at: 2026-11-16
---

# Task Packet: 2026-11-16-ops-readiness-plan-closeout

## Goal

落地本次审查计划和任务包，证据驱动归档已完成历史任务，保留开放业务及继承 gap。

## Priority

P1；L1；首期核心/治理范围。

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

- 任务包、计划/入口更新和 archive-aware 归档
- PR183 已合并事实补记；原始日志内容保留
- 新菜单运行未验收保持进行中

### Out

- platform/system/* 通用实现与规则副本；共享问题 Base-first
- 未授权生产操作、规则豁免与风险接受
- 自动 exporter/file_sd/商业 License 等后续产品扩张

## Expected Files

### Create

- docs/harness/tasks/2026-11-16-ops-readiness-plan-closeout.task.md
- .harness/tasks/2026-11-16-ops-readiness-plan-closeout/manifest.json
- .harness/evidence/2026-11-16-ops-readiness-plan-closeout/summary.md
- .harness/evidence/2026-11-16-ops-readiness-plan-closeout/review.md
- .harness/evidence/2026-11-16-ops-readiness-plan-closeout/commands.json

### Modify

- docs/README.md
- docs/README.en.md
- docs/harness/tasks/README.md
- docs/designs/BUSINESS_FUNCTIONAL_DESIGN.md
- docs/designs/BUSINESS_DEVELOPMENT_PLAN.md
- docs/V1_RELEASE_CHECKLIST.md
- .harness/ARCHIVE.md
- .harness/STATUS.md

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

- node scripts/harness/check-task-packet.mjs <本次任务包>
- npm run check:docs-frontmatter
- npm run check:task-packet-template
- node scripts/harness/check-evidence.mjs --strict <本次 commands.json>
- node scripts/harness/check-review.mjs --strict <本次 review.md>
- 归档 SHA-256 比对与 archive-aware 引用解析
- 完整历史门禁与 baseline 比对

## Acceptance Criteria

- 10 个子任务有 owner/依赖/范围/验收与证据链接，计划不声明业务完成。
- 12 个已完成任务成对归档，所有原始 evidence 字节相同；PR183 closeout 仅追加事实。
- 本次范围门禁通过，历史失败与前后差异显式记录，已有锁文件改动保持字节一致。

## Dependencies

- none

## Linkage

- Task ID: 2026-11-16-ops-readiness-plan-closeout
- OpenSpec Change: none
- Superpowers Plan: none
- Task Manifest: .harness/tasks/2026-11-16-ops-readiness-plan-closeout/manifest.json
- Evidence Directory: .harness/evidence/2026-11-16-ops-readiness-plan-closeout/
- Review File: .harness/evidence/2026-11-16-ops-readiness-plan-closeout/review.md
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
- [x] Verification run or exception recorded
- [x] Evidence saved or summarized
- [x] Review completed

## Status

completed；仅治理范围关闭，业务父包与10个子包仍planned。

## 本轮同步范围

- 更新中英文 README/docs 索引、业务功能现状、历史计划和检查点的 successor 链接。
- 同步已提交 Base v0.18.9 的继承文档锚点；不修改锁或装配源码。
- 菜单任务保持 in-progress；非日期 taskId 的旧文档别名保持 Superseded，机器任务与证据真实归档。
