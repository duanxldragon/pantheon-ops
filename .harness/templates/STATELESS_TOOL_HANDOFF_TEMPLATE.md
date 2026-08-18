# Stateless Tool Handoff Template

Use this template when work will be resumed by a new chat, another coding tool,
a lower-cost model, or several tools that do not share memory.

## One-Line Instruction

```text
按 .harness/templates/STATELESS_TOOL_HANDOFF_TEMPLATE.md 为 <task/program> 生成无记忆工具交接包。
```

The generated handoff must be self-contained enough that a new tool can start
from repository files only, without reading the original chat.

## When To Use

- The task has multiple follow-up steps or owners.
- The work may be split across expensive and cheaper models.
- The user wants to start a new session without carrying chat history.
- The task touches boundaries such as schema, authorization, state machines,
  cross-module APIs, runtime behavior, or PR/CI handoff.

Do not use this template for trivial one-file edits.

## Required Inputs

Record these before generating the handoff:

| Field | Meaning |
| --- | --- |
| `<task/program>` | Human-readable objective. |
| `<repo>` | Target repository path and ownership layer. |
| `<source-of-truth>` | Audit, plan, task packet, issue, PR, or design source. |
| `<hard-rules>` | Non-negotiable constraints. |
| `<child-tasks>` | Follow-up tasks that can be assigned independently. |
| `<validation>` | Commands, runtime checks, evidence, and stop conditions. |

## Output Files

For a program-level handoff, create:

```text
.harness/tasks/<program-id>/HANDOFF.md
.harness/tasks/<program-id>/EXECUTION_QUEUE.md
.harness/tasks/<program-id>/manifest.json
.harness/tasks/<program-id>/STATUS.md
.harness/evidence/<program-id>/summary.md
.harness/evidence/<program-id>/commands.json
.harness/evidence/<program-id>/review.md
```

For each child task, create or update:

```text
docs/harness/tasks/YYYY-MM-DD-<task-id>.task.md
.harness/tasks/<task-id>/HANDOFF.md
.harness/tasks/<task-id>/manifest.json
.harness/tasks/<task-id>/STATUS.md
```

## HANDOFF.md Shape

```markdown
# Stateless Program Handoff

This file is the entry point for any tool or model with no prior conversation
memory. Do not begin by editing code.

## Start Protocol

1. Set the repository root to `<repo>`.
2. Read the repository AGENTS.md and project inheritance/boundary docs.
3. Read `<source-of-truth>`, the parent task packet, `manifest.json`,
   `STATUS.md`, and `EXECUTION_QUEUE.md`.
4. Run `git status --short`.
5. Treat existing changes as owned by someone else until proven otherwise.
6. Choose only a child task whose dependencies are complete.
7. Read that child's task packet, handoff, manifest, and status.
8. Revalidate cited files against current code.
9. Before implementation, update the child `STATUS.md` with owner, timestamp,
   baseline commit, dirty-file collisions, and the first concrete step.

## Resume Protocol

1. Read only the parent handoff files and the selected child files first.
2. Inspect the selected evidence directory and recent diff.
3. Re-run the smallest validation that proves current baseline.
4. Continue from `Next atomic action`.
5. Update `Completed`, `Decisions`, `Evidence`, and `Next atomic action`.

## Status Vocabulary

- `planned`: untouched task packet.
- `in_progress`: one named owner is changing implementation.
- `blocked`: maintainer decision is required.
- `verification`: implementation frozen; tests/review/evidence remain.
- `complete`: all acceptance criteria and evidence are present.
- `deferred`: maintainer explicitly moved the task out of scope.

## Stop Conditions

Stop when a hard rule, ownership boundary, destructive action, external
production action, incompatible contract, or unowned dirty-file collision appears.
Record one precise question with options and impact.
```

## EXECUTION_QUEUE.md Shape

