---
title: Define Base operational workbench shared contracts
doc_type: Design
layer: platform
status: Active
updated_at: 2026-08-31
linked_contracts:
  - docs/designs/OPERATIONAL_WORKBENCH_COMPONENTS_DESIGN.md
---

# Task Packet: 2026-08-31-base-operational-workbench-design

## Goal

将外部成熟运维交互模式转化为 Pantheon Base 自有共享合同，并拆成可独立实施、验证和发布的 Base-first 工作包。

## Priority

`high`

## Estimated Complexity

`epic`

## Primary Layer

platform

## Workspace Context

- Target Repository: `pantheon-base`
- Repository Role: `foundation-source`
- Upstream Dependencies: `pantheon-harness`
- Downstream Consumers: `pantheon-ops`
- Sync Expectation: `deferred`
- Release Requirement: `foundation-release`

## Dependency Layers

- method-source: `pantheon-harness`
- business-consumer: `pantheon-ops`

## Dependencies

- blockedBy: `none`
- blocks: `2026-08-31-ops-operational-workbench-design` implementation batches that consume shared components

## Harness Profile

- Template: ui-heavy-product
- Overlay: pantheon-base
- Quality Profile: ui-runtime
- Portable Failure Class: architecture-drift
- Owner Layer: consumer-repository
- Coverage Dimensions:
  - behaviour
  - maintainability
  - architecture-fitness
  - runtime-quality
  - method-health

## Contract Anchors

- `AGENTS.md`
- `DESIGN.md`
- `docs/designs/OPERATIONAL_WORKBENCH_COMPONENTS_DESIGN.md`
- `docs/designs/FRONTEND_UI_SPEC.md`
- `docs/designs/FRONTEND_PAGE_TEMPLATES.md`
- `docs/designs/ACCESSIBILITY.md`
- `docs/designs/THEME_TOKENS_REFERENCE.md`

## Scope

### In

- 记录当前 UI 证据与外部参考的适用边界。
- 定义 sticky 表单动作、表格工作视图、日志、Diff、条件、上下文选择和步骤轨共享合同。
- 定义 Dashboard 语义槽、文案合同与视觉回归矩阵。
- 在 `.harness` 中拆分 B1-B5 实施卡并声明 Base -> Ops 交付顺序。

### Out

- 不修改 `backend/`、`frontend/` 或数据库。
- 不引入 BK Design 依赖或复制其视觉样式。
- 不实现任何 `business/*` 业务组合。
- 不发布 foundation release，不更新 Ops consumer lock。

## Assumptions and Open Questions

- Confirmed Facts: Base 已有 Arco、token、页面骨架与有限 visual baseline；Ops 是 foundation consumer。
- Working Assumptions: 后续实施继续使用现有前端依赖和用户偏好基础设施。
- Open Questions: 保存视图的服务端同步是否需要跨设备，留到 B2 实施前由维护者确定。

## Structural Scope

- Affected Subgraph: `Base design contracts -> shared frontend components -> foundation release -> Ops business composition`
- Boundary Crossings: `base -> ops`
- Risk Nodes: `shared component API, preference persistence, dashboard registry, visual baseline`
- Graph Focus: `hub-check`

## Expected Files

### Create

- `docs/designs/OPERATIONAL_WORKBENCH_COMPONENTS_DESIGN.md`
- `docs/harness/tasks/2026-08-31-base-operational-workbench-design.task.md`
- `.harness/tasks/2026-08-31-base-operational-workbench-design/**`
- `config/ui-quality-gate.json`
- `scripts/harness/check-ui-quality-gate.mjs`
- `tests/scripts/check-ui-quality-gate.test.mjs`

### Modify

- `package.json`
- `.github/workflows/quality.yml`
- `tests/scripts/quality-workflow.test.mjs`
- `frontend/playwright.visual.config.ts`
- `frontend/tests/visual/visual-baseline.spec.ts`
- `frontend/tests/visual/README.md`
- `frontend/tests/visual/visual-baseline.spec.ts-snapshots/login-mobile-win32.png`
- `frontend/tests/visual/visual-baseline.spec.ts-snapshots/login-dark-win32.png`

### Do Not Touch

- `backend/**`
- `frontend/**`
- `database/**`
- `../pantheon-ops/**`
- existing dirty worktree files

## Implementation Notes

- 共享行为先在 Base 交付；Ops 不得本地复制实现。
- 每个子卡保持现有默认行为兼容，只有页面显式启用高级能力。
- UI 实施必须先过 impeccable 清单，再以实际渲染和交互证据收口。

## Minimum Viable Approach

- Selected Rung: reuse
- Why This Is Enough: 复用 Arco、现有页面模式和偏好设施，只补缺失的共享合同与最小组件。
- Upgrade Trigger: 只有两个以上真实消费场景证明现有组件边界不足时才增加新抽象。

