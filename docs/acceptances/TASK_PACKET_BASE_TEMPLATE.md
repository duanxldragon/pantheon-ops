---
title: Pantheon Base Task Packet Template
doc_type: Acceptance
layer: platform
status: Active
linked_contracts:
  - docs/contracts/PLATFORM_CONTRACT.md
updated_at: 2026-06-08
---

# Pantheon Base Task Packet Template

English version: [TASK_PACKET_BASE_TEMPLATE.en.md](./TASK_PACKET_BASE_TEMPLATE.en.md)

这是一份给 `pantheon-base` 使用的结构化 task packet 样例。它继承 `pantheon-harness` 的通用方法，只补充当前后台基础平台的业务架构约束。

适用范围：

- `platform`
- `system/auth`
- `system/iam`
- `system/org`
- `system/config`
- `system/lowcode`

直接复制后补全即可：

目标仓库：pantheon-base。跨仓任务必须填写 `## Workspace Context`。同步要求：见 `Sync Expectation` 和 `## Sync expectation`。

```md
# Task Packet: <task-name>

## Goal

<one sentence>

## Primary Layer

platform | system/auth | system/iam | system/org | system/config | business/* | inheritance-sync

## Workspace Context (inheritance-sync 必填)

- Target Repository: `pantheon-base`
- Repository Role: `foundation-source`
- Upstream Dependencies: `pantheon-harness`
- Downstream Consumers: `pantheon-ops`
- Sync Expectation: `none | not-required | deferred | required | completed`
- Release Requirement: `none | method-release | foundation-release | consumer-lock-update`

`Sync Expectation` 和 `Release Requirement` 描述实际交接边界，不代表 release 或下游同步已经完成。

## Dependency Layers

- none

## Harness Profile

- Template: admin-platform
- Overlay: pantheon-base
- Quality Profile: auth-security | permission-policy | i18n | ui-runtime | generator | ci-workflow | none
- Portable Failure Class: instruction-gap | task-boundary-gap | architecture-drift | test-gap | static-sensor-gap | runtime-evidence-gap | security-boundary-gap | ci-signal-noise | method-health-gap | none
- Owner Layer: portable-method | consumer-template | consumer-repository | agent-adapter | no-action
- Coverage Dimensions:
  - behaviour
  - maintainability
  - architecture-fitness
  - runtime-quality
  - method-health

## Contract Anchors

- `pantheon-base/AGENTS.md`
- `pantheon-base/DESIGN.md`
- `pantheon-base/docs/README.md`
- `<matching contract / design / acceptance>`

## Scope

### In

- <explicit closure for this turn>

### Out

- <explicit non-goals for this turn>

## Structural Scope

- Affected Subgraph: `<entry -> core path -> exit/side effect>` | `none`
- Boundary Crossings: `none | platform -> system/auth | system/* -> pkg/* | base -> ops`
- Risk Nodes: `none | auth handler | permission service | menu registry | generator orchestrator`
- Graph Focus: `none | cycle-check | hub-check | call-depth | sensitive-input-flow`

## Expected Files

### Create

- `path` | none

### Modify

- `path` | none

### Do Not Touch

- `path` | none

## Implementation Notes

- 如果任务触碰共享分页、共享上传、共享表格、共享 i18n 或共享后台壳层，默认视为 base-owned。
- 如果任务改变 `pantheon-ops` 继承行为，最终说明必须写清是否需要 `base -> ops` 同步。
- 先应用最小复杂度阶梯：不做 / 复用 / 标准库 / 平台原生 / 已安装依赖 / 一条局部表达式 / 最小新增代码；不得用简化削弱鉴权、审计、i18n、可访问性或运行态证据。

## Method Readiness

- Consumer-Specific Controls: `pantheon-base` contract | checker | smoke path | none
- Required Sensors: command | review | runtime evidence | none
- Required Evidence: command summary | screenshot | smoke result | runtime gap | review summary
- Minimal Complexity Rung: skip | reuse | stdlib | native-platform | installed-dependency | one-local-expression | minimum-new-code
- Ratchet Decision: no-repeat-observed | guide-updated | sensor-added | gate-updated | template-updated | adapter-updated | registry-only
- Deferred Code Issues: none | symptom plus recommended follow-up task

## Delivery Governance

- Design Gate: spec reference | short boundary note | none
- Development Gate: expected files declared | do-not-touch declared | none
- QA Acceptance Gate: command | browser | runtime | human review | none
- GitHub Governance Gate: method-gate | repo-quality-gate | runtime-evidence-gate | external-flaky | not-applicable

## Execution Roles

- Implementer Posture: implementer | reviewer-assisted | docs-only | none
- Reviewer Posture: architecture | security | UX-QA | mechanical | none

## Stop Points

- `none`
- 或在 schema / contract / delete / release gate 前停下

## State Plan

- Checkpoint Expectation: none | path | artifact name
- Resume Artifacts: none | path

## Verification Plan

### Backend

- `go test ...` | `go test ./...` | none

### Frontend

- `npm run build` | none

### Browser / Smoke

- `node scripts/run-smoke-suite.mjs ...` | none

### Runtime Evidence

- focused logs | smoke path | metrics sample | trace | explicit runtime gap | none

## Linkage

- Task ID: `<YYYY-MM-DD-task-name>`
- OpenSpec Change: `openspec/changes/<name>/` | none
- Superpowers Plan: `docs/superpowers/plans/<file>.md` | none
- Evidence Directory: `.harness/evidence/<task-id>/`
- Review File: `.harness/evidence/<task-id>/review.md` | none

## Evidence Required

- command result summary
- screenshots if UI changed
- smoke JSON if browser flow changed
- runtime logs / metrics / traces / performance signal, or explicit runtime gap if runtime-sensitive
- review summary

## Human Gates

- none
- schema / contract / deletion / high-risk permission / release gate

## Sync expectation

- 仅修改 pantheon-base
- 如果共享能力会影响 pantheon-ops，只记录“后续需要同步”或“本轮同步”
- 如果 `base -> ops` sync is required，写明触发条件、同步范围和验证命令

## Foundation Release Handoff

- Shared change owner: `<pantheon-base | pantheon-harness | none>`
- Foundation release required: `yes | no | deferred`
- Consumer sync status: `not-started | planned | in-progress | completed | blocked`
- Downstream validation command: `<exact command or none>`
- Release/lock stop point: `<immutable release, consumer lock update, migration, human gate, or none>`

## Completion Checklist

- [ ] Layer and boundary declared
- [ ] Quality profile or explicit `none` declared
- [ ] Ratchet decision declared for repeated failures
- [ ] Delivery governance gates declared
- [ ] Contract anchors read
- [ ] Verification run or exception recorded
- [ ] Evidence saved or summarized
- [ ] Review completed
```

补充约束：

- 登录、权限、菜单路由、导入导出、生成器、动态模块、异步链路或外部集成默认需要 runtime-sensitive 证据计划。
- 如果同类错误再次出现，Ratchet Decision 不能写 `no-repeat-observed`。
- 如果本轮只是修代码而没有提升为规则、脚本或 smoke，需要在 Deferred Code Issues 或 failure registry 中说明原因。