```markdown
# Execution Queue

## Current State

- Program status: `<status>`.
- Implementation status: `<planned/in_progress/etc>`.
- Source of truth: `<source-of-truth>`.

## Non-Negotiable Rules

1. `<hard-rule-1>`
2. `<hard-rule-2>`

## Batch Order

| Batch | Start condition | Tasks | Owner profile | Stop condition |
| --- | --- | --- | --- | --- |
| A | Now | `<task-id>` | `<strong/low-cost/single-owner>` | `<stop>` |

## Child Task Cards

### <task-id>

- Task packet: `docs/harness/tasks/YYYY-MM-DD-<task-id>.task.md`.
- Handoff: `.harness/tasks/<task-id>/HANDOFF.md`.
- Primary result: `<outcome>`.
- Expected ownership: `<files/modules>`.
- Must not own: `<out-of-scope>`.
- Minimum evidence: `<tests/evidence>`.

## Cost-Aware Assignment Guidance

- Use strong models for schema, authorization, state machine, concurrency,
  security, public contracts, migrations, and external side effects.
- Use lower-cost models only for isolated tests, docs, deterministic scripts,
  and local frontend/type fixes after contracts are frozen.

## Required Handoff Update

Before a tool stops, it must update the selected child `STATUS.md` with status,
owner, baseline, dirty-file collisions, completed changes, decisions, evidence,
next atomic action, and blockers.
```

## manifest.json Shape

```json
{
  "schemaVersion": 1,
  "taskId": "<program-id>",
  "title": "<title>",
  "status": "planned",
  "priority": "<P0/P1/P2>",
  "deliveryTier": "L2",
  "goal": "<goal>",
  "taskDoc": "docs/harness/tasks/YYYY-MM-DD-<program-id>.task.md",
  "handoffDoc": ".harness/tasks/<program-id>/HANDOFF.md",
  "executionQueueDoc": ".harness/tasks/<program-id>/EXECUTION_QUEUE.md",
  "statusDoc": ".harness/tasks/<program-id>/STATUS.md",
  "auditSource": "<source-of-truth>",
  "ownerLayer": "<layer>",
  "children": ["<task-id>"],
  "executionBatches": [["<task-id>"]],
  "globalRules": ["<hard-rule>"],
  "verification": ["<command>"],
  "linkage": {
    "evidenceDir": ".harness/evidence/<program-id>/",
    "reviewFile": ".harness/evidence/<program-id>/review.md",
    "summaryFile": ".harness/evidence/<program-id>/summary.md"
  },
  "maintainerGates": ["<gate>"]
}
```

## STATUS.md Shape

```markdown
# Program Status

- Status: `planned`
- Updated: `<date>`
- Program owner: unassigned
- Baseline commit: record when execution starts
- Source: `<source-of-truth>`
- Execution queue: `.harness/tasks/<program-id>/EXECUTION_QUEUE.md`

## Child Status

| Task | Status | Owner | Evidence |
| --- | --- | --- | --- |
| `<task-id>` | planned | unassigned | pending |

## Known Baseline

- `<known failure/pass/gap>`

## Decisions

- `<decision>`

## Next Atomic Action

Assign the first dependency-ready owner and record dirty-file collisions before
editing code.

## Blockers

None recorded.
```

## Prompt To Give Another Tool

```text
你现在接手 <repo> 的 <task/program>。

先不要改代码。按顺序读取：
1. AGENTS.md
2. 项目继承/边界文档
3. <source-of-truth>
4. <parent task packet>
5. .harness/tasks/<program-id>/HANDOFF.md
6. .harness/tasks/<program-id>/EXECUTION_QUEUE.md
7. .harness/tasks/<program-id>/manifest.json
8. .harness/tasks/<program-id>/STATUS.md

然后执行 git status --short，记录 dirty worktree，不允许 reset/restore/revert
未经授权的改动。只选择依赖已完成的 child task。读取该 child task 的 task
packet、HANDOFF.md、manifest.json、STATUS.md，并在 child STATUS.md 记录
owner、baseline commit、dirty-file collisions、next atomic action 后再实现。

硬规则：
<hard-rules>

完成时必须更新 child STATUS.md 和 .harness/evidence/<task-id>/。
```

## Verification

At minimum, validate:

- generated JSON files parse;
- every referenced path exists;
- `STATUS.md` says whether implementation is planned, in progress, blocked,
  verification, complete, or deferred;
- evidence says which validation was run and which was not run.
