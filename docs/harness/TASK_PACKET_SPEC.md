---
title: Task Packet Spec
doc_type: Contract
layer: platform
status: Active
updated_at: 2026-06-26
version: v1.1
---

# Task Packet Spec

English version: [TASK_PACKET_SPEC.en.md](./TASK_PACKET_SPEC.en.md)

Task packet 是非 trivial 任务的工具无关输入格式。它让各类 agent 工具和人工工程师共享同一份任务边界。

## 0. 版本历史

| 版本 | 日期       | 变更                                                                                                 |
| ---- | ---------- | ---------------------------------------------------------------------------------------------------- |
| v1.0 | 2026-06-15 | 初始版本                                                                                             |
| v1.1 | 2026-06-26 | 新增 priority/estimatedComplexity/technicalDebtFlag/rollBackPlan/dependencies 字段，增强边界定义量化 |

## 1. 存放位置

任务包默认存放在：

```text
docs/harness/tasks/YYYY-MM-DD-<task-name>.task.md
```

如果任务来自已有 plan（含 legacy superpowers plan），可以在 task packet 中引用该 plan，而不是重复全文。

`pantheon-base` 还要求 task packet 配对 machine-readable task manifest：

```text
.harness/tasks/<task-id>/manifest.json
```

task packet 负责人类可读边界，task manifest 负责 evidence / review / automation linkage。

## 2. 必填模板

