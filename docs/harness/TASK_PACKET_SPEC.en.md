# Task Packet Spec

Chinese version: [TASK_PACKET_SPEC.md](./TASK_PACKET_SPEC.md)

Type: Contract
Layer: platform
Status: Active

A task packet is the tool-agnostic input format for non-trivial work. It lets various agent tools and human engineers share the same task boundary.

## 1. Location

Task packets should live by default at:

```text
docs/harness/tasks/YYYY-MM-DD-<task-name>.task.md
```

If the task comes from an existing superpowers plan, the task packet may reference that plan instead of repeating it in full.

`pantheon-base` also requires each task packet to be paired with a machine-readable task manifest:

```text
.harness/tasks/<task-id>/manifest.json
```

The task packet carries the human-readable boundary. The task manifest carries evidence, review, and automation linkage.

## 2. Required Template

```md
# Task Packet: <task-name>

## Goal

<one sentence>

## Primary Layer

platform | system/auth | system/iam | system/org | system/config | business/*

## Dependency Layers

- none

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

## Structural Scope

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

- Entry Sources: `AGENTS.md`, `CLAUDE.md`, current task packet, latest review summary | none
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
- or pause before schema / contract / delete / release gate work

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
- Superpowers Plan: `docs/superpowers/plans/<file>.md` | none
- Evidence Directory: `.harness/evidence/<task-id>/`
- Review File: `.harness/evidence/<task-id>/review.md` | none

## Evidence Required

- command result summary
- screenshots if UI changed
- smoke JSON if browser flow changed
- runtime logs / metrics / traces / performance signal, or an explicit runtime gap if the task is runtime-sensitive
- session economics snapshot or an explicit gap if the task is long-running, delegated, or cost-sensitive
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

`Execution Roles`, `Stop Points`, `State Plan`, and `Context Strategy` are optional sections, but once a repository contract requires them or a task declares them explicitly, their content must be complete, interpretable, and checkable.

To operationalize `EXECUTION_GUARDRAILS.md`, `non-trivial` work should also default to three short sections:

- `## Assumptions and Open Questions`
- `## Minimum Viable Approach`
- `## Success Criteria`

The current `pantheon-base` checker treats missing sections as warnings because they carry the method signals for thinking first, keeping complexity small, and defining falsifiable completion.

For long-running, high-context, cross-session, or sensitive work, `## Context Strategy` is also recommended. It should make three things explicit:

- which entry sources should be read first
- whether retrieval follows `entry -> summary -> raw`
- which structured retrieval helpers exist, such as codegraph, a graph report, or a wiki hot cache
- where repeated context should be promoted inside repo-owned memory
- how terse or detailed the execution loop should stay by default
- which inputs must be redacted, kept local-only, or excluded from shared durable artifacts

The goal is not extra paperwork. The goal is to make context-loading order and privacy boundaries explicit before the next handoff or resume.

`Economics Watch` in `Success Criteria` is an optional signal for long-running, delegated, or cost-sensitive work. The point is not to optimize every task around tokens. The point is to make retry churn, context replay, cache behavior, and spend visible when they materially affect throughput.

This sync is docs-only. `pantheon-base` has not yet upgraded its local checker or gate code for these new fields, so they are method-level recommendations today rather than mechanical hard requirements.

`Structural Scope` may be `none` for trivial work. For `non-trivial`, cross-layer, runtime-sensitive, permission/menu/i18n/audit/generator/dynamic-module tasks, it should usually capture the smallest affected subgraph so implementers and reviewers inspect the same structural boundary.

## 3. Trivial Tasks

The following tasks may skip task-packet creation:

- typo fixes
- documentation additions that do not change behavior
- read-only diagnosis
- narrow single-file formatting fixes

But if a trivial task touches permissions, menus, schema, i18n, audit, or base/ops inheritance, it must be upgraded into a task packet.

## 4. Human Gate Rules

Task packets must list human gates explicitly. If none exist, write `none`.

The following must be listed as gates:

- schema or migration changes
- deleting files or directories
- base contract changes
- overriding base behavior in a business repository
- new dependencies or external services
- model changes affecting permission, menu, audit, or i18n

## 5. Tool Usage

Tool adapters may translate a task packet into their execution format, but they must not lose:

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

The following fields are the minimum closed-loop keys linking a task packet to later artifacts:

- `Task ID`: primary key; must match the `<task-id>.task.md` filename
- `Task Manifest`: must point to `.harness/tasks/<task-id>/manifest.json`
- `Evidence Directory`: must point to `.harness/evidence/<task-id>/`
- `Review File`: if a review artifact is retained, it must point to a file under the evidence directory
- `OpenSpec Change`: if the task comes from OpenSpec, the change path must be recorded explicitly; otherwise use `none`
- `Superpowers Plan`: if the task comes from `docs/superpowers/plans/*`, it must be recorded explicitly; otherwise use `none`. Downstream manifest, evidence, and review artifacts should carry the same value under `planRefs`

These linkage fields connect `task packet / task manifest / OpenSpec change / superpowers plan / evidence / review` into a traceable chain.