## Success Criteria

- Behaviour Outcome: 未来实现者可按 B1-B5 独立交付，Ops 能通过明确的 foundation release 消费共享能力。
- Verification Signal: task packet、manifest、链接和 Markdown 门禁通过，所有子卡含可证伪验收与视觉矩阵。
- Regression Watch: 当前 Base 默认页面、Arco 体系和 Base/Ops 所有权不变。
- Economics Watch: handoff 不依赖聊天历史，下一执行者只需读取父包和选中子卡。

## Context Strategy

- Entry Sources: `AGENTS.md`, `DESIGN.md`, 本 task packet, 父包 `HANDOFF.md`
- Retrieval Order: `entry -> design -> selected packet -> current code -> evidence`
- Retrieval Helpers: codegraph
- Promotion Target: `docs/designs/OPERATIONAL_WORKBENCH_COMPONENTS_DESIGN.md`
- Response Budget: terse
- Sensitive Context: evidence 不保存凭据、真实业务数据或未脱敏日志

## Method Readiness

- Consumer-Specific Controls: Base task checker、UI contracts、foundation release model
- Required Sensors: command | review | runtime evidence
- Required Evidence: docs checks | desktop/mobile screenshots | interaction assertions | review summary
- Minimal Complexity Rung: reuse
- Ratchet Decision: guide-updated
- Deferred Code Issues: B1-B4 仍为后续实现任务；B5 UI 门禁已实现

## Delivery Governance

- Design Gate: 本 task packet 与 canonical design
- Development Gate: 每次只领取一个子卡并遵守 Expected Files
- QA Acceptance Gate: browser + runtime + human visual review
- GitHub Governance Gate: repo-quality-gate

## Execution Roles

- Implementer Posture: reviewer-assisted
- Reviewer Posture: UX-QA

## Stop Points

- 保存视图跨设备同步或新增后端偏好 schema。
- 共享组件 public API 出现不兼容变更。
- foundation release 发布和 Ops consumer lock 更新。
- 最终视觉/功能验收。

## State Plan

- Checkpoint Expectation: `.harness/tasks/2026-08-31-base-operational-workbench-design/STATUS.md`
- Resume Artifacts: manifest、HANDOFF、STATUS、EXECUTION_QUEUE、选中 packet 与 evidence

## Verification Plan

### Backend

- none

### Frontend

- `node scripts/harness/check-task-packet.mjs --root . docs/harness/tasks/2026-08-31-base-operational-workbench-design.task.md`
- `npm run check:harness-docs`
- `npm run check:harness-inventory`
- `npm run check:ui-quality-gate`
- `npm run test:ui-quality-gate`
- `npm run test:quality-workflow`

### Browser / Smoke

- B5 为治理实现且无新 UI 渲染，使用 manifest 中受限的 governance-only 豁免；B1-B4 与未来 UI 任务按各自 evidence matrix 执行 Playwright。

### Runtime Evidence

- explicit runtime gap: 本轮没有运行时代码变更。

## Linkage

- Task ID: `2026-08-31-base-operational-workbench-design`
- Task Manifest: `.harness/tasks/2026-08-31-base-operational-workbench-design/manifest.json`
- OpenSpec Change: `none`
- Superpowers Plan: `none`
- Plan References: `.harness/tasks/2026-08-31-base-operational-workbench-design/EXECUTION_QUEUE.md`
- Evidence Directory: `.harness/evidence/2026-08-31-base-operational-workbench-design/`
- Review File: `.harness/evidence/2026-08-31-base-operational-workbench-design/review.md`

## Evidence Required

- 文档门禁结果。
- impeccable 设计清单 review。
- 本轮未产出未来 UI 渲染证据的显式 gap。
- 后续每个 UI 子卡的桌面/移动/状态/键盘证据。

## Human Gates

- 新后端偏好存储或 schema。
- 不兼容共享 API。
- foundation release 与 consumer lock。
- 最终视觉/功能验收。

## Sync expectation

- 本轮仅修改 `pantheon-base` 设计与 `.harness` 包。
- B1-B5 实现完成并发布 immutable foundation release 后，Ops 才能开始依赖对应能力。

## Foundation Release Handoff

- Shared change owner: `pantheon-base`
- Foundation release required: `deferred`
- Consumer sync status: `not-started`
- Downstream validation command: `npm run check:foundation-consumer`
- Release/lock stop point: Base visual/runtime evidence green -> foundation release -> Ops lock update -> Ops business validation

## Completion Checklist

- [x] Layer and boundary declared
- [x] Quality profile declared
- [x] Ratchet decision declared
- [x] Delivery governance gates declared
- [x] Contract anchors read
- [x] Verification run or exception recorded
- [x] Evidence saved or summarized
- [x] Docs updated if contracts changed
- [x] Review completed