```md
# Task Packet: <task-name>

## Goal

<one sentence>

## Priority (v1.1+)

`critical | high | medium | low` (default: `medium`)

## Estimated Complexity (v1.1+)

`trivial | simple | moderate | complex | epic`

## Primary Layer

platform | system/auth | system/iam | system/org | system/config | business/*

## Dependency Layers

- none

## Dependencies (v1.1+)

- blockedBy: `none | task-ids that must complete first`
- blocks: `none | task-ids that depend on this task`

## Harness Profile

- Template: admin-platform | api-service | event-processor | dashboard | ui-heavy-product | custom
- Overlay: none
- Coverage Dimensions:
  - behaviour
  - maintainability
  - architecture-fitness
  - runtime-quality
  - method-health

## Contract Anchors

- `path/to/contract.md`
- `path/to/design.md`
- `path/to/acceptance.md`

## Scope

### In

- <explicit work>

### Out

- <explicit non-goals>

## Assumptions and Open Questions

- Confirmed Facts: `none | facts already verified from code, contracts, logs, or user input`
- Working Assumptions: `none | current assumption that keeps work moving`
- Open Questions: `none | ambiguity that should stop execution or change the plan`

## Structural Scope (v1.1+ Enhanced)

### Scope Quantification

- Modules affected: N
- Files: N (create N, modify N, delete N)
- API endpoints: N
- Database migrations: yes/no

### Risk Nodes

1. [Risk description] - Mitigation approach
2. [Risk description] - Mitigation approach

### Legacy Fields (for compatibility)

- Affected Subgraph: `<entry -> core path -> exit/side effect>` | `none`
- Boundary Crossings: `none | platform -> system/auth | system/* -> pkg/* | base -> ops`
- Risk Nodes: `none | auth handler | permission service | menu registry | generator orchestrator`
- Graph Focus: `none | cycle-check | hub-check | call-depth | sensitive-input-flow`

## Expected Files

### Create

- `path`

### Modify

- `path`

### Do Not Touch

- `path`

## Implementation Notes

- <boundary or sequencing notes>

## Minimum Viable Approach

- Selected Rung: `skip | reuse | stdlib | native platform | installed dependency | small local code | new dependency`
- Why This Is Enough: `<one sentence>`
- Upgrade Trigger: `none | condition that would justify the next rung`

## Success Criteria

- Behaviour Outcome: `<observable result>`
- Verification Signal: `<command, test, or evidence that proves the result>`
- Regression Watch: `<behavior that must remain unchanged>`
- Economics Watch: `none | token/cost/cache/retry/delegation signal that should stay within reason`

## Context Strategy

- Entry Sources: `AGENTS.md`, repo-local agent configuration file, current task packet, latest review summary | none
- Retrieval Order: `entry -> summary -> raw`
- Retrieval Helpers: `none | codegraph | graph report | wiki hot cache`
- Promotion Target: `none | repo wiki | decision log | guide update`
- Response Budget: `terse | standard | detailed`
- Sensitive Context: `none | redacted or local-only handling rule`

## Execution Roles

- Implementer Posture: `<implementer role or none>`
- Reviewer Posture: `<reviewer role or none>`

## Stop Points

- `none`
- 或在 schema / contract / delete / release gate 前停下

## Technical Debt (v1.1+)

- technicalDebtFlag: `yes | no`
- technicalDebtNote: `none | description of technical debt introduced or addressed`

## Rollback Plan (v1.1+)

**Required for destructive changes. Format:**

- Trigger Condition: `none | condition that would trigger rollback`
- Rollback Steps:
  1. `git revert <commit-hash>`
  2. `restore <backup-files>`
  3. `redeploy <if applicable>`
- Rollback Verification:
  - [ ] Functionality verified
  - [ ] Data integrity verified
  - [ ] No side effects observed

## State Plan

- Checkpoint Expectation: `none | path | artifact name`
- Resume Artifacts: `none | path`

## Verification Plan

### Backend

- `command`

### Frontend

- `command`

### Browser / Smoke

- `command or none`

## Linkage

- Task ID: `YYYY-MM-DD-task-name`
- Task Manifest: `.harness/tasks/<task-id>/manifest.json`
- OpenSpec Change: `openspec/changes/<name>/` | none
- Plan References: plan references, workflow references, or resumable execution artifacts | none
- Evidence Directory: `.harness/evidence/<task-id>/`
- Review File: `.harness/evidence/<task-id>/review.md` | none

## Evidence Required

- command result summary
- screenshots if UI changed
- smoke JSON if browser flow changed
- runtime logs / metrics / traces / performance signal, or explicit runtime gap if the task is runtime-sensitive
- session economics snapshot or explicit gap if the task is long-running, delegated, or cost-sensitive
- review summary

## Human Gates

- <gate or none>

## Completion Checklist

- [ ] Layer and boundary declared
- [ ] Contract anchors read
- [ ] Tests or checks updated
- [ ] Verification run or exception recorded
- [ ] Evidence saved or summarized
- [ ] Docs updated if contracts changed
- [ ] Review completed
```

`Execution Roles`、`Stop Points`、`State Plan`、`Context Strategy` 是可选 section，但一旦仓库合同要求或任务显式声明这些 section，内容就必须完整、可解释、可校验。

为落实 `EXECUTION_GUARDRAILS.zh.md`，对 `non-trivial` 任务，默认推荐补齐这三个短 section：

- `## Assumptions and Open Questions`
- `## Minimum Viable Approach`
- `## Success Criteria`

`pantheon-base` 当前 checker 会把这些 section 的缺失视为 warning，因为它们分别承担“先思考、控复杂度、定义可证伪完成信号”的方法职责。

对长任务、高上下文任务、跨 session 任务或涉及敏感信息的任务，推荐补 `## Context Strategy`。它用于显式记录：

- 这次任务先读哪些入口源
- 上下文检索是否遵循 `entry -> summary -> raw`
- 优先使用哪些结构化 retrieval helper，例如 codegraph、graph report 或 wiki hot cache
- 重复出现的上下文应提升到哪个 repo-owned memory surface
- 本轮执行默认保持多简洁的响应预算
- 哪些信息必须脱敏、只本地保留，或不得写入共享 artifact

它的目的不是增加文书，而是把上下文加载顺序和隐私边界写清楚，避免下一次 handoff 或 resume 重新大范围回放。

`Success Criteria` 里的 `Economics Watch` 是可选信号，用于长会话、带 delegation 或成本敏感任务。目标不是让所有任务都围绕 token 优化，而是在这类任务里显式观察 context replay、重试、cache 和费用是否开始反噬吞吐。

本轮只同步文档，不改 `pantheon-base` 的 checker / gate 代码。因此这些新增项当前属于方法层推荐，不是本仓库的机械硬门禁；后续需要强制时，再把 `scripts/harness/*` 同步升级。

`Structural Scope` 对 trivial 任务可写 `none`；对 `non-trivial`、跨层、runtime-sensitive、权限/菜单/i18n/审计/生成器/动态模块相关任务，默认应补最小受影响子图说明。它的目的不是要求维护全仓架构图，而是让实现者和 reviewer 审查同一个结构范围。

## 3. Trivial 任务

以下任务可以不创建 task packet：

- typo 修复
- 不影响行为的文档补充
- 只读诊断
- 单文件小范围格式修复

但如果 trivial 任务触碰权限、菜单、schema、i18n、审计、base/ops 继承关系，必须升级为 task packet。

## 4. Human Gate 规则

任务包必须显式列出 human gates。没有 gate 时写 `none`。

必须列为 gate 的事项：

- schema 或迁移变更
- 删除文件或目录
- base 合同变更
- 业务仓库 override base 行为
- 新依赖或新外部服务
- 影响权限、菜单、审计、i18n 的模型变更

## 5. 工具使用

工具 adapter 可以把 task packet 转换成自己的执行格式，但不得丢失：

- layer
- harness template and coverage dimensions
- scope
- execution roles when declared
- stop points when declared
- state/checkpoint expectations when declared
- context strategy when declared
- contract anchors
- verification plan
- evidence required
- human gates

## 6. Machine-Readable Linkage

以下字段是 task packet 与后续 artifact 的最小闭环键：

- `Task ID`：主键，必须与文件名 `<task-id>.task.md` 一致
- `Task Manifest`：必须指向 `.harness/tasks/<task-id>/manifest.json`
- `Evidence Directory`：必须指向 `.harness/evidence/<task-id>/`
- `Review File`：如保留 review artifact，必须指向 evidence 目录下文件
- `OpenSpec Change`：如任务来自 OpenSpec，必须显式记录 change 路径；否则写 `none`
- `Superpowers Plan`：legacy 字段。如任务来自 `docs/superpowers/plans/*` 历史工件，必须显式记录；否则写 `none`。下游 manifest / evidence / review 统一写入 `planRefs`

这组 linkage 字段用于把 `task packet / task manifest / OpenSpec change / plan reference / evidence / review` 串成可追踪链路。
